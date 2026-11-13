package slot

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/schedule"
)

// The waitlist's moves of a slot (docs/plans/appointments.md, SAL-35; ADR 0015 "The waitlist hold lives in Postgres"):
// a freed slot is held for one waitlisted patient's offer (busy-tentative), which the public directory does not show,
// then booked by the acceptance (busy) or given back (free). Each runs in the caller's transaction, in the slot's
// clinic, and moves the slot only from the status before it.

// ForWaitlist returns the slot id of the clinic org with the practitioner role of its schedule, whatever its status,
// locked until the transaction in ctx ends, so the jobs that hand one slot on run one after the other; ErrUnavailable
// when the clinic has no such slot.
func (s *Service) ForWaitlist(ctx context.Context, org, id string) (Bookable, error) {
	var out Bookable
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		sl, err := s.repo.Lock(ctx, id)
		if err != nil {
			return err
		}
		if sl.OrganizationID != org { // not on row-level security alone
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
		return Bookable{}, fmt.Errorf("slot: for waitlist: %w", err)
	}
	return out, nil
}

// Offer holds the free slot id of the clinic org for an offer (free to busy-tentative) and returns it; ErrUnavailable
// when it is not free (booked or offered meanwhile) or not the clinic's.
func (s *Service) Offer(ctx context.Context, org, id string) (Slot, error) {
	return s.transition(ctx, org, id, StatusFree, StatusTentative)
}

// TakeOffered books the offered slot id of the clinic org for the offer's acceptance (busy-tentative to busy) and
// returns it; ErrUnavailable when it is not held for an offer.
func (s *Service) TakeOffered(ctx context.Context, org, id string) (Slot, error) {
	return s.transition(ctx, org, id, StatusTentative, StatusBusy)
}

// Unoffer gives the offered slot id of the clinic org back (busy-tentative to free) when no offer holds it any more,
// and reports whether it did. A started slot is freed too: no offer can hold it, and retention deletes free slots only.
func (s *Service) Unoffer(ctx context.Context, org, id string) (bool, error) {
	_, err := s.transition(ctx, org, id, StatusTentative, StatusFree)
	if errors.Is(err, ErrUnavailable) {
		return false, nil
	}
	return err == nil, err
}

func (s *Service) transition(ctx context.Context, org, id string, from, to Status) (Slot, error) {
	var out Slot
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		var err error
		out, err = s.repo.Transition(ctx, id, from, to, s.clock.Now())
		return err
	})
	switch {
	case errors.Is(err, ErrUnavailable):
		return Slot{}, ErrUnavailable
	case err != nil:
		return Slot{}, fmt.Errorf("slot: %s to %s: %w", from, to, err)
	}
	return out, nil
}
