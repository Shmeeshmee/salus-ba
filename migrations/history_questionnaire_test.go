package migrations_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

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
	practitioner := practitionerOf(t, owner, h.r.roles[clinicA]) // patient A's practitioner at clinic A
	lookupAs := func(scope map[string]string, practitionerID string) []string {
		t.Helper()
		return idsIn(t, app, scope, `SELECT id::text FROM history_questionnaire('{`+strings.Join(h.r.versions[clinicB], ",")+`}'::uuid[], '`+practitionerID+`')`)
	}
	lookup := func(scope map[string]string) []string { t.Helper(); return lookupAs(scope, practitioner) }
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
	// A practitioner of the clinic without an appointment with the patient there has no history of them.
	if got := lookupAs(clinicScope, "a9000000-0000-4000-8000-0000000000a9"); len(got) != 0 {
		t.Errorf("shared, a practitioner without an appointment gets %v, want none", got)
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
	for _, fn := range []string{"history_questionnaire", "history_confirmed_conditions"} {
		if err := owner.QueryRow(ctx, `SELECT pg_get_userbyid(proowner), prosecdef, coalesce(proconfig, '{}')
			FROM pg_proc WHERE proname = $1`, fn).Scan(&fnOwner, &definer, &config); err != nil {
			t.Fatal(err)
		}
		if fnOwner != historyLookupRole || !definer || !slices.Contains(config, "search_path=pg_catalog, public") {
			t.Errorf("%s: owner %q, SECURITY DEFINER %v, config %v; want %s, true and a fixed search_path",
				fn, fnOwner, definer, config, historyLookupRole)
		}
	}
	var super, bypass, login, createRole, createDB bool
	if err := owner.QueryRow(ctx, `SELECT rolsuper, rolbypassrls, rolcanlogin, rolcreaterole, rolcreatedb FROM pg_roles WHERE rolname = $1`,
		historyLookupRole).Scan(&super, &bypass, &login, &createRole, &createDB); err != nil || super || bypass || login || createRole || createDB {
		t.Errorf("%s: superuser %v, BYPASSRLS %v, LOGIN %v, CREATEROLE %v, CREATEDB %v (%v); want all false",
			historyLookupRole, super, bypass, login, createRole, createDB, err)
	}
	for _, fn := range []string{"history_questionnaire(uuid[], uuid)", "history_confirmed_conditions(uuid[], uuid)"} {
		var publicExec, appExec bool
		if err := owner.QueryRow(ctx, `SELECT has_function_privilege('public', $2, 'EXECUTE'),
			has_function_privilege($1, $2, 'EXECUTE')`, pgtest.AppRole, fn).Scan(&publicExec, &appExec); err != nil || publicExec || !appExec {
			t.Errorf("%s: EXECUTE for PUBLIC %v, for %s %v (%v); want false and true", fn, publicExec, pgtest.AppRole, appExec, err)
		}
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
		!slices.Equal(tablesWithColumns, []string{"appointment SELECT", "condition SELECT", "consent SELECT", "encounter SELECT", "history_consent SELECT",
			"history_consent_encounter SELECT", "practitioner_role SELECT", "questionnaire_response SELECT"}) {
		t.Errorf("%s holds column privileges on %v (%v)", historyLookupRole, tablesWithColumns, err)
	}
}

// practitionerOf is the practitioner of the role id.
func practitionerOf(t *testing.T, owner *pgxpool.Pool, role string) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `SELECT practitioner_id::text FROM practitioner_role WHERE id = $1`, role).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// SAL-40: a shared visit's diagnoses read as confirmed only when their Condition is: the receiving clinic's scope
// learns which coded fields of a shared signed document are confirmed only through history_confirmed_conditions,
// only while the visit is shared, only for its practitioner with an appointment with the patient, and nothing of a
// Condition beyond the document and the field.
func TestHistoryConfirmedConditions_GivesOnlyASharedDocumentsConfirmedFields(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	shared, own := h.encounters[h.a.bPast], h.encounters[h.a.aPast]
	for _, e := range []string{shared, own} {
		ownerExec(t, owner, `INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id, document_draft_id,
				practitioner_role_id, field_key, code, verification_status)
			SELECT now(), e.organization_id, e.patient_id, e.id, $2, e.practitioner_role_id, k.key, 'J06.9', k.status
			FROM encounter e, (VALUES ('confirmed-key', 'confirmed'), ('unconfirmed-key', 'unconfirmed')) AS k (key, status)
			WHERE e.id = $1`, e, h.final[e])
	}
	clinicScope := map[string]string{"app.organization_id": clinicA}
	practitioner := practitionerOf(t, owner, h.r.roles[clinicA])
	lookupAs := func(scope map[string]string, practitionerID string) []string {
		t.Helper()
		return idsIn(t, app, scope, `SELECT document_draft_id::text || ' ' || field_key || ' ' || code FROM history_confirmed_conditions('{`+
			h.final[shared]+`,`+h.final[own]+`}'::uuid[], '`+practitionerID+`')`)
	}
	if got := lookupAs(clinicScope, practitioner); len(got) != 0 {
		t.Errorf("nothing shared: clinic A gets %v, want none", got)
	}
	share(t, owner, h.consent, shared)
	if got := lookupAs(clinicScope, practitioner); !slices.Equal(got, []string{h.final[shared] + " confirmed-key J06.9"}) {
		t.Errorf("shared: clinic A gets %v, want the shared document's confirmed field only", got)
	}
	if got := lookupAs(clinicScope, "a9000000-0000-4000-8000-0000000000a9"); len(got) != 0 {
		t.Errorf("shared, a practitioner without an appointment gets %v, want none", got)
	}
	for name, scope := range map[string]map[string]string{
		"patient A":                    {"app.patient_id": h.patients[0].patient},
		"clinic A with an account set": {"app.organization_id": clinicA, "app.account_id": h.patients[0].account},
		"no scope":                     {},
		"clinic B":                     {"app.organization_id": clinicB},
	} {
		if got := lookupAs(scope, practitioner); len(got) != 0 {
			t.Errorf("%s gets %v through the lookup, want none", name, got)
		}
	}
	ownerExec(t, owner, `UPDATE history_consent_encounter SET removed_at = now() WHERE consent_id = $1`, h.consent)
	if got := lookupAs(clinicScope, practitioner); len(got) != 0 {
		t.Errorf("removed: clinic A gets %v, want none", got)
	}
}
