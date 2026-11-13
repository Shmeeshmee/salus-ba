package migrations_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// 00043's shared-visit policies of encounter, document_draft, questionnaire_response and binary, as they were: what
// 00050's must admit, row for row. Each is evaluated over a copy of the table's rows named as the table, so the
// policy's own references to it ("binary".appointment_id) resolve to the copy.
var guardedBefore = map[string]string{
	"encounter_shared": `status = 'finished' AND history_shared(id, patient_id)`,
	"document_draft_shared": `status = 'final'
		AND NULLIF(current_setting('app.video_room', true), '') IS NULL
		AND EXISTS (SELECT 1 FROM encounter e
		            WHERE e.id = document_draft.encounter_id AND history_shared(e.id, e.patient_id))`,
	"questionnaire_response_shared": `status = 'completed'
		AND EXISTS (SELECT 1 FROM encounter e
		            WHERE e.appointment_id = questionnaire_response.appointment_id
		              AND e.patient_id = questionnaire_response.patient_id
		              AND history_shared(e.id, e.patient_id))`,
	"binary_shared": `status = 'clean'
		AND NULLIF(current_setting('app.questionnaire_response_id', true), '') IS NULL
		AND EXISTS (SELECT 1 FROM questionnaire_response r
		            WHERE r.id = "binary".questionnaire_response_id AND r.status = 'completed')
		AND EXISTS (SELECT 1 FROM encounter e
		            WHERE e.appointment_id = "binary".appointment_id
		              AND e.patient_id = "binary".patient_id
		              AND history_shared(e.id, e.patient_id))`,
}

// The table of each policy, as a type and as the name its policy refers to it by.
var guardedTables = map[string][2]string{
	"encounter_shared":              {"encounter", "encounter"},
	"document_draft_shared":         {"document_draft", "document_draft"},
	"questionnaire_response_shared": {"questionnaire_response", "questionnaire_response"},
	"binary_shared":                 {`"binary"`, `"binary"`},
}

// 00050 puts history_shared's own scope conditions in front of the call in the four shared-visit policies: whatever
// the scope (clinic, patient, account, room, response, directory, none) and the state of the sharing, each admits
// exactly the rows 00043's admitted.
func TestHistorySharing_TheGuardedPoliciesAdmitWhat00043Admitted(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	seedClinicC(t, owner, h)
	var policies map[string]string
	if err := owner.QueryRow(t.Context(), `SELECT jsonb_object_agg(policyname, qual) FROM pg_policies WHERE policyname = ANY($1)`,
		slices.Collect(maps.Keys(guardedBefore))).Scan(&policies); err != nil || len(policies) != len(guardedBefore) {
		t.Fatalf("policies %v (%v), want all four", policies, err)
	}
	scopes := sharingScopes(h)
	scopes["clinic A with a response set"] = map[string]string{"app.organization_id": clinicA, "app.questionnaire_response_id": h.r.completed[h.a.bPast]}
	scopes["patient B"] = map[string]string{"app.patient_id": h.patients[1].patient}
	admitted := map[string]int{}
	for _, state := range sharingStates(t, owner, h) {
		state.setup()
		for policy, before := range guardedBefore {
			table := guardedTables[policy]
			var rows []byte // the table's rows, read as the owner: no policy applies to the copy
			if err := owner.QueryRow(t.Context(), `SELECT coalesce(jsonb_agg(t), '[]') FROM `+table[0]+` t`).Scan(&rows); err != nil {
				t.Fatal(err)
			}
			q := `SELECT id::text FROM jsonb_populate_recordset(NULL::` + table[0] + `, $1) AS ` + table[1] + ` WHERE `
			for name, scope := range scopes {
				got := idsInWith(t, app, scope, q+"("+policies[policy]+")", rows)
				want := idsInWith(t, app, scope, q+"("+before+")", rows)
				admitted[policy] += len(want)
				if !slices.Equal(got, want) {
					t.Errorf("%s, %s: %s admits %v, 00043's admitted %v", state.name, name, policy, got, want)
				}
			}
		}
	}
	for policy := range guardedBefore {
		if admitted[policy] == 0 {
			t.Errorf("00043's %s admitted nothing in any state: the comparison proves nothing for it", policy)
		}
	}
}

// historySharedCalls reads every row of the four tables as the application role in scope and counts the calls of
// history_shared the reads made (track_functions, set by the owner for the transaction).
func historySharedCalls(t *testing.T, owner *pgxpool.Pool, scope map[string]string) int64 {
	t.Helper()
	ctx := t.Context()
	tx, err := owner.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	stmts := []string{`SET LOCAL track_functions = 'all'`, `SET LOCAL ROLE ` + pgtest.AppRole}
	for name, value := range scope {
		stmts = append(stmts, `SELECT set_config('`+name+`', '`+value+`', true)`)
	}
	for _, table := range guardedTables {
		stmts = append(stmts, `SELECT count(*) FROM `+table[0])
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	var calls int64
	if err := tx.QueryRow(ctx, `SELECT coalesce(sum(calls), 0) FROM pg_stat_xact_user_functions WHERE funcname = 'history_shared'`).
		Scan(&calls); err != nil {
		t.Fatal(err)
	}
	return calls
}

// A patient's and an account's reads never call history_shared, which is false at once in their scopes (the
// patient's timeline reads all four tables, SAL-39); a clinic's read with a visit shared with it still does.
func TestHistorySharing_APatientsReadDoesNotCallHistoryShared(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	share(t, owner, h.consent, h.encounters[h.a.bPast])
	for name, scope := range map[string]map[string]string{
		"patient A":                   {"app.patient_id": h.patients[0].patient},
		"patient A with clinic A set": {"app.patient_id": h.patients[0].patient, "app.organization_id": clinicA},
		"an account with clinic A":    {"app.account_id": h.patients[0].account, "app.organization_id": clinicA},
		"no scope":                    nil,
	} {
		if calls := historySharedCalls(t, owner, scope); calls != 0 {
			t.Errorf("%s: %d calls of history_shared, want none", name, calls)
		}
	}
	if calls := historySharedCalls(t, owner, map[string]string{"app.organization_id": clinicA}); calls == 0 {
		t.Error("clinic A, shared with: no call of history_shared counted; the count proves nothing")
	}
}
