package migrations_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const encounterLookupRole = "salus_encounter_lookup"

// The consultation's stream consumer (SAL-6) has only an encounter's id: encounter_organization gives it the
// clinic, and nothing else, so it can set the clinic's scope and read the encounter through row-level security.
// Outside any scope salus_app sees no encounter; the function answers for every encounter, a revoked one too
// (the scope decides the rest), and NULL for an unknown id.
func TestEncounter_AppRoleGetsTheClinicOfAnEncounterOnlyFromTheLookupFunction(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, _, a := seedAppointments(t, owner)
	e := seedEncounters(t, owner, a)
	ctx := t.Context()

	var n int
	if err := app.QueryRow(ctx, "SELECT count(*) FROM encounter").Scan(&n); err != nil || n != 0 {
		t.Errorf("salus_app without a scope sees %d encounters (%v), want 0", n, err)
	}
	for appt, clinic := range map[string]string{a.aPast: clinicA, a.bFuture: clinicB} {
		var org string
		if err := app.QueryRow(ctx, "SELECT encounter_organization($1)::text", e[appt]).Scan(&org); err != nil || org != clinic {
			t.Errorf("encounter_organization(%s) = %q, %v; want %s", e[appt], org, err, clinic)
		}
	}
	var none *string
	if err := app.QueryRow(ctx, "SELECT encounter_organization(gen_random_uuid())::text").Scan(&none); err != nil || none != nil {
		t.Errorf("encounter_organization(unknown) = %v, %v; want NULL", none, err)
	}
}

func TestEncounter_LookupFunctionIsOwnedByAnUnprivilegedRoleWithAFixedSearchPath(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	ctx := t.Context()

	var fnOwner string
	var definer bool
	var config []string
	err := owner.QueryRow(ctx, `SELECT pg_get_userbyid(proowner), prosecdef, coalesce(proconfig, '{}')
		FROM pg_proc WHERE proname = 'encounter_organization'`).Scan(&fnOwner, &definer, &config)
	if err != nil {
		t.Fatal(err)
	}
	if fnOwner != encounterLookupRole || !definer {
		t.Errorf("encounter_organization is owned by %q, SECURITY DEFINER %v; want %s and true", fnOwner, definer, encounterLookupRole)
	}
	if !slices.Contains(config, "search_path=pg_catalog, public") {
		t.Errorf("encounter_organization proconfig = %v, want search_path=pg_catalog, public", config)
	}
	var super, bypass, login, createRole, createDB bool
	err = owner.QueryRow(ctx, `SELECT rolsuper, rolbypassrls, rolcanlogin, rolcreaterole, rolcreatedb FROM pg_roles WHERE rolname = $1`,
		encounterLookupRole).Scan(&super, &bypass, &login, &createRole, &createDB)
	if err != nil || super || bypass || login || createRole || createDB {
		t.Errorf("%s: superuser %v, BYPASSRLS %v, LOGIN %v, CREATEROLE %v, CREATEDB %v (%v); want all false",
			encounterLookupRole, super, bypass, login, createRole, createDB, err)
	}
	var publicExec, appExec bool
	err = owner.QueryRow(ctx, `SELECT has_function_privilege('public', 'encounter_organization(uuid)', 'EXECUTE'),
		has_function_privilege($1, 'encounter_organization(uuid)', 'EXECUTE')`, pgtest.AppRole).Scan(&publicExec, &appExec)
	if err != nil || publicExec || !appExec {
		t.Errorf("EXECUTE for PUBLIC %v, for %s %v (%v); want false and true", publicExec, pgtest.AppRole, appExec, err)
	}
	// The owner of the function reads two columns of encounter and nothing else.
	var cols []string
	err = owner.QueryRow(ctx, `SELECT coalesce(array_agg(table_name || '.' || column_name ORDER BY column_name), '{}')
		FROM information_schema.column_privileges WHERE grantee = $1`, encounterLookupRole).Scan(&cols)
	if err != nil || !slices.Equal(cols, []string{"encounter.id", "encounter.organization_id"}) {
		t.Errorf("%s holds column privileges on %v (%v), want encounter.id and encounter.organization_id", encounterLookupRole, cols, err)
	}
	var tables int
	err = owner.QueryRow(ctx, `SELECT count(*) FROM information_schema.table_privileges WHERE grantee = $1`, encounterLookupRole).Scan(&tables)
	if err != nil || tables != 0 {
		t.Errorf("%s holds %d table privileges (%v), want none", encounterLookupRole, tables, err)
	}
}
