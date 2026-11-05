package videosession

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

// agentWebhook names the LiveKit webhook as the agent of its writes and audit events.
const agentWebhook = "livekit_webhook"

// HandleWebhook verifies a LiveKit webhook (SPEC.md §2 "Webhook endpoint") and handles its event
// idempotently and in any order (§2 "Idempotent, order-free handling"): every effect is a change guarded in
// SQL, so a repeat or a late event changes nothing. It returns the event's kind; ErrInvalidSignature when
// LiveKit did not sign it, ErrUnknownRoom (no effect) for a room or identity we do not know. Any other error
// is for LiveKit to deliver the webhook again.
func (s *Service) HandleWebhook(ctx context.Context, body []byte, authorization string) (EventKind, error) {
	e, err := s.Provider.ParseWebhook(body, authorization)
	switch {
	case errors.Is(err, ErrInvalidSignature):
		return "", err
	case err != nil:
		return "", fmt.Errorf("videosession: webhook: %w", err)
	}
	switch e.Kind {
	case EventParticipantJoined:
		err = s.joined(ctx, e)
	case EventParticipantLeft, EventRoomFinished:
		err = s.left(ctx, e)
	}
	if err != nil && !errors.Is(err, ErrUnknownRoom) {
		err = fmt.Errorf("videosession: webhook: %w", err)
	}
	return e.Kind, err
}

// joined records a participant's join, unless the session ended. The practitioner's first join starts the
// Encounter at LiveKit's join time. The patient's access is checked again on every join (consent active,
// appointment booked) and, when it is gone, they are removed from the room after the commit: a second guard
// behind the token's check, for a token issued just before a revocation or a cancellation committed.
func (s *Service) joined(ctx context.Context, e Event) error {
	now := s.Clock.Now()
	var session Session
	var remove bool
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if session, err = s.inRoom(ctx, e.Room); err != nil {
			return err
		}
		role, known := session.roleOf(e.Identity)
		switch {
		case !known:
			return ErrUnknownRoom
		case !session.EndedAt.IsZero():
			return nil
		}
		joined, changed, err := s.Repo.Joined(ctx, session.ID, role, e.At, now)
		if err != nil {
			return err
		}
		if changed {
			if err := s.recordSystem(ctx, agentWebhook, joined); err != nil {
				return err
			}
		}
		if role == RolePatient {
			remove, err = s.accessGone(ctx, session.AppointmentID)
			return err
		}
		if !changed {
			return nil
		}
		_, err = s.Encounters.Start(ctx, agentWebhook, session.AppointmentID, e.At)
		if errors.Is(err, encounter.ErrNotFound) { // the clinic's scope no longer shows the appointment
			return nil
		}
		return err
	})
	if err != nil || !remove {
		return err
	}
	return s.Provider.RemoveParticipant(ctx, session.Room, session.PatientIdentity)
}

// left ends the session when a participant left or the room finished after the window closed, as the
// window-close job would (CloseWindow's rules: window_closed with the room empty, practitioner_absent,
// time_limit; nothing while someone is in the room). Before the window closes it is not the end: a dropped
// connection reconnects.
func (s *Service) left(ctx context.Context, e Event) error {
	now := s.Clock.Now()
	var session Session
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if session, err = s.inRoom(ctx, e.Room); err != nil {
			return err
		}
		if _, known := session.roleOf(e.Identity); e.Kind == EventParticipantLeft && !known {
			return ErrUnknownRoom
		}
		return nil
	})
	if err != nil || !session.EndedAt.IsZero() || now.Before(session.ClosesAt) {
		return err
	}
	reason, snooze, err := s.closing(ctx, session, now)
	if err != nil || snooze > 0 {
		return err
	}
	return s.end(ctx, agentWebhook, session, reason, now)
}

// inRoom scopes the transaction in ctx to the room's session (the webhook carries no user) and then to the
// session's clinic, for the Encounter and Appointment it reads and writes; ErrUnknownRoom when there is none.
func (s *Service) inRoom(ctx context.Context, room string) (Session, error) {
	if err := s.Repo.SetRoom(ctx, room); err != nil {
		return Session{}, err
	}
	session, err := s.Repo.ByRoom(ctx, room)
	switch {
	case errors.Is(err, ErrNotFound):
		return Session{}, ErrUnknownRoom
	case err != nil:
		return Session{}, err
	}
	return session, s.Repo.SetClinic(ctx, session.OrganizationID)
}

// accessGone reports whether the patient may no longer be in the appointment's room: the clinic's scope no
// longer shows it, it is no longer booked, or the patient's consent with the clinic is not active.
func (s *Service) accessGone(ctx context.Context, appointmentID string) (bool, error) {
	a, err := s.Appointments.Lookup(ctx, appointmentID)
	switch {
	case errors.Is(err, appointment.ErrNotFound):
		return true, nil
	case err != nil:
		return false, err
	case a.Status != appointment.StatusBooked:
		return true, nil
	}
	consent, err := s.Consents.State(ctx, a.PatientID, a.OrganizationID)
	return !consent.Active, err
}

// roleOf is the role of the participant identity in the session; false for an identity not its own.
func (s Session) roleOf(identity string) (Role, bool) {
	switch identity {
	case "":
		return "", false
	case s.PatientIdentity:
		return RolePatient, true
	case s.PractitionerIdentity:
		return RolePractitioner, true
	}
	return "", false
}
