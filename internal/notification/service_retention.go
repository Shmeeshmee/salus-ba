package notification

import (
	"context"
	"time"
)

// DeadLetterPurger applies the retention rule of webhook_dead_letter (internal/retention).
type DeadLetterPurger struct {
	repo Repository
}

// NewDeadLetterPurger returns the purger of webhook_dead_letter on repo, built on the retention role's pool.
func NewDeadLetterPurger(repo Repository) *DeadLetterPurger {
	return &DeadLetterPurger{repo: repo}
}

// Purge deletes the dead letters recorded before before and returns how many.
func (p *DeadLetterPurger) Purge(ctx context.Context, before time.Time) (int64, error) {
	return p.repo.DeleteDeadLettersBefore(ctx, before)
}
