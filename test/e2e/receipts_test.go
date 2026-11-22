//go:build e2e

package e2e_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// receipt is the part of a receipt the journey reads.
type receipt struct {
	ID, FiscalStatus, FiscalID, ReversesID, ReversedByID, PatientID string
	Number, TotalMinor                                              int64
}

// TestReceipts_AVisitAtTheClinicIsReceiptedReportedAndReversed (docs/plans/clinic-cms.md, packages A to D): the admin
// keeps a price list and the clinic's billing details, a patient comes to the clinic, the admin issues the visit's
// receipt, the worker reports it to the (fake) tax authority, the admin reverses it, the reversal is reported too, the
// accountant's report sums both to zero, and the patient sees both receipts as their own.
func TestReceipts_AVisitAtTheClinicIsReceiptedReportedAndReversed(t *testing.T) {
	clinic := newBookableClinic(t)
	slot := atTheClinic(t, clinic)
	p := newPatientWithProfile(t)
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id}, nil)
	var visit struct{ ID string }
	p.must(http.StatusCreated, http.MethodPost, "/appointments", map[string]string{"slotId": slot.ID}, &visit)

	var service struct{ ID string }
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/services", map[string]any{
		"name": "Placeholder examination", "priceMinor": 5000, "modes": []string{"in-person"}, "active": true, "translations": []any{},
	}, &service)
	issue := map[string]any{
		"kind": "receipt", "appointmentId": visit.ID, "paymentMethod": "cash",
		"lines": []map[string]any{{"serviceId": service.ID, "quantity": 2, "discountMinor": 500}},
	}
	if code, body := clinic.admin.refusal(http.MethodPost, "/clinic/receipts", issue); code != http.StatusConflict || !strings.Contains(body, "billing_details_missing") {
		t.Errorf("a receipt before the billing details: %d %s, want 409 billing_details_missing", code, body)
	}
	clinic.admin.must(http.StatusOK, http.MethodPut, "/clinic/billing-details", map[string]any{
		"legalName": "E2E synthetic clinic d.o.o.", "taxId": "4200000000001", "vatRegistered": false, "billingEmail": "billing@example.test",
	}, nil)

	var issued receipt
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/receipts", issue, &issued)
	if issued.Number != 1 || issued.TotalMinor != 9500 || issued.FiscalStatus != "pending" || issued.PatientID != p.patientID {
		t.Errorf("POST /clinic/receipts = %+v, want no. 1 of 95.00 BAM to the patient, pending", issued)
	}
	fiscalized(t, clinic, issued.ID)
	if code := clinic.admin.do(http.MethodGet, "/clinic/receipts/"+issued.ID+"/pdf", nil, nil); code != http.StatusOK {
		t.Errorf("GET /clinic/receipts/%s/pdf = %d, want 200", issued.ID, code)
	}

	var reversal receipt
	clinic.admin.must(http.StatusCreated, http.MethodPost, "/clinic/receipts/"+issued.ID+"/reverse", map[string]string{"paymentMethod": "cash"}, &reversal)
	if reversal.Number != 2 || reversal.ReversesID != issued.ID || reversal.TotalMinor != -9500 {
		t.Errorf("the reversal = %+v, want no. 2 of -95.00 BAM reversing no. 1", reversal)
	}
	fiscalized(t, clinic, reversal.ID)
	var original receipt
	clinic.admin.must(http.StatusOK, http.MethodGet, "/clinic/receipts/"+issued.ID, nil, &original)
	if original.ReversedByID != reversal.ID {
		t.Errorf("receipt no. 1 reversedById %q, want %q", original.ReversedByID, reversal.ID)
	}
	if code, body := clinic.admin.refusal(http.MethodPost, "/clinic/receipts/"+issued.ID+"/reverse", map[string]string{"paymentMethod": "cash"}); code != http.StatusConflict {
		t.Errorf("reversing it again: %d %s, want 409", code, body)
	}

	now := clock.System{}.Now().UTC()
	q := url.Values{"from": {now.Add(-time.Hour).Format(time.RFC3339)}, "to": {now.Add(time.Hour).Format(time.RFC3339)}}
	var report struct{ Count, TotalMinor int64 }
	clinic.admin.must(http.StatusOK, http.MethodGet, "/clinic/reports/receipts?"+q.Encode(), nil, &report)
	if report.Count != 2 || report.TotalMinor != 0 {
		t.Errorf("the report = %+v, want two receipts summing to 0", report)
	}

	var mine []receipt
	p.must(http.StatusOK, http.MethodGet, "/me/receipts", nil, &mine)
	if len(mine) != 2 {
		t.Errorf("GET /me/receipts holds %d receipts, want the receipt and its reversal", len(mine))
	}
}

// fiscalized waits until the worker's invoice_fiscalize job has reported the receipt through the fake fiscal reporter,
// whose identifiers start with TEST-.
func fiscalized(t *testing.T, clinic *bookableClinic, id string) {
	t.Helper()
	eventually(t, "the invoice_fiscalize job reports receipt "+id, func() bool {
		var r receipt
		clinic.admin.must(http.StatusOK, http.MethodGet, "/clinic/receipts/"+id, nil, &r)
		if r.FiscalStatus == "failed" {
			t.Fatalf("receipt %s: the fiscal report failed", id)
		}
		return r.FiscalStatus == "fiscalized" && strings.HasPrefix(r.FiscalID, "TEST-")
	})
}
