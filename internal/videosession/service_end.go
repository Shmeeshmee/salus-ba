package videosession

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// agentWindowClose names the consultation_window_close job as the agent of its writes and audit events.
const agentWindowClose = "consultation_window_close"

// snoozeEvery is how often the window-close job looks again at a room that is still occupied.
const snoozeEvery = time.Minute

// CloseWindow is the consultation_window_close job for the session of the clinic org (SPEC.md §2 "The window
// closes", "The hard cap"), run at the window's close. It ends the session practitioner_absent when the
// practitioner never joined; time_limit at and after the hard cap (2 hours after the scheduled end), whoever
// is in the room; window_closed when the room is empty. Otherwise it returns how long to snooze: a minute,
// never past the cap. A session ended already gets its room deleted again (a retry after a failed delete);
// one the clinic's scope does not show is left alone.
func (s *Service) CloseWindow(ctx context.Context, sessionID, org string) (time.Duration, error) {
	now := s.Clock.Now()
	var session Session
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, org); err != nil {
			return err
		}
		var err error
		session, err = s.Repo.ByID(ctx, sessionID)
		return err
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return 0, nil
	case err != nil:
		return 0, fmt.Errorf("videosession: close window: %w", err)
	}
	reason, snooze, err := s.closing(ctx, session, now)
	if err != nil {
		return 0, fmt.Errorf("videosession: close window: %w", err)
	}
	if snooze > 0 {
		return snooze, nil
	}
	ended, err := s.end(ctx, agentWindowClose, session, reason, now)
	if err == nil && !ended { // the practitioner joined after the reason was chosen: look again
		return snoozeEvery, nil
	}
	return 0, err
}

// closing decides how the window's close at now ends the session, or how long to wait while the room is
// occupied before the hard cap.
func (s *Service) closing(ctx context.Context, session Session, now time.Time) (EndReason, time.Duration, error) {
	hardCap := session.ClosesAt.Add(hardCapAfter - closesAfter)
	switch {
	case !session.EndedAt.IsZero():
		return session.EndReason, 0, nil
	case session.PractitionerJoinedAt.IsZero():
		return EndPractitionerAbsent, 0, nil
	case !now.Before(hardCap):
		return EndTimeLimit, 0, nil
	}
	in, err := s.Provider.Participants(ctx, session.Room)
	if err != nil {
		return "", 0, err
	}
	if len(in) == 0 {
		return EndWindowClosed, 0, nil
	}
	return "", min(snoozeEvery, hardCap.Sub(now)), nil
}

// end ends the session for reason at at, by the agent (SPEC.md §2 "Ending"): in one transaction in its
// clinic's scope it sets the end (only if not ended), audits it and, when the practitioner joined, finishes
// the Encounter, which records the appointment's outcome. After the commit it deletes the room; a failure is
// returned for the calling path to retry, with the end committed. A session ended already only has its room
// deleted. It reports whether the session has ended: not when reason is practitioner_absent and the
// practitioner's join committed after the reason was chosen (the SQL guards it); then the room stays.
func (s *Service) end(ctx context.Context, agent string, session Session, reason EndReason, at time.Time) (bool, error) {
	if session.EndedAt.IsZero() {
		err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
			var err error
			session, err = s.ending(ctx, agent, session, reason, at)
			return err
		})
		if err != nil {
			return false, fmt.Errorf("videosession: end: %w", err)
		}
		if session.EndedAt.IsZero() {
			return false, nil
		}
	}
	if err := s.Provider.DeleteRoom(ctx, session.Room); err != nil {
		return true, fmt.Errorf("videosession: end: %w", err)
	}
	return true, nil
}

// ending is end's transaction, in the transaction in ctx: it returns the session as it ended, also when
// another path ended it first (then it changes, finishes and audits nothing). The session is ended in its
// room's scope, which shows it whatever the consent (EndPatientAccess ends one the clinic's scope no longer
// shows); the Encounter and the Appointment are written in its clinic's.
func (s *Service) ending(ctx context.Context, agent string, session Session, reason EndReason, at time.Time) (Session, error) {
	if err := s.Repo.SetRoom(ctx, session.Room); err != nil {
		return Session{}, err
	}
	if err := s.Repo.SetClinic(ctx, session.OrganizationID); err != nil {
		return Session{}, err
	}
	ended, changed, err := s.Repo.End(ctx, session.ID, reason, at)
	switch {
	case err != nil:
		return Session{}, err
	case !changed:
		return s.Repo.ByID(ctx, session.ID)
	}
	if err := s.recordSystem(ctx, agent, ended); err != nil {
		return Session{}, err
	}
	if ended.PractitionerJoinedAt.IsZero() { // no Encounter: the appointment stays as it is
		return ended, nil
	}
	return ended, s.Encounters.Finish(ctx, agent, ended.AppointmentID, at, !ended.PatientJoinedAt.IsZero())
}

// recordSystem audits the agent's operation on the session (its end, a join, the patient's removal) as an
// operation on its appointment, for operations.
func (s *Service) recordSystem(ctx context.Context, agent string, session Session) error {
	return s.recordSystemOutcome(ctx, agent, session, auditevent.OutcomeSuccess)
}

// recordSystemOutcome is recordSystem with the outcome; a session with no appointment (none was disclosed)
// names no entity.
func (s *Service) recordSystemOutcome(ctx context.Context, agent string, session Session, outcome auditevent.Outcome) error {
	entity := ""
	if session.AppointmentID != "" {
		entity = "Appointment/" + session.AppointmentID
	}
	return s.Audit.Record(ctx, auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeOperation},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          auditevent.ActionExecute,
		Outcome:         outcome,
		AgentName:       agent,
		OrganizationID:  session.OrganizationID,
		EntityReference: entity,
	})
}
