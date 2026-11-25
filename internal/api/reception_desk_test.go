package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The front desk works whether or not its clinic is listed (a clinic waiting for the operator's approval uses its
// desk, docs/clinic-cms F3 note 5): a receptionist sees the clinic's free times through its own scope, registers a
// walk-in with the in-person consent and books them at once with a service, sees the visit on the desk's day with the
// service's name, marks the arrival and issues the receipt. A suspended clinic's booking stop holds at the desk too.
func TestReceptionDesk_AnUnlistedClinicsDeskRegistersBooksChecksInAndIssuesTheReceipt(t *testing.T) {
	t.Parallel()
	r, count := newReception(t)
	seedFree(t, r, 2)
	var schedule, free, exam string
	var start time.Time
	if err := r.owner.QueryRow(t.Context(), `SELECT s.schedule_id::text, s.id::text, lower(s.period),
		(SELECT id::text FROM charge_item_definition WHERE name = 'Synthetic exam')
		FROM slot s WHERE s.status = 'free' ORDER BY lower(s.period) LIMIT 1`).Scan(&schedule, &free, &start, &exam); err != nil {
		t.Fatal(err)
	}
	if _, err := r.owner.Exec(t.Context(), `UPDATE organization SET listed = false WHERE id = $1`, inviteClinic); err != nil {
		t.Fatal(err)
	}
	end := clock.System{}.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	for _, path := range []string{"/fhir/R4/Schedule?actor=PractitionerRole/" + r.roleID, "/fhir/R4/Slot?schedule=Schedule/" + schedule + "&status=free&start=lt" + end} {
		if rec := r.receptionist.do(http.MethodGet, path, ""); rec.Code != http.StatusOK || fhirEmpty(rec.Body.String()) {
			t.Errorf("GET %s at the unlisted clinic = %d %s, want its own", path, rec.Code, rec.Body)
		}
		if rec := r.patient.do(http.MethodGet, path, ""); rec.Code != http.StatusOK || !fhirEmpty(rec.Body.String()) {
			t.Errorf("GET %s as a patient = %d %s, want nothing of an unlisted clinic", path, rec.Code, rec.Body)
		}
	}

	rec := r.receptionist.do(http.MethodPost, "/clinic/patients", `{"givenName":"Vedrana","familyName":"Walkinova","birthDate":"1966-06-06","phone":"+38761000666","consentGivenInPerson":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /clinic/patients = %d %s", rec.Code, rec.Body)
	}
	var walkIn struct{ ID string }
	decode(t, rec, &walkIn)
	rec = r.receptionist.do(http.MethodPost, "/clinic/appointments", `{"patientId":"`+walkIn.ID+`","slotId":"`+free+`","serviceId":"`+exam+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /clinic/appointments for the patient just registered = %d %s, want 201", rec.Code, rec.Body)
	}
	var visit struct{ ID string }
	decode(t, rec, &visit)

	sarajevo, _ := time.LoadLocation("Europe/Sarajevo")
	rec = r.receptionist.do(http.MethodGet, "/clinic/desk/today?date="+start.In(sarajevo).Format(time.DateOnly), "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"serviceName":"Synthetic exam"`) || !strings.Contains(rec.Body.String(), "Walkinova") {
		t.Errorf("GET /clinic/desk/today = %d %s, want the walk-in with the service's name", rec.Code, rec.Body)
	}
	if rec := r.receptionist.do(http.MethodPost, "/clinic/appointments/"+visit.ID+"/arrive", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"arrived"`) {
		t.Errorf("POST arrive = %d %s, want arrived", rec.Code, rec.Body)
	}
	issue := `{"kind":"receipt","appointmentId":"` + visit.ID + `","paymentMethod":"cash","lines":[{"serviceId":"` + exam + `","quantity":1}]}`
	rec = r.receptionist.do(http.MethodPost, "/clinic/receipts", issue)
	if rec.Code != http.StatusCreated {
		t.Errorf("POST /clinic/receipts for the walk-in = %d %s", rec.Code, rec.Body)
	}
	checkDeskReceipts(t, r, visit.ID, issue, rec.Body.String(), start.In(sarajevo).Format(time.DateOnly))
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_account_id = $1 AND outcome = 0 AND entity_reference = $2`, r.receptionAccount, "Appointment/"+visit.ID); n == 0 {
		t.Error("the receptionist's booking is not audited")
	}

	// A suspended membership stops new bookings, at the desk as in the app (docs/plans/clinic-cms.md Q4).
	seedFree(t, r, 3)
	if _, err := r.owner.Exec(t.Context(), `WITH p AS (INSERT INTO subscription_plan (code, name, price_per_seat_minor) VALUES ('basic', 'Basic', 10000))
		INSERT INTO clinic_subscription (organization_id, plan_code, status, trial_ends_at) VALUES ($1, 'basic', 'suspended', now())`, inviteClinic); err != nil {
		t.Fatal(err)
	}
	var another string
	if err := r.owner.QueryRow(t.Context(), `SELECT id::text FROM slot WHERE status = 'free' ORDER BY lower(period) LIMIT 1`).Scan(&another); err != nil {
		t.Fatal(err)
	}
	if rec := r.receptionist.do(http.MethodPost, "/clinic/appointments", `{"patientId":"`+walkIn.ID+`","slotId":"`+another+`"}`); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "slot_unavailable") {
		t.Errorf("POST /clinic/appointments at a suspended clinic = %d %s, want 409 slot_unavailable", rec.Code, rec.Body)
	}
}

// checkDeskReceipts: the receptionist's receipt (issued, its answer) and a second one the admin issues for the visit are
// on the desk's day of date, a colleague's printable at the desk; the visit's receipts list stays the admin's
// (docs/clinic-cms/walk/wave3, findings 1 and 2).
func checkDeskReceipts(t *testing.T, r reception, visit, issue, issued, date string) {
	t.Helper()
	var mine, colleagues struct{ ID string }
	if err := json.Unmarshal([]byte(issued), &mine); err != nil {
		t.Fatal(err)
	}
	decode(t, r.admin.do(http.MethodPost, "/clinic/receipts", issue), &colleagues) // a second receipt is allowed
	rec := r.receptionist.do(http.MethodGet, "/clinic/desk/today?date="+date, "")
	for _, id := range []string{mine.ID, colleagues.ID} {
		if !strings.Contains(rec.Body.String(), `{"id":"`+id+`","number":`) {
			t.Errorf("GET /clinic/desk/today = %s, want the visit's receipt %s", rec.Body, id)
		}
	}
	if rec := r.receptionist.do(http.MethodGet, "/clinic/receipts/"+colleagues.ID+"/pdf", ""); rec.Code != http.StatusOK {
		t.Errorf("GET the PDF of a colleague's receipt of the visit as the receptionist = %d, want 200: the row prints it", rec.Code)
	}
	if rec := r.receptionist.do(http.MethodGet, "/clinic/receipts?appointmentId="+visit, ""); rec.Code != http.StatusForbidden {
		t.Errorf("GET the visit's receipts as the receptionist = %d, want 403 (the admin's)", rec.Code)
	}
	if rec := r.admin.do(http.MethodGet, "/clinic/receipts?appointmentId="+visit, ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), mine.ID) || !strings.Contains(rec.Body.String(), colleagues.ID) {
		t.Errorf("GET the visit's receipts as the admin = %d %s, want both", rec.Code, rec.Body)
	}
}
