package trends

import (
	"context"
	"time"
)

// CellPurger and FlagPurger delete the platform figures five years after their build (retention.Purger, on
// salus_retention): the period itself is kept, so it is never built again (A3).
type (
	CellPurger struct{ repo Repository }
	FlagPurger struct{ repo Repository }
)

// NewCellPurger and NewFlagPurger return the purgers on repo.
func NewCellPurger(repo Repository) *CellPurger { return &CellPurger{repo: repo} }
func NewFlagPurger(repo Repository) *FlagPurger { return &FlagPurger{repo: repo} }

// Purge deletes the cells built before before and returns how many.
func (p *CellPurger) Purge(ctx context.Context, before time.Time) (int64, error) {
	return p.repo.DeleteCellsBefore(ctx, before)
}

// Purge deletes the rise flags built before before and returns how many.
func (p *FlagPurger) Purge(ctx context.Context, before time.Time) (int64, error) {
	return p.repo.DeleteFlagsBefore(ctx, before)
}
