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
	"time"

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
	// VersionID is the draft's version (meta.versionId), which a sign names (If-Match) and by which a page
	// tells a redraft from a replay of the one it has. Absent (0) only in an event from before SAL-8.
	VersionID int32 `json:"version_id,omitempty"`
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

// FieldType is what a template field holds (llm.FieldType names the same four for the model).
type FieldType string

// The field types: free text; a list, one item per line; a date, YYYY-MM-DD; a coded term, one ICD-10 code.
const (
	FieldText  FieldType = "text"
	FieldList  FieldType = "list"
	FieldDate  FieldType = "date"
	FieldCoded FieldType = "coded"
)

// TemplateStatus is whether a template's wording has been reviewed by its clinic: a draft (as the seeded
// examples are, placeholders for the clinic's doctor to review) or active. Both can be drafted from.
type TemplateStatus string

// The template statuses.
const (
	TemplateDraft  TemplateStatus = "draft"
	TemplateActive TemplateStatus = "active"
)

// SystemLOINC is the code system of a template's document type (Composition.type, DocumentReference.type).
const SystemLOINC = "http://loinc.org"

// SystemICD10 is the code system of a coded field's value.
const SystemICD10 = "http://hl7.org/fhir/sid/icd-10"

// Template is a clinic's document template: its title and fields in its original language, and its
// translations. No personal data.
type Template struct {
	LastUpdated    time.Time
	ID             string
	OrganizationID string
	Title          string
	Language       string // the original language, a tag of the enum language
	Status         TemplateStatus
	TypeCode       string // the LOINC code of the document type
	Fields         []TemplateField
	Translations   []Translation // in the enum's order
	VersionID      int32
}

// TemplateField is one field of a template, as stored in document_template.fields: its key (stable across
// languages and versions), type, label and instruction (what the model and the practitioner read), and
// whether the document needs it.
type TemplateField struct {
	Key         string    `json:"key"`
	Type        FieldType `json:"type"`
	Label       string    `json:"label"`
	Instruction string    `json:"instruction,omitempty"`
	Required    bool      `json:"required"`
}

// Translation is a template's texts in one other language, stored as the JSON below in
// document_template_translation.texts: the title and each field's texts by key. It is complete or refused.
type Translation struct {
	Fields   map[string]FieldTexts `json:"fields"`
	Language string                `json:"-"`
	Title    string                `json:"title"`
}

// FieldTexts are a field's texts in a translation.
type FieldTexts struct {
	Label       string `json:"label"`
	Instruction string `json:"instruction,omitempty"`
}

// TemplateContent is what a write of a template sets.
type TemplateContent struct {
	Title        string
	Language     string
	Status       TemplateStatus
	TypeCode     string
	Fields       []TemplateField
	Translations []Translation
}

// In returns the template's title and fields in the language lang, the original's or a translation's, and
// false for a language it does not have.
func (t Template) In(lang string) (string, []TemplateField, bool) {
	if lang == t.Language {
		return t.Title, t.Fields, true
	}
	for _, tr := range t.Translations {
		if tr.Language != lang {
			continue
		}
		fields := make([]TemplateField, len(t.Fields))
		for i, f := range t.Fields {
			texts := tr.Fields[f.Key]
			f.Label, f.Instruction = texts.Label, texts.Instruction
			fields[i] = f
		}
		return tr.Title, fields, true
	}
	return "", nil, false
}

// Draft is a document drafted from a template for an encounter: a FHIR Composition, preliminary while it is
// drafted and edited, final once its practitioner signed it. It keeps the title, language, type and field texts
// it was drafted with. A signed one has its DocumentReference (the PDF) and its Provenance.
type Draft struct {
	UpdatedAt           time.Time // meta.lastUpdated
	SignedAt            time.Time // zero until signed: Composition.attester.time, Provenance.recorded
	ID                  string
	EncounterID         string
	TemplateID          string
	Language            string
	Title               string
	TypeCode            string // LOINC
	SignerRoleID        string // the PractitionerRole who signed
	DocumentReferenceID string
	ProvenanceID        string
	Status              Status
	Fields              []DraftField
	PDF                 PDF
	VersionID           int32 // meta.versionId
}

// DraftField is a field of a draft as stored in document_draft.fields: the event's field with the template
// field's type, label, instruction and required flag as they were drafted.
type DraftField struct {
	Field
	Type        FieldType `json:"type"`
	Label       string    `json:"label"`
	Instruction string    `json:"instruction,omitempty"`
	Required    bool      `json:"required"`
}

// PDF is the stored rendering of a signed draft: its plaintext size and SHA-1, and the data key of its
// encryption, wrapped by the key-encryption key KEKID. The ciphertext is in the object store under ObjectKey.
type PDF struct {
	KEKID      string
	SHA1       []byte
	WrappedKey []byte
	Size       int64
}

// ObjectKey is the object store key of the PDF of the DocumentReference id. The prefix keeps it out of the
// attachments' purger, which lists the store's top level only.
func ObjectKey(documentReferenceID string) string { return "composition/" + documentReferenceID }

// Event is the draft as it is pushed to the practitioner.
func (d Draft) Event() Event {
	fields := make([]Field, len(d.Fields))
	for i, f := range d.Fields {
		fields[i] = f.Field
	}
	return Event{DraftID: d.ID, EncounterID: d.EncounterID, TemplateID: d.TemplateID, Status: d.Status, Fields: fields, VersionID: d.VersionID}
}

// Value is a field's value as the practitioner leaves it when they sign.
type Value struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
