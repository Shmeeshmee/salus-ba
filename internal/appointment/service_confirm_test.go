package appointment_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
)

func TestConfirm_ThePatientConfirmsTheirBookedAppointmentOnce(t *testing.T) {
	f := newSelfFixture(booked(apptA, patientA, roleA1, clinicA, start))
	got, err := f.svc.Confirm(t.Context(), me, apptA)
	if err != nil || !got.ConfirmedAt.Equal(now) || got.ConfirmedVia != appointment.ViaApp || got.VersionID != 2 {
		t.Fatalf("Confirm() = %+v, %v; want confirmed in the app now, at version 2", got, err)
	}
	if len(f.repo.changes) != 1 || f.repo.changes[0].Kind != appointment.ChangeConfirmed || f.repo.changes[0].Via != appointment.ViaApp ||
		f.repo.changes[0].ByAccountID != accountA || !f.repo.changes[0].At.Equal(now) {
		t.Errorf("history %+v, want one confirmation in the app by patient A's account", f.repo.changes)
	}
	again, err := f.svc.Confirm(t.Context(), me, apptA)
	if err != nil || again.VersionID != 2 || len(f.repo.changes) != 1 {
		t.Errorf("Confirm() again = %+v, %v with history %d; want the same appointment, nothing recorded again", again, err, len(f.repo.changes))
	}
	checkEvents(t, "Confirm", f.audit,
		event(me, auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess),
		event(me, auditevent.CodeUpdate, "Appointment", apptA, auditevent.OutcomeSuccess))
}

func TestConfirm_Refusals(t *testing.T) {
	cancelled := booked(apptA2, patientA, roleA2, clinicA, start)
	cancelled.Status = appointment.StatusCancelled
	for _, tc := range []struct {
		want   error
		name   string
		id     string
		caller appointment.Caller
	}{
		{name: "staff", caller: pract, id: apptA, want: appointment.ErrForbidden},
		{name: "another patient's", caller: other, id: apptA, want: appointment.ErrNotFound},
		{name: "no profile", caller: noProfile, id: apptA, want: appointment.ErrNotFound},
		{name: "cancelled", caller: me, id: apptA2, want: appointment.ErrClosed},
		{name: "started", caller: me, id: apptB, want: appointment.ErrClosed},
		{name: "unknown", caller: me, id: newID, want: appointment.ErrNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSelfFixture(booked(apptA, patientA, roleA1, clinicA, start), cancelled, booked(apptB, patientA, roleA1, clinicA, now.Add(-time.Minute)))
			if _, err := f.svc.Confirm(t.Context(), tc.caller, tc.id); !errors.Is(err, tc.want) {
				t.Fatalf("Confirm() = %v, want %v", err, tc.want)
			}
			if len(f.repo.changes) != 0 {
				t.Errorf("history %+v, want none", f.repo.changes)
			}
			checkEvents(t, "Confirm", f.audit, event(tc.caller, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure))
		})
	}
}

func TestConfirm_ADatabaseFailureIsNotARefusal(t *testing.T) {
	f := newSelfFixture(booked(apptA, patientA, roleA1, clinicA, start))
	f.repo.err = errBackend
	if _, err := f.svc.Confirm(t.Context(), me, apptA); !errors.Is(err, errBackend) {
		t.Fatalf("Confirm() = %v, want the backend's error", err)
	}
}

func TestActions_ByTheClinicsRules(t *testing.T) {
	confirmed := booked(apptA, patientA, roleA1, clinicA, start)
	confirmed.ConfirmedAt, confirmed.ConfirmedVia = now.Add(-time.Hour), appointment.ViaLink
	moved := booked(apptA, patientA, roleA1, clinicA, start)
	moved.RescheduleCount = 2
	for _, tc := range []struct {
		name string
		a    appointment.Appointment
		set  *organization.AppointmentSettings
		want appointment.Actions
	}{
		{"booked days ahead", booked(apptA, patientA, roleA1, clinicA, start), nil, appointment.Actions{
			Confirm: true, Cancel: true, CancelUntil: start.Add(-24 * time.Hour), Reschedule: true, RescheduleUntil: start.Add(-24 * time.Hour), ReschedulesLeft: 2,
		}},
		{"confirmed", confirmed, nil, appointment.Actions{
			Cancel: true, CancelUntil: start.Add(-24 * time.Hour), Reschedule: true, RescheduleUntil: start.Add(-24 * time.Hour), ReschedulesLeft: 2,
		}},
		{"moved as often as allowed", moved, nil, appointment.Actions{Confirm: true, Cancel: true, CancelUntil: start.Add(-24 * time.Hour)}},
		{"inside the 24 hours", booked(apptA, patientA, roleA1, clinicA, now.Add(5*time.Hour)), nil, appointment.Actions{Confirm: true, ReschedulesLeft: 2}},
		{"no notice: until the join window opens", booked(apptA, patientA, roleA1, clinicA, now.Add(5*time.Hour)),
			&organization.AppointmentSettings{RescheduleMax: 1},
			appointment.Actions{Confirm: true, Reschedule: true, RescheduleUntil: now.Add(5*time.Hour - 10*time.Minute), ReschedulesLeft: 1}},
		{"self-service off", booked(apptA, patientA, roleA1, clinicA, start), &organization.AppointmentSettings{RescheduleNoticeHours: 24},
			appointment.Actions{Confirm: true, Cancel: true, CancelUntil: start.Add(-24 * time.Hour)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSelfFixture(tc.a)
			if tc.set != nil {
				f.settings.set[clinicA] = *tc.set
			}
			got, err := f.svc.Actions(t.Context(), me, apptA)
			if err != nil || got != tc.want {
				t.Fatalf("Actions() = %+v, %v; want %+v", got, err, tc.want)
			}
			checkEvents(t, "Actions", f.audit, event(me, auditevent.CodeRead, "Appointment", apptA, auditevent.OutcomeSuccess))
		})
	}
}

func TestActions_NoneOnAClosedAppointmentAndRefusals(t *testing.T) {
	cancelled := booked(apptA, patientA, roleA1, clinicA, start)
	cancelled.Status = appointment.StatusCancelled
	f := newSelfFixture(cancelled)
	if got, err := f.svc.Actions(t.Context(), me, apptA); err != nil || got != (appointment.Actions{}) {
		t.Errorf("Actions() on a cancelled one = %+v, %v; want none", got, err)
	}
	for _, tc := range []struct {
		want   error
		caller appointment.Caller
	}{{appointment.ErrForbidden, admin}, {appointment.ErrNotFound, other}} {
		f := newSelfFixture(booked(apptA, patientA, roleA1, clinicA, start))
		if _, err := f.svc.Actions(t.Context(), tc.caller, apptA); !errors.Is(err, tc.want) {
			t.Errorf("Actions() as %s = %v, want %v", tc.caller.AccountID, err, tc.want)
		}
		checkEvents(t, "Actions", f.audit, event(tc.caller, auditevent.CodeRead, "", "", auditevent.OutcomeMinorFailure))
	}
}
