package subscription_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const (
	orgA = "0a000000-0000-4000-8000-000000000001"
	orgB = "0b000000-0000-4000-8000-000000000002"
)

var errBroken = errors.New("broken")

type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type fakeAudit struct{ events []auditevent.AuditEvent }

func (a *fakeAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	a.events = append(a.events, e)
	return nil
}

// by counts the events of agent.
func (a *fakeAudit) by(agent string) int {
	n := 0
	for _, e := range a.events {
		if e.AgentName == agent && e.EntityReference != "" && e.OrganizationID != "" {
			n++
		}
	}
	return n
}

type fakeClinics struct{}

func (fakeClinics) Clinic(_ context.Context, id string) (organization.Organization, error) {
	return organization.Organization{ID: id, TimeZone: "Europe/Sarajevo"}, nil
}

type fakeSeats map[string]int

func (s fakeSeats) ActiveCount(_ context.Context, org string) (int, error) {
	if s[org] < 0 {
		return 0, errBroken
	}
	return s[org], nil
}

// fakeRepo keeps memberships and invoices in memory; it sees every clinic (the service names the clinic it asks for).
type fakeRepo struct {
	plans    map[string]subscription.Plan
	subs     map[string]subscription.Subscription
	invoices []subscription.Invoice
	number   int64
}

func (r *fakeRepo) SetOrganization(context.Context, string) error { return nil }
func (r *fakeRepo) SetRun(context.Context) error                  { return nil }

func (r *fakeRepo) Plans(context.Context) ([]subscription.Plan, error) { return nil, nil }

func (r *fakeRepo) Plan(_ context.Context, code string) (subscription.Plan, error) {
	p, ok := r.plans[code]
	if !ok {
		return subscription.Plan{}, subscription.ErrNoPlan
	}
	return p, nil
}

func (r *fakeRepo) PutPlan(_ context.Context, p subscription.Plan) error {
	r.plans[p.Code] = p
	return nil
}

func (r *fakeRepo) Subscription(_ context.Context, org string) (subscription.Subscription, error) {
	s, ok := r.subs[org]
	if !ok {
		return subscription.Subscription{}, subscription.ErrNotFound
	}
	return s, nil
}

func (r *fakeRepo) Lock(ctx context.Context, org string) (subscription.Subscription, error) {
	return r.Subscription(ctx, org)
}

func (r *fakeRepo) Insert(_ context.Context, org, plan string, _, trialEndsAt time.Time) (bool, error) {
	if _, ok := r.subs[org]; ok {
		return false, nil
	}
	r.subs[org] = subscription.Subscription{OrganizationID: org, PlanCode: plan, Status: subscription.StatusTrial, TrialEndsAt: trialEndsAt}
	return true, nil
}

func (r *fakeRepo) SetStatus(_ context.Context, org string, status subscription.Status, _ time.Time) error {
	s := r.subs[org]
	s.Status = status
	r.subs[org] = s
	return nil
}

func (r *fakeRepo) SetPeriod(_ context.Context, org string, start, end, _ time.Time) error {
	s := r.subs[org]
	s.CurrentPeriodStart, s.CurrentPeriodEnd = start, end
	r.subs[org] = s
	return nil
}

func (r *fakeRepo) Subscribed(context.Context) ([]string, error) {
	var out []string
	for org, s := range r.subs {
		if s.Status != subscription.StatusCancelled {
			out = append(out, org)
		}
	}
	slices.Sort(out)
	return out, nil
}

func (r *fakeRepo) Invoices(_ context.Context, org string) ([]subscription.Invoice, error) {
	var out []subscription.Invoice
	for _, inv := range r.invoices {
		if inv.OrganizationID == org {
			out = append(out, inv)
		}
	}
	return out, nil
}

func (r *fakeRepo) InvoiceOf(_ context.Context, org string, start time.Time) (subscription.Invoice, bool, error) {
	for _, inv := range r.invoices {
		if inv.OrganizationID == org && inv.PeriodStart.Equal(start) {
			return inv, true, nil
		}
	}
	return subscription.Invoice{}, false, nil
}

func (r *fakeRepo) NextNumber(context.Context) (int64, error) {
	r.number++
	return r.number, nil
}

func (r *fakeRepo) InsertInvoice(_ context.Context, in subscription.Invoice) (subscription.Invoice, error) {
	in.ID, in.Status = "inv", subscription.InvoiceOpen
	r.invoices = append(r.invoices, in)
	return in, nil
}

func (r *fakeRepo) MarkPaid(_ context.Context, number int64, at time.Time) (subscription.Invoice, error) {
	for i, inv := range r.invoices {
		if inv.Number == number && inv.Status == subscription.InvoiceOpen {
			r.invoices[i].Status, r.invoices[i].PaidAt = subscription.InvoicePaid, at
			return r.invoices[i], nil
		}
	}
	return subscription.Invoice{}, subscription.ErrNotFound
}

func (r *fakeRepo) OldestPastDue(_ context.Context, org string, now time.Time) (time.Time, error) {
	var oldest time.Time
	for _, inv := range r.invoices {
		if inv.OrganizationID == org && inv.Status == subscription.InvoiceOpen && inv.DueAt.Before(now) && (oldest.IsZero() || inv.DueAt.Before(oldest)) {
			oldest = inv.DueAt
		}
	}
	return oldest, nil
}

type world struct {
	svc   *subscription.Service
	repo  *fakeRepo
	audit *fakeAudit
	clock *clock.Fixed
	seats fakeSeats
}

// dec1 is midnight of 1 December 2026 in Sarajevo (UTC+1).
var dec1 = time.Date(2026, 11, 30, 23, 0, 0, 0, time.UTC)

// jan1 is midnight of 1 January 2027 there.
var jan1 = time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC)

// newWorld is a world with the basic plan at 100.00 BAM a seat and clinic A in a trial that ended in November.
func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{
		repo: &fakeRepo{
			plans: map[string]subscription.Plan{"basic": {Code: "basic", Name: "Basic", Currency: "BAM", PricePerSeatMinor: 10000, Active: true}},
			subs:  map[string]subscription.Subscription{},
		},
		audit: &fakeAudit{}, clock: &clock.Fixed{T: dec1.Add(2 * time.Hour)}, seats: fakeSeats{orgA: 3, orgB: 1},
	}
	w.svc = subscription.NewService(subscription.Ports{Repo: w.repo, Audit: w.audit, Tx: fakeTx{}, Clock: w.clock, Clinics: fakeClinics{}, Seats: w.seats})
	w.repo.subs[orgA] = subscription.Subscription{OrganizationID: orgA, PlanCode: "basic", Status: subscription.StatusTrial, TrialEndsAt: dec1.AddDate(0, 0, -5)}
	return w
}

func (w *world) status(org string) subscription.Status { return w.repo.subs[org].Status }

var admin = subscription.Caller{AccountID: "acc", OrganizationID: orgA, Admin: true}

// Only the clinic's admin sees its membership and invoices.
func TestService_OnlyTheClinicsAdminSeesTheMembership(t *testing.T) {
	w := newWorld(t)
	for name, tc := range map[string]struct {
		want error
		c    subscription.Caller
	}{
		"admin":            {c: admin},
		"practitioner":     {c: subscription.Caller{AccountID: "acc", OrganizationID: orgA}, want: subscription.ErrForbidden},
		"patient":          {c: subscription.Caller{AccountID: "acc"}, want: subscription.ErrNotFound},
		"no membership":    {c: subscription.Caller{AccountID: "acc", OrganizationID: orgB, Admin: true}, want: subscription.ErrNotFound},
		"another's admin?": {c: subscription.Caller{AccountID: "acc", OrganizationID: orgB}, want: subscription.ErrForbidden},
	} {
		got, err := w.svc.Current(t.Context(), tc.c)
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: Current() = %v, want %v", name, err, tc.want)
		}
		if tc.want == nil && (got.Plan.Name != "Basic" || got.Status != subscription.StatusTrial) {
			t.Errorf("%s: Current() = %+v, want the trial with its plan", name, got)
		}
		if _, err := w.svc.Invoices(t.Context(), tc.c); name != "no membership" && !errors.Is(err, tc.want) {
			t.Errorf("%s: Invoices() = %v, want %v", name, err, tc.want)
		}
	}
	if len(w.audit.events) != 0 {
		t.Errorf("%d audit events for reads, want none: nothing in a membership is personal data", len(w.audit.events))
	}
}

// A new clinic gets a 30-day trial of the default plan, once, audited as salusctl; none while the plan is not set.
func TestStartTrial_GivesThirtyDaysOfTheDefaultPlanOnce(t *testing.T) {
	w := newWorld(t)
	got, err := w.svc.StartTrial(t.Context(), orgB)
	if err != nil {
		t.Fatal(err)
	}
	if want := w.clock.Now().Add(30 * 24 * time.Hour); got.Status != subscription.StatusTrial || got.PlanCode != "basic" || !got.TrialEndsAt.Equal(want) {
		t.Errorf("StartTrial() = %+v, want a basic trial until %v", got, want)
	}
	w.clock.T = w.clock.T.Add(48 * time.Hour)
	again, err := w.svc.StartTrial(t.Context(), orgB)
	if err != nil || !again.TrialEndsAt.Equal(got.TrialEndsAt) {
		t.Errorf("second StartTrial() = %+v, %v, want the first trial kept", again, err)
	}
	if n := w.audit.by(auditevent.AgentOperator); n != 1 {
		t.Errorf("%d audit events, want one", n)
	}
	delete(w.repo.plans, "basic")
	if _, err := w.svc.StartTrial(t.Context(), "0c000000-0000-4000-8000-000000000003"); !errors.Is(err, subscription.ErrNoPlan) {
		t.Errorf("without the plan: %v, want ErrNoPlan", err)
	}
}

// The monthly invoice: once per clinic and month, seats times the plan's price, for the month in the clinic's zone;
// the first one ends the trial, audited as the job.
func TestIssueMonthly_IssuesOneInvoicePerClinicAndMonth(t *testing.T) {
	w := newWorld(t)
	inv, err := w.svc.IssueMonthly(t.Context(), orgA, w.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if inv.Seats != 3 || inv.AmountMinor != 30000 || inv.Currency != "BAM" || inv.Number != 1 {
		t.Errorf("invoice = %+v, want 3 seats, 300.00 BAM, number 1", inv)
	}
	if !inv.PeriodStart.Equal(dec1) || !inv.PeriodEnd.Equal(jan1) || !inv.DueAt.Equal(w.clock.Now().Add(subscription.PaymentTerm)) {
		t.Errorf("invoice = %+v, want December in Sarajevo, due after the payment term", inv)
	}
	if sub := w.repo.subs[orgA]; sub.Status != subscription.StatusActive || !sub.CurrentPeriodStart.Equal(dec1) {
		t.Errorf("membership = %+v, want active in December", sub)
	}
	w.seats[orgA] = 9 // a role added mid-month is billed from the next month
	again, err := w.svc.IssueMonthly(t.Context(), orgA, w.clock.Now().AddDate(0, 0, 10))
	if err != nil || again.Number != 1 || again.Seats != 3 || len(w.repo.invoices) != 1 {
		t.Errorf("second run = %+v, %v with %d invoices, want the first invoice", again, err, len(w.repo.invoices))
	}
	if n := w.audit.by(subscription.AgentJob); n != 1 {
		t.Errorf("%d audit events of the job, want one: the trial's end", n)
	}
	next, err := w.svc.IssueMonthly(t.Context(), orgA, jan1)
	if err != nil || next.Number != 2 || next.Seats != 9 {
		t.Errorf("January = %+v, %v, want number 2 with 9 seats", next, err)
	}
}

// No invoice for a trial that has not ended when the month starts, a cancelled membership, or a clinic without one.
func TestIssueMonthly_BillsNoTrialAndNoCancelledClinic(t *testing.T) {
	w := newWorld(t)
	w.repo.subs[orgA] = subscription.Subscription{OrganizationID: orgA, PlanCode: "basic", Status: subscription.StatusTrial, TrialEndsAt: dec1.Add(time.Hour)}
	if _, err := w.svc.IssueMonthly(t.Context(), orgA, w.clock.Now()); !errors.Is(err, subscription.ErrNotBilled) {
		t.Errorf("in trial: %v, want ErrNotBilled", err)
	}
	w.repo.subs[orgA] = subscription.Subscription{OrganizationID: orgA, PlanCode: "basic", Status: subscription.StatusCancelled}
	if _, err := w.svc.IssueMonthly(t.Context(), orgA, w.clock.Now()); !errors.Is(err, subscription.ErrNotBilled) {
		t.Errorf("cancelled: %v, want ErrNotBilled", err)
	}
	if _, err := w.svc.IssueMonthly(t.Context(), orgB, w.clock.Now()); !errors.Is(err, subscription.ErrNotFound) {
		t.Errorf("no membership: %v, want ErrNotFound", err)
	}
	if len(w.repo.invoices) != 0 || w.status(orgA) != subscription.StatusCancelled {
		t.Errorf("%d invoices, status %s: want none and unchanged", len(w.repo.invoices), w.status(orgA))
	}
}

// A clinic that does not pay moves forward only: past due after the due date, suspended 30 days past it (new bookings
// stop), each move audited; paying returns it to active.
func TestReview_MovesAnUnpaidClinicForwardAndAPaymentBack(t *testing.T) {
	w := newWorld(t)
	inv, err := w.svc.IssueMonthly(t.Context(), orgA, w.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	steps := []struct {
		at      time.Time
		want    subscription.Status
		stopped bool
	}{
		{at: inv.DueAt, want: subscription.StatusActive},
		{at: inv.DueAt.Add(time.Minute), want: subscription.StatusPastDue},
		{at: inv.DueAt.Add(subscription.SuspendAfter - time.Minute), want: subscription.StatusPastDue},
		{at: inv.DueAt.Add(subscription.SuspendAfter), want: subscription.StatusSuspended, stopped: true},
	}
	for _, step := range steps {
		w.clock.T = step.at
		got, err := w.svc.Review(t.Context(), orgA)
		if err != nil || got != step.want || w.status(orgA) != step.want {
			t.Errorf("at %v: Review() = %s, %v (stored %s), want %s", step.at, got, err, w.status(orgA), step.want)
		}
		if stopped, err := w.svc.BookingsStopped(t.Context(), orgA); err != nil || stopped != step.stopped {
			t.Errorf("at %v: BookingsStopped() = %v, %v, want %v", step.at, stopped, err, step.stopped)
		}
	}
	if n := w.audit.by(subscription.AgentJob); n != 3 { // the trial's end, past due, suspended
		t.Errorf("%d audit events of the job, want 3", n)
	}
	cur, err := w.svc.Current(t.Context(), admin)
	if err != nil || !cur.PastDueSince.Equal(inv.DueAt) {
		t.Errorf("Current() = %+v, %v, want past due since the invoice's due date", cur, err)
	}
	if _, err := w.svc.MarkPaid(t.Context(), 99, w.clock.Now()); !errors.Is(err, subscription.ErrNotFound) {
		t.Errorf("MarkPaid(unknown) = %v, want ErrNotFound", err)
	}
	paid, err := w.svc.MarkPaid(t.Context(), inv.Number, w.clock.Now())
	if err != nil || paid.Status != subscription.InvoicePaid || !paid.PaidAt.Equal(w.clock.Now()) {
		t.Fatalf("MarkPaid() = %+v, %v", paid, err)
	}
	if w.status(orgA) != subscription.StatusActive {
		t.Errorf("after the payment: %s, want active", w.status(orgA))
	}
	if n := w.audit.by(auditevent.AgentOperator); n != 2 { // the payment, the return to active
		t.Errorf("%d audit events of salusctl, want 2", n)
	}
	if _, err := w.svc.MarkPaid(t.Context(), inv.Number, w.clock.Now()); !errors.Is(err, subscription.ErrNotFound) {
		t.Errorf("MarkPaid(paid) = %v, want ErrNotFound", err)
	}
	if stopped, _ := w.svc.BookingsStopped(t.Context(), orgB); stopped {
		t.Error("a clinic without a membership is stopped, want it to book as before")
	}
}

// One payment does not lift the stop while an older invoice is still past due.
func TestMarkPaid_KeepsAClinicStoppedWhileAnotherInvoiceIsPastDue(t *testing.T) {
	w := newWorld(t)
	first, _ := w.svc.IssueMonthly(t.Context(), orgA, w.clock.Now())
	second, err := w.svc.IssueMonthly(t.Context(), orgA, jan1)
	if err != nil {
		t.Fatal(err)
	}
	w.clock.T = first.DueAt.Add(subscription.SuspendAfter)
	if got, _ := w.svc.Review(t.Context(), orgA); got != subscription.StatusSuspended {
		t.Fatalf("Review() = %s, want suspended", got)
	}
	if _, err := w.svc.MarkPaid(t.Context(), second.Number, w.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if w.status(orgA) != subscription.StatusSuspended {
		t.Errorf("status %s, want suspended: the first invoice is unpaid", w.status(orgA))
	}
}

// salusctl moves a status forward only.
func TestSetStatus_MovesOnlyForward(t *testing.T) {
	w := newWorld(t)
	if _, err := w.svc.SetStatus(t.Context(), orgA, subscription.StatusSuspended); err != nil {
		t.Fatal(err)
	}
	for _, to := range []subscription.Status{subscription.StatusTrial, subscription.StatusActive, subscription.StatusSuspended, "paused"} {
		if _, err := w.svc.SetStatus(t.Context(), orgA, to); !errors.Is(err, subscription.ErrStatus) {
			t.Errorf("suspended to %s: %v, want ErrStatus", to, err)
		}
	}
	got, err := w.svc.SetStatus(t.Context(), orgA, subscription.StatusCancelled)
	if err != nil || got.Status != subscription.StatusCancelled {
		t.Errorf("to cancelled: %+v, %v", got, err)
	}
	if _, err := w.svc.SetStatus(t.Context(), orgB, subscription.StatusActive); !errors.Is(err, subscription.ErrNotFound) {
		t.Errorf("no membership: %v, want ErrNotFound", err)
	}
	if n := w.audit.by(auditevent.AgentOperator); n != 2 {
		t.Errorf("%d audit events, want one per move", n)
	}
}

// The owner's plans are checked before they are stored.
func TestPutPlan_RefusesAMalformedPlan(t *testing.T) {
	w := newWorld(t)
	good := subscription.Plan{Code: "pro", Name: " Pro ", Currency: "BAM", PricePerSeatMinor: 1, Active: true}
	if err := w.svc.PutPlan(t.Context(), good); err != nil || w.repo.plans["pro"].Name != "Pro" {
		t.Fatalf("PutPlan() = %v, stored %+v", err, w.repo.plans["pro"])
	}
	for name, change := range map[string]func(*subscription.Plan){
		"code":     func(p *subscription.Plan) { p.Code = "Pro plan" },
		"name":     func(p *subscription.Plan) { p.Name = " " },
		"price":    func(p *subscription.Plan) { p.PricePerSeatMinor = -1 },
		"currency": func(p *subscription.Plan) { p.Currency = "km" },
	} {
		p := good
		change(&p)
		if err := w.svc.PutPlan(t.Context(), p); !errors.Is(err, subscription.ErrInvalidPlan) {
			t.Errorf("%s: %v, want ErrInvalidPlan", name, err)
		}
	}
}

// The scheduler's run issues every clinic's invoice once; a clinic that fails does not stop the others.
func TestRunAll_IssuesEveryClinicsInvoiceOnce(t *testing.T) {
	w := newWorld(t)
	w.repo.subs[orgB] = subscription.Subscription{OrganizationID: orgB, PlanCode: "basic", Status: subscription.StatusActive}
	w.seats[orgA] = -1 // its seats cannot be counted
	issued, err := w.svc.RunAll(t.Context(), w.clock.Now())
	if issued != 1 || !errors.Is(err, errBroken) {
		t.Errorf("RunAll() = %d, %v, want clinic B's invoice and clinic A's failure", issued, err)
	}
	w.seats[orgA] = 3
	if issued, err := w.svc.RunAll(t.Context(), w.clock.Now()); issued != 1 || err != nil {
		t.Errorf("second RunAll() = %d, %v, want clinic A's invoice only", issued, err)
	}
	if issued, err := w.svc.RunAll(t.Context(), w.clock.Now()); issued != 0 || err != nil || len(w.repo.invoices) != 2 {
		t.Errorf("third RunAll() = %d, %v with %d invoices, want nothing new", issued, err, len(w.repo.invoices))
	}
}

// The monthly job carries opaque ids only.
func TestMonthlyArgs_IsTheSubscriptionMonthlyJob(t *testing.T) {
	if got := (subscription.MonthlyArgs{OrganizationID: "o"}).Kind(); got != "subscription_monthly" {
		t.Errorf("Kind() = %q, want subscription_monthly", got)
	}
}
