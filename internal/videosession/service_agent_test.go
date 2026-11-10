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

func agentJoin(t *testing.T, f *fixture) (videosession.AgentTicket, error) {
	t.Helper()
	return f.svc.AgentJoin(t.Context(), f.session().ID, clinicA)
}

// The agent (SPEC.md §2 "Changed for SAL-3") gets a token for the open session's room: subscribe only, as the
// session's agent identity (a_ and the room's random part), for 5 minutes; with the encounter, the appointment
// and who is who in the room, for its transcript (SAL-4). The room is created again outside the
// transaction, still for two people (LiveKit does not count an agent), and the request is audited.
func TestAgentJoin_GivesTheAgentASubscribeOnlyTokenForTheRoom(t *testing.T) {
	f := opened(during, start, start)
	got, err := agentJoin(t, f)
	if err != nil {
		t.Fatalf("AgentJoin() error = %v", err)
	}
	want := videosession.AgentTicket{
		Join:          videosession.Join{URL: publicURL, Token: "token:a_room", ExpiresAt: during.Add(5 * time.Minute), Role: videosession.RoleAgent},
		EncounterID:   encounterA,
		AppointmentID: apptA, PatientIdentity: "p_patient", PractitionerIdentity: "p_practitioner",
	}
	if got != want {
		t.Errorf("AgentJoin() = %+v, want %+v", got, want)
	}
	// The encounter the practitioner's join started, read through its idempotent start at that join.
	if want := []string{transcriptionAgent + " " + apptA + " " + start.Format(time.RFC3339Nano) + " in " + clinicA}; !slices.Equal(f.encounters.started, want) {
		t.Errorf("encounters started %v, want %v", f.encounters.started, want)
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
// that ended or whose window closed, an appointment no longer booked, without the patient's active consent or
// their active transcription consent (SAL-12), or before the practitioner joined (no Encounter yet, SAL-4).
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
		{name: "transcription declined", at: during, want: videosession.ErrConsentRequired, entity: true, setup: func(f *fixture) { f.consents.declined = true }},
		{name: "the practitioner has not joined", at: during, want: videosession.ErrNotStarted, entity: true, setup: func(f *fixture) {
			s := f.session()
			s.PractitionerJoinedAt = time.Time{}
			f.repo.sessions[apptA] = s
		}},
	} {
		f := opened(tc.at, start, start)
		tc.setup(f)
		if _, err := agentJoin(t, f); !errors.Is(err, tc.want) {
			t.Errorf("%s: AgentJoin() = %v, want %v", tc.name, err, tc.want)
		}
		if len(f.provider.grants) != 0 || len(f.provider.rooms) != 0 || len(f.encounters.started) != 0 {
			t.Errorf("%s: grants %v, rooms %v, encounters %v; want none", tc.name, f.provider.grants, f.provider.rooms, f.encounters.started)
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
		"the database":  func(f *fixture) { f.repo.err = errBackend },
		"the encounter": func(f *fixture) { f.encounters.startErr = errBackend },
		"LiveKit":       func(f *fixture) { f.provider.err = errBackend },
		"the signer":    func(f *fixture) { f.provider.tokenErr = errBackend },
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

// Once connected, the agent asks again whether it may stay (SAL-4): a revocation that committed between its token
// and its connection is seen, so it leaves at once rather than wait for a removal that ran before it was in the
// room. The same rules as AgentJoin, each answer audited; nothing reaches LiveKit.
func TestAgentAdmitted_AnswersWithTheJoinsRules(t *testing.T) {
	for name, tc := range map[string]struct {
		want  error
		setup func(f *fixture)
	}{
		"open":                    {setup: func(*fixture) {}},
		"consent revoked":         {want: videosession.ErrConsentRequired, setup: func(f *fixture) { f.consents.revoked = true }},
		"transcription withdrawn": {want: videosession.ErrConsentRequired, setup: func(f *fixture) { f.consents.declined = true }},
		"cancelled":               {want: videosession.ErrConsultationEnded, setup: func(f *fixture) { f.cancel() }},
		"ended": {want: videosession.ErrConsultationEnded, setup: func(f *fixture) {
			s := f.session()
			s.EndedAt, s.EndReason = during.Add(-time.Minute), videosession.EndPractitioner
			f.repo.sessions[apptA] = s
		}},
		"hidden by a revocation": {want: videosession.ErrNotFound, setup: func(f *fixture) { f.repo.hidden = true }},
		"the database failing":   {want: errBackend, setup: func(f *fixture) { f.repo.err = errBackend }},
	} {
		f := opened(during, start, start)
		id := f.session().ID
		tc.setup(f)
		err := f.svc.AgentAdmitted(t.Context(), id, clinicA)
		if !errors.Is(err, tc.want) || (tc.want == nil && err != nil) {
			t.Errorf("%s: AgentAdmitted() = %v, want %v", name, err, tc.want)
		}
		if len(f.provider.grants) != 0 || len(f.provider.rooms) != 0 {
			t.Errorf("%s: grants %v, rooms %v; want none", name, f.provider.grants, f.provider.rooms)
		}
		if tc.want == errBackend {
			continue
		}
		want := agentEvent(auditevent.OutcomeSuccess)
		switch {
		case tc.want == videosession.ErrNotFound:
			want.Outcome, want.EntityReference = auditevent.OutcomeMinorFailure, ""
		case tc.want != nil:
			want.Outcome = auditevent.OutcomeMinorFailure
		}
		checkEvents(t, name, f.audit, want)
	}
}

// frames is a closed channel of n frames of PCM.
func frames(n int) <-chan []byte {
	c := make(chan []byte, n)
	for i := range n {
		c <- []byte{byte(i), 0}
	}
	close(c)
	return c
}

// The agent joins the room with its token and hears each person's audio as their role in the consultation
// (SAL-4): the identities stay here. Audio of anyone else (an identity not the session's) is not handed over.
func TestAgentListen_HandsOverEachPersonsAudioAsTheirRole(t *testing.T) {
	f := opened(during, start, start)
	patient, practitioner := frames(2), frames(3)
	f.listener.in = []videosession.RemoteAudio{
		{Identity: "p_practitioner", Frames: practitioner}, {Identity: "x_stranger", Frames: frames(1)}, {Identity: "p_patient", Frames: patient},
	}
	call, err := f.svc.AgentListen(t.Context(), f.session().ID, clinicA)
	if err != nil {
		t.Fatalf("AgentListen() = %v", err)
	}
	if call.EncounterID != encounterA || call.AppointmentID != apptA {
		t.Errorf("call of encounter %q and appointment %q, want %q and %q", call.EncounterID, call.AppointmentID, encounterA, apptA)
	}
	if want := []string{publicURL + " token:a_room"}; !slices.Equal(f.listener.joined, want) {
		t.Errorf("joined %v, want %v", f.listener.joined, want)
	}
	call.Leave()
	var got []videosession.AudioTrack
	for tr := range call.Tracks {
		got = append(got, tr)
	}
	want := []videosession.AudioTrack{{Role: videosession.RolePractitioner, Frames: practitioner}, {Role: videosession.RolePatient, Frames: patient}}
	if !slices.Equal(got, want) {
		t.Errorf("tracks %v, want the practitioner's then the patient's", got)
	}
	// Two answers: the token's, and the one asked once connected.
	checkEvents(t, "AgentListen", f.audit, agentEvent(auditevent.OutcomeSuccess), agentEvent(auditevent.OutcomeSuccess))
}

// A refused token connects nothing; a revocation that commits while the agent connects makes it leave at once,
// before any audio is handed over; a failed connection is the error.
func TestAgentListen_LeavesOrNeverJoinsWithoutAccess(t *testing.T) {
	f := opened(during, start, start)
	f.consents.revoked = true
	if _, err := f.svc.AgentListen(t.Context(), f.session().ID, clinicA); !errors.Is(err, videosession.ErrConsentRequired) || len(f.listener.joined) != 0 {
		t.Errorf("revoked before: AgentListen() = %v, joined %v; want ErrConsentRequired and no connection", err, f.listener.joined)
	}

	f = opened(during, start, start)
	f.listener.in = []videosession.RemoteAudio{{Identity: "p_patient", Frames: frames(1)}}
	f.listener.onJoin = func() { f.consents.revoked = true }
	if _, err := f.svc.AgentListen(t.Context(), f.session().ID, clinicA); !errors.Is(err, videosession.ErrConsentRequired) {
		t.Errorf("revoked while connecting: AgentListen() = %v, want ErrConsentRequired", err)
	}
	if f.listener.leaves() != 1 {
		t.Errorf("left %d times, want once", f.listener.leaves())
	}

	f = opened(during, start, start)
	f.listener.err = errBackend
	if _, err := f.svc.AgentListen(t.Context(), f.session().ID, clinicA); !errors.Is(err, errBackend) {
		t.Errorf("a failed connection: AgentListen() = %v, want its error", err)
	}
}

// The transcription consent's withdrawal (SAL-12) takes the agent out of the patient's open consultations with the
// clinic, read in the patient's scope, after the commit; the patient and the practitioner stay and the call goes
// on. Nothing is audited: the agent is not a person, and no patient data is read but the sessions. Repeated, it
// asks LiveKit again (an agent no longer there is no error).
func TestEndAgentAccess_RemovesOnlyTheAgentFromTheOpenRoomsOfThePatientWithTheClinic(t *testing.T) {
	f := opened(during, start, start)
	f.consents.declined = true
	f.provider.in = []string{"p_patient", "p_practitioner"}
	f.repo.sessions[apptA2] = videosession.Session{
		ID: "5e000000-0000-4000-8000-000000000002", AppointmentID: apptA2, OrganizationID: clinicB, ClosesAt: closes,
		Room: "r_other", PatientIdentity: "p_other", PractitionerIdentity: "p_other_practitioner",
	}
	for range 2 {
		f.provider.removedAgents = nil
		if err := f.svc.EndAgentAccess(t.Context(), patientA, clinicA); err != nil {
			t.Fatalf("EndAgentAccess() = %v", err)
		}
		if !slices.Equal(f.provider.removedAgents, []string{"r_room a_room"}) || len(f.provider.removed) != 0 || f.provider.inTx {
			t.Errorf("agents removed %v, people %v (in a transaction: %t); want only clinic A's agent, outside it",
				f.provider.removedAgents, f.provider.removed, f.provider.inTx)
		}
	}
	if !f.session().EndedAt.IsZero() || len(f.provider.deleted) != 0 || len(f.audit.events) != 0 {
		t.Errorf("session %+v, deleted %v, events %v; want the call to go on", f.session(), f.provider.deleted, f.audit.events)
	}
}

func TestEndAgentAccess_FailsForARetry(t *testing.T) {
	for name, setup := range map[string]func(*fixture){
		"the sessions": func(f *fixture) { f.repo.err = errBackend },
		"the removal":  func(f *fixture) { f.provider.removeErr = errBackend },
	} {
		f := opened(during, start, start)
		setup(f)
		if err := f.svc.EndAgentAccess(t.Context(), patientA, clinicA); !errors.Is(err, errBackend) {
			t.Errorf("with %s failing: EndAgentAccess() = %v, want the error", name, err)
		}
	}
}
