package suggestion_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/llm/llmtest"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The synthetic words of the consultation, which no log line may hold.
const (
	spokenWords = "Placeholder words said"
	askAbout    = "Placeholder question about the cough"
)

// t0 is when the session starts.
var t0 = time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)

// stepClock is a clock the test moves on.
type stepClock struct {
	t  time.Time
	mu sync.Mutex
}

func (c *stepClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *stepClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// fakeLive is the encounters' streams: entries by encounter, ids "<n>-0" in order, and the leases.
type fakeLive struct {
	err     error
	entries map[string][]transcript.Entry
	owners  map[string]string // the lease holder by encounter
	mu      sync.Mutex
}

func (l *fakeLive) add(encounterID string, envelope []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.entries == nil {
		l.entries = map[string][]transcript.Entry{}
	}
	n := len(l.entries[encounterID]) + 1
	l.entries[encounterID] = append(l.entries[encounterID], transcript.Entry{ID: fmt.Sprintf("%d-0", n), Envelope: envelope})
}

func (l *fakeLive) Encounters(context.Context) ([]string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.err != nil {
		return nil, l.err
	}
	return slices.Sorted(func(yield func(string) bool) {
		for id := range l.entries {
			if !yield(id) {
				return
			}
		}
	}), nil
}

func (l *fakeLive) Claim(_ context.Context, encounterID, owner string, ttl time.Duration) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if ttl <= 0 {
		return false, fmt.Errorf("a lease of %v", ttl)
	}
	if l.owners == nil {
		l.owners = map[string]string{}
	}
	if o, ok := l.owners[encounterID]; ok && o != owner {
		return false, nil
	}
	l.owners[encounterID] = owner
	return true, nil
}

// Entries gives the entries after the cursor ("" for the first), at most count.
func (l *fakeLive) Entries(_ context.Context, encounterID, after string, count int) ([]transcript.Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.err != nil {
		return nil, l.err
	}
	all := l.entries[encounterID]
	i := 0
	if after != "" {
		i = slices.IndexFunc(all, func(e transcript.Entry) bool { return e.ID == after }) + 1
	}
	return slices.Clone(all[i:min(len(all), i+count)]), nil
}

// fakeClinics gives clinic A's default language.
type fakeClinics struct{}

func (fakeClinics) DefaultLanguage(_ context.Context, org string) (string, error) {
	if org != clinicA {
		return "", fmt.Errorf("clinic %s", org)
	}
	return "hr", nil
}

// syncBuffer is a log the worker writes and the test reads.
type syncBuffer struct {
	b  bytes.Buffer
	mu sync.Mutex
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

type assist struct {
	*fixture
	live      *fakeLive
	suggester *llmtest.Suggester
	clock     *stepClock
	log       *syncBuffer
	a         *suggestion.Assistant
	session   *suggestion.Session
}

func newAssist(t *testing.T) *assist {
	t.Helper()
	x := &assist{fixture: newFixture(), live: &fakeLive{}, suggester: &llmtest.Suggester{}, clock: &stepClock{t: t0}, log: &syncBuffer{}}
	x.a = suggestion.NewAssistant(x.svc, suggestion.AssistPorts{
		Suggester: x.suggester, Live: x.live, Clinics: fakeClinics{}, Clock: x.clock,
		Log: slog.New(slog.NewJSONHandler(x.log, nil)), Tick: time.Millisecond,
	})
	x.session = x.a.NewSession(encounterID)
	t.Cleanup(func() {
		if l := x.log.String(); strings.Contains(l, spokenWords) || strings.Contains(l, askAbout) {
			t.Errorf("the log holds what was said or suggested:\n%s", l)
		}
	})
	return x
}

// seg is the id of the n-th segment.
func seg(n int) string { return fmt.Sprintf("7d1e2f30-1a2b-4c3d-8e4f-%012d", n) }

// say puts the n-th segment on the stream: a partial, then (when final) its final.
func (x *assist) say(t *testing.T, n int, final bool) {
	t.Helper()
	e := transcript.Event{
		EncounterID: encounterID, SegmentID: seg(n), Speaker: transcript.SpeakerPatient, Lang: "bs",
		Original: fmt.Sprintf("%s %d", spokenWords, n), TsStart: t0, TsEnd: t0.Add(time.Second),
	}
	if n%2 == 0 {
		e.Speaker = transcript.SpeakerPractitioner
	}
	x.live.add(encounterID, encode(t, transcript.MessageType, transcript.MessageVersion, e))
	if final {
		e.IsFinal = true
		x.live.add(encounterID, encode(t, transcript.MessageType, transcript.MessageVersion, e))
	}
}

func encode(t *testing.T, typ string, version int, v any) []byte {
	t.Helper()
	b, err := wire.Encode(typ, version, v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (x *assist) step(t *testing.T) {
	t.Helper()
	if err := x.a.Step(t.Context(), x.session); err != nil {
		t.Fatalf("Step() error = %v", err)
	}
}

// shown is what was published, in order.
func (x *assist) shown(t *testing.T) []suggestion.Event {
	t.Helper()
	var out []suggestion.Event
	for _, p := range x.publisher.published {
		out = append(out, decodePublished(t, p))
	}
	return out
}

func question(text string, confidence float32, sources ...string) llm.Suggestion {
	return llm.Suggestion{Kind: "question", Text: text, Confidence: confidence, SourceIDs: sources}
}

// assistEvent is the worker's audit event of interaction on count suggestions of encounter A.
func assistEvent(interaction string, count int64) auditevent.AuditEvent {
	e := systemEvent(interaction, auditevent.OutcomeSuccess, count)
	e.AgentName = "suggestions_worker"
	if interaction == auditevent.CodeRead {
		e.Action = auditevent.ActionRead
	}
	return e
}

// Every RunEveryLines final lines the window goes to the suggester: the recent final lines in order (a partial
// is not a line), in the clinic's default language (the practitioner's). Each suggestion it gives is stored as
// shown, then published on the encounter's stream, labelled as AI. The read of what was shown before and each
// write are audited as the worker's.
func TestStep_SuggestsEveryNFinalLines(t *testing.T) {
	x := newAssist(t)
	x.suggester.Suggestions = []llm.Suggestion{question(askAbout, 0.6, seg(2))}
	for n := 1; n < suggestion.RunEveryLines; n++ {
		x.say(t, n, true)
	}
	x.say(t, 99, false)
	x.step(t)
	if calls := x.suggester.Calls(); len(calls) != 0 {
		t.Fatalf("suggested after %d lines: %+v", suggestion.RunEveryLines-1, calls)
	}
	x.say(t, suggestion.RunEveryLines, true)
	x.step(t)

	calls := x.suggester.Calls()
	if len(calls) != 1 {
		t.Fatalf("suggested %d times, want once", len(calls))
	}
	var want []llm.Utterance
	for n := 1; n <= suggestion.RunEveryLines; n++ {
		speaker := "patient"
		if n%2 == 0 {
			speaker = "practitioner"
		}
		want = append(want, llm.Utterance{ID: seg(n), Speaker: speaker, Text: fmt.Sprintf("%s %d", spokenWords, n)})
	}
	if calls[0].Lang != "hr" || !slices.Equal(calls[0].Transcript, want) || len(calls[0].Shown) != 0 {
		t.Errorf("request = %+v,\nwant hr and %+v", calls[0], want)
	}
	got := x.shown(t)
	if len(got) != 1 {
		t.Fatalf("published %d, want 1", len(got))
	}
	e := got[0]
	if !fhir.IsUUID(e.ID) || e.EncounterID != encounterID || e.Type != suggestion.TypeQuestion || e.Text != askAbout ||
		e.Confidence != 0.6 || !slices.Equal(e.SourceSegmentIDs, []string{seg(2)}) || e.Status != suggestion.StatusShown ||
		e.Origin != suggestion.OriginAI {
		t.Errorf("published %+v", e)
	}
	if stored, ok := x.repo.rows[e.ID]; !ok || stored.Text != askAbout {
		t.Errorf("stored %+v, want the suggestion before it was published", stored)
	}
	checkEvents(t, "run", x.audit, assistEvent(auditevent.CodeRead, 0), assistEvent(auditevent.CodeCreate, 1))
}

// A single new line is enough once RunEvery has passed since the session started or the last run; nothing new,
// no run at all.
func TestStep_SuggestsAboutEvery20SecondsWhenSomethingNewWasSaid(t *testing.T) {
	x := newAssist(t)
	x.say(t, 1, true)
	x.step(t)
	x.clock.advance(suggestion.RunEvery - time.Second)
	x.step(t)
	if n := len(x.suggester.Calls()); n != 0 {
		t.Fatalf("suggested %d times before %v", n, suggestion.RunEvery)
	}
	x.clock.advance(time.Second)
	x.step(t)
	if n := len(x.suggester.Calls()); n != 1 {
		t.Fatalf("suggested %d times at %v, want once", n, suggestion.RunEvery)
	}
	x.clock.advance(3 * suggestion.RunEvery)
	x.say(t, 2, false) // a partial is nothing new
	x.step(t)
	if n := len(x.suggester.Calls()); n != 1 {
		t.Errorf("suggested %d times with no new line, want once", n)
	}
}

// Runs are throttled: however fast lines come, two runs are at least MinGap apart.
func TestStep_RunsAtMostOnceEveryMinGap(t *testing.T) {
	x := newAssist(t)
	n := 0
	lines := func() {
		for range suggestion.RunEveryLines {
			n++
			x.say(t, n, true)
		}
	}
	lines()
	x.step(t)
	lines()
	x.clock.advance(suggestion.MinGap - time.Millisecond)
	x.step(t)
	if c := len(x.suggester.Calls()); c != 1 {
		t.Fatalf("suggested %d times within %v, want once", c, suggestion.MinGap)
	}
	x.clock.advance(time.Millisecond)
	x.step(t)
	if c := len(x.suggester.Calls()); c != 2 {
		t.Errorf("suggested %d times after %v, want twice", c, suggestion.MinGap)
	}
}

// The window is the latest WindowLines final lines; a final event of a line seen already (its translation
// arriving later) replaces it and is not a new line.
func TestStep_SendsTheLatestLinesOnceEach(t *testing.T) {
	x := newAssist(t)
	total := suggestion.WindowLines + 3
	for n := 1; n <= total; n++ {
		x.say(t, n, true)
		x.say(t, n, true) // the same final again
	}
	x.step(t)
	calls := x.suggester.Calls()
	if len(calls) != 1 {
		t.Fatalf("suggested %d times, want once", len(calls))
	}
	var ids []string
	for _, u := range calls[0].Transcript {
		ids = append(ids, u.ID)
	}
	var want []string
	for n := total - suggestion.WindowLines + 1; n <= total; n++ {
		want = append(want, seg(n))
	}
	if !slices.Equal(ids, want) {
		t.Errorf("window %v,\nwant %v", ids, want)
	}
}

// What the model gives is checked before anything is shown: a known kind, a confidence from 0 to 1, at least one
// source and every one in the window sent, a text present and at most MaxTextLength long. Anything else is
// dropped and counted in the log, without its text.
func TestStep_DropsWhatTheContractDoesNotAllowAndCountsIt(t *testing.T) {
	x := newAssist(t)
	for n := 1; n <= suggestion.RunEveryLines; n++ {
		x.say(t, n, true)
	}
	ok := question(askAbout, 0.5, seg(1), seg(3))
	bad := []llm.Suggestion{
		{Kind: "treatment", Text: askAbout + " 1", Confidence: 0.5, SourceIDs: []string{seg(1)}},
		{Kind: "", Text: askAbout + " 2", Confidence: 0.5, SourceIDs: []string{seg(1)}},
		question(askAbout+" 3", -0.1, seg(1)),
		question(askAbout+" 4", 1.5, seg(1)),
		question(askAbout+" 5", float32(math.NaN()), seg(1)),
		question(askAbout+" 6", 0.5),
		question(askAbout+" 7", 0.5, seg(1), seg(99)), // not in the window
		question(askAbout+" 8", 0.5, "s1"),
		question("   ", 0.5, seg(1)),
		question(strings.Repeat("a", suggestion.MaxTextLength+1), 0.5, seg(1)),
	}
	x.suggester.Suggestions = append(slices.Clone(bad), ok)
	x.step(t)
	got := x.shown(t)
	if len(got) != 1 || got[0].Text != askAbout {
		t.Fatalf("published %+v, want only the valid one", got)
	}
	if !strings.Contains(x.log.String(), fmt.Sprintf(`"dropped":%d`, len(bad))) {
		t.Errorf("the log does not count %d dropped:\n%s", len(bad), x.log.String())
	}
}

// Each run shows at most MaxPerRun new suggestions, the most confident first; the text is trimmed and the
// kinds map to the event's types.
func TestStep_ShowsAtMostMaxPerRunMostConfidentFirst(t *testing.T) {
	x := newAssist(t)
	for n := 1; n <= suggestion.RunEveryLines; n++ {
		x.say(t, n, true)
	}
	x.suggester.Suggestions = []llm.Suggestion{
		{Kind: "term", Text: "Placeholder term", Confidence: 0.2, SourceIDs: []string{seg(1)}},
		{Kind: "red_flag", Text: " Placeholder red flag ", Confidence: 0.9, SourceIDs: []string{seg(2)}},
		{Kind: "diagnosis", Text: "Placeholder condition to consider", Confidence: 0.7, SourceIDs: []string{seg(3)}},
		question(askAbout, 0.8, seg(4)),
	}
	x.step(t)
	var got []string
	for _, e := range x.shown(t) {
		got = append(got, string(e.Type)+": "+e.Text)
	}
	want := []string{"red_flag: Placeholder red flag", "question: " + askAbout, "diagnosis: Placeholder condition to consider"}
	if suggestion.MaxPerRun != len(want) || !slices.Equal(got, want) {
		t.Errorf("published %q, want %q", got, want)
	}
}

// A suggestion the practitioner dismissed never comes back, nor one saying nearly the same; nor does one
// shown already, and one run does not show the same twice. The model is told what was shown.
func TestStep_NeverRepeatsADismissedOrShownSuggestion(t *testing.T) {
	x := newAssist(t)
	n := 0
	lines := func() {
		for range suggestion.RunEveryLines {
			n++
			x.say(t, n, true)
		}
		x.clock.advance(suggestion.MinGap)
	}
	lines()
	x.suggester.Suggestions = []llm.Suggestion{question(askAbout, 0.6, seg(1)), question("Placeholder term to explain", 0.5, seg(2))}
	x.step(t)
	first := x.shown(t)
	if len(first) != 2 {
		t.Fatalf("published %d, want 2", len(first))
	}
	if _, err := x.svc.SetStatus(t.Context(), pract, encounterID, first[0].ID, suggestion.StatusDismissed); err != nil {
		t.Fatal(err)
	}
	published := len(x.publisher.published)

	lines()
	x.suggester.Suggestions = []llm.Suggestion{
		question(askAbout, 0.9, seg(6)),                                 // dismissed
		question("placeholder question, about the COUGH.", 0.9, seg(6)), // the same, written otherwise
		question("Placeholder question about the cough again", 0.9, seg(6)),
		question("Placeholder term to explain", 0.9, seg(7)), // shown
		question("Placeholder question about the fever", 0.8, seg(8)),
		question("Placeholder question about the fever", 0.7, seg(9)), // twice in one run
	}
	x.step(t)
	var got []string
	for _, e := range x.shown(t)[published:] {
		got = append(got, e.Text)
	}
	if want := []string{"Placeholder question about the fever"}; !slices.Equal(got, want) {
		t.Errorf("published %q, want %q", got, want)
	}
	calls := x.suggester.Calls()
	if shown := slices.Sorted(slices.Values(calls[len(calls)-1].Shown)); !slices.Equal(shown, []string{askAbout, "Placeholder term to explain"}) {
		t.Errorf("told the model %q were shown, want both earlier ones", shown)
	}
}

// After a revocation nothing goes to the model and nothing is shown; the refusal is audited. The lines that
// came meanwhile are not sent once it is over either: the run has passed.
func TestStep_SendsNothingToTheModelAfterARevocation(t *testing.T) {
	x := newAssist(t)
	x.consents.state.Active = false
	x.suggester.Suggestions = []llm.Suggestion{question(askAbout, 0.6, seg(1))}
	for n := 1; n <= suggestion.RunEveryLines; n++ {
		x.say(t, n, true)
	}
	x.step(t)
	if n := len(x.suggester.Calls()); n != 0 || len(x.publisher.published) != 0 {
		t.Errorf("suggested %d times and published %d after a revocation, want none", n, len(x.publisher.published))
	}
	refused := assistEvent(auditevent.CodeRead, 0)
	refused.Outcome = auditevent.OutcomeMinorFailure
	checkEvents(t, "revoked", x.audit, refused)
}

// A failed model is logged without its error's text (it may echo what it was sent) and the next run waits for
// new lines or RunEvery, as after any run; a failed stream read is returned.
func TestStep_LogsAFailedModelAndReturnsAFailedRead(t *testing.T) {
	x := newAssist(t)
	x.suggester.Err = fmt.Errorf("the model echoed %q", spokenWords)
	for n := 1; n <= suggestion.RunEveryLines; n++ {
		x.say(t, n, true)
	}
	x.step(t)
	x.clock.advance(suggestion.MinGap)
	x.step(t)
	if n := len(x.suggester.Calls()); n != 1 || !strings.Contains(x.log.String(), "suggester failed") {
		t.Errorf("suggested %d times, log:\n%s\nwant once, the failure logged", n, x.log.String())
	}
	x.live.err = errBackend
	if err := x.a.Step(t.Context(), x.session); err == nil {
		t.Error("Step() with the stream down succeeded, want its error")
	}
}

// What is not a final transcript line is skipped: other messages, undecodable entries, invalid events.
func TestStep_SkipsWhatIsNotAFinalLine(t *testing.T) {
	x := newAssist(t)
	x.live.add(encounterID, []byte("not an envelope"))
	x.live.add(encounterID, encode(t, suggestion.MessageType, suggestion.MessageVersion, valid()))
	x.live.add(encounterID, []byte(`{"type":"transcript","version":1,"payload":"a string"}`))
	x.live.add(encounterID, encode(t, transcript.MessageType, 2, transcript.Event{}))
	x.live.add(encounterID, encode(t, transcript.MessageType, transcript.MessageVersion, transcript.Event{EncounterID: encounterID, IsFinal: true}))
	x.live.add(encounterID, nil)
	for n := 1; n < suggestion.RunEveryLines; n++ {
		x.say(t, n, true)
	}
	x.step(t)
	if n := len(x.suggester.Calls()); n != 0 {
		t.Errorf("suggested after %d lines and other entries, want not yet", suggestion.RunEveryLines-1)
	}
}

// A store that fails after a suggestion is accepted ends the run with its error; nothing unstored is shown.
func TestStep_ShowsNothingItCouldNotStore(t *testing.T) {
	x := newAssist(t)
	x.suggester.Suggestions = []llm.Suggestion{question(askAbout, 0.6, seg(1))}
	for n := 1; n <= suggestion.RunEveryLines; n++ {
		x.say(t, n, true)
	}
	x.audit.err = nil
	x.repo.failInsert = errBackend
	if err := x.a.Step(t.Context(), x.session); err == nil {
		t.Error("Step() with the store down succeeded, want its error")
	}
	if len(x.publisher.published) != 0 {
		t.Errorf("published %d, want none", len(x.publisher.published))
	}
}

// Run follows every stream whose lease it holds, as long as the stream is there; a stream another member holds
// is left to it.
func TestRun_FollowsTheStreamsItHolds(t *testing.T) {
	x := newAssist(t)
	other := "0b6c3a52-6a0c-4c52-9e1b-000000000002"
	x.live.add(other, nil)
	x.live.owners = map[string]string{other: "another member"}
	x.suggester.Suggestions = []llm.Suggestion{question(askAbout, 0.6, seg(1))}
	for n := 1; n <= suggestion.RunEveryLines; n++ {
		x.say(t, n, true)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		x.a.Run(ctx, "this member")
	}()
	deadline := time.Now().Add(5 * time.Second)
	for len(x.published()) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
	if got := x.published(); len(got) != 1 || got[0].encounterID != encounterID {
		t.Errorf("published %+v, want the one suggestion of the stream held", got)
	}
	for _, c := range x.suggester.Calls() {
		for _, u := range c.Transcript {
			if !strings.HasPrefix(u.Text, spokenWords) {
				t.Errorf("sent %+v, want only the held stream's lines", u)
			}
		}
	}
}

// published is a copy of what was published so far, for a test reading while Run publishes.
func (x *assist) published() []published {
	x.publisher.mu.Lock()
	defer x.publisher.mu.Unlock()
	return slices.Clone(x.publisher.published)
}
