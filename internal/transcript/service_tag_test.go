package transcript_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/stt"
)

// A final line carries the explained terms of its original, in the speaker's language, and of its translation, in
// the listener's (SAL-13): an overlay for the practitioner's tooltip, the texts unchanged. Partial lines carry none.
func TestTranscribe_TagsTheTermsOfAFinalLineAndItsTranslation(t *testing.T) {
	f := newLiveFixture(t, script()...)
	spans := []terminology.Span{{TermID: "t-1", Start: 0, End: 11}}
	f.glossary.spans = spans
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient)
	var final transcript.Event
	for final.SegmentID == "" {
		switch _, e := f.next(t); {
		case e.IsFinal:
			final = e
		case e.Terms != nil:
			t.Errorf("partial %+v is tagged", e)
		}
	}
	want := &transcript.Terms{Original: spans, Translation: spans}
	if final.Original != "Placeholder words." || !reflect.DeepEqual(final.Terms, want) {
		t.Errorf("final %+v, terms %+v; want %+v", final, final.Terms, want)
	}
	wantTexts := [][]terminology.Text{{{Language: "tr", Body: "Placeholder words."}, {Language: "bs", Body: "[bs] Placeholder words."}}}
	if got := f.glossary.tagCalls(); !reflect.DeepEqual(got, wantTexts) {
		t.Errorf("tagged %+v, want %+v", got, wantTexts)
	}
}

// A line in the listener's language is tagged in it alone; when tagging fails, the line is published without terms
// and the failure logged without the line.
func TestTranscribe_TagsAnUntranslatedLineAndPublishesItWhenTaggingFails(t *testing.T) {
	f := newLiveFixture(t, stt.Result{Text: "placeholder-marker", Final: true})
	f.languages.patient, f.languages.clinic = "bs", "bs"
	f.glossary.spans = []terminology.Span{{TermID: "t-1", Start: 0, End: 5}}
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient)
	if e := f.finalOf(t); e.Terms == nil || e.Terms.Original == nil || e.Terms.Translation != nil {
		t.Errorf("an untranslated line's terms %+v, want the original's only", e.Terms)
	}
	f = newLiveFixture(t, stt.Result{Text: "placeholder-marker", Final: true})
	f.glossary.tagErr = errors.New("the database is down")
	if err := f.svc.Transcribe(t.Context(), sessionID, clinicID); err != nil {
		t.Fatal(err)
	}
	f.calls.publish(videosession.RolePatient)
	if e := f.finalOf(t); e.Terms != nil || e.Original != "placeholder-marker" {
		t.Errorf("with tagging down: %+v, want the line without terms", e)
	}
	if logs := f.logs.String(); !strings.Contains(logs, "term tagging failed") || strings.Contains(logs, "marker") {
		t.Errorf("logs %q: want the failure without the line", logs)
	}
}

// The stored transcript is tagged as it is read (SAL-13): each line's original in its speaker's language, its
// translation in the other speaker's, the app language read from the stored tag and, for Serbian (one speech locale
// for both scripts), from the text's letters. Without a line of the other speaker the translation's language is not
// known, and it is not tagged.
func TestExport_TagsTheStoredLinesInTheirLanguages(t *testing.T) {
	f := newFixture()
	spans := []terminology.Span{{TermID: "t-1", Start: 0, End: 4}}
	f.terms.spans = spans
	first := f.storeAt(t, segmentID, 0) // the patient, bs, translated into de
	if got, err := f.svc.Export(t.Context(), pract, encounterID); err != nil || len(got) != 1 || !reflect.DeepEqual(got[0].Terms, &transcript.Terms{Original: spans}) {
		t.Fatalf("Export() alone = %+v, %v; want the original tagged only", got, err)
	}
	second := valid()
	second.SegmentID, second.Speaker, second.Lang = segmentID2, transcript.SpeakerPractitioner, "sr-RS"
	second.Original, second.Translation = "плацехолдер", "[bs] placeholder"
	second.TsStart, second.TsEnd = first.TsStart.Add(time.Minute), first.TsEnd.Add(time.Minute)
	f.store(t, second)
	f.terms.tagged = nil

	got, err := f.svc.Export(t.Context(), pract, encounterID)
	if err != nil || len(got) != 2 || !reflect.DeepEqual(got[1].Terms, &transcript.Terms{Original: spans, Translation: spans}) {
		t.Fatalf("Export() = %+v, %v; want both lines tagged", got, err)
	}
	want := [][]terminology.Text{{
		{Language: "bs", Body: "placeholder original"}, {Language: "sr-Latn", Body: "[de] placeholder original"},
		{Language: "sr-Cyrl", Body: "плацехолдер"}, {Language: "bs", Body: "[bs] placeholder"},
	}}
	if calls := f.terms.tagCalls(); !reflect.DeepEqual(calls, want) {
		t.Errorf("tagged %+v,\nwant %+v", calls, want)
	}
}

// Tagging is an overlay: when it fails, the transcript is still read, without terms.
func TestExport_GivesTheTranscriptWithoutTermsWhenTaggingFails(t *testing.T) {
	f := newFixture()
	first := f.storeAt(t, segmentID, 0)
	f.terms.tagErr = errors.New("the database is down")
	got, err := f.svc.Export(t.Context(), pract, encounterID)
	if err != nil || len(got) != 1 || got[0] != first {
		t.Errorf("Export() = %+v, %v; want the line without terms", got, err)
	}
}

// store saves e as the agent's stored segment of clinic A.
func (f *fixture) store(t *testing.T, e transcript.Event) {
	t.Helper()
	if err := (fakeTransactor{f: f}).WithTx(t.Context(), func(ctx context.Context) error {
		f.scope = clinicA
		_, err := f.repo.Insert(ctx, e)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}
