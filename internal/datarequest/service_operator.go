package datarequest

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Reopener reopens a closed account before its erasure and returns its withdrawn erasure request (Closure).
type Reopener interface {
	ReopenAccount(ctx context.Context, accountID string) (Request, error)
}

// Operator is the operator's handling of the requests to the account (S1: Salus answers them), from salusctl on the
// tables' owner's connection, which row-level security does not hold: the methods check the target themselves. Every
// call is audited with salusctl as the agent (docs/clinic-cms/WAVE5.md D5).
type Operator struct {
	s      *Service // its Repo, Tx, Audit and Clock: the owner's
	closer Reopener
}

// NewOperator builds the Operator on s, the Service on the owner's connection, and closer, the Closure on it.
func NewOperator(s *Service, closer Reopener) *Operator {
	return &Operator{s: s, closer: closer}
}

// Requests returns the requests to the account, the open ones by deadline first, audited as one search.
func (o *Operator) Requests(ctx context.Context) ([]Request, error) {
	out := []Request{}
	err := o.s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		all, err := o.s.p.Repo.ForOperator(ctx)
		if err != nil {
			return err
		}
		out = append(out, all...)
		return o.s.p.Audit.Record(ctx, operatorEvent(Request{}, auditevent.CodeSearchType, int64(len(out))))
	})
	if err != nil {
		return nil, wrap("operator requests", err)
	}
	return out, nil
}

// Request returns one request to the account with its history of status changes, audited. ErrNotFound for none, or
// one to a clinic.
func (o *Operator) Request(ctx context.Context, id string) (Request, []Change, error) {
	var r Request
	var history []Change
	err := o.s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if r, err = o.accounts(ctx, id); err != nil {
			return err
		}
		if history, err = o.s.p.Repo.History(ctx, id); err != nil {
			return err
		}
		return o.s.p.Audit.Record(ctx, operatorEvent(r, auditevent.CodeRead, 1))
	})
	if err != nil {
		return Request{}, nil, wrap("operator request", err)
	}
	return r, history, nil
}

// Answer fulfils or refuses an open request to the account at the version read, with a code that fits, as the
// clinic's admin answers theirs (Service.Answer). Its history entry names no account. ErrInvalid, ErrNotFound,
// ErrClosed, ErrVersionConflict. Audited.
func (o *Operator) Answer(ctx context.Context, id string, a Answer) (Request, error) {
	if !a.valid() {
		return Request{}, ErrInvalid
	}
	return o.change(ctx, "operator answer", id, a.Version, func(r Request) (Request, error) {
		r.Status, r.AnswerCode, r.AnsweredAt = a.Status, a.Code, o.s.p.Clock.Now()
		return r, nil
	})
}

// Extend extends an open request to the account once, by ExtensionMonths after its due date in the operator's time
// zone, with its reason (Art. 12(3)), while its due date has not passed (S12). ErrInvalid for an unknown reason,
// ErrAlreadyExtended, ErrOverdue, ErrNotFound, ErrClosed, ErrVersionConflict. Audited.
func (o *Operator) Extend(ctx context.Context, id string, reason ExtensionReason, version int) (Request, error) {
	if reason != ExtensionComplex && reason != ExtensionManyRequests {
		return Request{}, ErrInvalid
	}
	loc, err := o.s.location(ctx, "")
	if err != nil {
		return Request{}, err
	}
	now := o.s.p.Clock.Now()
	return o.change(ctx, "operator extend", id, version, func(r Request) (Request, error) {
		switch {
		case !r.ExtendedUntil.IsZero():
			return Request{}, ErrAlreadyExtended
		case r.overdue(now):
			return Request{}, ErrOverdue
		}
		r.ExtendedUntil, r.ExtensionReason = extendedUntil(r.DueAt, loc), reason
		return r, nil
	})
}

// ReopenAccount reopens the closed account accountID before its erasure date (S5: within EraseAfter of its closure)
// through the Closure, which withdraws its erasure request; audited as the operator's update of that request (the
// account's own update is audited by auth). ErrNotFound for an account with no erasure in progress or past its date.
func (o *Operator) ReopenAccount(ctx context.Context, accountID string) (Request, error) {
	var out Request
	err := o.s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = o.closer.ReopenAccount(ctx, accountID); err != nil {
			return err
		}
		return o.s.p.Audit.Record(ctx, operatorEvent(out, auditevent.CodeUpdate, 1))
	})
	if err != nil {
		return Request{}, wrap("operator reopen account", err)
	}
	return out, nil
}

// change applies fn to the open request id to the account at version and stores it, audited, in one transaction.
func (o *Operator) change(ctx context.Context, op, id string, version int, fn func(Request) (Request, error)) (Request, error) {
	var out Request
	err := o.s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		r, err := o.accounts(ctx, id) // the target first: a clinic's request is not found, whatever its state
		switch {
		case err != nil:
			return err
		case r.closed(), r.erasing(): // a closure's erasure request is the erasure job's to answer (S6)
			return ErrClosed
		case r.Version != version:
			return ErrVersionConflict
		}
		if r, err = fn(r); err != nil {
			return err
		}
		if out, err = o.s.update(ctx, r, ""); err != nil {
			return err
		}
		return o.s.p.Audit.Record(ctx, operatorEvent(out, auditevent.CodeUpdate, 1))
	})
	if err != nil {
		return Request{}, wrap(op, err)
	}
	return out, nil
}

// accounts returns the request id if it is one to the account, or ErrNotFound.
func (o *Operator) accounts(ctx context.Context, id string) (Request, error) {
	r, err := o.s.p.Repo.ByID(ctx, id)
	if err == nil && r.Target() != TargetAccount {
		return Request{}, ErrNotFound
	}
	return r, err
}

// operatorEvent is the audit event of the operator's interaction with requests: salusctl, which is no account.
func operatorEvent(r Request, interaction string, count int64) auditevent.AuditEvent {
	e := event("", r, interaction, auditevent.CodeHOPERAT, count)
	e.AgentName = auditevent.AgentOperator
	return e
}
