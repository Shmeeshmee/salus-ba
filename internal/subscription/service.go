package subscription

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// AgentJob names the monthly job as the agent of the status moves it makes.
const AgentJob = "subscription_monthly"

var (
	planCode = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	currency = regexp.MustCompile(`^[A-Z]{3}$`)
)

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Clinics gives a clinic's time zone: its month starts at its own midnight (organization.Service).
type Clinics interface {
	Clinic(ctx context.Context, organizationID string) (organization.Organization, error)
}

// Seats counts a clinic's active practitioner roles (practitionerrole.Service).
type Seats interface {
	ActiveCount(ctx context.Context, org string) (int, error)
}

// Ports are what the Service runs on. Clinics and Seats are needed only by a Service that issues invoices.
type Ports struct {
	Repo    Repository
	Audit   Auditor
	Tx      Transactor
	Clock   clock.Clock
	Clinics Clinics
	Seats   Seats
}

// Service holds the membership's rules: the clinic's admin sees its membership and invoices, the monthly job issues
// one invoice per clinic and month and moves a clinic that does not pay to past_due and suspended, the owner sets
// plans, marks invoices paid and moves statuses in salusctl. Every status move is audited. Reads are not: nothing in
// a membership is personal data.
type Service struct{ Ports }

// NewService returns a Service on its ports.
func NewService(p Ports) *Service { return &Service{Ports: p} }

// Current returns the clinic's membership to its admin, with its plan and since when it is past due
// (GET /clinic/subscription). ErrForbidden for its other staff, ErrNotFound for anyone else and for a clinic without
// a membership.
func (s *Service) Current(ctx context.Context, c Caller) (Subscription, error) {
	if err := adminOf(c); err != nil {
		return Subscription{}, err
	}
	var out Subscription
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = s.Repo.Subscription(ctx, c.OrganizationID); err != nil {
			return err
		}
		if out.Plan, err = s.Repo.Plan(ctx, out.PlanCode); err != nil {
			return err
		}
		out.PastDueSince, err = s.Repo.OldestPastDue(ctx, c.OrganizationID, s.Clock.Now())
		return err
	})
	return out, wrap("current", err)
}

// Invoices returns the clinic's monthly invoices to its admin, newest first (GET /clinic/subscription/invoices).
// Refusals as Current's.
func (s *Service) Invoices(ctx context.Context, c Caller) ([]Invoice, error) {
	if err := adminOf(c); err != nil {
		return nil, err
	}
	var out []Invoice
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		out, err = s.Repo.Invoices(ctx, c.OrganizationID)
		return err
	})
	return out, wrap("invoices", err)
}

// BookingsStopped tells whether the clinic org takes no new bookings: its membership is suspended (30 days past due)
// or cancelled. A clinic without a membership books as before. Only new bookings ask: care never stops.
func (s *Service) BookingsStopped(ctx context.Context, org string) (bool, error) {
	var sub Subscription
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		var err error
		sub, err = s.Repo.Subscription(ctx, org)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return false, nil
	case err != nil:
		return false, wrap("bookings", err)
	}
	return sub.Status == StatusSuspended || sub.Status == StatusCancelled, nil
}

// StartTrial gives the clinic org its 30-day trial of the default plan (the operator's clinic creation), audited as
// salusctl. A clinic that has a membership keeps it. ErrNoPlan while the owner has not set the default plan.
func (s *Service) StartTrial(ctx context.Context, org string) (Subscription, error) {
	var out Subscription
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		plan, err := s.Repo.Plan(ctx, DefaultPlan)
		if err != nil {
			return err
		}
		if !plan.Active {
			return ErrNoPlan
		}
		now := s.Clock.Now()
		created, err := s.Repo.Insert(ctx, org, plan.Code, now, now.Add(TrialLength))
		if err != nil {
			return err
		}
		if created {
			if err := s.Audit.Record(ctx, event(auditevent.CodeCreate, auditevent.ActionCreate, auditevent.AgentOperator, org)); err != nil {
				return err
			}
		}
		out, err = s.Repo.Subscription(ctx, org)
		return err
	})
	return out, wrap("start trial", err)
}

// PutPlan stores or replaces a plan (salusctl, as the owner). ErrInvalidPlan for a malformed one. Not audited: a plan
// is reference data.
func (s *Service) PutPlan(ctx context.Context, p Plan) error {
	p.Name = strings.TrimSpace(p.Name)
	if !planCode.MatchString(p.Code) || p.Name == "" || p.PricePerSeatMinor < 0 || !currency.MatchString(p.Currency) {
		return ErrInvalidPlan
	}
	return wrap("plan", s.Tx.WithTx(ctx, func(ctx context.Context) error { return s.Repo.PutPlan(ctx, p) }))
}

// SetStatus moves the clinic org's membership forward to to (salusctl), audited. ErrStatus for a move that is not
// forward, ErrNotFound for a clinic without a membership.
func (s *Service) SetStatus(ctx context.Context, org string, to Status) (Subscription, error) {
	var out Subscription
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		sub, err := s.Repo.Lock(ctx, org)
		if err != nil {
			return err
		}
		if rank[to] <= rank[sub.Status] {
			return ErrStatus
		}
		if err := s.move(ctx, org, to, auditevent.AgentOperator); err != nil {
			return err
		}
		out, err = s.Repo.Subscription(ctx, org)
		return err
	})
	return out, wrap("status", err)
}

// MarkPaid marks the open invoice number paid at at (salusctl, as the owner), audited. A clinic that was past due or
// suspended and has no invoice past due any more returns to active. ErrNotFound when no open invoice has that number.
func (s *Service) MarkPaid(ctx context.Context, number int64, at time.Time) (Invoice, error) {
	var out Invoice
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.Repo.MarkPaid(ctx, number, at); err != nil {
			return err
		}
		org := out.OrganizationID
		if err := s.Audit.Record(ctx, event(auditevent.CodeUpdate, auditevent.ActionUpdate, auditevent.AgentOperator, org)); err != nil {
			return err
		}
		sub, err := s.Repo.Lock(ctx, org)
		if err != nil {
			return err
		}
		if sub.Status != StatusPastDue && sub.Status != StatusSuspended {
			return nil
		}
		since, err := s.Repo.OldestPastDue(ctx, org, s.Clock.Now())
		if err != nil || !since.IsZero() {
			return err
		}
		return s.move(ctx, org, StatusActive, auditevent.AgentOperator)
	})
	return out, wrap("mark paid", err)
}

// move sets the clinic's status in the transaction in ctx and audits it as agent.
func (s *Service) move(ctx context.Context, org string, to Status, agent string) error {
	if err := s.Repo.SetStatus(ctx, org, to, s.Clock.Now()); err != nil {
		return err
	}
	return s.Audit.Record(ctx, event(auditevent.CodeUpdate, auditevent.ActionUpdate, agent, org))
}

// adminOf is nil for the admin of a clinic, ErrForbidden for its other staff and ErrNotFound for anyone else.
func adminOf(c Caller) error {
	switch {
	case c.OrganizationID == "":
		return ErrNotFound
	case !c.Admin:
		return ErrForbidden
	}
	return nil
}

// event is a write of the clinic org's membership by an agent that is no account (salusctl or the monthly job), for
// operations. It names the clinic only.
func event(interaction string, action auditevent.Action, agent, org string) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          action,
		Outcome:         auditevent.OutcomeSuccess,
		AgentName:       agent,
		EntityReference: "Organization/" + org,
		OrganizationID:  org,
	}
}

// wrap names the operation on an unexpected error; the domain's own errors pass as they are.
func wrap(op string, err error) error {
	for _, known := range []error{ErrNotFound, ErrForbidden, ErrStatus, ErrNoPlan, ErrInvalidPlan, ErrNotBilled} {
		if errors.Is(err, known) {
			return known
		}
	}
	if err != nil {
		return fmt.Errorf("subscription: %s: %w", op, err)
	}
	return nil
}
