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

// patientC is registered by clinic A and has no account (docs/clinic-cms F2).
const patientC = "dd000000-0000-4000-8000-0000000000dd"

var desk = appointment.Caller{AccountID: "de000000-0000-4000-8000-0000000000de", OrganizationID: clinicA, Reception: true}

func staffFixture(appts ...appointment.Appointment) *fixture {
	f := newFixture(appts...)
	f.consents.state[patientC+clinicA] = access.Consent{Active: true}
	return f
}

func TestBookForPatient_BooksAnyFreeSlotOfTheClinicWithWhoBookedAndAuditsIt(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]appointment.Caller{"admin": admin, "receptionist": desk, "its practitioner": pract} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := staffFixture()
			got, err := f.svc.BookForPatient(t.Context(), c, appointment.StaffBooking{PatientID: patientA, SlotID: slotFree})
			if err != nil || got.ID != newID || got.BookedBy != c.AccountID || got.PatientID != patientA || got.PractitionerRoleID != roleA1 || !got.Start.Equal(start) {
				t.Fatalf("BookForPatient = %+v, %v; want patient A's booking by %s", got, err, c.AccountID)
			}
			if !slices.Equal(f.slots.held, []string{slotFree}) || !slices.Contains(f.jobs.enqueued, "booked "+newID+" "+clinicA) {
				t.Errorf("held %v, enqueued %v; want the slot held and the confirmation", f.slots.held, f.jobs.enqueued)
			}
			if f.repo.patient != "" {
				t.Errorf("booked in patient %q's scope, want the clinic's", f.repo.patient)
			}
			if want := []auditevent.AuditEvent{event(c, auditevent.CodeCreate, "Appointment", newID, auditevent.OutcomeSuccess)}; !slices.Equal(f.audit.events, want) {
				t.Errorf("audit %+v, want %+v", f.audit.events, want)
			}
		})
	}
}

func TestBookForPatient_SendsAPatientWithoutAnAccountNothing(t *testing.T) {
	t.Parallel()
	f := staffFixture()
	got, err := f.svc.BookForPatient(t.Context(), desk, appointment.StaffBooking{PatientID: patientC, SlotID: slotFree})
	if err != nil || got.PatientID != patientC {
		t.Fatalf("BookForPatient = %+v, %v", got, err)
	}
	if len(f.jobs.enqueued) != 0 || len(f.reminders.reminders) != 0 || len(f.stats.refreshes) != 1 {
		t.Errorf("enqueued %v, reminders %v, refreshes %v; want only the statistics", f.jobs.enqueued, f.reminders.reminders, f.stats.refreshes)
	}
}

func TestBookForPatient_RefusesAndAuditsEachCase(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want   error
		b      appointment.StaffBooking
		caller appointment.Caller
	}{
		"a patient":             {caller: me, b: appointment.StaffBooking{PatientID: patientA, SlotID: slotFree}, want: appointment.ErrForbidden},
		"another practitioner":  {caller: pract2, b: appointment.StaffBooking{PatientID: patientA, SlotID: slotFree}, want: appointment.ErrForbidden},
		"another clinic's slot": {caller: admin, b: appointment.StaffBooking{PatientID: patientA, SlotID: slotB}, want: appointment.ErrNotFound},
		"a past slot":           {caller: admin, b: appointment.StaffBooking{PatientID: patientA, SlotID: slotPast}, want: appointment.ErrSlotUnavailable},
		"an unknown patient":    {caller: admin, b: appointment.StaffBooking{PatientID: newID, SlotID: slotFree}, want: appointment.ErrNotFound},
		"no consent":            {caller: admin, b: appointment.StaffBooking{PatientID: patientB, SlotID: slotFree}, want: appointment.ErrConsentRequired},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := staffFixture()
			if _, err := f.svc.BookForPatient(t.Context(), tc.caller, tc.b); !errors.Is(err, tc.want) {
				t.Fatalf("BookForPatient = %v, want %v", err, tc.want)
			}
			if len(f.repo.appts) != 0 || len(f.slots.held) != 0 {
				t.Errorf("booked %v, held %v; want nothing", f.repo.appts, f.slots.held)
			}
			if want := []auditevent.AuditEvent{event(tc.caller, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
				t.Errorf("audit %+v, want %+v", f.audit.events, want)
			}
		})
	}
}

func TestBookForPatient_TwoStaffForOneSlotBookItOnce(t *testing.T) {
	t.Parallel()
	f := staffFixture()
	if _, err := f.svc.BookForPatient(t.Context(), admin, appointment.StaffBooking{PatientID: patientA, SlotID: slotFree}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.BookForPatient(t.Context(), desk, appointment.StaffBooking{PatientID: patientC, SlotID: slotFree}); !errors.Is(err, appointment.ErrSlotUnavailable) {
		t.Errorf("the second booking = %v, want ErrSlotUnavailable", err)
	}
}

func TestCancelForPatient_CancelsAtTheVersionOnlyAndAudits(t *testing.T) {
	t.Parallel()
	const id = "ab000000-0000-4000-8000-0000000000ab"
	f := staffFixture(booked(id, patientA, roleA1, clinicA, start))
	if _, err := f.svc.CancelForPatient(t.Context(), desk, id, 7); !errors.Is(err, appointment.ErrVersionConflict) {
		t.Fatalf("a stale version = %v, want ErrVersionConflict", err)
	}
	if f.repo.appts[0].Status != appointment.StatusBooked || len(f.jobs.enqueued) != 0 {
		t.Fatalf("the stale cancel changed %+v, enqueued %v", f.repo.appts[0], f.jobs.enqueued)
	}
	got, err := f.svc.CancelForPatient(t.Context(), desk, id, 1)
	if err != nil || got.Status != appointment.StatusCancelled || got.Reason != appointment.ReasonProvider || got.CancelledBy != desk.AccountID {
		t.Fatalf("CancelForPatient = %+v, %v; want cancelled by the desk (prov)", got, err)
	}
	asDesk := desk
	asDesk.Admin = true
	want := []auditevent.AuditEvent{
		event(asDesk, auditevent.CodeUpdate, "", "", auditevent.OutcomeMinorFailure),
		event(asDesk, auditevent.CodeUpdate, "Appointment", id, auditevent.OutcomeSuccess),
	}
	if !slices.Equal(f.audit.events, want) {
		t.Errorf("audit %+v, want %+v", f.audit.events, want)
	}
	if _, err := f.svc.CancelForPatient(t.Context(), me, id, 2); !errors.Is(err, appointment.ErrForbidden) {
		t.Errorf("a patient = %v, want ErrForbidden", err)
	}
}

func TestArrive_MarksABookedInPersonVisitArrivedAtTheDeskOnly(t *testing.T) {
	t.Parallel()
	const inPerson, video = "ab000000-0000-4000-8000-0000000000ab", "ac000000-0000-4000-8000-0000000000ac"
	a, v := booked(inPerson, patientA, roleA1, clinicA, start), booked(video, patientA, roleA2, clinicA, start)
	a.Mode, v.Mode = schedule.ModeInPerson, schedule.ModeVideo
	f := staffFixture(a, v)
	got, err := f.svc.Arrive(t.Context(), desk, inPerson)
	if err != nil || got.Status != appointment.StatusArrived || !got.ArrivedAt.Equal(now) || got.VersionID != 2 {
		t.Fatalf("Arrive = %+v, %v; want arrived now", got, err)
	}
	for name, tc := range map[string]struct {
		want   error
		id     string
		caller appointment.Caller
	}{
		"again":            {caller: desk, id: inPerson, want: appointment.ErrClosed},
		"a video visit":    {caller: admin, id: video, want: appointment.ErrNotInPerson},
		"a practitioner":   {caller: pract, id: inPerson, want: appointment.ErrForbidden},
		"another clinic":   {caller: adminB, id: inPerson, want: appointment.ErrNotFound},
		"an unknown visit": {caller: admin, id: newID, want: appointment.ErrNotFound},
	} {
		if _, err := f.svc.Arrive(t.Context(), tc.caller, tc.id); !errors.Is(err, tc.want) {
			t.Errorf("%s: Arrive = %v, want %v", name, err, tc.want)
		}
	}
	if n := len(f.audit.events); n != 6 || f.audit.events[0] != event(desk, auditevent.CodeUpdate, "Appointment", inPerson, auditevent.OutcomeSuccess) {
		t.Errorf("audit %+v, want the arrival and five refusals", f.audit.events)
	}
}

func TestDeskDay_GivesTheClinicsDayInItsZoneWithEachPatientAndAuditsTheReads(t *testing.T) {
	t.Parallel()
	sarajevo, _ := time.LoadLocation("Europe/Sarajevo")
	day := time.Date(2030, 1, 7, 0, 0, 0, 0, sarajevo)
	early := booked("ab000000-0000-4000-8000-0000000000ab", patientA, roleA1, clinicA, day.Add(30*time.Minute)) // 23:30 UTC the day before
	late := booked("ac000000-0000-4000-8000-0000000000ac", patientA, roleA1, clinicA, day.Add(10*time.Hour))
	next := booked("ad000000-0000-4000-8000-0000000000ad", patientA, roleA1, clinicA, day.Add(24*time.Hour))
	f := staffFixture(early, late, next)
	got, err := f.svc.DeskDay(t.Context(), desk, time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC))
	if err != nil || len(got) != 2 || got[0].Appointment.ID != early.ID || got[1].Appointment.ID != late.ID || got[0].Patient.GivenName != "Ada" {
		t.Fatalf("DeskDay = %+v, %v; want the two visits of 7 January in Sarajevo with patient A", got, err)
	}
	want := []auditevent.AuditEvent{
		event(desk, auditevent.CodeRead, "Patient", patientA, auditevent.OutcomeSuccess),
		event(desk, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess),
	}
	if !slices.Equal(f.audit.events, want) {
		t.Errorf("audit %+v, want %+v", f.audit.events, want)
	}
	if _, err := f.svc.DeskDay(t.Context(), pract, day); !errors.Is(err, appointment.ErrForbidden) {
		t.Errorf("a practitioner = %v, want ErrForbidden", err)
	}
}

// A staff move is not built yet (docs/clinic-cms/PROGRESS.md, F2 "Deferred").
func TestMoveForPatient_RefusesUntilItIsBuilt(t *testing.T) {
	t.Parallel()
	if _, err := staffFixture().svc.MoveForPatient(t.Context(), desk, newID, slotFree); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("MoveForPatient = %v, want errors.ErrUnsupported", err)
	}
}
