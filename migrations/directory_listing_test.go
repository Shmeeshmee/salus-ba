package migrations_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// inDirectory runs fn in a transaction on the app pool in the public directory scope.
func inDirectory(t *testing.T, app *pgxpool.Pool, fn func(ctx context.Context, db postgres.Querier) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.directory", "public"); err != nil {
			return err
		}
		return fn(ctx, postgres.Conn(ctx, app))
	})
}

func TestDirectoryListing_ANewClinicIsUnlistedInSarajevoAndARoleHasNoSpecialty(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	seeded := seedDirectory(t, owner)

	var listed bool
	var zone string
	var specialty *string
	err := owner.QueryRow(t.Context(), `SELECT o.listed, o.time_zone, r.specialty_code
		FROM organization o JOIN practitioner_role r ON r.organization_id = o.id WHERE r.id = $1`,
		seeded[clinicA].role).Scan(&listed, &zone, &specialty)
	if err != nil || listed || zone != "Europe/Sarajevo" || specialty != nil {
		t.Errorf("a new clinic and role: listed %v, zone %q, specialty %v, %v; want false, Europe/Sarajevo, NULL",
			listed, zone, specialty, err)
	}
	if _, err := owner.Exec(t.Context(), "UPDATE organization SET time_zone = ' ' WHERE id = $1", clinicA); sqlState(err) != "23514" {
		t.Errorf("a blank time zone: %v, want check_violation", err)
	}
}

func TestDirectoryListing_TheDirectoryScopeSeesOnlyTheActiveRowsOfListedClinics(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seeded := seedDirectory(t, owner)
	ctx := t.Context()
	// Clinic A is listed and has a second, inactive practitioner with an inactive role; clinic B is not listed.
	_, err := owner.Exec(ctx, `UPDATE organization SET listed = true WHERE id = $1;`, clinicA)
	if err != nil {
		t.Fatal(err)
	}
	_, err = owner.Exec(ctx, `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name, active)
			VALUES ($1, 'Bo', 'Example', false) RETURNING id, organization_id)
		INSERT INTO practitioner_role (organization_id, practitioner_id, active) SELECT organization_id, id, false FROM p`, clinicA)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"organization":      clinicA,
		"practitioner":      seeded[clinicA].practitioner,
		"practitioner_role": seeded[clinicA].role,
	}
	for table, id := range want {
		var ids []string
		err := inDirectory(t, app, func(ctx context.Context, db postgres.Querier) error {
			return db.QueryRow(ctx, "SELECT coalesce(array_agg(id::text), '{}') FROM "+table).Scan(&ids)
		})
		if err != nil || len(ids) != 1 || ids[0] != id {
			t.Errorf("%s in the directory scope: %v, %v; want only %s", table, ids, err, id)
		}
	}

	// Unset or another value matches nothing.
	for _, value := range []string{"", "private"} {
		for table := range want {
			var n int
			err := postgres.NewTransactor(app).WithTx(ctx, func(ctx context.Context) error {
				if err := postgres.SetLocal(ctx, "app.directory", value); err != nil {
					return err
				}
				return postgres.Conn(ctx, app).QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n)
			})
			if err != nil || n != 0 {
				t.Errorf("%s with app.directory %q: %d rows, %v; want 0", table, value, n, err)
			}
		}
	}
}

func TestDirectoryListing_TheDirectoryScopeWritesNothing(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)
	if _, err := owner.Exec(t.Context(), "UPDATE organization SET listed = true"); err != nil {
		t.Fatal(err)
	}

	for _, stmt := range []string{
		"UPDATE organization SET name = 'Renamed', listed = false",
		"UPDATE practitioner_role SET specialty_code = '394579002'",
	} {
		var n int64
		err := inDirectory(t, app, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, stmt)
			n = tag.RowsAffected()
			return err
		})
		if err != nil || n != 0 {
			t.Errorf("%s in the directory scope: %d rows, %v; want 0 rows", stmt, n, err)
		}
	}
}

func TestDirectoryListing_AnAdminSetsListingZoneAndSpecialtyAndNothingElse(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)

	allowed := []string{
		"UPDATE organization SET listed = true, time_zone = 'Europe/Vienna', version_id = version_id + 1, updated_at = now()",
		"UPDATE practitioner_role SET specialty_code = '394579002', version_id = version_id + 1, updated_at = now()",
	}
	for _, stmt := range allowed {
		var n int64
		err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, stmt)
			n = tag.RowsAffected()
			return err
		})
		if err != nil || n != 1 {
			t.Errorf("%s as %s in clinic A: %d rows, %v; want 1", stmt, pgtest.AppRole, n, err)
		}
	}
	for _, stmt := range []string{
		"UPDATE practitioner_role SET organization_id = organization_id",
		"UPDATE practitioner_role SET practitioner_id = practitioner_id",
		"UPDATE practitioner SET active = false",
	} {
		err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege (42501)", stmt, pgtest.AppRole, err)
		}
	}
}
