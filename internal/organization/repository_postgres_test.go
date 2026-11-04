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
	want := organization.Organization{ID: clinicA, Name: "Synthetic clinic A", TimeZone: "Europe/Sarajevo", VersionID: 1, Active: true,
		LastUpdated: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)}
	if got.LastUpdated = got.LastUpdated.UTC(); err != nil || got != want {
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

func TestUpdate_MovesToTheNextVersionOrConflicts(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	repo := organization.NewPostgresRepository(app)
	at := time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

	var got organization.Organization
	err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		got, err = repo.Update(ctx, organization.Organization{ID: clinicA, Name: "Renamed clinic", TimeZone: "Europe/Vienna", Listed: true, VersionID: 1}, at)
		return err
	})
	want := organization.Organization{ID: clinicA, Name: "Renamed clinic", TimeZone: "Europe/Vienna", Listed: true, Active: true, VersionID: 2, LastUpdated: at}
	if got.LastUpdated = got.LastUpdated.UTC(); err != nil || got != want {
		t.Errorf("Update() = %+v, %v;\nwant %+v", got, err, want)
	}
	var read organization.Organization
	err = inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		read, err = repo.ByID(ctx, clinicA)
		return err
	})
	if read.LastUpdated = read.LastUpdated.UTC(); err != nil || read != want {
		t.Errorf("ByID() after Update = %+v, %v;\nwant %+v", read, err, want)
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
			_, err := repo.Update(ctx, organization.Organization{ID: tc.id, Name: "Again", TimeZone: "UTC", VersionID: tc.version}, at)
			return err
		})
		if !errors.Is(err, organization.ErrConflict) {
			t.Errorf("Update() of %s: %v, want ErrConflict", name, err)
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

// Clinics are created by the operator command, which connects as the owner; salus_app may not (migrations).
func TestCreate_StoresAnActiveClinicAtVersionOne(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)

	got, err := organization.NewPostgresRepository(owner).Create(t.Context(), "Synthetic clinic C")

	if err != nil || got.ID == "" || got.Name != "Synthetic clinic C" || got.VersionID != 1 || !got.Active || got.LastUpdated.IsZero() {
		t.Errorf("Create() = %+v, %v; want the new clinic at version 1", got, err)
	}
}

func TestListed_ShowsOnlyListedClinicsInTheDirectoryScopeByName(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	const clinicC = "c0000000-0000-4000-8000-00000000000c"
	_, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name, listed, time_zone, updated_at)
		VALUES ($1, 'Another synthetic clinic', true, 'Europe/Vienna', '2026-11-01T08:00:00Z')`, clinicC)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), "UPDATE organization SET listed = true WHERE id = $1", clinicB); err != nil {
		t.Fatal(err)
	}
	repo := organization.NewPostgresRepository(app)
	listed := func(directory bool) ([]organization.Organization, error) {
		var out []organization.Organization
		err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			if directory {
				if err := repo.SetDirectory(ctx); err != nil {
					return err
				}
			}
			var err error
			out, err = repo.Listed(ctx)
			return err
		})
		return out, err
	}

	got, err := listed(true)
	at := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	want := []organization.Organization{
		{ID: clinicC, Name: "Another synthetic clinic", TimeZone: "Europe/Vienna", VersionID: 1, Active: true, Listed: true, LastUpdated: at},
		{ID: clinicB, Name: "Synthetic clinic B", TimeZone: "Europe/Sarajevo", VersionID: 1, Active: true, Listed: true, LastUpdated: at},
	}
	for i := range got {
		got[i].LastUpdated = got[i].LastUpdated.UTC()
	}
	if err != nil || len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Listed() = %+v, %v;\nwant %+v", got, err, want)
	}
	if got, err := listed(false); err != nil || len(got) != 0 {
		t.Errorf("Listed() without the directory scope = %+v, %v; want none", got, err)
	}
	if err := repo.SetDirectory(t.Context()); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("SetDirectory() outside a transaction: %v, want postgres.ErrNoTx", err)
	}
}
