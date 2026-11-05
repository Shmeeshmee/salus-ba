package migrations_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// patientRow is one seeded patient: the account and the profile.
type patientRow struct {
	account, patient string
}

// seedPatients creates, as the owner, two patient accounts with a synthetic profile each.
func seedPatients(t *testing.T, owner *pgxpool.Pool) [2]patientRow {
	t.Helper()
	var rows [2]patientRow
	for i, email := range []string{"pat.a@example.test", "pat.b@example.test"} {
		err := owner.QueryRow(t.Context(), `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES ($1, 'h', 'patient') RETURNING id)
			INSERT INTO patient (account_id, given_name, family_name, birth_date)
			SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING account_id, id`, email).
			Scan(&rows[i].account, &rows[i].patient)
		if err != nil {
			t.Fatal(err)
		}
	}
	return rows
}

// asApp runs fn in a transaction on the app pool with each setting of scope set.
func asApp(t *testing.T, app *pgxpool.Pool, scope map[string]string, fn func(ctx context.Context, db postgres.Querier) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		for name, value := range scope {
			if err := postgres.SetLocal(ctx, name, value); err != nil {
				return err
			}
		}
		return fn(ctx, postgres.Conn(ctx, app))
	})
}

func TestPatient_AppRoleSeesOnlyThePatientOfItsScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p := seedPatients(t, owner)

	tests := []struct {
		scope map[string]string
		name  string
		want  string // the one patient id visible, "" for none
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: p[0].patient},
		{name: "account A", scope: map[string]string{"app.account_id": p[0].account}, want: p[0].patient},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}, want: p[1].patient},
		{name: "a clinic", scope: map[string]string{"app.organization_id": clinicA}},
		{name: "a staff account", scope: map[string]string{"app.account_id": "c0000000-0000-4000-8000-00000000000c"}},
	}
	for _, tt := range tests {
		var ids []string
		err := asApp(t, app, tt.scope, func(ctx context.Context, db postgres.Querier) error {
			rows, err := db.Query(ctx, "SELECT id::text FROM patient")
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err != nil {
					return err
				}
				ids = append(ids, id)
			}
			return rows.Err()
		})
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if (tt.want == "" && len(ids) != 0) || (tt.want != "" && (len(ids) != 1 || ids[0] != tt.want)) {
			t.Errorf("%s: sees %v, want only %q", tt.name, ids, tt.want)
		}
	}
	// After a transaction with a scope, the pooled connection reads the setting as '', which matches nothing.
	var n int
	if err := app.QueryRow(t.Context(), "SELECT count(*) FROM patient").Scan(&n); err != nil || n != 0 {
		t.Errorf("after a scoped transaction: %d rows, %v; want 0", n, err)
	}
}

func TestPatient_AppRoleWritesOnlyItsOwnProfile(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p := seedPatients(t, owner)
	var other string
	if err := owner.QueryRow(t.Context(), "INSERT INTO account (email, password_hash, role) VALUES ('pat.c@example.test', 'h', 'patient') RETURNING id").Scan(&other); err != nil {
		t.Fatal(err)
	}
	scopeA := map[string]string{"app.account_id": p[0].account}

	err := asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, "INSERT INTO patient (account_id, given_name, family_name, birth_date) VALUES ($1, 'Bo', 'Example', '1991-02-02')", other)
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("insert of another account's profile: %v, want an RLS violation (42501)", err)
	}
	var n int64
	err = asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
		tag, err := db.Exec(ctx, "UPDATE patient SET given_name = 'Cy', version_id = version_id + 1 WHERE id = $1", p[1].patient)
		n = tag.RowsAffected()
		return err
	})
	if err != nil || n != 0 {
		t.Errorf("update of patient B in account A's scope: %d rows, %v; want 0", n, err)
	}
	err = asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
		tag, err := db.Exec(ctx, "UPDATE patient SET given_name = 'Cy', version_id = version_id + 1 WHERE id = $1", p[0].patient)
		n = tag.RowsAffected()
		return err
	})
	if err != nil || n != 1 {
		t.Errorf("update of patient A in its own scope: %d rows, %v; want 1", n, err)
	}
	for _, stmt := range []string{"DELETE FROM patient", "TRUNCATE patient", "UPDATE patient SET account_id = account_id"} {
		err := asApp(t, app, scopeA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.AppRole, err)
		}
	}
}

func TestPatient_HoldsOneValidProfilePerAccount(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	p := seedPatients(t, owner)
	const insert = "INSERT INTO patient (account_id, given_name, family_name, birth_date, phone) VALUES ($1, $2, 'Example', $3, $4)"
	tests := []struct {
		phone                       any
		name, given, birth, wantErr string
	}{
		{name: "a second profile", given: "Ada", birth: "1990-01-01", wantErr: "23505"},
		{name: "a blank name", given: "  ", birth: "1990-01-01", wantErr: "23514"},
		{name: "before 1900", given: "Ada", birth: "1899-12-31", wantErr: "23514"},
		{name: "a phone not in E.164", given: "Ada", birth: "1990-01-01", phone: "060 000 0001", wantErr: "23514"},
	}
	for _, tt := range tests {
		_, err := owner.Exec(t.Context(), insert, p[0].account, tt.given, tt.birth, tt.phone)
		if got := sqlState(err); got != tt.wantErr {
			t.Errorf("%s: SQLSTATE %q (%v), want %q", tt.name, got, err, tt.wantErr)
		}
	}
}
