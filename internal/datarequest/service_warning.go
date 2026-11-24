package datarequest

import (
	"context"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// AgentWarning names the sending of a deadline warning in its audit events (AuditEvent.agent.name): the job, which is
// no account.
const AgentWarning = "data_request_deadline_warning"

// Warning returns the open request a deadline warning is about (S3, queued by QueueWarnings) and its answerer's time
// zone: the clinic's for a clinic target, OperatorTimeZone for the account. It is read in the deadline job's scope,
// which shows open requests only, and audited as the warning's. ErrNotFound for a request answered or withdrawn since
// the warning was queued, or none, and for a stale warning: its deadline has passed (the send was retried that long) or
// moved later (extended since: the new deadline is warned of in its turn). There is nothing to warn of.
func (s *Service) Warning(ctx context.Context, args WarningArgs) (Request, *time.Location, error) {
	var r Request
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetDeadlines(ctx); err != nil {
			return err
		}
		var err error
		if r, err = s.p.Repo.ByID(ctx, args.RequestID); err != nil {
			return err
		}
		e := event("", r, auditevent.CodeRead, auditevent.CodeHOPERAT, 1)
		e.AgentName = AgentWarning
		return s.p.Audit.Record(ctx, e)
	})
	if err != nil {
		return Request{}, nil, wrap("warning", err)
	}
	now, deadline := s.p.Clock.Now(), r.Deadline()
	if !now.Before(deadline) || deadline.Add(-time.Duration(args.Days)*day).After(now.Add(day)) {
		return Request{}, nil, ErrNotFound
	}
	loc, err := s.location(ctx, r.OrganizationID)
	if err != nil {
		return Request{}, nil, err
	}
	return r, loc, nil
}
