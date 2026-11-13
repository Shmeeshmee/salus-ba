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

// Slots of role A1 in clinic A for the reschedule's tests.
const (
	slotLater = "52000000-0000-4000-8000-000000000001" // start + 1 day
	slotOther = "52000000-0000-4000-8000-000000000002" // role A2, start + 1 day
)

func rescheduleFixture(a appointment.Appointment) *selfFixture {
	f := newSelfFixture(a)
	f.slots.bookable[slotLater] = slotAt(slotLater, roleA1, clinicA, start.Add(24*time.Hour))
	f.slots.bookable[slotOther] = slotAt(slotOther, roleA2, clinicA, start.Add(24*time.Hour))
	f.slots.bookable[a.SlotID] = slotAt(a.SlotID, a.PractitionerRoleID, a.OrganizationID, a.Start)
	return f
}

func TestReschedule_MovesTheSameAppointmentToAFreeSlotOfItsRole(t *testing.T) {
	a := booked(apptA, patientA, roleA1, clinicA, start)
	f := rescheduleFixture(a)
	got, err := f.svc.Reschedule(t.Context(), me, apptA, slotLater)
	newStart := start.Add(24 * time.Hour)
	if err != nil || got.ID != apptA || !got.Start.Equal(newStart) || got.SlotID != slotLater || got.RescheduleCount != 1 || got.VersionID != 2 ||
		got.Status != appointment.StatusBooked || !got.ConfirmedAt.Equal(now) || got.ConfirmedVia != appointment.ViaApp {
		t.Fatalf("Reschedule() = %+v, %v; want the same appointment at the new slot, counted once and confirmed in the app", got, err)
	}
	if want := []string{slotLater}; !slices.Equal(f.slots.held, want) {
		t.Errorf("held %v, want %v", f.slots.held, want)
	}
	if want := []string{clinicA + "/" + a.SlotID}; !slices.Equal(f.slots.released, want) {
		t.Errorf("released %v, want the old slot %v", f.slots.released, want)
	}
	wantJobs := []string{
		"slot_freed " + a.SlotID + " " + clinicA,
		"reminder " + apptA + " " + clinicA + " at " + newStart.Add(-24*time.Hour).Format(time.RFC3339),
		"rescheduled " + apptA + " " + clinicA,
	}
	if !slices.Equal(f.jobs.enqueued, wantJobs) {
		t.Errorf("enqueued %v, want %v", f.jobs.enqueued, wantJobs)
	}
	if want := []string{clinicA + " " + start.Format(time.RFC3339), clinicA + " " + newStart.Format(time.RFC3339)}; !slices.Equal(f.stats.refreshes, want) {
		t.Errorf("refreshes %v, want both days %v", f.stats.refreshes, want)
	}
	if len(f.repo.changes) != 2 {
		t.Fatalf("history %+v, want the reschedule and the confirmation", f.repo.changes)
	}
	moved, confirmed := f.repo.changes[0], f.repo.changes[1]
	if moved.Kind != appointment.ChangeRescheduled || moved.Via != appointment.ViaApp || moved.ByAccountID != accountA ||
		!moved.FromStart.Equal(start) || !moved.ToStart.Equal(newStart) || moved.FromSlotID != a.SlotID || moved.ToSlotID != slotLater ||
		!moved.FromEnd.Equal(a.End) || !moved.ToEnd.Equal(newStart.Add(30*time.Minute)) {
		t.Errorf("reschedule's change %+v, want both periods and slots, by patient A in the app", moved)
	}
	if confirmed.Kind != appointment.ChangeConfirmed || confirmed.Via != appointment.ViaApp {
		t.Errorf("second change %+v, want the confirmation in the app", confirmed)
	}
	checkEvents(t, "Reschedule", f.audit, event(me, auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess))
}

func TestReschedule_AConfirmedAppointmentKeepsItsConfirmation(t *testing.T) {
	a := booked(apptA, patientA, roleA1, clinicA, start)
	a.ConfirmedAt, a.ConfirmedVia = now.Add(-time.Hour), appointment.ViaLink
	f := rescheduleFixture(a)
	got, err := f.svc.Reschedule(t.Context(), me, apptA, slotLater)
	if err != nil || !got.ConfirmedAt.Equal(a.ConfirmedAt) || got.ConfirmedVia != appointment.ViaLink {
		t.Fatalf("Reschedule() = %+v, %v; want the first confirmation kept", got, err)
	}
	if len(f.repo.changes) != 1 || f.repo.changes[0].Kind != appointment.ChangeRescheduled {
		t.Errorf("history %+v, want the reschedule only", f.repo.changes)
	}
}

func TestReschedule_Refusals(t *testing.T) {
	cancelled := booked(apptA, patientA, roleA1, clinicA, start)
	cancelled.Status = appointment.StatusCancelled
	limit := booked(apptA, patientA, roleA1, clinicA, start)
	limit.RescheduleCount = 2
	for _, tc := range []struct {
		want    error
		set     *organization.AppointmentSettings
		consent *access.Consent
		name    string
		slot    string
		a       appointment.Appointment
		caller  appointment.Caller
	}{
		{name: "staff", caller: pract, slot: slotLater, want: appointment.ErrForbidden},
		{name: "another patient's", caller: other, slot: slotLater, want: appointment.ErrNotFound},
		{name: "unknown slot", caller: me, slot: newID, want: appointment.ErrSlotUnavailable},
		{name: "past slot", caller: me, slot: slotPast, want: appointment.ErrSlotUnavailable},
		{name: "its own slot", caller: me, slot: "51" + apptA[2:], want: appointment.ErrSlotUnavailable},
		{name: "another role's slot", caller: me, slot: slotOther, want: appointment.ErrNotReschedulable},
		{name: "another clinic's slot", caller: me, slot: slotB, want: appointment.ErrNotReschedulable},
		{name: "self-service off", caller: me, slot: slotLater, set: &organization.AppointmentSettings{RescheduleNoticeHours: 24}, want: appointment.ErrNotReschedulable},
		{name: "inside the notice", a: booked(apptA, patientA, roleA1, clinicA, now.Add(23*time.Hour)), caller: me, slot: slotLater, want: appointment.ErrRescheduleWindowClosed},
		{name: "join window open", a: booked(apptA, patientA, roleA1, clinicA, now.Add(9*time.Minute)), caller: me, slot: slotLater,
			set: &organization.AppointmentSettings{RescheduleMax: 2}, want: appointment.ErrNotReschedulable},
		{name: "limit reached", a: limit, caller: me, slot: slotLater, want: appointment.ErrRescheduleLimit},
		{name: "cancelled", a: cancelled, caller: me, slot: slotLater, want: appointment.ErrClosed},
		{name: "consent revoked", caller: me, slot: slotLater, consent: &access.Consent{RevokedAt: revokedAt}, want: appointment.ErrConsentRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := tc.a
			if a.ID == "" {
				a = booked(apptA, patientA, roleA1, clinicA, start)
			}
			f := rescheduleFixture(a)
			if tc.name == "its own slot" {
				f.slots.held = append(f.slots.held, a.SlotID) // busy with the appointment
			}
			if tc.set != nil {
				f.settings.set[clinicA] = *tc.set
			}
			if tc.consent != nil {
				f.consents.state[patientA+clinicA] = *tc.consent
			}
			if _, err := f.svc.Reschedule(t.Context(), tc.caller, apptA, tc.slot); !errors.Is(err, tc.want) {
				t.Fatalf("Reschedule() = %v, want %v", err, tc.want)
			}
			if got := f.repo.appts[0]; got.Start != a.Start || got.SlotID != a.SlotID || got.RescheduleCount != a.RescheduleCount {
				t.Errorf("appointment %+v, want it unchanged", got)
			}
			if len(f.repo.changes) != 0 || len(f.jobs.enqueued) != 0 || len(f.slots.released) != 0 {
				t.Errorf("history %v, jobs %v, released %v; want nothing", f.repo.changes, f.jobs.enqueued, f.slots.released)
			}
			checkEvents(t, "Reschedule", f.audit, event(tc.caller, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure))
		})
	}
}

func TestReschedule_ATimeTheRoleIsBookedAtIsRefusedAndTheHoldRolledBack(t *testing.T) {
	a := booked(apptA, patientA, roleA1, clinicA, start)
	f := rescheduleFixture(a)
	// Patient B is booked with the role at the new slot's time through another slot (an overlapping schedule).
	f.repo.appts = append(f.repo.appts, booked(apptB, patientB, roleA1, clinicA, start.Add(24*time.Hour)))
	if _, err := f.svc.Reschedule(t.Context(), me, apptA, slotLater); !errors.Is(err, appointment.ErrSlotUnavailable) {
		t.Fatalf("Reschedule() = %v, want ErrSlotUnavailable", err)
	}
	if len(f.slots.held) != 0 {
		t.Errorf("held %v, want the hold rolled back", f.slots.held)
	}
	checkEvents(t, "Reschedule", f.audit, event(me, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure))
}

func TestReschedule_ADatabaseFailureIsNotARefusal(t *testing.T) {
	f := rescheduleFixture(booked(apptA, patientA, roleA1, clinicA, start))
	f.settings.err = errBackend
	if _, err := f.svc.Reschedule(t.Context(), me, apptA, slotLater); !errors.Is(err, errBackend) {
		t.Fatalf("Reschedule() = %v, want the backend's error", err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %v, want nothing (rolled back)", f.audit.events)
	}
}
