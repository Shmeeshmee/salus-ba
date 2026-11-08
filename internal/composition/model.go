// Package composition is the documents a practitioner writes during a consultation from a clinic's template
// (docs/plans/consult-intelligence.md): a referral, a visit summary. The draft is a FHIR Composition: status
// preliminary while the drafting service fills its fields from the transcript and the practitioner edits it,
// final once they sign it off, entered-in-error if they discard it. Each filled field cites the transcript
// segments it came from; a field the conversation has not covered yet is marked missing. Owns the
// document_template and document_draft tables.
package composition

import (
	"strconv"
	"strings"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// The type and version of a draft event's wire.Envelope.
const (
	MessageType    = "draft"
	MessageVersion = 1
)

// Status is Composition.status, the three a draft goes through.
type Status string

// The statuses.
const (
	StatusPreliminary    Status = "preliminary"
	StatusFinal          Status = "final"
	StatusEnteredInError Status = "entered-in-error"
)

// Event is a draft as it is pushed to the practitioner, on every change. Fields follow the template's order.
type Event struct {
	DraftID     string  `json:"draft_id"`
	EncounterID string  `json:"encounter_id"`
	TemplateID  string  `json:"template_id"`
	Status      Status  `json:"status"`
	Fields      []Field `json:"fields"`
}

// Field is one field of the draft, by its template key. A filled field has a value and the segments it came
// from (none when the practitioner typed it); a missing one has neither.
type Field struct {
	Key              string   `json:"key"`
	Value            string   `json:"value"`
	SourceSegmentIDs []string `json:"source_segment_ids,omitempty"`
	Missing          bool     `json:"missing"`
}

// Validate checks the event is one the contract allows; the error names the first wrong field.
func (e Event) Validate() error {
	switch {
	case !fhir.IsUUID(e.DraftID):
		return invalid("draft_id")
	case !fhir.IsUUID(e.EncounterID):
		return invalid("encounter_id")
	case !fhir.IsUUID(e.TemplateID):
		return invalid("template_id")
	case e.Status != StatusPreliminary && e.Status != StatusFinal && e.Status != StatusEnteredInError:
		return invalid("status")
	case len(e.Fields) == 0:
		return invalid("fields")
	}
	keys := make(map[string]bool, len(e.Fields))
	for i, f := range e.Fields {
		if name := f.invalidField(); name != "" {
			return invalid("fields[" + strconv.Itoa(i) + "]." + name)
		}
		if keys[f.Key] {
			return invalid("fields[" + strconv.Itoa(i) + "].key")
		}
		keys[f.Key] = true
	}
	return nil
}

// invalidField returns the name of the field's first wrong element, or "".
func (f Field) invalidField() string {
	switch {
	case strings.TrimSpace(f.Key) == "":
		return "key"
	case f.Missing == (strings.TrimSpace(f.Value) != ""):
		return "value"
	case f.Missing && len(f.SourceSegmentIDs) > 0:
		return "source_segment_ids"
	}
	for _, id := range f.SourceSegmentIDs {
		if !fhir.IsUUID(id) {
			return "source_segment_ids"
		}
	}
	return ""
}
