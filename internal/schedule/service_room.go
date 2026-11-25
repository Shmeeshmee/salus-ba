package schedule

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/Shmeeshmee/salus/internal/location"
)

// Rooms locks a room of a clinic for a rule that names it (location.Service): two saves that name one room run one
// after the other, so neither misses the other's hours.
type Rooms interface {
	LockRoom(ctx context.Context, org, id string) (location.Room, error)
}

// roomsFree returns why the role's rules may not hold their rooms (U3 of docs/clinic-cms/WAVE6.md, RM1), or nil:
// ErrInvalidAvailability for a room that is not an active room of the clinic, ErrResourceBusy when another
// role's in-person rule holds one of them at an overlapping time on the same day. In the save's transaction, in
// the clinic's scope; err is any other error.
func (s *Service) roomsFree(ctx context.Context, c Caller, roleID string, rules []Rule) (failed, err error) {
	var rooms []string
	for i, r := range rules {
		if r.RoomID != "" {
			rules[i].RoomID = strings.ToLower(r.RoomID) // one spelling, as the database prints a uuid
			rooms = append(rooms, rules[i].RoomID)
		}
	}
	if len(rooms) == 0 {
		return nil, nil
	}
	if s.Rooms == nil {
		return ErrInvalidAvailability, nil
	}
	slices.Sort(rooms) // locked in one order, so two saves never wait for each other's
	for _, id := range slices.Compact(rooms) {
		room, err := s.Rooms.LockRoom(ctx, c.OrganizationID, id)
		switch {
		case errors.Is(err, location.ErrNotFound), err == nil && !room.Active:
			return ErrInvalidAvailability, nil
		case err != nil:
			return nil, err
		}
	}
	others, err := s.Repo.Listed(ctx)
	if err != nil {
		return nil, err
	}
	for _, o := range others {
		if o.PractitionerRoleID == roleID || o.OrganizationID != c.OrganizationID {
			continue
		}
		for _, held := range o.Rules {
			if held.RoomID != "" && slices.ContainsFunc(rules, func(r Rule) bool { return clash(r, held) }) {
				return ErrResourceBusy, nil
			}
		}
	}
	return nil, nil
}

// clash reports whether two rules hold one room at an overlapping time on the same day.
func clash(a, b Rule) bool {
	return a.RoomID == b.RoomID && a.Day == b.Day && a.Start < b.End && b.Start < a.End
}
