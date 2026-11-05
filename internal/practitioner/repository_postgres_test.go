package practitioner_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seed creates, as the owner, clinics A and B with one practitioner each.
func seed(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	_, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic A'), ($2, 'Synthetic clinic B');
		`, clinicA, clinicB)
	if err != nil {
		t.Fatal(err)
	}
	_, err = owner.Exec(t.Context(), `INSERT INTO practitioner (id, organization_id, given_name, family_name, updated_at)
		VALUES ($1, $2, 'Ada', 'Example', '2026-11-01T08:00:00Z'), ($3, $4, 'Bo', 'Example', '2026-11-01T08:00:00Z')`,
		practitionerA, clinicA, practitionerB, clinicB)
	if err != nil {
		t.Fatal(err)
	}
}

func TestByID_SeesOnlyTheClinicSet(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := practitioner.NewPostgresRepository(app)
	tr := postgres.NewTransactor(app)

	byID := func(org, id string) (practitioner.Practitioner, error) {
		var p practitioner.Practitioner
		err := tr.WithTx(t.Context(), func(ctx context.Context) error {
			if err := repo.SetOrganization(ctx, org); err != nil {
				return err
			}
			var err error
			p, err = repo.ByID(ctx, id)
			return err
		})
		return p, err
	}

	want := practitioner.Practitioner{ID: practitionerA, OrganizationID: clinicA, GivenName: "Ada", FamilyName: "Example",
		VersionID: 1, Active: true, LastUpdated: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)}
	got, err := byID(clinicA, practitionerA)
	got.LastUpdated = got.LastUpdated.UTC()
	if err != nil || got != want {
		t.Errorf("ByID(A) in clinic A = %+v, %v;\nwant %+v", got, err, want)
	}
	for _, id := range []string{practitionerB, "not-a-uuid"} {
		if _, err := byID(clinicA, id); !errors.Is(err, practitioner.ErrNotFound) {
			t.Errorf("ByID(%q) in clinic A: %v, want ErrNotFound", id, err)
		}
	}
	if _, err := repo.ByID(t.Context(), practitionerA); !errors.Is(err, practitioner.ErrNotFound) {
		t.Errorf("ByID() with no clinic set: %v, want ErrNotFound", err)
	}
	if err := repo.SetOrganization(t.Context(), clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("SetOrganization() outside a transaction: %v, want postgres.ErrNoTx", err)
	}
}

func TestCreate_InsertsIntoTheClinicSetOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := practitioner.NewPostgresRepository(app)
	tr := postgres.NewTransactor(app)
	create := func(set, org string) (practitioner.Practitioner, error) {
		var p practitioner.Practitioner
		err := tr.WithTx(t.Context(), func(ctx context.Context) error {
			if err := repo.SetOrganization(ctx, set); err != nil {
				return err
			}
			var err error
			p, err = repo.Create(ctx, org, "Cy", "Example")
			return err
		})
		return p, err
	}

	got, err := create(clinicA, clinicA)
	if err != nil || got.ID == "" || got.OrganizationID != clinicA || got.GivenName != "Cy" || got.FamilyName != "Example" ||
		got.VersionID != 1 || !got.Active || got.LastUpdated.IsZero() {
		t.Fatalf("Create() in clinic A = %+v, %v; want the new practitioner at version 1", got, err)
	}
	if _, err := create(clinicA, clinicB); err == nil {
		t.Error("Create() for clinic B with clinic A set: error = nil, want the row-level security refusal")
	}
	if _, err := create(clinicA, "not-a-uuid"); err == nil {
		t.Error("Create() for a clinic id that is not a UUID: error = nil")
	}
}

func TestPublicByIDs_GivesOnlyTheNamesOfActivePractitionersOfListedClinicsInTheDirectoryScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	ctx := t.Context()
	const inactiveA = "a3000000-0000-4000-8000-0000000000ff"
	_, err := owner.Exec(ctx, `UPDATE organization SET listed = true WHERE id = '`+clinicA+`';
		INSERT INTO practitioner (id, organization_id, given_name, family_name, active)
		VALUES ('`+inactiveA+`', '`+clinicA+`', 'Cy', 'Example', false)`)
	if err != nil {
		t.Fatal(err)
	}
	repo := practitioner.NewPostgresRepository(app)
	public := func(directory bool, ids ...string) ([]practitioner.Practitioner, error) {
		var out []practitioner.Practitioner
		err := postgres.NewTransactor(app).WithTx(ctx, func(ctx context.Context) error {
			if directory {
				if err := repo.SetDirectory(ctx); err != nil {
					return err
				}
			}
			var err error
			out, err = repo.PublicByIDs(ctx, ids)
			return err
		})
		return out, err
	}

	got, err := public(true, practitionerA, practitionerB, inactiveA, "not-a-uuid")
	// No clinic: nothing beyond the name, the meta and whether the record is active.
	want := practitioner.Practitioner{ID: practitionerA, GivenName: "Ada", FamilyName: "Example", VersionID: 1, Active: true,
		LastUpdated: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)}
	if err != nil || len(got) != 1 {
		t.Fatalf("PublicByIDs(A, B of an unlisted clinic, inactive, not a UUID) = %+v, %v; want only A", got, err)
	}
	if got[0].LastUpdated = got[0].LastUpdated.UTC(); got[0] != want {
		t.Errorf("PublicByIDs() = %+v,\nwant %+v", got[0], want)
	}
	if got, err := public(false, practitionerA); err != nil || len(got) != 0 {
		t.Errorf("PublicByIDs() without the directory scope = %+v, %v; want none", got, err)
	}
	if err := repo.SetDirectory(ctx); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("SetDirectory() outside a transaction: %v, want postgres.ErrNoTx", err)
	}
}
