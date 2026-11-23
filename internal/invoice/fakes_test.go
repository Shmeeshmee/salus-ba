package invoice_test

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
	"github.com/Shmeeshmee/salus/pkg/fiscal/fiscaltest"
)

// Synthetic ids and a synthetic clinic. Nothing here is a real person or company.
const (
	org      = "org-1"
	admin    = "acct-admin"
	doctor   = "acct-doctor"
	patientA = "patient-a"
	visit    = "appt-1"
	// revokedVisit is a visit whose patient revoked their care consent since.
	revokedVisit = "appt-revoked"
	// registeredVisit is a visit of a patient the clinic registered, without an account (docs/clinic-cms W1), with an
	// active care consent.
	registeredVisit = "appt-registered"
	exam            = "svc-exam"
	lab             = "svc-lab"
)

var (
	at        = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	adminC    = invoice.Caller{AccountID: admin, OrganizationID: org, Admin: true}
	doctorC   = invoice.Caller{AccountID: doctor, OrganizationID: org, PractitionerID: "pract-1"}
	patientC  = invoice.Caller{AccountID: "acct-patient", PatientID: patientA}
	theSeller = invoice.Seller{LegalName: "Synthetic clinic d.o.o.", TaxID: "4200000000000", Address: "Placeholder street 1, 71000 Sarajevo"}
)

// fakeTx runs fn and, like a transaction, undoes the repository's and the jobs' writes when fn fails.
type fakeTx struct {
	repo *fakeRepo
	jobs *fakeJobs
}

func (t fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if t.repo.depth > 0 { // a nested WithTx joins the outer one
		return fn(ctx)
	}
	t.repo.depth++
	invoices, last, enqueued := slices.Clone(t.repo.invoices), t.repo.last, slices.Clone(t.jobs.enqueued)
	quotes, emails := slices.Clone(t.repo.quotes), slices.Clone(t.jobs.emails)
	err := fn(ctx)
	t.repo.depth--
	t.repo.scopeOrg, t.repo.scopePatient = "", ""
	if err != nil {
		t.repo.invoices, t.repo.last, t.jobs.enqueued = invoices, last, enqueued
		t.repo.quotes, t.jobs.emails = quotes, emails
	}
	return err
}

// fakeRepo keeps receipts in memory and shows them as row-level security does: the scoped clinic's, or the scoped
// patient's own.
type fakeRepo struct {
	scopeOrg     string
	scopePatient string
	invoices     []invoice.Invoice
	quotes       []invoice.Quote
	last         int64
	depth        int
}

func (r *fakeRepo) SetOrganization(_ context.Context, id string) error { r.scopeOrg = id; return nil }
func (r *fakeRepo) SetPatient(_ context.Context, id string) error      { r.scopePatient = id; return nil }

func (r *fakeRepo) seen(i invoice.Invoice) bool {
	return (r.scopeOrg != "" && i.OrganizationID == r.scopeOrg) || (r.scopePatient != "" && i.PatientID == r.scopePatient)
}

func (r *fakeRepo) ByID(_ context.Context, id string) (invoice.Invoice, error) {
	for _, i := range r.invoices {
		if i.ID == id && r.seen(i) {
			for _, o := range r.invoices {
				if o.ReversesID == id {
					i.ReversedByID = o.ID
				}
			}
			return i, nil
		}
	}
	return invoice.Invoice{}, invoice.ErrNotFound
}

func (r *fakeRepo) ByIdempotencyKey(ctx context.Context, clinic, key string) (invoice.Invoice, error) {
	for _, i := range r.invoices {
		if i.OrganizationID == clinic && i.IdempotencyKey == key && key != "" && r.seen(i) {
			return r.ByID(ctx, i.ID)
		}
	}
	return invoice.Invoice{}, invoice.ErrNotFound
}

func (r *fakeRepo) Locked(ctx context.Context, id string) (invoice.Invoice, error) {
	i, err := r.ByID(ctx, id)
	i.ReversedByID = ""
	return i, err
}

func (r *fakeRepo) Issued(ctx context.Context, org string, p invoice.Period) ([]invoice.Invoice, error) {
	var out []invoice.Invoice
	for _, i := range slices.Backward(r.invoices) {
		if i.OrganizationID == org && r.seen(i) && !i.IssuedAt.Before(p.From) && i.IssuedAt.Before(p.To) {
			i, _ = r.ByID(ctx, i.ID)
			out = append(out, i)
		}
	}
	return out, nil
}

func (r *fakeRepo) OfPatient(_ context.Context, patientID string) ([]invoice.Invoice, error) {
	var out []invoice.Invoice
	for _, i := range slices.Backward(r.invoices) {
		if i.PatientID == patientID && r.seen(i) {
			out = append(out, i)
		}
	}
	return out, nil
}

func (r *fakeRepo) NextNumber(context.Context, string) (int64, error) {
	r.last++
	return r.last, nil
}

func (r *fakeRepo) Insert(_ context.Context, inv invoice.Invoice) (invoice.Invoice, error) {
	for _, o := range r.invoices {
		if inv.ReversesID != "" && o.ReversesID == inv.ReversesID {
			return invoice.Invoice{}, invoice.ErrAlreadyReversed
		}
	}
	inv.ID, inv.FiscalStatus = fmt.Sprintf("inv-%d", len(r.invoices)+1), invoice.FiscalPending
	r.invoices = append(r.invoices, inv)
	return inv, nil
}

func (r *fakeRepo) SetFiscalized(_ context.Context, id string, f fiscal.Fiscalized) error {
	for n := range r.invoices {
		if i := &r.invoices[n]; i.ID == id && i.FiscalStatus != invoice.FiscalFiscalized {
			i.FiscalStatus, i.FiscalID, i.FiscalCode, i.FiscalQR, i.FiscalizedAt = invoice.FiscalFiscalized, f.ID, f.Code, f.QR, f.At
		}
	}
	return nil
}

func (r *fakeRepo) SetFailed(_ context.Context, id string) error {
	for n := range r.invoices {
		if i := &r.invoices[n]; i.ID == id && i.FiscalStatus == invoice.FiscalPending {
			i.FiscalStatus = invoice.FiscalFailed
		}
	}
	return nil
}

func (r *fakeRepo) InsertQuote(_ context.Context, q invoice.Quote) (invoice.Quote, error) {
	q.ID = fmt.Sprintf("quote-%d", len(r.quotes)+1)
	r.quotes = append(r.quotes, q)
	return q, nil
}

func (r *fakeRepo) seenQuote(q invoice.Quote) bool {
	return (r.scopeOrg != "" && q.OrganizationID == r.scopeOrg) || (r.scopePatient != "" && q.PatientID == r.scopePatient)
}

func (r *fakeRepo) QuoteByID(_ context.Context, id string) (invoice.Quote, error) {
	for _, q := range r.quotes {
		if q.ID == id && r.seenQuote(q) {
			return q, nil
		}
	}
	return invoice.Quote{}, invoice.ErrNotFound
}

func (r *fakeRepo) QuotesOf(_ context.Context, clinic string) ([]invoice.Quote, error) {
	var out []invoice.Quote
	for _, q := range slices.Backward(r.quotes) {
		if q.OrganizationID == clinic && r.seenQuote(q) {
			out = append(out, q)
		}
	}
	return out, nil
}

func (r *fakeRepo) QuotesOfPatient(_ context.Context, patientID string) ([]invoice.Quote, error) {
	var out []invoice.Quote
	for _, q := range slices.Backward(r.quotes) {
		if q.PatientID == patientID && r.seenQuote(q) {
			out = append(out, q)
		}
	}
	return out, nil
}

// fakePrices is the price list package B builds: two synthetic services of the clinic, the second with VAT.
type fakePrices struct{}

func (fakePrices) Lines(_ context.Context, clinic string, picks []chargeitemdefinition.Pick) ([]chargeitemdefinition.Line, error) {
	vat := int32(1700)
	list := map[string]chargeitemdefinition.Line{
		exam: {DefinitionID: exam, Name: "Placeholder examination", Currency: "BAM", UnitPriceMinor: 5000},
		lab:  {DefinitionID: lab, Name: "Placeholder laboratory item", Currency: "BAM", UnitPriceMinor: 1170, VATRateBP: &vat},
	}
	var out []chargeitemdefinition.Line
	for _, p := range picks {
		l, ok := list[p.DefinitionID]
		if !ok || clinic != org {
			return nil, chargeitemdefinition.ErrNotFound
		}
		l.Quantity, l.DiscountMinor = p.Quantity, p.DiscountMinor
		l.TotalMinor = int64(p.Quantity)*l.UnitPriceMinor - p.DiscountMinor
		out = append(out, l)
	}
	return out, nil
}

type fakeSellers struct{ seller invoice.Seller }

func (f fakeSellers) Seller(context.Context, string) (invoice.Seller, error) { return f.seller, nil }

type fakeClinics struct{}

func (fakeClinics) Clinic(context.Context, string) (organization.Organization, error) {
	return organization.Organization{ID: org, Name: "Synthetic clinic", TimeZone: "Europe/Sarajevo", DefaultLanguage: "en"}, nil
}

// fakeVisits holds two visits of the clinic, both the doctor's: with patientA, and with a patient who revoked their
// care consent.
type fakeVisits struct{}

func (fakeVisits) Lookup(_ context.Context, id string) (appointment.Appointment, error) {
	switch id {
	case visit:
		return appointment.Appointment{ID: visit, OrganizationID: org, PatientID: patientA}, nil
	case revokedVisit:
		return appointment.Appointment{ID: revokedVisit, OrganizationID: org, PatientID: "patient-revoked"}, nil
	case registeredVisit:
		return appointment.Appointment{ID: registeredVisit, OrganizationID: org, PatientID: "patient-registered"}, nil
	}
	return appointment.Appointment{}, appointment.ErrNotFound
}

func (fakeVisits) LookupPractitioner(context.Context, string) (string, error) { return "pract-1", nil }

func (fakeVisits) PatientHasAccount(_ context.Context, id string) (bool, error) {
	if id == registeredVisit {
		return false, nil
	}
	return true, nil
}

// fakeCare: the clinic treats patientA (a visit and an active care consent) and "patient-consented" (an active care
// consent), nobody else.
type fakeCare struct{}

func (fakeCare) Visits(_ context.Context, patientID, _ string) ([]access.Visit, error) {
	if patientID == patientA {
		return []access.Visit{{Start: at}}, nil
	}
	return nil, nil
}

func (fakeCare) State(_ context.Context, patientID, _ string) (access.Consent, error) {
	return access.Consent{Active: patientID == "patient-consented" || patientID == patientA || patientID == "patient-registered"}, nil
}

type fakeJobs struct {
	enqueued []string // the fiscal reports' receipts
	emails   []string // the quote e-mails' visits
}

func (j *fakeJobs) EnqueueQuoteEmail(_ context.Context, appointmentID, _ string) error {
	j.emails = append(j.emails, appointmentID)
	return nil
}

func (j *fakeJobs) EnqueueFiscalize(_ context.Context, id, _ string) error {
	j.enqueued = append(j.enqueued, id)
	return nil
}

type fakeAudit struct{ events []auditevent.AuditEvent }

func (a *fakeAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	a.events = append(a.events, e)
	return nil
}

// last is the last event recorded.
func (a *fakeAudit) last() auditevent.AuditEvent {
	if len(a.events) == 0 {
		return auditevent.AuditEvent{}
	}
	return a.events[len(a.events)-1]
}

// fixture is a Service on its fakes.
type fixture struct {
	svc      *invoice.Service
	repo     *fakeRepo
	jobs     *fakeJobs
	audit    *fakeAudit
	reporter *fiscaltest.Reporter
}

func newFixture(seller invoice.Seller) fixture { return newFixtureAt(seller, clock.Fixed{T: at}) }

// newFixtureAt is newFixture on the clock clk.
func newFixtureAt(seller invoice.Seller, clk clock.Fixed) fixture {
	f := fixture{repo: &fakeRepo{}, jobs: &fakeJobs{}, audit: &fakeAudit{}, reporter: &fiscaltest.Reporter{Clock: clk}}
	f.svc = invoice.NewService(invoice.Ports{
		Repo: f.repo, Prices: fakePrices{}, Reporter: f.reporter, Sellers: fakeSellers{seller}, Clinics: fakeClinics{},
		Visits: fakeVisits{}, Consents: fakeCare{}, Care: fakeCare{}, Jobs: f.jobs, Audit: f.audit,
		Tx: fakeTx{repo: f.repo, jobs: f.jobs}, Clock: clk,
	})
	return f
}

// draft is a cash receipt for the visit: two examinations with a discount, and a laboratory item.
func draft() invoice.Draft {
	return invoice.Draft{
		AppointmentID: visit, Kind: invoice.KindReceipt, PaymentMethod: invoice.PaymentCash,
		Picks: []chargeitemdefinition.Pick{{DefinitionID: exam, Quantity: 2, DiscountMinor: 500}, {DefinitionID: lab, Quantity: 1}},
	}
}
