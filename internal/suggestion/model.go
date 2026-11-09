// Package suggestion is what the suggestions worker proposes to the practitioner during a consultation
// (docs/plans/consult-intelligence.md): a diagnosis to consider, a question to ask, a red flag, or a term to
// explain, each citing the transcript segments it came from. A suggestion is a prompt, never a diagnosis; only
// the practitioner sees it, and dismisses or pins it. Stored in the suggestion table, with what the
// practitioner did with it.
package suggestion

import (
	"strings"
	"unicode/utf8"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// The type and version of a suggestion event's wire.Envelope.
const (
	MessageType    = "suggestion"
	MessageVersion = 1
)

// Type is what kind of suggestion it is.
type Type string

// The types.
const (
	TypeDiagnosis Type = "diagnosis"
	TypeQuestion  Type = "question"
	TypeRedFlag   Type = "red_flag"
	TypeTerm      Type = "term"
)

// Status is what the practitioner did with a suggestion: nothing yet (shown), dismissed or pinned it.
type Status string

// The statuses.
const (
	StatusShown     Status = "shown"
	StatusDismissed Status = "dismissed"
	StatusPinned    Status = "pinned"
)

// Origin says where a suggestion comes from. There is one: an AI model. Every suggestion carries it, so whatever
// shows one labels it as AI assistance, never as a diagnosis.
type Origin string

// OriginAI is a suggestion from a language model (llm.Suggester): AI assistance for the practitioner.
const OriginAI Origin = "ai"

// MaxTextLength bounds a suggestion's text, in characters: a prompt to glance at during a consultation, not a
// paragraph.
const MaxTextLength = 300

// Event is a suggestion as it is pushed to the practitioner, first shown, then again on every change of its
// status. Confidence is the model's, from 0 to 1. Origin is always OriginAI.
type Event struct {
	ID               string   `json:"id"`
	EncounterID      string   `json:"encounter_id"`
	Type             Type     `json:"type"`
	Text             string   `json:"text"`
	Status           Status   `json:"status"`
	Origin           Origin   `json:"origin"`
	SourceSegmentIDs []string `json:"source_segment_ids"`
	Confidence       float32  `json:"confidence"`
}

// Validate checks the event is one the contract allows; the error names the first wrong field.
func (e Event) Validate() error {
	switch {
	case !fhir.IsUUID(e.ID):
		return invalid("id")
	case !fhir.IsUUID(e.EncounterID):
		return invalid("encounter_id")
	case e.Type != TypeDiagnosis && e.Type != TypeQuestion && e.Type != TypeRedFlag && e.Type != TypeTerm:
		return invalid("type")
	case strings.TrimSpace(e.Text) == "" || utf8.RuneCountInString(e.Text) > MaxTextLength:
		return invalid("text")
	case !(e.Confidence >= 0 && e.Confidence <= 1): // NaN fails both
		return invalid("confidence")
	case !validSources(e.SourceSegmentIDs):
		return invalid("source_segment_ids")
	case e.Status != StatusShown && e.Status != StatusDismissed && e.Status != StatusPinned:
		return invalid("status")
	case e.Origin != OriginAI:
		return invalid("origin")
	}
	return nil
}

// validSources requires at least one segment, each a UUID: every suggestion says where it came from.
func validSources(ids []string) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		if !fhir.IsUUID(id) {
			return false
		}
	}
	return true
}
