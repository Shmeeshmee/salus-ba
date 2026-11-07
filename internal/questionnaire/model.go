// Package questionnaire is a clinic's library of questionnaires: FHIR R4 Questionnaire, its versions and
// lifecycle, and the default questionnaire of each practitioner role (docs/questionnaire/SPEC.md §2). A
// questionnaire is a series of versions; a version is a draft, then active (published, never changed
// again), then retired. Every practitioner and admin of the clinic builds and publishes them; the service
// scopes every query to the caller's clinic and row-level security shows no other. It holds no patient data.
package questionnaire

import "time"

// Status is Questionnaire.status (FHIR value set publication-status).
type Status string

// The statuses.
const (
	StatusDraft   Status = "draft"
	StatusActive  Status = "active"
	StatusRetired Status = "retired"
)

// Questionnaire is one version of a questionnaire.
type Questionnaire struct {
	LastUpdated    time.Time // meta.lastUpdated and Questionnaire.date
	PublishedAt    time.Time // zero for a draft
	RetiredAt      time.Time // zero unless retired
	ID             string    // the version's FHIR id
	SeriesID       string    // the same for every version of one questionnaire; its canonical url
	OrganizationID string
	Title          string
	Description    string
	// Language is Questionnaire.language, the original language: a BCP 47 tag of the enum language
	// (docs/i18n/SPEC.md §3 "Questionnaires"). Empty on a write keeps the draft's (or, for a new one, bs).
	Language  string
	Status    Status
	Items     []Item
	Version   int32 // Questionnaire.version: 1, 2, …
	VersionID int32 // meta.versionId
}

// Content is what a draft holds and an edit sets.
type Content struct {
	Title        string
	Description  string
	Language     string // the original language; empty keeps the draft's
	Items        []Item
	Translations []Translation
}

// Translation is a version's texts in one other language (docs/i18n/SPEC.md §3 "Questionnaires"), stored as
// the JSON below in questionnaire_translation.texts: the title, the description, and each item's texts by
// linkId. A translation belongs to one version and changes only while it is a draft.
type Translation struct {
	Items       map[string]ItemTranslation `json:"items"`
	Language    string                     `json:"-"` // a tag of the enum language, not the original's
	Title       string                     `json:"title"`
	Description string                     `json:"description,omitempty"`
}

// ItemTranslation is one item's translated text, help text and option texts (by the options' position).
type ItemTranslation struct {
	Text    string   `json:"text,omitempty"`
	Help    string   `json:"help,omitempty"`
	Options []string `json:"options,omitempty"`
}

// ItemType is Questionnaire.item.type. The list is not closed: a type the service does not accept yet is
// refused on save, not by the model or the storage.
type ItemType string

// The item types (SPEC.md §2 "Item model"). Single and multiple choice are both ItemChoice; Repeats tells
// them apart.
const (
	ItemGroup   ItemType = "group"   // a section, top level only
	ItemDisplay ItemType = "display" // instructions
	ItemString  ItemType = "string"
	ItemText    ItemType = "text"
	ItemBoolean ItemType = "boolean"
	ItemChoice  ItemType = "choice"
	ItemInteger ItemType = "integer"
	ItemDecimal ItemType = "decimal"
	ItemDate    ItemType = "date"
	// ItemAttachment is a file upload (docs/attachments/SPEC.md §2 "Item model"), answered with
	// valueAttachment: the files it accepts are Accept, at most MaxFiles of them.
	ItemAttachment ItemType = "attachment"
)

// FileKind is a kind of file an attachment item accepts (docs/attachments/SPEC.md §2 "Item model"); the
// server detects it from the bytes and holds each kind's size limit.
type FileKind string

// The kinds of file.
const (
	FilePhoto    FileKind = "photo"    // JPEG, PNG
	FileDocument FileKind = "document" // PDF
	FileVideo    FileKind = "video"    // MP4
)

// Item is one Questionnaire.item, stored as the JSON below in questionnaire.items.
type Item struct {
	EnableWhen *Condition `json:"enableWhen,omitempty"`
	LinkID     string     `json:"linkId"` // opaque and random, made by the builder
	Text       string     `json:"text"`
	Type       ItemType   `json:"type"`
	// Help is the help text: in FHIR a child display item with the questionnaire-itemControl "help".
	Help     string     `json:"help,omitempty"`
	Options  []Option   `json:"answerOption,omitempty"` // choice only
	Items    []Item     `json:"item,omitempty"`         // a section's questions and instructions
	Accept   []FileKind `json:"accept,omitempty"`       // attachment only: 1 to 3 different kinds
	Required bool       `json:"required,omitempty"`     // questions only
	Repeats  bool       `json:"repeats,omitempty"`      // choice only: several answers
	MaxFiles int32      `json:"maxFiles,omitempty"`     // attachment only: 1 to 10
}

// Option is one Questionnaire.item.answerOption: the clinic's own wording (valueString), so no code is
// invented.
type Option struct {
	Value string `json:"valueString"`
}

// Condition is an item's one enableWhen: shown only when the earlier boolean or choice item Question was
// answered Boolean (a boolean item) or String (one of a choice item's options). The operator is always "=".
type Condition struct {
	Boolean  *bool  `json:"answerBoolean,omitempty"`
	Question string `json:"question"`
	String   string `json:"answerString,omitempty"`
}

// Filter narrows a search to a series (Questionnaire?url=…), a version and a status; zero values match all.
type Filter struct {
	SeriesID string
	Status   Status
	Version  int32
}

// Default is a practitioner role's default questionnaire: a series, whose active version is requested with
// every booking with the role. SeriesID is empty while the role has none.
type Default struct {
	LastUpdated        time.Time
	PractitionerRoleID string
	OrganizationID     string
	SeriesID           string
	VersionID          int32 // zero while the role has none
}

// Caller is who asks: the signed-in account and, for staff, its clinic, whether they are its admin and the
// practitioner they are (empty for an admin who is not one); for a patient, their profile's id.
type Caller struct {
	AccountID      string
	OrganizationID string // empty for a patient
	PractitionerID string
	PatientID      string // a patient's profile; empty for staff and before the profile exists
	Admin          bool
}
