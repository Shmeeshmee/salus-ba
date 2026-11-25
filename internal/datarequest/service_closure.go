package datarequest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Scope sets the erasure scope on the transaction in ctx (app.erasure, 00075): the account's own erasure request of a
// closed account, read and written by the closure and the erasure job.
type Scope interface {
	SetErasure(ctx context.Context) error
}

// Accounts closes, reopens and erases accounts (auth.Service).
type Accounts interface {
	Close(ctx context.Context, accountID, password string, eraseAfter time.Duration) error
	Reopen(ctx context.Context, accountID string) error
	DueForErasure(ctx context.Context) ([]string, error)
	Erase(ctx context.Context, accountID string) error
}

// ClosureConsents lists and revokes the patient's consents (consent.Service); a care consent's revocation cancels the future
// visits with the clinic and withdraws the waitlist entries there, with their notices (the consent_revoked job).
type ClosureConsents interface {
	Consents
	Revoke(ctx context.Context, c consent.Caller, id string) (consent.Consent, error)
	// RevokeChats ends every active chat consent of the patient (which Mine does not list), audited.
	RevokeChats(ctx context.Context, c consent.Caller) ([]consent.Consent, error)
}

// Devices removes the push devices of an account (notification.Chat).
type Devices interface {
	RemoveDevices(ctx context.Context, accountID string) (int64, error)
}

// ClosurePorts are what Closure runs on.
type ClosurePorts struct {
	Tx       Transactor
	Repo     Repository
	Scope    Scope
	Accounts Accounts
	Profiles Profiles
	Consents ClosureConsents
	Devices  Devices
	Clock    clock.Clock
}

// Closure closes a patient's account (docs/clinic-cms/WAVE5.md S5) and reopens it for the operator.
type Closure struct {
	p ClosurePorts
}

// NewClosure builds a Closure on p.
func NewClosure(p ClosurePorts) *Closure {
	return &Closure{p: p}
}

// CloseAccount closes the patient's account once the password confirms it (S5), in one transaction: sign-in ends and
// the sessions are revoked, every active consent is revoked (which cancels the future visits and withdraws the
// waitlist entries, with their notices), the push devices are removed, and the account's erasure request is filed and
// kept in progress until the erasure EraseAfter later (S6) answers it. Not reversible by the patient. ErrWrongPassword
// for a wrong password, ErrNotFound for an account that is not an open patient account, and auth's RateLimitedError
// (wrapped) after too many wrong passwords.
func (c *Closure) CloseAccount(ctx context.Context, p Patient, password string) (Request, error) {
	profile, err := c.p.Profiles.Me(ctx, patient.Caller{AccountID: p.AccountID})
	if err != nil && !errors.Is(err, patient.ErrNotFound) {
		return Request{}, fmt.Errorf("datarequest: close account: %w", err)
	}
	var out Request
	err = c.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.p.Accounts.Close(ctx, p.AccountID, password, EraseAfter); err != nil {
			return err
		}
		// The request first: the consents' scope (the patient's profile) is not the erasure scope.
		var err error
		if out, err = c.fileErasure(ctx, p.AccountID, profile.ID); err != nil {
			return err
		}
		if profile.ID != "" {
			if err := c.revokeConsents(ctx, consent.Caller{AccountID: p.AccountID, PatientID: profile.ID}); err != nil {
				return err
			}
		}
		_, err = c.p.Devices.RemoveDevices(ctx, p.AccountID)
		return err
	})
	switch {
	case errors.Is(err, auth.ErrWrongCredentials):
		return Request{}, ErrWrongPassword
	case errors.Is(err, auth.ErrNotFound):
		return Request{}, ErrNotFound
	case err != nil:
		return Request{}, fmt.Errorf("datarequest: close account: %w", err)
	}
	return out, nil
}

// fileErasure files the account's erasure request, received now and due one month later (after the erasure date), and
// moves it in progress, in the erasure scope.
func (c *Closure) fileErasure(ctx context.Context, accountID, patientID string) (Request, error) {
	if err := c.p.Scope.SetErasure(ctx); err != nil {
		return Request{}, err
	}
	now := c.p.Clock.Now()
	r, err := c.p.Repo.Insert(ctx, Request{
		AccountID: accountID, PatientID: patientID, FiledBy: accountID, Kind: KindErasure, Status: StatusReceived,
		ReceivedAt: now, DueAt: now.AddDate(0, DueMonths, 0), Version: 1,
	})
	if err != nil {
		return Request{}, err
	}
	r.Status, r.Version = StatusInProgress, r.Version+1
	if ok, err := c.p.Repo.Update(ctx, r, ""); err != nil || !ok {
		return Request{}, errors.Join(err, ErrVersionConflict)
	}
	return r, nil
}

// revokeConsents revokes every active consent of the patient, the care consents first: theirs end the others with the
// clinic, which then read as inactive; then the chat consents, so no doctor of another clinic the patient invited
// into a conversation keeps access (owner, 2026-11-25).
func (c *Closure) revokeConsents(ctx context.Context, caller consent.Caller) error {
	all, err := c.p.Consents.Mine(ctx, caller)
	if err != nil {
		return err
	}
	slices.SortStableFunc(all, func(a, b consent.Consent) int {
		switch {
		case a.Kind == b.Kind, a.Kind != consent.KindCare && b.Kind != consent.KindCare:
			return 0
		case a.Kind == consent.KindCare:
			return -1
		}
		return 1
	})
	for _, x := range all {
		if x.Status != consent.StatusActive {
			continue
		}
		if _, err := c.p.Consents.Revoke(ctx, caller, x.ID); err != nil && !errors.Is(err, consent.ErrInactive) {
			return err
		}
	}
	_, err = c.p.Consents.RevokeChats(ctx, caller)
	return err
}

// ReopenAccount reopens a closed account before its erasure date (the operator's, from salusctl as the tables' owner):
// the account's erasure request is withdrawn and the closure cleared. The consents revoked and the visits cancelled at
// closure stay so; the patient signs in again. It returns the withdrawn request. ErrNotFound for an account with no
// erasure in progress or past its date.
func (c *Closure) ReopenAccount(ctx context.Context, accountID string) (Request, error) {
	var out Request
	err := c.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		r, err := openErasure(ctx, c.p.Scope, c.p.Repo, accountID)
		if err != nil {
			return err
		}
		r.Status, r.Version = StatusWithdrawn, r.Version+1
		if ok, err := c.p.Repo.Update(ctx, r, ""); err != nil || !ok {
			return errors.Join(err, ErrVersionConflict)
		}
		out = r
		return c.p.Accounts.Reopen(ctx, accountID)
	})
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, auth.ErrNotFound):
		return Request{}, ErrNotFound
	case err != nil:
		return Request{}, fmt.Errorf("datarequest: reopen account: %w", err)
	}
	return out, nil
}

// openErasure returns the account's erasure request that is still open, in the erasure scope, or ErrNotFound.
func openErasure(ctx context.Context, scope Scope, repo Repository, accountID string) (Request, error) {
	if err := scope.SetErasure(ctx); err != nil {
		return Request{}, err
	}
	all, err := repo.ForAccount(ctx, accountID)
	if err != nil {
		return Request{}, err
	}
	for _, r := range all {
		if r.Kind == KindErasure && r.Target() == TargetAccount && (r.Status == StatusReceived || r.Status == StatusInProgress) {
			return r, nil
		}
	}
	return Request{}, ErrNotFound
}
