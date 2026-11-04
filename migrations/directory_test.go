package migrations_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The directory's clinic-scoped tables and the column that names a row's clinic (organization's own id).
var directoryTables = map[string]string{
	"organization":      "id",
	"practitioner":      "organization_id",
	"practitioner_role": "organization_id",
}

// directory is one clinic's synthetic rows.
type directory struct {
	practitioner, role string
}

// seedDirectory creates, as the owner, clinics A and B with one practitioner and role each.
func seedDirectory(t *testing.T, owner *pgxpool.Pool) map[string]directory {
	t.Helper()
	ctx := t.Context()
	seeded := map[string]directory{}
	for _, org := range []string{clinicA, clinicB} {
		var d directory
		err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic') RETURNING id),
			p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
			      SELECT id, 'Ada', 'Example' FROM o RETURNING id, organization_id)
			INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p
			RETURNING practitioner_id, id`, org).Scan(&d.practitioner, &d.role)
		if err != nil {
			t.Fatal(err)
		}
		seeded[org] = d
	}
	return seeded
}

// inClinic runs fn in a transaction on the app pool with the clinic set; "" sets none.
func inClinic(t *testing.T, app *pgxpool.Pool, org string, fn func(ctx context.Context, db postgres.Querier) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if org != "" {
			if err := postgres.SetLocal(ctx, "app.organization_id", org); err != nil {
				return err
			}
		}
		return fn(ctx, postgres.Conn(ctx, app))
	})
}

func TestDirectory_AppRoleSeesOnlyTheClinicSet(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)

	for table, col := range directoryTables {
		for _, org := range []string{"", clinicA} {
			var n, other int
			err := inClinic(t, app, org, func(ctx context.Context, db postgres.Querier) error {
				return db.QueryRow(ctx, "SELECT count(*), count(*) FILTER (WHERE "+col+" <> $1) FROM "+table, clinicA).
					Scan(&n, &other)
			})
			if err != nil {
				t.Fatalf("%s, clinic %q: %v", table, org, err)
			}
			want := 1
			if org == "" {
				want = 0
			}
			if n != want || other != 0 {
				t.Errorf("%s, clinic %q set: %d rows visible, %d of another clinic; want %d and 0", table, org, n, other, want)
			}
		}
		// After the transaction the pooled connection reads the setting as '', which must still match nothing.
		var n int
		if err := app.QueryRow(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s after a transaction with a clinic: %d rows, %v; want 0", table, n, err)
		}
	}
}

func TestDirectory_AppRoleCannotWriteAnotherClinicsRows(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seeded := seedDirectory(t, owner)

	inserts := map[string]func(ctx context.Context, db postgres.Querier) error{
		"practitioner": func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, "INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Bo', 'Example')", clinicB)
			return err
		},
		"practitioner_role": func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, "INSERT INTO practitioner_role (organization_id, practitioner_id) VALUES ($1, $2)", clinicB, seeded[clinicB].practitioner)
			return err
		},
	}
	for table, insert := range inserts {
		if err := inClinic(t, app, clinicA, insert); sqlState(err) != "42501" {
			t.Errorf("insert into %s for clinic B with clinic A set: %v, want an RLS violation (42501)", table, err)
		}
	}
	var n int64
	err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
		tag, err := db.Exec(ctx, "UPDATE organization SET name = 'Renamed', version_id = version_id + 1 WHERE id = $1", clinicB)
		n = tag.RowsAffected()
		return err
	})
	if err != nil || n != 0 {
		t.Errorf("update of clinic B with clinic A set: %d rows, %v; want 0 rows", n, err)
	}
}

func TestDirectory_AppRoleCannotTurnRowLevelSecurityOff(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)

	for table := range directoryTables {
		err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			if _, err := db.Exec(ctx, "SET LOCAL row_security = off"); err != nil {
				return err
			}
			_, err := db.Exec(ctx, "SELECT * FROM "+table)
			return err
		})
		if sqlState(err) != "42501" || !strings.Contains(err.Error(), "row-level security") {
			t.Errorf("%s with row_security off: %v, want the RLS refusal (42501)", table, err)
		}
	}
}

func TestDirectory_AppRoleWritesOnlyWhatTheApplicationNeeds(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)

	allowed := []string{
		"UPDATE organization SET name = 'Renamed', version_id = version_id + 1, updated_at = now()",
		"INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ('" + clinicA + "', 'Bo', 'Example')",
	}
	for _, stmt := range allowed {
		if err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		}); err != nil {
			t.Errorf("%s as %s: %v", stmt, pgtest.AppRole, err)
		}
	}
	denied := []string{
		"INSERT INTO organization (name) VALUES ('Another clinic')",
		"UPDATE organization SET active = false",
		"UPDATE practitioner SET given_name = 'Cy'",
		"UPDATE practitioner_role SET active = false",
	}
	for table := range directoryTables {
		denied = append(denied, "DELETE FROM "+table, "TRUNCATE "+table)
	}
	for _, stmt := range denied {
		err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege (42501)", stmt, pgtest.AppRole, err)
		}
	}
}

func TestDirectory_APractitionerHasOneRoleInTheirOwnClinic(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	seeded := seedDirectory(t, owner)
	ctx := t.Context()
	var noRole string // a practitioner of clinic A without a role
	err := owner.QueryRow(ctx, "INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Bo', 'Example') RETURNING id",
		clinicA).Scan(&noRole)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, org, practitioner, wantErr string
	}{
		{name: "a second role", org: clinicA, practitioner: seeded[clinicA].practitioner, wantErr: "23505"},
		{name: "a role in another clinic", org: clinicB, practitioner: noRole, wantErr: "23503"},
		{name: "no such clinic", org: "00000000-0000-4000-8000-000000000000", practitioner: noRole, wantErr: "23503"},
		{name: "no such practitioner", org: clinicA, practitioner: "00000000-0000-4000-8000-000000000000", wantErr: "23503"},
		{name: "the first role", org: clinicA, practitioner: noRole},
	}
	for _, tt := range tests {
		_, err := owner.Exec(ctx, "INSERT INTO practitioner_role (organization_id, practitioner_id) VALUES ($1, $2)", tt.org, tt.practitioner)
		if got := sqlState(err); got != tt.wantErr {
			t.Errorf("%s: SQLSTATE %q (%v), want %q", tt.name, got, err, tt.wantErr)
		}
	}
	if _, err := owner.Exec(ctx, "INSERT INTO organization (name) VALUES ('  ')"); sqlState(err) != "23514" {
		t.Errorf("an organization with a blank name: %v, want check_violation", err)
	}
}
