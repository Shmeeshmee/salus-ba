package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// IssueMonthly issues the clinic org's invoice for the month that holds month in the clinic's time zone, once: a
// second call returns the first one's invoice. The seats are the clinic's active practitioner roles when it runs (the
// scheduler runs it on the 1st), the price its plan's. The first invoice ends the trial. ErrNotBilled for a cancelled
// membership and for a trial that has not ended when the month starts; ErrNotFound for a clinic without a membership.
func (s *Service) IssueMonthly(ctx context.Context, org string, month time.Time) (Invoice, error) {
	inv, _, err := s.issueMonthly(ctx, org, month)
	return inv, err
}

// issueMonthly is IssueMonthly, telling whether this call issued the invoice.
func (s *Service) issueMonthly(ctx context.Context, org string, month time.Time) (out Invoice, issued bool, err error) {
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		// The lock makes two runs for one clinic take turns: the second finds the first one's invoice.
		sub, err := s.Repo.Lock(ctx, org)
		if err != nil {
			return err
		}
		clinic, err := s.Clinics.Clinic(ctx, org)
		if err != nil {
			return err
		}
		loc, err := time.LoadLocation(clinic.TimeZone)
		if err != nil {
			return fmt.Errorf("the clinic's time zone: %w", err)
		}
		local := month.In(loc)
		start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
		if inv, ok, err := s.Repo.InvoiceOf(ctx, org, start); err != nil || ok {
			out = inv
			return err
		}
		if sub.Status == StatusCancelled || (sub.Status == StatusTrial && sub.TrialEndsAt.After(start)) {
			return ErrNotBilled
		}
		out, err = s.issue(ctx, sub, start)
		issued = err == nil
		return err
	})
	return out, issued && err == nil, wrap("monthly invoice", err)
}

// issue stores the invoice of sub's clinic for the month from start and moves its period, in the transaction in ctx.
func (s *Service) issue(ctx context.Context, sub Subscription, start time.Time) (Invoice, error) {
	org, end, now := sub.OrganizationID, start.AddDate(0, 1, 0), s.Clock.Now()
	plan, err := s.Repo.Plan(ctx, sub.PlanCode)
	if err != nil {
		return Invoice{}, err
	}
	seats, err := s.Seats.ActiveCount(ctx, org)
	if err != nil {
		return Invoice{}, err
	}
	number, err := s.Repo.NextNumber(ctx)
	if err != nil {
		return Invoice{}, err
	}
	inv, err := s.Repo.InsertInvoice(ctx, Invoice{
		PeriodStart: start, PeriodEnd: end, IssuedAt: now, DueAt: now.Add(PaymentTerm), OrganizationID: org, Currency: plan.Currency,
		Number: number, AmountMinor: int64(seats) * plan.PricePerSeatMinor, Seats: int32(seats), //nolint:gosec // a clinic's roles fit
	})
	if err != nil {
		return Invoice{}, err
	}
	if err := s.Repo.SetPeriod(ctx, org, start, end, now); err != nil {
		return Invoice{}, err
	}
	if err := s.Jobs.EnqueueInvoiceEmail(ctx, InvoiceEmailArgs{InvoiceID: inv.ID, OrganizationID: org}); err != nil {
		return Invoice{}, err
	}
	if sub.Status == StatusTrial {
		return inv, s.move(ctx, org, StatusActive, AgentJob)
	}
	return inv, nil
}

// Review moves the clinic org forward when it does not pay: to past_due once an invoice is past its due date, to
// suspended once one is SuspendAfter past it. Audited as the monthly job. It returns the status the clinic is left in.
func (s *Service) Review(ctx context.Context, org string) (Status, error) {
	var out Status
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		sub, err := s.Repo.Lock(ctx, org)
		if err != nil {
			return err
		}
		out = sub.Status
		now := s.Clock.Now()
		since, err := s.Repo.OldestPastDue(ctx, org, now)
		if err != nil || since.IsZero() {
			return err
		}
		to := StatusPastDue
		if !now.Before(since.Add(SuspendAfter)) {
			to = StatusSuspended
		}
		if rank[to] <= rank[sub.Status] {
			return nil
		}
		out = to
		return s.move(ctx, org, to, AgentJob)
	})
	return out, wrap("review", err)
}

// RunAll is the scheduler's daily run: for every clinic with a membership, the invoice of the month that holds month
// (issued once, so on the 1st, or on the first run after it) and the review of what it owes. One clinic's failure does
// not stop the others; the failures are returned joined. It returns how many invoices it issued.
func (s *Service) RunAll(ctx context.Context, month time.Time) (int, error) {
	var orgs []string
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetRun(ctx); err != nil {
			return err
		}
		var err error
		orgs, err = s.Repo.Subscribed(ctx)
		return err
	})
	if err != nil {
		return 0, wrap("monthly run", err)
	}
	var issued int
	var failed []error
	for _, org := range orgs {
		_, ok, err := s.issueMonthly(ctx, org, month)
		switch {
		case ok:
			issued++
		case err != nil && !errors.Is(err, ErrNotBilled):
			failed = append(failed, err)
		}
		if _, err := s.Review(ctx, org); err != nil {
			failed = append(failed, err)
		}
	}
	return issued, errors.Join(failed...)
}
