package api_test

import (
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A treatment plan books every step or none, on the real stack with row-level security on (docs/clinic-cms/WAVE6.md
// U6, package TP1): the patient sees it once booked, and cancelling it cancels its future visits.
func TestNew_ATreatmentPlanBooksEveryStepOrNoneAndCancelsItsVisits(t *testing.T) {
	cfg := testConfig(t)
	owner := ownerPool(t, cfg)
	count := openDB(t, cfg)
	cfg.DatabaseURL = pgtest.AppURL(t, cfg.DatabaseURL)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	d := newDirectory(t, app.Handler, owner)
	slots := seedSlots(t, owner, d.roleID, 48*time.Hour, 72*time.Hour)
	if _, err := owner.Exec(t.Context(), `UPDATE slot SET visit_mode = 'in-person' WHERE id = $1`, slots[0]); err != nil {
		t.Fatal(err)
	}
	if rec := d.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Ada","familyName":"Example","birthDate":"1990-01-01"}`); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /consents = %d %s", rec.Code, rec.Body)
	}
	exam := saveService(t, d, `{"name":"Synthetic exam","priceMinor":5000,"modes":["video","in-person"],"active":true,"translations":[]}`)
	var patientID string
	if err := owner.QueryRow(t.Context(), `SELECT id::text FROM patient`).Scan(&patientID); err != nil {
		t.Fatal(err)
	}
	rec := d.admin.do(http.MethodPost, "/clinic/care-plans", `{"patientId":"`+patientID+`","title":"Synthetic plan","steps":[
		{"serviceId":"`+exam+`","dayOffset":0,"visitMode":"in-person"},{"serviceId":"`+exam+`","dayOffset":1,"visitMode":"video"}]}`)
	var plan struct {
		ID     string
		Status string
	}
	decode(t, rec, &plan)
	if rec.Code != http.StatusCreated || plan.Status != "draft" {
		t.Fatalf("POST /clinic/care-plans = %d %s", rec.Code, rec.Body)
	}
	book := func(first, second string) int {
		d.admin.ifMatch = rec.Header().Get("ETag")
		defer func() { d.admin.ifMatch = "" }()
		return d.admin.do(http.MethodPost, "/clinic/care-plans/"+plan.ID+"/book",
			`{"startsOn":"2030-01-07","slotIds":["`+first+`","`+second+`"]}`).Code
	}
	if got := book(slots[0], slots[0]); got != http.StatusConflict { // the second step's slot is taken by the first
		t.Fatalf("booking one slot twice = %d, want 409", got)
	}
	if n := count(`SELECT count(*) FROM appointment`); n != 0 {
		t.Fatalf("%d visits after a refused booking, want none", n)
	}
	if got := book(slots[0], slots[1]); got != http.StatusOK {
		t.Fatalf("booking = %d", got)
	}
	if n := count(`SELECT count(*) FROM appointment WHERE status = 'booked' AND booked_by_account_id IS NOT NULL`); n != 2 {
		t.Errorf("%d visits booked, want 2", n)
	}
	mine := d.patient.do(http.MethodGet, "/me/care-plans", "")
	var plans []struct{ Status string }
	decode(t, mine, &plans)
	if len(plans) != 1 || plans[0].Status != "active" {
		t.Errorf("GET /me/care-plans = %d %s, want the active plan", mine.Code, mine.Body)
	}
	d.admin.ifMatch = `W/"2"`
	if rec := d.admin.do(http.MethodPost, "/clinic/care-plans/"+plan.ID+"/cancel", ""); rec.Code != http.StatusOK {
		t.Fatalf("cancel = %d %s", rec.Code, rec.Body)
	}
	d.admin.ifMatch = ""
	if n := count(`SELECT count(*) FROM appointment WHERE status = 'cancelled'`); n != 2 {
		t.Errorf("%d visits cancelled, want both", n)
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE entity_reference = $1`, "CarePlan/"+plan.ID); n < 4 {
		t.Errorf("%d audit events of the plan, want its write, booking, cancellation and read", n)
	}
}
