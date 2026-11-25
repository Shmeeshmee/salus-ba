package invoice_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// quoteDraft is a quote at the visit: two examinations with a discount and a laboratory item, with a note.
func quoteDraft() invoice.QuoteDraft {
	return invoice.QuoteDraft{
		PatientID: patientA, AppointmentID: visit, Note: "Placeholder note",
		Picks: []chargeitemdefinition.Pick{{DefinitionID: exam, Quantity: 2, DiscountMinor: 500}, {DefinitionID: lab, Quantity: 1}},
	}
}

// A quote: the price list's lines copied with their totals, valid 30 days, sent, its e-mail enqueued in the same
// transaction, audited as a create. The practitioner of the visit may send one too.
func TestSendQuote_PricesTheLinesForThirtyDaysAndEnqueuesTheEmail(t *testing.T) {
	f := newFixture(theSeller)
	got, err := f.svc.SendQuote(t.Context(), adminC, quoteDraft())
	if err != nil {
		t.Fatalf("SendQuote() = %v", err)
	}
	if got.ID == "" || got.OrganizationID != org || got.PatientID != patientA || got.AppointmentID != visit ||
		got.CreatedByAccountID != admin || got.Note != "Placeholder note" || got.State != invoice.QuoteSent {
		t.Errorf("SendQuote() = %+v", got)
	}
	if !got.IssuedAt.Equal(at) || !got.ValidUntil.Equal(at.Add(30*24*time.Hour)) {
		t.Errorf("issued %v, valid until %v; want %v and 30 days after", got.IssuedAt, got.ValidUntil, at)
	}
	if len(got.Lines) != 2 || got.Lines[0].TotalMinor != 9500 || got.Lines[1].TotalMinor != 1170 || got.TotalMinor != 10670 ||
		got.Currency != "BAM" || got.Lines[0].Name == "" || got.Lines[1].VATRateBP == nil {
		t.Errorf("lines = %+v, total %d %s", got.Lines, got.TotalMinor, got.Currency)
	}
	if len(f.jobs.emails) != 1 || f.jobs.emails[0] != visit {
		t.Errorf("e-mails = %v, want the visit's", f.jobs.emails)
	}
	e := f.audit.last()
	if e.Subtype.Code != auditevent.CodeCreate || e.EntityReference != "Quote/"+got.ID || e.AgentAccountID != admin ||
		e.OrganizationID != org || e.Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit = %+v", e)
	}
	if _, err := f.svc.SendQuote(t.Context(), doctorC, quoteDraft()); err != nil {
		t.Errorf("the visit's practitioner's SendQuote() = %v", err)
	}
}

// A patient the clinic registered has no account (docs/clinic-cms W1): their quote is stored and audited, and no e-mail
// is enqueued for it.
func TestSendQuote_EnqueuesNoEmailForAPatientWithoutAnAccount(t *testing.T) {
	f := newFixture(theSeller)
	d := quoteDraft()
	d.AppointmentID, d.PatientID = registeredVisit, "patient-registered"
	got, err := f.svc.SendQuote(t.Context(), adminC, d)
	if err != nil || got.ID == "" || got.PatientID != "patient-registered" {
		t.Fatalf("SendQuote() = %+v, %v; want the quote", got, err)
	}
	if len(f.jobs.emails) != 0 {
		t.Errorf("e-mails = %v, want none", f.jobs.emails)
	}
	if e := f.audit.last(); e.EntityReference != "Quote/"+got.ID || e.Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit = %+v, want the quote's create", e)
	}
}

// A quote is no receipt: it takes no number from the clinic's receipt sequence and enqueues no fiscal report, and the
// receipt that follows still gets number 1.
func TestSendQuote_TakesNoReceiptNumberAndEnqueuesNoFiscalReport(t *testing.T) {
	f := newFixture(theSeller)
	for range 3 {
		if _, err := f.svc.SendQuote(t.Context(), adminC, quoteDraft()); err != nil {
			t.Fatal(err)
		}
	}
	if f.repo.last != 0 || len(f.repo.invoices) != 0 {
		t.Errorf("after three quotes: last number %d, %d receipts; want none", f.repo.last, len(f.repo.invoices))
	}
	if len(f.jobs.enqueued) != 0 {
		t.Errorf("fiscal reports enqueued = %v, want none", f.jobs.enqueued)
	}
	receipt, err := f.svc.Issue(t.Context(), adminC, draft())
	if err != nil || receipt.Number != 1 {
		t.Errorf("the receipt after the quotes = number %d, %v; want 1", receipt.Number, err)
	}
}

func TestSendQuote_Refuses(t *testing.T) {
	with := func(change func(*invoice.QuoteDraft)) invoice.QuoteDraft {
		d := quoteDraft()
		change(&d)
		return d
	}
	for _, tc := range []struct {
		want    error
		name    string
		caller  invoice.Caller
		draft   invoice.QuoteDraft
		audited bool
	}{
		{name: "no lines", want: invoice.ErrInvalid, caller: adminC, draft: with(func(d *invoice.QuoteDraft) { d.Picks = nil })},
		{name: "no visit", want: invoice.ErrInvalid, caller: adminC, draft: with(func(d *invoice.QuoteDraft) { d.AppointmentID = "" })},
		{name: "no patient", want: invoice.ErrInvalid, caller: adminC, draft: with(func(d *invoice.QuoteDraft) { d.PatientID = "" })},
		{name: "a zero quantity", want: invoice.ErrInvalid, caller: adminC, draft: with(func(d *invoice.QuoteDraft) { d.Picks[0].Quantity = 0 })},
		{name: "a negative discount", want: invoice.ErrInvalid, caller: adminC, draft: with(func(d *invoice.QuoteDraft) { d.Picks[0].DiscountMinor = -1 })},
		{name: "a discount above its line", want: invoice.ErrInvalid, caller: adminC, draft: with(func(d *invoice.QuoteDraft) { d.Picks[1].DiscountMinor = 1171 })},
		{name: "a service off the price list", want: invoice.ErrInvalid, caller: adminC, draft: with(func(d *invoice.QuoteDraft) { d.Picks[0].DefinitionID = "svc-other" })},
		{name: "a note too long", want: invoice.ErrInvalid, caller: adminC, draft: with(func(d *invoice.QuoteDraft) { d.Note = strings.Repeat("n", 2001) })},
		{name: "a patient", want: invoice.ErrForbidden, caller: patientC, draft: quoteDraft(), audited: true},
		{name: "staff who are no practitioner", want: invoice.ErrForbidden, caller: invoice.Caller{AccountID: "acct-other", OrganizationID: org}, draft: quoteDraft(), audited: true},
		{name: "another practitioner", want: invoice.ErrForbidden, caller: invoice.Caller{AccountID: "acct-2", OrganizationID: org, PractitionerID: "pract-2"}, draft: quoteDraft(), audited: true},
		{name: "an unknown visit", want: invoice.ErrNotFound, caller: adminC, draft: with(func(d *invoice.QuoteDraft) { d.AppointmentID = "appt-x" }), audited: true},
		{name: "another clinic", want: invoice.ErrNotFound, caller: invoice.Caller{AccountID: "acct-b", OrganizationID: "org-2", Admin: true}, draft: quoteDraft(), audited: true},
		{name: "a patient not the visit's", want: invoice.ErrNotFound, caller: adminC, draft: with(func(d *invoice.QuoteDraft) { d.PatientID = "patient-consented" }), audited: true},
		{name: "a revoked care consent", want: invoice.ErrNotFound, caller: adminC, draft: with(func(d *invoice.QuoteDraft) {
			d.AppointmentID, d.PatientID = revokedVisit, "patient-revoked"
		}), audited: true},
		{name: "a restricted record (S7)", want: invoice.ErrNotFound, caller: adminC, draft: with(func(d *invoice.QuoteDraft) {
			d.AppointmentID, d.PatientID = restrictedVisit, "patient-restricted"
		}), audited: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(theSeller)
			_, err := f.svc.SendQuote(t.Context(), tc.caller, tc.draft)
			if !errors.Is(err, tc.want) {
				t.Fatalf("SendQuote() = %v, want %v", err, tc.want)
			}
			if len(f.repo.quotes) != 0 || len(f.jobs.emails) != 0 {
				t.Errorf("refused, yet %d quotes and e-mails %v", len(f.repo.quotes), f.jobs.emails)
			}
			if audited := f.audit.last().Outcome == auditevent.OutcomeMinorFailure; audited != tc.audited {
				t.Errorf("refusal audited = %v, want %v (%+v)", audited, tc.audited, f.audit.events)
			}
		})
	}
}

// The clinic's admin sees every quote of the clinic, a practitioner those they sent, newest first, expired once past
// their validity; each list and read audited.
func TestQuotes_ShowsTheClinicsQuotesWithTheirState(t *testing.T) {
	f := newFixture(theSeller)
	first, err := f.svc.SendQuote(t.Context(), adminC, quoteDraft())
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.svc.SendQuote(t.Context(), doctorC, quoteDraft())
	if err != nil {
		t.Fatal(err)
	}
	all, err := f.svc.Quotes(t.Context(), adminC, "")
	if err != nil || len(all) != 2 || all[0].ID != second.ID || all[1].ID != first.ID {
		t.Fatalf("admin's Quotes() = %v, %v; want both, newest first", all, err)
	}
	if e := f.audit.last(); e.Subtype.Code != auditevent.CodeSearchType || e.Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit = %+v", e)
	}
	own, err := f.svc.Quotes(t.Context(), doctorC, "")
	if err != nil || len(own) != 1 || own[0].ID != second.ID {
		t.Errorf("practitioner's Quotes() = %v, %v; want theirs", own, err)
	}
	if got, err := f.svc.ReadQuote(t.Context(), doctorC, first.ID); !errors.Is(err, invoice.ErrNotFound) || got.ID != "" {
		t.Errorf("practitioner reads the admin's quote: %+v, %v; want nothing and ErrNotFound", got, err)
	}
	if other, err := f.svc.Quotes(t.Context(), adminC, "appt-other"); err != nil || len(other) != 0 {
		t.Errorf("another visit's Quotes() = %v, %v; want none", other, err)
	}
	if one, err := f.svc.Quotes(t.Context(), adminC, visit); err != nil || len(one) != 2 {
		t.Errorf("the visit's Quotes() = %v, %v; want both", one, err)
	}
	later := f.later(31 * 24 * time.Hour)
	got, err := later.svc.ReadQuote(t.Context(), adminC, first.ID)
	if err != nil || got.State != invoice.QuoteExpired {
		t.Errorf("ReadQuote() 31 days on = %s, %v; want expired", got.State, err)
	}
	if e := later.audit.last(); e.Subtype.Code != auditevent.CodeRead || e.EntityReference != "Quote/"+first.ID {
		t.Errorf("audit = %+v", e)
	}
	if _, err := f.svc.Quotes(t.Context(), patientC, ""); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("a patient's Quotes() = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.ReadQuote(t.Context(), invoice.Caller{AccountID: "acct-b", OrganizationID: "org-2", Admin: true}, first.ID); !errors.Is(err, invoice.ErrNotFound) {
		t.Errorf("another clinic's ReadQuote() = %v, want ErrNotFound", err)
	}
}

// later is the fixture's data on a clock d later.
func (f fixture) later(d time.Duration) fixture {
	l := newFixtureAt(theSeller, clock.Fixed{T: at.Add(d)})
	l.repo.quotes, l.repo.invoices, l.repo.last = f.repo.quotes, f.repo.invoices, f.repo.last
	return l
}

// The patient sees their own quotes from every clinic, and only their own, and their PDF, which says it is a quote and
// not a receipt. Staff have no "my quotes".
func TestMyQuotes_ShowsThePatientTheirOwnAndThePDF(t *testing.T) {
	f := newFixture(theSeller)
	q, err := f.svc.SendQuote(t.Context(), adminC, quoteDraft())
	if err != nil {
		t.Fatal(err)
	}
	mine, err := f.svc.MyQuotes(t.Context(), patientC)
	if err != nil || len(mine) != 1 || mine[0].ID != q.ID || mine[0].State != invoice.QuoteSent {
		t.Fatalf("MyQuotes() = %v, %v", mine, err)
	}
	if e := f.audit.last(); e.Subtype.Code != auditevent.CodeSearchType || e.Purpose.Code != auditevent.CodePATRQT {
		t.Errorf("audit = %+v", e)
	}
	if got, err := f.svc.MyQuotesAt(t.Context(), patientC, org); err != nil || len(got) != 1 {
		t.Errorf("MyQuotesAt(the clinic) = %v, %v; want the one", got, err)
	}
	if got, err := f.svc.MyQuotesAt(t.Context(), patientC, "00000000-0000-4000-8000-0000000000ee"); err != nil || len(got) != 0 {
		t.Errorf("MyQuotesAt(another clinic) = %v, %v; want none", got, err)
	}
	other := invoice.Caller{AccountID: "acct-other", PatientID: "patient-other"}
	if got, err := f.svc.MyQuotes(t.Context(), other); err != nil || len(got) != 0 {
		t.Errorf("another patient's MyQuotes() = %v, %v; want none", got, err)
	}
	if _, err := f.svc.MyQuotes(t.Context(), adminC); !errors.Is(err, invoice.ErrForbidden) {
		t.Errorf("staff's MyQuotes() = %v, want ErrForbidden", err)
	}
	pdf, err := f.svc.QuotePDF(t.Context(), patientC, q.ID)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("QuotePDF() = %d bytes, %v", len(pdf), err)
	}
	if e := f.audit.last(); e.Subtype.Code != auditevent.CodeRead || e.EntityReference != "Quote/"+q.ID {
		t.Errorf("audit = %+v", e)
	}
	if _, err := f.svc.QuotePDF(t.Context(), other, q.ID); !errors.Is(err, invoice.ErrNotFound) {
		t.Errorf("another patient's QuotePDF() = %v, want ErrNotFound", err)
	}
	if f.audit.last().Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("the refused PDF is not audited: %+v", f.audit.last())
	}
}

// The clinic's admin opens any quote of the clinic as its PDF, a practitioner those they sent; anyone else, another
// clinic included, gets ErrNotFound. Audited as the read, also when refused.
func TestClinicQuotePDF_GivesTheClinicItsQuotesAsPDF(t *testing.T) {
	f := newFixture(theSeller)
	q, err := f.svc.SendQuote(t.Context(), adminC, quoteDraft())
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := f.svc.ClinicQuotePDF(t.Context(), adminC, q.ID)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("ClinicQuotePDF() = %d bytes, %v", len(pdf), err)
	}
	if e := f.audit.last(); e.Subtype.Code != auditevent.CodeRead || e.EntityReference != "Quote/"+q.ID || e.Outcome != auditevent.OutcomeSuccess {
		t.Errorf("audit = %+v", e)
	}
	for name, c := range map[string]invoice.Caller{
		"a practitioner who did not send it": doctorC,
		"another clinic's admin":             {AccountID: "acct-b", OrganizationID: "org-2", Admin: true},
		"a patient":                          patientC,
	} {
		if got, err := f.svc.ClinicQuotePDF(t.Context(), c, q.ID); !errors.Is(err, invoice.ErrNotFound) || got != nil {
			t.Errorf("%s: ClinicQuotePDF() = %d bytes, %v; want ErrNotFound", name, len(got), err)
		}
		if f.audit.last().Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s: the refused PDF is not audited: %+v", name, f.audit.last())
		}
	}
}
