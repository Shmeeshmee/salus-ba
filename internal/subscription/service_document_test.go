package subscription_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/subscription"
)

type fakeJobs struct {
	emails []subscription.InvoiceEmailArgs
}

func (j *fakeJobs) EnqueueInvoiceEmail(_ context.Context, a subscription.InvoiceEmailArgs) error {
	j.emails = append(j.emails, a)
	return nil
}

type fakeBilling map[string]organization.BillingDetails

func (b fakeBilling) ForClinic(_ context.Context, org string) (organization.BillingDetails, error) {
	return b[org], nil
}

type fakeMailer struct {
	err  error
	sent []notification.MembershipInvoice
}

func (m *fakeMailer) SendMembershipInvoice(_ context.Context, in notification.MembershipInvoice) error {
	m.sent = append(m.sent, in)
	return m.err
}

// documentWorld is newWorld with the ports of the invoice's document: the jobs, the billing details and the mailer.
func documentWorld(t *testing.T) (*world, *fakeJobs, *fakeMailer, fakeBilling) {
	t.Helper()
	w := newWorld(t)
	jobs, mailer := &fakeJobs{}, &fakeMailer{}
	billing := fakeBilling{orgA: {LegalName: "Synthetic clinic d.o.o.", TaxID: "4200000000001", BillingEmail: "billing@example.test"}}
	w.svc = subscription.NewService(subscription.Ports{
		Repo: w.repo, Audit: w.audit, Tx: fakeTx{}, Clock: w.clock, Clinics: fakeClinics{}, Seats: w.seats,
		Jobs: jobs, Billing: billing, Mailer: mailer,
		Seller: subscription.Seller{Name: "[SELLER]", TaxID: "[SELLER TAX ID]", Address: "[SELLER ADDRESS]", Account: "[SELLER ACCOUNT]"},
	})
	return w, jobs, mailer, billing
}

// Issuing an invoice enqueues its e-mail in the same transaction, once: a second run for the month enqueues nothing.
func TestService_IssuingAnInvoiceEnqueuesItsEmailOnce(t *testing.T) {
	w, jobs, _, _ := documentWorld(t)
	inv, err := w.svc.IssueMonthly(t.Context(), orgA, dec1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.svc.IssueMonthly(t.Context(), orgA, dec1); err != nil {
		t.Fatal(err)
	}
	if len(jobs.emails) != 1 || jobs.emails[0] != (subscription.InvoiceEmailArgs{InvoiceID: inv.ID, OrganizationID: orgA}) {
		t.Errorf("enqueued %+v, want one e-mail of %s", jobs.emails, inv.ID)
	}
}

// The clinic's admin downloads an invoice as a PDF; its other staff may not, and nobody reads another clinic's.
func TestService_OnlyTheClinicsAdminDownloadsTheInvoice(t *testing.T) {
	w, _, _, _ := documentWorld(t)
	inv, err := w.svc.IssueMonthly(t.Context(), orgA, dec1)
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := w.svc.PDF(t.Context(), admin, inv.ID)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("PDF() = %.8q, %v, want a PDF", pdf, err)
	}
	for name, tc := range map[string]struct {
		want error
		id   string
		c    subscription.Caller
	}{
		"its practitioner":  {subscription.ErrForbidden, inv.ID, subscription.Caller{AccountID: "p", OrganizationID: orgA}},
		"another clinic":    {subscription.ErrNotFound, inv.ID, subscription.Caller{AccountID: "b", OrganizationID: orgB, Admin: true}},
		"no clinic":         {subscription.ErrNotFound, inv.ID, subscription.Caller{AccountID: "x"}},
		"an unknown number": {subscription.ErrNotFound, "unknown", admin},
	} {
		if _, err := w.svc.PDF(t.Context(), tc.c, tc.id); !errors.Is(err, tc.want) {
			t.Errorf("%s: PDF() error = %v, want %v", name, err, tc.want)
		}
	}
}

// The job sends the invoice to the clinic's billing e-mail in its default language with the PDF attached; a clinic
// without a billing e-mail gets none, and a failed send is returned for River to retry.
func TestService_SendInvoiceMailsThePDFToTheBillingEmail(t *testing.T) {
	w, _, mailer, billing := documentWorld(t)
	inv, err := w.svc.IssueMonthly(t.Context(), orgA, dec1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.svc.SendInvoice(t.Context(), orgA, inv.ID); err != nil {
		t.Fatal(err)
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("sent %d e-mails, want 1", len(mailer.sent))
	}
	got := mailer.sent[0]
	if got.To != "billing@example.test" || got.Language != "bs" || got.Number != inv.Number || !bytes.HasPrefix(got.PDF, []byte("%PDF-")) {
		t.Errorf("sent to %q in %q, number %d, want the billing e-mail in bs with the PDF of %d", got.To, got.Language, got.Number, inv.Number)
	}
	if err := w.svc.SendInvoice(t.Context(), orgA, "unknown"); !errors.Is(err, subscription.ErrNotFound) {
		t.Errorf("an unknown invoice: %v, want ErrNotFound", err)
	}
	mailer.err = errBroken
	if err := w.svc.SendInvoice(t.Context(), orgA, inv.ID); !errors.Is(err, errBroken) {
		t.Errorf("a failed send: %v, want it returned", err)
	}
	billing[orgA] = organization.BillingDetails{LegalName: "Synthetic clinic d.o.o."}
	mailer.err, mailer.sent = nil, nil
	if err := w.svc.SendInvoice(t.Context(), orgA, inv.ID); err != nil || len(mailer.sent) != 0 {
		t.Errorf("no billing e-mail: %v, sent %d, want nothing sent", err, len(mailer.sent))
	}
}
