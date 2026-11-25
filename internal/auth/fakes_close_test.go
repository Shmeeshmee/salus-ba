package auth_test

import (
	"context"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
)

func (r *fakeRepository) CloseAccount(ctx context.Context, id string, at, eraseAfter time.Time) error {
	r.seen(ctx, "CloseAccount")
	a, ok := r.accounts[id]
	if !ok || a.Role != auth.RolePatient || !a.ClosedAt.IsZero() {
		return auth.ErrNotFound
	}
	a.ClosedAt, a.EraseAfter = at, eraseAfter
	r.accounts[id] = a
	return r.err
}

func (r *fakeRepository) ReopenAccount(ctx context.Context, id string, now time.Time) error {
	r.seen(ctx, "ReopenAccount")
	a, ok := r.accounts[id]
	if !ok || a.ClosedAt.IsZero() || !a.EraseAfter.After(now) {
		return auth.ErrNotFound
	}
	a.ClosedAt, a.EraseAfter = time.Time{}, time.Time{}
	r.accounts[id] = a
	return r.err
}

func (r *fakeRepository) due(a auth.Account, now time.Time) bool {
	return !a.ClosedAt.IsZero() && !a.EraseAfter.After(now) && a.ErasedAt.IsZero()
}

func (r *fakeRepository) AccountsDueForErasure(ctx context.Context, now time.Time) ([]string, error) {
	r.seen(ctx, "AccountsDueForErasure")
	var ids []string
	for id, a := range r.accounts {
		if r.due(a, now) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids, r.err
}

func (r *fakeRepository) EraseAccount(ctx context.Context, id string, now time.Time) error {
	r.seen(ctx, "EraseAccount")
	a, ok := r.accounts[id]
	if !ok || !r.due(a, now) {
		return auth.ErrNotFound
	}
	a.Email, a.PasswordHash, a.Language, a.DetectedLanguage = id+"@erased.invalid", "", "", ""
	a.MFAEnrolledAt, a.ErasedAt = time.Time{}, now
	r.accounts[id] = a
	delete(r.secrets, id)
	delete(r.lastStep, id)
	return r.err
}
