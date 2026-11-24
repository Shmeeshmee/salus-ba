package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A request's journey through the API as the app role (docs/clinic-cms WAVE5.md D1): the patient files to the clinic
// they gave a consent, the clinic's admin finds it in the queue, extends it once and answers it, the patient sees the
// answer and cannot withdraw it any more; each step is audited, and the practitioner is refused the queue.
func TestNew_APatientsRequestIsFiledExtendedAndAnswered(t *testing.T) {
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	count := openDB(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL) // row-level security on, as in production
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	d := newDirectory(t, app.Handler, owner)
	filing := `{"kind":"access","target":"clinic","organizationId":"` + inviteClinic + `"}`
	if rec := d.patient.do(http.MethodPost, "/me/data-requests", filing); rec.Code != http.StatusNotFound {
		t.Errorf("filing to a clinic without a record = %d %s, want 404", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07"}`); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /consents = %d %s", rec.Code, rec.Body)
	}
	rec := d.patient.do(http.MethodPost, "/me/data-requests", filing)
	var filed struct{ ID, Status, PatientID string }
	decode(t, rec, &filed)
	if rec.Code != http.StatusCreated || filed.Status != "received" || filed.PatientID != "" {
		t.Fatalf("POST /me/data-requests = %d %s, want 201 received without the profile's id", rec.Code, rec.Body)
	}
	if rec := d.practitioner.do(http.MethodGet, "/clinic/data-requests", ""); rec.Code != http.StatusForbidden {
		t.Errorf("the practitioner lists = %d %s, want 403", rec.Code, rec.Body)
	}
	if rec := d.admin.do(http.MethodGet, "/clinic/data-requests", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), filed.ID) {
		t.Errorf("the admin lists = %d %s, want the request", rec.Code, rec.Body)
	}
	d.admin.ifMatch = `W/"1"`
	if rec := d.admin.do(http.MethodPost, "/clinic/data-requests/"+filed.ID+"/extend", `{"reason":"complex"}`); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"extensionReason":"complex"`) {
		t.Fatalf("extend = %d %s", rec.Code, rec.Body)
	}
	d.admin.ifMatch = `W/"2"`
	if rec := d.admin.do(http.MethodPost, "/clinic/data-requests/"+filed.ID+"/extend", `{"reason":"complex"}`); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "already_extended") {
		t.Errorf("a second extension = %d %s, want 409 already_extended", rec.Code, rec.Body)
	}
	if rec := d.admin.do(http.MethodPost, "/clinic/data-requests/"+filed.ID+"/answer", `{"status":"fulfilled","code":"done"}`); rec.Code != http.StatusOK {
		t.Fatalf("answer = %d %s", rec.Code, rec.Body)
	}
	d.admin.ifMatch = ""
	rec = d.admin.do(http.MethodGet, "/clinic/data-requests/"+filed.ID, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"history":[{"changedAt":`) || !strings.Contains(rec.Body.String(), `"status":"fulfilled"}],"version":3`) {
		t.Errorf("GET one = %d %s, want its history ending fulfilled", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodGet, "/me/data-requests", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"answerCode":"done"`) {
		t.Errorf("the patient lists = %d %s, want the answer", rec.Code, rec.Body)
	}
	d.patient.ifMatch = `W/"3"`
	if rec := d.patient.do(http.MethodPost, "/me/data-requests/"+filed.ID+"/withdraw", ""); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "data_request_closed") {
		t.Errorf("withdraw an answered request = %d %s, want 409 data_request_closed", rec.Code, rec.Body)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE entity_name = 'data_request' AND outcome = 0`); n < 6 {
		t.Errorf("%d audit events of the requests, want one per filing, list, read and change", n)
	}
}

// Filing an access request queues its export, which the patient sees as preparing; a portability request filed while
// it is still queued answers 409 export_in_progress and files nothing (D1 with D2, S4).
func TestNew_AnAccessRequestQueuesItsExportOncePerPatient(t *testing.T) {
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	d := newDirectory(t, app.Handler, owner)
	rec := d.patient.do(http.MethodPost, "/me/data-requests", `{"kind":"access","target":"account"}`)
	var filed struct{ ID string }
	decode(t, rec, &filed)
	if rec.Code != http.StatusCreated {
		t.Fatalf("file access = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodGet, "/me/data-requests/"+filed.ID+"/export", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"queued"`) {
		t.Errorf("its export = %d %s, want queued", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/me/data-requests", `{"kind":"portability","target":"account"}`); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "export_in_progress") {
		t.Errorf("a second export = %d %s, want 409 export_in_progress", rec.Code, rec.Body)
	}
	rec = d.patient.do(http.MethodGet, "/me/data-requests", "")
	if rec.Code != http.StatusOK || strings.Count(rec.Body.String(), `"kind":`) != 1 {
		t.Errorf("the patient's requests = %d %s, want the access request only", rec.Code, rec.Body)
	}
}

// The closure is mounted: a wrong password is refused and changes nothing; the right one closes the account, files its
// erasure request in progress, and sign-in is refused afterwards (D3, S5).
func TestNew_APatientClosesTheirAccount(t *testing.T) {
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	d := newDirectory(t, app.Handler, owner)
	if rec := d.patient.do(http.MethodPost, "/me/account/close", `{"password":"not the password"}`); rec.Code != http.StatusForbidden {
		t.Errorf("a wrong password = %d %s, want 403", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/me/account/close", `{"password":"`+testPassword+`"}`); rec.Code/100 != 2 {
		t.Fatalf("close = %d %s, want 2xx", rec.Code, rec.Body)
	}
	var status string
	if err := owner.QueryRow(t.Context(), `SELECT r.status FROM data_request r JOIN account a ON a.id = r.account_id
		WHERE a.email = 'ana@example.test' AND r.kind = 'erasure' AND a.closed_at IS NOT NULL`).Scan(&status); err != nil || status != "in_progress" {
		t.Errorf("the erasure request = %q, %v; want in_progress on a closed account", status, err)
	}
	again := &client{t: t, h: app.Handler}
	if rec := again.do(http.MethodPost, "/auth/login", `{"email":"ana@example.test","password":"`+testPassword+`"}`); rec.Code == http.StatusOK {
		t.Errorf("sign-in after closure = %d, want refused", rec.Code)
	}
}

// An objection filed in the app marks the profile at once, under row-level security as the app role (S9).
func TestNew_AnObjectionLeavesThePatientOutOfTheAnalytics(t *testing.T) {
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	d := newDirectory(t, app.Handler, owner)
	if rec := d.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07"}`); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/me/data-requests", `{"kind":"objection","target":"account"}`); rec.Code != http.StatusCreated {
		t.Fatalf("file objection = %d %s", rec.Code, rec.Body)
	}
	var marked bool
	if err := owner.QueryRow(t.Context(), `SELECT analytics_objected_at IS NOT NULL FROM patient p JOIN account a ON a.id = p.account_id
		WHERE a.email = 'ana@example.test'`).Scan(&marked); err != nil || !marked {
		t.Errorf("objected %v, %v; want the profile marked", marked, err)
	}
}
