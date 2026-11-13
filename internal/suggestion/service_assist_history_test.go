package suggestion_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/transcript"
)

// ADR 0014 "No AI" (SAL-42): the suggestions read only the consultation's own stream. What is said in another visit
// of the patient's, shared with the clinic or not, never reaches the model's input.
func TestStep_NeverSendsAnotherVisitsContentToTheModel(t *testing.T) {
	const (
		sharedVisit   = "ec000000-0000-4000-8000-0000000000b1"
		unsharedVisit = "ec000000-0000-4000-8000-0000000000b2"
		otherWords    = "Placeholder words of another visit"
	)
	x := newAssist(t)
	for _, id := range []string{sharedVisit, unsharedVisit} {
		for n := 1; n <= suggestion.RunEveryLines; n++ {
			x.live.add(id, encode(t, transcript.MessageType, transcript.MessageVersion, transcript.Event{
				EncounterID: id, SegmentID: seg(100 + n), Speaker: transcript.SpeakerPatient, Lang: "bs", Original: otherWords, IsFinal: true, TsStart: t0, TsEnd: t0,
			}))
		}
	}
	for n := 1; n <= suggestion.RunEveryLines; n++ {
		x.say(t, n, true)
	}
	x.step(t)
	calls := x.suggester.Calls()
	if len(calls) != 1 {
		t.Fatalf("suggested %d times, want once", len(calls))
	}
	if in := fmt.Sprintf("%+v", calls[0]); strings.Contains(in, otherWords) || strings.Contains(in, sharedVisit) || strings.Contains(in, unsharedVisit) {
		t.Errorf("the model's input holds another visit's content: %s", in)
	}
}
