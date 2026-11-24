package clinicexport

import (
	"context"
	"time"
)

// purgeBatch bounds the exports one round of the purger reads.
const purgeBatch = 500

// Purger is the retention rule of "clinic_export" (internal/retention): it deletes each export asked for before the
// cut-off, its archive first, so no object outlives its row's key.
type Purger struct {
	repo  Repository
	store Store
}

// NewPurger returns the purger on repo (salus_retention) and the store.
func NewPurger(repo Repository, store Store) *Purger {
	return &Purger{repo: repo, store: store}
}

// Purge deletes the exports asked for before before, and their archives, and counts them.
func (p *Purger) Purge(ctx context.Context, before time.Time) (int64, error) {
	var n int64
	for {
		ids, err := p.repo.RequestedBefore(ctx, before, purgeBatch)
		if err != nil || len(ids) == 0 {
			return n, err
		}
		round := n
		for _, id := range ids {
			if err := p.store.Delete(ctx, ObjectKey(id)); err != nil {
				return n, err
			}
			ok, err := p.repo.Delete(ctx, id)
			if err != nil {
				return n, err
			}
			if ok {
				n++
			}
		}
		if n == round { // nothing deleted: the rest is not ours to delete
			return n, nil
		}
	}
}
