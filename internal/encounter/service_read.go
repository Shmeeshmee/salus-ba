package encounter

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Read returns the encounter id to whoever access.Appointment gives the name or more of its appointment (its
// patient; with an active consent its practitioner and the clinic's admin; after a revocation, for a past
// appointment, at the name level), read in the caller's own scope and audited. ErrNotFound to anyone else,
// audited without the entity.
func (s *Service) Read(ctx context.Context, c appointment.Caller, id string) (Encounter, error) {
	var out Encounter
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.read(ctx, c, id)
		if errors.Is(err, ErrNotFound) {
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		}
		if err != nil {
			return err
		}
		out = e
		return s.record(ctx, c, auditevent.CodeRead, e.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Encounter{}, fmt.Errorf("encounter: read: %w", err)
	case out.ID == "":
		return Encounter{}, ErrNotFound
	}
	return out, nil
}

// read gives the encounter id when the caller may see its appointment, in the transaction in ctx.
func (s *Service) read(ctx context.Context, c appointment.Caller, id string) (Encounter, error) {
	ok, err := s.scope(ctx, c)
	switch {
	case err != nil:
		return Encounter{}, err
	case !ok:
		return Encounter{}, ErrNotFound
	}
	e, err := s.Repo.ByID(ctx, id)
	if err != nil {
		return Encounter{}, err
	}
	_, ok, err = s.attendance(ctx, c, e)
	switch {
	case err != nil:
		return Encounter{}, err
	case !ok:
		return Encounter{}, ErrNotFound
	}
	return e, nil
}

// scope scopes the transaction in ctx to the caller: a patient's own, or the staff caller's clinic; false for
// a patient without a profile, who has none.
func (s *Service) scope(ctx context.Context, c appointment.Caller) (bool, error) {
	if c.OrganizationID != "" {
		return true, s.Repo.SetClinic(ctx, c.OrganizationID)
	}
	if c.PatientID == "" {
		return false, nil
	}
	return true, s.Repo.SetPatient(ctx, c.PatientID)
}

// attendance gives the caller's attendance of e's appointment, and whether it lets them see e.
func (s *Service) attendance(ctx context.Context, c appointment.Caller, e Encounter) (appointment.Attendance, bool, error) {
	att, err := s.Appointments.Attendance(ctx, c, e.AppointmentID)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return appointment.Attendance{}, false, nil
	case err != nil:
		return appointment.Attendance{}, false, err
	}
	return att, att.Level == access.Name || att.Level == access.Full, nil
}

// Search returns the encounters of the filter as the caller may see them, scoped like the Appointment
// search: a patient only their own (whatever patient the filter names), a practitioner only those of their
// own role (ErrForbidden, audited, when the filter names another practitioner), the clinic's admin the
// clinic's. PractitionerID is the practitioner behind the role. Each encounter returned is audited; a search
// that returns none writes one event without an entity.
func (s *Service) Search(ctx context.Context, c appointment.Caller, f Filter) ([]Encounter, error) {
	var out []Encounter
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, refused, err = s.search(ctx, c, f); err != nil {
			return err
		}
		switch {
		case refused != nil:
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure)
		case len(out) == 0:
			return s.record(ctx, c, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
		}
		return nil
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("encounter: search: %w", err)
	case refused != nil:
		return nil, refused
	}
	return out, nil
}

// search runs the caller's search in the transaction in ctx and audits each result, or gives the refusal.
func (s *Service) search(ctx context.Context, c appointment.Caller, f Filter) (out []Encounter, refused, err error) {
	staff, practitioner := c.OrganizationID != "", c.OrganizationID != "" && !c.Admin
	switch {
	case practitioner && f.PractitionerID != "" && f.PractitionerID != c.PractitionerID:
		return nil, ErrForbidden, nil
	case !staff:
		f.PatientID = c.PatientID
	}
	if ok, err := s.scope(ctx, c); err != nil || !ok {
		return nil, nil, err
	}
	all, err := s.Repo.Search(ctx, f)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range all {
		att, ok, err := s.attendance(ctx, c, e)
		switch {
		case err != nil:
			return nil, nil, err
		case !ok, practitioner && !att.Practitioner, f.PractitionerID != "" && att.PractitionerID != f.PractitionerID:
			continue
		}
		if err := s.record(ctx, c, auditevent.CodeSearchType, e.ID, auditevent.OutcomeSuccess); err != nil {
			return nil, nil, err
		}
		out = append(out, e)
	}
	return out, nil, nil
}
