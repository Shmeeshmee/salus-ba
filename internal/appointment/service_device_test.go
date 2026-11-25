package appointment_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// A booking or a move whose service needs a device another visit holds then is refused with ErrResourceBusy (U4 of
// docs/clinic-cms/WAVE6.md, RM1), audited as refused after the rollback, as a time the practitioner is booked at is.
func TestBusyDevice_RefusesTheBookingOrMoveAudited(t *testing.T) {
	t.Parallel()
	t.Run("the patient's booking", func(t *testing.T) {
		t.Parallel()
		f := newFixture()
		f.repo.busy = true
		if _, err := f.svc.Book(t.Context(), me, slotFree, ""); !errors.Is(err, appointment.ErrResourceBusy) {
			t.Fatalf("Book() = %v, want ErrResourceBusy", err)
		}
		checkEvents(t, "Book", f.audit, event(me, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure))
	})
	t.Run("the desk's booking", func(t *testing.T) {
		t.Parallel()
		f := staffFixture()
		f.consents.state[patientA+clinicA] = access.Consent{Active: true}
		f.repo.busy = true
		if _, err := f.svc.BookForPatient(t.Context(), desk, appointment.StaffBooking{PatientID: patientA, SlotID: slotFree}); !errors.Is(err, appointment.ErrResourceBusy) {
			t.Fatalf("BookForPatient() = %v, want ErrResourceBusy", err)
		}
		checkEvents(t, "BookForPatient", f.audit, event(desk, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure))
	})
	t.Run("the patient's reschedule", func(t *testing.T) {
		t.Parallel()
		f := rescheduleFixture(booked(apptA, patientA, roleA1, clinicA, start))
		f.repo.busy = true
		if _, err := f.svc.Reschedule(t.Context(), me, apptA, slotLater); !errors.Is(err, appointment.ErrResourceBusy) {
			t.Fatalf("Reschedule() = %v, want ErrResourceBusy", err)
		}
		checkEvents(t, "Reschedule", f.audit, event(me, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure))
	})
	t.Run("the desk's move", func(t *testing.T) {
		t.Parallel()
		f := moveFixture(booked(apptA, patientA, roleA1, clinicA, start))
		f.repo.busy = true
		if _, err := f.svc.MoveForPatient(t.Context(), desk, apptA, slotOther, 1); !errors.Is(err, appointment.ErrResourceBusy) {
			t.Fatalf("MoveForPatient() = %v, want ErrResourceBusy", err)
		}
		checkEvents(t, "MoveForPatient", f.audit, event(desk, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure))
	})
}
