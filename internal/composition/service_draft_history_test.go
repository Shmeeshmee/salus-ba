package composition_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/transcript"
)

// ADR 0014 "No AI" (SAL-42): the drafting reads only the consultation's own transcript. No other visit of the
// patient's, shared with the clinic or not, reaches the model's input: not its transcript, not its documents.
func TestGenerate_NeverSendsAnotherVisitsContentToTheModel(t *testing.T) {
	t.Parallel()
	const (
		sharedVisit   = "ec000000-0000-4000-8000-0000000000b1"
		unsharedVisit = "ec000000-0000-4000-8000-0000000000b2"
		otherWords    = "Placeholder words of another visit"
	)
	f := newFixture(t)
	tpl := f.template(t)
	f.drafter.Fields = filled()
	f.transcripts.others = map[string][]transcript.Event{}
	for _, id := range []string{sharedVisit, unsharedVisit} {
		f.transcripts.others[id] = []transcript.Event{{
			EncounterID: id, SegmentID: seg1, Speaker: transcript.SpeakerPatient, Lang: "bs", Original: otherWords, IsFinal: true, TsStart: now.Add(-time.Hour),
		}}
	}
	if _, err := generate(t, f, pract, tpl.ID, ""); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.transcripts.asked, []string{encA}) {
		t.Errorf("transcripts asked for %v, want only the consultation's %s", f.transcripts.asked, encA)
	}
	calls := f.drafter.Calls()
	if len(calls) != 1 {
		t.Fatalf("drafted %d times, want once", len(calls))
	}
	if in := fmt.Sprintf("%+v", calls[0]); strings.Contains(in, otherWords) || strings.Contains(in, sharedVisit) || strings.Contains(in, unsharedVisit) {
		t.Errorf("the model's input holds another visit's content: %s", in)
	}
}
