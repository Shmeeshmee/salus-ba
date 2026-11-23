package slot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/schedule"
)

// SearchFor is Search for a caller who is staff of the clinic org (empty for anyone else): a schedule of that clinic is
// searched in its own scope, so its staff see its free times whether or not it is listed (the front desk's walk-in,
// docs/clinic-cms F3); any other schedule as the public directory shows it. Not audited, like Search.
func (s *Service) SearchFor(ctx context.Context, org, scheduleID string, from, to time.Time) ([]Slot, error) {
	if org == "" {
		return s.Search(ctx, scheduleID, from, to)
	}
	if now := s.clock.Now(); from.Before(now) {
		from = now
	}
	if !to.After(from) {
		return nil, nil
	}
	var out []Slot
	var public bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		_, err := s.schedules.Lookup(ctx, org, scheduleID)
		if errors.Is(err, schedule.ErrNotFound) {
			public = true
			return nil
		} else if err != nil {
			return err
		}
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		out, err = s.repo.Search(ctx, scheduleID, from, to)
		return err
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("slot: search for clinic: %w", err)
	case public: // another clinic's schedule: as the directory shows it
		return s.Search(ctx, scheduleID, from, to)
	}
	return out, nil
}

// BookableInClinic is Bookable for the clinic org's own staff booking (docs/clinic-cms F2): the free slot id of one of
// the clinic's schedules, with its practitioner role, read in the clinic's scope whether or not the clinic is listed,
// in a transaction of its own. ErrUnavailable for a slot that is busy, another clinic's or unknown. Whether the clinic
// takes bookings, and the role is active, are the booking's to check. Not audited, like Bookable.
func (s *Service) BookableInClinic(ctx context.Context, org, id string) (Bookable, error) {
	var out Bookable
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		sl, err := s.repo.ByID(ctx, id)
		if err != nil {
			return err
		}
		if sl.Status != StatusFree || sl.OrganizationID != org { // not on row-level security alone
			return ErrUnavailable
		}
		sc, err := s.schedules.Lookup(ctx, org, sl.ScheduleID)
		if errors.Is(err, schedule.ErrNotFound) {
			return ErrUnavailable
		}
		out = Bookable{Slot: sl, PractitionerRoleID: sc.PractitionerRoleID}
		return err
	})
	switch {
	case errors.Is(err, ErrUnavailable):
		return Bookable{}, ErrUnavailable
	case err != nil:
		return Bookable{}, fmt.Errorf("slot: bookable in clinic: %w", err)
	}
	return out, nil
}
