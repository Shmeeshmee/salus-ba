package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The demo seed's SQL, on the owner's connection as every salusctl command. Everything a user does goes through the
// api (demo_http.go); these statements are what no endpoint and no service offers, each with its reason:
//
//   - demoState reads whether the database is fresh: the seed's refusal of any database but a fresh one, or one it
//     filled itself.
//   - demoRole, demoSchedule and demoFreeSlots read ids the seed needs and the api returns no endpoint for in this
//     shape (a practitioner's role, the free slots of a schedule ahead).
//   - demoMoveBack moves a booked visit four weeks back. The api books only ahead (appointment.Book: "a slot that is
//     not free and ahead"), and a demo needs a month of visits behind it. The owner's connection is the only one the
//     appointment's guard lets move a visit that is not its patient's own reschedule (00054 appointment_guard_change).
//   - demoEndTrial ends the clinic's 30-day trial before this month began: a new clinic is in its trial, and the demo
//     shows a member with its first invoice, which the monthly run issues only after the trial.

// demoState is what the database holds: every clinic, the demo clinic's id ("" when there is none), and whether the
// demo clinic has its membership invoice, the seed's last write.
func demoState(ctx context.Context, db *pgxpool.Pool) (clinics int, demo string, complete bool, err error) {
	err = db.QueryRow(ctx, `SELECT (SELECT count(*) FROM organization),
		coalesce((SELECT id::text FROM organization WHERE name = $1 ORDER BY created_at LIMIT 1), ''),
		EXISTS (SELECT 1 FROM subscription_invoice i JOIN organization o ON o.id = i.organization_id WHERE o.name = $1)`,
		demoClinicName).Scan(&clinics, &demo, &complete)
	return clinics, demo, complete, err
}

// demoPlanSet is whether the owner's default plan exists.
func demoPlanSet(ctx context.Context, db *pgxpool.Pool, code string) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM subscription_plan WHERE code = $1)`, code).Scan(&ok)
	return ok, err
}

// demoRole is the practitioner's one role at the clinic.
func demoRole(ctx context.Context, db *pgxpool.Pool, practitionerID string) (string, error) {
	var id string
	err := db.QueryRow(ctx, `SELECT id::text FROM practitioner_role WHERE practitioner_id = $1`, practitionerID).Scan(&id)
	return id, err
}

// demoSlot is a free slot ahead: its id, its period and its mode.
type demoSlot struct {
	start, end time.Time
	id, mode   string
}

// demoFreeSlots are the free slots of the schedule from now on, in order.
func demoFreeSlots(ctx context.Context, db *pgxpool.Pool, scheduleID string) ([]demoSlot, error) {
	rows, err := db.Query(ctx, `SELECT id::text, lower(period), upper(period), visit_mode FROM slot
		WHERE schedule_id = $1 AND status = 'free' AND lower(period) > now() ORDER BY lower(period)`, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []demoSlot
	for rows.Next() {
		var s demoSlot
		if err := rows.Scan(&s.id, &s.start, &s.end, &s.mode); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// demoMoveBack moves the appointment and its slot to start..end, together, in one transaction.
func demoMoveBack(ctx context.Context, db *pgxpool.Pool, appointmentID string, start, end time.Time) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE slot SET period = tstzrange($2, $3), updated_at = now()
		WHERE id = (SELECT slot_id FROM appointment WHERE id = $1)`, appointmentID, start, end)
	if err != nil {
		return fmt.Errorf("demo-seed: move a slot back: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("demo-seed: move a slot back: %d slots", tag.RowsAffected())
	}
	if _, err := tx.Exec(ctx, `UPDATE appointment SET period = tstzrange($2, $3) WHERE id = $1`, appointmentID, start, end); err != nil {
		return fmt.Errorf("demo-seed: move a visit back: %w", err)
	}
	return tx.Commit(ctx)
}

// demoEndTrial ends the clinic's trial a day before this month began, in the clinic's time zone.
func demoEndTrial(ctx context.Context, db *pgxpool.Pool, org string) error {
	_, err := db.Exec(ctx, `UPDATE clinic_subscription
		SET trial_ends_at = date_trunc('month', now() AT TIME ZONE o.time_zone) AT TIME ZONE o.time_zone - interval '1 day'
		FROM organization o WHERE o.id = clinic_subscription.organization_id AND o.id = $1`, org)
	return err
}
