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
// open and inside its window, the appointment booked and the patient's consent with the clinic active. The room is
// created again outside the transaction (idempotent), still for two people: LiveKit does not count an agent.
// ErrNotFound for a session the clinic's scope does not show, ErrConsultationEnded and ErrConsentRequired; every
// outcome is audited.
func (s *Service) AgentJoin(ctx context.Context, sessionID, org string) (Join, error) {
	now := s.Clock.Now()
	var session Session
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if session, refused, err = s.agentAdmitted(ctx, sessionID, org); err != nil {
			return err
		}
		outcome := auditevent.OutcomeSuccess
		if refused != nil {
			outcome = auditevent.OutcomeMinorFailure
		}
		return s.recordSystemOutcome(ctx, agentTranscription, session, outcome)
	})
	switch {
	case err != nil:
		return Join{}, fmt.Errorf("videosession: agent join: %w", err)
	case refused != nil:
		return Join{}, refused
	}
	hardCap := session.ClosesAt.Add(hardCapAfter - closesAfter)
	room := Room{Name: session.Room, MaxParticipants: maxParticipants, EmptyTimeout: hardCap.Sub(now)}
	if err := s.Provider.CreateRoom(ctx, room); err != nil {
		return Join{}, fmt.Errorf("videosession: agent join: %w", err)
	}
	token, err := s.Provider.Token(Grant{Room: session.Room, Identity: session.agentIdentity(), ValidFor: tokenValidFor, Agent: true})
	if err != nil {
		return Join{}, fmt.Errorf("videosession: agent join: %w", err)
	}
	return Join{URL: s.PublicURL, Token: token, ExpiresAt: now.Add(tokenValidFor), Role: RoleAgent}, nil
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
	}
	return session, nil, nil
}
