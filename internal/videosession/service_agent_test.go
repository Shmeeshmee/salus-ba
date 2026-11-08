package videosession_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/videosession"
)

const transcriptionAgent = "transcription_agent"

// agentEvent is the audit event of the agent's token request for appointment A's session, with outcome.
func agentEvent(outcome auditevent.Outcome) auditevent.AuditEvent {
	e := agentEnd(transcriptionAgent)
	e.Outcome = outcome
	return e
}

func agentJoin(t *testing.T, f *fixture) (videosession.Join, error) {
	t.Helper()
	return f.svc.AgentJoin(t.Context(), f.session().ID, clinicA)
}

// The agent (SPEC.md §2 "Changed for SAL-3") gets a token for the open session's room: subscribe only, as the
// session's agent identity (a_ and the room's random part), for 5 minutes. The room is created again outside the
// transaction, still for two people (LiveKit does not count an agent), and the request is audited.
func TestAgentJoin_GivesTheAgentASubscribeOnlyTokenForTheRoom(t *testing.T) {
	f := opened(during, start, start)
	got, err := agentJoin(t, f)
	if err != nil {
		t.Fatalf("AgentJoin() error = %v", err)
	}
	want := videosession.Join{URL: publicURL, Token: "token:a_room", ExpiresAt: during.Add(5 * time.Minute), Role: videosession.RoleAgent}
	if got != want {
		t.Errorf("AgentJoin() = %+v, want %+v", got, want)
	}
	if want := []videosession.Grant{{Room: "r_room", Identity: "a_room", ValidFor: 5 * time.Minute, Agent: true}}; !slices.Equal(f.provider.grants, want) {
		t.Errorf("grants %+v, want %+v", f.provider.grants, want)
	}
	if want := []videosession.Room{{Name: "r_room", MaxParticipants: 2, EmptyTimeout: hardCap.Sub(during)}}; !slices.Equal(f.provider.rooms, want) || f.provider.inTx {
		t.Errorf("rooms %+v (in a transaction: %t), want %+v outside it", f.provider.rooms, f.provider.inTx, want)
	}
	checkEvents(t, "AgentJoin", f.audit, agentEvent(auditevent.OutcomeSuccess))
}

// The agent gets no token for a session the clinic's scope does not show (none, or hidden by a revocation), one
// that ended or whose window closed, an appointment no longer booked, or without the patient's active consent.
// Each refusal is audited; nothing reaches LiveKit.
func TestAgentJoin_RefusesWhenTheConsultationIsNotOpenToIt(t *testing.T) {
	for _, tc := range []struct {
		at     time.Time
		want   error
		setup  func(f *fixture)
		name   string
		entity bool // the refusal's audit event names the appointment
	}{
		{name: "another clinic's session", at: during, want: videosession.ErrNotFound, setup: func(f *fixture) {
			s := f.session()
			s.OrganizationID = clinicB
			f.repo.sessions[apptA] = s
		}},
		{name: "hidden by a revocation", at: during, want: videosession.ErrNotFound, setup: func(f *fixture) { f.repo.hidden = true }},
		{name: "ended", at: during, want: videosession.ErrConsultationEnded, entity: true, setup: func(f *fixture) {
			s := f.session()
			s.EndedAt, s.EndReason = during.Add(-time.Minute), videosession.EndPractitioner
			f.repo.sessions[apptA] = s
		}},
		{name: "the window closed", at: closes, want: videosession.ErrConsultationEnded, entity: true, setup: func(*fixture) {}},
		{name: "cancelled", at: during, want: videosession.ErrConsultationEnded, entity: true, setup: func(f *fixture) { f.cancel() }},
		{name: "consent revoked", at: during, want: videosession.ErrConsentRequired, entity: true, setup: func(f *fixture) { f.consents.revoked = true }},
	} {
		f := opened(tc.at, start, start)
		tc.setup(f)
		if _, err := agentJoin(t, f); !errors.Is(err, tc.want) {
			t.Errorf("%s: AgentJoin() = %v, want %v", tc.name, err, tc.want)
		}
		if len(f.provider.grants) != 0 || len(f.provider.rooms) != 0 {
			t.Errorf("%s: grants %v, rooms %v; want none", tc.name, f.provider.grants, f.provider.rooms)
		}
		want := agentEvent(auditevent.OutcomeMinorFailure)
		if !tc.entity {
			want.EntityReference = ""
		}
		checkEvents(t, tc.name, f.audit, want)
	}
}

// A failing database, LiveKit or signer fails the agent's request with the cause.
func TestAgentJoin_FailsWithTheBackend(t *testing.T) {
	for name, setup := range map[string]func(f *fixture){
		"the database": func(f *fixture) { f.repo.err = errBackend },
		"LiveKit":      func(f *fixture) { f.provider.err = errBackend },
		"the signer":   func(f *fixture) { f.provider.tokenErr = errBackend },
	} {
		f := opened(during, start, start)
		id := f.session().ID
		setup(f)
		if _, err := f.svc.AgentJoin(t.Context(), id, clinicA); !errors.Is(err, errBackend) {
			t.Errorf("%s failing: AgentJoin() = %v, want its error", name, err)
		}
	}
}

// Whenever the patient is removed from a room, the agent is removed from it too, after the commit: it may not
// stay to listen once the patient's access is gone. Nothing is audited for the agent, which is not a person.
func TestRemoval_RemovesTheAgentWithThePatient(t *testing.T) {
	f := opened(during, start, start)
	f.provider.in = []string{"p_patient", "p_practitioner"}
	f.consents.revoked = true
	if err := f.svc.EndPatientAccess(t.Context(), patientA, clinicA); err != nil {
		t.Fatalf("EndPatientAccess() = %v", err)
	}
	if !slices.Equal(f.provider.removed, []string{"r_room p_patient"}) || !slices.Equal(f.provider.removedAgents, []string{"r_room a_room"}) || f.provider.inTx {
		t.Errorf("removed %v and agents %v (in a transaction: %t); want the patient and the agent outside it", f.provider.removed, f.provider.removedAgents, f.provider.inTx)
	}
	checkEvents(t, "EndPatientAccess", f.audit, removalEvent(revocationAgent))
}
