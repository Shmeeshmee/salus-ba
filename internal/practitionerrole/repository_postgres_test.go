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

func TestCreate_InsertsARoleIntoTheClinicSetOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	var fresh string // a practitioner of clinic A without a role
	if err := owner.QueryRow(t.Context(), "INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Cy', 'Example') RETURNING id",
		clinicA).Scan(&fresh); err != nil {
		t.Fatal(err)
	}
	repo := practitionerrole.NewPostgresRepository(app)
	create := func(org, practitioner string) (practitionerrole.PractitionerRole, error) {
		var p practitionerrole.PractitionerRole
		err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			if err := repo.SetOrganization(ctx, clinicA); err != nil {
				return err
			}
			var err error
			p, err = repo.Create(ctx, org, practitioner)
			return err
		})
		return p, err
	}

	got, err := create(clinicA, fresh)
	if err != nil || got.ID == "" || got.OrganizationID != clinicA || got.PractitionerID != fresh || got.VersionID != 1 || !got.Active ||
		got.LastUpdated.IsZero() {
		t.Fatalf("Create() in clinic A = %+v, %v; want the new role at version 1", got, err)
	}
	for name, tc := range map[string][2]string{
		"a second role":                {clinicA, fresh},
		"another clinic":               {clinicB, fresh},
		"a clinic not UUID":            {"not-a-uuid", fresh},
		"a practitioner id not a UUID": {clinicA, "not-a-uuid"},
	} {
		if _, err := create(tc[0], tc[1]); err == nil {
			t.Errorf("Create() of %s: error = nil", name)
		}
	}
}

func TestSetSpecialty_MovesToTheNextVersionOrConflicts(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := practitionerrole.NewPostgresRepository(app)
	at := time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)
	set := func(id string, version int32) (practitionerrole.PractitionerRole, error) {
		var p practitionerrole.PractitionerRole
		err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
			var err error
			p, err = repo.SetSpecialty(ctx, id, "394579002", version, at)
			return err
		})
		return p, err
	}

	got, err := set(roleA1, 1)
	if err != nil || got.ID != roleA1 || got.SpecialtyCode != "394579002" || got.VersionID != 2 || !got.LastUpdated.Equal(at) {
		t.Errorf("SetSpecialty() = %+v, %v; want cardiology at version 2, updated at %v", got, err, at)
	}
	var read practitionerrole.PractitionerRole
	if err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		read, err = repo.ByID(ctx, roleA1)
		return err
	}); err != nil || read.SpecialtyCode != "394579002" {
		t.Errorf("ByID() after SetSpecialty = %+v, %v; want the code", read, err)
	}
	for name, tc := range map[string]struct {
		id      string
		version int32
	}{
		"a stale version":  {roleA1, 1},
		"another clinic":   {roleB, 1},
		"an id not a UUID": {"not-a-uuid", 1},
	} {
		if _, err := set(tc.id, tc.version); !errors.Is(err, practitionerrole.ErrConflict) {
			t.Errorf("SetSpecialty() of %s: %v, want ErrConflict", name, err)
		}
	}
}

func TestPublicByOrganization_ShowsOnlyActiveRolesOfAListedClinicInTheDirectoryScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	practitionerA1 := seed(t, owner)
	ctx := t.Context()
	_, err := owner.Exec(ctx, `UPDATE organization SET listed = true;
		UPDATE organization SET listed = false WHERE id = '`+clinicB+`';
		UPDATE practitioner_role SET active = false WHERE id = '`+roleA2+`';
		UPDATE practitioner_role SET specialty_code = '394579002' WHERE id = '`+roleA1+`'`)
	if err != nil {
		t.Fatal(err)
	}
	repo := practitionerrole.NewPostgresRepository(app)
	public := func(org string, directory bool) ([]practitionerrole.PractitionerRole, error) {
		var roles []practitionerrole.PractitionerRole
		err := postgres.NewTransactor(app).WithTx(ctx, func(ctx context.Context) error {
			if directory {
				if err := repo.SetDirectory(ctx); err != nil {
					return err
				}
			}
			var err error
			roles, err = repo.PublicByOrganization(ctx, org)
			return err
		})
		return roles, err
	}

	got, err := public(clinicA, true)
	want := practitionerrole.PractitionerRole{ID: roleA1, OrganizationID: clinicA, PractitionerID: practitionerA1, SpecialtyCode: "394579002",
		VersionID: 1, Active: true, LastUpdated: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)}
	if err != nil || len(got) != 1 {
		t.Fatalf("PublicByOrganization(listed clinic A) = %+v, %v; want only the active role A1", got, err)
	}
	if got[0].LastUpdated = got[0].LastUpdated.UTC(); got[0] != want {
		t.Errorf("PublicByOrganization(listed clinic A) = %+v,\nwant %+v", got[0], want)
	}
	for name, tc := range map[string]struct {
		org       string
		directory bool
	}{
		"an unlisted clinic": {clinicB, true},
		"no directory scope": {clinicA, false},
		"an id not a UUID":   {"not-a-uuid", true},
	} {
		if got, err := public(tc.org, tc.directory); err != nil || len(got) != 0 {
			t.Errorf("PublicByOrganization() of %s = %+v, %v; want none", name, got, err)
		}
	}
	if err := repo.SetDirectory(ctx); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("SetDirectory() outside a transaction: %v, want postgres.ErrNoTx", err)
	}
}
