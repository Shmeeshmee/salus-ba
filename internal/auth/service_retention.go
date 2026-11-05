package auth

import (
	"context"
	"time"
)

// SessionPurger deletes the sessions whose retention period is over (internal/retention: 30 days after
// revoked_at or expires_at). It is the retention job's applier for session, built on the retention role's
// pool in internal/worker; the period is the rule's, the cut-off the retention service's.
type SessionPurger struct {
	repo Repository
}

// NewSessionPurger returns the purger of session on repo.
func NewSessionPurger(repo Repository) *SessionPurger {
	return &SessionPurger{repo: repo}
}

// Purge deletes the sessions that ended before before and returns how many.
func (p *SessionPurger) Purge(ctx context.Context, before time.Time) (int64, error) {
	return p.repo.DeleteSessionsEndedBefore(ctx, before)
}

// InvitePurger deletes the invites whose retention period is over (30 days after accepted_at or expires_at),
// in every clinic, like SessionPurger.
type InvitePurger struct {
	repo Repository
}

// NewInvitePurger returns the purger of invite on repo.
func NewInvitePurger(repo Repository) *InvitePurger {
	return &InvitePurger{repo: repo}
}

// Purge deletes the invites that ended before before and returns how many.
func (p *InvitePurger) Purge(ctx context.Context, before time.Time) (int64, error) {
	return p.repo.DeleteInvitesEndedBefore(ctx, before)
}
