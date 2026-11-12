package migrations_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const historyLookupRole = "salus_history_lookup"

// SAL-40: the version a shared completed response answers reaches the receiving clinic's scope only through
// history_questionnaire, only while the visit is shared, and never another of the other clinic's versions; the
// patient's scope and the other clinic's own reads are unchanged.
func TestHistoryQuestionnaire_GivesOnlyTheVersionASharedResponseAnswers(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	var answered string
	if err := owner.QueryRow(t.Context(), `SELECT questionnaire_id::text FROM questionnaire_response WHERE id = $1`,
		h.r.completed[h.a.bPast]).Scan(&answered); err != nil {
		t.Fatal(err)
	}
	clinicScope := map[string]string{"app.organization_id": clinicA}
	lookup := func(scope map[string]string) []string {
		t.Helper()
		var got []string
		for _, v := range h.r.versions[clinicB] {
			got = append(got, idsIn(t, app, scope, `SELECT id::text FROM history_questionnaire('`+v+`')`)...)
		}
		return got
	}
	if got := lookup(clinicScope); len(got) != 0 {
		t.Errorf("nothing shared: clinic A gets %v, want none", got)
	}
	if got := idsIn(t, app, clinicScope, `SELECT id::text FROM questionnaire WHERE organization_id = '`+clinicB+`'`); len(got) != 0 {
		t.Errorf("clinic A's scope sees clinic B's versions %v, want none", got)
	}

	share(t, owner, h.consent, h.encounters[h.a.bPast])
	if got := lookup(clinicScope); !slices.Equal(got, []string{answered}) {
		t.Errorf("shared: clinic A gets %v, want the version answered %s", got, answered)
	}
	if got := idsIn(t, app, clinicScope, `SELECT id::text FROM questionnaire WHERE organization_id = '`+clinicB+`'`); len(got) != 0 {
		t.Errorf("shared: clinic A's scope sees clinic B's versions %v directly, want none", got)
	}
	for name, scope := range map[string]map[string]string{
		"patient A":                        {"app.patient_id": h.patients[0].patient},
		"clinic A with an account set":     {"app.organization_id": clinicA, "app.account_id": h.patients[0].account},
		"no scope":                         {},
		"clinic B, the visit's own clinic": {"app.organization_id": clinicB},
	} {
		if got := lookup(scope); len(got) != 0 {
			t.Errorf("%s gets %v through the lookup, want none", name, got)
		}
	}

	ownerExec(t, owner, `UPDATE history_consent_encounter SET removed_at = now() WHERE consent_id = $1`, h.consent)
	if got := lookup(clinicScope); len(got) != 0 {
		t.Errorf("removed: clinic A gets %v, want none", got)
	}
}

func TestHistoryQuestionnaire_IsOwnedByAnUnprivilegedRoleWithAFixedSearchPath(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	ctx := t.Context()
	var fnOwner string
	var definer bool
	var config []string
	if err := owner.QueryRow(ctx, `SELECT pg_get_userbyid(proowner), prosecdef, coalesce(proconfig, '{}')
		FROM pg_proc WHERE proname = 'history_questionnaire'`).Scan(&fnOwner, &definer, &config); err != nil {
		t.Fatal(err)
	}
	if fnOwner != historyLookupRole || !definer || !slices.Contains(config, "search_path=pg_catalog, public") {
		t.Errorf("history_questionnaire: owner %q, SECURITY DEFINER %v, config %v; want %s, true and a fixed search_path",
			fnOwner, definer, config, historyLookupRole)
	}
	var super, bypass, login, createRole, createDB bool
	if err := owner.QueryRow(ctx, `SELECT rolsuper, rolbypassrls, rolcanlogin, rolcreaterole, rolcreatedb FROM pg_roles WHERE rolname = $1`,
		historyLookupRole).Scan(&super, &bypass, &login, &createRole, &createDB); err != nil || super || bypass || login || createRole || createDB {
		t.Errorf("%s: superuser %v, BYPASSRLS %v, LOGIN %v, CREATEROLE %v, CREATEDB %v (%v); want all false",
			historyLookupRole, super, bypass, login, createRole, createDB, err)
	}
	var publicExec, appExec bool
	if err := owner.QueryRow(ctx, `SELECT has_function_privilege('public', 'history_questionnaire(uuid)', 'EXECUTE'),
		has_function_privilege($1, 'history_questionnaire(uuid)', 'EXECUTE')`, pgtest.AppRole).Scan(&publicExec, &appExec); err != nil || publicExec || !appExec {
		t.Errorf("EXECUTE for PUBLIC %v, for %s %v (%v); want false and true", publicExec, pgtest.AppRole, appExec, err)
	}
	// The owner reads the questionnaire and only the columns of the sharing check elsewhere; it writes nothing.
	var tables []string
	if err := owner.QueryRow(ctx, `SELECT coalesce(array_agg(table_name || ' ' || privilege_type ORDER BY table_name), '{}')
		FROM information_schema.table_privileges WHERE grantee = $1`, historyLookupRole).Scan(&tables); err != nil ||
		!slices.Equal(tables, []string{"questionnaire SELECT"}) {
		t.Errorf("%s holds table privileges %v (%v), want SELECT on questionnaire only", historyLookupRole, tables, err)
	}
	var tablesWithColumns []string
	if err := owner.QueryRow(ctx, `SELECT coalesce(array_agg(DISTINCT table_name || ' ' || privilege_type), '{}')
		FROM information_schema.column_privileges WHERE grantee = $1 AND table_name <> 'questionnaire'`, historyLookupRole).Scan(&tablesWithColumns); err != nil ||
		!slices.Equal(tablesWithColumns, []string{"consent SELECT", "encounter SELECT", "history_consent SELECT", "history_consent_encounter SELECT", "questionnaire_response SELECT"}) {
		t.Errorf("%s holds column privileges on %v (%v)", historyLookupRole, tablesWithColumns, err)
	}
}
