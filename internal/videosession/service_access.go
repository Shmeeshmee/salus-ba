package videosession

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
)

// The jobs that end a patient's access, as the agents of their writes and audit events (their job kinds).
const (
	agentAccessEnd  = "consultation_access_end"
	agentRevocation = "consent_revoked"
)

// EndAccess is the consultation_access_end job, enqueued with every cancellation of the appointment of the
// clinic org (SPEC.md §2 "Removal"): it removes the patient from the room and, when the appointment is
// cancelled, ends the session cancelled, which deletes the room. It reads the session in the clinic's scope;
// with none there (nobody asked for a token, or a revocation hides it, which EndPatientAccess handles) it does
// nothing. Repeated, it changes nothing.
func (s *Service) EndAccess(ctx context.Context, appointmentID, org string) error {
	now := s.Clock.Now()
	var session Session
	var cancelled bool
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		var err error
		if session, err = s.Repo.ByAppointment(ctx, appointmentID); err != nil {
			return err
		}
		cancelled, err = s.isCancelled(ctx, appointmentID)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("videosession: end access: %w", err)
	}
	if err := s.removePatient(ctx, agentAccessEnd, session, cancelled, now); err != nil {
		return fmt.Errorf("videosession: end access: %w", err)
	}
	return nil
}

// EndPatientAccess is the consent_revoked job's part in the consultation (SPEC.md §2 "Removal"), after the
// patient's appointments with the clinic org that had not started were cancelled. In the patient's scope it
// reads their sessions there whose room may still be open (the hard cap not passed) and removes the patient
// from each. The practitioner stays in the room of an appointment that started before the revocation, and
// ends it as usual. The session of an appointment the revocation cancelled ends cancelled here, in its room's
// scope, because the clinic's scope, which EndAccess reads in, no longer shows it. A failure in one room does
// not keep the patient in another; repeated, it changes nothing.
func (s *Service) EndPatientAccess(ctx context.Context, patientID, org string) error {
	now := s.Clock.Now()
	var sessions []Session
	var cancelled []bool
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		sessions, cancelled = nil, nil
		if err := s.Repo.SetPatient(ctx, patientID); err != nil {
			return err
		}
		var err error
		if sessions, err = s.Repo.ClosingAfter(ctx, org, now.Add(closesAfter-hardCapAfter)); err != nil {
			return err
		}
		for _, session := range sessions {
			c, err := s.isCancelled(ctx, session.AppointmentID)
			if err != nil {
				return err
			}
			cancelled = append(cancelled, c)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("videosession: end patient access: %w", err)
	}
	var errs []error
	for i, session := range sessions {
		errs = append(errs, s.removePatient(ctx, agentRevocation, session, cancelled[i], now))
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("videosession: end patient access: %w", err)
	}
	return nil
}

// isCancelled reports whether the appointment is cancelled, read in the transaction's scope.
func (s *Service) isCancelled(ctx context.Context, appointmentID string) (bool, error) {
	a, err := s.Appointments.Lookup(ctx, appointmentID)
	if err != nil {
		return false, err
	}
	return a.Status == appointment.StatusCancelled, nil
}

// removePatient removes the patient from the session's room and, when its appointment is cancelled, ends the
// session cancelled by the agent at now, which deletes the room (a session ended already only has its room deleted
// again).
func (s *Service) removePatient(ctx context.Context, agent string, session Session, cancelled bool, now time.Time) error {
	if err := s.removeFromRoom(ctx, agent, session); err != nil {
		return err
	}
	if !cancelled {
		return nil
	}
	return s.end(ctx, agent, session, EndCancelled, now)
}

// removeFromRoom removes the patient from the session's room for the agent, whose path decided their access
// is gone (SPEC.md §2 "Changed after package 5"). When the patient is in the room, the removal is audited
// first, so no removal goes unaudited; one who is not there (gone already: a repeat) is not removed, and
// nothing is audited. LiveKit is asked to remove them either way, outside any transaction.
func (s *Service) removeFromRoom(ctx context.Context, agent string, session Session) error {
	in, err := s.Provider.Participants(ctx, session.Room)
	if err != nil {
		return err
	}
	if slices.Contains(in, session.PatientIdentity) {
		err := s.Tx.WithTx(ctx, func(ctx context.Context) error { return s.recordSystem(ctx, agent, session) })
		if err != nil {
			return err
		}
	}
	return s.Provider.RemoveParticipant(ctx, session.Room, session.PatientIdentity)
}
