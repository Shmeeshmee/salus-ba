package appointment_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const (
	serviceVideo    = "c1000000-0000-4000-8000-000000000001" // clinic A's, offered by video
	serviceInPerson = "c1000000-0000-4000-8000-000000000002" // clinic A's, in person only
)

// fakeServices is clinic A's price list as chargeitemdefinition.Service.Offered gives it: the active services of a
// listed clinic in a mode.
type fakeServices struct {
	err   error
	asked []string
}

func (s *fakeServices) Offered(_ context.Context, org string, mode schedule.VisitMode, _ string) ([]chargeitemdefinition.Definition, error) {
	s.asked = append(s.asked, org+" "+string(mode))
	if s.err != nil {
		return nil, s.err
	}
	if org != clinicA {
		return nil, chargeitemdefinition.ErrNotFound
	}
	all := []chargeitemdefinition.Definition{
		{ID: serviceVideo, OrganizationID: clinicA, Modes: []schedule.VisitMode{schedule.ModeVideo, schedule.ModeInPerson}},
		{ID: serviceInPerson, OrganizationID: clinicA, Modes: []schedule.VisitMode{schedule.ModeInPerson}},
	}
	var out []chargeitemdefinition.Definition
	for _, d := range all {
		for _, m := range d.Modes {
			if m == mode {
				out = append(out, d)
			}
		}
	}
	return out, nil
}

// OfClinic is clinic A's whole price list, read in its own scope whether or not it is listed (a staff booking).
func (s *fakeServices) OfClinic(_ context.Context, org string) ([]chargeitemdefinition.Definition, error) {
	s.asked = append(s.asked, org+" all")
	if s.err != nil || org != clinicA {
		return nil, s.err
	}
	return []chargeitemdefinition.Definition{
		{ID: serviceVideo, OrganizationID: clinicA, Name: "Synthetic video exam", Active: true, Modes: []schedule.VisitMode{schedule.ModeVideo, schedule.ModeInPerson}},
		{ID: serviceInPerson, OrganizationID: clinicA, Name: "Synthetic exam", Active: true, Modes: []schedule.VisitMode{schedule.ModeInPerson}},
		{ID: serviceRetired, OrganizationID: clinicA, Name: "Synthetic retired exam", Modes: []schedule.VisitMode{schedule.ModeVideo, schedule.ModeInPerson}},
	}, nil
}

const serviceRetired = "c1000000-0000-4000-8000-000000000003" // clinic A's, no longer active

func serviceFixture() (*fixture, *fakeServices) {
	services := &fakeServices{}
	f := newFixture()
	f.svc = appointment.NewService(appointment.Ports{
		Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{f: f}, Clock: clock.Fixed{T: now},
		Consents: f.consents, Slots: f.slots, Patients: f.patients, Clinics: f.clinics, Roles: f.roles, Jobs: f.jobs,
		Tokens: f.tokens, LinkURL: linkURL, Stats: f.stats, Reminders: f.reminders, Rules: f.rules, Services: services, Stops: f.stops,
	})
	return f, services
}

func TestBook_KeepsTheServiceTheClinicOffersInTheSlotsMode(t *testing.T) {
	f, services := serviceFixture()
	got, err := f.svc.Book(t.Context(), me, slotFree, serviceVideo)
	if err != nil || got.ServiceID != serviceVideo || f.repo.appts[0].ServiceID != serviceVideo {
		t.Fatalf("Book() = %+v, %v; want the appointment stored with the service", got, err)
	}
	if len(services.asked) != 1 || services.asked[0] != clinicA+" video" {
		t.Errorf("asked %v, want clinic A's video services", services.asked)
	}
}

func TestBook_WithoutAServiceAsksForNone(t *testing.T) {
	f, services := serviceFixture()
	got, err := f.svc.Book(t.Context(), me, slotFree, "")
	if err != nil || got.ServiceID != "" || len(services.asked) != 0 {
		t.Fatalf("Book() = %+v, %v, asked %v; want no service and no question", got, err, services.asked)
	}
}

func TestBook_RefusesAndAuditsAServiceNotOfferedInTheSlotsMode(t *testing.T) {
	for name, id := range map[string]string{
		"in person only, for a video slot": serviceInPerson,
		"not on the clinic's list":         "c1000000-0000-4000-8000-0000000000ff",
		"not a UUID":                       "x",
	} {
		f, _ := serviceFixture()
		_, err := f.svc.Book(t.Context(), me, slotFree, id)
		if !errors.Is(err, appointment.ErrInvalidService) || len(f.repo.appts) != 0 || len(f.slots.held) != 0 {
			t.Errorf("%s: Book() = %v, stored %d, held %v; want ErrInvalidService and nothing booked", name, err, len(f.repo.appts), f.slots.held)
			continue
		}
		checkEvents(t, name, f.audit, event(me, auditevent.CodeCreate, "", "", auditevent.OutcomeMinorFailure))
	}
}

func TestBook_FailsWhenThePriceListCannotBeRead(t *testing.T) {
	f, services := serviceFixture()
	services.err = errBackend
	if _, err := f.svc.Book(t.Context(), me, slotFree, serviceVideo); !errors.Is(err, errBackend) || len(f.repo.appts) != 0 {
		t.Fatalf("Book() = %v, stored %d; want the backend's error and nothing booked", err, len(f.repo.appts))
	}
}

func TestVisits_GivesTheClinicsAppointmentsInTheRangeInItsScope(t *testing.T) {
	inRange := booked(apptA, patientA, roleA1, clinicA, start)
	inRange.Mode = schedule.ModeInPerson
	f := newFixture(inRange, booked(apptB, patientA, roleA1, clinicA, start.Add(48*time.Hour)), booked(apptX, patientA, roleB, clinicB, start))
	got, err := f.svc.Visits(t.Context(), clinicA, start, start.Add(24*time.Hour))
	want := []appointment.Visit{{ID: apptA, PractitionerRoleID: roleA1, Mode: schedule.ModeInPerson, Status: appointment.StatusBooked}}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("Visits() = %+v, %v; want %+v", got, err, want)
	}
	if f.repo.clinic != clinicA || len(f.audit.events) != 0 {
		t.Errorf("read in clinic %q with %d audit events, want clinic A's scope and none (the report audits)", f.repo.clinic, len(f.audit.events))
	}
	f.repo.err = errBackend
	if _, err := f.svc.Visits(t.Context(), clinicA, start, start.Add(time.Hour)); !errors.Is(err, errBackend) {
		t.Errorf("Visits() = %v, want the backend's error", err)
	}
}
