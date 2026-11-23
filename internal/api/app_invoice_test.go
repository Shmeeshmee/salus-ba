package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/api"
)

// Packages B, C and D of docs/plans/clinic-cms.md wired together through the api: a receipt for a visit is priced from
// the clinic's own price list and names the clinic as its seller from its billing details (D) and address (A). While
// either is missing the receipt is refused with 409 billing_details_missing; a service that is unknown, inactive or
// another clinic's is a client error, never a 500.
func TestNew_AReceiptIsPricedFromTheClinicsListAndNamesItsSeller(t *testing.T) {
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
	visit := bookedVisit(t, d, owner)
	exam := saveService(t, d, `{"name":"Synthetic exam","priceMinor":5000,"modes":["video","in-person"],"active":true,"translations":[]}`)
	inactive := saveService(t, d, `{"name":"Synthetic retired exam","priceMinor":3000,"modes":["video"],"active":false,"translations":[]}`)
	var foreign string
	if err := owner.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (id, name) VALUES ('e0000000-0000-4000-8000-00000000000e', 'Synthetic clinic E') RETURNING id)
		INSERT INTO charge_item_definition (organization_id, name, price_minor, currency, modes, active)
		SELECT id, 'Synthetic other exam', 4000, 'BAM', '{video}', true FROM o RETURNING id`).Scan(&foreign); err != nil {
		t.Fatal(err)
	}
	receipt := func(c *client, service string) (int, string) {
		rec := c.do(http.MethodPost, "/clinic/receipts", `{"kind":"receipt","appointmentId":"`+visit+`","paymentMethod":"cash","lines":[{"serviceId":"`+service+`","quantity":2,"discountMinor":1000}]}`)
		return rec.Code, rec.Body.String()
	}

	if code, body := receipt(d.admin, exam); code != http.StatusConflict || !strings.Contains(body, "billing_details_missing") {
		t.Errorf("no billing details: %d %s, want 409 billing_details_missing", code, body)
	}
	details := `{"vatRegistered":false,"legalName":"Synthetic clinic d.o.o.","taxId":"4200000000001","billingEmail":"billing@example.test"}`
	if rec := d.admin.do(http.MethodPut, "/clinic/billing-details", details); rec.Code != http.StatusOK {
		t.Fatalf("PUT /clinic/billing-details = %d %s", rec.Code, rec.Body)
	}
	if code, body := receipt(d.admin, exam); code != http.StatusConflict || !strings.Contains(body, "billing_details_missing") {
		t.Errorf("billing details without an address: %d %s, want 409 billing_details_missing", code, body)
	}
	exec(`UPDATE organization SET region_id = 'BA-09', city_id = 'Q1270190', location_since = now(), address_line = 'Synthetic street 1', postal_code = '71000' WHERE id = $1`, inviteClinic)

	for name, service := range map[string]string{"unknown": "00000000-0000-4000-8000-0000000000ff", "inactive": inactive, "another clinic's": foreign} {
		if code, body := receipt(d.admin, service); code != http.StatusBadRequest || !strings.Contains(body, "invalid_receipt") {
			t.Errorf("%s service: %d %s, want 400 invalid_receipt", name, code, body)
		}
	}
	code, body := receipt(d.practitioner, exam) // the visit's own practitioner issues it
	if code != http.StatusCreated {
		t.Fatalf("POST /clinic/receipts = %d %s, want 201", code, body)
	}
	var got struct {
		ID         string
		TotalMinor int64
		Number     int64
	}
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatal(err)
	}
	if got.Number != 1 || got.TotalMinor != 9000 {
		t.Errorf("receipt = %s, want no. 1 of 90.00 BAM", body)
	}
	var legalName, taxID, address string
	if err := owner.QueryRow(t.Context(), `SELECT seller_legal_name, seller_tax_id, seller_address FROM invoice WHERE id = $1`, got.ID).Scan(&legalName, &taxID, &address); err != nil {
		t.Fatal(err)
	}
	if legalName != "Synthetic clinic d.o.o." || taxID != "4200000000001" || !strings.HasPrefix(address, "Synthetic street 1, 71000 ") || address == "Synthetic street 1, 71000 " {
		t.Errorf("seller = %q %q %q, want the clinic's billing details and its address with the city", legalName, taxID, address)
	}
	if strings.Contains(log.String(), "4200000000001") || strings.Contains(log.String(), "Synthetic street") {
		t.Error("the log carries the seller's details")
	}
}

// The receipts' service sees any appointment of the clinic through appointment.Lookup against Postgres (C tested a
// fake only): the admin issues for a visit the admin is not part of; a visit of another clinic is not found.
func TestNew_TheReceiptsSeeEveryVisitOfTheClinicAndNoOther(t *testing.T) {
	cfg := testConfig(t)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(app.Close)
	owner := ownerPool(t, cfg)
	d := newDirectory(t, app.Handler, owner)
	visit := bookedVisit(t, d, owner)
	if _, err := owner.Exec(t.Context(), `UPDATE organization SET legal_name = 'Synthetic clinic d.o.o.', tax_id = '4200000000001',
		region_id = 'BA-09', city_id = 'Q1270190', location_since = now(), address_line = 'Synthetic street 1' WHERE id = $1`, inviteClinic); err != nil {
		t.Fatal(err)
	}
	exam := saveService(t, d, `{"name":"Synthetic exam","priceMinor":5000,"modes":["video","in-person"],"active":true,"translations":[]}`)
	// A visit of another clinic, of the same patient: the receipts' service must not find it from this clinic.
	var foreignVisit string
	if err := owner.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (id, name) VALUES ('e0000000-0000-4000-8000-00000000000e', 'Synthetic clinic E') RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Ea', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id, specialty_code) SELECT organization_id, id, '419192003' FROM p RETURNING id, organization_id)
		INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
		SELECT a.patient_id, r.id, r.organization_id, gen_random_uuid(), tstzrange(now() - interval '2 days', now() - interval '2 days' + interval '30 minutes'), 'fulfilled'
		FROM r, appointment a WHERE a.id = $1 RETURNING id::text`, visit).Scan(&foreignVisit); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		appointment string
		status      int
	}{
		"the clinic's visit":     {appointment: visit, status: http.StatusCreated},
		"another clinic's visit": {appointment: foreignVisit, status: http.StatusNotFound},
	} {
		rec := d.admin.do(http.MethodPost, "/clinic/receipts", `{"kind":"receipt","appointmentId":"`+c.appointment+`","paymentMethod":"card","lines":[{"serviceId":"`+exam+`","quantity":1}]}`)
		if rec.Code != c.status {
			t.Errorf("%s: POST /clinic/receipts = %d %s, want %d", name, rec.Code, rec.Body, c.status)
		}
	}
}

// bookedVisit books a visit at the clinic for the directory's patient with the directory's practitioner, in person, and
// returns its id; the admin is not part of it.
func bookedVisit(t *testing.T, d directory, owner *pgxpool.Pool) string {
	t.Helper()
	slot := seedSlots(t, owner, d.roleID, 24*time.Hour)[0]
	if _, err := owner.Exec(t.Context(), `UPDATE slot SET visit_mode = 'in-person' WHERE id = $1`, slot); err != nil {
		t.Fatal(err)
	}
	if rec := d.patient.do(http.MethodPut, "/me/patient", `{"givenName":"Zorana","familyName":"Syntheticova","birthDate":"1977-07-07","phone":"+38760000777"}`); rec.Code != http.StatusCreated {
		t.Fatalf("PUT /me/patient = %d %s", rec.Code, rec.Body)
	}
	if rec := d.patient.do(http.MethodPost, "/consents", `{"organizationId":"`+inviteClinic+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST /consents = %d %s", rec.Code, rec.Body)
	}
	rec := d.patient.do(http.MethodPost, "/appointments", `{"slotId":"`+slot+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /appointments = %d %s", rec.Code, rec.Body)
	}
	var booked struct{ ID string }
	if err := json.Unmarshal(rec.Body.Bytes(), &booked); err != nil {
		t.Fatal(err)
	}
	return booked.ID
}

// saveService adds a service to the clinic's price list as its admin and returns its id.
func saveService(t *testing.T, d directory, body string) string {
	t.Helper()
	rec := d.admin.do(http.MethodPost, "/clinic/services", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /clinic/services = %d %s", rec.Code, rec.Body)
	}
	var s struct{ ID string }
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	return s.ID
}
