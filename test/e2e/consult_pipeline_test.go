//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The consultation pipeline's journeys (SAL-10, docs/plans/v1-finish.md), on the fake engines playing a scripted
// conversation (test/testdata/consult): a German, an Arabic and a Turkish patient with a Bosnian practitioner. The
// speakers' LiveKit participants send generated tones, the transcription agent in the room hears them, the fake
// speech engine says each tone's line and the fake model translates it; the events go through the encounter's Redis
// stream to the practitioner's WebSocket and are stored; the suggestions worker suggests from the stored lines, the
// practitioner pins one and dismisses another; a document is drafted from the transcript, edited and signed. The
// German journey's practitioner is the web app in Chromium (test/e2e/web/consult-pipeline.spec.ts); the Arabic and
// Turkish ones are Go participants on the event stream, and the Turkish patient withdraws the transcription
// consent mid-call.

// bookedIn is the booking confirmation's subject in the patient's language (internal/notification/templates).
var bookedIn = map[string]string{"de": germanBooked, "ar": "تم حجز موعدك", "tr": "Randevunuz oluşturuldu"}

// editedValue is what the practitioner types over the drafted value of the second field before signing.
const editedValue = "Primjer: vrijednost izmijenjena u e2e testu."

// practitionerLanguage is the practitioner's language: the clinic's default, Bosnian.
const practitionerLanguage = "bs"

func TestConsultPipeline_AnArabicPatientAndABosnianPractitioner(t *testing.T) {
	c := newPipelineCall(t, "ar")
	_, practitioner, voices := c.joinFromGo(t)
	c.converse(t, voices)
	c.pinAndDismissOverTheStream(t)
	c.transcriptStored(t)
	c.draftEditAndSign(t)
	latencies.check(t, c.journey)

	c.clinic.practitioner.must(http.StatusOK, http.MethodPost, "/appointments/"+c.apptID+"/consultation/end", nil, nil)
	if reason := practitioner.waitGone(t); reason != livekit.DisconnectReason_ROOM_DELETED {
		t.Errorf("the practitioner was disconnected with %v, want ROOM_DELETED", reason)
	}
	finished(t, c.p, c.apptID)
}

func TestConsultPipeline_ATurkishPatientWithdrawsTheTranscriptionMidCallAndTheAgentLeaves(t *testing.T) {
	c := newPipelineCall(t, "tr")
	patient, practitioner, voices := c.joinFromGo(t)
	c.converse(t, voices)
	c.pinAndDismissOverTheStream(t)
	c.transcriptStored(t)
	c.draftEditAndSign(t)
	latencies.check(t, c.journey)

	// The patient withdraws the transcription consent (SAL-12): the agent leaves at once, the call goes on, and the
	// document can no longer be drafted from the conversation.
	consent := c.p.activeConsent("transcription", c.clinic)
	if consent == "" {
		t.Fatal("the patient has no active transcription consent with the clinic")
	}
	c.p.must(http.StatusOK, http.MethodPost, "/consents/"+consent+"/revoke", nil, nil)
	eventually(t, "the agent gone, the patient still in", func() bool {
		return !agentIn(practitioner.room) && len(practitioner.room.GetRemoteParticipants()) == 1
	})
	select {
	case <-patient.gone:
		t.Fatal("the patient was disconnected by the withdrawal of the transcription")
	default:
	}
	if got, code := c.clinic.practitioner.refusal(http.MethodPost, "/encounters/"+c.encounterID+"/drafts",
		map[string]string{"templateId": c.template.id, "language": "bs"}); got != http.StatusForbidden || code != "transcription_consent_required" {
		t.Errorf("drafting after the withdrawal = %d %q, want 403 transcription_consent_required", got, code)
	}

	c.clinic.practitioner.must(http.StatusOK, http.MethodPost, "/appointments/"+c.apptID+"/consultation/end", nil, nil)
	if reason := practitioner.waitGone(t); reason != livekit.DisconnectReason_ROOM_DELETED {
		t.Errorf("the practitioner was disconnected with %v, want ROOM_DELETED (they stay through the withdrawal)", reason)
	}
	finished(t, c.p, c.apptID)
}

func TestConsultPipeline_AGermanPatientAndTheBosnianPractitionerInTheWebApp(t *testing.T) {
	c := newPipelineCall(t, "de")
	startAt(t, c.apptID, 5*time.Minute)
	patient := connect(t, c.p.api, c.apptID, false)
	recordInLiveKitLog(t, "room", patient.room.Name())
	voices := map[string]*voice{"patient": speak(t, patient.room)}
	draftTiming := filepath.Join(t.TempDir(), "draft.json")
	b := startSpec(t, c.clinic, c.p, c.apptID, patient.join.URL, "consult-pipeline.spec.ts",
		"E2E_CONVERSATION="+c.conv.path, "E2E_TEMPLATE_TITLE="+c.template.title, "E2E_EDITED_VALUE="+editedValue,
		"E2E_DRAFT_TIMING="+draftTiming)
	eventually(t, "the Encounter in progress once the practitioner joined", func() bool {
		return b.running(t) && encounterOf(t, c.p, c.apptID).Status == "in-progress"
	})
	c.listen(t, "patient")
	c.converse(t, voices)

	// The web journey shows the lines, pins and dismisses the suggestions, drafts, edits and signs.
	waitFor(t, "the document signed in the web app", 3*time.Minute, func() bool {
		return b.running(t) && c.signedDraft(t) != nil
	})
	c.checkSigned(t, *c.signedDraft(t))
	c.storedStatuses(t)
	c.transcriptStored(t)
	b.end(t)

	var timing struct{ DraftMs float64 }
	if raw, err := os.ReadFile(draftTiming); err != nil || json.Unmarshal(raw, &timing) != nil || timing.DraftMs <= 0 {
		t.Errorf("the web journey's draft timing %s: %v", raw, err)
	} else {
		latencies.add(c.journey, hopDraft, time.Duration(timing.DraftMs*float64(time.Millisecond)))
	}
	latencies.check(t, c.journey)
	finished(t, c.p, c.apptID)
}

// pipelineCall is one journey's consultation.
type pipelineCall struct {
	clinic      *bookableClinic
	p           *patient
	events      *events
	segments    map[string]string // a line's text to its final segment's id
	language    string
	journey     string
	apptID      string
	encounterID string
	template    scriptedTemplate
	conv        conversation
}

// newPipelineCall books a synthetic appointment of a patient speaking language with a clinic whose default
// language is Bosnian, and makes the clinic's template of the conversation.
func newPipelineCall(t *testing.T, language string) *pipelineCall {
	t.Helper()
	c := &pipelineCall{conv: loadConversation(t, language), clinic: newBookableClinic(t), language: language,
		journey: language + " patient, bs practitioner", segments: map[string]string{}}
	c.p = newPatientIn(t, language)
	c.apptID = c.p.bookIn(c.clinic, freeSlots(t, c.p.api, c.clinic.scheduleID)[0], bookedIn[language])
	recordSynthetic(t, c.clinic, c.p, c.apptID)
	recordConversation(t, c.conv)
	c.p.giveTranscriptionConsent(c.clinic) // the app's notice before the call (SAL-12): the agent may join
	c.template = newScriptedTemplate(t, c.clinic, c.conv)
	return c
}

// joinFromGo starts the consultation with both sides as Go participants with a microphone each, the patient first,
// and waits until the agent is in the room and listening to both.
func (c *pipelineCall) joinFromGo(t *testing.T) (patient, practitioner *participant, voices map[string]*voice) {
	t.Helper()
	startAt(t, c.apptID, -time.Minute)
	patient = connect(t, c.p.api, c.apptID, false)
	recordInLiveKitLog(t, "room", patient.room.Name())
	practitioner = connect(t, c.clinic.practitioner, c.apptID, false)
	voices = map[string]*voice{"patient": speak(t, patient.room), "practitioner": speak(t, practitioner.room)}
	eventually(t, "the Encounter in progress once the practitioner joined", func() bool {
		return encounterOf(t, c.p, c.apptID).Status == "in-progress"
	})
	eventually(t, "the transcription agent in the room", func() bool { return agentIn(practitioner.room) })
	c.listen(t, "patient", "practitioner")
	return patient, practitioner, voices
}

// listen opens the practitioner's event stream and waits for the agent to hear the speakers: the fake engine's
// placeholder line of each microphone, said as its recognition opens.
func (c *pipelineCall) listen(t *testing.T, speakers ...string) {
	t.Helper()
	c.encounterID = encounterOf(t, c.p, c.apptID).ID
	c.events = openEvents(t, c.clinic.practitioner, c.encounterID)
	for _, s := range speakers {
		c.events.transcript(t, "the agent hears the "+s, func(e transcriptLine) bool { return e.Speaker == s && e.IsFinal })
	}
}

// converse says the conversation's lines of the speakers with a voice, one after another, and checks each one's
// partial and translated final on the practitioner's event stream, measuring both.
func (c *pipelineCall) converse(t *testing.T, voices map[string]*voice) {
	t.Helper()
	for i, l := range c.conv.Lines {
		v, ok := voices[l.Speaker]
		if !ok {
			continue
		}
		what := "line " + strconv.Itoa(i+1) + " (" + l.Speaker + ")"
		on, off := v.line(t, l.Tone)
		_, partialAt := c.events.transcript(t, "the partial of "+what, func(e transcriptLine) bool {
			return !e.IsFinal && e.Speaker == l.Speaker && e.Original == firstHalf(l.Text)
		})
		final, finalAt := c.events.transcript(t, "the final of "+what, func(e transcriptLine) bool {
			return e.IsFinal && e.Speaker == l.Speaker && e.Original == l.Text
		})
		if want := l.Translations[c.listener(l.Speaker)]; final.Translation != want {
			t.Errorf("%s is translated as %q, want %q", what, final.Translation, want)
		}
		if final.Lang != l.Lang && !strings.HasPrefix(final.Lang, l.Lang+"-") {
			t.Errorf("%s was recognised in %q, want %s", what, final.Lang, l.Lang)
		}
		c.segments[l.Text] = final.SegmentID
		latencies.add(c.journey, hopCaption, partialAt.Sub(on))
		latencies.add(c.journey, hopTranslation, finalAt.Sub(off))
	}
}

// listener is the language the other participant reads a speaker's line in.
func (c *pipelineCall) listener(speaker string) string {
	if speaker == "patient" {
		return practitionerLanguage
	}
	return c.language
}

// pinAndDismissOverTheStream waits for the conversation's two suggestions, each citing its line and labelled as
// AI, pins the first and dismisses the second with the stream's commands, as the web app does, and sees both
// changes come back and stored.
func (c *pipelineCall) pinAndDismissOverTheStream(t *testing.T) {
	t.Helper()
	for i, status := range []string{"pinned", "dismissed"} {
		want := c.conv.Suggestions[i]
		s := c.events.suggestion(t, want.Text, "shown")
		if s.Origin != "ai" || s.Type != want.Type || !slices.Equal(s.SourceSegmentIDs, []string{c.segments[want.After]}) {
			t.Errorf("suggestion %d is %+v, want an ai %s citing its line", i+1, s, want.Type)
		}
		c.events.send(t, "suggestion_status", map[string]string{"id": s.ID, "status": status})
		c.events.suggestion(t, want.Text, status)
	}
	c.storedStatuses(t)
}

// storedStatuses: the first suggestion is stored pinned and the second dismissed.
func (c *pipelineCall) storedStatuses(t *testing.T) {
	t.Helper()
	rows, err := ownerConn(t).Query(t.Context(), "SELECT text, status FROM suggestion WHERE encounter_id = $1", c.encounterID)
	if err != nil {
		t.Fatal(err)
	}
	stored := map[string]string{}
	got, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (struct{}, error) {
		var text, status string
		err := r.Scan(&text, &status)
		stored[text] = status
		return struct{}{}, err
	})
	if err != nil || len(got) != 2 {
		t.Fatalf("the encounter's stored suggestions: %d, %v; want 2", len(got), err)
	}
	if a, b := stored[c.conv.Suggestions[0].Text], stored[c.conv.Suggestions[1].Text]; a != "pinned" || b != "dismissed" {
		t.Errorf("the suggestions are stored %q and %q, want pinned and dismissed", a, b)
	}
}

// transcriptStored: every line said is in the encounter's stored transcript with its translation.
func (c *pipelineCall) transcriptStored(t *testing.T) {
	t.Helper()
	eventually(t, "every line said in the stored transcript with its translation", func() bool {
		var stored struct{ Segments []transcriptLine }
		c.clinic.practitioner.must(http.StatusOK, http.MethodGet, "/encounters/"+c.encounterID+"/transcript", nil, &stored)
		for _, l := range c.conv.Lines {
			id, said := c.segments[l.Text]
			if said && !slices.ContainsFunc(stored.Segments, func(s transcriptLine) bool {
				return s.SegmentID == id && s.Original == l.Text && s.Translation == l.Translations[c.listener(l.Speaker)]
			}) {
				return false
			}
		}
		return true
	})
}

// draftEvent is a document draft as the API answers it (DraftEvent).
type draftEvent struct {
	DraftID string `json:"draft_id"`
	Status  string `json:"status"`
	Fields  []struct {
		Key              string   `json:"key"`
		Value            string   `json:"value"`
		SourceSegmentIDs []string `json:"source_segment_ids"`
		Missing          bool     `json:"missing"`
	} `json:"fields"`
	VersionID int `json:"version_id"`
}

// draftEditAndSign drafts the template from the transcript (measured), checks each field is filled from its line
// or flagged missing, types over the second field and signs.
func (c *pipelineCall) draftEditAndSign(t *testing.T) {
	t.Helper()
	pr := c.clinic.practitioner
	var d draftEvent
	began := clock.System{}.Now()
	pr.must(http.StatusOK, http.MethodPost, "/encounters/"+c.encounterID+"/drafts", map[string]string{"templateId": c.template.id, "language": "bs"}, &d)
	latencies.add(c.journey, hopDraft, clock.System{}.Now().Sub(began))
	values := make([]map[string]string, len(d.Fields))
	for i, f := range d.Fields {
		want := c.conv.Fields[i]
		switch {
		case f.Key != want.Key:
			t.Fatalf("drafted field %d is %q, want %q", i, f.Key, want.Key)
		case want.From == "" && !f.Missing:
			t.Errorf("the field %s is drafted %+v, want it missing", f.Key, f)
		case want.From != "" && (f.Value != want.Value || !slices.Equal(f.SourceSegmentIDs, []string{c.segments[want.From]})):
			t.Errorf("the field %s is drafted %+v, want %q citing its line", f.Key, f, want.Value)
		}
		values[i] = map[string]string{"key": f.Key, "value": f.Value}
	}
	values[1]["value"] = editedValue
	var signed draftEvent
	pr.mustWith(http.Header{"If-Match": {`W/"` + strconv.Itoa(d.VersionID) + `"`}}, http.StatusOK, http.MethodPost,
		"/drafts/"+d.DraftID+"/sign", map[string]any{"fields": values}, &signed)
	c.checkSigned(t, signed)
}

// checkSigned: the document is final, the first field as drafted with its source, the second as typed with none,
// the third empty.
func (c *pipelineCall) checkSigned(t *testing.T, d draftEvent) {
	t.Helper()
	if d.Status != "final" || len(d.Fields) != 3 {
		t.Fatalf("the signed draft is %+v, want final with 3 fields", d)
	}
	first, second, third := d.Fields[0], d.Fields[1], d.Fields[2]
	if first.Value != c.conv.Fields[0].Value || !slices.Equal(first.SourceSegmentIDs, []string{c.segments[c.conv.Fields[0].From]}) {
		t.Errorf("the signed first field is %+v, want its drafted value and source", first)
	}
	if second.Value != editedValue || len(second.SourceSegmentIDs) != 0 {
		t.Errorf("the signed second field is %+v, want the typed value without a source", second)
	}
	if third.Value != "" {
		t.Errorf("the signed third field is %+v, want it empty", third)
	}
}

// signedDraft is the encounter's signed draft, or nil while there is none.
func (c *pipelineCall) signedDraft(t *testing.T) *draftEvent {
	t.Helper()
	var all struct{ Drafts []draftEvent }
	c.clinic.practitioner.must(http.StatusOK, http.MethodGet, "/encounters/"+c.encounterID+"/drafts", nil, &all)
	for _, d := range all.Drafts {
		if d.Status == "final" {
			return &d
		}
	}
	return nil
}

// scriptedTemplate is the clinic's template of a conversation's fields.
type scriptedTemplate struct{ id, title string }

// newScriptedTemplate makes the clinic's Bosnian template with the conversation's fields, which the fake model
// fills from their lines.
func newScriptedTemplate(t *testing.T, clinic *bookableClinic, c conversation) scriptedTemplate {
	t.Helper()
	fields := make([]map[string]any, len(c.Fields))
	for i, f := range c.Fields {
		fields[i] = map[string]any{"key": f.Key, "type": "text", "label": f.Label, "required": f.Required}
	}
	tmpl := scriptedTemplate{title: "Primjer: e2e nalaz " + suffix(t)}
	var made struct{ ID string }
	clinic.practitioner.must(http.StatusCreated, http.MethodPost, "/document-templates", map[string]any{
		"title": tmpl.title, "language": "bs", "status": "draft", "typeCode": "11488-4", "fields": fields,
	}, &made)
	tmpl.id = made.ID
	return tmpl
}

// agentIn reports whether the transcription agent is in the room, as one of its participants sees it.
func agentIn(room *lksdk.Room) bool {
	return slices.ContainsFunc(room.GetRemoteParticipants(), func(p *lksdk.RemoteParticipant) bool {
		return p.Kind() == lksdk.ParticipantAgent
	})
}

// waitFor is eventually with its own timeout, checking every 250 ms.
func waitFor(t *testing.T, what string, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.After(timeout)
	for !cond() {
		select {
		case <-deadline:
			t.Fatalf("%s: not within %v", what, timeout)
		case <-time.After(250 * time.Millisecond):
		}
	}
}
