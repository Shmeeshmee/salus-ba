package migrations_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const insertAccount = `INSERT INTO account (email, password_hash, role, organization_id) VALUES ($1, 'h', $2, $3)`

func TestAccount_OrganizationIsSetExactlyForStaff(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	tests := []struct {
		org     any
		role    string
		wantErr string // SQLSTATE, "" for none
	}{
		{role: "patient", org: nil},
		{role: "patient", org: clinicA, wantErr: "23514"},
		{role: "admin", org: clinicA},
		{role: "admin", org: nil, wantErr: "23514"},
		{role: "practitioner", org: clinicB},
		{role: "practitioner", org: nil, wantErr: "23514"},
		{role: "nurse", org: clinicA, wantErr: "23514"},
	}
	for i, tt := range tests {
		email := "user" + string(rune('a'+i)) + "@example.test"
		_, err := app.Exec(t.Context(), insertAccount, email, tt.role, tt.org)
		if got := sqlState(err); got != tt.wantErr {
			t.Errorf("insert role %s, clinic %v: SQLSTATE %q (%v), want %q", tt.role, tt.org, got, err, tt.wantErr)
		}
	}
}

func TestAccount_EmailIsUniqueAndStoredNormalised(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	if _, err := app.Exec(ctx, insertAccount, "ana@example.test", "patient", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Exec(ctx, insertAccount, "ana@example.test", "patient", nil); sqlState(err) != "23505" {
		t.Errorf("a second account with the same e-mail: %v, want unique_violation", err)
	}
	for _, e := range []string{"Ana@example.test", " ana2@example.test"} {
		if _, err := app.Exec(ctx, insertAccount, e, "patient", nil); sqlState(err) != "23514" {
			t.Errorf("an e-mail that is not trimmed and lower-cased: %v, want check_violation", err)
		}
	}
}

func TestAccount_AppRoleCanReadWriteButNotDelete(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	if _, err := app.Exec(ctx, insertAccount, "ben@example.test", "patient", nil); err != nil {
		t.Fatalf("insert as %s: %v", pgtest.AppRole, err)
	}
	if _, err := app.Exec(ctx, "UPDATE account SET password_hash = 'h2'"); err != nil {
		t.Errorf("update as %s: %v", pgtest.AppRole, err)
	}
	var n int
	if err := app.QueryRow(ctx, "SELECT count(*) FROM account").Scan(&n); err != nil || n != 1 {
		t.Errorf("select as %s: %d rows, %v", pgtest.AppRole, n, err)
	}
	for _, stmt := range []string{"DELETE FROM account", "TRUNCATE account"} {
		if _, err := app.Exec(ctx, stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.AppRole, err)
		}
	}
}
