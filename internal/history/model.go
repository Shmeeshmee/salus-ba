// Package history is the patient's history as one view over the domains that own it (docs/plans/patient-history.md,
// ADR 0014): the patient's timeline across every clinic (SAL-39), and a practitioner's view of the visits they may
// see with the pre-visit summary (SAL-40). It owns no table: each item comes from its owning domain's service, which
// applies its access rule and audits the read. Nothing the viewer may not see (access.HistoryVisit) is listed,
// counted, searched or summarised.
package history

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/medicationstatement"
)

// ItemType is what a history item is. Each is a FHIR resource the item's Reference opens.
type ItemType string

// The item types. TypeCommunication is the extension point for chat messages (SAL-26, being built): it is not served
// until that domain exists; a source for it plugs into the timeline as the others do.
const (
	TypeVisit                 ItemType = "visit"                  // Encounter
	TypeDocument              ItemType = "document"               // a signed Composition
	TypeQuestionnaireResponse ItemType = "questionnaire_response" // a completed QuestionnaireResponse
	TypeAttachment            ItemType = "attachment"             // a clean Binary of a completed response
	TypeCommunication         ItemType = "communication"          // reserved: a chat's Communication
)

// Clinic is the clinic an item comes from, by id and name.
type Clinic struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Item is one entry of a history (api/openapi.yaml HistoryItem). Date orders the history, newest first; Reference
// is the FHIR resource to open ("Encounter/…", "Composition/…", "QuestionnaireResponse/…", "Binary/…"); EncounterID
// the visit it belongs to (empty for an item of none); Specialty the SNOMED CT code of the visit's practitioner
// role; Title the document's, questionnaire's or file's title; Diagnoses, of a signed document, the ICD-10 codes
// (http://hl7.org/fhir/sid/icd-10) of its filled coded fields. Shared marks, for a practitioner, a visit of another
// clinic the patient shared, and everything of it.
type Item struct {
	Date        time.Time `json:"date"`
	Clinic      Clinic    `json:"clinic"`
	Type        ItemType  `json:"type"`
	Reference   string    `json:"reference"`
	EncounterID string    `json:"encounterId,omitempty"`
	Specialty   string    `json:"specialty,omitempty"`
	Title       string    `json:"title,omitempty"`
	Diagnoses   []string  `json:"diagnoses,omitempty"`
	Shared      bool      `json:"shared,omitempty"`
}

// Page is one page of a history, newest first, with the cursor of the next page (api/openapi.yaml HistoryPage);
// Next is empty on the last page.
type Page struct {
	Next  string `json:"next,omitempty"`
	Items []Item `json:"items"`
}

// Filter narrows a history: one clinic, some types, a date range [From, To); zero values do not narrow.
type Filter struct {
	From           time.Time
	To             time.Time
	OrganizationID string
	Types          []ItemType
}

// ErrCursor is a cursor this package did not make.
var ErrCursor = errors.New("history: invalid cursor")

// ErrNotFound is a patient, or an appointment, the caller has no history of: nothing about it is disclosed.
var ErrNotFound = errors.New("history: not found")

// Cursor is a position in a history: the date and reference of the last item of a page. The zero Cursor is the
// start.
type Cursor struct {
	Date      time.Time
	Reference string
}

// CursorAt is the position of item.
func CursorAt(item Item) Cursor { return Cursor{Date: item.Date, Reference: item.Reference} }

// Before reports whether item comes after the position in the history's order (newest first, the reference
// breaking a tie), that is, on a later page; every item does at the start.
func (c Cursor) Before(item Item) bool {
	if c.Date.IsZero() {
		return true
	}
	if !item.Date.Equal(c.Date) {
		return item.Date.Before(c.Date)
	}
	return item.Reference < c.Reference
}

// String is the cursor as the API sends it: opaque, URL-safe; empty at the start.
func (c Cursor) String() string {
	if c.Date.IsZero() {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(c.Date.UTC().Format(time.RFC3339Nano) + " " + c.Reference))
}

// ParseCursor reads a cursor String made; "" is the start.
func ParseCursor(s string) (Cursor, error) {
	if s == "" {
		return Cursor{}, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrCursor
	}
	date, ref, ok := strings.Cut(string(b), " ")
	at, err := time.Parse(time.RFC3339Nano, date)
	if !ok || err != nil || at.IsZero() || !validReference(ref) {
		return Cursor{}, ErrCursor
	}
	return Cursor{Date: at, Reference: ref}, nil
}

// validReference reports whether ref is Type/id with a type of the history.
func validReference(ref string) bool {
	typ, id, ok := strings.Cut(ref, "/")
	if !ok || id == "" || strings.ContainsAny(id, "/ ") {
		return false
	}
	switch typ {
	case "Encounter", "Composition", "QuestionnaireResponse", "Binary", "Communication":
		return true
	}
	return false
}

// Summary is the pre-visit summary at the top of a practitioner's view of the patient (api/openapi.yaml
// HistorySummary, SAL-40): the allergies and current medications the practitioner may see, the latest visit they
// may see, and what is open for the appointment. Built from the same visibility as the view: a visit not seen is in
// none of it.
type Summary struct {
	LastVisit   *Item               `json:"lastVisit,omitempty"`
	Allergies   []SummaryAllergy    `json:"allergies"`
	Medications []SummaryMedication `json:"medications"`
	OpenItems   []OpenItem          `json:"openItems"`
}

// SummaryAllergy is an active allergy in the summary: the substance, the reaction and its severity, whether a
// practitioner confirmed it, and the AllergyIntolerance to open.
type SummaryAllergy struct {
	Reference string `json:"reference"`
	Text      string `json:"text"`
	Reaction  string `json:"reaction,omitempty"`
	Severity  string `json:"severity,omitempty"`
	Confirmed bool   `json:"confirmed"`
}

// SummaryMedication is an active medication in the summary, with how it is taken.
type SummaryMedication struct {
	Reference string                     `json:"reference"`
	Text      string                     `json:"text"`
	Dosage    medicationstatement.Dosage `json:"dosage"`
	Confirmed bool                       `json:"confirmed"`
}

// OpenItemType is what is open for the appointment.
type OpenItemType string

// The open items: a questionnaire the patient has not completed, a document the practitioner has not signed.
const (
	OpenQuestionnaire OpenItemType = "questionnaire_pending"
	OpenDraft         OpenItemType = "draft_unsigned"
)

// OpenItem is one thing open for the appointment, and the resource to open.
type OpenItem struct {
	Type      OpenItemType `json:"type"`
	Reference string       `json:"reference"`
}
