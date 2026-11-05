package slot

import (
	"context"
	"time"
)

// Purger deletes the free slots whose retention period is over (internal/retention: 30 days after their
// end), in every clinic. It is the retention job's applier for slot, built on the retention role's pool in
// internal/worker; a busy slot is kept with its appointment.
type Purger struct {
	repo Repository
}

// NewPurger returns the purger of slot on repo.
func NewPurger(repo Repository) *Purger {
	return &Purger{repo: repo}
}

// Purge deletes the free slots that ended before before and returns how many.
func (p *Purger) Purge(ctx context.Context, before time.Time) (int64, error) {
	return p.repo.DeleteFreeEndedBefore(ctx, before)
}
