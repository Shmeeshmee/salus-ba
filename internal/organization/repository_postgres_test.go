package organization_test

import (
	"context"
	"errors"
	"fmt"
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
	_, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name, updated_at, contributes_since) VALUES
		($1, 'Synthetic clinic A', '2026-11-01T08:00:00Z', '2026-11-01T08:00:00Z'),
		($2, 'Synthetic clinic B', '2026-11-01T08:00:00Z', '2026-11-01T08:00:00Z')`, clinicA, clinicB)
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
	seeded := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	want := organization.Organization{ID: clinicA, Name: "Synthetic clinic A", TimeZone: "Europe/Sarajevo", DefaultLanguage: "bs", VersionID: 1, Active: true,
		LastUpdated: seeded, AnalyticsContributes: true, ContributesSince: seeded}
	if got.LastUpdated, got.ContributesSince = got.LastUpdated.UTC(), got.ContributesSince.UTC(); err != nil || got != want {
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
		got, err = repo.Update(ctx, organization.Organization{ID: clinicA, Name: "Renamed clinic", TimeZone: "Europe/Vienna", DefaultLanguage: "sr-Cyrl", Listed: true, PatientTranscriptCopy: true, VersionID: 1,
			Region: "BA-09", City: "Q1270190", LocationSince: at, ContributesSince: at, AddressLine: "Synthetic street 1", PostalCode: "71000"}, at)
		return err
	})
	want := organization.Organization{ID: clinicA, Name: "Renamed clinic", TimeZone: "Europe/Vienna", DefaultLanguage: "sr-Cyrl", Listed: true, PatientTranscriptCopy: true, Active: true, VersionID: 2, LastUpdated: at,
		Region: "BA-09", City: "Q1270190", LocationSince: at, ContributesSince: at, AddressLine: "Synthetic street 1", PostalCode: "71000"}
	utc := func(o *organization.Organization) {
		o.LastUpdated, o.LocationSince, o.ContributesSince = o.LastUpdated.UTC(), o.LocationSince.UTC(), o.ContributesSince.UTC()
	}
	if utc(&got); err != nil || got != want {
		t.Errorf("Update() = %+v, %v;\nwant %+v", got, err, want)
	}
	var read organization.Organization
	err = inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		read, err = repo.ByID(ctx, clinicA)
		return err
	})
	if utc(&read); err != nil || read != want {
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
			_, err := repo.Update(ctx, organization.Organization{ID: tc.id, Name: "Again", TimeZone: "UTC", DefaultLanguage: "en", VersionID: tc.version, ContributesSince: at}, at)
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

// The operator command creates a clinic as the owner. Create sets the new clinic's scope, so it needs a transaction
// (00066: a verified signup creates its clinic as salus_app, TestCreate_AsTheAppInTheNewClinicsScopeUnlisted).
func TestCreate_StoresAnActiveClinicAtVersionOne(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)

	var got organization.Organization
	err := postgres.NewTransactor(owner).WithTx(t.Context(), func(ctx context.Context) error {
		var err error
		got, err = organization.NewPostgresRepository(owner).Create(ctx, "Synthetic clinic C")
		return err
	})

	if err != nil || got.ID == "" || got.Name != "Synthetic clinic C" || got.VersionID != 1 || !got.Active || got.DefaultLanguage != "bs" || got.LastUpdated.IsZero() {
		t.Errorf("Create() = %+v, %v; want the new clinic at version 1, in Bosnian", got, err)
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
		{ID: clinicC, Name: "Another synthetic clinic", TimeZone: "Europe/Vienna", DefaultLanguage: "bs", VersionID: 1, Active: true, Listed: true, LastUpdated: at},
		{ID: clinicB, Name: "Synthetic clinic B", TimeZone: "Europe/Sarajevo", DefaultLanguage: "bs", VersionID: 1, Active: true, Listed: true, LastUpdated: at},
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

// A location that is not a city of its leaf region is refused by the database too.
func TestUpdate_RefusesACityOutsideItsRegion(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	repo := organization.NewPostgresRepository(app)
	at := time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)
	err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		_, err := repo.Update(ctx, organization.Organization{ID: clinicA, Name: "A", TimeZone: "UTC", DefaultLanguage: "en", VersionID: 1,
			Region: "BA-01", City: "Q1270190", LocationSince: at, ContributesSince: at}, at)
		return stillOpen(ctx, repo, err)
	})
	if !errors.Is(err, organization.ErrInvalidLocation) {
		t.Errorf("Update() = %v, want ErrInvalidLocation", err)
	}
}

// A signed-up clinic's admin cannot list it before the operator approves the signup (00066's trigger): the repository
// gives ErrNotApproved, which the handler answers as a client error, not a 500.
func TestUpdate_GivesErrNotApprovedForASignedUpClinicWaitingForTheOperator(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	for _, q := range []string{`INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic A')`,
		`INSERT INTO clinic_signup (email, clinic_name, language, token_hash, verified_at, organization_id)
		VALUES ('admin@example.test', 'Synthetic clinic A', 'en', sha256('a'), now(), $1)`} {
		if _, err := owner.Exec(t.Context(), q, clinicA); err != nil {
			t.Fatal(err)
		}
	}
	repo := organization.NewPostgresRepository(app)
	err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		_, err := repo.Update(ctx, organization.Organization{ID: clinicA, Name: "Synthetic clinic A", Listed: true, TimeZone: "UTC", DefaultLanguage: "en", VersionID: 1}, now)
		return stillOpen(ctx, repo, err)
	})
	if !errors.Is(err, organization.ErrNotApproved) {
		t.Errorf("Update(listed) before approval = %v, want ErrNotApproved", err)
	}
}

// stillOpen returns err once the transaction in ctx has shown it still works after the refused update (BG4): the
// refusal rolls back to a savepoint, so the service can audit it in the same transaction.
func stillOpen(ctx context.Context, repo organization.Repository, err error) error {
	if _, readErr := repo.ByID(ctx, clinicA); readErr != nil {
		return fmt.Errorf("the transaction after the refusal %v: %w", err, readErr)
	}
	return err
}
