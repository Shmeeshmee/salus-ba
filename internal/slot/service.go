package slot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Schedules reads the schedules slots are generated from (schedule.Service).
type Schedules interface {
	// ForGeneration returns the schedule of the clinic, locked until the transaction in ctx ends.
	ForGeneration(ctx context.Context, org, id string) (schedule.Schedule, error)
	// Listed returns the schedules of the active roles of listed clinics.
	Listed(ctx context.Context) ([]schedule.Schedule, error)
	// Read returns the schedule id as the directory shows it (of an active role of a listed clinic), or
	// schedule.ErrNotFound.
	Read(ctx context.Context, id string) (schedule.Schedule, error)
}

// Clinics reads a clinic, for its time zone (organization.Service).
type Clinics interface {
	Read(ctx context.Context, c organization.Caller, id string) (organization.Organization, error)
}

// Service holds the slot rules: generation from a schedule (SPEC.md "Slot generation"), the public search of
// free future slots, and holding and releasing a slot in the transaction of the appointment change that
// causes it. Nothing here is audited: a slot holds no patient data, and the appointment that holds one is
// audited by its own domain.
type Service struct {
	repo      Repository
	tx        Transactor
	clock     clock.Clock
	schedules Schedules
	clinics   Clinics
}

// NewService returns a Service on its ports.
func NewService(repo Repository, tx Transactor, clk clock.Clock, schedules Schedules, clinics Clinics) *Service {
	return &Service{repo: repo, tx: tx, clock: clk, schedules: schedules, clinics: clinics}
}

// Generate brings the slots of the schedule id of the clinic org in line with its rules, from now to 28 days
// ahead in the clinic's time zone, in one transaction: it deletes the future free slots the rules no longer
// give and adds those they give that overlap no slot. Busy slots, and the appointments that hold them, are
// never touched. Running it again changes nothing.
func (s *Service) Generate(ctx context.Context, org, id string) (Generated, error) {
	now := s.clock.Now()
	var out Generated
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		sc, err := s.schedules.ForGeneration(ctx, org, id)
		if err != nil {
			return err
		}
		clinic, err := s.clinics.Read(ctx, organization.Caller{OrganizationID: org}, org)
		if err != nil {
			return err
		}
		loc, err := time.LoadLocation(clinic.TimeZone)
		if err != nil {
			return err
		}
		given := periods(sc.Rules, sc.SlotMinutes, loc, now)
		if out.Deleted, err = s.repo.DeleteFreeExcept(ctx, id, now, given); err != nil {
			return err
		}
		out.Inserted, err = s.repo.Insert(ctx, id, org, given)
		return err
	})
	if err != nil {
		return Generated{}, fmt.Errorf("slot: generate: %w", err)
	}
	out.Schedules = 1
	return out, nil
}

// GenerateAll generates the slots of every schedule of an active role of a listed clinic, each in its own
// transaction (the scheduler's hourly run). A schedule that fails does not stop the others; the errors are
// returned together.
func (s *Service) GenerateAll(ctx context.Context) (Generated, error) {
	all, err := s.schedules.Listed(ctx)
	if err != nil {
		return Generated{}, fmt.Errorf("slot: generate all: %w", err)
	}
	var total Generated
	var errs []error
	for _, sc := range all {
		g, err := s.Generate(ctx, sc.OrganizationID, sc.ID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		total.Schedules += g.Schedules
		total.Inserted += g.Inserted
		total.Deleted += g.Deleted
	}
	return total, errors.Join(errs...)
}

// Search returns the free slots of the schedule that start in [from, to) and after now, by start: the public
// directory's, for anyone, so only of active roles of listed clinics (row-level security). Not audited.
func (s *Service) Search(ctx context.Context, scheduleID string, from, to time.Time) ([]Slot, error) {
	if now := s.clock.Now(); from.Before(now) {
		from = now
	}
	if !to.After(from) {
		return nil, nil
	}
	var out []Slot
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetDirectory(ctx); err != nil {
			return err
		}
		var err error
		out, err = s.repo.Search(ctx, scheduleID, from, to)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("slot: search: %w", err)
	}
	return out, nil
}

// Hold marks the free slot id of the clinic org busy and returns it, in the transaction in ctx (the
// appointment's booking), or returns ErrUnavailable when it is not free or not the clinic's.
func (s *Service) Hold(ctx context.Context, org, id string) (Slot, error) {
	var out Slot
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		var err error
		out, err = s.repo.Hold(ctx, id, s.clock.Now())
		return err
	})
	switch {
	case errors.Is(err, ErrUnavailable):
		return Slot{}, ErrUnavailable
	case err != nil:
		return Slot{}, fmt.Errorf("slot: hold: %w", err)
	}
	return out, nil
}

// Release frees the busy slot id of the clinic org when it still starts in the future, in the transaction in
// ctx (the appointment's cancellation), and reports whether it did; a past slot stays busy with its
// appointment.
func (s *Service) Release(ctx context.Context, org, id string) (bool, error) {
	var freed bool
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		var err error
		freed, err = s.repo.Release(ctx, id, s.clock.Now())
		return err
	})
	if err != nil {
		return false, fmt.Errorf("slot: release: %w", err)
	}
	return freed, nil
}

// Bookable returns the free slot id with the practitioner role it is booked with, as the public directory
// shows it (a free slot of an active role of a listed clinic), in a transaction of its own, before the
// booking's: ErrUnavailable when the directory does not show it. Not audited, like the directory.
func (s *Service) Bookable(ctx context.Context, id string) (Bookable, error) {
	var out Bookable
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetDirectory(ctx); err != nil {
			return err
		}
		sl, err := s.repo.ByID(ctx, id)
		if err != nil {
			return err
		}
		sc, err := s.schedules.Read(ctx, sl.ScheduleID)
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
		return Bookable{}, fmt.Errorf("slot: bookable: %w", err)
	}
	return out, nil
}
