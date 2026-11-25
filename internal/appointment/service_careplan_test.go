package appointment_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// A treatment plan's steps are booked all or none (docs/clinic-cms/WAVE6.md U6, package TP1).
func TestBookPlan_BooksEveryStepAndAuditsEach(t *testing.T) {
	t.Parallel()
	f := staffFixture()
	got, err := f.svc.BookPlan(t.Context(), admin, patientA, []appointment.StaffBooking{{SlotID: slotFree}, {SlotID: slotSoon}})
	if err != nil || len(got) != 2 || got[0].PatientID != patientA || got[1].BookedBy != admin.AccountID {
		t.Fatalf("BookPlan = %+v, %v; want two visits of patient A", got, err)
	}
	if !slices.Equal(f.slots.held, []string{slotFree, slotSoon}) {
		t.Errorf("held %v, want both slots", f.slots.held)
	}
	want := event(admin, auditevent.CodeCreate, "Appointment", got[0].ID, auditevent.OutcomeSuccess)
	if len(f.audit.events) != 2 || f.audit.events[0] != want {
		t.Errorf("audit %+v, want each booking", f.audit.events)
	}
}

func TestBookPlan_BooksNoneWhenAStepIsRefused(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want   error
		caller appointment.Caller
		steps  []appointment.StaffBooking
	}{
		"a past slot":    {appointment.ErrSlotUnavailable, admin, []appointment.StaffBooking{{SlotID: slotFree}, {SlotID: slotPast}}},
		"another clinic": {appointment.ErrSlotUnavailable, admin, []appointment.StaffBooking{{SlotID: slotFree}, {SlotID: slotB}}},
		"a patient":      {appointment.ErrForbidden, me, []appointment.StaffBooking{{SlotID: slotFree}}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := staffFixture()
			if _, err := f.svc.BookPlan(t.Context(), tc.caller, patientA, tc.steps); !errors.Is(err, tc.want) {
				t.Fatalf("BookPlan: %v, want %v", err, tc.want)
			}
			if len(f.slots.held) != 0 || len(f.audit.events) != 0 || len(f.repo.appts) != 0 {
				t.Errorf("held %v, audited %v, stored %v; want nothing (the caller audits the refusal)", f.slots.held, f.audit.events, f.repo.appts)
			}
		})
	}
}

func TestCancelPlan_CancelsTheFutureBookedVisitsOnly(t *testing.T) {
	t.Parallel()
	const ahead, past = "a1000000-0000-4000-8000-0000000000a1", "a2000000-0000-4000-8000-0000000000a2"
	f := staffFixture(booked(ahead, patientA, roleA1, clinicA, start), booked(past, patientA, roleA1, clinicA, now.Add(-48*time.Hour)))
	if err := f.svc.CancelPlan(t.Context(), pract, []string{ahead, past, newID}); err != nil {
		t.Fatalf("CancelPlan: %v", err)
	}
	visits, err := f.svc.PlanVisits(t.Context(), clinicA, []string{ahead, past, newID})
	if err != nil || len(visits) != 2 {
		t.Fatalf("PlanVisits = %+v, %v; want the two visits", visits, err)
	}
	if visits[0].Status != appointment.StatusCancelled || visits[1].Status != appointment.StatusBooked {
		t.Errorf("statuses %s, %s; want the future one cancelled, the past one kept", visits[0].Status, visits[1].Status)
	}
	if want := event(pract, auditevent.CodeUpdate, "Appointment", ahead, auditevent.OutcomeSuccess); !slices.Contains(f.audit.events, want) {
		t.Errorf("audit %+v, want the cancellation", f.audit.events)
	}
}
