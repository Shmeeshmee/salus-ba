package transcript_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/transcript"
)

const (
	encounterID = "0b6c3a52-6a0c-4c52-9e1b-6f1c2a3b4c5d"
	segmentID   = "7d1e2f30-1a2b-4c3d-8e4f-5a6b7c8d9e0f"
)

var start = time.Date(2026, 11, 8, 9, 30, 0, 0, time.UTC)

func valid() transcript.Event {
	return transcript.Event{
		EncounterID: encounterID,
		SegmentID:   segmentID,
		Speaker:     transcript.SpeakerPatient,
		Lang:        "bs",
		Original:    "placeholder original",
		Translation: "[de] placeholder original",
		IsFinal:     true,
		TsStart:     start,
		TsEnd:       start.Add(1500 * time.Millisecond),
	}
}

func TestEvent_EncodesToTheContractsJSON(t *testing.T) {
	t.Parallel()
	got, err := json.Marshal(valid())
	if err != nil {
		t.Fatal(err)
	}
	// Key order follows the struct, laid out for govet's fieldalignment; JSON gives it no meaning.
	want := `{"ts_start":"2026-11-08T09:30:00Z","ts_end":"2026-11-08T09:30:01.5Z",` +
		`"encounter_id":"` + encounterID + `","segment_id":"` + segmentID + `","speaker":"patient","lang":"bs",` +
		`"original":"placeholder original","translation":"[de] placeholder original","is_final":true}`
	if string(got) != want {
		t.Errorf("json =\n%s\nwant\n%s", got, want)
	}
}

func TestEvent_ValidateAcceptsAValidEvent(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]func(*transcript.Event){
		"final":                 func(*transcript.Event) {},
		"partial":               func(e *transcript.Event) { e.IsFinal = false },
		"the practitioner":      func(e *transcript.Event) { e.Speaker = transcript.SpeakerPractitioner },
		"not translated yet":    func(e *transcript.Event) { e.Translation = "" },
		"a region subtag":       func(e *transcript.Event) { e.Lang = "sr-Latn-BA" },
		"a three-letter lang":   func(e *transcript.Event) { e.Lang = "gsw" },
		"an instant":            func(e *transcript.Event) { e.TsEnd = e.TsStart },
		"an upper-case encoded": func(e *transcript.Event) { e.SegmentID = "7D1E2F30-1A2B-4C3D-8E4F-5A6B7C8D9E0F" },
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
	for name, change := range map[string]func(*transcript.Event){
		"no encounter":          func(e *transcript.Event) { e.EncounterID = "" },
		"an encounter no UUID":  func(e *transcript.Event) { e.EncounterID = "Encounter/1" },
		"no segment":            func(e *transcript.Event) { e.SegmentID = "" },
		"an unknown speaker":    func(e *transcript.Event) { e.Speaker = "doctor" },
		"no speaker":            func(e *transcript.Event) { e.Speaker = "" },
		"no lang":               func(e *transcript.Event) { e.Lang = "" },
		"a lang with a space":   func(e *transcript.Event) { e.Lang = "b s" },
		"an upper-case primary": func(e *transcript.Event) { e.Lang = "BS" },
		"a one-letter primary":  func(e *transcript.Event) { e.Lang = "b" },
		"an empty subtag":       func(e *transcript.Event) { e.Lang = "sr--BA" },
		"a trailing hyphen":     func(e *transcript.Event) { e.Lang = "bs-" },
		"a symbol in a subtag":  func(e *transcript.Event) { e.Lang = "sr-B@" },
		"a long subtag":         func(e *transcript.Event) { e.Lang = "sr-abcdefghi" },
		"no original":           func(e *transcript.Event) { e.Original = "" },
		"a blank original":      func(e *transcript.Event) { e.Original = " \n\t" },
		"no start":              func(e *transcript.Event) { e.TsStart = time.Time{} },
		"an end before a start": func(e *transcript.Event) { e.TsEnd = e.TsStart.Add(-time.Millisecond) },
	} {
		e := valid()
		change(&e)
		if err := e.Validate(); !errors.Is(err, transcript.ErrInvalid) {
			t.Errorf("%s: Validate() = %v, want ErrInvalid", name, err)
		}
	}
}

func TestEvent_ValidateNamesTheFieldButNeverItsValue(t *testing.T) {
	t.Parallel()
	e := valid()
	e.Lang = "secret value"
	err := e.Validate()
	if err == nil || err.Error() != "transcript: invalid event: lang" {
		t.Errorf("Validate() = %v, want %q", err, "transcript: invalid event: lang")
	}
}

func TestStreamKey_IsTheEncountersEventStream(t *testing.T) {
	t.Parallel()
	if got, want := transcript.StreamKey(encounterID), "consult:"+encounterID+":events"; got != want {
		t.Errorf("StreamKey() = %q, want %q", got, want)
	}
}

func TestMessage_IsTheTranscriptTypeVersion1(t *testing.T) {
	t.Parallel()
	if transcript.MessageType != "transcript" || transcript.MessageVersion != 1 {
		t.Errorf("message = %q v%d, want transcript v1", transcript.MessageType, transcript.MessageVersion)
	}
}
