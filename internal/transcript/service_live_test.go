package transcript_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
	"github.com/Shmeeshmee/salus/pkg/stt"
	"github.com/Shmeeshmee/salus/pkg/stt/stttest"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// Synthetic ids.
const (
	sessionID = "5e000000-0000-4000-8000-000000000001"
	clinicID  = "c0000000-0000-4000-8000-00000000000c"
	apptID    = "e1000000-0000-4000-8000-0000000000a1"
	accountID = "acc00000-0000-4000-8000-0000000000aa"
)

var now = time.Date(2026, 11, 9, 9, 30, 0, 0, time.UTC)

// fakeCalls puts the agent in a room whose tracks the test feeds; it records every listen and every leave.
type fakeCalls struct {
	err    error
	end    func() // closes the tracks once
	tracks chan videosession.AudioTrack
	mu     sync.Mutex
	joins  int
	leaves int
}

func newCalls() *fakeCalls { return &fakeCalls{} }

func (c *fakeCalls) AgentListen(_ context.Context, id, org string) (videosession.AgentCall, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if id != sessionID || org != clinicID {
		return videosession.AgentCall{}, videosession.ErrNotFound
	}
	if c.err != nil {
		return videosession.AgentCall{}, c.err
	}
	c.joins++
	tracks := make(chan videosession.AudioTrack, 4)
	var once sync.Once
	c.tracks, c.end = tracks, func() { once.Do(func() { close(tracks) }) }
	end := c.end
	return videosession.AgentCall{
		Tracks: tracks, EncounterID: encounterID, AppointmentID: apptID,
		Leave: func() {
			c.mu.Lock()
			c.leaves++
			c.mu.Unlock()
			end()
		},
	}, nil
}

// publish hands a track to the agent and returns its frames' sender.
func (c *fakeCalls) publish(role videosession.Role) chan []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	frames := make(chan []byte, 10)
	c.tracks <- videosession.AudioTrack{Role: role, Frames: frames}
	return frames
}

// remove is LiveKit removing the agent: its tracks end.
func (c *fakeCalls) remove() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.end()
}

func (c *fakeCalls) counts() (int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.joins, c.leaves
}

// fakeLanguages is the appointment's notice, the patient's account language and the clinic's default.
type fakeLanguages struct {
	err      error
	patient  string
	clinic   string
	agents   []string
	accounts []string
}

func (l *fakeLanguages) Notice(_ context.Context, agent, org, id string) (appointment.Notice, error) {
	l.agents = append(l.agents, agent)
	if l.err != nil {
		return appointment.Notice{}, l.err
	}
	return appointment.Notice{AccountID: accountID, Appointment: appointment.Appointment{ID: id, OrganizationID: org}}, nil
}

func (l *fakeLanguages) AccountLanguage(_ context.Context, agent, id string) (string, error) {
	l.agents, l.accounts = append(l.agents, agent), append(l.accounts, id)
	return l.patient, nil
}

func (l *fakeLanguages) DefaultLanguage(context.Context, string) (string, error) {
	return l.clinic, nil
}

// fakePublisher keeps every envelope put on an encounter's stream, and signals each.
type fakePublisher struct {
	err  error
	sent chan published
}

type published struct {
	encounter string
	envelope  []byte
}

func (p *fakePublisher) Publish(_ context.Context, encounterID string, envelope []byte) error {
	if p.err != nil {
		return p.err
	}
	p.sent <- published{encounter: encounterID, envelope: envelope}
	return nil
}

type liveFixture struct {
	calls      *fakeCalls
	languages  *fakeLanguages
	publisher  *fakePublisher
	stt        *stttest.Recognizer
	translator *llmtest.Translator
	glossary   *fakeGlossary
	logs       *syncBuffer
	svc        *transcript.Live
}

func newLiveFixture(t *testing.T, script ...stt.Result) *liveFixture {
	t.Helper()
	f := &liveFixture{
		calls: newCalls(), languages: &fakeLanguages{patient: "tr", clinic: "bs"},
		publisher: &fakePublisher{sent: make(chan published, 100)}, stt: &stttest.Recognizer{Results: script},
		translator: &llmtest.Translator{}, glossary: &fakeGlossary{}, logs: &syncBuffer{},
	}
	f.svc = f.service(f.stt)
	t.Cleanup(f.svc.Stop)
	return f
}

// service is the Live on the fixture's ports and r; budget, when given, is its translation budget.
func (f *liveFixture) service(r stt.Recognizer, budget ...time.Duration) *transcript.Live {
	p := transcript.LivePorts{
		Calls: f.calls, Appointments: f.languages, Accounts: f.languages, Clinics: f.languages,
		Publisher: f.publisher, Recognizer: r, Translator: f.translator, Glossary: f.glossary, Terms: f.glossary,
		Clock: clock.Fixed{T: now}, Retry: time.Millisecond, Log: slog.New(slog.NewJSONHandler(f.logs, nil)),
	}
	if len(budget) > 0 {
		p.TranslateBudget = budget[0]
	}
	return transcript.NewLive(p)
}

// next waits for the next envelope on the stream and decodes its event.
func (f *liveFixture) next(t *testing.T) (string, transcript.Event) {
	t.Helper()
	select {
	case p := <-f.publisher.sent:
		env, err := wire.Decode(p.envelope)
		if err != nil || env.Type != transcript.MessageType || env.Version != transcript.MessageVersion {
			t.Fatalf("envelope %s: %v; want a transcript v1", p.envelope, err)
		}
		var e transcript.Event
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			t.Fatal(err)
		}
		return p.encounter, e
	case <-time.After(5 * time.Second):
		t.Fatal("no event on the stream within 5 s")
		return "", transcript.Event{}
	}
}

// waitFor polls cond for up to 5 s.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	wall := clock.System{}
	deadline := wall.Now().Add(5 * time.Second)
	for !cond() {
		if wall.Now().After(deadline) {
			t.Fatalf("%s: not within 5 s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func script() []stt.Result {
	return []stt.Result{
		{Text: "placeholder", Start: 200 * time.Millisecond, End: 600 * time.Millisecond},
		{Text: "placeholder words", Lang: "tr-TR", Start: 200 * time.Millisecond, End: 900 * time.Millisecond},
		{Text: "Placeholder words.", Lang: "tr-TR", Start: 200 * time.Millisecond, End: 1000 * time.Millisecond, Final: true},
		{Text: "Next", Start: 1500 * time.Millisecond, End: 1700 * time.Millisecond},
	}
}

// The agent joins the session's room and, for each person's microphone, opens a recognition in their language
// at 16 kHz (the patient's: their account's; the practitioner's: the clinic's), sends it the audio, and puts each
// result on the encounter's stream as a transcript envelope: partial ones revising one segment until the final,
// the next result a new segment, with the times of the audio; partial ones are not translated (SAL-5).
func TestTranscribe_PutsEachSpeakersSegmentsOnTheEncountersStream(t *testing.T) {
	f := newLiveFixture(t, script()...)
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatalf("Transcribe() = %v", err)
	}
	frames := f.calls.publish(videosession.RolePatient)
	frames <- []byte{1, 2}
	frames <- []byte{3, 4}

	var events []transcript.Event
	for range 4 {
		encounter, e := f.next(t)
		if encounter != encounterID {
			t.Errorf("published on encounter %q, want %q", encounter, encounterID)
		}
		events = append(events, e)
	}
	first := events[0]
	want := transcript.Event{
		EncounterID: encounterID, SegmentID: first.SegmentID, Speaker: transcript.SpeakerPatient, Lang: "tr-TR",
		Original: "placeholder", TsStart: now.Add(200 * time.Millisecond), TsEnd: now.Add(600 * time.Millisecond),
	}
	if first != want {
		t.Errorf("first event %+v,\nwant %+v", first, want)
	}
	if err := first.Validate(); err != nil {
		t.Errorf("the first event is not valid: %v", err)
	}
	// The final is published once translated (SAL-5), so the next segment's partial may come before it.
	var segment, next []transcript.Event
	for _, e := range events[1:] {
		if e.SegmentID == first.SegmentID {
			segment = append(segment, e)
		} else {
			next = append(next, e)
		}
	}
	if len(segment) != 2 || segment[0].IsFinal || !segment[1].IsFinal {
		t.Errorf("events %+v; want two partials and the final of one segment", events)
	}
	if len(next) != 1 || next[0].IsFinal || next[0].Original != "Next" {
		t.Errorf("events %+v; want a partial of a new segment", next)
	}

	streams := f.stt.Streams()
	if len(streams) != 1 || streams[0].Config != (stt.Config{Lang: "tr-TR", SampleRate: 16000}) {
		t.Fatalf("streams %+v; want one in tr-TR at 16 kHz", streams)
	}
	waitFor(t, "the audio sent", func() bool { return string(streams[0].Audio()) == string([]byte{1, 2, 3, 4}) })
	if !slices.Equal(f.languages.agents, []string{"transcription_agent", "transcription_agent"}) || !slices.Equal(f.languages.accounts, []string{accountID}) {
		t.Errorf("languages read by %v for %v; want the agent reading the patient's account", f.languages.agents, f.languages.accounts)
	}
}

// The practitioner speaks the clinic's language, as does a patient whose account has none; each language is
// recognised in the locale its speakers use (Arabic as ar-SA).
func TestTranscribe_RecognisesEachSpeakerInTheirLocale(t *testing.T) {
	for _, tc := range []struct {
		role            videosession.Role
		patient, clinic string
		want            string
		speaker         transcript.Speaker
	}{
		{role: videosession.RolePractitioner, patient: "tr", clinic: "bs", want: "bs-BA", speaker: transcript.SpeakerPractitioner},
		{role: videosession.RolePatient, patient: "", clinic: "hr", want: "hr-HR", speaker: transcript.SpeakerPatient},
		{role: videosession.RolePatient, patient: "ar", clinic: "bs", want: "ar-SA", speaker: transcript.SpeakerPatient},
		{role: videosession.RolePatient, patient: "de", clinic: "bs", want: "de-DE", speaker: transcript.SpeakerPatient},
		{role: videosession.RolePatient, patient: "sr-Cyrl", clinic: "bs", want: "sr-RS", speaker: transcript.SpeakerPatient},
	} {
		f := newLiveFixture(t, stt.Result{Text: "placeholder", Final: true})
		f.languages.patient, f.languages.clinic = tc.patient, tc.clinic
		if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
			t.Fatal(err)
		}
		f.calls.publish(tc.role)
		_, e := f.next(t)
		if e.Lang != tc.want || e.Speaker != tc.speaker {
			t.Errorf("%s with patient %q, clinic %q: lang %q, speaker %q; want %q, %q", tc.role, tc.patient, tc.clinic, e.Lang, e.Speaker, tc.want, tc.speaker)
		}
	}
}

// The agent joins a session once: another join while it is in the room is nothing; after it was removed, the
// next one joins again.
func TestTranscribe_JoinsASessionOnce(t *testing.T) {
	f := newLiveFixture(t)
	for range 3 {
		if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
			t.Fatal(err)
		}
	}
	if joins, _ := f.calls.counts(); joins != 1 {
		t.Fatalf("%d joins, want 1", joins)
	}
	f.calls.remove()
	waitFor(t, "the next join after the removal", func() bool {
		_ = f.svc.Transcribe(t.Context(), sessionID, clinicID)
		joins, _ := f.calls.counts()
		return joins == 2
	})
}

// A refusal (consent withdrawn, the consultation ended) is returned and joins nothing; a later request may.
func TestTranscribe_ReturnsTheRefusal(t *testing.T) {
	f := newLiveFixture(t)
	f.calls.err = videosession.ErrConsentRequired
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); !errors.Is(err, videosession.ErrConsentRequired) {
		t.Fatalf("Transcribe() = %v, want ErrConsentRequired", err)
	}
	f.calls.err = nil
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatalf("Transcribe() after the refusal = %v", err)
	}
	if joins, _ := f.calls.counts(); joins != 1 {
		t.Errorf("%d joins, want 1", joins)
	}
}

// Without the languages the agent cannot transcribe: it leaves the room and returns the error.
func TestTranscribe_LeavesWhenTheLanguagesCannotBeRead(t *testing.T) {
	f := newLiveFixture(t)
	f.languages.err = errors.New("database down")
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err == nil {
		t.Fatal("Transcribe() = nil, want the error")
	}
	if _, leaves := f.calls.counts(); leaves != 1 {
		t.Errorf("left %d times, want once", leaves)
	}
}

// blocking is a recognizer whose streams give nothing until their context ends, as a real engine waits for
// speech; it counts the streams open.
type blocking struct {
	mu   sync.Mutex
	open int
}

func (b *blocking) Stream(ctx context.Context, cfg stt.Config) (stt.Stream, error) {
	b.mu.Lock()
	b.open++
	b.mu.Unlock()
	return &blockingStream{ctx: ctx, done: func() { b.mu.Lock(); b.open--; b.mu.Unlock() }}, nil
}

func (b *blocking) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.open
}

type blockingStream struct {
	ctx  context.Context
	done func()
	once sync.Once
}

func (s *blockingStream) Send([]byte) error { return s.ctx.Err() }
func (s *blockingStream) CloseSend() error  { return nil }
func (s *blockingStream) Recv() (stt.Result, error) {
	<-s.ctx.Done()
	s.once.Do(s.done)
	return stt.Result{}, s.ctx.Err()
}

// Once the agent is out of the room (removed when the consent was withdrawn, or the room deleted), every
// recognition ends at once: nothing more is sent or published.
func TestTranscribe_EndsEveryRecognitionWhenTheAgentIsOutOfTheRoom(t *testing.T) {
	f := newLiveFixture(t)
	r := &blocking{}
	f.svc = f.service(r)
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient) <- []byte{1, 2}
	f.calls.publish(videosession.RolePractitioner) <- []byte{3, 4}
	waitFor(t, "two recognitions open", func() bool { return r.count() == 2 })
	f.calls.remove()
	waitFor(t, "every recognition ended", func() bool { return r.count() == 0 })
}

// Stop ends every call: the agent leaves its rooms, and Stop returns once their recognitions ended.
func TestStop_LeavesEveryRoom(t *testing.T) {
	f := newLiveFixture(t)
	r := &blocking{}
	f.svc = f.service(r)
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient) <- []byte{1, 2}
	waitFor(t, "a recognition open", func() bool { return r.count() == 1 })
	f.svc.Stop()
	if _, leaves := f.calls.counts(); leaves < 1 || r.count() != 0 {
		t.Errorf("left %d times with %d recognitions open; want left and none open", leaves, r.count())
	}
}

// failing is a recognizer whose first stream fails after its first result; the next ones play the script.
type failing struct {
	script []stt.Result
	mu     sync.Mutex
	opened int
}

func (r *failing) Stream(ctx context.Context, cfg stt.Config) (stt.Stream, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.opened++
	s, err := (&stttest.Recognizer{Results: r.script}).Stream(ctx, cfg)
	if r.opened == 1 {
		return &failAfterOne{Stream: s}, err
	}
	return s, err
}

type failAfterOne struct {
	stt.Stream
	n int
}

func (s *failAfterOne) Recv() (stt.Result, error) {
	if s.n++; s.n > 1 {
		return stt.Result{}, errors.New("engine dropped the connection")
	}
	return s.Stream.Recv()
}

// A recognition that fails while the person is still speaking is opened again; the audio goes on.
func TestTranscribe_OpensARecognitionAgainAfterItFails(t *testing.T) {
	f := newLiveFixture(t)
	r := &failing{script: []stt.Result{{Text: "placeholder", Final: true}}}
	f.svc = f.service(r)
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	frames := f.calls.publish(videosession.RolePatient)
	frames <- []byte{1, 2}
	f.next(t)
	f.next(t) // from the second recognition
	if e := f.logs.String(); !strings.Contains(e, "speech-to-text failed") || !strings.Contains(e, sessionID) {
		t.Errorf("logs %s; want the failure with the session id", e)
	}
}

// Results without words are not segments; a result whose language is not a BCP 47 tag is in the speaker's.
func TestTranscribe_SkipsEmptyResultsAndKeepsTheSpeakersLanguage(t *testing.T) {
	f := newLiveFixture(t,
		stt.Result{Text: " \n"},
		stt.Result{Text: "placeholder", Lang: "Turkish (Türkiye)", Final: true},
	)
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient)
	if _, e := f.next(t); e.Original != "placeholder" || e.Lang != "tr-TR" {
		t.Errorf("event %+v; want the words in tr-TR", e)
	}
}

// Nothing of what was said reaches a log: a failing stream is logged with the session, the speaker's role and
// the error alone.
func TestTranscribe_LogsNoTranscript(t *testing.T) {
	const words = "placeholder secret words"
	f := newLiveFixture(t, stt.Result{Text: words, Final: true})
	f.publisher.err = errors.New("redis down")
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient)
	waitFor(t, "the publish failure logged", func() bool { return strings.Contains(f.logs.String(), "publish") })
	if strings.Contains(f.logs.String(), words) || strings.Contains(f.logs.String(), accountID) {
		t.Errorf("logs hold the words or the account: %s", f.logs.String())
	}
}

// syncBuffer is a log sink the calls' goroutines write while the test reads it.
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
