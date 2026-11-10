//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The consultation journeys (docs/consultation/IMPLEMENTATION_PLAN.md, package 8), on synthetic data made fresh
// through the API on every run. The practitioner's side of the first is the web app in Chromium with fake media
// devices (test/e2e/web), started by this test; the patient's side of both is a Go SDK participant on the compose
// network that publishes a generated tone, as the mobile app would publish its microphone.

// TestConsultation_ThePractitionerEndsTheCallFromTheWebAppAndTheAppointmentIsFulfilled: the patient joins first and
// waits; the practitioner signs in to the web app, joins from the appointment, sees and hears the patient; the
// Encounter is in-progress; the practitioner ends the consultation, the patient is disconnected, the Encounter is
// finished and the appointment fulfilled. The document the practitioner signed in the call is the patient's to open.
func TestConsultation_ThePractitionerEndsTheCallFromTheWebAppAndTheAppointmentIsFulfilled(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appt := p.bookFirstTime(clinic)
	startAt(t, appt.ID, 5*time.Minute)
	recordSynthetic(t, clinic, p, appt.ID)

	template := newDocumentTemplate(t, clinic)
	term := newExplainedTerm(t, spokenTerm)

	patient := connect(t, p.api, appt.ID, true)
	recordInLiveKitLog(t, "room", patient.room.Name())
	browser := startPractitionerInBrowser(t, clinic, p, appt.ID, patient.join.URL, template, term)

	eventually(t, "the Encounter in progress once the practitioner joined", func() bool {
		return browser.running(t) && encounterOf(t, p, appt.ID).Status == "in-progress"
	})
	patient.waitForMedia(t, "the practitioner's camera and microphone")
	browser.end(t)

	if reason := patient.waitGone(t); reason != livekit.DisconnectReason_ROOM_DELETED {
		t.Errorf("the patient was disconnected with %v, want ROOM_DELETED (the practitioner's end deletes the room)", reason)
	}
	finished(t, p, appt.ID)
	patientOpensTheSignedDocument(t, clinic, p, appt.ID, template, term)
}

// TestConsultation_ARevocationDuringTheCallRemovesThePatient: both are in the call of an appointment that has
// started; the patient revokes the consent; LiveKit disconnects the patient (removed), the practitioner stays, and
// the patient's next token request is 409 consent_required. The practitioner then ends the consultation.
func TestConsultation_ARevocationDuringTheCallRemovesThePatient(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appt := p.bookFirstTime(clinic)
	startAt(t, appt.ID, -time.Minute)
	recordSynthetic(t, clinic, p, appt.ID)

	patient := connect(t, p.api, appt.ID, true)
	recordInLiveKitLog(t, "room", patient.room.Name())
	practitioner := connect(t, clinic.practitioner, appt.ID, false)
	practitioner.waitForMedia(t, "the patient's tone")
	eventually(t, "the Encounter in progress once the practitioner joined", func() bool {
		return encounterOf(t, p, appt.ID).Status == "in-progress"
	})
	// The transcription agent (SAL-4) joined once the practitioner was in and hears the patient's tone: the fake
	// engine's placeholder captions are on the encounter's stream, the patient's, partial and final.
	captions := transcriptOf(t, encounterOf(t, p, appt.ID).ID)
	eventually(t, "the patient's placeholder captions on the encounter's stream", func() bool {
		got := captions()
		return got["patient partial"] && got["patient final"]
	})

	var consents struct{ Consents []struct{ ID string } }
	p.must(http.StatusOK, http.MethodGet, "/me/consents", nil, &consents)
	p.must(http.StatusOK, http.MethodPost, "/consents/"+consents.Consents[0].ID+"/revoke", nil, nil)

	if reason := patient.waitGone(t); reason != livekit.DisconnectReason_PARTICIPANT_REMOVED {
		t.Errorf("the patient was disconnected with %v, want PARTICIPANT_REMOVED", reason)
	}
	join := "/appointments/" + appt.ID + "/consultation/join"
	if status, code := p.refusal(http.MethodPost, join, nil); status != http.StatusConflict || code != "consent_required" {
		t.Errorf("the patient's next join = %d %q, want 409 consent_required", status, code)
	}
	// Nobody left: the agent was removed with the patient (it may not listen once the consent is withdrawn).
	eventually(t, "the practitioner sees the patient and the agent gone", func() bool {
		return len(practitioner.room.GetRemoteParticipants()) == 0
	})

	// The practitioner stayed through the revocation: their first and only disconnection is the end's
	// ROOM_DELETED, not a removal (PARTICIPANT_REMOVED) by the revocation.
	clinic.practitioner.must(http.StatusOK, http.MethodPost, "/appointments/"+appt.ID+"/consultation/end", nil, nil)
	if reason := practitioner.waitGone(t); reason != livekit.DisconnectReason_ROOM_DELETED {
		t.Errorf("the practitioner was disconnected with %v, want ROOM_DELETED from the end (they stay through the revocation)", reason)
	}
	finished(t, p, appt.ID)
}

// finished: the Encounter of the appointment is finished with its period's end, and the appointment fulfilled.
func finished(t *testing.T, p *patient, appointmentID string) {
	t.Helper()
	eventually(t, "the Encounter finished", func() bool { return encounterOf(t, p, appointmentID).Status == "finished" })
	if e := encounterOf(t, p, appointmentID); e.Period.Start == "" || e.Period.End == "" {
		t.Errorf("the finished Encounter's period is %+v, want a start and an end", e.Period)
	}
	if got := p.appointment(appointmentID).Status; got != "fulfilled" {
		t.Errorf("the appointment's status = %q, want fulfilled", got)
	}
}

// encounter is the part of a FHIR Encounter the journeys read.
type encounter struct {
	ID     string
	Status string
	Period struct{ Start, End string }
}

// encounterOf is the patient's FHIR search of the appointment's Encounter: the zero value while there is none.
func encounterOf(t *testing.T, p *patient, appointmentID string) encounter {
	t.Helper()
	var found struct {
		Entry []struct{ Resource encounter }
	}
	p.must(http.StatusOK, http.MethodGet, "/fhir/R4/Encounter?appointment="+appointmentID, nil, &found)
	if len(found.Entry) > 1 {
		t.Fatalf("%d Encounters for the appointment, want at most 1", len(found.Entry))
	}
	if len(found.Entry) == 0 {
		return encounter{}
	}
	return found.Entry[0].Resource
}

// joinAnswer is the join endpoint's 200 (api/openapi.yaml, ConsultationJoin).
type joinAnswer struct {
	URL      string `json:"url"`
	Token    string `json:"token"`
	PeerName string `json:"peerName"`
}

// participant is a test participant connected to the compose LiveKit with the token the API gave its caller.
type participant struct {
	room  *lksdk.Room
	gone  chan time.Time // the time of its first disconnection
	media chan struct{}  // closed on the first RTP packet of a remote track
	join  joinAnswer
}

// connect asks the API for a's token for the appointment and connects with it on the compose network
// (TEST_LIVEKIT_URL: the join answer's URL is the one for a browser on the host). It subscribes to the other side's
// tracks; with tone it publishes a generated tone as its microphone. It leaves at the end of the test.
func connect(t *testing.T, a *api, appointmentID string, tone bool) *participant {
	t.Helper()
	p := &participant{gone: make(chan time.Time, 1), media: make(chan struct{})}
	a.must(http.StatusOK, http.MethodPost, "/appointments/"+appointmentID+"/consultation/join", nil, &p.join)
	var once sync.Once
	cb := lksdk.NewRoomCallback()
	cb.OnDisconnected = func() { // the first disconnection is kept; a repeated callback never blocks the SDK
		select {
		case p.gone <- clock.System{}.Now():
		default:
		}
	}
	cb.OnTrackSubscribed = func(track *webrtc.TrackRemote, _ *lksdk.RemoteTrackPublication, _ *lksdk.RemoteParticipant) {
		go func() {
			for {
				if _, _, err := track.ReadRTP(); err != nil {
					return
				}
				once.Do(func() { close(p.media) })
			}
		}()
	}
	url := strings.Replace(env(t, "TEST_LIVEKIT_URL"), "http", "ws", 1)
	room, err := lksdk.ConnectToRoomWithToken(url, p.join.Token, cb)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	p.room = room
	t.Cleanup(room.Disconnect)
	if tone {
		publishTone(t, room)
	}
	return p
}

// waitForMedia waits up to 30 s for the first media packet from the other side.
func (p *participant) waitForMedia(t *testing.T, what string) {
	t.Helper()
	select {
	case <-p.media:
	case <-time.After(30 * time.Second):
		t.Fatalf("%s did not arrive within 30s", what)
	}
}

// waitGone waits up to 30 s for the participant's disconnection and returns LiveKit's reason for it.
func (p *participant) waitGone(t *testing.T) livekit.DisconnectReason {
	t.Helper()
	select {
	case <-p.gone:
		return p.room.DisconnectReason()
	case <-time.After(30 * time.Second):
		t.Fatal("not disconnected within 30s")
		return 0
	}
}

// publishTone publishes a 440 Hz tone as the room's microphone track: G.711 µ-law (PCMU, 8 kHz), which needs no
// encoder library, in 20 ms frames until the test ends.
func publishTone(t *testing.T, room *lksdk.Room) {
	t.Helper()
	track, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{Source: livekit.TrackSource_MICROPHONE}); err != nil {
		t.Fatalf("publish the tone: %v", err)
	}
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	go func() {
		const rate, frame = 8000, 160 // 20 ms
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for n := 0; ; n++ {
			data := make([]byte, frame)
			for i := range data {
				data[i] = mulaw(int16(8000 * math.Sin(2*math.Pi*440*float64(n*frame+i)/rate)))
			}
			if err := track.WriteSample(media.Sample{Data: data, Duration: 20 * time.Millisecond}, nil); err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}

// mulaw is the G.711 µ-law code of a 16-bit sample.
func mulaw(sample int16) byte {
	const bias, clip = 0x84, 32635
	v, sign := int(sample), byte(0)
	if v < 0 {
		v, sign = -v, 0x80
	}
	v = min(v, clip) + bias
	exp := 7
	for mask := 0x4000; v&mask == 0 && exp > 0; mask >>= 1 {
		exp--
	}
	return ^(sign | byte(exp<<4) | byte((v>>(exp+3))&0x0f))
}

// browser is the web journey (test/e2e/web/consultation.spec.ts) running in a Playwright subprocess.
type browser struct {
	err    error         // the subprocess's exit, once done is closed
	done   chan struct{} // closed when the subprocess has exited
	output *bytes.Buffer // its output, read once done is closed
	signal string        // the file whose creation tells the spec to end the consultation
}

// startPractitionerInBrowser starts the Playwright spec: Chromium with fake devices signs the practitioner in to the
// web app with the password and a TOTP code, joins the appointment's consultation, checks the patient is seen and
// heard, and waits for the test's signal to end it. livekitURL is the join answer's URL, which the spec forwards
// to the compose LiveKit as Docker forwards it on a developer's host.
// newDocumentTemplate makes a synthetic English template in the clinic, one required field and one optional, and
// returns its title.
func newDocumentTemplate(t *testing.T, clinic *bookableClinic) string {
	t.Helper()
	title := "Placeholder e2e findings " + suffix(t)
	clinic.practitioner.must(http.StatusCreated, http.MethodPost, "/document-templates", map[string]any{
		"title": title, "language": "en", "status": "draft", "typeCode": "11488-4",
		"fields": []map[string]any{
			{"key": "reason", "type": "text", "label": "Placeholder reason", "required": true},
			{"key": "plan", "type": "text", "label": "Placeholder plan", "required": false},
		},
	}, nil)
	return title
}

func startPractitionerInBrowser(t *testing.T, clinic *bookableClinic, p *patient, appointmentID, livekitURL, template string, term explainedTerm) *browser {
	t.Helper()
	b := &browser{done: make(chan struct{}), output: &bytes.Buffer{}, signal: filepath.Join(t.TempDir(), "end")}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	cmd := exec.CommandContext(ctx, "web/node_modules/.bin/playwright", "test", "--config", "web/playwright.config.ts", "consultation.spec.ts")
	cmd.Env = append(os.Environ(),
		"E2E_APPOINTMENT_ID="+appointmentID,
		"E2E_PRACTITIONER_EMAIL="+clinic.practitionerAddr,
		"E2E_PASSWORD="+password,
		"E2E_TOTP_SECRET="+hex.EncodeToString(clinic.totp.secret),
		fmt.Sprintf("E2E_TOTP_LAST_STEP=%d", clinic.totp.last),
		"E2E_PATIENT_NAME="+p.name,
		"E2E_LIVEKIT_PUBLIC_URL="+livekitURL,
		"E2E_END_SIGNAL="+b.signal,
		"E2E_TEMPLATE_TITLE="+template,
		"E2E_TERM_LABEL="+term.label,
		"E2E_TERM_EXPLANATION="+term.explanation,
	)
	cmd.Stdout, cmd.Stderr = b.output, b.output
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start Playwright: %v", err)
	}
	go func() {
		b.err = cmd.Wait()
		cancel()
		close(b.done)
	}()
	t.Cleanup(func() {
		cancel()
		<-b.done
		if t.Failed() {
			t.Logf("Playwright:\n%s", b.output)
		}
	})
	return b
}

// running fails the test when the web journey has already exited: before the end signal it can only have failed.
func (b *browser) running(t *testing.T) bool {
	t.Helper()
	select {
	case <-b.done:
		t.Fatalf("the web journey failed before the end: %v", b.err)
	default:
	}
	return true
}

// end signals the spec to end the consultation and waits for it to pass.
func (b *browser) end(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(b.signal, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	<-b.done
	if b.err != nil {
		t.Fatalf("the web journey failed: %v", b.err)
	}
}

// recordSynthetic records the run's synthetic names, e-mails and ids that LiveKit must never see; `make e2e`
// checks its log for them after the journeys (scripts/e2e-livekit-log.sh).
func recordSynthetic(t *testing.T, clinic *bookableClinic, p *patient, appointmentID string) {
	t.Helper()
	given, family, _ := strings.Cut(p.name, " ")
	for _, v := range [][2]string{
		{"appointment id", appointmentID}, {"patient id", p.patientID}, {"clinic id", clinic.id},
		{"practitioner id", clinic.practitionerID}, {"patient e-mail", p.addr}, {"practitioner e-mail", clinic.practitionerAddr},
		{"patient name", p.name}, {"patient given name", given}, {"patient family name", family},
		{"practitioner name", clinic.practitionerName}, {"clinic name", clinic.name},
	} {
		appendLiveKitCheck(t, "absent", v[0], v[1])
	}
}

// recordInLiveKitLog records a value LiveKit's log must hold (an opaque room name of the run), so the check reads
// the right log.
func recordInLiveKitLog(t *testing.T, label, value string) {
	t.Helper()
	appendLiveKitCheck(t, "present", label, value)
}

func appendLiveKitCheck(t *testing.T, kind, label, value string) {
	t.Helper()
	f, err := os.OpenFile(env(t, "E2E_LIVEKIT_CHECK"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := fmt.Fprintf(f, "%s\t%s\t%s\n", kind, label, value); err != nil {
		t.Fatal(err)
	}
}

// startAt moves the synthetic appointment id to start in from now, for 30 minutes, as the owner (DATABASE_URL):
// the slots start at least an hour ahead.
func startAt(t *testing.T, id string, in time.Duration) {
	t.Helper()
	conn := ownerConn(t)
	if _, err := conn.Exec(t.Context(), `UPDATE appointment SET period = tstzrange(date_trunc('second', now()) + make_interval(secs => $2),
		date_trunc('second', now()) + make_interval(secs => $2) + interval '30 minutes') WHERE id = $1`, id, in.Seconds()); err != nil {
		t.Fatal(err)
	}
}

func ownerConn(t *testing.T) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), env(t, "DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

// transcriptOf reads the encounter's event stream in the compose Redis (E2E_REDIS_URL) and gives, on each call,
// which speakers' partial and final transcript events it holds ("patient partial", "patient final", ...).
func transcriptOf(t *testing.T, encounterID string) func() map[string]bool {
	t.Helper()
	opts, err := goredis.ParseURL(env(t, "E2E_REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	rdb := goredis.NewClient(opts)
	t.Cleanup(func() { _ = rdb.Close() })
	return func() map[string]bool {
		entries, err := rdb.XRange(t.Context(), "consult:"+encounterID+":events", "-", "+").Result()
		if err != nil {
			t.Fatalf("read the encounter's stream: %v", err)
		}
		got := map[string]bool{}
		for _, e := range entries {
			var env struct {
				Type    string
				Payload struct {
					Speaker string `json:"speaker"`
					IsFinal bool   `json:"is_final"`
				}
			}
			raw, _ := e.Values["envelope"].(string)
			if json.Unmarshal([]byte(raw), &env) != nil || env.Type != "transcript" {
				continue
			}
			kind := "partial"
			if env.Payload.IsFinal {
				kind = "final"
			}
			got[env.Payload.Speaker+" "+kind] = true
		}
		return got
	}
}
