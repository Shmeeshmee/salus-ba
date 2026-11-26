package schedule

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// Rooms locks a room of a clinic for a rule that names it (location.Service): two saves that name one room run one
// after the other, so neither misses the other's hours.
type Rooms interface {
	LockRoom(ctx context.Context, org, id string) (location.Room, error)
}

// roomsFree returns why the role's rules may not hold their rooms (U3 of docs/clinic-cms/WAVE6.md, RM1), or nil:
// ErrInvalidAvailability for a room that is not an active room of the clinic or not of the rule's location (set by
// placed), ErrResourceBusy when another active role's in-person rule holds one of them at an overlapping time on the
// same day. In the save's transaction, in the clinic's scope; err is any other error.
func (s *Service) roomsFree(ctx context.Context, c Caller, roleID string, rules []Rule) (failed, err error) {
	var ids []string
	for i, r := range rules {
		if r.RoomID != "" {
			rules[i].RoomID = strings.ToLower(r.RoomID) // one spelling, as the database prints a uuid
			ids = append(ids, rules[i].RoomID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if s.Rooms == nil {
		return ErrInvalidAvailability, nil
	}
	slices.Sort(ids) // locked in one order, so two saves never wait for each other's
	rooms := make(map[string]location.Room, len(ids))
	for _, id := range slices.Compact(ids) {
		room, err := s.Rooms.LockRoom(ctx, c.OrganizationID, id)
		switch {
		case errors.Is(err, location.ErrNotFound), err == nil && !room.Active:
			return ErrInvalidAvailability, nil
		case err != nil:
			return nil, err
		}
		rooms[id] = room
	}
	for _, r := range rules {
		if r.RoomID != "" && rooms[r.RoomID].LocationID != r.LocationID {
			return ErrInvalidAvailability, nil
		}
	}
	return s.roomsHeld(ctx, c.OrganizationID, roleID, rules)
}

// roomsHeld returns ErrResourceBusy when an in-person rule of another active role of the clinic org holds a room of
// rules at an overlapping time on the same day, or nil; err is any other error. A role taken out of use holds no room.
func (s *Service) roomsHeld(ctx context.Context, org, roleID string, rules []Rule) (failed, err error) {
	busy, err := s.roomsBusy(ctx, org, roleID, rules)
	if err != nil || !slices.Contains(busy, true) {
		return nil, err
	}
	return ErrResourceBusy, nil
}

// markRoomsBusy sets RoomBusy on each rule of sc whose room another active role of its clinic holds at an overlapping
// time on the same day: one of the two gets no slots there (a role brought back into use is not re-checked, RM1).
func (s *Service) markRoomsBusy(ctx context.Context, sc Schedule) (Schedule, error) {
	busy, err := s.roomsBusy(ctx, sc.OrganizationID, sc.PractitionerRoleID, sc.Rules)
	if err != nil {
		return Schedule{}, err
	}
	sc.Rules = slices.Clone(sc.Rules)
	for i := range sc.Rules {
		sc.Rules[i].RoomBusy = busy[i]
	}
	return sc, nil
}

// roomsBusy reports for each of rules whether an in-person rule of another active role of the clinic org holds its room
// at an overlapping time on the same day. Read in the transaction's clinic scope.
func (s *Service) roomsBusy(ctx context.Context, org, roleID string, rules []Rule) ([]bool, error) {
	busy := make([]bool, len(rules))
	if !slices.ContainsFunc(rules, func(r Rule) bool { return r.RoomID != "" }) {
		return busy, nil
	}
	others, err := s.Repo.Listed(ctx)
	if err != nil {
		return nil, err
	}
	for _, o := range others {
		if o.PractitionerRoleID == roleID || o.OrganizationID != org {
			continue
		}
		hit := make([]bool, len(rules))
		for i, r := range rules {
			hit[i] = r.RoomID != "" && slices.ContainsFunc(o.Rules, func(held Rule) bool { return clash(r, held) })
		}
		if !slices.Contains(hit, true) {
			continue
		}
		role, err := s.Roles.Lookup(ctx, org, o.PractitionerRoleID)
		switch {
		case errors.Is(err, practitionerrole.ErrNotFound):
		case err != nil:
			return nil, err
		case role.Active:
			for i := range busy {
				busy[i] = busy[i] || hit[i]
			}
		}
	}
	return busy, nil
}

// clash reports whether two rules hold one room at an overlapping time on the same day.
func clash(a, b Rule) bool {
	return a.RoomID == b.RoomID && a.Day == b.Day && a.Start < b.End && b.Start < a.End
}
