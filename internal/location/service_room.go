package location

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// maxName is the longest room or device name (api/openapi.yaml SaveRoomRequest).
const maxName = 200

// Rooms returns the rooms of a location of the staff's clinic, by name (RM1): any of its staff, the desk included.
// ErrNotFound for a location that is not the clinic's. Not audited: a room is not personal data.
func (s *Service) Rooms(ctx context.Context, st Staff, locationID string) ([]Room, error) {
	var out []Room
	err := s.inClinic(ctx, st, func(ctx context.Context) (err error) {
		out, err = s.p.Repo.Rooms(ctx, locationID)
		return err
	})
	return out, err
}

// Devices returns the devices of a location of the staff's clinic, by name (RM1), as Rooms.
func (s *Service) Devices(ctx context.Context, st Staff, locationID string) ([]Device, error) {
	var out []Device
	err := s.inClinic(ctx, st, func(ctx context.Context) (err error) {
		out, err = s.p.Repo.Devices(ctx, locationID)
		return err
	})
	return out, err
}

// LockRoom returns the room id of the clinic org, locked until the transaction in ctx ends, for an availability rule
// that names it (schedule.Service, RM1); ErrNotFound for none. A system step of the save, not audited.
func (s *Service) LockRoom(ctx context.Context, org, id string) (Room, error) {
	var out Room
	err := s.inClinic(ctx, Staff{OrganizationID: org}, func(ctx context.Context) (err error) {
		out, err = s.p.Repo.LockRoom(ctx, id)
		return err
	})
	return out, err
}

// Device returns the device id of the clinic org, held until the transaction in ctx ends, for a price-list service
// that names it (chargeitemdefinition.Service, W6); ErrNotFound for none. A system step of the save, not audited.
func (s *Service) Device(ctx context.Context, org, id string) (Device, error) {
	var out Device
	err := s.inClinic(ctx, Staff{OrganizationID: org}, func(ctx context.Context) (err error) {
		out, err = s.p.Repo.Device(ctx, id)
		return err
	})
	return out, err
}

// SaveRoom adds a room to a location of the admin's clinic (no id), or changes one at r.VersionID, the version the
// caller read (RM1). Audited, also when refused: ErrForbidden to other staff, ErrNotFound for a location or room that
// is not the clinic's, ErrVersionConflict for another version; ErrInvalid for an empty or too long name. A room never
// moves to another location.
func (s *Service) SaveRoom(ctx context.Context, st Staff, r Room) (Room, error) {
	return s.save(ctx, st, "Location/", r, s.p.Repo.SaveRoom)
}

// SaveDevice adds a device to a location of the admin's clinic, or changes one, as SaveRoom (RM1).
func (s *Service) SaveDevice(ctx context.Context, st Staff, d Device) (Device, error) {
	out, err := s.save(ctx, st, "Device/", Room(d), func(ctx context.Context, r Room) (Room, bool, error) {
		d, ok, err := s.p.Repo.SaveDevice(ctx, Device(r))
		return Room(d), ok, err
	})
	return Device(out), err
}

func (s *Service) inClinic(ctx context.Context, st Staff, read func(ctx context.Context) error) error {
	if st.OrganizationID == "" {
		return ErrNotFound
	}
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, st.OrganizationID); err != nil {
			return err
		}
		return read(ctx)
	})
	if err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("location: read: %w", err)
	}
	return err
}

// save is SaveRoom and SaveDevice (a device has a room's fields): the admin's check, the name, the write and its
// audit in one transaction; entity is the audit's reference prefix.
func (s *Service) save(ctx context.Context, st Staff, entity string, r Room, write func(context.Context, Room) (Room, bool, error)) (Room, error) {
	interaction, action := auditevent.CodeCreate, auditevent.ActionCreate
	if r.ID != "" {
		interaction, action = auditevent.CodeUpdate, auditevent.ActionUpdate
	}
	switch {
	case st.OrganizationID == "":
		return Room{}, s.refused(ctx, st, interaction, action, "", ErrNotFound)
	case !st.Admin:
		return Room{}, s.refused(ctx, st, interaction, action, "", ErrForbidden)
	}
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" || utf8.RuneCountInString(r.Name) > maxName {
		return Room{}, ErrInvalid
	}
	if r.ID != "" {
		r.VersionID++ // the repository updates the row at r.VersionID-1
	}
	var out Room
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetOrganization(ctx, st.OrganizationID); err != nil {
			return err
		}
		saved, ok, err := write(ctx, r)
		switch {
		case err != nil:
			return err
		case !ok:
			return ErrVersionConflict
		}
		out = saved
		return s.p.Audit.Record(ctx, event(interaction, action, st, entity+out.ID, auditevent.OutcomeSuccess))
	})
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrVersionConflict): // the write's transaction rolled back
		return Room{}, s.refused(ctx, st, interaction, action, "", err)
	case err != nil:
		return Room{}, fmt.Errorf("location: save: %w", err)
	}
	return out, nil
}
