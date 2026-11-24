package migrations_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The clinic's desk moves a booked visit to another practitioner of the clinic without counting it as the patient's
// reschedule (docs/clinic-cms WAVE4, M1); the patient's own scope still counts each move once and never changes the
// practitioner; nobody gives a visit to another clinic's practitioner, or changes the practitioner without a move.
func TestAppointment_TheClinicMovesAVisitToAnotherPractitionerWithoutCounting(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, dir, a := seedAppointments(t, owner)
	var roleA2 string
	if err := owner.QueryRow(t.Context(), `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
		VALUES ($1, 'Bo', 'Example') RETURNING id, organization_id)
		INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id`, clinicA).Scan(&roleA2); err != nil {
		t.Fatal(err)
	}
	own := map[string]string{"app.patient_id": p[0].patient}
	clinic := map[string]string{"app.organization_id": clinicA}
	run := func(scope map[string]string, q string, args ...any) error {
		return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, q, args...)
			if err == nil && tag.RowsAffected() != 1 {
				return errNoRow
			}
			return err
		})
	}
	const (
		moveTo = `UPDATE appointment SET period = tstzrange(lower(period) + interval '1 hour', upper(period) + interval '1 hour'),
			slot_id = gen_random_uuid(), practitioner_role_id = $2, version_id = version_id + 1 WHERE id = $1`
		moveToCounted = `UPDATE appointment SET period = tstzrange(lower(period) + interval '1 hour', upper(period) + interval '1 hour'),
			slot_id = gen_random_uuid(), practitioner_role_id = $2, reschedule_count = reschedule_count + 1 WHERE id = $1`
		roleOnly = `UPDATE appointment SET practitioner_role_id = $2 WHERE id = $1`
	)
	for _, tt := range []struct {
		scope   map[string]string
		name, q string
		role    string
		wantErr string // SQLSTATE; "" for stored
	}{
		{name: "the patient moves to another practitioner", scope: own, q: moveToCounted, role: roleA2, wantErr: "42501"},
		{name: "the patient names their practitioner", scope: own, q: roleOnly, role: dir[clinicA].role, wantErr: "42501"},
		{name: "the clinic changes the practitioner without a move", scope: clinic, q: roleOnly, role: roleA2, wantErr: "23514"},
		{name: "the clinic gives it to another clinic's practitioner", scope: clinic, q: moveTo, role: dir[clinicB].role, wantErr: "23514"},
		{name: "the clinic moves it to another practitioner", scope: clinic, q: moveTo, role: roleA2},
		{name: "the clinic moves it back, counting once", scope: clinic, q: moveToCounted, role: dir[clinicA].role},
	} {
		err := run(tt.scope, tt.q, a.aFuture, tt.role)
		if got := sqlState(err); got != tt.wantErr || tt.wantErr == "" && err != nil {
			t.Errorf("%s: %v, want %q", tt.name, err, tt.wantErr)
		}
	}
	var role string
	var count int
	if err := owner.QueryRow(t.Context(), `SELECT practitioner_role_id::text, reschedule_count FROM appointment WHERE id = $1`, a.aFuture).Scan(&role, &count); err != nil ||
		role != dir[clinicA].role || count != 1 {
		t.Errorf("role %s, count %d, %v; want back with its practitioner, counted once (the clinic's second move)", role, count, err)
	}
	// The patient's own move still counts exactly once.
	const moveFree = `UPDATE appointment SET period = tstzrange(lower(period) + interval '1 hour', upper(period) + interval '1 hour'),
		slot_id = gen_random_uuid() WHERE id = $1`
	if err := run(own, moveFree, a.aFuture); sqlState(err) != "23514" {
		t.Errorf("the patient moves without counting: %v, want a check violation", err)
	}
	if err := run(clinic, moveFree, a.aFuture); err != nil {
		t.Errorf("the clinic moves without counting: %v", err)
	}
}
