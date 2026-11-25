package appointment

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/slot"
)

// The visits of a treatment plan (docs/clinic-cms/WAVE6.md U6, package TP1): careplan.Service books, reads and
// cancels them inside its own transaction, authorizes its caller for the plan and audits the plan.

// BookPlan books the slot of every step for the patient patientID as staff c, in one transaction (the caller's, when
// ctx carries one): every visit, or none. Each step is checked as BookForPatient checks a booking and each visit is
// audited as booked; a refusal (ErrForbidden, ErrNotFound, ErrSlotUnavailable, ErrInvalidService,
// ErrConsentRequired) is returned unaudited, for the caller to audit after its rollback.
func (s *Service) BookPlan(ctx context.Context, c Caller, patientID string, steps []StaffBooking) ([]Appointment, error) {
	if c.patient() {
		return nil, ErrForbidden
	}
	if s.Stops == nil {
		return nil, errors.New("appointment: book plan: no booking stops")
	}
	slots, err := s.planSlots(ctx, c, steps)
	if err != nil {
		return nil, fmt.Errorf("appointment: book plan: %w", err)
	}
	var out []Appointment
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		now := s.Clock.Now()
		for i, b := range steps {
			b.PatientID = patientID
			a, refused, err := s.bookFor(ctx, c, b, slots[i], now)
			if err != nil || refused != nil {
				return errors.Join(err, refused)
			}
			if err := s.record(ctx, c, auditevent.CodeCreate, "Appointment", a.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
			out = append(out, a)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("appointment: book plan: %w", err)
	}
	return out, nil
}

// planSlots resolves the slot of every step in the clinic's own scope, as BookForPatient does: ErrSlotUnavailable
// while the clinic's bookings are stopped or for a slot that is not its, free and ahead, ErrInvalidService for a
// service it does not offer in the slot's mode.
func (s *Service) planSlots(ctx context.Context, c Caller, steps []StaffBooking) ([]slot.Bookable, error) {
	if stopped, err := s.Stops.BookingsStopped(ctx, c.OrganizationID); err != nil || stopped {
		return nil, errors.Join(err, stoppedErr(stopped))
	}
	now := s.Clock.Now()
	out := make([]slot.Bookable, len(steps))
	for i, b := range steps {
		sl, err := s.Slots.BookableInClinic(ctx, c.OrganizationID, b.SlotID)
		switch {
		case errors.Is(err, slot.ErrUnavailable), err == nil && !sl.Start.After(now):
			return nil, ErrSlotUnavailable
		case err != nil:
			return nil, err
		}
		if offered, err := s.offersInClinic(ctx, sl, b.ServiceID); err != nil || !offered {
			return nil, errors.Join(err, invalidServiceErr(offered))
		}
		out[i] = sl
	}
	return out, nil
}

func stoppedErr(stopped bool) error {
	if stopped {
		return ErrSlotUnavailable
	}
	return nil
}

func invalidServiceErr(offered bool) error {
	if offered {
		return nil
	}
	return ErrInvalidService
}

// PlanVisits returns the visits ids that the clinic org sees, in the order of ids (one it does not see is left
// out), read in the transaction in ctx, unaudited: the plan's read audits itself. An empty org reads in the scope
// the caller set on the transaction (the patient's own plans).
func (s *Service) PlanVisits(ctx context.Context, org string, ids []string) ([]Appointment, error) {
	var out []Appointment
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if org != "" {
			if err := s.Repo.SetClinic(ctx, org); err != nil {
				return err
			}
		}
		for _, id := range ids {
			a, err := s.Repo.ByID(ctx, id)
			if errors.Is(err, ErrNotFound) {
				continue
			} else if err != nil {
				return err
			}
			out = append(out, a)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("appointment: plan visits: %w", err)
	}
	return out, nil
}

// CancelPlan cancels, as the clinic (reason prov, by c's account), every visit of ids that is booked and still ahead,
// in one transaction (the caller's, when ctx carries one), and audits each; a past, closed or unseen visit is kept as
// it is; any other refusal (ErrForbidden, ErrNotFound) cancels nothing and is returned. careplan.Service has
// authorized c for the plan, so c acts with the admin's access to its visits.
func (s *Service) CancelPlan(ctx context.Context, c Caller, ids []string) error {
	if c.patient() {
		return ErrForbidden
	}
	c.Admin = true
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		visits, err := s.PlanVisits(ctx, c.OrganizationID, ids)
		if err != nil {
			return err
		}
		now := s.Clock.Now()
		for _, a := range visits {
			if a.Status != StatusBooked || !a.Start.After(now) {
				continue
			}
			out, refused, err := s.cancel(ctx, c, a.ID)
			if err != nil {
				return err
			}
			if errors.Is(refused, ErrClosed) { // closed by a concurrent change
				continue
			} else if refused != nil {
				return refused
			}
			if err := s.record(ctx, c, auditevent.CodeUpdate, "Appointment", out.ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("appointment: cancel plan: %w", err)
	}
	return nil
}
