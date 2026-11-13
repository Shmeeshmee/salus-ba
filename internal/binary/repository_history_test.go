package binary_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// SAL-39: Clean gives the clean files among the ids the scope shows: not a quarantined one, nor another patient's.
func TestPostgres_CleanGivesTheCleanFilesTheScopeShows(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	at := time.Date(2030, 1, 6, 9, 0, 0, 0, time.UTC)
	clean, quarantined := s.file("files", 100, 1), s.file("files", 200, 2)
	err := in(t, app, asPatient(s.patient), func(ctx context.Context, r binary.Repository) error {
		for _, b := range []binary.Binary{clean, quarantined} {
			if _, err := r.Insert(ctx, b, at); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE "binary" SET status = 'clean', scanned_at = now() WHERE id = $1`, clean.ID); err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		patient string
		want    []string
	}{"the patient": {s.patient, []string{clean.ID}}, "another patient": {s.other, nil}} {
		err := in(t, app, asPatient(tc.patient), func(ctx context.Context, r binary.Repository) error {
			got, err := r.Clean(ctx, []string{clean.ID, quarantined.ID, "not-a-uuid"})
			var gotIDs []string
			for _, b := range got {
				gotIDs = append(gotIDs, b.ID)
			}
			if !slices.Equal(gotIDs, tc.want) {
				t.Errorf("%s: Clean() = %v, want %v", name, gotIDs, tc.want)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	err = in(t, app, asPatient(s.patient), func(ctx context.Context, r binary.Repository) error {
		got, err := r.Clean(ctx, []string{"x"})
		if len(got) != 0 {
			t.Errorf("Clean(no UUID) = %v, want none", got)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
