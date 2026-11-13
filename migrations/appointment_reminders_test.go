package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// insertReminder writes, as the owner, a scheduled reminder of the appointment id offset minutes before its start
// and returns its id.
func insertReminder(t *testing.T, owner *pgxpool.Pool, id string, offset int) string {
	t.Helper()
	var reminder string
	err := owner.QueryRow(t.Context(), `INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at)
		SELECT id, organization_id, patient_id, $2, lower(period), lower(period) - make_interval(mins => $2) FROM appointment WHERE id = $1
		RETURNING id`, id, offset).Scan(&reminder)
	if err != nil {
		t.Fatal(err)
	}
	return reminder
}

// A reminder is seen by its patient and by its clinic (the dispatch job, the staff's cancellation); nothing without a
// scope, and not by another clinic or patient.
func TestAppointmentReminder_IsSeenByItsPatientAndItsClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	r := map[string]string{}
	for _, id := range []string{a.aFuture, a.bFuture, a.other} {
		r[id] = insertReminder(t, owner, id, 60)
	}
	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope"},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: sorted(r[a.aFuture], r[a.bFuture])},
		{name: "patient B with clinic A set", scope: map[string]string{"app.patient_id": p[1].patient, "app.organization_id": clinicA}, want: sorted(r[a.other])},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(r[a.aFuture], r[a.other])},
		{name: "clinic B", scope: map[string]string{"app.organization_id": clinicB}, want: sorted(r[a.bFuture])},
		{name: "an account with clinic A set", scope: map[string]string{"app.account_id": p[0].account, "app.organization_id": clinicA}},
	} {
		if got := idsIn(t, app, tt.scope, "SELECT id::text FROM appointment_reminder"); !slices.Equal(got, tt.want) {
			t.Errorf("%s: sees %v, want %v", tt.name, got, tt.want)
		}
	}
}

// A reminder holds its appointment's clinic, patient and start, is due its offset before that start, and moves from
// scheduled once, to dispatched (with its time), cancelled or skipped: never back, for every role.
func TestAppointmentReminder_IsDueItsOffsetBeforeTheStartAndLeavesScheduledOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, a := seedAppointments(t, owner)
	ctx := t.Context()
	for name, q := range map[string]string{
		"another clinic": `INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at)
			SELECT id, '` + clinicB + `', patient_id, 60, lower(period), lower(period) - interval '1 hour' FROM appointment WHERE id = '` + a.aFuture + `'`,
		"a due time off its offset": `INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at)
			SELECT id, organization_id, patient_id, 60, lower(period), lower(period) - interval '2 hours' FROM appointment WHERE id = '` + a.aFuture + `'`,
		"an offset under 15 minutes": `INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at)
			SELECT id, organization_id, patient_id, 10, lower(period), lower(period) - interval '10 minutes' FROM appointment WHERE id = '` + a.aFuture + `'`,
		"dispatched without its time": `INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at, state)
			SELECT id, organization_id, patient_id, 60, lower(period), lower(period) - interval '1 hour', 'dispatched' FROM appointment WHERE id = '` + a.aFuture + `'`,
	} {
		if _, err := owner.Exec(ctx, q); err == nil {
			t.Errorf("%s: stored", name)
		}
	}
	id := insertReminder(t, owner, a.aFuture, 60)
	if _, err := owner.Exec(ctx, `INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at)
		SELECT appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at FROM appointment_reminder WHERE id = $1`, id); err == nil {
		t.Error("a second reminder of the same appointment, offset and start was stored")
	}
	clinic := map[string]string{"app.organization_id": clinicA}
	err := asApp(t, app, clinic, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE appointment_reminder SET state = 'dispatched', dispatched_at = now() WHERE id = $1`, id)
		return err
	})
	if err != nil {
		t.Fatalf("the clinic's job dispatches the reminder: %v", err)
	}
	for _, q := range []string{
		`UPDATE appointment_reminder SET state = 'scheduled', dispatched_at = NULL WHERE id = '` + id + `'`,
		`UPDATE appointment_reminder SET state = 'cancelled', dispatched_at = NULL WHERE id = '` + id + `'`,
	} {
		if _, err := owner.Exec(ctx, q); err == nil {
			t.Errorf("a dispatched reminder changed: %s", q)
		}
	}
	other := insertReminder(t, owner, a.aFuture, 1440)
	err = asApp(t, app, map[string]string{"app.patient_id": p[0].patient, "app.organization_id": clinicA}, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE appointment_reminder SET state = 'cancelled' WHERE id = $1`, other)
		return err
	})
	if err != nil {
		t.Errorf("the patient's cancellation cancels the reminder: %v", err)
	}
}

// The sweep finds the scheduled reminders past their due time whose start is still ahead, of every clinic, as ids and
// times; salus_app reads them without a scope through the view only.
func TestAppointmentReminderScheduled_GivesTheSweepTheScheduledRemindersOfEveryClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, _, a := seedAppointments(t, owner)
	ctx := t.Context()
	due := insertReminder(t, owner, a.aFuture, 60)
	dueB := insertReminder(t, owner, a.other, 60)
	dispatched := insertReminder(t, owner, a.aFuture, 1440)
	if _, err := owner.Exec(ctx, `UPDATE appointment_reminder SET state = 'dispatched', dispatched_at = now() WHERE id = $1`, dispatched); err != nil {
		t.Fatal(err)
	}
	var at string
	if err := owner.QueryRow(ctx, `SELECT (lower(period) - interval '30 minutes')::text FROM appointment WHERE id = $1`, a.aFuture).Scan(&at); err != nil {
		t.Fatal(err)
	}
	got := idsIn(t, app, nil, `SELECT id::text FROM appointment_reminder_scheduled WHERE due_at <= '`+at+`' AND start_at > '`+at+`' AND id IN ('`+due+`', '`+dispatched+`')`)
	if !slices.Equal(got, sorted(due)) {
		t.Errorf("overdue at %s: %v, want %v", at, got, []string{due})
	}
	if got := idsIn(t, app, nil, `SELECT id::text FROM appointment_reminder_scheduled WHERE id = '`+dueB+`'`); len(got) != 1 {
		t.Errorf("clinic A's other patient's reminder not found by the sweep: %v", got)
	}
	if got := idsIn(t, app, nil, `SELECT id::text FROM appointment_reminder`); len(got) != 0 {
		t.Errorf("without a scope the app sees %d reminders in the table", len(got))
	}
}

// The deliveries and preferences are the account's, named by their ids (as push_device): no address, number or text
// column exists; a delivery is once per source, channel and device, and leaves pending once. A clinic's SMS usage is
// seen in its own scope only.
func TestNotificationDelivery_OncePerSourceChannelAndDevice(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, _ := seedAppointments(t, owner)
	ctx := t.Context()
	insert := func(channel, device string) error {
		return asApp(t, app, nil, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, `INSERT INTO notification_delivery (kind, source_id, organization_id, account_id, channel, device_id)
				VALUES ('reminder', 'c0000000-0000-4000-8000-00000000000c', $1, $2, $3, NULLIF($4, '')::uuid)`, clinicA, p[0].account, channel, device)
			return err
		})
	}
	for _, tt := range []struct {
		channel, device string
		ok              bool
	}{
		{"email", "", true}, {"email", "", false}, {"sms", "", true},
		{"push", "d0000000-0000-4000-8000-00000000000d", true}, {"push", "d0000000-0000-4000-8000-00000000000d", false},
		{"push", "e0000000-0000-4000-8000-00000000000e", true}, {"fax", "", false},
	} {
		if err := insert(tt.channel, tt.device); (err == nil) != tt.ok {
			t.Errorf("%s %q: err %v, want stored %v", tt.channel, tt.device, err, tt.ok)
		}
	}
	err := asApp(t, app, nil, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE notification_delivery SET state = 'sent', sent_at = now(), segments = 1 WHERE channel = 'sms'`)
		return err
	})
	if err != nil {
		t.Fatalf("the send marks its delivery sent: %v", err)
	}
	if _, err := owner.Exec(ctx, `UPDATE notification_delivery SET state = 'failed' WHERE channel = 'sms'`); err == nil {
		t.Error("a sent delivery changed")
	}
	var columns []string
	rows, err := owner.Query(ctx, `SELECT column_name::text FROM information_schema.columns
		WHERE table_name IN ('notification_delivery', 'notification_preference', 'appointment_reminder') ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, c)
	}
	rows.Close()
	for _, c := range []string{"phone", "token", "body", "text", "address", "number", "recipient"} {
		if slices.Contains(columns, c) {
			t.Errorf("a column %q", c)
		}
	}
	if _, err := owner.Exec(ctx, `INSERT INTO sms_usage (organization_id, month, messages, segments) VALUES ($1, '2026-11-01', 2, 3), ($2, '2026-11-01', 1, 1)`,
		clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO sms_usage (organization_id, month, messages, segments) VALUES ($1, '2026-11-02', 1, 1)`, clinicA); err == nil {
		t.Error("a month not on its first day was stored")
	}
	if got := idsIn(t, app, map[string]string{"app.organization_id": clinicA}, "SELECT organization_id::text FROM sms_usage"); !slices.Equal(got, []string{clinicA}) {
		t.Errorf("clinic A sees the usage of %v", got)
	}
}

// The patient's preferences: quiet hours both or neither, a zone name of at most 64 characters, deleted with the
// account.
func TestNotificationPreference_QuietHoursBothOrNeitherDeletedWithTheAccount(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, _ := seedAppointments(t, owner)
	ctx := t.Context()
	for name, q := range map[string]string{
		"one end of the quiet hours": `INSERT INTO notification_preference (account_id, push, email, sms, quiet_start) VALUES ('` + p[0].account + `', true, true, false, '22:00')`,
		"a zone too long":            `INSERT INTO notification_preference (account_id, push, email, sms, time_zone) VALUES ('` + p[0].account + `', true, true, false, repeat('x', 65))`,
	} {
		if _, err := owner.Exec(ctx, q); err == nil {
			t.Errorf("%s: stored", name)
		}
	}
	err := asApp(t, app, nil, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `INSERT INTO notification_preference (account_id, push, email, sms, quiet_start, quiet_end, time_zone)
			VALUES ($1, true, false, true, '23:00', '07:00', 'Asia/Dubai')
			ON CONFLICT (account_id) DO UPDATE SET push = excluded.push, updated_at = now()`, p[0].account)
		return err
	})
	if err != nil {
		t.Fatalf("the app stores the patient's preferences: %v", err)
	}
	var cascade bool
	if err := owner.QueryRow(ctx, `SELECT confdeltype = 'c' FROM pg_constraint
		WHERE conrelid = 'notification_preference'::regclass AND contype = 'f'`).Scan(&cascade); err != nil || !cascade {
		t.Errorf("the preferences are not deleted with the account: %v", err)
	}
}
