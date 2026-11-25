package location_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

func newRoomService(repo *fakeRepo, audit *fakeAudit) *location.Service {
	return location.NewService(location.Ports{
		Repo: repo, Tx: fakeTransactor{}, Audit: audit, Clock: clock.Fixed{T: time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)},
	})
}

func TestService_SaveRoom_TheAdminAddsAndChangesARoomAudited(t *testing.T) {
	t.Parallel()
	repo, audit := &fakeRepo{}, &fakeAudit{}
	s := newRoomService(repo, audit)
	added, err := s.SaveRoom(t.Context(), roomAdmin, location.Room{LocationID: siteA, Name: "  Room 2 ", Active: true})
	if err != nil || added.ID == "" || added.Name != "Room 2" || added.VersionID != 1 || !added.Active {
		t.Fatalf("add: %+v, %v", added, err)
	}
	changed, err := s.SaveRoom(t.Context(), roomAdmin, location.Room{ID: added.ID, LocationID: siteB, Name: "Room 3", VersionID: 1})
	if err != nil || changed.Name != "Room 3" || changed.VersionID != 2 || changed.Active || changed.LocationID != siteA {
		t.Fatalf("change: %+v, %v (a room never moves to another location)", changed, err)
	}
	if repo.outside {
		t.Error("a repository call ran outside a transaction")
	}
	want := []struct {
		code    string
		outcome auditevent.Outcome
	}{{auditevent.CodeCreate, auditevent.OutcomeSuccess}, {auditevent.CodeUpdate, auditevent.OutcomeSuccess}}
	if len(audit.events) != len(want) {
		t.Fatalf("audit: %d events, want %d", len(audit.events), len(want))
	}
	for i, w := range want {
		e := audit.events[i]
		if e.Subtype.Code != w.code || e.Outcome != w.outcome || e.EntityReference != "Location/"+added.ID ||
			e.OrganizationID != clinicA || e.AgentAccountID != adminAccID {
			t.Errorf("audit %d: %+v", i, e)
		}
	}
}

func TestService_SaveRoom_Refusals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		want    error
		name    string
		staff   location.Staff
		room    location.Room
		audited bool
	}{
		{name: "staff who are not the admin", staff: roomStaff, room: location.Room{LocationID: siteA, Name: "Room 1"}, want: location.ErrForbidden, audited: true},
		{name: "no clinic", staff: noClinic, room: location.Room{LocationID: siteA, Name: "Room 1"}, want: location.ErrNotFound, audited: true},
		{name: "another clinic's location", staff: roomAdmin, room: location.Room{LocationID: siteB, Name: "Room 1"}, want: location.ErrNotFound, audited: true},
		{name: "an unknown room", staff: roomAdmin, room: location.Room{ID: "a4000000-0000-4000-8000-0000000000ff", Name: "Room 1", VersionID: 1}, want: location.ErrNotFound, audited: true},
		{name: "a version that moved on", staff: roomAdmin, room: location.Room{ID: "a4000000-0000-4000-8000-000000000000", Name: "Room 1", VersionID: 7}, want: location.ErrVersionConflict, audited: true},
		{name: "an empty name", staff: roomAdmin, room: location.Room{LocationID: siteA, Name: "  "}, want: location.ErrInvalid},
		{name: "a name too long", staff: roomAdmin, room: location.Room{LocationID: siteA, Name: string(make([]byte, 201))}, want: location.ErrInvalid},
		{name: "a NUL in the name", staff: roomAdmin, room: location.Room{LocationID: siteA, Name: "Room\x001"}, want: location.ErrInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo, audit := &fakeRepo{rooms: []location.Room{{ID: "a4000000-0000-4000-8000-000000000000", LocationID: siteA, Name: "Room 0", VersionID: 1}}}, &fakeAudit{}
			_, err := newRoomService(repo, audit).SaveRoom(t.Context(), tc.staff, tc.room)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err %v, want %v", err, tc.want)
			}
			if got := len(audit.events) == 1 && audit.events[0].Outcome == auditevent.OutcomeMinorFailure; got != tc.audited {
				t.Errorf("audited as refused: %v, want %v (%+v)", got, tc.audited, audit.events)
			}
		})
	}
}

func TestService_Rooms_TheClinicsStaffReadALocationsRooms(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{rooms: []location.Room{{ID: "r2", LocationID: siteA, Name: "Room B"}, {ID: "r1", LocationID: siteA, Name: "Room A"}}}
	s := newRoomService(repo, &fakeAudit{})
	got, err := s.Rooms(t.Context(), roomStaff, siteA)
	if err != nil || len(got) != 2 || got[0].Name != "Room A" {
		t.Fatalf("rooms: %+v, %v", got, err)
	}
	if _, err := s.Rooms(t.Context(), roomStaff, siteB); !errors.Is(err, location.ErrNotFound) {
		t.Errorf("another clinic's location: %v, want ErrNotFound", err)
	}
	if _, err := s.Rooms(t.Context(), noClinic, siteA); !errors.Is(err, location.ErrNotFound) {
		t.Errorf("no clinic: %v, want ErrNotFound", err)
	}
	repo.err = errBroken
	if _, err := s.Rooms(t.Context(), roomStaff, siteA); !errors.Is(err, errBroken) {
		t.Errorf("a broken repository: %v", err)
	}
}

func TestService_Devices_AsRooms(t *testing.T) {
	t.Parallel()
	repo, audit := &fakeRepo{}, &fakeAudit{}
	s := newRoomService(repo, audit)
	d, err := s.SaveDevice(t.Context(), roomAdmin, location.Device{LocationID: siteA, Name: " Ultrasound ", Active: true})
	if err != nil || d.Name != "Ultrasound" || d.VersionID != 1 {
		t.Fatalf("add: %+v, %v", d, err)
	}
	if len(audit.events) != 1 || audit.events[0].EntityReference != "Device/"+d.ID {
		t.Errorf("audit: %+v", audit.events)
	}
	got, err := s.Devices(t.Context(), roomStaff, siteA)
	if err != nil || len(got) != 1 || got[0].ID != d.ID {
		t.Fatalf("devices: %+v, %v", got, err)
	}
	if _, err := s.SaveDevice(t.Context(), roomStaff, location.Device{LocationID: siteA, Name: "ECG"}); !errors.Is(err, location.ErrForbidden) {
		t.Errorf("staff: %v, want ErrForbidden", err)
	}
	if _, err := s.Devices(t.Context(), roomStaff, siteB); !errors.Is(err, location.ErrNotFound) {
		t.Errorf("another clinic's location: %v", err)
	}
}

func TestService_LockRoom_TheClinicsRoomOnly(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{rooms: []location.Room{{ID: "r1", LocationID: siteA, Name: "Room 1", Active: true}}}
	s := newRoomService(repo, &fakeAudit{})
	if got, err := s.LockRoom(t.Context(), clinicA, "r1"); err != nil || got.ID != "r1" || repo.outside {
		t.Fatalf("LockRoom() = %+v, %v", got, err)
	}
	if _, err := s.LockRoom(t.Context(), "b0000000-0000-4000-8000-00000000000b", "r1"); !errors.Is(err, location.ErrNotFound) {
		t.Errorf("another clinic: %v, want ErrNotFound", err)
	}
	if _, err := s.LockRoom(t.Context(), "", "r1"); !errors.Is(err, location.ErrNotFound) {
		t.Errorf("no clinic: %v, want ErrNotFound", err)
	}
}

func TestService_Device_TheClinicsDeviceOnly(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{devices: []location.Device{{ID: "d1", LocationID: siteA, Name: "Device 1"}}}
	s := newRoomService(repo, &fakeAudit{})
	if got, err := s.Device(t.Context(), clinicA, "d1"); err != nil || got.ID != "d1" || got.Active || repo.outside {
		t.Fatalf("Device() = %+v, %v; want device 1 as stored, read in a transaction", got, err)
	}
	if _, err := s.Device(t.Context(), "b0000000-0000-4000-8000-00000000000b", "d1"); !errors.Is(err, location.ErrNotFound) {
		t.Errorf("another clinic: %v, want ErrNotFound", err)
	}
	if _, err := s.Device(t.Context(), "", "d1"); !errors.Is(err, location.ErrNotFound) {
		t.Errorf("no clinic: %v, want ErrNotFound", err)
	}
}
