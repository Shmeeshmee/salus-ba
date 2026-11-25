package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

// The price list (docs/plans/clinic-cms.md, package B): the clinic's admin keeps it, its staff read it, anyone sees
// the active services of a listed clinic in a visit's mode, and every write is audited, also when refused.
func TestNew_KeepsTheClinicsPriceListAndShowsItsPricesAtBooking(t *testing.T) {
	t.Parallel()
	cfg := testConfig(t)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	anyone := &client{t: t, h: app.Handler}
	const services = "/clinic/services"
	const exam = `{"name":"Placeholder exam","priceMinor":5000,"vatRateBp":1700,"modes":["in-person"],"active":true,` +
		`"translations":[{"language":"de","name":"Platzhalter"}]}`

	rec := d.admin.do(http.MethodPost, services, exam)
	var made struct {
		ID       string `json:"id"`
		Currency string `json:"currency"`
	}
	if rec.Code != http.StatusCreated || rec.Header().Get("ETag") != `W/"1"` {
		t.Fatalf("POST as the admin = %d %s", rec.Code, rec.Body)
	}
	decode(t, rec, &made)
	if made.ID == "" || made.Currency != "BAM" {
		t.Errorf("POST stored %+v, want an id in BAM", made)
	}
	if rec := d.practitioner.do(http.MethodPost, services, exam); rec.Code != http.StatusForbidden {
		t.Errorf("POST as the practitioner = %d %s, want 403", rec.Code, rec.Body)
	}
	if rec := d.admin.do(http.MethodPost, services, strings.Replace(exam, "5000", "-1", 1)); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_service") {
		t.Errorf("POST of a negative price = %d %s, want 400 invalid_service", rec.Code, rec.Body)
	}
	if rec := d.practitioner.do(http.MethodGet, services, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), made.ID) {
		t.Errorf("GET as the practitioner = %d %s, want the service", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodGet, services, ""); rec.Code != http.StatusNotFound {
		t.Errorf("GET as a patient = %d %s, want 404", rec.Code, rec.Body)
	}

	// At booking: only a listed clinic's, in the visit's mode, in the patient's language.
	offered := "/organizations/" + inviteClinic + "/services?mode="
	if rec := anyone.do(http.MethodGet, offered+"in-person", ""); rec.Code != http.StatusNotFound {
		t.Errorf("GET the prices of a clinic that is not listed = %d %s, want 404", rec.Code, rec.Body)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE organization SET listed = true WHERE id = $1`, inviteClinic); err != nil {
		t.Fatal(err)
	}
	want := `[{"id":"` + made.ID + `","name":"Platzhalter","currency":"BAM","priceMinor":5000}]` + "\n"
	if rec := anyone.do(http.MethodGet, offered+"in-person&language=de", ""); rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET the prices in person = %d %s, want %s", rec.Code, rec.Body, want)
	}
	if rec := d.patient.do(http.MethodGet, offered+"video", ""); rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Errorf("GET the prices by video = %d %s, want none", rec.Code, rec.Body)
	}
	if rec := anyone.do(http.MethodGet, offered+"phone", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("GET the prices of an unknown mode = %d, want 400", rec.Code)
	}

	// A change at the version read; taking it off the list hides it at booking.
	one := services + "/" + made.ID
	off := strings.Replace(exam, `"active":true`, `"active":false`, 1)
	d.admin.ifMatch = `W/"1"`
	if rec := d.admin.do(http.MethodPut, one, off); rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` {
		t.Fatalf("PUT as the admin = %d %s", rec.Code, rec.Body)
	}
	if rec := d.admin.do(http.MethodPut, one, off); rec.Code != http.StatusPreconditionFailed {
		t.Errorf("PUT at the old version = %d %s, want 412", rec.Code, rec.Body)
	}
	d.admin.ifMatch = ""
	if rec := d.admin.do(http.MethodPut, one, off); rec.Code != http.StatusBadRequest {
		t.Errorf("PUT without If-Match = %d %s, want 400", rec.Code, rec.Body)
	}
	if rec := anyone.do(http.MethodGet, offered+"in-person", ""); rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Errorf("GET the prices after it was taken off = %d %s, want none", rec.Code, rec.Body)
	}
	var written, refused int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE outcome = 0 AND entity_reference = 'ChargeItemDefinition/' || $1),
			count(*) FILTER (WHERE outcome = 4 AND entity_reference IS NULL)
		FROM audit_event WHERE subtype_code IN ('create', 'update') AND organization_id = $2`, made.ID, inviteClinic).Scan(&written, &refused); err != nil || written != 2 || refused != 2 {
		t.Errorf("audited %d writes of the service and %d refusals, %v; want 2 and 2", written, refused, err)
	}
}
