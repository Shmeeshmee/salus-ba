package api_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
)

// seedSlots lists the directory's clinic and gives its role a schedule with a free slot at each offset from
// now (whole minutes), and returns the slots' ids in that order.
func seedSlots(t *testing.T, owner *pgxpool.Pool, roleID string, offsets ...time.Duration) []string {
	t.Helper()
	ctx := t.Context()
	var sched string
	if err := owner.QueryRow(ctx, `WITH o AS (UPDATE organization SET listed = true WHERE id = $1)
		INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) VALUES ($1, $2, 30) RETURNING id`, inviteClinic, roleID).Scan(&sched); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(offsets))
	for _, off := range offsets {
		var id string
		if err := owner.QueryRow(ctx, `INSERT INTO slot (schedule_id, organization_id, period)
			VALUES ($1, $2, tstzrange(date_trunc('minute', now()) + $3::interval, date_trunc('minute', now()) + $3::interval + interval '30 minutes'))
			RETURNING id`, sched, inviteClinic, fmt.Sprintf("%d seconds", int(off.Seconds()))).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

func TestNew_APatientBooksAndCancelsAndThePractitionerMarksAppointmentsAudited(t *testing.T) {
	cfg := testConfig(t)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	count := openDB(t, cfg)
	slots := seedSlots(t, owner, d.roleID, 7*24*time.Hour, 23*time.Hour)
	week, soon := slots[0], slots[1]
	book := func(slot string) (int, string) {
		rec := d.patient.do(http.MethodPost, "/appointments", `{"slotId":"`+slot+`"}`)
		var out struct{ ID string }
		if rec.Code == http.StatusCreated {
			decode(t, rec, &out)
		}
		return rec.Code, out.ID + rec.Body.String()
	}

	if code, body := book(week); code != http.StatusConflict || !strings.Contains(body, "profile_required") {
		t.Errorf("booking without a profile = %d %s, want 409 profile_required", code, body)
	}
	profile := `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07","phone":"+38760000777"}`
	if rec := d.patient.do(http.MethodPut, "/me/patient", profile); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if code, body := book(week); code != http.StatusConflict || !strings.Contains(body, "consent_required") {
		t.Errorf("booking without a consent = %d %s, want 409 consent_required", code, body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /consents = %d %s", rec.Code, rec.Body)
	}
	rec := d.patient.do(http.MethodPost, "/appointments", `{"slotId":"`+week+`"}`)
	var weekAppt struct{ ID, Status, SlotID string }
	decode(t, rec, &weekAppt)
	if rec.Code != http.StatusCreated || weekAppt.Status != "booked" || weekAppt.SlotID != week {
		t.Fatalf("POST /appointments = %d %s, want 201 booked", rec.Code, rec.Body)
	}
	if code, body := book(week); code != http.StatusConflict || !strings.Contains(body, "slot_unavailable") {
		t.Errorf("booking a taken slot = %d %s, want 409 slot_unavailable", code, body)
	}
	code, soonID := book(soon)
	soonID = soonID[:36]
	if code != http.StatusCreated {
		t.Fatalf("booking 23 hours ahead = %d %s", code, soonID)
	}
	if n := count("SELECT count(*) FROM slot WHERE status = 'busy' AND id IN ($1, $2)", week, soon); n != 2 {
		t.Errorf("%d of the booked slots are busy, want both", n)
	}
	// The confirmations, and a reminder only for the appointment more than 24 hours ahead.
	for kind, want := range map[string]int{"appointment_booked_email": 2, "appointment_reminder_email": 1} {
		if n := count("SELECT count(*) FROM river_job WHERE kind = $1 AND queue = 'email'", kind); n != want {
			t.Errorf("%d %s jobs, want %d", n, kind, want)
		}
	}

	checkCancelling(t, d, count, week, soon, weekAppt.ID, soonID)
	checkMarking(t, d, owner)
	checkAppointmentTrail(t, count, log.String())
}

// checkCancelling checks who may book and cancel the two booked appointments of the test above, and that
// their slots are freed and the patient e-mailed.
func checkCancelling(t *testing.T, d directory, count func(q string, args ...any) int, week, soon, weekID, soonID string) {
	t.Helper()
	for _, staff := range []*client{d.admin, d.practitioner} {
		if rec := staff.do(http.MethodPost, "/appointments", `{"slotId":"`+week+`"}`); rec.Code != http.StatusForbidden {
			t.Errorf("staff booking = %d %s, want 403", rec.Code, rec.Body)
		}
	}
	if rec := d.patient.do(http.MethodPost, "/appointments/"+soonID+"/cancel", ""); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "cancellation_window_closed") {
		t.Errorf("the patient cancelling 23 hours ahead = %d %s, want 409 cancellation_window_closed", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/appointments/"+weekID+"/cancel", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"cancellationReason":"pat"`) {
		t.Errorf("the patient cancelling a week ahead = %d %s, want 200 (pat)", rec.Code, rec.Body)
	}
	if rec := d.practitioner.do(http.MethodPost, "/appointments/"+soonID+"/cancel", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"cancellationReason":"prov"`) {
		t.Errorf("the practitioner cancelling 23 hours ahead = %d %s, want 200 (prov)", rec.Code, rec.Body)
	}
	if n := count("SELECT count(*) FROM slot WHERE status = 'free' AND id IN ($1, $2)", week, soon); n != 2 {
		t.Errorf("%d of the cancelled appointments' slots are free, want both", n)
	}
	if n := count("SELECT count(*) FROM river_job WHERE kind = 'appointment_cancelled_email' AND queue = 'email'"); n != 2 {
		t.Errorf("%d cancellation e-mails, want 2", n)
	}

}

// checkMarking checks marking an appointment that started an hour ago (seeded as the owner).
func checkMarking(t *testing.T, d directory, owner *pgxpool.Pool) {
	t.Helper()
	var past string
	if err := owner.QueryRow(t.Context(), `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
		SELECT id, $1, $2, gen_random_uuid(), tstzrange(now() - interval '1 hour', now() - interval '30 minutes') FROM patient RETURNING id`,
		d.roleID, inviteClinic).Scan(&past); err != nil {
		t.Fatal(err)
	}
	path := "/appointments/" + past + "/status"
	for _, tt := range []struct {
		c      *client
		name   string
		body   string
		status int
	}{
		{d.admin, "the admin", `{"status":"fulfilled"}`, http.StatusForbidden},
		{d.patient, "the patient", `{"status":"fulfilled"}`, http.StatusForbidden},
		{d.practitioner, "a status that is not theirs to set", `{"status":"cancelled"}`, http.StatusBadRequest},
		{d.practitioner, "the practitioner", `{"status":"fulfilled"}`, http.StatusOK},
		// Marked once: the same status again is closed (a noshow would be a correction, SAL-36's Q6).
		{d.practitioner, "the practitioner again", `{"status":"fulfilled"}`, http.StatusConflict},
	} {
		if rec := tt.c.do(http.MethodPost, path, tt.body); rec.Code != tt.status {
			t.Errorf("POST %s as %s = %d %s, want %d", path, tt.name, rec.Code, rec.Body, tt.status)
		}
	}
}

// checkAppointmentTrail checks the audit events of the test above and that its log holds none of the
// patient's data.
func checkAppointmentTrail(t *testing.T, count func(q string, args ...any) int, log string) {
	t.Helper()
	for _, tt := range []struct {
		name, where string
		want        int
	}{
		{"the bookings", "subtype_code = 'create' AND outcome = 0 AND entity_reference LIKE 'Appointment/%' AND purpose_code = 'PATRQT'", 2},
		{"the refused bookings", "subtype_code = 'create' AND outcome = 4 AND entity_reference IS NULL AND purpose_code = 'PATRQT' AND organization_id IS NULL", 3},
		{"the staff's refused bookings", "subtype_code = 'create' AND outcome = 4 AND purpose_code = 'TREAT'", 2},
		{"the cancellations", "subtype_code = 'update' AND outcome = 0 AND entity_reference LIKE 'Appointment/%' AND agent_account_id IS NOT NULL", 3},
		{"the refused changes", "subtype_code = 'update' AND outcome = 4 AND entity_reference IS NULL AND (purpose_code = 'TREAT' OR purpose_code = 'PATRQT')", 4},
	} {
		if got := count("SELECT count(*) FROM audit_event WHERE type_code = 'rest' AND " + tt.where); got != tt.want {
			t.Errorf("audit events for %s: %d, want %d", tt.name, got, tt.want)
		}
	}
	for _, field := range []string{"Zorana", "Syntheticova", "1977", "38760000777", "ana@example.test"} {
		if strings.Contains(log, field) {
			t.Errorf("the log holds %q:\n%s", field, log)
		}
	}
}

// Two bookings of one slot at once: one is booked, the other refused.
func TestNew_TwoConcurrentBookingsOfOneSlotGiveOneAppointment(t *testing.T) {
	app, cfg := newApp(t)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	if rec := d.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07"}`); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /consents = %d %s", rec.Code, rec.Body)
	}
	slots := seedSlots(t, owner, d.roleID, 48*time.Hour, 72*time.Hour, 96*time.Hour, 120*time.Hour, 144*time.Hour)
	for _, slot := range slots {
		codes := make([]int, 2)
		var wg sync.WaitGroup
		for i := range codes {
			wg.Go(func() {
				c := &client{t: t, h: app.Handler, bearer: d.patient.bearer}
				codes[i] = c.do(http.MethodPost, "/appointments", `{"slotId":"`+slot+`"}`).Code
			})
		}
		wg.Wait()
		if (codes[0] != http.StatusCreated || codes[1] != http.StatusConflict) && (codes[1] != http.StatusCreated || codes[0] != http.StatusConflict) {
			t.Errorf("two bookings of one slot at once = %v, want one 201 and one 409", codes)
		}
	}
}
