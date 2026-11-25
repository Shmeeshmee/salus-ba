package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// ErasureAgent names the erasure job as the agent of its audit events (docs/clinic-cms/WAVE5.md S6).
const ErasureAgent = "data_request_erasure"

// Close closes the patient's account once password confirms it (docs/clinic-cms/WAVE5.md S5): from the commit no
// sign-in succeeds, every open session is revoked, and the account is erased eraseAfter later unless the operator
// reopens it first. Audited as an update of the account, in the transaction in ctx when there is one (the closure's,
// datarequest). ErrWrongCredentials for a wrong password, ErrNotFound for staff or an account closed already. After
// maxFailuresPerEmail wrong passwords in a loginWindow it returns a RateLimitedError without checking the password
// (docs/clinic-cms/WAVE6.md PX).
func (s *Service) Close(ctx context.Context, accountID, password string, eraseAfter time.Duration) error {
	a, err := s.repo.ByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("auth: close: %w", err)
	}
	if a.Role != RolePatient || !a.ClosedAt.IsZero() {
		return ErrNotFound
	}
	key := s.counterKey("close:account:", accountID)
	n, left, err := s.counters.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("auth: close: rate limit: %w", err)
	}
	if n >= maxFailuresPerEmail {
		return &RateLimitedError{RetryAfter: left}
	}
	ok, _, err := s.hasher.Verify(ctx, password, a.PasswordHash)
	if err != nil {
		return fmt.Errorf("auth: close: %w", err)
	}
	if !ok {
		if _, _, err := s.counters.Incr(ctx, key, loginWindow); err != nil {
			return fmt.Errorf("auth: close: count the failure: %w", err)
		}
		return ErrWrongCredentials
	}
	at := s.clock.Now()
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.CloseAccount(ctx, accountID, at, at.Add(eraseAfter)); err != nil {
			return err
		}
		if err := s.repo.RevokeSessions(ctx, accountID, at); err != nil {
			return err
		}
		return s.audit.Record(ctx, event(restCoding, interaction(auditevent.CodeUpdate), auditevent.ActionUpdate, a, auditevent.OutcomeSuccess))
	})
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("auth: close: %w", err)
	}
	return nil
}

// Reopen clears the closure of account id before its erasure date (the operator's, from salusctl). Sessions revoked
// at closure stay revoked: the patient signs in again. ErrNotFound for an account that is open or due for erasure.
func (s *Service) Reopen(ctx context.Context, id string) error {
	return s.byJob(ctx, id, auditevent.AgentOperator, func(ctx context.Context) error {
		return s.repo.ReopenAccount(ctx, id, s.clock.Now())
	})
}

// DueForErasure returns the closed accounts whose erasure date has come and that are not erased yet.
func (s *Service) DueForErasure(ctx context.Context) ([]string, error) {
	ids, err := s.repo.AccountsDueForErasure(ctx, s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("auth: due for erasure: %w", err)
	}
	return ids, nil
}

// Erase erases what the account holds that has no legal basis to stay (S6): its e-mail (replaced by one derived from
// the opaque id, so the row stays unique and referenced), its password hash, its MFA secret and enrolment and its
// language choices. The row itself stays: data requests, audit events and the clinic's records reference it.
// Audited as the erasure job's update. ErrNotFound for an account not due or erased already.
func (s *Service) Erase(ctx context.Context, id string) error {
	return s.byJob(ctx, id, ErasureAgent, func(ctx context.Context) error {
		return s.repo.EraseAccount(ctx, id, s.clock.Now())
	})
}

// byJob runs change on account id in a transaction and audits it as an update by the named agent (no account).
func (s *Service) byJob(ctx context.Context, id, agent string, change func(ctx context.Context) error) error {
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := change(ctx); err != nil {
			return err
		}
		e := event(restCoding, interaction(auditevent.CodeUpdate), auditevent.ActionUpdate, Account{}, auditevent.OutcomeSuccess)
		e.AgentName, e.EntityReference = agent, "Account/"+id
		return s.audit.Record(ctx, e)
	})
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("auth: %s: %w", agent, err)
	}
	return nil
}
