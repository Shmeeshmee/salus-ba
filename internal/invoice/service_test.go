package invoice_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/invoice"
)

// The price list's service is the Prices port, wired in as it is (no adapter).
var _ invoice.Prices = (*chargeitemdefinition.Service)(nil)

// A receipt for a visit: the visit's patient, the price list's lines copied with their totals, the clinic's next
// number, the seller as it is now, pending, its report enqueued in the same transaction, audited as a create.
func TestIssue_NumbersCopiesTheSellerAndTheLinesAndEnqueuesTheReport(t *testing.T) {
	f := newFixture(theSeller)
	got, err := f.svc.Issue(t.Context(), adminC, draft())
	if err != nil {
		t.Fatalf("Issue() = %v", err)
	}
	if got.Number != 1 || got.PatientID != patientA || got.AppointmentID != visit || got.OrganizationID != org ||
		got.IssuedByAccountID != admin || !got.IssuedAt.Equal(at) || got.FiscalStatus != invoice.FiscalPending {
		t.Errorf("Issue() = %+v", got)
	}
	if got.SellerLegalName != theSeller.LegalName || got.SellerTaxID != theSeller.TaxID || got.SellerAddress != theSeller.Address {
		t.Errorf("seller = %q %q %q, want the clinic's at issue", got.SellerLegalName, got.SellerTaxID, got.SellerAddress)
	}
	if len(got.Lines) != 2 || got.Lines[0].TotalMinor != 9500 || got.Lines[1].TotalMinor != 1170 || got.TotalMinor != 10670 ||
		got.Currency != "BAM" || got.Lines[1].VATRateBP == nil || *got.Lines[1].VATRateBP != 1700 || got.Lines[0].Name == "" {
		t.Errorf("lines = %+v, total %d %s", got.Lines, got.TotalMinor, got.Currency)
	}
	if len(f.jobs.enqueued) != 1 || f.jobs.enqueued[0] != got.ID {
		t.Errorf("enqueued = %v, want the receipt's report", f.jobs.enqueued)
	}
	e := f.audit.last()
	if e.Subtype.Code != auditevent.CodeCreate || e.EntityReference != "Invoice/"+got.ID || e.AgentAccountID != admin ||
		e.OrganizationID != org || e.Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit = %+v", e)
	}
	second, err := f.svc.Issue(t.Context(), doctorC, draft()) // the practitioner of the visit may issue too
	if err != nil || second.Number != 2 {
		t.Errorf("the practitioner's Issue() = number %d, %v; want 2", second.Number, err)
	}
}

// A receipt is still issued for a visit of a patient whose record the clinic restricts: the fiscal law obliges it
// (docs/clinic-cms/WAVE5.md S7, Art. 18(2)).
func TestIssue_AReceiptForARestrictedRecord(t *testing.T) {
	f := newFixture(theSeller)
	d := draft()
	d.AppointmentID = restrictedVisit
	if got, err := f.svc.Issue(t.Context(), adminC, d); err != nil || got.PatientID != "patient-restricted" {
		t.Errorf("Issue() = %+v, %v; want the receipt", got, err)
	}
}

// The seller on a receipt is the clinic's as it was at issue, also after the clinic changes its details.
func TestIssue_KeepsTheSellerAsItWasAtIssue(t *testing.T) {
	f := newFixture(theSeller)
	first, err := f.svc.Issue(t.Context(), adminC, draft())
	if err != nil {
		t.Fatal(err)
	}
	moved := newFixture(invoice.Seller{LegalName: "Renamed clinic d.o.o.", TaxID: theSeller.TaxID, Address: "Another placeholder street 2"})
	moved.repo.invoices, moved.repo.last = f.repo.invoices, f.repo.last
	got, err := moved.svc.Read(t.Context(), adminC, first.ID)
	if err != nil || got.SellerLegalName != theSeller.LegalName || got.SellerAddress != theSeller.Address {
		t.Errorf("Read() = %q, %q, %v; want the seller at issue", got.SellerLegalName, got.SellerAddress, err)
	}
	rev, err := moved.svc.Reverse(t.Context(), adminC, first.ID, invoice.PaymentCash)
	if err != nil || rev.SellerLegalName != theSeller.LegalName {
		t.Errorf("Reverse() seller = %q, %v; want the reversed receipt's", rev.SellerLegalName, err)
	}
}

// refusal is a draft Issue refuses, with whether the refusal is audited.
type refusal struct {
	want    error
	name    string
	seller  invoice.Seller
	caller  invoice.Caller
	draft   invoice.Draft
	audited bool
}

func TestIssue_Refuses(t *testing.T) {
	with := func(change func(*invoice.Draft)) invoice.Draft {
		d := draft()
		change(&d)
		return d
	}
	onItsOwn := func(d *invoice.Draft) { d.AppointmentID = "" }
	tests := []refusal{
		{name: "no lines", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) { d.Picks = nil })},
		{name: "a quantity above the cap", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) { d.Picks[0].Quantity = 10001 })},
		{name: "more lines than the cap", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) {
			for len(d.Picks) <= 100 {
				d.Picks = append(d.Picks, d.Picks[1])
			}
		})},
		{name: "a quantity of zero", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) { d.Picks[0].Quantity = 0 })},
		{name: "a negative discount", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) { d.Picks[0].DiscountMinor = -1 })},
		{name: "a discount above the line", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) { d.Picks[1].DiscountMinor = 1171 })},
		{name: "an unknown payment method", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) { d.PaymentMethod = "barter" })},
		{name: "an unknown kind", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) { d.Kind = "note" })},
		{name: "a patient and a company", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) {
			d.Kind, d.PatientID, d.CompanyName = invoice.KindInvoice, patientA, "Synthetic company d.o.o."
		})},
		{name: "an invoice without a company", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) { d.Kind = invoice.KindInvoice })},
		{name: "a company's tax id that is none", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) {
			d.Kind, d.CompanyName, d.CompanyTaxID = invoice.KindInvoice, "Synthetic company d.o.o.", "12"
		})},
		{name: "a service off the price list", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) { d.Picks[0].DefinitionID = "svc-other" })},
		{name: "another patient than the visit's", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) { d.PatientID = "patient-b" })},
		{name: "on its own, a patient the clinic does not treat", caller: adminC, seller: theSeller, want: invoice.ErrInvalid, draft: with(func(d *invoice.Draft) {
			onItsOwn(d)
			d.PatientID = "patient-stranger"
		})},
		{name: "a visit the clinic does not have", caller: adminC, seller: theSeller, want: invoice.ErrNotFound, audited: true, draft: with(func(d *invoice.Draft) { d.AppointmentID = "appt-other" })},
		{name: "a patient", caller: patientC, seller: theSeller, want: invoice.ErrForbidden, audited: true, draft: draft()},
		{name: "a practitioner without a visit", caller: doctorC, seller: theSeller, want: invoice.ErrForbidden, audited: true, draft: with(onItsOwn)},
		{name: "a practitioner of another visit", caller: invoice.Caller{AccountID: "acct-other", OrganizationID: org, PractitionerID: "pract-2"}, seller: theSeller, want: invoice.ErrForbidden, audited: true, draft: draft()},
		{name: "staff who are neither", caller: invoice.Caller{AccountID: "acct-x", OrganizationID: org}, seller: theSeller, want: invoice.ErrForbidden, audited: true, draft: draft()},
		{name: "a clinic without a tax id", caller: adminC, seller: invoice.Seller{LegalName: "Synthetic clinic d.o.o.", Address: "Placeholder street 1"}, want: invoice.ErrSellerIncomplete, draft: draft()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(tt.seller)
			if _, err := f.svc.Issue(t.Context(), tt.caller, tt.draft); !errors.Is(err, tt.want) {
				t.Fatalf("Issue() = %v, want %v", err, tt.want)
			}
			if len(f.repo.invoices) != 0 || f.repo.last != 0 || len(f.jobs.enqueued) != 0 {
				t.Errorf("a refused receipt left %d receipts, number %d, %d jobs", len(f.repo.invoices), f.repo.last, len(f.jobs.enqueued))
			}
			if audited := len(f.audit.events) == 1 && f.audit.last().Outcome == auditevent.OutcomeMinorFailure; audited != tt.audited {
				t.Errorf("audit = %+v, want a refusal recorded: %v", f.audit.events, tt.audited)
			}
		})
	}
}

// A receipt on its own: to a patient the clinic treats (a visit, or an active care consent), to a company, or to
// nobody (an anonymous cash sale).
func TestIssue_OnItsOwn(t *testing.T) {
	f := newFixture(theSeller)
	for name, change := range map[string]func(*invoice.Draft){
		"a patient with a visit":  func(d *invoice.Draft) { d.PatientID = patientA },
		"a patient who consented": func(d *invoice.Draft) { d.PatientID = "patient-consented" },
		"a company": func(d *invoice.Draft) {
			d.Kind, d.CompanyName, d.CompanyTaxID = invoice.KindInvoice, "Synthetic company d.o.o.", "4200000000001"
		},
		"an anonymous cash sale": func(*invoice.Draft) {},
	} {
		d := draft()
		d.AppointmentID = ""
		change(&d)
		got, err := f.svc.Issue(t.Context(), adminC, d)
		if err != nil || got.PatientID != d.PatientID || got.CompanyName != d.CompanyName || got.AppointmentID != "" {
			t.Errorf("%s: Issue() = %+v, %v", name, got, err)
		}
	}
}

// A reversal negates the lines and the total, takes the next number, names the receipt it reverses, and is issued
// once: never a second time, and never for a reversal.
// A receipt whose fiscal report failed is not reversed: the reversal could never be reported either (it names the
// receipt's fiscal number), so it would take a number of the sequence for good and stay unreportable.
func TestReverse_RefusesAReceiptWhoseReportFailed(t *testing.T) {
	f := newFixture(theSeller)
	orig, err := f.svc.Issue(t.Context(), adminC, draft())
	if err != nil {
		t.Fatal(err)
	}
	for i := range f.repo.invoices {
		if f.repo.invoices[i].ID == orig.ID {
			f.repo.invoices[i].FiscalStatus = invoice.FiscalFailed
		}
	}
	if _, err := f.svc.Reverse(t.Context(), adminC, orig.ID, invoice.PaymentCash); !errors.Is(err, invoice.ErrReportFailed) {
		t.Errorf("Reverse() of a receipt whose report failed = %v, want ErrReportFailed", err)
	}
	if len(f.repo.invoices) != 1 {
		t.Errorf("%d receipts stored, want only the original", len(f.repo.invoices))
	}
}

func TestReverse_NegatesOnceAndNeverAReversal(t *testing.T) {
	f := newFixture(theSeller)
	orig, err := f.svc.Issue(t.Context(), adminC, draft())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Reverse(t.Context(), doctorC, orig.ID, invoice.PaymentCash); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("a practitioner's Reverse() = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.Reverse(t.Context(), adminC, orig.ID, "barter"); !errors.Is(err, invoice.ErrInvalid) {
		t.Errorf("Reverse() with an unknown method = %v, want ErrInvalid", err)
	}
	if _, err := f.svc.Reverse(t.Context(), adminC, "inv-none", invoice.PaymentCash); !errors.Is(err, invoice.ErrNotFound) {
		t.Errorf("Reverse() of no receipt = %v, want ErrNotFound", err)
	}
	rev, err := f.svc.Reverse(t.Context(), adminC, orig.ID, invoice.PaymentCard)
	if err != nil {
		t.Fatalf("Reverse() = %v", err)
	}
	if rev.Number != 2 || rev.ReversesID != orig.ID || rev.TotalMinor != -orig.TotalMinor || rev.PatientID != orig.PatientID ||
		rev.AppointmentID != orig.AppointmentID || rev.PaymentMethod != invoice.PaymentCard || rev.Kind != orig.Kind {
		t.Errorf("Reverse() = %+v", rev)
	}
	for i, l := range rev.Lines {
		o := orig.Lines[i]
		if l.Quantity != -o.Quantity || l.TotalMinor != -o.TotalMinor || l.UnitPriceMinor != o.UnitPriceMinor ||
			l.DiscountMinor != o.DiscountMinor || l.Name != o.Name || l.ChargeItemDefinitionID != o.ChargeItemDefinitionID {
			t.Errorf("line %d = %+v, want %+v negated", i, l, o)
		}
	}
	if len(f.jobs.enqueued) != 2 || f.jobs.enqueued[1] != rev.ID {
		t.Errorf("enqueued = %v, want the reversal's report too", f.jobs.enqueued)
	}
	if _, err := f.svc.Reverse(t.Context(), adminC, orig.ID, invoice.PaymentCash); !errors.Is(err, invoice.ErrAlreadyReversed) {
		t.Errorf("a second Reverse() = %v, want ErrAlreadyReversed", err)
	}
	if _, err := f.svc.Reverse(t.Context(), adminC, rev.ID, invoice.PaymentCash); !errors.Is(err, invoice.ErrAlreadyReversed) {
		t.Errorf("Reverse() of a reversal = %v, want ErrAlreadyReversed", err)
	}
	if len(f.repo.invoices) != 2 || f.repo.last != 2 {
		t.Errorf("%d receipts, last number %d; want 2 and 2 (no gap)", len(f.repo.invoices), f.repo.last)
	}
	got, _ := f.svc.Read(t.Context(), adminC, orig.ID)
	if got.ReversedByID != rev.ID {
		t.Errorf("the receipt's ReversedByID = %q, want %q", got.ReversedByID, rev.ID)
	}
}

// The clinic's admin reads every receipt of the clinic, a practitioner the ones they issued, another clinic none;
// a patient their own. Every read is audited.
func TestReadListAndMine_ShowOnlyWhatTheCallerMaySee(t *testing.T) {
	f := newFixture(theSeller)
	byDoctor, _ := f.svc.Issue(t.Context(), doctorC, draft())
	sale := draft()
	sale.AppointmentID = ""
	byAdmin, err := f.svc.Issue(t.Context(), adminC, sale)
	if err != nil {
		t.Fatal(err)
	}
	other := invoice.Caller{AccountID: "acct-z", OrganizationID: "org-2", Admin: true}
	for name, tt := range map[string]struct {
		want   error
		id     string
		caller invoice.Caller
	}{
		"the admin":                           {caller: adminC, id: byDoctor.ID},
		"the practitioner who issued it":      {caller: doctorC, id: byDoctor.ID},
		"a practitioner who did not issue it": {caller: doctorC, id: byAdmin.ID, want: invoice.ErrNotFound},
		"another clinic's admin":              {caller: other, id: byDoctor.ID, want: invoice.ErrNotFound},
		"a patient on the clinic's route":     {caller: patientC, id: byDoctor.ID, want: invoice.ErrNotFound},
	} {
		got, err := f.svc.Read(t.Context(), tt.caller, tt.id)
		if !errors.Is(err, tt.want) || (err == nil && got.ID != tt.id) {
			t.Errorf("%s: Read() = %q, %v; want %v", name, got.ID, err, tt.want)
		}
		if e := f.audit.last(); e.Subtype.Code != auditevent.CodeRead || (e.Outcome == auditevent.OutcomeSuccess) != (tt.want == nil) {
			t.Errorf("%s: audit = %+v", name, e)
		}
	}
	day := invoice.Period{From: at.Add(-time12h), To: at.Add(time12h)}
	list, err := f.svc.List(t.Context(), adminC, day)
	if err != nil || len(list) != 2 || list[0].ID != byAdmin.ID {
		t.Errorf("List() = %d receipts, %v; want 2, newest first", len(list), err)
	}
	if _, err := f.svc.List(t.Context(), doctorC, day); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("a practitioner's List() = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.List(t.Context(), adminC, invoice.Period{From: at, To: at}); !errors.Is(err, invoice.ErrInvalid) {
		t.Errorf("List() of an empty period = %v, want ErrInvalid", err)
	}
	theirs, err := f.svc.OfClinicPatient(t.Context(), adminC, patientC.PatientID)
	if err != nil || len(theirs) != 1 || theirs[0].ID != byDoctor.ID {
		t.Errorf("OfClinicPatient() = %d receipts, %v; want the patient's one at the clinic", len(theirs), err)
	}
	if e := f.audit.last(); e.Subtype.Code != auditevent.CodeSearchType || e.AgentAccountID != admin {
		t.Errorf("OfClinicPatient() audit = %+v", e)
	}
	if got, err := f.svc.OfClinicPatient(t.Context(), other, patientC.PatientID); err != nil || len(got) != 0 {
		t.Errorf("another clinic's OfClinicPatient() = %d receipts, %v; want none", len(got), err)
	}
	if _, err := f.svc.OfClinicPatient(t.Context(), doctorC, patientC.PatientID); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("a practitioner's OfClinicPatient() = %v, want ErrForbidden", err)
	}
	mine, err := f.svc.Mine(t.Context(), patientC)
	if err != nil || len(mine) != 1 || mine[0].ID != byDoctor.ID {
		t.Errorf("Mine() = %d receipts, %v; want the patient's one", len(mine), err)
	}
	if e := f.audit.last(); e.Purpose.Code != auditevent.CodePATRQT || e.AgentAccountID != patientC.AccountID {
		t.Errorf("Mine() audit = %+v", e)
	}
	if _, err := f.svc.Mine(t.Context(), adminC); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("staff's Mine() = %v, want ErrForbidden", err)
	}
}

// A draft issued twice with the same idempotency key (a double click, a retried request) issues one receipt: the
// second answer is the first receipt, read and audited as a read. A key issued by another account is refused.
func TestIssue_WithTheSameIdempotencyKeyIssuesOneReceipt(t *testing.T) {
	f := newFixture(theSeller)
	d := draft()
	d.IdempotencyKey = "e1000000-0000-4000-8000-0000000000e1"
	first, err := f.svc.Issue(t.Context(), adminC, d)
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.svc.Issue(t.Context(), adminC, d)
	if err != nil || again.ID != first.ID || again.Number != first.Number {
		t.Errorf("the second Issue() = %+v, %v; want the first receipt", again, err)
	}
	if len(f.repo.invoices) != 1 || len(f.jobs.enqueued) != 1 {
		t.Errorf("stored %d receipts and enqueued %d reports, want one each", len(f.repo.invoices), len(f.jobs.enqueued))
	}
	if e := f.audit.last(); e.Subtype.Code != auditevent.CodeRead || e.EntityReference != "Invoice/"+first.ID {
		t.Errorf("audit = %+v, want the receipt's read", e)
	}
	if _, err := f.svc.Issue(t.Context(), doctorC, d); !errors.Is(err, invoice.ErrInvalid) {
		t.Errorf("the key from another account = %v, want ErrInvalid", err)
	}
	d.IdempotencyKey = "e2000000-0000-4000-8000-0000000000e2"
	if other, err := f.svc.Issue(t.Context(), adminC, d); err != nil || other.Number != 2 {
		t.Errorf("another key = number %d, %v; want a second receipt", other.Number, err)
	}
	if f.repo.invoices[0].IdempotencyKey != "e1000000-0000-4000-8000-0000000000e1" {
		t.Errorf("stored key = %q", f.repo.invoices[0].IdempotencyKey)
	}
}

// The front desk (docs/clinic-cms W2, F1): a receptionist issues a receipt for any visit of the clinic, and reads and
// prints the receipts they issued; the list of all receipts, a reversal and a retried report stay the admin's.
func TestReception_IssuesReadsAndPrintsTheirReceiptsAndNothingElse(t *testing.T) {
	f := newFixture(theSeller)
	desk := invoice.Caller{AccountID: "acct-desk", OrganizationID: org, Reception: true}
	issued, err := f.svc.Issue(t.Context(), desk, draft())
	if err != nil {
		t.Fatalf("the receptionist's Issue() = %v", err)
	}
	if issued.IssuedByAccountID != "acct-desk" || f.audit.last().AgentAccountID != "acct-desk" {
		t.Errorf("Issue() = %+v, audit %+v; want the receptionist as the issuer and the agent", issued, f.audit.last())
	}
	byAdmin, err := f.svc.Issue(t.Context(), adminC, draft())
	if err != nil {
		t.Fatal(err)
	}
	if got, err := f.svc.Read(t.Context(), desk, issued.ID); err != nil || got.ID != issued.ID {
		t.Errorf("Read() of their own receipt = %q, %v", got.ID, err)
	}
	if _, err := f.svc.PDF(t.Context(), desk, issued.ID); err != nil {
		t.Errorf("PDF() of their own receipt = %v", err)
	}
	// A visit's receipt a colleague issued: the desk shows it on the visit's row and prints it (docs/clinic-cms/walk/
	// wave3, findings 1 and 2). A receipt on its own a colleague issued stays closed to them.
	if got, err := f.svc.Read(t.Context(), desk, byAdmin.ID); err != nil || got.ID != byAdmin.ID {
		t.Errorf("Read() of the admin's receipt of a visit = %q, %v; want it", got.ID, err)
	}
	sale := draft()
	sale.AppointmentID = ""
	onItsOwn, err := f.svc.Issue(t.Context(), adminC, sale)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Read(t.Context(), desk, onItsOwn.ID); !errors.Is(err, invoice.ErrNotFound) {
		t.Errorf("Read() of the admin's receipt on its own = %v, want ErrNotFound", err)
	}
	day := invoice.Period{From: at.Add(-time12h), To: at.Add(time12h)}
	if _, err := f.svc.List(t.Context(), desk, day); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("List() = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.Reverse(t.Context(), desk, issued.ID, invoice.PaymentCash); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("Reverse() = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.Retry(t.Context(), desk, issued.ID); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("Retry() = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.Report(t.Context(), desk, day); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("Report() = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.Quotes(t.Context(), desk, ""); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("Quotes() = %v, want ErrForbidden", err)
	}
}
