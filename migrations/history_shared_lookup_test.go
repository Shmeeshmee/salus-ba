package migrations_test

import (
	"context"
	"regexp"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// seedSlotSearch creates, as the owner, a directory of the size the free-slot search meets in production and
// returns the schedule searched: 20 listed clinics of 10 roles with a schedule each, about 22 000 slots (1 400 free
// ones in the next 14 days in the searched schedule), 600 patients with a care consent and two finished visits at
// two clinics, and 300 history consents sharing the visit at the other clinic. Then ANALYZE, so the planner costs
// the policies on the rows they read.
func seedSlotSearch(t *testing.T, owner *pgxpool.Pool) string {
	t.Helper()
	ctx := t.Context()
	if _, err := owner.Exec(ctx, `
		INSERT INTO organization (name, listed) SELECT 'Synthetic clinic', true FROM generate_series(1, 20);
		INSERT INTO practitioner (organization_id, given_name, family_name)
			SELECT o.id, 'Ada', 'Example' FROM organization o, generate_series(1, 10);
		INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM practitioner;
		INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
			SELECT organization_id, id, 10, '[{"day": 1, "start": "08:00", "end": "16:00"}]' FROM practitioner_role;
		INSERT INTO slot (schedule_id, organization_id, period, status)
			SELECT s.id, s.organization_id,
			       tstzrange(date_trunc('day', now()) + interval '1 day' + i * interval '10 minutes',
			                 date_trunc('day', now()) + interval '1 day' + (i + 1) * interval '10 minutes'),
			       CASE WHEN i % 3 = 0 THEN 'busy' ELSE 'free' END
			FROM schedule s, generate_series(0, 103) i;
		INSERT INTO account (email, password_hash, role)
			SELECT 'pat' || i || '@example.test', 'h', 'patient' FROM generate_series(1, 600) i;
		INSERT INTO patient (account_id, given_name, family_name, birth_date)
			SELECT id, 'Ada', 'Example', '1990-01-01' FROM account WHERE role = 'patient';
		WITH r AS (SELECT id, organization_id, row_number() OVER (ORDER BY id) AS n FROM practitioner_role),
		     p AS (SELECT id, row_number() OVER (ORDER BY id) AS n FROM patient)
		INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
			SELECT p.id, r.id, r.organization_id, gen_random_uuid(),
			       tstzrange('2026-10-01T09:00:00Z'::timestamptz + (p.n * 2 + v) * interval '1 hour',
			                 '2026-10-01T09:30:00Z'::timestamptz + (p.n * 2 + v) * interval '1 hour'), 'fulfilled'
			FROM p, generate_series(0, 1) v, r
			WHERE r.n = 1 + (p.n * 7 + v * 101) % 200;
		INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end)
			SELECT id, organization_id, patient_id, practitioner_role_id, 'finished', lower(period), lower(period) + interval '20 minutes'
			FROM appointment;
		INSERT INTO consent (patient_id, organization_id, status, date_time)
			SELECT DISTINCT patient_id, organization_id, 'active', '2026-09-01T08:00:00Z'::timestamptz FROM appointment;
		WITH visits AS (SELECT patient_id, organization_id, id, row_number() OVER (PARTITION BY patient_id ORDER BY period_start) AS v
		                FROM encounter),
		     h AS (INSERT INTO history_consent (patient_id, organization_id, status, date_time)
		           SELECT a.patient_id, a.organization_id, 'active', '2026-11-01T08:00:00Z'
		           FROM visits a JOIN visits b ON b.patient_id = a.patient_id AND b.v = 2
		           WHERE a.v = 1 AND a.organization_id <> b.organization_id
		             AND a.patient_id IN (SELECT id FROM patient ORDER BY id LIMIT 300)
		           RETURNING id, patient_id, organization_id)
		INSERT INTO history_consent_encounter (consent_id, encounter_id, added_at)
			SELECT h.id, b.id, now() FROM h JOIN visits b ON b.patient_id = h.patient_id AND b.v = 2;
		ANALYZE;`); err != nil {
		t.Fatal(err)
	}
	var schedule string
	if err := owner.QueryRow(ctx, `SELECT id FROM schedule ORDER BY id LIMIT 1`).Scan(&schedule); err != nil {
		t.Fatal(err)
	}
	// The searched schedule is a busy doctor's: free 10-minute slots around the clock for the next 14 days.
	ownerExec(t, owner, `DELETE FROM slot WHERE schedule_id = $1`, schedule)
	ownerExec(t, owner, `INSERT INTO slot (schedule_id, organization_id, period, status)
		SELECT s.id, s.organization_id,
		       tstzrange(date_trunc('day', now()) + interval '1 hour' + i * interval '10 minutes',
		                 date_trunc('day', now()) + interval '1 hour' + (i + 1) * interval '10 minutes'), 'free'
		FROM schedule s, generate_series(0, 1399) i WHERE s.id = $1`, schedule)
	ownerExec(t, owner, `ANALYZE slot`)
	return schedule
}

// The tables 00043's practitioner_role_shared and organization_shared pulled into every directory read of a slot.
var historyRelations = regexp.MustCompile(`\son (encounter|consent|history_consent|history_consent_encounter|appointment)\s`)

// The free-slot search (GET /fhir/R4/Slot?schedule=…&status=free, slot's SearchFreeSlots) stays under
// jit_optimize_above_cost with a margin, in the scopes the API reads it in, on a directory of production size. 00043's
// practitioner_role_shared and organization_shared were evaluated as a subplan per slot row (the slot's schedule's,
// the schedule's role's), each pulling in encounter, consent and appointment through their policies: cost 560 000
// to 765 000, and 82 s of JIT compilation for a 7 ms read (docs/plans/patient-history.PROGRESS.md). Neither scope can
// see a shared visit, so the plan reads none of those tables.
func TestHistorySharing_TheFreeSlotSearchStaysUnderJITsOptimizeCost(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	schedule := seedSlotSearch(t, owner)
	var patient string
	if err := owner.QueryRow(t.Context(), `SELECT id FROM patient ORDER BY id LIMIT 1`).Scan(&patient); err != nil {
		t.Fatal(err)
	}
	const search = `SELECT id, updated_at, schedule_id, organization_id, version_id, status, lower(period)::timestamptz AS start_at,
		upper(period)::timestamptz AS end_at
		FROM slot
		WHERE schedule_id = $1 AND status = 'free'
		  AND period && tstzrange(now(), now() + interval '14 days')
		  AND lower(period) >= now() AND lower(period) < now() + interval '14 days'
		ORDER BY lower(period)`
	for name, scope := range map[string]map[string]string{
		"directory":           {"app.directory": "public"},
		"directory + patient": {"app.directory": "public", "app.patient_id": patient},
	} {
		var cost, limit float64
		var text []string
		err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			var plan []struct {
				Plan struct {
					TotalCost float64 `json:"Total Cost"`
				}
			}
			if err := db.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+search, schedule).Scan(&plan); err != nil {
				return err
			}
			cost = plan[0].Plan.TotalCost
			rows, err := db.Query(ctx, "EXPLAIN "+search, schedule)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var line string
				if err := rows.Scan(&line); err != nil {
					return err
				}
				text = append(text, line)
			}
			if err := rows.Err(); err != nil {
				return err
			}
			return db.QueryRow(ctx, "SELECT current_setting('jit_optimize_above_cost')::float8").Scan(&limit)
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: cost %.0f", name, cost)
		if cost >= limit/5 {
			t.Errorf("%s: the free-slot search costs %.0f, want under a fifth of jit_optimize_above_cost %.0f", name, cost, limit)
		}
		if i := slices.IndexFunc(text, historyRelations.MatchString); i >= 0 {
			t.Errorf("%s: the free-slot search reads the history tables per row: %q", name, text[i])
		}
	}
}

// Clinic C: a third clinic, with patient A's active care consent and a role, that no visit is shared with.
const (
	historyClinicC = "c0000000-0000-4000-8000-00000000000c"
	historyRoleC   = "c1000000-0000-4000-8000-0000000000c1"
)

// seedClinicC adds, as the owner, clinic C, its role, and patient A's care consent with it.
func seedClinicC(t *testing.T, owner *pgxpool.Pool, h historyRows) {
	t.Helper()
	ownerExec(t, owner, `INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic C')`, historyClinicC)
	ownerExec(t, owner, `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Cy', 'Example') RETURNING id)
		INSERT INTO practitioner_role (id, organization_id, practitioner_id) SELECT $2, $1, id FROM p`, historyClinicC, historyRoleC)
	ownerExec(t, owner, `INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', '2026-11-01T08:00:00Z')`,
		h.patients[0].patient, historyClinicC)
}

// The shared visit's clinic and practitioner role are seen, of everyone but the visit's own clinic, only by the
// clinic the visit is shared with, while it is: not by another clinic with the patient's care consent and nothing
// shared, not after the history consent is revoked, not by the patient, an account, the directory or no scope.
func TestHistorySharing_TheVisitsClinicAndRoleAreSeenOnlyWhereTheVisitIsShared(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	seedClinicC(t, owner, h)
	share(t, owner, h.consent, h.encounters[h.a.bPast])
	shared := map[string][]string{"organization": {clinicB}, "practitioner_role": {h.r.roles[clinicB]}}
	scopes := map[string]struct {
		scope map[string]string
		want  map[string][]string
	}{
		"no scope":                    {},
		"patient A":                   {scope: map[string]string{"app.patient_id": h.patients[0].patient}},
		"patient A with clinic A set": {scope: map[string]string{"app.patient_id": h.patients[0].patient, "app.organization_id": clinicA}},
		"an account with clinic A":    {scope: map[string]string{"app.account_id": h.patients[0].account, "app.organization_id": clinicA}},
		"the directory":               {scope: map[string]string{"app.directory": "public"}},
		"the directory and patient A": {scope: map[string]string{"app.directory": "public", "app.patient_id": h.patients[0].patient}},
		"clinic A, shared with":       {scope: map[string]string{"app.organization_id": clinicA}, want: shared},
		"clinic C, nothing shared":    {scope: map[string]string{"app.organization_id": historyClinicC}},
		"clinic B, its own":           {scope: map[string]string{"app.organization_id": clinicB}, want: shared},
	}
	for name, tt := range scopes {
		for _, table := range []string{"organization", "practitioner_role"} {
			if got, want := idsIn(t, app, tt.scope, sharedQueries[table]), sorted(tt.want[table]...); !slices.Equal(got, want) {
				t.Errorf("%s: sees %s %v of clinic B, want %v", name, table, got, want)
			}
		}
	}
	ownerExec(t, owner, `UPDATE history_consent SET status = 'inactive', revoked_at = now() WHERE id = $1`, h.consent)
	for _, table := range []string{"organization", "practitioner_role"} {
		if got := idsIn(t, app, map[string]string{"app.organization_id": clinicA}, sharedQueries[table]); len(got) != 0 {
			t.Errorf("history consent revoked: clinic A sees %s %v of clinic B, want none", table, got)
		}
	}
}

// 00043's practitioner_role_shared and organization_shared, as they were: what 00046's policies must admit, row for
// row, in every scope and state.
const (
	sharedRoleBefore = `EXISTS (SELECT 1 FROM history_consent_encounter s JOIN encounter e ON e.id = s.encounter_id
		WHERE s.removed_at IS NULL AND e.practitioner_role_id = x.id AND history_shared(e.id, e.patient_id))`
	sharedOrganizationBefore = `EXISTS (SELECT 1 FROM history_consent_encounter s JOIN encounter e ON e.id = s.encounter_id
		WHERE s.removed_at IS NULL AND e.organization_id = x.id AND history_shared(e.id, e.patient_id))`
)

// 00046 changes how often the shared-visit check runs, not what it admits: its policies' conditions hold for exactly
// the rows 00043's held for, whatever the scope (clinic, patient, account, room, directory, none) and whatever the
// state of the sharing (nothing shared, shared, removed, revoked, the care consent revoked, the visit not finished).
func TestHistorySharing_ThePoliciesAdmitWhat00043Admitted(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	h := seedHistory(t, owner)
	seedClinicC(t, owner, h)
	var policies map[string]string
	if err := owner.QueryRow(t.Context(), `SELECT jsonb_object_agg(policyname, qual) FROM pg_policies
		WHERE policyname IN ('practitioner_role_shared', 'organization_shared')`).Scan(&policies); err != nil || len(policies) != 2 {
		t.Fatalf("policies %v (%v), want both", policies, err)
	}
	var roles, orgs []string
	if err := owner.QueryRow(t.Context(), `SELECT array_agg(id::text), (SELECT array_agg(id::text) FROM organization) FROM practitioner_role`).
		Scan(&roles, &orgs); err != nil {
		t.Fatal(err)
	}
	e := h.encounters[h.a.bPast]
	states := []struct {
		setup func()
		name  string
	}{
		{name: "nothing shared", setup: func() {}},
		{name: "shared", setup: func() { share(t, owner, h.consent, e) }},
		{name: "the visit not finished", setup: func() {
			ownerExec(t, owner, `UPDATE encounter SET status = 'in-progress', period_end = NULL WHERE id = $1`, e)
		}},
		{name: "removed", setup: func() {
			ownerExec(t, owner, `UPDATE encounter SET status = 'finished', period_end = period_start + interval '20 minutes' WHERE id = $1`, e)
			ownerExec(t, owner, `UPDATE history_consent_encounter SET removed_at = now() WHERE encounter_id = $1`, e)
		}},
		{name: "shared again", setup: func() { share(t, owner, h.consent, e) }},
		{name: "care consent revoked", setup: func() {
			ownerExec(t, owner, `UPDATE consent SET status = 'inactive', revoked_at = now() WHERE patient_id = $1 AND organization_id = $2`,
				h.patients[0].patient, clinicA)
		}},
		{name: "care consent again, history consent revoked", setup: func() {
			ownerExec(t, owner, `INSERT INTO consent (patient_id, organization_id, status, date_time) VALUES ($1, $2, 'active', now())`,
				h.patients[0].patient, clinicA)
			ownerExec(t, owner, `UPDATE history_consent SET status = 'inactive', revoked_at = now() WHERE id = $1`, h.consent)
		}},
	}
	scopes := map[string]map[string]string{
		"no scope":                    nil,
		"patient A":                   {"app.patient_id": h.patients[0].patient},
		"patient A with clinic A set": {"app.patient_id": h.patients[0].patient, "app.organization_id": clinicA},
		"an account with clinic A":    {"app.account_id": h.patients[0].account, "app.organization_id": clinicA},
		"the directory":               {"app.directory": "public"},
		"the directory and patient A": {"app.directory": "public", "app.patient_id": h.patients[0].patient},
		"clinic A":                    {"app.organization_id": clinicA},
		"clinic A in a room":          {"app.organization_id": clinicA, "app.video_room": "r_" + h.a.aFuture},
		"clinic B":                    {"app.organization_id": clinicB},
		"clinic C":                    {"app.organization_id": historyClinicC},
	}
	admitted := 0
	for _, state := range states {
		state.setup()
		for name, scope := range scopes {
			for _, c := range []struct {
				policy, before string
				ids            []string
			}{
				{"practitioner_role_shared", sharedRoleBefore, roles},
				{"organization_shared", sharedOrganizationBefore, orgs},
			} {
				const q = `SELECT x.id::text FROM unnest($1::uuid[]) AS x (id) WHERE `
				got := idsInWith(t, app, scope, q+"("+policies[c.policy]+")", c.ids)
				want := idsInWith(t, app, scope, q+c.before, c.ids)
				admitted += len(want)
				if !slices.Equal(got, want) {
					t.Errorf("%s, %s: %s admits %v, 00043's admitted %v", state.name, name, c.policy, got, want)
				}
			}
		}
	}
	if admitted == 0 {
		t.Error("00043's policies admitted nothing in any state: the comparison proves nothing")
	}
}

// idsInWith is idsIn with the query's arguments.
func idsInWith(t *testing.T, app *pgxpool.Pool, scope map[string]string, query string, args ...any) []string {
	t.Helper()
	var ids []string
	if err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
		return db.QueryRow(ctx, `SELECT coalesce(array_agg(id ORDER BY id), '{}') FROM (`+query+`) AS q (id)`, args...).Scan(&ids)
	}); err != nil {
		t.Fatal(err)
	}
	return ids
}

// The lookups run with the invoker's rights, so the policies of what they read apply as they did in 00043's EXISTS;
// with a fixed search path, and executable by the application role only.
func TestHistorySharing_TheLookupsKeepTheInvokersRightsAndAFixedSearchPath(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	for _, fn := range []string{"history_shared_organizations()", "history_shared_practitioner_roles()"} {
		var definer, publicExec, appExec bool
		var config []string
		if err := owner.QueryRow(t.Context(), `SELECT prosecdef, coalesce(proconfig, '{}'), has_function_privilege('public', oid, 'EXECUTE'),
			has_function_privilege($2, oid, 'EXECUTE') FROM pg_proc WHERE oid = $1::regprocedure`, fn, pgtest.AppRole).
			Scan(&definer, &config, &publicExec, &appExec); err != nil {
			t.Fatal(err)
		}
		if definer || !slices.Contains(config, "search_path=pg_catalog, public") || publicExec || !appExec {
			t.Errorf("%s: SECURITY DEFINER %v, config %v, EXECUTE for PUBLIC %v, for %s %v; want false, the search path, false, true",
				fn, definer, config, publicExec, pgtest.AppRole, appExec)
		}
	}
}
