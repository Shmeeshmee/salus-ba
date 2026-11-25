package migrations_test

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A treatment plan's status only moves forward, for every role, the owner too (00085, U6): draft to active (booked)
// or revoked, active to completed or revoked, and never back; a plan is written as a draft. Once a step has its visit
// (a booked plan's), it is never deleted nor moved to another plan or visit, and a booked plan takes no new step; a
// draft's steps are laid out freely, and booking gives each its visit.
func TestCarePlanGuard_StatusMovesForwardAndABookedPlansStepsStay(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	g := seedPlans(t, owner)
	draft, booked, ended, service, a := g.draft, g.booked, g.ended, g.service, g.visits
	exec := func(stmt string, args ...any) error {
		_, err := owner.Exec(t.Context(), stmt, args...)
		return err
	}
	for _, tc := range []struct {
		name string
		stmt string
		args []any
		ok   bool
	}{ // in order: each runs on what the ones before left
		{"active back to draft", `UPDATE care_plan SET status = 'draft', starts_on = NULL, booked_at = NULL WHERE id = $1`, []any{booked}, false},
		{"draft straight to complete", `UPDATE care_plan SET status = 'completed', starts_on = '2030-01-07', booked_at = now() WHERE id = $1`, []any{draft}, false},
		{"the title of a booked plan", `UPDATE care_plan SET title = 'Renamed' WHERE id = $1`, []any{booked}, true},
		{"active to completed", `UPDATE care_plan SET status = 'completed' WHERE id = $1`, []any{ended}, true},
		{"completed back to active", `UPDATE care_plan SET status = 'active' WHERE id = $1`, []any{ended}, false},
		{"completed to revoked", `UPDATE care_plan SET status = 'revoked', revoked_at = now() WHERE id = $1`, []any{ended}, false},
		{"a booked plan's step deleted", `DELETE FROM care_plan_activity WHERE care_plan_id = $1 AND position = 1`, []any{booked}, false},
		{"a step to another visit", `UPDATE care_plan_activity SET appointment_id = $2 WHERE care_plan_id = $1 AND position = 1`, []any{booked, a.other}, false},
		{"a step's visit dropped", `UPDATE care_plan_activity SET appointment_id = NULL WHERE care_plan_id = $1 AND position = 1`, []any{booked}, false},
		{"a step to another plan", `UPDATE care_plan_activity SET care_plan_id = $2 WHERE care_plan_id = $1 AND position = 1`, []any{booked, draft}, false},
		{"a step to another service", `UPDATE care_plan_activity SET charge_item_definition_id = $2 WHERE care_plan_id = $1 AND position = 1`, []any{booked, service}, true},
		{"a step added to a booked plan", `INSERT INTO care_plan_activity (care_plan_id, position, charge_item_definition_id, day_offset, visit_mode)
			VALUES ($1, 3, $2, 0, 'video')`, []any{booked, service}, false},
		{"a booked plan's steps renumbered", `UPDATE care_plan_activity SET position = 3 - position WHERE care_plan_id = $1`, []any{booked}, true},
		{"a draft's step deleted", `DELETE FROM care_plan_activity WHERE care_plan_id = $1`, []any{draft}, true},
	} {
		err := exec(tc.stmt, tc.args...)
		if tc.ok && err != nil {
			t.Errorf("%s: %v, want it done", tc.name, err)
		} else if !tc.ok && sqlState(err) != "23514" {
			t.Errorf("%s = %v, want a check violation", tc.name, err)
		}
	}
	// The app's scope is held to it as well: the clinic cannot delete a booked plan's step either.
	if err := execIn(t, app, map[string]string{"app.organization_id": clinicA}, `DELETE FROM care_plan_activity WHERE care_plan_id = $1`, booked); sqlState(err) != "23514" {
		t.Errorf("the clinic deletes a booked plan's steps = %v, want a check violation", err)
	}
	if err := exec(`UPDATE care_plan SET status = 'revoked', revoked_at = now() WHERE id = $1`, booked); err != nil {
		t.Errorf("active to revoked: %v", err)
	}
	if err := exec(`UPDATE care_plan SET status = 'revoked', revoked_at = now() WHERE id = $1`, draft); err != nil {
		t.Errorf("draft to revoked: %v", err)
	}
	if err := exec(`UPDATE care_plan SET status = 'draft', revoked_at = NULL WHERE id = $1`, draft); sqlState(err) != "23514" {
		t.Errorf("revoked back to draft = %v, want a check violation", err)
	}
}

// guarded are the plans of the guard's test, of patient A at clinic A: a draft with one step, a plan booked with two
// steps and their visits, and a second booked plan to end.
type guarded struct {
	visits                        appointmentRows
	draft, booked, ended, service string
}

func seedPlans(t *testing.T, owner *pgxpool.Pool) guarded {
	t.Helper()
	p, _, a := seedAppointments(t, owner)
	ctx := t.Context()
	doctor := seedAccount(t, owner, clinicA, "practitioner", "doctor@example.test")
	g := guarded{visits: a, draft: "c0000000-0000-4000-8000-0000000000d1", booked: "c0000000-0000-4000-8000-0000000000d2", ended: "c0000000-0000-4000-8000-0000000000d3"}
	if err := owner.QueryRow(ctx, `INSERT INTO charge_item_definition (organization_id, name, price_minor) VALUES ($1, 'Synthetic service', 5000) RETURNING id`,
		clinicA).Scan(&g.service); err != nil {
		t.Fatal(err)
	}
	exec := func(stmt string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(ctx, stmt, args...); err != nil {
			t.Fatal(err)
		}
	}
	const write = `INSERT INTO care_plan (id, organization_id, patient_id, created_by_account_id, title) VALUES ($1, $2, $3, $4, 'Synthetic plan')`
	for _, id := range []string{g.draft, g.booked, g.ended} {
		exec(write, id, clinicA, p[0].patient, doctor) // a draft is written
	}
	if _, err := owner.Exec(ctx, `INSERT INTO care_plan (organization_id, patient_id, created_by_account_id, title, status, starts_on, booked_at)
		VALUES ($1, $2, $3, 'Synthetic plan', 'active', '2030-01-07', now())`, clinicA, p[0].patient, doctor); sqlState(err) != "23514" {
		t.Errorf("a plan written booked = %v, want a check violation", err)
	}
	const step = `INSERT INTO care_plan_activity (care_plan_id, position, charge_item_definition_id, day_offset, visit_mode) VALUES ($1, $2, $3, 0, 'video')`
	exec(step, g.draft, 1, g.service)
	exec(step, g.booked, 1, g.service)
	exec(step, g.booked, 2, g.service)
	// Booking gives each step its visit, and the plan its start.
	const visit = `UPDATE care_plan_activity SET appointment_id = $3 WHERE care_plan_id = $1 AND position = $2`
	exec(visit, g.booked, 1, a.aPast)
	exec(visit, g.booked, 2, a.aFuture)
	for _, id := range []string{g.booked, g.ended} {
		exec(`UPDATE care_plan SET status = 'active', starts_on = '2030-01-07', booked_at = now() WHERE id = $1`, id)
	}
	return g
}
