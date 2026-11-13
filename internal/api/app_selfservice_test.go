package api_test

import (
	"crypto/sha256"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
)

// A patient's confirmation through a reminder's one-time link, their actions and a reschedule in the app, through the
// api as it is composed (SAL-34). The link is stored as a notifier job would (SAL-33 issues it from its reminders).
func TestNew_APatientConfirmsThroughTheLinkAndReschedulesInTheApp(t *testing.T) {
	cfg := testConfig(t)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	count := openDB(t, cfg)
	slots := seedSlots(t, owner, d.roleID, 7*24*time.Hour, 8*24*time.Hour)
	profile := `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07"}`
	if rec := d.patient.do(http.MethodPut, "/me/patient", profile); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /consents = %d %s", rec.Code, rec.Body)
	}
	rec := d.patient.do(http.MethodPost, "/appointments", `{"slotId":"`+slots[0]+`"}`)
	var appt struct{ ID string }
	decode(t, rec, &appt)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /appointments = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodGet, "/appointments/"+appt.ID+"/actions", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"confirm":true`) || !strings.Contains(rec.Body.String(), `"reschedules_left":2`) {
		t.Errorf("GET actions = %d %s, want confirm and 2 reschedules left", rec.Code, rec.Body)
	}
	if rec := d.practitioner.do(http.MethodPost, "/appointments/"+appt.ID+"/confirm", ""); rec.Code != http.StatusForbidden {
		t.Errorf("the practitioner confirming = %d %s, want 403", rec.Code, rec.Body)
	}

	confirmThroughTheLink(t, app.Handler, owner, count, appt.ID)
	rescheduleInTheApp(t, d, count, appt.ID, slots)
}

// confirmThroughTheLink stores a link of the appointment id as the notifier would and confirms through its page.
func confirmThroughTheLink(t *testing.T, h http.Handler, owner *pgxpool.Pool, count func(string, ...any) int, id string) {
	t.Helper()
	raw := sha256.Sum256([]byte("a synthetic link token"))
	hash := sha256.Sum256(raw[:])
	token := base64.RawURLEncoding.EncodeToString(raw[:])
	if _, err := owner.Exec(t.Context(), `INSERT INTO appointment_action_token (created_at, expires_at, start_at, appointment_id, organization_id,
		patient_id, account_id, token_hash, channel, language)
		SELECT now(), lower(a.period), lower(a.period), a.id, a.organization_id, a.patient_id, p.account_id, $2, 'email', 'bs'
		FROM appointment a JOIN patient p ON p.id = a.patient_id WHERE a.id = $1`, id, hash[:]); err != nil {
		t.Fatal(err)
	}
	page := func() string {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/a/"+token, nil))
		if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("GET /a/<token> = %d", w.Code)
		}
		return w.Body.String()
	}
	if body := page(); !strings.Contains(body, "Potvrđujem da ću doći") || !strings.Contains(body, `name="confirm_cancel"`) {
		t.Errorf("the link's page lacks its actions in Bosnian:\n%s", body)
	}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/a/"+token, strings.NewReader(url.Values{"action": {"confirm"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://"+r.Host)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("POST /a/<token> = %d %s, want 303", w.Code, w.Body)
	}
	if body := page(); !strings.Contains(body, "Potvrdili ste ovaj termin.") || strings.Contains(body, "<form") {
		t.Errorf("the used link's page is not its outcome:\n%s", body)
	}
	if n := count(`SELECT count(*) FROM appointment WHERE id = $1 AND confirmed_via = 'link'`, id); n != 1 {
		t.Error("the appointment is not confirmed through the link")
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_name = 'action_link' AND agent_account_id IS NOT NULL`); n < 3 {
		t.Errorf("%d audit events of the link, want its reads and its use by the patient's account", n)
	}
}

// rescheduleInTheApp moves the confirmed appointment id to the second slot in the app.
func rescheduleInTheApp(t *testing.T, d directory, count func(string, ...any) int, id string, slots []string) {
	t.Helper()

	rec := d.patient.do(http.MethodPost, "/appointments/"+id+"/reschedule", `{"slotId":"`+slots[1]+`"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"rescheduleCount":1`) || !strings.Contains(rec.Body.String(), `"slotId":"`+slots[1]+`"`) ||
		!strings.Contains(rec.Body.String(), `"confirmedVia":"link"`) {
		t.Errorf("POST reschedule = %d %s, want the new slot, counted, the link's confirmation kept", rec.Code, rec.Body)
	}
	if n := count("SELECT count(*) FROM slot WHERE (id = $1 AND status = 'free') OR (id = $2 AND status = 'busy')", slots[0], slots[1]); n != 2 {
		t.Errorf("the old slot free and the new busy: %d of 2", n)
	}
	for kind, want := range map[string]int{"appointment_rescheduled_email": 1, "waitlist_slot_freed": 1} {
		if n := count("SELECT count(*) FROM river_job WHERE kind = $1", kind); n != want {
			t.Errorf("%d %s jobs, want %d", n, kind, want)
		}
	}
	rec = d.patient.do(http.MethodGet, "/appointments/"+id+"/changes", "")
	for _, kind := range []string{`"kind":"confirmed","via":"link"`, `"kind":"rescheduled","via":"app"`} {
		if !strings.Contains(rec.Body.String(), kind) {
			t.Errorf("GET changes = %s, lacks %s", rec.Body, kind)
		}
	}
}
