package suggestion_test

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/suggestion"
)

const (
	suggestionID = "3f2a1b0c-9d8e-4f7a-8b6c-5d4e3f2a1b0c"
	encounterID  = "0b6c3a52-6a0c-4c52-9e1b-6f1c2a3b4c5d"
	segmentID    = "7d1e2f30-1a2b-4c3d-8e4f-5a6b7c8d9e0f"
)

func valid() suggestion.Event {
	return suggestion.Event{
		ID:               suggestionID,
		EncounterID:      encounterID,
		Type:             suggestion.TypeQuestion,
		Text:             "Placeholder question to ask",
		Confidence:       0.75,
		SourceSegmentIDs: []string{segmentID},
		Status:           suggestion.StatusShown,
		Origin:           suggestion.OriginAI,
	}
}

func TestEvent_EncodesToTheContractsJSON(t *testing.T) {
	t.Parallel()
	got, err := json.Marshal(valid())
	if err != nil {
		t.Fatal(err)
	}
	// Key order follows the struct, laid out for govet's fieldalignment; JSON gives it no meaning.
	want := `{"id":"` + suggestionID + `","encounter_id":"` + encounterID + `","type":"question",` +
		`"text":"Placeholder question to ask","status":"shown","origin":"ai","source_segment_ids":["` + segmentID + `"],"confidence":0.75}`
	if string(got) != want {
		t.Errorf("json =\n%s\nwant\n%s", got, want)
	}
}

func TestEvent_ValidateAcceptsAValidEvent(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]func(*suggestion.Event){
		"a question":        func(*suggestion.Event) {},
		"a diagnosis":       func(e *suggestion.Event) { e.Type = suggestion.TypeDiagnosis },
		"a red flag":        func(e *suggestion.Event) { e.Type = suggestion.TypeRedFlag },
		"a term":            func(e *suggestion.Event) { e.Type = suggestion.TypeTerm },
		"dismissed":         func(e *suggestion.Event) { e.Status = suggestion.StatusDismissed },
		"pinned":            func(e *suggestion.Event) { e.Status = suggestion.StatusPinned },
		"no confidence":     func(e *suggestion.Event) { e.Confidence = 0 },
		"full confidence":   func(e *suggestion.Event) { e.Confidence = 1 },
		"from two segments": func(e *suggestion.Event) { e.SourceSegmentIDs = append(e.SourceSegmentIDs, suggestionID) },
		"the longest text":  func(e *suggestion.Event) { e.Text = strings.Repeat("ž", suggestion.MaxTextLength) },
	} {
		e := valid()
		change(&e)
		if err := e.Validate(); err != nil {
			t.Errorf("%s: Validate() = %v, want nil", name, err)
		}
	}
}

func TestEvent_ValidateRefusesAnInvalidEvent(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]func(*suggestion.Event){
		"no id":                 func(e *suggestion.Event) { e.ID = "" },
		"no encounter":          func(e *suggestion.Event) { e.EncounterID = "nope" },
		"an unknown type":       func(e *suggestion.Event) { e.Type = "treatment" },
		"no text":               func(e *suggestion.Event) { e.Text = "  " },
		"a negative confidence": func(e *suggestion.Event) { e.Confidence = -0.01 },
		"over full confidence":  func(e *suggestion.Event) { e.Confidence = 1.01 },
		"a NaN confidence":      func(e *suggestion.Event) { e.Confidence = float32(math.NaN()) },
		"no source":             func(e *suggestion.Event) { e.SourceSegmentIDs = nil },
		"a source no UUID":      func(e *suggestion.Event) { e.SourceSegmentIDs = []string{segmentID, "s2"} },
		"an unknown status":     func(e *suggestion.Event) { e.Status = "accepted" },
		"a text too long":       func(e *suggestion.Event) { e.Text = strings.Repeat("ž", suggestion.MaxTextLength+1) },
		"no origin":             func(e *suggestion.Event) { e.Origin = "" },
		"a practitioner origin": func(e *suggestion.Event) { e.Origin = "practitioner" },
	} {
		e := valid()
		change(&e)
		if err := e.Validate(); !errors.Is(err, suggestion.ErrInvalid) {
			t.Errorf("%s: Validate() = %v, want ErrInvalid", name, err)
		}
	}
}

func TestEvent_ValidateNamesTheFieldButNeverItsValue(t *testing.T) {
	t.Parallel()
	e := valid()
	e.Type = "secret value"
	if err := e.Validate(); err == nil || err.Error() != "suggestion: invalid event: type" {
		t.Errorf("Validate() = %v, want %q", err, "suggestion: invalid event: type")
	}
}

func TestMessage_IsTheSuggestionTypeVersion1(t *testing.T) {
	t.Parallel()
	if suggestion.MessageType != "suggestion" || suggestion.MessageVersion != 1 {
		t.Errorf("message = %q v%d, want suggestion v1", suggestion.MessageType, suggestion.MessageVersion)
	}
}
