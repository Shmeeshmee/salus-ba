package practitionerrole_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seed creates, as the owner, clinics A and B, two practitioners with roles in A (A1 created first) and one
// in B, and returns the practitioner of role A1.
func seed(t *testing.T, owner *pgxpool.Pool) string {
	t.Helper()
	ctx := t.Context()
	if _, err := owner.Exec(ctx, "INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic A'), ($2, 'Synthetic clinic B')",
		clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	var first string
	for i, r := range []struct{ id, org, created string }{
		{roleA2, clinicA, "2026-11-02T08:00:00Z"},
		{roleA1, clinicA, "2026-11-01T08:00:00Z"},
		{roleB, clinicB, "2026-11-01T08:00:00Z"},
	} {
		var practitioner string
		err := owner.QueryRow(ctx, `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
			VALUES ($2, 'Ada', 'Example') RETURNING id, organization_id)
			INSERT INTO practitioner_role (id, organization_id, practitioner_id, created_at, updated_at)
			SELECT $1, organization_id, id, $3, $3 FROM p RETURNING practitioner_id`, r.id, r.org, r.created).Scan(&practitioner)
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 {
			first = practitioner
		}
	}
	return first
}

// inClinic runs fn in a transaction on app scoped to org.
func inClinic(t *testing.T, app *pgxpool.Pool, repo practitionerrole.Repository, org string, fn func(ctx context.Context) error) error {
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
	practitionerA1 := seed(t, owner)
	repo := practitionerrole.NewPostgresRepository(app)

	var got practitionerrole.PractitionerRole
	err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		got, err = repo.ByID(ctx, roleA1)
		return err
	})
	got.LastUpdated = got.LastUpdated.UTC()
	want := practitionerrole.PractitionerRole{ID: roleA1, OrganizationID: clinicA, PractitionerID: practitionerA1, VersionID: 1,
		Active: true, LastUpdated: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)}
	if err != nil || got != want {
		t.Errorf("ByID(A1) in clinic A = %+v, %v;\nwant %+v", got, err, want)
	}
	for _, id := range []string{roleB, "not-a-uuid"} {
		err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
			_, err := repo.ByID(ctx, id)
			return err
		})
		if !errors.Is(err, practitionerrole.ErrNotFound) {
			t.Errorf("ByID(%q) in clinic A: %v, want ErrNotFound", id, err)
		}
	}
	if _, err := repo.ByID(t.Context(), roleA1); !errors.Is(err, practitionerrole.ErrNotFound) {
		t.Errorf("ByID() with no clinic set: %v, want ErrNotFound", err)
	}
	if err := repo.SetOrganization(t.Context(), clinicA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("SetOrganization() outside a transaction: %v, want postgres.ErrNoTx", err)
	}
}

func TestByOrganization_ListsTheClinicsRolesOldestFirst(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := practitionerrole.NewPostgresRepository(app)

	search := func(scope, org string) []string {
		t.Helper()
		var ids []string
		err := inClinic(t, app, repo, scope, func(ctx context.Context) error {
			roles, err := repo.ByOrganization(ctx, org)
			for _, r := range roles {
				ids = append(ids, r.ID)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return ids
	}
	if got := search(clinicA, clinicA); len(got) != 2 || got[0] != roleA1 || got[1] != roleA2 {
		t.Errorf("clinic A's roles in clinic A = %v, want [A1 A2]", got)
	}
	for name, org := range map[string]string{"clinic B's roles": clinicB, "an id not a UUID": "not-a-uuid"} {
		if got := search(clinicA, org); len(got) != 0 {
			t.Errorf("%s in clinic A = %v, want none", name, got)
		}
	}
}
