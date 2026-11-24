package appointment_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

// moveFixture is the staff fixture with a, the free slots of the reschedule's tests (role A1 and A2 a day after
// start) and a's own slot.
func moveFixture(a appointment.Appointment) *fixture {
	f := staffFixture(a)
	f.slots.bookable[slotLater] = slotAt(slotLater, roleA1, clinicA, start.Add(24*time.Hour))
	f.slots.bookable[slotOther] = slotAt(slotOther, roleA2, clinicA, start.Add(24*time.Hour))
	f.slots.bookable[a.SlotID] = slotAt(a.SlotID, a.PractitionerRoleID, a.OrganizationID, a.Start)
	f.slots.held = append(f.slots.held, a.SlotID) // busy with the appointment
	return f
}

// The desk (admin and receptionist) moves a visit to any free slot of the clinic of its visit type, another
// practitioner's too (docs/clinic-cms WAVE4 V4): the patient's reschedule count and confirmation stay, the history says
// staff with the account, the reminders follow the new start and the patient is told.
func TestMoveForPatient_MovesToAnotherPractitionersSlotAsStaffWithoutCountingOrConfirming(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]appointment.Caller{"admin": admin, "receptionist": desk} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := booked(apptA, patientA, roleA1, clinicA, start)
			f := moveFixture(a)
			f.reminders.reminders = append(f.reminders.reminders, appointment.Reminder{
				ID: oldReminder, AppointmentID: apptA, OrganizationID: clinicA, PatientID: patientA, OffsetMinutes: 1440,
				StartAt: start, DueAt: start.Add(-24 * time.Hour), State: appointment.ReminderScheduled,
			})
			newStart := start.Add(24 * time.Hour)
			got, err := f.svc.MoveForPatient(t.Context(), c, apptA, slotOther, 1)
			if err != nil || got.ID != apptA || got.PractitionerRoleID != roleA2 || got.SlotID != slotOther || !got.Start.Equal(newStart) ||
				got.RescheduleCount != 0 || !got.ConfirmedAt.IsZero() || got.VersionID != 2 || got.Status != appointment.StatusBooked {
				t.Fatalf("MoveForPatient = %+v, %v; want role A2's slot, uncounted and unconfirmed, at version 2", got, err)
			}
			if !slices.Contains(f.slots.held, slotOther) || !slices.Equal(f.slots.released, []string{clinicA + "/" + a.SlotID}) {
				t.Errorf("held %v, released %v; want the new slot held and the old one freed", f.slots.held, f.slots.released)
			}
			checkMovedReminders(t, f, newStart)
			if n := len(f.jobs.enqueued); n != 4 || f.jobs.enqueued[0] != "slot_freed "+a.SlotID+" "+clinicA || f.jobs.enqueued[n-1] != "rescheduled "+apptA+" "+clinicA {
				t.Errorf("enqueued %v, want the freed slot, the reminders and the e-mail with the new time", f.jobs.enqueued)
			}
			if want := []string{clinicA + " " + start.Format(time.RFC3339), clinicA + " " + newStart.Format(time.RFC3339)}; !slices.Equal(f.stats.refreshes, want) {
				t.Errorf("refreshes %v, want both days %v", f.stats.refreshes, want)
			}
			if len(f.repo.changes) != 1 {
				t.Fatalf("history %+v, want the move only", f.repo.changes)
			}
			if m := f.repo.changes[0]; m.Kind != appointment.ChangeRescheduled || m.Via != appointment.ViaStaff || m.ByAccountID != c.AccountID ||
				!m.FromStart.Equal(start) || !m.ToStart.Equal(newStart) || m.FromSlotID != a.SlotID || m.ToSlotID != slotOther {
				t.Errorf("history %+v, want the move via staff by %s", m, c.AccountID)
			}
			if f.repo.patient != "" {
				t.Errorf("moved in patient %q's scope, want the clinic's", f.repo.patient)
			}
			checkEvents(t, "MoveForPatient", f.audit, event(c, auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess))
		})
	}
}

// checkMovedReminders: the old start's reminder is cancelled and the clinic's offsets are scheduled for newStart.
func checkMovedReminders(t *testing.T, f *fixture, newStart time.Time) {
	t.Helper()
	var due []string
	for _, r := range f.reminders.reminders {
		switch {
		case r.ID == oldReminder && r.State != appointment.ReminderCancelled:
			t.Errorf("the old start's reminder is %s, want cancelled", r.State)
		case r.State == appointment.ReminderScheduled && r.StartAt.Equal(newStart):
			due = append(due, r.DueAt.Format(time.RFC3339))
		}
	}
	if want := []string{newStart.Add(-24 * time.Hour).Format(time.RFC3339), newStart.Add(-time.Hour).Format(time.RFC3339)}; !slices.Equal(due, want) {
		t.Errorf("reminders due %v, want %v", due, want)
	}
}

// A patient the clinic registered has no account: no e-mail and no reminder is scheduled for them (as BookForPatient).
func TestMoveForPatient_SendsAPatientWithoutAnAccountNothing(t *testing.T) {
	t.Parallel()
	f := moveFixture(booked(apptA, patientC, roleA1, clinicA, start))
	if _, err := f.svc.MoveForPatient(t.Context(), desk, apptA, slotLater, 1); err != nil {
		t.Fatalf("MoveForPatient = %v", err)
	}
	if len(f.reminders.reminders) != 0 || !slices.Equal(f.jobs.enqueued, []string{"slot_freed 51" + apptA[2:] + " " + clinicA}) || len(f.stats.refreshes) != 2 {
		t.Errorf("reminders %v, enqueued %v, refreshes %v; want only the freed slot and the statistics", f.reminders.reminders, f.jobs.enqueued, f.stats.refreshes)
	}
}

// The desk moves a booked in-person visit whose time has come (the patient is late) to a later slot; a video visit
// stops moving when its join window opens, so the window always follows the start it was given.
func TestMoveForPatient_AnInPersonVisitMovesUntilItArrivesAVideoOneUntilItsJoinWindow(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		at   time.Time
		want error
		mode schedule.VisitMode
	}{
		"in person, started":              {mode: schedule.ModeInPerson, at: now.Add(-10 * time.Minute)},
		"video, before its window":        {mode: schedule.ModeVideo, at: now.Add(11 * time.Minute)},
		"video, its window open":          {mode: schedule.ModeVideo, at: now.Add(9 * time.Minute), want: appointment.ErrNotReschedulable},
		"video, before the clinic system": {at: now.Add(9 * time.Minute), want: appointment.ErrNotReschedulable},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := booked(apptA, patientA, roleA1, clinicA, tc.at)
			a.Mode = tc.mode
			f := moveFixture(a)
			later := f.slots.bookable[slotLater]
			later.Mode = tc.mode
			f.slots.bookable[slotLater] = later
			if _, err := f.svc.MoveForPatient(t.Context(), desk, apptA, slotLater, 1); !errors.Is(err, tc.want) {
				t.Fatalf("MoveForPatient = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestMoveForPatient_RefusesAndAuditsEachCase(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want    error
		consent *access.Consent
		slot    string
		status  appointment.Status
		mode    schedule.VisitMode // of the new slot
		caller  appointment.Caller
		version int32
	}{
		"a patient":             {caller: me, slot: slotLater, want: appointment.ErrForbidden},
		"a practitioner":        {caller: pract, slot: slotLater, want: appointment.ErrForbidden},
		"another clinic's desk": {caller: adminB, slot: slotB, want: appointment.ErrNotFound},
		"an unknown slot":       {caller: desk, slot: newID, want: appointment.ErrSlotUnavailable},
		"a past slot":           {caller: desk, slot: slotPast, want: appointment.ErrSlotUnavailable},
		"another clinic's slot": {caller: desk, slot: slotB, want: appointment.ErrSlotUnavailable},
		"its own slot":          {caller: desk, slot: "51" + apptA[2:], want: appointment.ErrSlotUnavailable},
		"an inactive role":      {caller: desk, slot: slotAx, want: appointment.ErrSlotUnavailable},
		"the other visit type":  {caller: desk, slot: slotLater, mode: schedule.ModeInPerson, want: appointment.ErrNotReschedulable},
		"a stale version":       {caller: desk, slot: slotLater, version: 2, want: appointment.ErrVersionConflict},
		"arrived":               {caller: desk, slot: slotLater, status: appointment.StatusArrived, want: appointment.ErrClosed},
		"fulfilled":             {caller: desk, slot: slotLater, status: appointment.StatusFulfilled, want: appointment.ErrClosed},
		"cancelled":             {caller: desk, slot: slotLater, status: appointment.StatusCancelled, want: appointment.ErrClosed},
		"a no-show":             {caller: desk, slot: slotLater, status: appointment.StatusNoShow, want: appointment.ErrClosed},
		// Without the patient's active consent the clinic's scope does not show a future visit.
		"consent revoked": {caller: desk, slot: slotLater, consent: &access.Consent{RevokedAt: revokedAt}, want: appointment.ErrNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := booked(apptA, patientA, roleA1, clinicA, start)
			if tc.status != "" {
				a.Status = tc.status
			}
			f := moveFixture(a)
			if tc.mode != "" {
				later := f.slots.bookable[tc.slot]
				later.Mode = tc.mode
				f.slots.bookable[tc.slot] = later
			}
			if tc.consent != nil {
				f.consents.state[patientA+clinicA] = *tc.consent
			}
			version := tc.version
			if version == 0 {
				version = 1
			}
			if _, err := f.svc.MoveForPatient(t.Context(), tc.caller, apptA, tc.slot, version); !errors.Is(err, tc.want) {
				t.Fatalf("MoveForPatient = %v, want %v", err, tc.want)
			}
			if got := f.repo.appts[0]; got != a {
				t.Errorf("appointment %+v, want it unchanged", got)
			}
			if len(f.repo.changes) != 0 || len(f.jobs.enqueued) != 0 || len(f.slots.released) != 0 || !slices.Equal(f.slots.held, []string{a.SlotID}) {
				t.Errorf("history %v, jobs %v, released %v, held %v; want nothing", f.repo.changes, f.jobs.enqueued, f.slots.released, f.slots.held)
			}
			checkEvents(t, "MoveForPatient", f.audit, event(tc.caller, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure))
		})
	}
}

// Another visit of the new slot's practitioner at that time (an overlapping schedule) refuses the move, and the hold
// is rolled back: the exclusion constraint guards it as it guards the patient's reschedule.
func TestMoveForPatient_ATimeTheRoleIsBookedAtIsRefusedAndTheHoldRolledBack(t *testing.T) {
	t.Parallel()
	f := moveFixture(booked(apptA, patientA, roleA1, clinicA, start))
	f.repo.appts = append(f.repo.appts, booked(apptB, patientB, roleA2, clinicA, start.Add(24*time.Hour)))
	f.consents.state[patientB+clinicA] = access.Consent{Active: true}
	if _, err := f.svc.MoveForPatient(t.Context(), desk, apptA, slotOther, 1); !errors.Is(err, appointment.ErrSlotUnavailable) {
		t.Fatalf("MoveForPatient = %v, want ErrSlotUnavailable", err)
	}
	if slices.Contains(f.slots.held, slotOther) || f.repo.appts[0].SlotID != "51"+apptA[2:] {
		t.Errorf("held %v, appointment %+v; want the hold rolled back and the visit where it was", f.slots.held, f.repo.appts[0])
	}
	checkEvents(t, "MoveForPatient", f.audit, event(desk, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure))
}

func TestMoveForPatient_ADatabaseFailureIsNotARefusal(t *testing.T) {
	t.Parallel()
	f := moveFixture(booked(apptA, patientA, roleA1, clinicA, start))
	f.repo.err = errBackend
	if _, err := f.svc.MoveForPatient(t.Context(), desk, apptA, slotLater, 1); !errors.Is(err, errBackend) {
		t.Fatalf("MoveForPatient = %v, want the backend's error", err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %v, want nothing (rolled back)", f.audit.events)
	}
}
