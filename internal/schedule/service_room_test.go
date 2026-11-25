package schedule_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

var errBroken = errors.New("fake: broken")

const (
	room1        = "a6000000-0000-4000-8000-0000000000a1"
	room2        = "a6000000-0000-4000-8000-0000000000a2"
	roomInactive = "a6000000-0000-4000-8000-0000000000a3"
	roomAtSite2  = "a6000000-0000-4000-8000-0000000000a4" // a room of clinic A's second location
)

// fakeRooms are clinic A's rooms, as location.Service.LockRoom gives them; it remembers what it locked.
type fakeRooms struct {
	*log
	err    error
	locked []string
}

func (r *fakeRooms) LockRoom(ctx context.Context, org, id string) (location.Room, error) {
	r.seen(ctx, "LockRoom")
	r.locked = append(r.locked, id)
	switch {
	case r.err != nil:
		return location.Room{}, r.err
	case org != clinicA:
		return location.Room{}, location.ErrNotFound
	case id == room1 || id == room2:
		return location.Room{ID: id, LocationID: siteA1, Name: "Room", Active: true}, nil
	case id == roomInactive:
		return location.Room{ID: id, LocationID: siteA1, Name: "Room", Active: false}, nil
	case id == roomAtSite2:
		return location.Room{ID: id, LocationID: siteA2, Name: "Room", Active: true}, nil
	}
	return location.Room{}, location.ErrNotFound
}

// roomFixture is newFixture with rooms; role A2's schedule holds Tuesday 08:00 to 10:00 in person in room 1.
func roomFixture() (fixture, *fakeRooms) {
	f := newFixture()
	f.repo.schedules[0].Rules = []schedule.Rule{{Day: 2, Start: "08:00", End: "10:00", Mode: schedule.ModeInPerson, RoomID: room1}}
	rooms := &fakeRooms{log: f.log}
	f.svc = schedule.NewService(schedule.Ports{
		Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{}, Clock: clock.Fixed{T: now}, Roles: f.roles, Jobs: f.jobs,
		Clinics: f.clinics, Sites: f.sites, Rooms: rooms,
	})
	return f, rooms
}

func inPerson(day int, start, end, room string) schedule.Availability {
	return schedule.Availability{SlotMinutes: 30, Rules: []schedule.Rule{{Day: day, Start: start, End: end, Mode: schedule.ModeInPerson, RoomID: room}}}
}

func TestSaveAvailability_ARoomHoldsOnePractitionersHoursAtATime(t *testing.T) {
	t.Parallel()
	cases := []struct {
		want error
		name string
		in   schedule.Availability
	}{
		{name: "the same room at an overlapping time", in: inPerson(2, "09:30", "11:00", room1), want: schedule.ErrResourceBusy},
		{name: "the same room spelt in capitals", in: inPerson(2, "09:30", "11:00", strings.ToUpper(room1)), want: schedule.ErrResourceBusy},
		{name: "the same room on another day", in: inPerson(3, "08:00", "10:00", room1)},
		{name: "the same room back to back", in: inPerson(2, "10:00", "12:00", room1)},
		{name: "another room at the same time", in: inPerson(2, "08:00", "10:00", room2)},
		{name: "no room at the same time", in: inPerson(2, "08:00", "10:00", "")},
		{name: "a room that is not the clinic's", in: inPerson(2, "12:00", "13:00", "a6000000-0000-4000-8000-0000000000ff"), want: schedule.ErrInvalidAvailability},
		{name: "a room out of use", in: inPerson(2, "12:00", "13:00", roomInactive), want: schedule.ErrInvalidAvailability},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, _ := roomFixture()
			got, err := f.svc.SaveAvailability(t.Context(), practitionerA1, roleA1, tc.in, 0)
			if !errors.Is(err, tc.want) {
				t.Fatalf("SaveAvailability() = %+v, %v; want %v", got, err, tc.want)
			}
			if tc.want == nil && (len(got.Rules) != 1 || got.Rules[0].RoomID != strings.ToLower(tc.in.Rules[0].RoomID)) {
				t.Errorf("rules %+v, want the room kept", got.Rules)
			}
			if tc.want != nil {
				want := []auditevent.AuditEvent{event(practitionerA1, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)}
				if !slices.Equal(f.audit.events, want) || len(f.jobs.jobs) != 0 {
					t.Errorf("audit %+v, jobs %v; want one refusal and no generation", f.audit.events, f.jobs.jobs)
				}
			}
		})
	}
}

func TestSaveAvailability_ChangingItsOwnRoomHoursIsNoClash(t *testing.T) {
	t.Parallel()
	f, rooms := roomFixture()
	if _, err := f.svc.SaveAvailability(t.Context(), adminA, roleA2, inPerson(2, "09:00", "11:00", room1), 3); err != nil {
		t.Fatalf("role A2 moving its own hours in room 1: %v", err)
	}
	if !slices.Equal(rooms.locked, []string{room1}) || !slices.Contains(f.calls, "LockRoom") {
		t.Errorf("locked %v after %v, want room 1 locked in the save's transaction", rooms.locked, f.calls)
	}
}

func TestSaveAvailability_RoomRefusals(t *testing.T) {
	t.Parallel()
	f, rooms := roomFixture()
	video := schedule.Availability{SlotMinutes: 30, Rules: []schedule.Rule{{Day: 1, Start: "09:00", End: "10:00", Mode: schedule.ModeVideo, RoomID: room1}}}
	if _, err := f.svc.SaveAvailability(t.Context(), practitionerA1, roleA1, video, 0); !errors.Is(err, schedule.ErrInvalidAvailability) {
		t.Errorf("a video rule with a room: %v, want ErrInvalidAvailability", err)
	}
	rooms.err = errBroken
	if _, err := f.svc.SaveAvailability(t.Context(), practitionerA1, roleA1, inPerson(1, "09:00", "10:00", room2), 0); !errors.Is(err, errBroken) {
		t.Errorf("a broken lookup: %v", err)
	}
	none := newFixture() // a service built without rooms (the worker's) refuses a rule that names one
	if _, err := none.svc.SaveAvailability(t.Context(), practitionerA1, roleA1, inPerson(1, "09:00", "10:00", room2), 0); !errors.Is(err, schedule.ErrInvalidAvailability) {
		t.Errorf("no rooms port: %v, want ErrInvalidAvailability", err)
	}
}

// A rule's room is a room of the rule's location (LC1's placed sets it): a room of another location is refused.
func TestSaveAvailability_ARulesRoomIsOfItsLocation(t *testing.T) {
	t.Parallel()
	at := func(site, room string) schedule.Availability {
		return schedule.Availability{SlotMinutes: 30, Rules: []schedule.Rule{
			{Day: 1, Start: "09:00", End: "10:00", Mode: schedule.ModeInPerson, LocationID: site, RoomID: room},
		}}
	}
	cases := []struct {
		want error
		name string
		in   schedule.Availability
	}{
		{name: "the clinic's only location, its room", in: at("", room1)},
		{name: "the clinic's only location, a room of another", in: at("", roomAtSite2), want: schedule.ErrInvalidAvailability},
		{name: "the second location, its room", in: at(siteA2, roomAtSite2)},
		{name: "the second location, a room of the first", in: at(siteA2, room1), want: schedule.ErrInvalidAvailability},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, _ := roomFixture()
			if tc.in.Rules[0].LocationID == siteA2 {
				f.sites.rows = append(f.sites.rows, location.Location{ID: siteA2, OrganizationID: clinicA, Name: "Branch", Active: true})
			}
			got, err := f.svc.SaveAvailability(t.Context(), practitionerA1, roleA1, tc.in, 0)
			if !errors.Is(err, tc.want) {
				t.Fatalf("SaveAvailability() = %+v, %v; want %v", got, err, tc.want)
			}
			if tc.want != nil && len(f.jobs.jobs) != 0 {
				t.Errorf("jobs %v, want no generation after a refusal", f.jobs.jobs)
			}
		})
	}
}

// Rules of an inactive role stop holding their room: role A2, taken out of use, no longer refuses its hours in room 1.
func TestSaveAvailability_AnInactiveRolesRulesHoldNoRoom(t *testing.T) {
	t.Parallel()
	f, _ := roomFixture()
	f.roles.inactive = roleA2
	if _, err := f.svc.SaveAvailability(t.Context(), practitionerA1, roleA1, inPerson(2, "09:30", "11:00", room1), 0); err != nil {
		t.Fatalf("room 1 held only by an inactive role: %v, want saved", err)
	}
	f, _ = roomFixture()
	f.roles.failing = roleA2
	if _, err := f.svc.SaveAvailability(t.Context(), practitionerA1, roleA1, inPerson(2, "09:30", "11:00", room1), 0); err == nil {
		t.Errorf("a failing role lookup saved, want the error")
	}
}
