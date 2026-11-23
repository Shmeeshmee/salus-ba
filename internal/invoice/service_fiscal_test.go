package invoice_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
)

const time12h = 12 * time.Hour

// The fiscal job reports a receipt once: run twice it reports once, and a receipt whose transaction rolled back has
// no job and no report (the fake counts every call).
func TestFiscalize_ReportsOnceHoweverOftenItRuns(t *testing.T) {
	f := newFixture(theSeller)
	bad := draft()
	bad.Picks[0].DefinitionID = "svc-other" // rolled back: no receipt, no number, no job
	if _, err := f.svc.Issue(t.Context(), adminC, bad); err == nil {
		t.Fatal("Issue() of a service off the price list succeeded")
	}
	inv, err := f.svc.Issue(t.Context(), adminC, draft())
	if err != nil || inv.Number != 1 || len(f.jobs.enqueued) != 1 {
		t.Fatalf("Issue() = number %d, %v, jobs %v", inv.Number, err, f.jobs.enqueued)
	}
	for range 2 {
		if err := f.svc.Fiscalize(t.Context(), org, inv.ID); err != nil {
			t.Fatalf("Fiscalize() = %v", err)
		}
	}
	if got := f.reporter.Reported(inv.ID); got != 1 {
		t.Errorf("reported %d times, want once", got)
	}
	got, _ := f.svc.Read(t.Context(), adminC, inv.ID)
	if got.FiscalStatus != invoice.FiscalFiscalized || got.FiscalID == "" || got.FiscalCode == "" || got.FiscalQR == "" || !got.FiscalizedAt.Equal(at) {
		t.Errorf("after Fiscalize() = %+v", got)
	}
}

// An authority that cannot be reached leaves the receipt pending and fails the job, which River retries; when the job
// gives up the receipt is failed, and a later report still fiscalizes it. A fiscalized receipt never changes.
func TestFiscalize_FailedMayStillBecomeFiscalized(t *testing.T) {
	f := newFixture(theSeller)
	inv, _ := f.svc.Issue(t.Context(), adminC, draft())
	down := errors.New("synthetic outage")
	f.reporter.Err = down
	if err := f.svc.Fiscalize(t.Context(), org, inv.ID); !errors.Is(err, down) {
		t.Fatalf("Fiscalize() = %v, want the outage", err)
	}
	if got, _ := f.svc.Read(t.Context(), adminC, inv.ID); got.FiscalStatus != invoice.FiscalPending {
		t.Errorf("status = %s, want pending while the job retries", got.FiscalStatus)
	}
	if err := f.svc.GiveUp(t.Context(), org, inv.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.svc.Read(t.Context(), adminC, inv.ID); got.FiscalStatus != invoice.FiscalFailed {
		t.Errorf("status = %s, want failed after the job gave up", got.FiscalStatus)
	}
	f.reporter.Err = nil
	if err := f.svc.Fiscalize(t.Context(), org, inv.ID); err != nil {
		t.Fatal(err)
	}
	done, _ := f.svc.Read(t.Context(), adminC, inv.ID)
	if done.FiscalStatus != invoice.FiscalFiscalized || done.FiscalQR == "" {
		t.Errorf("a retried report = %+v, want fiscalized with its QR", done)
	}
	if err := f.svc.GiveUp(t.Context(), org, inv.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.svc.Read(t.Context(), adminC, inv.ID); got.FiscalStatus != invoice.FiscalFiscalized || got.FiscalID != done.FiscalID {
		t.Errorf("a fiscalized receipt changed: %+v", got)
	}
}

// A receipt the authority rejects for good is failed at once and the job ends: a retry would fail the same way.
func TestFiscalize_ARejectedReceiptIsFailedAndNotRetried(t *testing.T) {
	f := newFixture(theSeller)
	inv, _ := f.svc.Issue(t.Context(), adminC, draft())
	f.reporter.Err = fiscal.ErrRejected
	if err := f.svc.Fiscalize(t.Context(), org, inv.ID); err != nil {
		t.Fatalf("Fiscalize() = %v, want the job to end", err)
	}
	if got, _ := f.svc.Read(t.Context(), adminC, inv.ID); got.FiscalStatus != invoice.FiscalFailed {
		t.Errorf("status = %s, want failed", got.FiscalStatus)
	}
}

// A reversal is reported with the fiscal id of the receipt it reverses, so it waits for that receipt's report.
func TestFiscalize_AReversalWaitsForItsReceipt(t *testing.T) {
	f := newFixture(theSeller)
	inv, _ := f.svc.Issue(t.Context(), adminC, draft())
	rev, err := f.svc.Reverse(t.Context(), adminC, inv.ID, invoice.PaymentCash)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Fiscalize(t.Context(), org, rev.ID); !errors.Is(err, invoice.ErrOriginalPending) {
		t.Fatalf("Fiscalize() of the reversal first = %v, want ErrOriginalPending", err)
	}
	if f.reporter.Reported(rev.ID) != 0 {
		t.Error("the reversal was reported before its receipt")
	}
	if err := f.svc.Fiscalize(t.Context(), org, inv.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Fiscalize(t.Context(), org, rev.ID); err != nil {
		t.Fatalf("Fiscalize() of the reversal = %v", err)
	}
	got, _ := f.svc.Read(t.Context(), adminC, rev.ID)
	if got.FiscalStatus != invoice.FiscalFiscalized || f.reporter.Fiscalized() != 2 {
		t.Errorf("the reversal = %s, %d fiscalized; want fiscalized, 2", got.FiscalStatus, f.reporter.Fiscalized())
	}
}

// The report equals the raw receipts: the count and total, by payment method and by service, with a reversal taking
// its receipt's amounts back.
func TestReport_EqualsTheReceipts(t *testing.T) {
	f := newFixture(theSeller)
	first, _ := f.svc.Issue(t.Context(), adminC, draft()) // cash, 106.70
	card := draft()
	card.PaymentMethod, card.Picks = invoice.PaymentCard, card.Picks[:1] // card, 95.00
	if _, err := f.svc.Issue(t.Context(), adminC, card); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Reverse(t.Context(), adminC, first.ID, invoice.PaymentCash); err != nil {
		t.Fatal(err)
	}
	day := invoice.Period{From: at.Add(-time12h), To: at.Add(time12h)}
	r, err := f.svc.Report(t.Context(), adminC, day)
	if err != nil {
		t.Fatal(err)
	}
	list, _ := f.svc.List(t.Context(), adminC, day)
	var total, lines int64
	for _, inv := range list {
		total += inv.TotalMinor
		for _, l := range inv.Lines {
			lines += l.TotalMinor
		}
	}
	if r.Count != len(list) || r.Count != 3 || r.TotalMinor != total || r.TotalMinor != 9500 || r.Currency != "BAM" || r.Period != day || r.Language != "en" {
		t.Errorf("Report() = %+v, want 3 receipts totalling %d", r, total)
	}
	want := []invoice.MethodTotal{{PaymentMethod: invoice.PaymentCash, Count: 2, TotalMinor: 0}, {PaymentMethod: invoice.PaymentCard, Count: 1, TotalMinor: 9500}}
	if len(r.ByPaymentMethod) != 2 || r.ByPaymentMethod[0] != want[0] || r.ByPaymentMethod[1] != want[1] {
		t.Errorf("by payment method = %+v, want %+v", r.ByPaymentMethod, want)
	}
	var services int64
	for _, s := range r.ByService {
		services += s.TotalMinor
	}
	if len(r.ByService) != 2 || services != lines || r.ByService[0].ChargeItemDefinitionID != exam || r.ByService[0].Quantity != 2 ||
		r.ByService[0].TotalMinor != 9500 || r.ByService[1].Quantity != 0 || r.ByService[1].TotalMinor != 0 {
		t.Errorf("by service = %+v, want the examination twice and the laboratory item taken back", r.ByService)
	}
	if _, err := f.svc.Report(t.Context(), doctorC, day); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("a practitioner's Report() = %v, want ErrForbidden", err)
	}
}

// The PDF is for whoever may read the receipt: the clinic's staff as Read decides, a patient their own.
func TestPDF_ForWhoeverMayReadTheReceipt(t *testing.T) {
	f := newFixture(theSeller)
	inv, _ := f.svc.Issue(t.Context(), adminC, draft())
	rev, _ := f.svc.Reverse(t.Context(), adminC, inv.ID, invoice.PaymentCash)
	for name, c := range map[string]invoice.Caller{"the admin": adminC, "the patient": patientC} {
		for _, id := range []string{inv.ID, rev.ID} {
			b, err := f.svc.PDF(t.Context(), c, id)
			if err != nil || string(b[:5]) != "%PDF-" {
				t.Errorf("%s: PDF(%s) = %v", name, id, err)
			}
			if e := f.audit.last(); e.EntityReference != "Invoice/"+id || e.AgentAccountID != c.AccountID {
				t.Errorf("%s: audit = %+v", name, e)
			}
		}
	}
	stranger := invoice.Caller{AccountID: "acct-s", PatientID: "patient-b"}
	if _, err := f.svc.PDF(t.Context(), stranger, inv.ID); !errors.Is(err, invoice.ErrNotFound) {
		t.Errorf("another patient's PDF() = %v, want ErrNotFound", err)
	}
	if _, err := f.svc.PDF(t.Context(), doctorC, inv.ID); !errors.Is(err, invoice.ErrNotFound) {
		t.Errorf("a practitioner who did not issue it: PDF() = %v, want ErrNotFound", err)
	}
}
