package auditevent

import (
	"context"
	"fmt"
	"time"

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

// RecordAll stores the events in one insert, each stamped as Record stamps it, in the caller's transaction: for a
// read that discloses many resources at once (a page of a patient's history, SAL-39). One invalid event stores none
// (ErrInvalid); none is nothing to store.
func (s *Service) RecordAll(ctx context.Context, es []AuditEvent) error {
	if len(es) == 0 {
		return nil
	}
	stamped := make([]AuditEvent, len(es))
	now, request := s.clock.Now(), middleware.RequestIDFrom(ctx)
	for i, e := range es {
		if !valid(e) {
			return ErrInvalid
		}
		e.Recorded, e.RequestID = now, request
		stamped[i] = e
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.InsertAll(ctx, stamped); err != nil {
			return fmt.Errorf("auditevent: record all: %w", err)
		}
		return nil
	})
}

// CountRecordedBefore counts the events recorded before before: the retention job's check that no event is
// past its period while nothing can remove one yet (docs/booking/SPEC.md §2 "Applying the rules"). A count,
// never the events.
func (s *Service) CountRecordedBefore(ctx context.Context, before time.Time) (int64, error) {
	n, err := s.repo.CountRecordedBefore(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("auditevent: count: %w", err)
	}
	return n, nil
}

func valid(e AuditEvent) bool {
	if e.EntityCount < 0 || (e.EntityCount != 0 && e.EntityName == "") {
		return false
	}
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
