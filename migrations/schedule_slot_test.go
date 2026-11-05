package migrations_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// scheduled is one clinic's schedule and a free and a busy slot in it, starting in a day.
type scheduled struct {
	schedule, free, busy string
}

// seedSchedules creates, as the owner, clinics A and B (seedDirectory) with a schedule for their role and two
// slots each: one free, one busy.
func seedSchedules(t *testing.T, owner *pgxpool.Pool) map[string]scheduled {
	t.Helper()
	seeded := map[string]scheduled{}
	for org, d := range seedDirectory(t, owner) {
		var s scheduled
		err := owner.QueryRow(t.Context(), `WITH s AS (
				INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
				VALUES ($1, $2, 30, '[{"day": 1, "start": "09:00", "end": "12:00"}]') RETURNING id, organization_id),
			f AS (INSERT INTO slot (schedule_id, organization_id, period, status)
				SELECT id, organization_id, tstzrange(now() + interval '1 day', now() + interval '1 day 30 minutes'), 'free' FROM s
				RETURNING id),
			b AS (INSERT INTO slot (schedule_id, organization_id, period, status)
				SELECT id, organization_id, tstzrange(now() + interval '1 day 30 minutes', now() + interval '1 day 60 minutes'), 'busy' FROM s
				RETURNING id)
			SELECT (SELECT id FROM s), (SELECT id FROM f), (SELECT id FROM b)`, org, d.role).Scan(&s.schedule, &s.free, &s.busy)
		if err != nil {
			t.Fatal(err)
		}
		seeded[org] = s
	}
	return seeded
}

func TestScheduleSlot_TheClinicScopeSeesOnlyItsOwnRows(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedSchedules(t, owner)

	for _, table := range []string{"schedule", "slot"} {
		for org, want := range map[string]int{"": 0, clinicA: map[string]int{"schedule": 1, "slot": 2}[table]} {
			var n, other int
			err := inClinic(t, app, org, func(ctx context.Context, db postgres.Querier) error {
				return db.QueryRow(ctx, "SELECT count(*), count(*) FILTER (WHERE organization_id <> $1) FROM "+table, clinicA).Scan(&n, &other)
			})
			if err != nil || n != want || other != 0 {
				t.Errorf("%s with clinic %q: %d rows, %d of another clinic, %v; want %d and 0", table, org, n, other, err, want)
			}
		}
	}
}

func TestScheduleSlot_TheDirectoryScopeSeesTheScheduleAndFreeSlotsOfListedClinicsOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seeded := seedSchedules(t, owner)
	if _, err := owner.Exec(t.Context(), "UPDATE organization SET listed = true WHERE id = $1", clinicA); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"schedule": seeded[clinicA].schedule, "slot": seeded[clinicA].free}
	for table, id := range want {
		var ids []string
		err := inDirectory(t, app, func(ctx context.Context, db postgres.Querier) error {
			return db.QueryRow(ctx, "SELECT coalesce(array_agg(id::text), '{}') FROM "+table).Scan(&ids)
		})
		if err != nil || len(ids) != 1 || ids[0] != id {
			t.Errorf("%s in the directory scope: %v, %v; want only %s", table, ids, err, id)
		}
	}
	for _, stmt := range []string{
		"UPDATE slot SET status = 'busy'",
		"DELETE FROM slot",
		"UPDATE schedule SET slot_minutes = 60",
	} {
		var n int64
		err := inDirectory(t, app, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, stmt)
			n = tag.RowsAffected()
			return err
		})
		if err != nil || n != 0 {
			t.Errorf("%s in the directory scope: %d rows, %v; want 0 rows", stmt, n, err)
		}
	}
}

func TestScheduleSlot_OverlappingSlotsOfOneScheduleAreRejected(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	seeded := seedSchedules(t, owner)
	ctx := t.Context()

	const insert = `INSERT INTO slot (schedule_id, organization_id, period, status)
		VALUES ($1, $2, tstzrange(now() + interval '1 day 15 minutes', now() + interval '1 day 45 minutes'), 'free')`
	if _, err := owner.Exec(ctx, insert, seeded[clinicA].schedule, clinicA); sqlState(err) != "23P01" {
		t.Errorf("an overlapping slot: %v, want exclusion_violation (23P01)", err)
	}
	// Back to back is not an overlap: the ranges are half-open.
	if _, err := owner.Exec(ctx, `INSERT INTO slot (schedule_id, organization_id, period, status)
		VALUES ($1, $2, tstzrange(now() + interval '1 day 60 minutes', now() + interval '1 day 90 minutes'), 'free')`,
		seeded[clinicA].schedule, clinicA); err != nil {
		t.Errorf("a slot right after another: %v", err)
	}
	// A slot is in its schedule's clinic.
	if _, err := owner.Exec(ctx, `INSERT INTO slot (schedule_id, organization_id, period, status)
		VALUES ($1, $2, tstzrange(now() + interval '3 days', now() + interval '3 days 30 minutes'), 'free')`,
		seeded[clinicA].schedule, clinicB); sqlState(err) != "23503" {
		t.Errorf("a slot in another clinic than its schedule: %v, want foreign_key_violation (23503)", err)
	}
}

func TestScheduleSlot_AScheduleIsOnePerRoleInTheRolesClinicWithAValidLength(t *testing.T) {
	t.Parallel()
	owner := pgtest.NewDatabase(t)
	d := seedDirectory(t, owner)
	const insert = `INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) VALUES ($1, $2, $3)`

	tests := []struct {
		name, org, role, wantErr string
		minutes                  int
	}{
		{name: "too short", org: clinicA, role: d[clinicA].role, minutes: 0, wantErr: "23514"},
		{name: "not a multiple of 5", org: clinicA, role: d[clinicA].role, minutes: 32, wantErr: "23514"},
		{name: "too long", org: clinicA, role: d[clinicA].role, minutes: 245, wantErr: "23514"},
		{name: "another clinic's role", org: clinicA, role: d[clinicB].role, minutes: 30, wantErr: "23503"},
		{name: "the first", org: clinicA, role: d[clinicA].role, minutes: 30},
		{name: "a second for the role", org: clinicA, role: d[clinicA].role, minutes: 30, wantErr: "23505"},
	}
	for _, tt := range tests {
		_, err := owner.Exec(t.Context(), insert, tt.org, tt.role, tt.minutes)
		if got := sqlState(err); got != tt.wantErr {
			t.Errorf("%s: %v, want %q", tt.name, err, tt.wantErr)
		}
	}
	if _, err := owner.Exec(t.Context(), `UPDATE schedule SET availability = '{}'`); sqlState(err) != "23514" {
		t.Errorf("availability that is not an array: %v, want check_violation", err)
	}
}

func TestScheduleSlot_TheAppRoleWritesOnlyWhatTheApplicationNeeds(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seeded := seedSchedules(t, owner)
	allowed := []string{
		"UPDATE schedule SET slot_minutes = 60, availability = '[]', version_id = version_id + 1, updated_at = now()",
		"UPDATE slot SET status = 'free', version_id = version_id + 1, updated_at = now() WHERE id = '" + seeded[clinicA].busy + "'",
		"DELETE FROM slot WHERE id = '" + seeded[clinicA].free + "'",
		`INSERT INTO slot (schedule_id, organization_id, period)
			VALUES ('` + seeded[clinicA].schedule + `', '` + clinicA + `', tstzrange(now() + interval '5 days', now() + interval '5 days 30 minutes'))`,
	}
	for _, stmt := range allowed {
		var n int64
		err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, stmt)
			n = tag.RowsAffected()
			return err
		})
		if err != nil || n != 1 {
			t.Errorf("%s as %s in clinic A: %d rows, %v; want 1", stmt, pgtest.AppRole, n, err)
		}
	}
	for _, stmt := range []string{
		"UPDATE schedule SET practitioner_role_id = practitioner_role_id",
		"UPDATE slot SET period = period",
		"DELETE FROM schedule",
		"TRUNCATE slot",
	} {
		err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, stmt)
			return err
		})
		if sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege (42501)", stmt, pgtest.AppRole, err)
		}
	}
}

// Generation deletes free slots only: a busy slot stays with its appointment, whatever the application asks.
func TestScheduleSlot_TheAppRoleCannotDeleteABusySlot(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seeded := seedSchedules(t, owner)

	var n int64
	err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
		tag, err := db.Exec(ctx, "DELETE FROM slot WHERE id = $1", seeded[clinicA].busy)
		n = tag.RowsAffected()
		return err
	})
	if err != nil || n != 0 {
		t.Errorf("deleting a busy slot: %d rows, %v; want 0", n, err)
	}
}

func TestScheduleSlot_TheRetentionRoleDeletesFreeSlotsOfEveryClinic(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	seedSchedules(t, owner)

	tag, err := ret.Exec(t.Context(), "DELETE FROM slot WHERE status = 'free'")
	if err != nil || tag.RowsAffected() != 2 {
		t.Errorf("as %s: %d rows, %v; want the free slot of each clinic", pgtest.RetentionRole, tag.RowsAffected(), err)
	}
	if _, err := ret.Exec(t.Context(), "SELECT count(*) FROM schedule"); sqlState(err) != "42501" {
		t.Errorf("reading schedule as %s: %v, want insufficient_privilege", pgtest.RetentionRole, err)
	}
}
