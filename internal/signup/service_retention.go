package signup

import (
	"context"
	"time"
)

// Purger applies the retention rule of clinic_signup (internal/retention): an unverified signup is deleted once its
// period has passed; a verified one stays with its clinic.
type Purger struct {
	repo Repository
}

// NewPurger returns the Purger on repo (salus_retention's connection).
func NewPurger(repo Repository) *Purger {
	return &Purger{repo: repo}
}

// Purge deletes the unverified signups made before before and returns how many.
func (p *Purger) Purge(ctx context.Context, before time.Time) (int64, error) {
	return p.repo.DeleteUnverifiedBefore(ctx, before)
}
