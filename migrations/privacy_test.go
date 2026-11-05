package migrations_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const (
	clinicA = "a0000000-0000-4000-8000-00000000000a"
	clinicB = "b0000000-0000-4000-8000-00000000000b"
)

// insertAuditEvent writes one synthetic audit event through db.
func insertAuditEvent(ctx context.Context, db postgres.Querier) error {
	_, err := db.Exec(ctx, `INSERT INTO audit_event
		(recorded, outcome, action, type_system, type_code, subtype_system, subtype_code, purpose_system, purpose_code)
		VALUES (now(), 0, 'E', 'http://dicom.nema.org/resources/ontology/DCM', '110114',
		        'http://dicom.nema.org/resources/ontology/DCM', '110122',
		        'http://terminology.hl7.org/CodeSystem/v3-ActReason', 'HOPERAT')`)
	return err
}

// sqlState returns the SQLSTATE of a Postgres error, or "" for nil and other errors.
func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func TestUp_AppRoleHasNoPrivilegedAttributes(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)

	var login, super, bypassRLS, createRole, createDB bool
	err := db.QueryRow(t.Context(), `SELECT rolcanlogin, rolsuper, rolbypassrls, rolcreaterole, rolcreatedb
		FROM pg_roles WHERE rolname = $1`, pgtest.AppRole).Scan(&login, &super, &bypassRLS, &createRole, &createDB)
	if err != nil {
		t.Fatalf("role %s: %v", pgtest.AppRole, err)
	}
	if !login || super || bypassRLS || createRole || createDB {
		t.Errorf("login=%v superuser=%v bypassrls=%v createrole=%v createdb=%v, want only login",
			login, super, bypassRLS, createRole, createDB)
	}
}

func TestUp_AppRoleOwnsNothing(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)

	var owned int
	err := db.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM pg_class WHERE relowner = r.oid) +
		(SELECT count(*) FROM pg_proc WHERE proowner = r.oid) +
		(SELECT count(*) FROM pg_namespace WHERE nspowner = r.oid)
		FROM pg_roles r WHERE r.rolname = $1`, pgtest.AppRole).Scan(&owned)
	if err != nil {
		t.Fatal(err)
	}
	if owned != 0 {
		t.Errorf("%s owns %d tables, functions or schemas, want none", pgtest.AppRole, owned)
	}
}

// newClinicTable creates a clinic-scoped table the way package 4's tables will be (RLS enabled, a policy on
// app.organization_id, grants to the app role) and one row for each of two clinics. Test-only: no migration
// has such a table yet.
func newClinicTable(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	_, err := owner.Exec(t.Context(), `
		CREATE TABLE clinic_probe (organization_id uuid NOT NULL);
		ALTER TABLE clinic_probe ENABLE ROW LEVEL SECURITY;
		CREATE POLICY clinic_probe_clinic ON clinic_probe
			USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
		GRANT SELECT, INSERT ON clinic_probe TO `+pgtest.AppRole+`;
		INSERT INTO clinic_probe VALUES ('`+clinicA+`'), ('`+clinicB+`');`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUp_AppRoleIsBoundByRowLevelSecurity(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	newClinicTable(t, owner)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()

	visible := func(ctx context.Context) (n int, other int) {
		t.Helper()
		err := postgres.Conn(ctx, app).QueryRow(ctx,
			"SELECT count(*), count(*) FILTER (WHERE organization_id <> $1) FROM clinic_probe", clinicA).Scan(&n, &other)
		if err != nil {
			t.Fatal(err)
		}
		return n, other
	}

	if n, _ := visible(ctx); n != 0 {
		t.Errorf("no clinic set: %d rows visible, want 0", n)
	}
	err := tr.WithTx(ctx, func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.organization_id", clinicA); err != nil {
			return err
		}
		if n, other := visible(ctx); n != 1 || other != 0 {
			t.Errorf("clinic A set: %d rows visible, %d of another clinic; want 1 and 0", n, other)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.organization_id", clinicA); err != nil {
			return err
		}
		_, err := postgres.Conn(ctx, app).Exec(ctx, "INSERT INTO clinic_probe VALUES ($1)", clinicB)
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("insert for clinic B with clinic A set: error = %v, want an RLS violation (42501)", err)
	}
	if n, _ := visible(ctx); n != 0 {
		t.Errorf("after the transaction: %d rows visible, want 0", n)
	}
}

func TestUp_AppRoleCannotTurnRowLevelSecurityOff(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	newClinicTable(t, owner)

	err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if _, err := postgres.Conn(ctx, app).Exec(ctx, "SET LOCAL row_security = off"); err != nil {
			return err
		}
		_, err := postgres.Conn(ctx, app).Exec(ctx, "SELECT * FROM clinic_probe")
		return err
	})
	if sqlState(err) != "42501" || !strings.Contains(err.Error(), "row-level security") {
		t.Fatalf("query with row_security off: error = %v, want the RLS refusal (42501)", err)
	}
}

func TestUp_AuditEventIsInsertAndSelectOnlyForTheAppRole(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()

	if err := insertAuditEvent(ctx, app); err != nil {
		t.Fatalf("insert as %s: %v", pgtest.AppRole, err)
	}
	var n int
	if err := app.QueryRow(ctx, "SELECT count(*) FROM audit_event").Scan(&n); err != nil || n != 1 {
		t.Fatalf("select as %s: %d rows, error %v; want 1 row", pgtest.AppRole, n, err)
	}
	for _, stmt := range []string{
		"UPDATE audit_event SET outcome = 4",
		"DELETE FROM audit_event",
		"TRUNCATE audit_event",
	} {
		// 42501 is the privilege check, which runs before any trigger.
		if _, err := app.Exec(ctx, stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: error = %v, want permission denied (42501)", stmt, pgtest.AppRole, err)
		}
	}
}

func TestUp_AuditEventRejectsChangesEvenFromTheOwner(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	ctx := t.Context()
	if err := insertAuditEvent(ctx, owner); err != nil {
		t.Fatal(err)
	}

	for _, stmt := range []string{
		"UPDATE audit_event SET outcome = 4",
		"DELETE FROM audit_event",
		"TRUNCATE audit_event",
	} {
		_, err := owner.Exec(ctx, stmt)
		if sqlState(err) != "P0001" || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("%s as the owner: error = %v, want the append-only trigger (P0001)", stmt, err)
		}
	}
}

// The retention job's events name a table and its row count: both or neither, and never a negative count.
func TestUp_AuditEventTakesATableWithItsRowCountOnly(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	insert := func(name, count string) error {
		_, err := owner.Exec(t.Context(), `INSERT INTO audit_event
			(recorded, outcome, action, type_system, type_code, subtype_system, subtype_code, purpose_system, purpose_code,
			 entity_name, entity_count)
			VALUES (now(), 0, 'D', 's', 'c', 's', 'c', 's', 'c', `+name+`, `+count+`)`)
		return err
	}
	for _, ok := range [][2]string{{"'session'", "0"}, {"'slot'", "12"}, {"NULL", "NULL"}} {
		if err := insert(ok[0], ok[1]); err != nil {
			t.Errorf("table %s with count %s: %v, want stored", ok[0], ok[1], err)
		}
	}
	for _, bad := range [][2]string{{"'session'", "NULL"}, {"NULL", "3"}, {"'session'", "-1"}} {
		if err := insert(bad[0], bad[1]); sqlState(err) != "23514" {
			t.Errorf("table %s with count %s: %v, want a check violation (23514)", bad[0], bad[1], err)
		}
	}
}

func TestUp_AuditEventIsIndexedForEntityAndAgentLookups(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)

	rows, err := db.Query(t.Context(), "SELECT indexdef FROM pg_indexes WHERE tablename = 'audit_event'")
	if err != nil {
		t.Fatal(err)
	}
	var defs []string
	for rows.Next() {
		var def string
		if err := rows.Scan(&def); err != nil {
			t.Fatal(err)
		}
		defs = append(defs, def)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(defs, "\n")
	for _, cols := range []string{"(entity_reference, recorded)", "(agent_account_id, recorded)"} {
		if !strings.Contains(all, cols) {
			t.Errorf("no index on %s; indexes:\n%s", cols, all)
		}
	}
}
