package migrations_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const insertSession = `INSERT INTO session (account_id, organization_id, token_hash, mfa_verified, expires_at)
	VALUES ($1, $2, $3, $4, now() + interval '1 hour')`

func TestSession_BelongsToAnAccountAndAStaffSessionIsMFAVerified(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	var patient, admin string
	if err := app.QueryRow(ctx, insertAccount+" RETURNING id", "ana@example.test", "patient", nil).Scan(&patient); err != nil {
		t.Fatal(err)
	}
	if err := app.QueryRow(ctx, insertAccount+" RETURNING id", "ben@example.test", "admin", clinicA).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		account  string
		org      any
		hash     string
		name     string
		wantErr  string // SQLSTATE, "" for none
		verified bool
	}{
		{name: "a patient without MFA", account: patient, hash: "h1"},
		{name: "a patient with MFA", account: patient, hash: "h2", verified: true},
		{name: "staff with MFA", account: admin, org: clinicA, hash: "h3", verified: true},
		{name: "staff without MFA", account: admin, org: clinicA, hash: "h4", wantErr: "23514"},
		{name: "no such account", account: "00000000-0000-4000-8000-000000000000", hash: "h5", wantErr: "23503"},
		{name: "a token hash twice", account: patient, hash: "h1", wantErr: "23505"},
	}
	for _, tt := range tests {
		_, err := app.Exec(ctx, insertSession, tt.account, tt.org, []byte(tt.hash), tt.verified)
		if got := sqlState(err); got != tt.wantErr {
			t.Errorf("%s: SQLSTATE %q (%v), want %q", tt.name, got, err, tt.wantErr)
		}
	}
}

func TestSession_AppRoleCanReadWriteButNotDelete(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	var id string
	if err := app.QueryRow(ctx, insertAccount+" RETURNING id", "ana@example.test", "patient", nil).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Exec(ctx, insertSession, id, nil, []byte("h"), false); err != nil {
		t.Fatalf("insert as %s: %v", pgtest.AppRole, err)
	}
	if _, err := app.Exec(ctx, "UPDATE session SET revoked_at = now()"); err != nil {
		t.Errorf("update as %s: %v", pgtest.AppRole, err)
	}
	var n int
	if err := app.QueryRow(ctx, "SELECT count(*) FROM session").Scan(&n); err != nil || n != 1 {
		t.Errorf("select as %s: %d rows, %v", pgtest.AppRole, n, err)
	}
	for _, stmt := range []string{"DELETE FROM session", "TRUNCATE session"} {
		if _, err := app.Exec(ctx, stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.AppRole, err)
		}
	}
}
