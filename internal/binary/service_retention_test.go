package binary_test

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// prefixStore is the real store seen through a key prefix: the purger lists the whole bucket, which every test
// running against the stack shares, so a test's purger sees, and deletes, only the objects under its own prefix.
type prefixStore struct {
	binary.Store
	prefix string
}

func (s prefixStore) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	return s.Store.Put(ctx, s.prefix+key, r, size)
}

func (s prefixStore) GetRange(ctx context.Context, key string, from, to int64) (io.ReadCloser, error) {
	return s.Store.GetRange(ctx, s.prefix+key, from, to)
}

func (s prefixStore) Delete(ctx context.Context, key string) error {
	return s.Store.Delete(ctx, s.prefix+key)
}

func (s prefixStore) ListOlderThan(ctx context.Context, before time.Time, fn func(keys []string) error) error {
	return s.Store.ListOlderThan(ctx, before, func(keys []string) error {
		var own []string
		for _, k := range keys {
			if rest, ok := strings.CutPrefix(k, s.prefix); ok {
				own = append(own, rest)
			}
		}
		if len(own) == 0 {
			return nil
		}
		return fn(own)
	})
}

// waitPast waits until the wall clock has passed t, so the next object stored is written after it, whatever
// the store's precision.
func waitPast(t *testing.T, at time.Time) {
	t.Helper()
	wall := clock.System{}
	for !wall.Now().After(at) {
		time.Sleep(20 * time.Millisecond)
	}
}

// The purger (docs/attachments/SPEC.md §5 "Retention"), as salus_retention on the real database and store: the
// rejected rows scanned before the cut-off, then every object written before it that has no row; it counts both.
func TestPurger_DeletesOldRejectedRowsThenOldObjectsWithoutARow(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	s := seedDB(t, owner)
	store := prefixStore{Store: newStore(t, 2), prefix: prefix(t)} // pages of 2: the purger walks every page
	putObject := func(key string) {
		put(t, store.Store, store.prefix+key, []byte("synthetic ciphertext"))
	}

	oldRejected, freshRejected, withRow := s.file("files", 10, 1), s.file("files", 11, 2), s.file("files", 12, 3)
	// foreign is another owner's object (a signed document's PDF, internal/composition): no binary id, kept.
	orphan, young, foreign := binaryID(), binaryID(), "composition/"+binaryID()
	for _, key := range []string{oldRejected.ID, freshRejected.ID, withRow.ID, orphan, foreign} {
		putObject(key)
	}
	written := clock.System{}.Now()
	waitPast(t, written.Add(1100*time.Millisecond))
	cut := clock.System{}.Now()
	waitPast(t, cut.Add(1100*time.Millisecond))
	putObject(young)

	oldRejected.Status, oldRejected.ScannedAt = binary.StatusRejected, cut.Add(-time.Second)
	freshRejected.Status, freshRejected.ScannedAt = binary.StatusRejected, cut.Add(time.Second)
	withRow.Status, withRow.ScannedAt = binary.StatusClean, cut.Add(-time.Hour)
	for _, b := range []binary.Binary{oldRejected, freshRejected, withRow} {
		insertAsOwner(t, owner, b)
	}

	n, err := binary.NewPurger(binary.NewPostgresRepository(ret), store).Purge(t.Context(), cut)

	if err != nil || n != 3 {
		t.Fatalf("Purge() = %d, %v; want 3 (the old rejected row, its object and the orphan)", n, err)
	}
	want := []string{freshRejected.ID, withRow.ID}
	slices.Sort(want)
	var got []string
	if err := owner.QueryRow(t.Context(), `SELECT coalesce(array_agg(id::text ORDER BY id), '{}') FROM "binary"`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("rows left %v, want the fresh rejected one and the clean one", got)
	}
	for key, kept := range map[string]bool{oldRejected.ID: false, orphan: false, freshRejected.ID: true, withRow.ID: true, young: true, foreign: true} {
		_, err := store.GetRange(t.Context(), key, 0, 0)
		switch {
		case kept && err != nil:
			t.Errorf("object %s deleted: %v", key, err)
		case !kept && !errors.Is(err, binary.ErrObjectNotFound):
			t.Errorf("object %s kept: %v", key, err)
		}
	}
}

func TestPurger_StopsAtTheDatabasesError(t *testing.T) {
	t.Parallel()
	_, ret := pgtest.NewRetentionDatabase(t)
	ret.Close()
	if _, err := binary.NewPurger(binary.NewPostgresRepository(ret), prefixStore{Store: newStore(t, 2), prefix: prefix(t)}).Purge(t.Context(), now); err == nil {
		t.Fatal("Purge() on a closed pool = nil, want an error")
	}
}
