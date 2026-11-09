package composition_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/composition"
)

const (
	draftID     = "5e4d3c2b-1a09-4f8e-9d7c-6b5a4f3e2d1c"
	encounterID = "0b6c3a52-6a0c-4c52-9e1b-6f1c2a3b4c5d"
	templateID  = "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"
	segmentID   = "7d1e2f30-1a2b-4c3d-8e4f-5a6b7c8d9e0f"
)

func valid() composition.Event {
	return composition.Event{
		DraftID:     draftID,
		EncounterID: encounterID,
		TemplateID:  templateID,
		Status:      composition.StatusPreliminary,
		Fields: []composition.Field{
			{Key: "complaint", Value: "Placeholder complaint", SourceSegmentIDs: []string{segmentID}},
			{Key: "plan", Missing: true},
		},
	}
}

func TestEvent_EncodesToTheContractsJSON(t *testing.T) {
	t.Parallel()
	got, err := json.Marshal(valid())
	if err != nil {
		t.Fatal(err)
	}
	// Key order follows the struct, laid out for govet's fieldalignment; JSON gives it no meaning. A missing
	// field cites no segments, so it has no source_segment_ids.
	want := `{"draft_id":"` + draftID + `","encounter_id":"` + encounterID + `","template_id":"` + templateID + `",` +
		`"status":"preliminary","fields":[` +
		`{"key":"complaint","value":"Placeholder complaint","source_segment_ids":["` + segmentID + `"],"missing":false},` +
		`{"key":"plan","value":"","missing":true}]}`
	if string(got) != want {
		t.Errorf("json =\n%s\nwant\n%s", got, want)
	}
}

func TestEvent_ValidateAcceptsAValidEvent(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]func(*composition.Event){
		"preliminary":              func(*composition.Event) {},
		"final":                    func(e *composition.Event) { e.Status = composition.StatusFinal },
		"entered in error":         func(e *composition.Event) { e.Status = composition.StatusEnteredInError },
		"a value the doctor typed": func(e *composition.Event) { e.Fields[0].SourceSegmentIDs = nil },
		"every field missing":      func(e *composition.Event) { e.Fields[0] = composition.Field{Key: "complaint", Missing: true} },
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
	for name, change := range map[string]func(*composition.Event){
		"no draft":                    func(e *composition.Event) { e.DraftID = "" },
		"no encounter":                func(e *composition.Event) { e.EncounterID = "x" },
		"no template":                 func(e *composition.Event) { e.TemplateID = "" },
		"an unknown status":           func(e *composition.Event) { e.Status = "draft" },
		"no fields":                   func(e *composition.Event) { e.Fields = nil },
		"a field without a key":       func(e *composition.Event) { e.Fields[0].Key = " " },
		"a key twice":                 func(e *composition.Event) { e.Fields[1].Key = "complaint" },
		"a filled field without text": func(e *composition.Event) { e.Fields[0].Value = "" },
		"a missing field with text":   func(e *composition.Event) { e.Fields[1].Value = "Placeholder" },
		"a missing field with sources": func(e *composition.Event) {
			e.Fields[1].SourceSegmentIDs = []string{segmentID}
		},
		"a source no UUID": func(e *composition.Event) { e.Fields[0].SourceSegmentIDs = []string{"s1"} },
	} {
		e := valid()
		change(&e)
		if err := e.Validate(); !errors.Is(err, composition.ErrInvalid) {
			t.Errorf("%s: Validate() = %v, want ErrInvalid", name, err)
		}
	}
}

func TestEvent_ValidateNamesTheFieldButNeverItsValue(t *testing.T) {
	t.Parallel()
	e := valid()
	e.Fields[1].Value = "secret value"
	if err := e.Validate(); err == nil || err.Error() != "composition: invalid event: fields[1].value" {
		t.Errorf("Validate() = %v, want %q", err, "composition: invalid event: fields[1].value")
	}
}

func TestMessage_IsTheDraftTypeVersion1(t *testing.T) {
	t.Parallel()
	if composition.MessageType != "draft" || composition.MessageVersion != 1 {
		t.Errorf("message = %q v%d, want draft v1", composition.MessageType, composition.MessageVersion)
	}
}
