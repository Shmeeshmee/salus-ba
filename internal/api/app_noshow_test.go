package api_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// SAL-36 through the stack (synthetic data): the practitioner marks a no-show, corrects it, the admin corrects it
// back, each kept in the history and audited; a correction after 30 days and the patient are refused. With the
// clinic's threshold on, staff get the repeat no-show flag beside the patient's next appointment; the patient gets 404.
func TestNew_NoShowsAreCorrectedWithin30DaysAndFlaggedForStaffOnly(t *testing.T) {
	app, cfg := newApp(t)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	count := openDB(t, cfg)
	if rec := d.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07"}`); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /consents = %d %s", rec.Code, rec.Body)
	}
	seed := func(from, status string) string {
		t.Helper()
		var id string
		if err := owner.QueryRow(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
			SELECT id, $1, $2, gen_random_uuid(), tstzrange(now() + $3::interval, now() + $3::interval + interval '30 minutes'), $4 FROM patient RETURNING id`,
			d.roleID, inviteClinic, from, status).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	recent, old, next := seed("-1 hour", "booked"), seed("-31 days", "noshow"), seed("2 days", "booked")

	status := func(c *client, id, to string) *httptest.ResponseRecorder {
		return c.do(http.MethodPost, "/appointments/"+id+"/status", `{"status":"`+to+`"}`)
	}
	for _, tt := range []struct {
		c      *client
		name   string
		id, to string
		want   int
	}{
		{d.practitioner, "the practitioner marks the no-show", recent, "noshow", http.StatusOK},
		{d.practitioner, "the practitioner corrects it", recent, "fulfilled", http.StatusOK},
		{d.admin, "the admin corrects it back", recent, "noshow", http.StatusOK},
		{d.patient, "the patient", recent, "fulfilled", http.StatusForbidden},
		{d.practitioner, "after 30 days", old, "fulfilled", http.StatusConflict},
		{d.admin, "the admin on a booked one", next, "noshow", http.StatusForbidden},
	} {
		if rec := status(tt.c, tt.id, tt.to); rec.Code != tt.want {
			t.Errorf("%s: %d %s, want %d", tt.name, rec.Code, rec.Body, tt.want)
		}
	}
	if rec := status(d.practitioner, old, "fulfilled"); !strings.Contains(rec.Body.String(), "correction_window_closed") {
		t.Errorf("after 30 days: %s, want correction_window_closed", rec.Body)
	}
	rec := d.practitioner.do(http.MethodGet, "/appointments/"+recent+"/changes", "")
	if body := rec.Body.String(); rec.Code != http.StatusOK || strings.Count(body, `"status_corrected"`) != 2 || strings.Count(body, `"status_set"`) != 1 {
		t.Errorf("GET changes = %d %s, want a status_set and two status_corrected", rec.Code, body)
	}
	if n := count(`SELECT count(*) FROM appointment_change WHERE appointment_id = $1 AND kind = 'status_corrected' AND via = 'staff'
		AND by_account_id IS NOT NULL`, recent); n != 2 {
		t.Errorf("%d corrections by staff in the history, want 2", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE subtype_code = 'update' AND outcome = 0 AND entity_reference = $1`, "Appointment/"+recent); n != 3 {
		t.Errorf("%d audited updates of the appointment, want 3", n)
	}

	settings := `{"reminder_offsets_minutes":[1440,60],"sms_reminders":false,"reschedule_notice_hours":24,"reschedule_max":2,` +
		`"waitlist_enabled":false,"waitlist_hold_minutes":15,"noshow_flag_threshold":2,"noshow_flag_window_days":180}`
	var dbNow time.Time // the database's clock, which the seeded appointments were placed by
	if err := owner.QueryRow(t.Context(), "SELECT now()").Scan(&dbNow); err != nil {
		t.Fatal(err)
	}
	flags := "/clinic/noshow-flags?from=" + url.QueryEscape(dbNow.UTC().Format(time.RFC3339)) +
		"&to=" + url.QueryEscape(dbNow.Add(7*24*time.Hour).UTC().Format(time.RFC3339))
	if rec := d.admin.do(http.MethodGet, flags, ""); rec.Code != http.StatusOK || rec.Body.String() != `{"flags":[]}`+"\n" {
		t.Errorf("flags while the rule is off = %d %s, want none", rec.Code, rec.Body)
	}
	if rec := d.admin.do(http.MethodPut, "/clinic/appointment-settings", settings); rec.Code != http.StatusOK {
		t.Fatalf("PUT /clinic/appointment-settings = %d %s", rec.Code, rec.Body)
	}
	want := `{"flags":[{"appointment_id":"` + next + `","noshows":2,"window_days":180}]}` + "\n"
	for _, c := range []*client{d.admin, d.practitioner} {
		if rec := c.do(http.MethodGet, flags, ""); rec.Code != http.StatusOK || rec.Body.String() != want {
			t.Errorf("flags = %d %s, want %s", rec.Code, rec.Body, want)
		}
	}
	if rec := d.patient.do(http.MethodGet, flags, ""); rec.Code != http.StatusNotFound {
		t.Errorf("the patient's flags = %d %s, want 404", rec.Code, rec.Body)
	}
}
