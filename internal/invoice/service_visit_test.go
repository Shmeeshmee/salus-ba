package invoice_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/invoice"
)

// The desk's day shows each visit's receipts (docs/clinic-cms/walk/wave3, findings 1 and 2): their number, id and
// whether reversed, the reversals themselves left out, read in the clinic's scope for the day's service, which
// authorizes and audits.
func TestVisitReceipts_GivesEachVisitsReceiptsWithWhetherReversed(t *testing.T) {
	f := newFixture(theSeller)
	first, err := f.svc.Issue(t.Context(), adminC, draft())
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.svc.Issue(t.Context(), adminC, draft())
	if err != nil {
		t.Fatal(err)
	}
	reversal, err := f.svc.Reverse(t.Context(), adminC, first.ID, invoice.PaymentCash)
	if err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil
	got, err := f.svc.VisitReceipts(t.Context(), org, []string{visit, "appt-without"})
	want := []appointment.VisitReceipt{{ID: first.ID, Number: first.Number, Reversed: true}, {ID: second.ID, Number: second.Number}}
	if err != nil || !slices.Equal(got[visit], want) || len(got["appt-without"]) != 0 {
		t.Errorf("VisitReceipts() = %+v, %v; want %+v (not the reversal %s)", got, err, want, reversal.ID)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("%d audit events, want none (the desk's day audits what it shows)", len(f.audit.events))
	}
	if got, _ := f.svc.VisitReceipts(t.Context(), "org-2", []string{visit}); len(got[visit]) != 0 {
		t.Errorf("another clinic's VisitReceipts() = %+v, want none", got)
	}
}

// The admin's visit page lists the visit's receipts: the admin's only (as the list of all receipts), audited.
func TestForVisit_GivesTheAdminTheVisitsReceiptsAndRefusesAnyoneElse(t *testing.T) {
	f := newFixture(theSeller)
	issued, err := f.svc.Issue(t.Context(), doctorC, draft())
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.ForVisit(t.Context(), adminC, visit)
	if err != nil || len(got) != 1 || got[0].ID != issued.ID {
		t.Fatalf("ForVisit() = %+v, %v; want the visit's receipt", got, err)
	}
	if e := f.audit.last(); e.Subtype.Code != auditevent.CodeSearchType || e.EntityReference != "Invoice/"+issued.ID || e.AgentAccountID != admin {
		t.Errorf("audit = %+v, want the receipt's read by the admin", e)
	}
	desk := invoice.Caller{AccountID: "acct-desk", OrganizationID: org, Reception: true}
	for _, c := range []invoice.Caller{doctorC, desk} {
		if _, err := f.svc.ForVisit(t.Context(), c, visit); !errors.Is(err, invoice.ErrForbidden) {
			t.Errorf("ForVisit() as %s = %v, want ErrForbidden", c.AccountID, err)
		}
	}
}

// A receipt of an in-person visit copies where the service was given, the visit's location with its address, at issue
// (docs/clinic-cms/WAVE6.md U2); the seller stays the clinic's registered details. A video visit's names no place.
func TestIssue_AnInPersonVisitsReceiptCopiesThePlaceOfService(t *testing.T) {
	f := newFixture(theSeller)
	d := draft()
	d.AppointmentID = inPersonVisit
	got, err := f.svc.Issue(t.Context(), adminC, d)
	if err != nil || got.PlaceOfService != "Synthetic annex, Synthetic annex 2, 88000 Mostar" || got.SellerAddress != theSeller.Address {
		t.Errorf("Issue() = %+v, %v; want the annex as the place of service and the seller's own address", got, err)
	}
	if got, err := f.svc.Issue(t.Context(), adminC, draft()); err != nil || got.PlaceOfService != "" {
		t.Errorf("a video visit's receipt: place %q, %v; want none", got.PlaceOfService, err)
	}
}
