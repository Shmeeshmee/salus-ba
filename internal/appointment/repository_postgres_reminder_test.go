package appointment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A booking's reminders are stored in the patient's scope once per offset and start, locked, dispatched or cancelled
// in the clinic's, and leave scheduled once; the sweep finds the lost ones of every clinic without a scope, and the
// retention role deletes them after their period.
func TestReminders_ScheduledOnceDispatchedOnceSweptAndPurged(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seedDB(t, owner)
	reminders := appointment.NewPostgresReminderRepository(app)
	var a appointment.Appointment
	var r60, r1440 appointment.Reminder
	err := inScope(t, app, asPatient(s.patientA), func(ctx context.Context, r appointment.Repository) error {
		var err error
		if a, err = r.Create(ctx, newAppt(s, s.patientA, start)); err != nil {
			return err
		}
		for _, x := range []struct {
			out    *appointment.Reminder
			offset int
		}{{&r60, 60}, {&r1440, 1440}} {
			var ok bool
			*x.out, ok, err = reminders.AddReminder(ctx, appointment.Reminder{
				StartAt: a.Start, DueAt: a.Start.Add(-time.Duration(x.offset) * time.Minute), AppointmentID: a.ID,
				OrganizationID: clinicA, PatientID: s.patientA, OffsetMinutes: x.offset,
			})
			if err != nil || !ok {
				return errors.Join(err, errors.New("not stored"))
			}
		}
		_, ok, err := reminders.AddReminder(ctx, r60)
		if ok || err != nil {
			t.Errorf("the same reminder again: stored %v, %v", ok, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = inScope(t, app, asClinic, func(ctx context.Context, _ appointment.Repository) error {
		got, err := reminders.LockReminder(ctx, r60.ID)
		if err != nil {
			return err
		}
		if got.AppointmentID != a.ID || got.State != appointment.ReminderScheduled || !got.DueAt.Equal(start.Add(-time.Hour)) || got.OffsetMinutes != 60 {
			t.Errorf("LockReminder() = %+v", got)
		}
		return reminders.SetReminderState(ctx, r60.ID, appointment.ReminderDispatched, now)
	})
	if err != nil {
		t.Fatal(err)
	}
	checkSweptCancelledPurged(t, owner, app, a, r60, r1440)
}

// checkSweptCancelledPurged: the sweep finds the day-before reminder only, the patient's cancellation cancels it, and
// the retention role deletes the reminders past their period.
func checkSweptCancelledPurged(t *testing.T, owner, app *pgxpool.Pool, a appointment.Appointment, r60, r1440 appointment.Reminder) {
	t.Helper()
	reminders := appointment.NewPostgresReminderRepository(app)
	overdue := func(at time.Time) []appointment.Reminder {
		var out []appointment.Reminder
		if err := inScope(t, app, func(context.Context, appointment.Repository) error { return nil }, func(ctx context.Context, _ appointment.Repository) error {
			var err error
			out, err = reminders.OverdueReminders(ctx, at, time.Time{}, "00000000-0000-0000-0000-000000000000", 10)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return out
	}
	if got := overdue(start.Add(-30 * time.Minute)); len(got) != 1 || got[0].ID != r1440.ID || got[0].OrganizationID != clinicA {
		t.Errorf("overdue half an hour before the start: %+v, want the day-before reminder only", got)
	}
	if got := overdue(start); len(got) != 0 {
		t.Errorf("overdue at the start: %+v, want none", got)
	}
	err := inScope(t, app, asPatient(a.PatientID), func(ctx context.Context, _ appointment.Repository) error {
		n, err := reminders.CancelReminders(ctx, a.ID)
		if n != 1 {
			t.Errorf("cancelled %d, want the one still scheduled", n)
		}
		if err := reminders.SetReminderState(ctx, r60.ID, appointment.ReminderSkipped, now); !errors.Is(err, appointment.ErrReminderGone) {
			t.Errorf("a dispatched reminder skipped: %v, want ErrReminderGone", err)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	err = inScope(t, app, asClinic, func(ctx context.Context, _ appointment.Repository) error {
		got, err := reminders.Reminder(ctx, r1440.ID)
		if got.State != appointment.ReminderCancelled {
			t.Errorf("Reminder() = %+v, %v; want cancelled", got, err)
		}
		_, err = reminders.Reminder(ctx, "not-a-uuid")
		if !errors.Is(err, appointment.ErrNotFound) {
			t.Errorf("a malformed id: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	purged, err := appointment.NewReminderPurger(appointment.NewPostgresReminderRepository(owner)).Purge(t.Context(), start.Add(-time.Hour))
	if err != nil || purged != 1 {
		t.Errorf("Purge() = %d, %v; want the day-before reminder", purged, err)
	}
}
