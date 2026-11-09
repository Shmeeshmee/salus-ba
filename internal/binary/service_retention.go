package binary

import (
	"context"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// Purger is the retention job's applier for "binary" (internal/retention: delete, 24 h), built on the retention
// role's pool and the object store in internal/worker. Each row otherwise lives exactly as long as its
// questionnaire response and goes with it or with its answer; the purger removes what is left over.
type Purger struct {
	repo  Repository
	store Store
}

// NewPurger returns the purger of "binary" on repo (salus_retention) and the store.
func NewPurger(repo Repository, store Store) *Purger {
	return &Purger{repo: repo, store: store}
}

// Purge deletes the rejected rows scanned before before, then every object written before before whose key
// is a binary id (a UUID; other keys, such as composition's PDFs under composition/, are another owner's) and
// that has no row (a deleted or rejected file's ciphertext, an upload that never reached its answer), and
// counts both. An upload stores its object before its row, so a younger object without a row may be an upload
// in progress and is kept.
func (p *Purger) Purge(ctx context.Context, before time.Time) (int64, error) {
	n, err := p.repo.DeleteRejectedBefore(ctx, before)
	if err != nil {
		return 0, err
	}
	err = p.store.ListOlderThan(ctx, before, func(keys []string) error {
		existing, err := p.repo.Existing(ctx, keys)
		if err != nil {
			return err
		}
		rows := make(map[string]bool, len(existing))
		for _, id := range existing {
			rows[id] = true
		}
		for _, key := range keys {
			if rows[key] || !fhir.IsUUID(key) { // a key that is no binary id is another owner's object
				continue
			}
			if err := p.store.Delete(ctx, key); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	return n, err
}
