package migrations_test

import (
	"context"
	"crypto/sha256"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const inviteLookupRole = "salus_invite_lookup"

// seedInvites creates, as the owner, clinics A and B with one open practitioner invite each, issued with the
// token "token <clinic>", and returns the token hashes by clinic.
func seedInvites(t *testing.T, owner *pgxpool.Pool) map[string][]byte {
	t.Helper()
	hashes := map[string][]byte{}
	for _, org := range []string{clinicA, clinicB} {
		sum := sha256.Sum256([]byte("token " + org))
		_, err := owner.Exec(t.Context(), `WITH o AS (INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic') RETURNING id)
			INSERT INTO invite (organization_id, expires_at, role, email, given_name, family_name, token_hash)
			SELECT id, now() + interval '72 hours', 'practitioner', 'bo@example.test', 'Bo', 'Example', $2 FROM o`, org, sum[:])
		if err != nil {
			t.Fatal(err)
		}
		hashes[org] = sum[:]
	}
	return hashes
}

func TestInvite_AppRoleSeesOnlyTheClinicSet(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedInvites(t, owner)

	for org, want := range map[string]int{"": 0, clinicA: 1} {
		var n, other int
		err := inClinic(t, app, org, func(ctx context.Context, db postgres.Querier) error {
			return db.QueryRow(ctx, "SELECT count(*), count(*) FILTER (WHERE organization_id <> $1) FROM invite", clinicA).Scan(&n, &other)
		})
		if err != nil || n != want || other != 0 {
			t.Errorf("invite with clinic %q set: %d rows, %d of another clinic, %v; want %d and 0", org, n, other, err, want)
		}
	}
	err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `INSERT INTO invite (organization_id, expires_at, role, email) VALUES ($1, now(), 'admin', 'cy@example.test')`, clinicB)
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("insert of an invite for clinic B with clinic A set: %v, want an RLS violation (42501)", err)
	}
	err = inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
		if _, err := db.Exec(ctx, "SET LOCAL row_security = off"); err != nil {
			return err
		}
		_, err := db.Exec(ctx, "SELECT * FROM invite")
		return err
	})
	if sqlState(err) != "42501" {
		t.Errorf("invite with row_security off: %v, want the RLS refusal (42501)", err)
	}
}

func TestInvite_AppRoleGetsTheClinicOfATokenOnlyFromTheLookupFunction(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	hashes := seedInvites(t, owner)
	ctx := t.Context()

	var n int
	if err := app.QueryRow(ctx, "SELECT count(*) FROM invite WHERE token_hash = $1", hashes[clinicB]).Scan(&n); err != nil || n != 0 {
		t.Errorf("salus_app without a clinic sees %d invites by token (%v), want 0", n, err)
	}
	var org string
	if err := app.QueryRow(ctx, "SELECT invite_organization($1)::text", hashes[clinicB]).Scan(&org); err != nil || org != clinicB {
		t.Errorf("invite_organization(token of clinic B) = %q, %v; want %s", org, err, clinicB)
	}
	var none *string
	unknown := sha256.Sum256([]byte("no such token"))
	if err := app.QueryRow(ctx, "SELECT invite_organization($1)::text", unknown[:]).Scan(&none); err != nil || none != nil {
		t.Errorf("invite_organization(unknown token) = %v, %v; want NULL", none, err)
	}
}

func TestInvite_LookupFunctionIsOwnedByAnUnprivilegedRoleWithAFixedSearchPath(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	ctx := t.Context()

	var fnOwner string
	var definer bool
	var config []string
	err := owner.QueryRow(ctx, `SELECT pg_get_userbyid(proowner), prosecdef, coalesce(proconfig, '{}')
		FROM pg_proc WHERE proname = 'invite_organization'`).Scan(&fnOwner, &definer, &config)
	if err != nil {
		t.Fatal(err)
	}
	if fnOwner != inviteLookupRole || !definer {
		t.Errorf("invite_organization is owned by %q, SECURITY DEFINER %v; want %s and true", fnOwner, definer, inviteLookupRole)
	}
	if !slices.Contains(config, "search_path=pg_catalog, public") {
		t.Errorf("invite_organization proconfig = %v, want search_path=pg_catalog, public", config)
	}
	var super, bypass, login, createRole, createDB bool
	err = owner.QueryRow(ctx, `SELECT rolsuper, rolbypassrls, rolcanlogin, rolcreaterole, rolcreatedb FROM pg_roles WHERE rolname = $1`,
		inviteLookupRole).Scan(&super, &bypass, &login, &createRole, &createDB)
	if err != nil || super || bypass || login || createRole || createDB {
		t.Errorf("%s: superuser %v, BYPASSRLS %v, LOGIN %v, CREATEROLE %v, CREATEDB %v (%v); want all false",
			inviteLookupRole, super, bypass, login, createRole, createDB, err)
	}
	var publicExec, appExec bool
	err = owner.QueryRow(ctx, `SELECT has_function_privilege('public', 'invite_organization(bytea)', 'EXECUTE'),
		has_function_privilege($1, 'invite_organization(bytea)', 'EXECUTE')`, pgtest.AppRole).Scan(&publicExec, &appExec)
	if err != nil || publicExec || !appExec {
		t.Errorf("EXECUTE for PUBLIC %v, for %s %v (%v); want false and true", publicExec, pgtest.AppRole, appExec, err)
	}
	// The owner of the function reads two columns of invite and nothing else.
	var cols []string
	err = owner.QueryRow(ctx, `SELECT coalesce(array_agg(column_name::text ORDER BY column_name), '{}')
		FROM information_schema.column_privileges WHERE grantee = $1`, inviteLookupRole).Scan(&cols)
	if err != nil || !slices.Equal(cols, []string{"organization_id", "token_hash"}) {
		t.Errorf("%s holds column privileges on %v (%v), want organization_id and token_hash", inviteLookupRole, cols, err)
	}
	var tables int
	err = owner.QueryRow(ctx, `SELECT count(*) FROM information_schema.table_privileges WHERE grantee = $1`, inviteLookupRole).Scan(&tables)
	if err != nil || tables != 0 {
		t.Errorf("%s holds %d table privileges (%v), want none", inviteLookupRole, tables, err)
	}
}

func TestInvite_AppRoleWritesOnlyWhatTheApplicationNeeds(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedInvites(t, owner)

	allowed := []string{
		`INSERT INTO invite (organization_id, expires_at, role, email) VALUES ('` + clinicA + `', now(), 'admin', 'bo@example.test')
			ON CONFLICT (organization_id, email) WHERE accepted_at IS NULL
			DO UPDATE SET created_at = now(), expires_at = now(), invited_by = NULL, role = 'admin', given_name = '', family_name = ''`,
		"UPDATE invite SET token_hash = '\\x00', accepted_at = now()",
	}
	for _, stmt := range allowed {
		if err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		}); err != nil {
			t.Errorf("%s as %s: %v", stmt, pgtest.AppRole, err)
		}
	}
	for _, stmt := range []string{"UPDATE invite SET email = 'dee@example.test'", "UPDATE invite SET organization_id = '" + clinicB + "'", "DELETE FROM invite", "TRUNCATE invite"} {
		err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege (42501)", stmt, pgtest.AppRole, err)
		}
	}
}

func TestInvite_HoldsOneOpenInvitePerEMailAndClinicAndNamesOnlyForPractitioners(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	seedInvites(t, owner)
	const insert = `INSERT INTO invite (organization_id, expires_at, role, email, given_name, family_name, accepted_at)
		VALUES ($1, now(), $2, $3, $4, $5, $6)`
	tests := []struct {
		name, org, role, email, given, family, wantErr string
		accepted                                       bool
	}{
		{name: "a second open invite", org: clinicA, role: "practitioner", email: "bo@example.test", given: "Bo", family: "Example", wantErr: "23505"},
		{name: "beside an accepted one", org: clinicA, role: "practitioner", email: "bo@example.test", given: "Bo", family: "Example", accepted: true},
		{name: "the same e-mail in another clinic", org: clinicB, role: "admin", email: "cy@example.test"},
		{name: "an admin with a name", org: clinicA, role: "admin", email: "dee@example.test", given: "Dee", wantErr: "23514"},
		{name: "a practitioner without a family name", org: clinicA, role: "practitioner", email: "eve@example.test", given: "Eve", wantErr: "23514"},
		{name: "a patient", org: clinicA, role: "patient", email: "fay@example.test", wantErr: "23514"},
		{name: "an e-mail not normalised", org: clinicA, role: "admin", email: "Gus@example.test", wantErr: "23514"},
	}
	for _, tt := range tests {
		var accepted any
		if tt.accepted {
			accepted = "2026-11-04T09:00:00Z"
		}
		_, err := owner.Exec(t.Context(), insert, tt.org, tt.role, tt.email, tt.given, tt.family, accepted)
		if got := sqlState(err); got != tt.wantErr {
			t.Errorf("%s: SQLSTATE %q (%v), want %q", tt.name, got, err, tt.wantErr)
		}
	}
}

func TestAuditEvent_NamesAnAgentThatIsNoAccount(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	_, err := app.Exec(t.Context(), `INSERT INTO audit_event (recorded, outcome, action, type_system, type_code, subtype_system,
		subtype_code, purpose_system, purpose_code, agent_name) VALUES (now(), 0, 'U', 's', 'c', 's', 'c', 's', 'c', 'salusctl')`)
	if err != nil {
		t.Errorf("insert of an audit event with agent_name: %v", err)
	}
}
