package videosession

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// agentTranscription names the transcription agent (cmd/agent) as the agent of its token requests' audit events.
const agentTranscription = "transcription_agent"

// AgentJoin gives the transcription agent (cmd/agent, SPEC.md §2 "Changed for SAL-3") a token for the room of the
// session of the clinic org: subscribe only, of kind agent, so both clients show that it is in the call. It is
// called in-process with the opaque ids a job carries, never over HTTP. Like a person's join, it needs the session
// open and inside its window, the appointment booked and the patient's consent with the clinic active; and the
// practitioner joined, so the consultation's Encounter exists (SAL-4). The ticket names that Encounter and who is
// who in the room. The room is created again outside the transaction (idempotent), still for two people: LiveKit
// does not count an agent. ErrNotFound for a session the clinic's scope does not show, ErrConsultationEnded,
// ErrConsentRequired and ErrNotStarted; every outcome is audited.
func (s *Service) AgentJoin(ctx context.Context, sessionID, org string) (AgentTicket, error) {
	now := s.Clock.Now()
	var session Session
	var encounterID string
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if session, refused, err = s.agentAdmittedAudited(ctx, sessionID, org); err != nil || refused != nil {
			return err
		}
		// The practitioner's join started it (the webhook): Start gives it back unchanged.
		e, err := s.Encounters.Start(ctx, agentTranscription, session.AppointmentID, session.PractitionerJoinedAt)
		encounterID = e.ID
		return err
	})
	switch {
	case err != nil:
		return AgentTicket{}, fmt.Errorf("videosession: agent join: %w", err)
	case refused != nil:
		return AgentTicket{}, refused
	}
	hardCap := session.ClosesAt.Add(hardCapAfter - closesAfter)
	room := Room{Name: session.Room, MaxParticipants: maxParticipants, EmptyTimeout: hardCap.Sub(now)}
	if err := s.Provider.CreateRoom(ctx, room); err != nil {
		return AgentTicket{}, fmt.Errorf("videosession: agent join: %w", err)
	}
	token, err := s.Provider.Token(Grant{Room: session.Room, Identity: session.agentIdentity(), ValidFor: tokenValidFor, Agent: true})
	if err != nil {
		return AgentTicket{}, fmt.Errorf("videosession: agent join: %w", err)
	}
	return AgentTicket{
		Join:        Join{URL: s.PublicURL, Token: token, ExpiresAt: now.Add(tokenValidFor), Role: RoleAgent},
		EncounterID: encounterID, AppointmentID: session.AppointmentID,
		PatientIdentity: session.PatientIdentity, PractitionerIdentity: session.PractitionerIdentity,
	}, nil
}

// AgentAdmitted reports whether the transcription agent may still be in the session's room, by AgentJoin's
// rules, audited the same way: nil, or why not. The agent asks once it is connected (SAL-4), so a revocation
// that committed between its token and its connection, whose removal may have run before it was in the room,
// still makes it leave.
func (s *Service) AgentAdmitted(ctx context.Context, sessionID, org string) error {
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		_, refused, err = s.agentAdmittedAudited(ctx, sessionID, org)
		return err
	})
	if err != nil {
		return fmt.Errorf("videosession: agent admitted: %w", err)
	}
	return refused
}

// agentAdmittedAudited is agentAdmitted, with its outcome audited in the transaction in ctx.
func (s *Service) agentAdmittedAudited(ctx context.Context, sessionID, org string) (Session, error, error) {
	session, refused, err := s.agentAdmitted(ctx, sessionID, org)
	if err != nil {
		return Session{}, nil, err
	}
	outcome := auditevent.OutcomeSuccess
	if refused != nil {
		outcome = auditevent.OutcomeMinorFailure
	}
	return session, refused, s.recordSystemOutcome(ctx, agentTranscription, session, outcome)
}

// agentAdmitted reads the session in the clinic's scope, in the transaction in ctx, and gives why the agent may
// not join it; a session not found is returned with the clinic only, so its refusal names no appointment.
func (s *Service) agentAdmitted(ctx context.Context, sessionID, org string) (Session, error, error) {
	if err := s.Repo.SetClinic(ctx, org); err != nil {
		return Session{}, nil, err
	}
	session, err := s.Repo.ByID(ctx, sessionID)
	switch {
	case errors.Is(err, ErrNotFound):
		return Session{OrganizationID: org}, ErrNotFound, nil
	case err != nil:
		return Session{}, nil, err
	case !session.EndedAt.IsZero() || !s.Clock.Now().Before(session.ClosesAt):
		return session, ErrConsultationEnded, nil
	}
	gone, reason, err := s.accessGone(ctx, session.AppointmentID)
	switch {
	case err != nil:
		return Session{}, nil, err
	case gone && reason == ReasonConsentWithdrawn:
		return session, ErrConsentRequired, nil
	case gone:
		return session, ErrConsultationEnded, nil
	case session.PractitionerJoinedAt.IsZero():
		return session, ErrNotStarted, nil
	}
	return session, nil, nil
}

// AgentListen puts the transcription agent in the session's room (SAL-4): AgentJoin's token, the connection, and
// AgentAdmitted once connected, so a revocation that committed meanwhile makes it leave before any audio is handed
// over. Each person's audio comes as their role; the room's identities stay here. AgentJoin's refusals, and
// AgentAdmitted's after it left.
func (s *Service) AgentListen(ctx context.Context, sessionID, org string) (AgentCall, error) {
	t, err := s.AgentJoin(ctx, sessionID, org)
	if err != nil {
		return AgentCall{}, err
	}
	l, err := s.Listener.Listen(ctx, t.Join.URL, t.Join.Token)
	if err != nil {
		return AgentCall{}, fmt.Errorf("videosession: agent listen: %w", err)
	}
	if err := s.AgentAdmitted(ctx, sessionID, org); err != nil {
		l.Leave()
		return AgentCall{}, err
	}
	tracks := make(chan AudioTrack)
	go func() {
		defer close(tracks)
		for a := range l.Tracks {
			switch a.Identity {
			case t.PatientIdentity:
				tracks <- AudioTrack{Role: RolePatient, Frames: a.Frames}
			case t.PractitionerIdentity:
				tracks <- AudioTrack{Role: RolePractitioner, Frames: a.Frames}
			}
		}
	}()
	return AgentCall{Tracks: tracks, Leave: l.Leave, EncounterID: t.EncounterID, AppointmentID: t.AppointmentID}, nil
}
