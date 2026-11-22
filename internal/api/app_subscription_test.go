package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/api"
)

// Package D of docs/plans/clinic-cms.md through the api: the admin reads the clinic's membership and keeps its billing
// details; a suspended clinic takes no new booking while care goes on, and takes them again once it has paid.
func TestNew_ASuspendedClinicTakesNoNewBookingAndCareGoesOn(t *testing.T) {
	cfg := testConfig(t)
	var log syncBuffer
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	checkMembershipRoutes(t, d, exec)

	slots := seedSlots(t, owner, d.roleID, 7*24*time.Hour, 8*24*time.Hour, 9*24*time.Hour)
	if rec := d.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07","phone":"+38760000777"}`); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /consents = %d %s", rec.Code, rec.Body)
	}
	book := func(slot string) (int, string) {
		rec := d.patient.do(http.MethodPost, "/appointments", `{"slotId":"`+slot+`"}`)
		return rec.Code, rec.Body.String()
	}
	code, body := book(slots[0])
	if code != http.StatusCreated {
		t.Fatalf("booking in the trial = %d %s", code, body)
	}
	var booked struct{ ID string }
	if err := json.Unmarshal([]byte(body), &booked); err != nil {
		t.Fatal(err)
	}
	exec("UPDATE clinic_subscription SET status = 'suspended' WHERE organization_id = $1", inviteClinic)
	if code, body := book(slots[1]); code != http.StatusConflict || !strings.Contains(body, "slot_unavailable") {
		t.Errorf("booking with a suspended clinic = %d %s, want 409 slot_unavailable", code, body)
	}
	// Care never stops: the visit that was booked is still read by its practitioner and cancelled by its patient.
	if rec := d.practitioner.do(http.MethodGet, "/fhir/R4/Appointment/"+booked.ID, ""); rec.Code != http.StatusOK {
		t.Errorf("the practitioner reading the visit of a suspended clinic = %d %s, want 200", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/appointments/"+booked.ID+"/cancel", ""); rec.Code != http.StatusOK {
		t.Errorf("cancelling with a suspended clinic = %d %s, want 200", rec.Code, rec.Body)
	}
	exec("UPDATE clinic_subscription SET status = 'active' WHERE organization_id = $1", inviteClinic)
	if code, body := book(slots[2]); code != http.StatusCreated {
		t.Errorf("booking once the clinic is active again = %d %s, want 201", code, body)
	}
	if strings.Contains(log.String(), "billing@example.test") || strings.Contains(log.String(), "4200000000001") {
		t.Error("the log carries the billing details")
	}
}

// checkMembershipRoutes checks who reads the clinic's membership, its invoices and its billing details, and that the
// admin replaces the details; it leaves the clinic in a trial of the basic plan.
func checkMembershipRoutes(t *testing.T, d directory, exec func(q string, args ...any)) {
	t.Helper()
	if rec := d.admin.do(http.MethodGet, "/clinic/subscription", ""); rec.Code != http.StatusNotFound {
		t.Errorf("no membership: GET /clinic/subscription = %d %s, want 404", rec.Code, rec.Body)
	}
	exec("INSERT INTO subscription_plan (code, name, price_per_seat_minor) VALUES ('basic', 'Basic', 10000)")
	exec("INSERT INTO clinic_subscription (organization_id, plan_code, status, trial_ends_at) VALUES ($1, 'basic', 'trial', now() + interval '30 days')", inviteClinic)
	rec := d.admin.do(http.MethodGet, "/clinic/subscription", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"trial"`) || !strings.Contains(rec.Body.String(), `"planName":"Basic"`) {
		t.Errorf("GET /clinic/subscription = %d %s, want the trial", rec.Code, rec.Body)
	}
	if rec := d.admin.do(http.MethodGet, "/clinic/subscription/invoices", ""); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("GET /clinic/subscription/invoices = %d %s, want none", rec.Code, rec.Body)
	}
	for _, path := range []string{"/clinic/subscription", "/clinic/subscription/invoices", "/clinic/billing-details"} {
		if rec := d.practitioner.do(http.MethodGet, path, ""); rec.Code != http.StatusForbidden {
			t.Errorf("practitioner GET %s = %d, want 403", path, rec.Code)
		}
		if rec := d.patient.do(http.MethodGet, path, ""); rec.Code != http.StatusNotFound {
			t.Errorf("patient GET %s = %d, want 404", path, rec.Code)
		}
	}
	details := `{"vatRegistered":true,"legalName":"Synthetic clinic d.o.o.","taxId":"4200000000001","billingEmail":"billing@example.test"}`
	if rec := d.admin.do(http.MethodPut, "/clinic/billing-details", details); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != details {
		t.Errorf("PUT /clinic/billing-details = %d %s", rec.Code, rec.Body)
	}
	if rec := d.admin.do(http.MethodGet, "/clinic/billing-details", ""); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != details {
		t.Errorf("GET /clinic/billing-details = %d %s", rec.Code, rec.Body)
	}
	if rec := d.admin.do(http.MethodPut, "/clinic/billing-details", `{"vatRegistered":true,"taxId":"42"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a short tax id: %d, want 400", rec.Code)
	}
}
