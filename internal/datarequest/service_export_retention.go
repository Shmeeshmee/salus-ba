package datarequest

import (
	"context"
	"time"
)

// purgeBatch bounds the exports one round of the purger reads.
const purgeBatch = 500

// ExportPurger is the retention rule of "data_request_export" (internal/retention): it deletes each export ready, or
// failed, before the cut-off, its archive first, so no object outlives its row's key.
type ExportPurger struct {
	repo  ExportRepository
	store Store
}

// NewExportPurger returns the purger on repo (salus_retention) and the store.
func NewExportPurger(repo ExportRepository, store Store) *ExportPurger {
	return &ExportPurger{repo: repo, store: store}
}

// Purge deletes the exports ready or failed before before, and their archives, and counts them.
func (p *ExportPurger) Purge(ctx context.Context, before time.Time) (int64, error) {
	var n int64
	for {
		ids, err := p.repo.ExportsDoneBefore(ctx, before, purgeBatch)
		if err != nil || len(ids) == 0 {
			return n, err
		}
		round := n
		for _, id := range ids {
			if err := p.store.Delete(ctx, ExportObjectKey(id)); err != nil {
				return n, err
			}
			ok, err := p.repo.DeleteExport(ctx, id)
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
