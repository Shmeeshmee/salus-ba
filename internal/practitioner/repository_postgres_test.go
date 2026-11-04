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
