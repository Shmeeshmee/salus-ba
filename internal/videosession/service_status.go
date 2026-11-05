package videosession

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// agentEnd names the practitioner's end of the consultation as the agent of the writes it causes (the
// session's end, the Encounter's finish and the appointment's outcome); the request itself is audited with the
// practitioner's account.
const agentEnd = "consultation_end"

// Status gives where the appointment's consultation stands (SPEC.md §2 "Consultation status") to its
// patient, its practitioner and the clinic's admin, as access.Appointment lets them see the appointment (the
// name level or more), read in their own scope and audited as a read of the appointment. ErrNotFound to
// anyone else, audited without the entity.
func (s *Service) Status(ctx context.Context, c appointment.Caller, appointmentID string) (Status, error) {
	var out Status
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		_, err := s.Appointments.Attendance(ctx, c, appointmentID)
		switch {
		case errors.Is(err, appointment.ErrNotFound):
			refused = ErrNotFound
			return s.record(ctx, c, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)
		case err != nil:
			return err
		}
		if err := s.scope(ctx, c); err != nil {
			return err
		}
		session, err := s.Repo.ByAppointment(ctx, appointmentID)
		switch {
		case errors.Is(err, ErrNotFound):
			out = Status{State: StateNotStarted}
		case err != nil:
			return err
		default:
			out = session.status()
		}
		return s.record(ctx, c, auditevent.CodeRead, "Appointment/"+appointmentID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Status{}, fmt.Errorf("videosession: status: %w", err)
	case refused != nil:
		return Status{}, refused
	}
	return out, nil
}

// End is the practitioner's end of the consultation (SPEC.md §2 "Ending"): the appointment's practitioner
// only, once they joined, also after a revocation. In one transaction it audits the request and ends the
// session (practitioner), which finishes the Encounter and records the outcome; after the commit it deletes
// the room. Repeated, it answers the ended consultation again and deletes the room again. ErrNotFound to
// anyone else and ErrNotStarted before the practitioner joined, audited.
func (s *Service) End(ctx context.Context, c appointment.Caller, appointmentID string) (Status, error) {
	now := s.Clock.Now()
	var session Session
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var entity string
		var err error
		if session, entity, refused, err = s.endable(ctx, c, appointmentID); err != nil {
			return err
		}
		if refused != nil {
			return s.record(ctx, c, auditevent.CodeOperation, entity, auditevent.OutcomeMinorFailure)
		}
		if err := s.record(ctx, c, auditevent.CodeOperation, entity, auditevent.OutcomeSuccess); err != nil {
			return err
		}
		if session.EndedAt.IsZero() {
			session, err = s.ending(ctx, agentEnd, session, EndPractitioner, now)
		}
		return err
	})
	switch {
	case err != nil:
		return Status{}, fmt.Errorf("videosession: end: %w", err)
	case refused != nil:
		return Status{}, refused
	}
	if err := s.Provider.DeleteRoom(ctx, session.Room); err != nil {
		return Status{}, fmt.Errorf("videosession: end: %w", err)
	}
	return session.status(), nil
}

// endable gives the session of the appointment id that the caller may end, read in the transaction in ctx,
// or why not, with the entity the audit event names ("" when the caller may not see the appointment).
func (s *Service) endable(ctx context.Context, c appointment.Caller, id string) (session Session, entity string, refused, err error) {
	att, err := s.Appointments.Attendance(ctx, c, id)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return Session{}, "", ErrNotFound, nil
	case err != nil:
		return Session{}, "", nil, err
	}
	entity = "Appointment/" + att.Appointment.ID
	if !att.Practitioner {
		return Session{}, entity, ErrNotFound, nil
	}
	if err := s.scope(ctx, c); err != nil {
		return Session{}, "", nil, err
	}
	session, err = s.Repo.ByAppointment(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return Session{}, entity, ErrNotStarted, nil
	case err != nil:
		return Session{}, "", nil, err
	case session.EndedAt.IsZero() && session.PractitionerJoinedAt.IsZero():
		return Session{}, entity, ErrNotStarted, nil
	}
	return session, entity, nil, nil
}
