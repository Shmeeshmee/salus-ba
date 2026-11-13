package appointment_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
)

// reminderJob is how the fake jobs record a reminder's dispatch.
func reminderJob(n int, org string, due time.Time) string {
	return "reminder " + reminderID(n) + " " + org + " at " + due.Format(time.RFC3339)
}

// A booking schedules one reminder per offset of the clinic's settings at that moment, due the offset before the
// start, each with its dispatch job at its due time, in the booking's transaction.
func TestBook_SchedulesAReminderPerOffsetOfTheClinic(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Book(t.Context(), me, slotFree); err != nil {
		t.Fatal(err)
	}
	want := []appointment.Reminder{
		{ID: reminderID(1), AppointmentID: newID, OrganizationID: clinicA, PatientID: patientA, OffsetMinutes: 1440, StartAt: start, DueAt: start.Add(-24 * time.Hour), State: appointment.ReminderScheduled},
		{ID: reminderID(2), AppointmentID: newID, OrganizationID: clinicA, PatientID: patientA, OffsetMinutes: 60, StartAt: start, DueAt: start.Add(-time.Hour), State: appointment.ReminderScheduled},
	}
	if !slices.Equal(f.reminders.reminders, want) {
		t.Errorf("reminders %+v,\nwant %+v", f.reminders.reminders, want)
	}
	for _, job := range []string{reminderJob(1, clinicA, start.Add(-24*time.Hour)), reminderJob(2, clinicA, start.Add(-time.Hour))} {
		if !slices.Contains(f.jobs.enqueued, job) {
			t.Errorf("enqueued %v, want %q", f.jobs.enqueued, job)
		}
	}
}

// A reminder whose due time has passed is not created; a clinic without offsets gets none; the clinic's own offsets
// are taken as they are when the appointment is booked.
func TestBook_SchedulesOnlyTheRemindersStillAheadOfTheClinicsOffsets(t *testing.T) {
	for _, tt := range []struct {
		name    string
		slot    string
		offsets []int
		want    []int
	}{
		{name: "23 hours ahead, the defaults", slot: slotSoon, want: []int{60}},
		{name: "no reminders", slot: slotFree, offsets: []int{}},
		{name: "a week and 15 minutes", slot: slotFree, offsets: []int{10080, 15}, want: []int{15}}, // start is 6 days ahead
	} {
		f := newFixture()
		if tt.offsets != nil {
			s := organization.DefaultAppointmentSettings()
			s.ReminderOffsetsMinutes = tt.offsets
			f.rules.settings = map[string]organization.AppointmentSettings{clinicA: s}
		}
		if _, err := f.svc.Book(t.Context(), me, tt.slot); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		var got []int
		for _, r := range f.reminders.reminders {
			got = append(got, r.OffsetMinutes)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: offsets %v, want %v", tt.name, got, tt.want)
		}
	}
}

// A booking whose reminders cannot be scheduled books nothing: the rollback takes the appointment with it.
func TestBook_FailsWholeWhenItsRemindersCannotBeScheduled(t *testing.T) {
	f := newFixture()
	f.rules.err = errBackend
	if _, err := f.svc.Book(t.Context(), me, slotFree); !errors.Is(err, errBackend) {
		t.Fatalf("Book() error = %v, want the settings' failure", err)
	}
	if len(f.repo.appts) != 0 || len(f.reminders.reminders) != 0 {
		t.Errorf("stored %d appointments and %d reminders, want none", len(f.repo.appts), len(f.reminders.reminders))
	}
}

// scheduledFixture is patient A's appointment apptA at start, booked with its two default reminders.
func scheduledFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture()
	if _, err := f.svc.Book(t.Context(), me, slotFree); err != nil {
		t.Fatal(err)
	}
	f.audit.events, f.jobs.enqueued = nil, nil
	return f
}

func states(f *fixture) []appointment.ReminderState {
	var out []appointment.ReminderState
	for _, r := range f.reminders.reminders {
		out = append(out, r.State)
	}
	return out
}

// Every cancellation cancels the appointment's scheduled reminders in its transaction: the patient's, the staff's and
// the revocation's.
func TestCancel_CancelsTheScheduledReminders(t *testing.T) {
	for _, tt := range []struct {
		cancel func(f *fixture) error
		name   string
	}{
		{name: "the patient", cancel: func(f *fixture) error { _, err := f.svc.Cancel(t.Context(), me, newID); return err }},
		{name: "the admin", cancel: func(f *fixture) error { _, err := f.svc.Cancel(t.Context(), admin, newID); return err }},
		{name: "the revocation", cancel: func(f *fixture) error {
			f.consents.state[patientA+clinicA] = access.Consent{RevokedAt: now.Add(-time.Minute)}
			f.repo.appts[0].Created = now.Add(-time.Hour)
			_, err := f.svc.CancelForRevokedConsent(t.Context(), patientA, clinicA)
			return err
		}},
	} {
		f := scheduledFixture(t)
		if err := tt.cancel(f); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got, want := states(f), []appointment.ReminderState{appointment.ReminderCancelled, appointment.ReminderCancelled}; !slices.Equal(got, want) {
			t.Errorf("%s: reminders %v, want %v", tt.name, got, want)
		}
	}
}

// The dispatch job takes a scheduled reminder of a booked appointment at the start it was scheduled for: locked, read
// in the clinic's scope with the job's appointment set, audited with the job as agent. Taken again once dispatched,
// it is gone.
func TestTakeReminder_LocksAScheduledReminderOfItsBookedAppointment(t *testing.T) {
	f := scheduledFixture(t)
	got, err := f.svc.TakeReminder(t.Context(), "appointment_reminder", clinicA, reminderID(1))
	want := appointment.ReminderNotice{Start: start, ReminderID: reminderID(1), AppointmentID: newID, OrganizationID: clinicA, PatientID: patientA, AccountID: accountA}
	if err != nil || got != want {
		t.Fatalf("TakeReminder() = %+v, %v;\nwant %+v", got, err, want)
	}
	if !slices.Equal(f.reminders.locked, []string{reminderID(1)}) || f.repo.notice != newID || f.repo.clinic != clinicA {
		t.Errorf("locked %v with notice %q in clinic %q", f.reminders.locked, f.repo.notice, f.repo.clinic)
	}
	checkEvents(t, "TakeReminder", f.audit, jobEvent("appointment_reminder", clinicA, auditevent.CodeRead, "Appointment", newID))
	if err := f.svc.ReminderDispatched(t.Context(), clinicA, reminderID(1)); err != nil {
		t.Fatal(err)
	}
	if r := f.reminders.reminders[0]; r.State != appointment.ReminderDispatched || !r.DispatchedAt.Equal(now) {
		t.Errorf("reminder %+v, want dispatched now", r)
	}
	if _, err := f.svc.TakeReminder(t.Context(), "appointment_reminder", clinicA, reminderID(1)); !errors.Is(err, appointment.ErrReminderGone) {
		t.Errorf("taken twice: %v, want ErrReminderGone", err)
	}
}

// A reminder whose appointment was cancelled, moved to another start or has started, is skipped when its job runs; one
// of another clinic, or that does not exist, is not found.
func TestTakeReminder_SkipsAReminderWhoseAppointmentNoLongerHolds(t *testing.T) {
	for _, tt := range []struct {
		change func(f *fixture)
		name   string
		org    string
		want   error
		state  appointment.ReminderState
	}{
		{name: "cancelled", change: func(f *fixture) { f.repo.appts[0].Status = appointment.StatusCancelled }, org: clinicA, want: appointment.ErrReminderGone, state: appointment.ReminderSkipped},
		{name: "moved", change: func(f *fixture) { f.repo.appts[0].Start = start.Add(time.Hour) }, org: clinicA, want: appointment.ErrReminderGone, state: appointment.ReminderSkipped},
		{name: "started", change: func(f *fixture) { f.repo.appts[0].Start = now }, org: clinicA, want: appointment.ErrReminderGone, state: appointment.ReminderSkipped},
		{name: "another clinic", change: func(*fixture) {}, org: clinicB, want: appointment.ErrNotFound, state: appointment.ReminderScheduled},
	} {
		f := scheduledFixture(t)
		tt.change(f)
		if _, err := f.svc.TakeReminder(t.Context(), "appointment_reminder", tt.org, reminderID(1)); !errors.Is(err, tt.want) {
			t.Errorf("%s: %v, want %v", tt.name, err, tt.want)
		}
		if got := f.reminders.reminders[0].State; got != tt.state {
			t.Errorf("%s: reminder %s, want %s", tt.name, got, tt.state)
		}
	}
}

// A send job reads a dispatched reminder while its appointment still holds, audited; once the appointment is
// cancelled or moved, the reminder is gone.
func TestReminderNotice_GivesADispatchedReminderWhileItsAppointmentHolds(t *testing.T) {
	f := scheduledFixture(t)
	if _, err := f.svc.ReminderNotice(t.Context(), "reminder_email", clinicA, reminderID(1)); !errors.Is(err, appointment.ErrReminderGone) {
		t.Errorf("a reminder not dispatched yet: %v, want ErrReminderGone", err)
	}
	if _, err := f.svc.TakeReminder(t.Context(), "appointment_reminder", clinicA, reminderID(1)); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.ReminderDispatched(t.Context(), clinicA, reminderID(1)); err != nil {
		t.Fatal(err)
	}
	f.audit.events = nil
	got, err := f.svc.ReminderNotice(t.Context(), "reminder_email", clinicA, reminderID(1))
	if err != nil || got.AccountID != accountA || !got.Start.Equal(start) {
		t.Fatalf("ReminderNotice() = %+v, %v", got, err)
	}
	checkEvents(t, "ReminderNotice", f.audit, jobEvent("reminder_email", clinicA, auditevent.CodeRead, "Appointment", newID))
	f.repo.appts[0].Start = start.Add(24 * time.Hour)
	if _, err := f.svc.ReminderNotice(t.Context(), "reminder_email", clinicA, reminderID(1)); !errors.Is(err, appointment.ErrReminderGone) {
		t.Errorf("a moved appointment: %v, want ErrReminderGone", err)
	}
}

// The sweep enqueues again the dispatch of every reminder still scheduled past its due time whose start is ahead, in
// pages, and nothing else.
func TestSweepReminders_EnqueuesTheLostReminders(t *testing.T) {
	f := newFixture()
	soon := now.Add(30 * time.Minute)
	for i, r := range []appointment.Reminder{
		{OffsetMinutes: 60, StartAt: soon, DueAt: soon.Add(-time.Hour)},                   // lost
		{OffsetMinutes: 1440, StartAt: soon, DueAt: soon.Add(-24 * time.Hour)},            // lost
		{OffsetMinutes: 60, StartAt: start, DueAt: start.Add(-time.Hour)},                 // ahead
		{OffsetMinutes: 60, StartAt: now.Add(-time.Hour), DueAt: now.Add(-2 * time.Hour)}, // started
	} {
		r.ID, r.AppointmentID, r.OrganizationID, r.PatientID, r.State = reminderID(i+1), newID[:35]+string(rune('0'+i)), clinicA, patientA, appointment.ReminderScheduled
		f.reminders.reminders = append(f.reminders.reminders, r)
	}
	n, err := f.svc.SweepReminders(t.Context(), 1)
	if err != nil || n != 2 {
		t.Fatalf("SweepReminders() = %d, %v; want 2", n, err)
	}
	want := []string{reminderJob(2, clinicA, soon.Add(-24*time.Hour)), reminderJob(1, clinicA, soon.Add(-time.Hour))}
	if !slices.Equal(f.jobs.enqueued, want) {
		t.Errorf("enqueued %v,\nwant %v", f.jobs.enqueued, want)
	}
}

// A move from A to B and back to A (SAL-34's reschedule cancels the reminders, then schedules the new start's): A's
// reminders are scheduled again, and the first generation's, dispatched before the first move, stays dead.
func TestScheduleReminders_AMoveBackSchedulesTheStartAgainAndLeavesTheOldGenerationDead(t *testing.T) {
	f := scheduledFixture(t)
	if _, err := f.svc.TakeReminder(t.Context(), "appointment_reminder", clinicA, reminderID(1)); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.ReminderDispatched(t.Context(), clinicA, reminderID(1)); err != nil {
		t.Fatal(err)
	}
	// The move to B: the patient cancels (the reminders are cancelled as a reschedule cancels them), then books A again.
	if _, err := f.svc.Cancel(t.Context(), me, newID); err != nil {
		t.Fatal(err)
	}
	if got := states(f); !slices.Equal(got, []appointment.ReminderState{appointment.ReminderCancelled, appointment.ReminderCancelled}) {
		t.Fatalf("after the move: %v, want both cancelled, the dispatched one too", got)
	}
	if _, err := f.svc.ReminderNotice(t.Context(), "reminder_email", clinicA, reminderID(1)); !errors.Is(err, appointment.ErrReminderGone) {
		t.Errorf("a send of the first generation: %v, want ErrReminderGone", err)
	}
	f.repo.appts = nil
	f.slots.held = nil
	if _, err := f.svc.Book(t.Context(), me, slotFree); err != nil {
		t.Fatal(err)
	}
	if n := len(f.reminders.reminders); n != 4 || f.reminders.reminders[2].State != appointment.ReminderScheduled || f.reminders.reminders[3].State != appointment.ReminderScheduled {
		t.Errorf("reminders %+v, want the start's two scheduled anew", f.reminders.reminders)
	}
}
