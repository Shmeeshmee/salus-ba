package auditevent

import (
	"context"
	"fmt"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/middleware"
)

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Service records audit events.
type Service struct {
	repo  Repository
	tx    Transactor
	clock clock.Clock
}

// NewService returns a Service storing events in repo.
func NewService(repo Repository, tx Transactor, clk clock.Clock) *Service {
	return &Service{repo: repo, tx: tx, clock: clk}
}

// Record stores e, stamped with the current time and the context's request id. Called inside the caller's
// WithTx it joins that transaction, so the event commits or rolls back with the access it records.
func (s *Service) Record(ctx context.Context, e AuditEvent) error {
	if !valid(e) {
		return ErrInvalid
	}
	e.Recorded = s.clock.Now()
	e.RequestID = middleware.RequestIDFrom(ctx)
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Insert(ctx, e); err != nil {
			return fmt.Errorf("auditevent: record: %w", err)
		}
		return nil
	})
}

func valid(e AuditEvent) bool {
	for _, c := range []Coding{e.Type, e.Subtype, e.Purpose} {
		if c.System == "" || c.Code == "" {
			return false
		}
	}
	switch e.Action {
	case ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionExecute:
	default:
		return false
	}
	switch e.Outcome {
	case OutcomeSuccess, OutcomeMinorFailure, OutcomeSeriousFailure, OutcomeMajorFailure:
		return true
	}
	return false
}
