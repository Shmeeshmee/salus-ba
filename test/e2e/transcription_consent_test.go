//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/livekit/protocol/livekit"
)

// The transcription consent's journeys (SAL-12, docs/plans/v1-finish.md "Decisions"): the agent joins a call only
// while the patient's care consent and transcription consent with the clinic are both active. Declining, the call
// runs without the agent; withdrawing during the call takes the agent out at once and leaves both people in it.
// The helpers give, find and withdraw the consent through the API as the mobile app does, for any journey that
// needs the agent (SAL-10).

// giveTranscriptionConsent agrees, as the patient, to the transcription and translation of their consultations
// with the clinic, with AI assistance for the practitioner (the mobile app's notice before the call), and returns
// the consent's id.
func (p *patient) giveTranscriptionConsent(clinic *bookableClinic) string {
	p.t.Helper()
	var c struct{ ID, Kind, Status string }
	p.must(http.StatusCreated, http.MethodPost, "/consents", map[string]string{"organizationId": clinic.id, "kind": "transcription"}, &c)
	if c.Kind != "transcription" || c.Status != "active" {
		p.t.Fatalf("POST /consents (transcription) = %+v, want an active transcription consent", c)
	}
	return c.ID
}

// activeConsent is the id of the patient's active consent of the kind (care or transcription) with the clinic, or
// "" when there is none.
func (p *patient) activeConsent(kind string, clinic *bookableClinic) string {
	p.t.Helper()
	var list struct {
		Consents []struct{ ID, Kind, Status, OrganizationID string }
	}
	p.must(http.StatusOK, http.MethodGet, "/me/consents", nil, &list)
	for _, c := range list.Consents {
		if c.Kind == kind && c.Status == "active" && c.OrganizationID == clinic.id {
			return c.ID
		}
	}
	return ""
}

// transcribed is what the consultation's status says of transcription, as the patient reads it.
func (p *patient) transcribed(appointmentID string) bool {
	p.t.Helper()
	var s struct{ Transcription bool }
	p.must(http.StatusOK, http.MethodGet, "/appointments/"+appointmentID+"/consultation", nil, &s)
	return s.Transcription
}

// agentJoinsAsked counts the transcription_join jobs enqueued for the appointment's consultation.
func agentJoinsAsked(t *testing.T, appointmentID string) int {
	t.Helper()
	var n int
	if err := ownerConn(t).QueryRow(t.Context(), `SELECT count(*) FROM river_job j JOIN video_session s ON s.id::text = j.args->>'session_id'
		WHERE j.kind = 'transcription_join' AND s.appointment_id = $1`, appointmentID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestTranscriptionConsent_DecliningTheCallRunsWithoutTheAgent: the patient does not agree (the app's "Continue
// without transcription" asks for nothing); both join and hear each other; no agent is asked in, none is in the
// room, nothing is transcribed, and the status says the consultation is not transcribed.
func TestTranscriptionConsent_DecliningTheCallRunsWithoutTheAgent(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appt := p.bookFirstTime(clinic)
	startAt(t, appt.ID, -time.Minute)
	if p.transcribed(appt.ID) {
		t.Fatal("the status says transcribed before the patient agreed")
	}

	patient := connect(t, p.api, appt.ID, true)
	practitioner := connect(t, clinic.practitioner, appt.ID, false)
	practitioner.waitForMedia(t, "the patient's tone")
	eventually(t, "the Encounter in progress once the practitioner joined", func() bool {
		return encounterOf(t, p, appt.ID).Status == "in-progress"
	})
	eventually(t, "both joins recorded", func() bool { return p.consultationJoined(appt.ID) })
	captions := transcriptOf(t, encounterOf(t, p, appt.ID).ID)

	// The webhook of both joins has been handled (both times are stored) and asked no agent in; a few seconds on,
	// the practitioner still sees the patient alone and nothing was said into the transcript.
	if n := agentJoinsAsked(t, appt.ID); n != 0 {
		t.Errorf("%d transcription_join jobs, want none without the patient's agreement", n)
	}
	time.Sleep(3 * time.Second) // a quiet period: the absence of an agent cannot be awaited
	if n := len(practitioner.room.GetRemoteParticipants()); n != 1 {
		t.Errorf("the practitioner sees %d participants, want the patient only", n)
	}
	if got := captions(); len(got) != 0 {
		t.Errorf("transcript events %v, want none", got)
	}

	clinic.practitioner.must(http.StatusOK, http.MethodPost, "/appointments/"+appt.ID+"/consultation/end", nil, nil)
	if reason := patient.waitGone(t); reason != livekit.DisconnectReason_ROOM_DELETED {
		t.Errorf("the patient was disconnected with %v, want ROOM_DELETED", reason)
	}
	finished(t, p, appt.ID)
}

// TestTranscriptionConsent_WithdrawingDuringTheCallRemovesOnlyTheAgent: the patient agreed before the call; the
// agent joins and transcribes; the patient withdraws the transcription consent; the agent leaves the room at once,
// the patient and the practitioner stay, the care consent and the appointment are untouched, and the patient may
// still join. The practitioner then ends the consultation.
func TestTranscriptionConsent_WithdrawingDuringTheCallRemovesOnlyTheAgent(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appt := p.bookFirstTime(clinic)
	startAt(t, appt.ID, -time.Minute)
	consent := p.giveTranscriptionConsent(clinic)
	if !p.transcribed(appt.ID) {
		t.Fatal("the status does not say transcribed after the patient agreed")
	}

	patient := connect(t, p.api, appt.ID, true)
	practitioner := connect(t, clinic.practitioner, appt.ID, false)
	practitioner.waitForMedia(t, "the patient's tone")
	eventually(t, "the agent in the room with the patient", func() bool { return len(practitioner.room.GetRemoteParticipants()) == 2 })
	captions := transcriptOf(t, encounterOf(t, p, appt.ID).ID)
	eventually(t, "the patient's placeholder captions on the encounter's stream", func() bool { return captions()["patient final"] })

	p.must(http.StatusOK, http.MethodPost, "/consents/"+consent+"/revoke", nil, nil)

	eventually(t, "the agent gone, the patient still in", func() bool { return len(practitioner.room.GetRemoteParticipants()) == 1 })
	select {
	case <-patient.gone:
		t.Fatal("the patient was disconnected by the withdrawal")
	default:
	}
	if p.transcribed(appt.ID) || p.activeConsent("care", clinic) == "" {
		t.Error("after the withdrawal: want the status not transcribed and the care consent still active")
	}
	if got := p.appointment(appt.ID).Status; got != "booked" {
		t.Errorf("the appointment's status = %q, want booked", got)
	}
	p.must(http.StatusOK, http.MethodPost, "/appointments/"+appt.ID+"/consultation/join", nil, nil)

	clinic.practitioner.must(http.StatusOK, http.MethodPost, "/appointments/"+appt.ID+"/consultation/end", nil, nil)
	if reason := practitioner.waitGone(t); reason != livekit.DisconnectReason_ROOM_DELETED {
		t.Errorf("the practitioner was disconnected with %v, want ROOM_DELETED from the end", reason)
	}
	finished(t, p, appt.ID)
}

// consultationJoined reports whether the consultation's status has both joins.
func (p *patient) consultationJoined(appointmentID string) bool {
	p.t.Helper()
	var s struct{ PatientJoinedAt, PractitionerJoinedAt string }
	p.must(http.StatusOK, http.MethodGet, "/appointments/"+appointmentID+"/consultation", nil, &s)
	return s.PatientJoinedAt != "" && s.PractitionerJoinedAt != ""
}
