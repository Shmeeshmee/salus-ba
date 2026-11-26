package appointment_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// patientC is registered by clinic A and has no account (docs/clinic-cms F2).
const patientC = "dd000000-0000-4000-8000-0000000000dd"

// patientD is registered by clinic A and has no account and no phone (docs/clinic-cms R3).
const patientD = "d4000000-0000-4000-8000-0000000000d4"

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

// A patient without an account gets no confirmation and no questionnaire request; their reminders are scheduled
// (docs/clinic-cms R3: an SMS when the clinic offers it, service_reminder_patient_test.go).
func TestBookForPatient_SendsAPatientWithoutAnAccountNoEmail(t *testing.T) {
	t.Parallel()
	f := staffFixture()
	got, err := f.svc.BookForPatient(t.Context(), desk, appointment.StaffBooking{PatientID: patientC, SlotID: slotFree})
	if err != nil || got.PatientID != patientC {
		t.Fatalf("BookForPatient = %+v, %v", got, err)
	}
	for _, j := range f.jobs.enqueued {
		if !strings.HasPrefix(j, "reminder ") {
			t.Errorf("enqueued %q, want only reminders", j)
		}
	}
	if len(f.stats.refreshes) != 1 {
		t.Errorf("refreshes %v, want the statistics", f.stats.refreshes)
	}
}

func TestBookForPatient_RefusesAndAuditsEachCase(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want   error
		b      appointment.StaffBooking
		caller appointment.Caller
	}{
		"a patient":            {caller: me, b: appointment.StaffBooking{PatientID: patientA, SlotID: slotFree}, want: appointment.ErrForbidden},
		"another practitioner": {caller: pract2, b: appointment.StaffBooking{PatientID: patientA, SlotID: slotFree}, want: appointment.ErrForbidden},
		// Resolved in the clinic's own scope, another clinic's slot is one it does not have: unavailable, as an unknown one.
		"another clinic's slot": {caller: admin, b: appointment.StaffBooking{PatientID: patientA, SlotID: slotB}, want: appointment.ErrSlotUnavailable},
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

// While the clinic restricts the patient's record, its staff book them nothing (docs/clinic-cms/WAVE5.md S7): not
// found, audited, and nothing held or booked.
func TestBookForPatient_RefusesARestrictedRecord(t *testing.T) {
	t.Parallel()
	f := staffFixture()
	f.consents.state[patientA+clinicA] = access.Consent{Active: true, Restricted: true}
	if _, err := f.svc.BookForPatient(t.Context(), admin, appointment.StaffBooking{PatientID: patientA, SlotID: slotFree}); !errors.Is(err, appointment.ErrNotFound) {
		t.Fatalf("BookForPatient = %v, want ErrNotFound", err)
	}
	if len(f.repo.appts) != 0 || len(f.slots.held) != 0 {
		t.Errorf("booked %v, held %v; want nothing", f.repo.appts, f.slots.held)
	}
	if want := []auditevent.AuditEvent{event(admin, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("audit %+v, want %+v", f.audit.events, want)
	}
}

// The clinic's own staff book in its own scope, whether or not it is listed (a clinic waiting for the operator's
// approval uses its desk): the slot is resolved in the clinic, not in the public directory.
func TestBookForPatient_BooksInTheClinicsOwnScopeListedOrNot(t *testing.T) {
	t.Parallel()
	f := staffFixture()
	adminU := appointment.Caller{AccountID: "ae000000-0000-4000-8000-0000000000ae", OrganizationID: clinicU, Admin: true}
	got, err := f.svc.BookForPatient(t.Context(), adminU, appointment.StaffBooking{PatientID: patientA, SlotID: slotU})
	if err != nil || got.OrganizationID != clinicU || got.PractitionerRoleID != roleU {
		t.Fatalf("BookForPatient at the unlisted clinic = %+v, %v; want the booking", got, err)
	}
	if !slices.Equal(f.slots.inClinic, []string{clinicU + "/" + slotU}) {
		t.Errorf("slots resolved %v, want the clinic's own scope", f.slots.inClinic)
	}
}

// A clinic whose membership is suspended or cancelled takes no new booking, at the desk either (docs/plans/clinic-cms.md
// Q4); a slot of a role that is no longer active is not bookable. Both refusals are audited.
func TestBookForPatient_RefusesWhenTheClinicStoppedBookingsOrTheRoleIsInactive(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		slot    string
		stopped bool
	}{
		"a suspended clinic": {slot: slotFree, stopped: true},
		"an inactive role":   {slot: slotAx},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := staffFixture()
			f.stops[clinicA] = tc.stopped
			if _, err := f.svc.BookForPatient(t.Context(), desk, appointment.StaffBooking{PatientID: patientA, SlotID: tc.slot}); !errors.Is(err, appointment.ErrSlotUnavailable) {
				t.Fatalf("BookForPatient = %v, want ErrSlotUnavailable", err)
			}
			if len(f.repo.appts) != 0 || len(f.slots.held) != 0 {
				t.Errorf("booked %v, held %v; want nothing", f.repo.appts, f.slots.held)
			}
			if want := []auditevent.AuditEvent{event(desk, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
				t.Errorf("audit %+v, want %+v", f.audit.events, want)
			}
		})
	}
}

// A staff booking names a service of the clinic's own price list, active and offered in the slot's mode, read in the
// clinic's scope (the public price list is a listed clinic's only).
func TestBookForPatient_TakesAServiceOfTheClinicsOwnPriceList(t *testing.T) {
	t.Parallel()
	f, services := serviceFixture()
	f.consents.state[patientA+clinicA] = access.Consent{Active: true}
	if _, err := f.svc.BookForPatient(t.Context(), desk, appointment.StaffBooking{PatientID: patientA, SlotID: slotFree, ServiceID: serviceVideo}); err != nil {
		t.Fatalf("BookForPatient with a service = %v", err)
	}
	if !slices.Equal(services.asked, []string{clinicA + " all"}) {
		t.Errorf("services asked %v, want the clinic's own list", services.asked)
	}
	for _, id := range []string{serviceRetired, "c1000000-0000-4000-8000-0000000000ff"} {
		if _, err := f.svc.BookForPatient(t.Context(), desk, appointment.StaffBooking{PatientID: patientA, SlotID: slotSoon, ServiceID: id}); !errors.Is(err, appointment.ErrInvalidService) {
			t.Errorf("BookForPatient with service %s = %v, want ErrInvalidService", id, err)
		}
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

// A visit of a patient whose record is restricted at the clinic (docs/clinic-cms/WAVE5.md S7) is not on the desk's
// day, and the desk cannot check it in through its id either: refused as not found, audited, nothing changed
// (docs/clinic-cms/WAVE7.md RD1).
func TestArrive_RefusesAVisitOfARestrictedRecordAsNotFound(t *testing.T) {
	t.Parallel()
	// A video visit too: answering not_in_person would tell the desk the hidden visit exists and is a video visit.
	for _, mode := range []schedule.VisitMode{schedule.ModeInPerson, schedule.ModeVideo} {
		a := booked(apptA, patientA, roleA1, clinicA, now.Add(time.Hour))
		a.Mode = mode
		f := staffFixture(a)
		f.patients.restricted = patientA + clinicA
		for _, c := range []appointment.Caller{desk, admin} {
			if _, err := f.svc.Arrive(t.Context(), c, apptA); !errors.Is(err, appointment.ErrNotFound) {
				t.Errorf("Arrive of a restricted record's %s visit as %s = %v, want ErrNotFound", mode, c.AccountID, err)
			}
		}
		if got := f.repo.appts[0]; got.Status != appointment.StatusBooked || len(f.repo.changes) != 0 {
			t.Errorf("the %s visit %+v, changes %+v; want it booked and unchanged", mode, got, f.repo.changes)
		}
		if n := len(f.audit.events); n != 2 || f.audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("audit of the %s visit %+v, want two refusals", mode, f.audit.events)
		}
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
	out, err := f.svc.DeskDay(t.Context(), desk, time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC), "")
	got := out.Visits
	if err != nil || len(got) != 2 || got[0].Appointment.ID != early.ID || got[1].Appointment.ID != late.ID || got[0].Patient.GivenName != "Ada" ||
		!out.Day.Equal(day) {
		t.Fatalf("DeskDay = %+v, %v; want the two visits of 7 January in Sarajevo with patient A", out, err)
	}
	want := []auditevent.AuditEvent{
		event(desk, auditevent.CodeRead, "Patient", patientA, auditevent.OutcomeSuccess),
		event(desk, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess),
	}
	if !slices.Equal(f.audit.events, want) {
		t.Errorf("audit %+v, want %+v", f.audit.events, want)
	}
	if _, err := f.svc.DeskDay(t.Context(), pract, day, ""); !errors.Is(err, appointment.ErrForbidden) {
		t.Errorf("a practitioner = %v, want ErrForbidden", err)
	}
}

// With a location the desk's day is that location's in-person visits and every video visit (docs/clinic-cms/WAVE6.md
// U2).
func TestDeskDay_AtALocationGivesItsVisitsAndTheVideoVisits(t *testing.T) {
	t.Parallel()
	sarajevo, _ := time.LoadLocation("Europe/Sarajevo")
	day := time.Date(2030, 1, 7, 0, 0, 0, 0, sarajevo)
	const siteA, siteB = "a3000000-0000-4000-8000-0000000000a1", "a3000000-0000-4000-8000-0000000000a2"
	here := booked("ab000000-0000-4000-8000-0000000000ab", patientA, roleA1, clinicA, day.Add(9*time.Hour))
	here.Mode, here.LocationID = schedule.ModeInPerson, siteA
	there := booked("ac000000-0000-4000-8000-0000000000ac", patientA, roleA1, clinicA, day.Add(10*time.Hour))
	there.Mode, there.LocationID = schedule.ModeInPerson, siteB
	video := booked("ad000000-0000-4000-8000-0000000000ad", patientA, roleA1, clinicA, day.Add(11*time.Hour))
	f := staffFixture(here, there, video)
	out, err := f.svc.DeskDay(t.Context(), desk, day, siteA)
	if err != nil || len(out.Visits) != 2 || out.Visits[0].Appointment.ID != here.ID || out.Visits[1].Appointment.ID != video.ID {
		t.Fatalf("DeskDay(site A) = %+v, %v; want site A's visit and the video visit", out.Visits, err)
	}
	if all, err := f.svc.DeskDay(t.Context(), desk, day, ""); err != nil || len(all.Visits) != 3 {
		t.Errorf("DeskDay(every location) = %d visits, %v; want 3", len(all.Visits), err)
	}
}

// Without a date the desk's day is today in the clinic's zone (00:30 in Sarajevo is still 23:30 UTC the day before),
// and each visit names its booked service from the clinic's own price list.
func TestDeskDay_IsTodayInTheClinicsZoneAndNamesEachService(t *testing.T) {
	t.Parallel()
	sarajevo, _ := time.LoadLocation("Europe/Sarajevo")
	today := time.Date(2030, 1, 7, 0, 0, 0, 0, sarajevo)
	visit := booked("ab000000-0000-4000-8000-0000000000ab", patientA, roleA1, clinicA, today.Add(10*time.Hour))
	visit.ServiceID = serviceInPerson
	plain := booked("ac000000-0000-4000-8000-0000000000ac", patientA, roleA1, clinicA, today.Add(11*time.Hour))
	f, services := serviceFixture()
	f.repo.appts = []appointment.Appointment{visit, plain}
	svc := appointment.NewService(appointment.Ports{
		Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: today.Add(30 * time.Minute)},
		Patients: f.patients, Clinics: f.clinics, Services: services, Rules: f.rules, Consents: f.consents,
	})
	out, err := svc.DeskDay(t.Context(), desk, time.Time{}, "")
	if err != nil || !out.Day.Equal(today) || len(out.Visits) != 2 {
		t.Fatalf("DeskDay(no date) = %+v, %v; want 7 January in Sarajevo with its two visits", out, err)
	}
	if out.Visits[0].ServiceName != "Synthetic exam" || out.Visits[1].ServiceName != "" {
		t.Errorf("service names %q, %q; want the booked service's and none", out.Visits[0].ServiceName, out.Visits[1].ServiceName)
	}
}

// A visit the desk marked arrived is then closed by its practitioner (fulfilled or a no-show), or cancelled by staff;
// the arrival and the outcome each leave a change in the visit's history.
func TestArrivedVisit_IsClosedByThePractitionerOrCancelledByStaff(t *testing.T) {
	t.Parallel()
	arrived := booked(apptA, patientA, roleA1, clinicA, now.Add(-time.Hour))
	arrived.Status, arrived.Mode = appointment.StatusArrived, schedule.ModeInPerson
	f := staffFixture(arrived)
	got, err := f.svc.SetStatus(t.Context(), pract, apptA, appointment.StatusFulfilled)
	if err != nil || got.Status != appointment.StatusFulfilled {
		t.Fatalf("SetStatus(fulfilled) of an arrived visit = %+v, %v", got, err)
	}
	if n := len(f.repo.changes); n != 1 || f.repo.changes[0].FromStatus != appointment.StatusArrived {
		t.Errorf("changes %+v, want one from arrived", f.repo.changes)
	}
	g := staffFixture(arrived)
	if got, err := g.svc.CancelForPatient(t.Context(), desk, apptA, arrived.VersionID); err != nil || got.Status != appointment.StatusCancelled {
		t.Errorf("CancelForPatient of an arrived visit = %+v, %v; want cancelled", got, err)
	}
	h := staffFixture(booked(apptA, patientA, roleA1, clinicA, now.Add(time.Hour)))
	h.repo.appts[0].Mode = schedule.ModeInPerson
	if _, err := h.svc.Arrive(t.Context(), desk, apptA); err != nil {
		t.Fatal(err)
	}
	if n := len(h.repo.changes); n != 1 || h.repo.changes[0].FromStatus != appointment.StatusBooked || h.repo.changes[0].ToStatus != appointment.StatusArrived {
		t.Errorf("changes %+v, want the arrival", h.repo.changes)
	}
}

// A visit whose patient the clinic's scope no longer shows (a revocation, before the job cancels it) is left off the
// desk's day; the rest of the day is still given.
func TestDeskDay_LeavesOutAVisitWhosePatientTheScopeHides(t *testing.T) {
	t.Parallel()
	sarajevo, _ := time.LoadLocation("Europe/Sarajevo")
	day := time.Date(2030, 1, 7, 0, 0, 0, 0, sarajevo)
	seen := booked("ab000000-0000-4000-8000-0000000000ab", patientA, roleA1, clinicA, day.Add(9*time.Hour))
	hidden := booked("ac000000-0000-4000-8000-0000000000ac", "ee000000-0000-4000-8000-0000000000ee", roleA1, clinicA, day.Add(10*time.Hour))
	f := staffFixture(seen, hidden)
	out, err := f.svc.DeskDay(t.Context(), desk, day, "")
	if err != nil || len(out.Visits) != 1 || out.Visits[0].Appointment.ID != seen.ID {
		t.Fatalf("DeskDay = %+v, %v; want the one visit whose patient the scope shows", out, err)
	}
}

// A patient whose record is restricted at the clinic (docs/clinic-cms/WAVE5.md S7) is not on the desk: their visits
// are left out, unread and unaudited.
func TestDeskDay_LeavesOutARestrictedPatient(t *testing.T) {
	t.Parallel()
	sarajevo, _ := time.LoadLocation("Europe/Sarajevo")
	day := time.Date(2030, 1, 7, 0, 0, 0, 0, sarajevo)
	restricted := booked("ab000000-0000-4000-8000-0000000000ab", patientA, roleA1, clinicA, day.Add(9*time.Hour))
	f := staffFixture(restricted)
	f.consents.state[patientA+clinicA] = access.Consent{Active: true, Restricted: true}
	out, err := f.svc.DeskDay(t.Context(), desk, day, "")
	if err != nil || len(out.Visits) != 0 {
		t.Fatalf("DeskDay = %+v, %v; want no visit of the restricted patient", out, err)
	}
	for _, e := range f.audit.events {
		if strings.HasPrefix(e.EntityReference, "Patient/") {
			t.Errorf("audited %+v, want no read of the restricted patient", e)
		}
	}
}

// fakeReceipts gives the visits' receipts the desk shows, from the clinic it is asked for.
type fakeReceipts map[string][]appointment.VisitReceipt

func (r fakeReceipts) VisitReceipts(_ context.Context, org string, ids []string) (map[string][]appointment.VisitReceipt, error) {
	out := map[string][]appointment.VisitReceipt{}
	for _, id := range ids {
		if org == clinicA {
			out[id] = r[id]
		}
	}
	return out, nil
}

// Each visit on the desk's day carries its receipts (number, id, whether reversed), whoever issued them, so the row
// can show them and open their PDF (docs/clinic-cms/walk/wave3, findings 1 and 2); each receipt shown is audited as a
// read by the desk's caller.
func TestDeskDay_GivesEachVisitItsReceiptsAndAuditsThem(t *testing.T) {
	t.Parallel()
	sarajevo, _ := time.LoadLocation("Europe/Sarajevo")
	day := time.Date(2030, 1, 7, 0, 0, 0, 0, sarajevo)
	receipted := booked("ab000000-0000-4000-8000-0000000000ab", patientA, roleA1, clinicA, day.Add(9*time.Hour))
	plain := booked("ac000000-0000-4000-8000-0000000000ac", patientA, roleA1, clinicA, day.Add(10*time.Hour))
	f := staffFixture(receipted, plain)
	receipts := fakeReceipts{receipted.ID: {{ID: "e1000000-0000-4000-8000-0000000000e1", Number: 7, Reversed: true}, {ID: "e2000000-0000-4000-8000-0000000000e2", Number: 8}}}
	svc := appointment.NewService(appointment.Ports{Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: now},
		Patients: f.patients, Clinics: f.clinics, Receipts: receipts, Rules: f.rules, Consents: f.consents})
	out, err := svc.DeskDay(t.Context(), desk, day, "")
	if err != nil || len(out.Visits) != 2 {
		t.Fatalf("DeskDay = %+v, %v", out, err)
	}
	if got := out.Visits[0].Receipts; !slices.Equal(got, receipts[receipted.ID]) || len(out.Visits[1].Receipts) != 0 {
		t.Errorf("receipts %+v and %+v, want the first visit's two and none", got, out.Visits[1].Receipts)
	}
	for _, id := range []string{"e1000000-0000-4000-8000-0000000000e1", "e2000000-0000-4000-8000-0000000000e2"} {
		if !slices.Contains(f.audit.events, event(desk, auditevent.CodeRead, "Invoice", id, auditevent.OutcomeSuccess)) {
			t.Errorf("no audited read of Invoice/%s by the desk in %+v", id, f.audit.events)
		}
	}
}

// The desk's day writes its events (each patient's read, each receipt's and the search) in one statement, not one a
// row (docs/clinic-cms/load/REPORT.md).
func TestDeskDay_WritesItsEventsAtOnce(t *testing.T) {
	t.Parallel()
	sarajevo, _ := time.LoadLocation("Europe/Sarajevo")
	day := time.Date(2030, 1, 7, 0, 0, 0, 0, sarajevo)
	first := booked("ab000000-0000-4000-8000-0000000000ab", patientA, roleA1, clinicA, day.Add(9*time.Hour))
	second := booked("ac000000-0000-4000-8000-0000000000ac", patientA, roleA1, clinicA, day.Add(10*time.Hour))
	f := staffFixture(first, second)
	receipts := fakeReceipts{first.ID: {{ID: "e1000000-0000-4000-8000-0000000000e1", Number: 7}}}
	svc := appointment.NewService(appointment.Ports{Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: now},
		Patients: f.patients, Clinics: f.clinics, Receipts: receipts, Rules: f.rules, Consents: f.consents})
	if _, err := svc.DeskDay(t.Context(), desk, day, ""); err != nil {
		t.Fatal(err)
	}
	want := []auditevent.AuditEvent{
		event(desk, auditevent.CodeRead, "Patient", patientA, auditevent.OutcomeSuccess),
		event(desk, auditevent.CodeRead, "Invoice", "e1000000-0000-4000-8000-0000000000e1", auditevent.OutcomeSuccess),
		event(desk, auditevent.CodeSearchType, "", "", auditevent.OutcomeSuccess),
	}
	if f.audit.writes != 1 || !slices.Equal(f.audit.events, want) {
		t.Errorf("%d writes of %+v, want one of %+v", f.audit.writes, f.audit.events, want)
	}
	if f.consents.asked != 0 || f.consents.batches != 1 || f.patients.batches != 1 {
		t.Errorf("consents %d one by one and %d at once, profiles in %d reads; want one read of each", f.consents.asked, f.consents.batches, f.patients.batches)
	}
}
