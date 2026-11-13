// Package questionnaireresponse is a request to a patient to fill in one questionnaire version before one
// appointment, and the patient's answers: FHIR R4 QuestionnaireResponse (docs/questionnaire/SPEC.md §2
// "Requests", "Patient flow", "Access rule"). The appointment's practitioner adds one to an appointment (and a
// role's default is requested with every booking); the patient answers on mobile and submits once; the
// practitioner reads the answers once submitted. Answers are health data: who sees what is decided by
// access.QuestionnaireResponse, and every read and write is audited, refusals included.
package questionnaireresponse

import (
	"encoding/json"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/binary"
)

// Status is QuestionnaireResponse.status, the three the product uses.
type Status string

// The statuses.
const (
	StatusInProgress Status = "in-progress" // requested, the patient may answer
	StatusCompleted  Status = "completed"   // submitted, final
	StatusStopped    Status = "stopped"     // withdrawn by the practitioner, with no answers
)

// Source is how a response was requested.
type Source string

// The sources.
const (
	SourceDefault Source = "default" // the role's default, with the booking
	SourceAdded   Source = "added"   // added to the appointment by its practitioner
)

// Response is one request and its answers. OrganizationID, PatientID, PractitionerRoleID and DueAt are
// copied once from the appointment.
type Response struct {
	DueAt              time.Time // the appointment's start: the answers are due before it
	AuthoredAt         time.Time // QuestionnaireResponse.authored: the submit; zero before
	LastUpdated        time.Time // meta.lastUpdated
	ID                 string
	OrganizationID     string
	AppointmentID      string // the questionnaireresponse-appointment extension
	PatientID          string // QuestionnaireResponse.subject and author
	PractitionerRoleID string
	QuestionnaireID    string // the exact version asked
	SeriesID           string // the questionnaire, whatever its version
	Source             Source
	Status             Status
	// Language is QuestionnaireResponse.language, the language the patient answers in (docs/i18n/SPEC.md §3
	// "The patient's language"): a tag of the enum language, empty until they start.
	Language string
	Answers  []Answer // a flat list; the FHIR handler nests it under the version's sections
	// Attachments are the files of the attachment items, in upload order (docs/attachments/SPEC.md §2 "Where
	// an attachment answer lives"): kept in step by binary through the holder, outside Answers and VersionID.
	Attachments []binary.Attachment
	VersionID   int32 // meta.versionId
}

// Answer is the answer to one item, stored as the JSON below in questionnaire_response.answers.
type Answer struct {
	LinkID string  `json:"linkId"`
	Values []Value `json:"values"`
}

// Value is one QuestionnaireResponse.item.answer.value[x]: exactly one field is set, after the item's type.
// The list is not closed (valueAttachment comes with the attachments feature).
type Value struct {
	Boolean *bool       `json:"valueBoolean,omitempty"`
	Integer *int32      `json:"valueInteger,omitempty"`
	Decimal json.Number `json:"valueDecimal,omitempty"` // as written, so no digit is lost
	String  string      `json:"valueString,omitempty"`  // string, text and choice
	Date    string      `json:"valueDate,omitempty"`    // YYYY-MM-DD
}

// Seen is a response as a viewer may see it: at Full everything; at Name (that it exists, its status and
// when it was submitted) no answers.
type Seen struct {
	Response Response
	Level    access.Level
}

// Attached is a file of a completed response as the patient's timeline lists it (SAL-39): the response, without
// its answers, and the file as the response lists it.
type Attached struct {
	Response Response
	File     binary.Attachment
}

// Filter narrows a search: the appointment, the patient and the status, when set.
type Filter struct {
	AppointmentID string
	PatientID     string
	Status        Status
}
