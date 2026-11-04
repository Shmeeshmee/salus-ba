package organization_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seedClinics creates clinics A and B as the owner.
func seedClinics(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	_, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name, updated_at) VALUES
		($1, 'Synthetic clinic A', '2026-11-01T08:00:00Z'), ($2, 'Synthetic clinic B', '2026-11-01T08:00:00Z')`, clinicA, clinicB)
	if err != nil {
		t.Fatal(err)
	}
}

// inClinic runs fn in a transaction on app scoped to org.
func inClinic(t *testing.T, app *pgxpool.Pool, repo organization.Repository, org string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		return fn(ctx)
	})
}

func TestByID_SeesOnlyTheClinicSet(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	repo := organization.NewPostgresRepository(app)

	var got organization.Organization
	err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		got, err = repo.ByID(ctx, clinicA)
		return err
	})
	want := organization.Organization{ID: clinicA, Name: "Synthetic clinic A", VersionID: 1, Active: true,
		LastUpdated: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)}
	if err != nil || got.ID != want.ID || got.Name != want.Name || got.VersionID != 1 || !got.Active || !got.LastUpdated.Equal(want.LastUpdated) {
		t.Errorf("ByID(clinic A) in clinic A = %+v, %v;\nwant %+v", got, err, want)
	}
	for _, id := range []string{clinicB, "not-a-uuid"} {
		err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
			_, err := repo.ByID(ctx, id)
			return err
		})
		if !errors.Is(err, organization.ErrNotFound) {
			t.Errorf("ByID(%q) in clinic A: %v, want ErrNotFound", id, err)
		}
	}
	if _, err := repo.ByID(t.Context(), clinicA); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("ByID() with no clinic set: %v, want ErrNotFound", err)
	}
}

func TestRename_MovesToTheNextVersionOrConflicts(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	repo := organization.NewPostgresRepository(app)
	at := time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

	var got organization.Organization
	err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		got, err = repo.Rename(ctx, clinicA, "Renamed clinic", 1, at)
		return err
	})
	if err != nil || got.Name != "Renamed clinic" || got.VersionID != 2 || !got.LastUpdated.Equal(at) {
		t.Errorf("Rename() = %+v, %v; want the new name at version 2, updated at %v", got, err, at)
	}
	for name, tc := range map[string]struct {
		id      string
		version int32
	}{
		"a stale version":  {id: clinicA, version: 1},
		"another clinic":   {id: clinicB, version: 1},
		"an id not a UUID": {id: "not-a-uuid", version: 1},
	} {
		err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
			_, err := repo.Rename(ctx, tc.id, "Again", tc.version, at)
			return err
		})
		if !errors.Is(err, organization.ErrConflict) {
			t.Errorf("Rename() of %s: %v, want ErrConflict", name, err)
		}
	}
	var name string
	if err := owner.QueryRow(t.Context(), "SELECT name FROM organization WHERE id = $1", clinicB).Scan(&name); err != nil || name != "Synthetic clinic B" {
		t.Errorf("clinic B is named %q (%v), want it unchanged", name, err)
	}
}

func TestSetOrganization_NeedsATransaction(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	if err := organization.NewPostgresRepository(app).SetOrganization(t.Context(), clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("SetOrganization() outside a transaction: %v, want postgres.ErrNoTx", err)
	}
}
