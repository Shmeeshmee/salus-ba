package transcript_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/llm"
	"github.com/Shmeeshmee/salus/pkg/stt"
)

// fakeGlossary gives every line the same entries, recording what it was asked.
type fakeGlossary struct {
	err     error
	entries []terminology.GlossaryEntry
	asked   []glossaryCall
	mu      sync.Mutex
}

type glossaryCall struct{ text, from, to string }

func (g *fakeGlossary) Glossary(_ context.Context, text, from, to string) ([]terminology.GlossaryEntry, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.asked = append(g.asked, glossaryCall{text: text, from: from, to: to})
	return g.entries, g.err
}

func (g *fakeGlossary) calls() []glossaryCall {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]glossaryCall(nil), g.asked...)
}

// finalOf waits for the final event of the stream, skipping partial ones.
func (f *liveFixture) finalOf(t *testing.T) transcript.Event {
	t.Helper()
	for {
		if _, e := f.next(t); e.IsFinal {
			return e
		}
	}
}

// A patient's final line is published once, translated into the practitioner's language (the clinic's) with the
// terms the terminology found in it; its partial lines are not translated, and the translator never sees them.
func TestTranscribe_TranslatesAPatientsFinalLineIntoTheClinicsLanguage(t *testing.T) {
	f := newLiveFixture(t, script()...)
	f.glossary.entries = []terminology.GlossaryEntry{{Source: "baş ağrısı", Target: "glavobolja"}}
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient)
	var final transcript.Event
	for range 4 {
		_, e := f.next(t)
		switch {
		case e.IsFinal:
			final = e
		case e.Translation != "" || e.TranslationFailed:
			t.Errorf("partial %+v is translated", e)
		}
	}
	if final.Original != "Placeholder words." || final.Translation != "[bs] Placeholder words." || final.TranslationFailed {
		t.Errorf("final %+v; want its translation into bs", final)
	}
	want := []llm.TranslateRequest{{Text: "Placeholder words.", From: "tr", To: "bs", Glossary: []llm.Term{{Source: "baş ağrısı", Target: "glavobolja"}}}}
	if got := f.translator.Calls(); !reflect.DeepEqual(got, want) {
		t.Errorf("translator asked %+v,\nwant %+v", got, want)
	}
	if got := f.glossary.calls(); len(got) != 1 || got[0] != (glossaryCall{text: "Placeholder words.", from: "tr", to: "bs"}) {
		t.Errorf("glossary asked %+v; want the final line, tr to bs", got)
	}
}

// Each side reads the other in their own language: the practitioner's final line goes into the patient's
// language; a line already in the listener's language is not translated, and not flagged.
func TestTranscribe_TranslatesIntoTheOtherParticipantsLanguageOnly(t *testing.T) {
	for _, tc := range []struct {
		role            videosession.Role
		patient, clinic string
		want            string
		to              string
	}{
		{role: videosession.RolePractitioner, patient: "ar", clinic: "bs", want: "[ar] placeholder", to: "ar"},
		{role: videosession.RolePatient, patient: "de", clinic: "hr", want: "[hr] placeholder", to: "hr"},
		{role: videosession.RolePatient, patient: "", clinic: "bs", want: ""},
		{role: videosession.RolePractitioner, patient: "sr-Latn", clinic: "sr-Latn", want: ""},
	} {
		f := newLiveFixture(t, stt.Result{Text: "placeholder", Final: true})
		f.languages.patient, f.languages.clinic = tc.patient, tc.clinic
		if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
			t.Fatal(err)
		}
		f.calls.publish(tc.role)
		e := f.finalOf(t)
		if e.Translation != tc.want || e.TranslationFailed {
			t.Errorf("%s, patient %q, clinic %q: translation %q, failed %v; want %q", tc.role, tc.patient, tc.clinic, e.Translation, e.TranslationFailed, tc.want)
		}
		calls := f.translator.Calls()
		if (tc.want == "") != (len(calls) == 0) || (len(calls) == 1 && calls[0].To != tc.to) {
			t.Errorf("%s, patient %q, clinic %q: translator asked %+v", tc.role, tc.patient, tc.clinic, calls)
		}
	}
}

// A failed translation is tried once more; when that fails too, or the budget is spent, the line is published
// with its original alone and flagged, never dropped.
func TestTranscribe_RetriesOnceThenPublishesTheOriginalFlagged(t *testing.T) {
	for _, tc := range []struct {
		name       string
		want       string
		failures   int
		delay      time.Duration
		maxAttempt int
		failed     bool
	}{
		{name: "one failure", failures: 1, want: "[bs] placeholder", maxAttempt: 2},
		{name: "two failures", failures: 2, failed: true, maxAttempt: 2},
		{name: "slower than the budget", delay: time.Hour, failed: true, maxAttempt: 1},
	} {
		f := newLiveFixture(t, stt.Result{Text: "placeholder", Final: true})
		f.translator.Failures, f.translator.Delay = tc.failures, tc.delay
		f.svc = f.service(f.stt, 50*time.Millisecond)
		t.Cleanup(f.svc.Stop)
		if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
			t.Fatal(err)
		}
		f.calls.publish(videosession.RolePatient)
		e := f.finalOf(t)
		if e.Original != "placeholder" || e.Translation != tc.want || e.TranslationFailed != tc.failed {
			t.Errorf("%s: event %+v; want translation %q, failed %v", tc.name, e, tc.want, tc.failed)
		}
		if n := len(f.translator.Calls()); n < 1 || n > tc.maxAttempt {
			t.Errorf("%s: %d attempts, want 1 to %d", tc.name, n, tc.maxAttempt)
		}
		if tc.failed && (!strings.Contains(f.logs.String(), "translation failed") || strings.Contains(f.logs.String(), "placeholder")) {
			t.Errorf("%s: logs %s; want the failure without the line", tc.name, f.logs.String())
		}
	}
}

// Without its glossary (the terminology failed) the line is still translated.
func TestTranscribe_TranslatesWithoutTheGlossaryWhenItFails(t *testing.T) {
	f := newLiveFixture(t, stt.Result{Text: "placeholder", Final: true})
	f.glossary.err = errors.New("the database is down")
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient)
	if e := f.finalOf(t); e.Translation != "[bs] placeholder" || e.TranslationFailed {
		t.Errorf("event %+v; want it translated", e)
	}
	if calls := f.translator.Calls(); len(calls) != 1 || calls[0].Glossary != nil {
		t.Errorf("translator asked %+v; want one call without a glossary", calls)
	}
}

// A final line's translation does not hold up what is said next: the next segment's partials are published
// while it waits.
func TestTranscribe_PublishesTheNextPartialsWhileAFinalLineIsTranslated(t *testing.T) {
	f := newLiveFixture(t,
		stt.Result{Text: "placeholder", Final: true},
		stt.Result{Text: "Next", Start: time.Second, End: 1200 * time.Millisecond},
	)
	f.translator.Delay = 300 * time.Millisecond
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient)
	if _, e := f.next(t); e.IsFinal || e.Original != "Next" {
		t.Errorf("first event %+v; want the next partial before the translated final", e)
	}
	if e := f.finalOf(t); e.Translation != "[bs] placeholder" {
		t.Errorf("final %+v; want it translated", e)
	}
}

// Once the agent is out of the room, a translation still pending is dropped with the call: nothing said is
// processed or published after a removal.
func TestTranscribe_PublishesNoPendingTranslationAfterTheRemoval(t *testing.T) {
	f := newLiveFixture(t, stt.Result{Text: "placeholder", Final: true})
	f.translator.Delay = time.Hour
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient)
	waitFor(t, "the translation asked", func() bool { return len(f.translator.Calls()) == 1 })
	f.calls.remove()
	f.svc.Stop()
	select {
	case p := <-f.publisher.sent:
		t.Errorf("published %s after the removal", p.envelope)
	default:
	}
}
