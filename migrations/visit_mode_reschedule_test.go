package migrations_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A reschedule keeps the appointment's visit mode (docs/plans/clinic-cms.md, package A): the patient moves a video
// appointment to a video slot, never to an in-person one, and an in-person appointment the other way round.
func TestAppointment_ARescheduleStaysInItsVisitMode(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, dir, a := seedAppointments(t, owner)
	var schedule string
	if err := owner.QueryRow(t.Context(), `INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
		VALUES ($1, $2, 30, '[]') RETURNING id`, clinicA, dir[clinicA].role).Scan(&schedule); err != nil {
		t.Fatal(err)
	}
	slot := func(start, mode string) string {
		var id string
		if err := owner.QueryRow(t.Context(), `INSERT INTO slot (schedule_id, organization_id, period, status, visit_mode)
			VALUES ($1, $2, tstzrange($3::timestamptz, $3::timestamptz + interval '30 minutes'), 'busy', $4) RETURNING id`,
			schedule, clinicA, start, mode).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	move := func(start, slotID string) error {
		return asApp(t, app, map[string]string{"app.patient_id": p[0].patient}, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `UPDATE appointment SET period = tstzrange($2::timestamptz, $2::timestamptz + interval '30 minutes'),
				slot_id = $3, reschedule_count = reschedule_count + 1, version_id = version_id + 1 WHERE id = $1`, a.aFuture, start, slotID)
			return err
		})
	}
	inPerson, video := slot("2030-02-04T09:00:00Z", "in-person"), slot("2030-02-04T10:00:00Z", "video")
	if err := move("2030-02-04T09:00:00Z", inPerson); sqlState(err) != "23514" {
		t.Errorf("a video appointment moves to an in-person slot: %v, want a check violation", err)
	}
	if err := move("2030-02-04T10:00:00Z", video); err != nil {
		t.Errorf("a video appointment moves to a video slot: %v", err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE appointment SET visit_mode = 'in-person' WHERE id = $1`, a.aFuture); err != nil {
		t.Fatal(err)
	}
	other := slot("2030-02-04T11:00:00Z", "video")
	if err := move("2030-02-04T11:00:00Z", other); sqlState(err) != "23514" {
		t.Errorf("an in-person appointment moves to a video slot: %v, want a check violation", err)
	}
	if err := move("2030-02-04T09:00:00Z", inPerson); err != nil {
		t.Errorf("an in-person appointment moves to an in-person slot: %v", err)
	}
}
