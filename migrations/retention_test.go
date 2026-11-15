package migrations_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/retention"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// seedEnded creates, as the owner, clinics A and B, a patient account, a session of that account and an
// accepted invite in each clinic, all long over.
func seedEnded(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	_, err := owner.Exec(t.Context(), `
		INSERT INTO organization (id, name) VALUES ('`+clinicA+`', 'Synthetic clinic A'), ('`+clinicB+`', 'Synthetic clinic B');
		WITH a AS (INSERT INTO account (email, password_hash, role) VALUES ('ana@example.test', 'h', 'patient') RETURNING id)
		INSERT INTO session (account_id, token_hash, mfa_verified, expires_at)
			SELECT id, 'h', false, now() - interval '90 days' FROM a;
		INSERT INTO invite (organization_id, expires_at, accepted_at, role, email)
			VALUES ('`+clinicA+`', now() - interval '90 days', now() - interval '91 days', 'admin', 'bo@example.test'),
			       ('`+clinicB+`', now() - interval '90 days', now() - interval '91 days', 'admin', 'cy@example.test');`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRetention_RoleBypassesRowLevelSecurityAndHasNoOtherPrivilegedAttribute(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)

	var login, super, bypassRLS, createRole, createDB bool
	err := db.QueryRow(t.Context(), `SELECT rolcanlogin, rolsuper, rolbypassrls, rolcreaterole, rolcreatedb
		FROM pg_roles WHERE rolname = $1`, pgtest.RetentionRole).Scan(&login, &super, &bypassRLS, &createRole, &createDB)
	if err != nil {
		t.Fatalf("role %s: %v", pgtest.RetentionRole, err)
	}
	if !login || !bypassRLS || super || createRole || createDB {
		t.Errorf("login=%v bypassrls=%v superuser=%v createrole=%v createdb=%v, want login and bypassrls only",
			login, bypassRLS, super, createRole, createDB)
	}
}

func TestRetention_RoleOwnsNothing(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)

	var owned int
	err := db.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM pg_class WHERE relowner = r.oid) +
		(SELECT count(*) FROM pg_proc WHERE proowner = r.oid) +
		(SELECT count(*) FROM pg_namespace WHERE nspowner = r.oid)
		FROM pg_roles r WHERE r.rolname = $1`, pgtest.RetentionRole).Scan(&owned)
	if err != nil {
		t.Fatal(err)
	}
	if owned != 0 {
		t.Errorf("%s owns %d tables, functions or schemas, want none", pgtest.RetentionRole, owned)
	}
}

// The role holds SELECT and DELETE on exactly the tables whose rule is delete, and no other privilege on any
// table or column: a migration that adds a delete rule grants it in the same file.
func TestRetention_RoleIsGrantedSelectAndDeleteOnExactlyTheTablesWhoseRuleIsDelete(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)
	want := map[string][]string{}
	for _, r := range retention.Rules(retention.DefaultPolicy()) {
		if r.Then == retention.ThenDelete {
			want[r.Table] = []string{"DELETE", "SELECT"}
		}
	}

	rows, err := db.Query(t.Context(), `
		SELECT c.relname, a.privilege_type
			FROM pg_class c, aclexplode(c.relacl) a
			WHERE a.grantee = $1::regrole
		UNION
		SELECT c.relname || '.' || att.attname, a.privilege_type
			FROM pg_attribute att JOIN pg_class c ON c.oid = att.attrelid, aclexplode(att.attacl) a
			WHERE a.grantee = $1::regrole
		ORDER BY 1, 2`, pgtest.RetentionRole)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for rows.Next() {
		var table, privilege string
		if err := rows.Scan(&table, &privilege); err != nil {
			t.Fatal(err)
		}
		got[table] = append(got[table], privilege)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("%s holds %v, want %v", pgtest.RetentionRole, got, want)
	}
}

func TestRetention_RoleDeletesEndedSessionsAndInvitesOfEveryClinic(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	seedEnded(t, owner)

	// No clinic is set: the role sees every clinic's invites, past row-level security.
	for stmt, want := range map[string]int64{
		"DELETE FROM session WHERE expires_at < now() - interval '30 days'": 1,
		"DELETE FROM invite WHERE accepted_at < now() - interval '30 days'": 2,
	} {
		tag, err := ret.Exec(t.Context(), stmt)
		if err != nil || tag.RowsAffected() != want {
			t.Errorf("%s as %s: %d rows, %v; want %d", stmt, pgtest.RetentionRole, tag.RowsAffected(), err, want)
		}
	}
}

func TestRetention_RoleCannotInsertOrUpdate(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	seedEnded(t, owner)

	for _, stmt := range []string{
		"UPDATE session SET revoked_at = now()",
		"INSERT INTO session (account_id, token_hash, mfa_verified, expires_at) SELECT account_id, 'h2', false, now() FROM session",
		"UPDATE invite SET accepted_at = now()",
		"INSERT INTO invite (organization_id, expires_at, role, email) VALUES ('" + clinicA + "', now(), 'admin', 'dee@example.test')",
		"TRUNCATE session",
		"TRUNCATE invite",
	} {
		if _, err := ret.Exec(t.Context(), stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.RetentionRole, err)
		}
	}
}

func TestRetention_RoleCannotTouchAuditEvents(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	if err := insertAuditEvent(t.Context(), owner); err != nil {
		t.Fatal(err)
	}

	for _, stmt := range []string{
		"SELECT count(*) FROM audit_event",
		"DELETE FROM audit_event",
		"TRUNCATE audit_event",
		"UPDATE audit_event SET outcome = 4",
	} {
		// 42501 is the privilege check, which runs before the append-only triggers.
		if _, err := ret.Exec(t.Context(), stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.RetentionRole, err)
		}
	}
}
