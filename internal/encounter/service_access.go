package encounter

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/appointment"
)

// Lookup is for a system path that has only the encounter's id (the consultation's stream consumer,
// docs/plans/consult-intelligence.md): it scopes the caller's transaction to the encounter's clinic and
// returns the encounter as that scope shows it (with an active consent or, after a revocation, for an
// appointment that started before it), unaudited: the caller audits its own write. ErrNotFound when there is
// none or the scope does not show it.
func (s *Service) Lookup(ctx context.Context, id string) (Encounter, error) {
	var out Encounter
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		org, err := s.Repo.Organization(ctx, id)
		if err != nil {
			return err
		}
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		out, err = s.Repo.ByID(ctx, id)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Encounter{}, ErrNotFound
	case err != nil:
		return Encounter{}, fmt.Errorf("encounter: lookup: %w", err)
	}
	return out, nil
}

// Access returns the encounter id with the caller's attendance of its appointment, read in the caller's own
// scope in the transaction in ctx (or one of its own), unaudited: for a service that reads what was said in
// the consultation, which applies its own rule to the attendance and audits its decision. ErrNotFound when the
// caller may not see the appointment at any level.
func (s *Service) Access(ctx context.Context, c appointment.Caller, id string) (Encounter, appointment.Attendance, error) {
	var e Encounter
	var att appointment.Attendance
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		ok, err := s.scope(ctx, c)
		switch {
		case err != nil:
			return err
		case !ok:
			return ErrNotFound
		}
		if e, err = s.Repo.ByID(ctx, id); err != nil {
			return err
		}
		att, err = s.Appointments.Attendance(ctx, c, e.AppointmentID)
		if errors.Is(err, appointment.ErrNotFound) {
			return ErrNotFound
		}
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return Encounter{}, appointment.Attendance{}, ErrNotFound
	case err != nil:
		return Encounter{}, appointment.Attendance{}, fmt.Errorf("encounter: access: %w", err)
	}
	return e, att, nil
}
