// Package transcript is the bilingual transcript of a consultation (docs/plans/consult-intelligence.md): each
// segment one participant said, in their language, with its translation into the other's, tied to the FHIR
// Encounter. Partial segments are only pushed live; final ones are also stored (transcript_segment). Every
// event of a consultation (transcripts, suggestions, drafts) goes on the encounter's Redis Stream, StreamKey.
package transcript

import (
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// The type and version of a transcript event's wire.Envelope.
const (
	MessageType    = "transcript"
	MessageVersion = 1
)

// Speaker is who said a segment.
type Speaker string

// The speakers.
const (
	SpeakerPractitioner Speaker = "practitioner"
	SpeakerPatient      Speaker = "patient"
)

// Event is one segment as it is recognised: partial ones revise each other under the same SegmentID until the
// final one. Lang is the language it was spoken in (BCP 47); Translation is empty until it is translated.
type Event struct {
	TsStart     time.Time `json:"ts_start"`
	TsEnd       time.Time `json:"ts_end"`
	EncounterID string    `json:"encounter_id"`
	SegmentID   string    `json:"segment_id"`
	Speaker     Speaker   `json:"speaker"`
	Lang        string    `json:"lang"`
	Original    string    `json:"original"`
	Translation string    `json:"translation"`
	IsFinal     bool      `json:"is_final"`
}

// Validate checks the event is one the contract allows; the error names the first wrong field.
func (e Event) Validate() error {
	switch {
	case !fhir.IsUUID(e.EncounterID):
		return invalid("encounter_id")
	case !fhir.IsUUID(e.SegmentID):
		return invalid("segment_id")
	case e.Speaker != SpeakerPractitioner && e.Speaker != SpeakerPatient:
		return invalid("speaker")
	case !validLang(e.Lang):
		return invalid("lang")
	case strings.TrimSpace(e.Original) == "":
		return invalid("original")
	case e.TsStart.IsZero():
		return invalid("ts_start")
	case e.TsEnd.Before(e.TsStart):
		return invalid("ts_end")
	}
	return nil
}

// StreamKey is the Redis Stream of every live event of an encounter's consultation.
func StreamKey(encounterID string) string { return "consult:" + encounterID + ":events" }

// validLang accepts a BCP 47 tag's shape: a primary language of two or three lower-case letters, then subtags
// of one to eight letters or digits. The engines decide which languages they support.
func validLang(tag string) bool {
	primary, rest, _ := strings.Cut(tag, "-")
	if len(primary) < 2 || len(primary) > 3 || strings.Trim(primary, "abcdefghijklmnopqrstuvwxyz") != "" {
		return false
	}
	if rest == "" {
		return !strings.HasSuffix(tag, "-")
	}
	for sub := range strings.SplitSeq(rest, "-") {
		if len(sub) < 1 || len(sub) > 8 || !alnum(sub) {
			return false
		}
	}
	return true
}

func alnum(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
