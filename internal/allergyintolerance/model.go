// Package allergyintolerance is the patient's allergies: FHIR R4 AllergyIntolerance (docs/plans/patient-history.md,
// ADR 0014, SAL-41). An allergy is the patient's own, entered in the app and always in their profile, or recorded
// by a clinic in one of its visits, which belongs to that visit until the patient adds it to their profile. The
// profile is seen by every clinic with the patient's active care consent and an appointment; a visit's entry
// wherever the visit is seen (access.ProfileEntry). Which clinic recorded or confirmed an entry is shown only to the
// patient and to that clinic (access.Attributed). Every read and write is audited.
package allergyintolerance

import "time"

// ClinicalStatus is AllergyIntolerance.clinicalStatus (allergyintolerance-clinical).
type ClinicalStatus string

// The clinical statuses.
const (
	ClinicalActive   ClinicalStatus = "active"
	ClinicalInactive ClinicalStatus = "inactive"
	ClinicalResolved ClinicalStatus = "resolved"
)

// VerificationStatus is AllergyIntolerance.verificationStatus (allergyintolerance-verification), the three we use:
// the patient's own entry is unconfirmed until a practitioner confirms it; a clinic's entry is confirmed when
// recorded; a mistaken entry that is kept with the record is entered-in-error and shown to no one.
type VerificationStatus string

// The verification statuses.
const (
	VerificationUnconfirmed    VerificationStatus = "unconfirmed"
	VerificationConfirmed      VerificationStatus = "confirmed"
	VerificationEnteredInError VerificationStatus = "entered-in-error"
)

// Category is AllergyIntolerance.category (allergy-intolerance-category); empty for none.
type Category string

// The categories.
const (
	CategoryFood        Category = "food"
	CategoryMedication  Category = "medication"
	CategoryEnvironment Category = "environment"
	CategoryBiologic    Category = "biologic"
)

// Severity is AllergyIntolerance.reaction.severity (reaction-event-severity); empty for none. A severity needs a
// reaction.
type Severity string

// The severities.
const (
	SeverityMild     Severity = "mild"
	SeverityModerate Severity = "moderate"
	SeveritySevere   Severity = "severe"
)

// Code is what an entry is about: a term of the terminology when it was picked from it (its id, and its ATC code
// when it has one), and the text shown, which is the term's label in the language it was picked in or what the
// patient typed.
type Code struct {
	TermID string
	ATC    string
	Text   string
}

// Source is where an entry comes from: a clinic's visit and the practitioner role that recorded it, or, all empty,
// the patient.
type Source struct {
	OrganizationID string
	EncounterID    string
	RecorderRoleID string
}

// Patient reports whether the entry is the patient's own.
func (s Source) Patient() bool { return s.OrganizationID == "" }

// Confirmation is who confirmed an entry and when; zero while unconfirmed.
type Confirmation struct {
	At             time.Time
	RoleID         string
	OrganizationID string
}

// AllergyIntolerance is one allergy of one patient.
type AllergyIntolerance struct {
	RecordedDate       time.Time // AllergyIntolerance.recordedDate
	LastUpdated        time.Time // meta.lastUpdated
	Confirmation       Confirmation
	ID                 string
	PatientID          string // AllergyIntolerance.patient
	Source             Source
	Code               Code
	Reaction           string // reaction.manifestation.text; empty for none
	Category           Category
	Severity           Severity
	ClinicalStatus     ClinicalStatus
	VerificationStatus VerificationStatus
	VersionID          int32 // meta.versionId
	InProfile          bool  // seen as the profile is; always true for the patient's own
}

// Input is what the patient or a practitioner writes (api/openapi.yaml AllergyIntoleranceInput): the substance from
// the terminology (termId) or typed (text), its category, the reaction and its severity, and whether it is current.
type Input struct {
	TermID         string         `json:"termId,omitempty"`
	Text           string         `json:"text"`
	Category       Category       `json:"category,omitempty"`
	Reaction       string         `json:"reaction,omitempty"`
	Severity       Severity       `json:"severity,omitempty"`
	ClinicalStatus ClinicalStatus `json:"clinicalStatus"`
}

// Seen is an allergy as one caller may see it (access.ProfileEntry): Attributed (access.Attributed) when the caller
// may see which clinic recorded or confirmed it; without it nothing derived from Source or Confirmation but whether
// it is confirmed may reach the caller.
type Seen struct {
	AllergyIntolerance
	Attributed bool
}

// EntryWritten is the answer to a write (api/openapi.yaml EntryWritten): the entry's id and its version, which its
// ETag repeats; the entry itself is read through the FHIR API.
type EntryWritten struct {
	ID        string `json:"id"`
	VersionID string `json:"versionId"`
}

// Provenance is a practitioner's write of an allergy: FHIR R4 Provenance, its target the version the write produced,
// its agent the practitioner role on behalf of its clinic. Activity is a v3-DataOperation code (fhir.ActivityCreate,
// fhir.ActivityUpdate, fhir.ActivityNullify for a removal) and AgentType a provenance-participant-type one
// (fhir.AgentAuthor, fhir.AgentVerifier for a confirmation). The patient's own writes have none: the entry names the
// patient as its asserter and the AuditEvent records the write.
type Provenance struct {
	Recorded       time.Time // Provenance.recorded: when the version was written
	ID             string
	EntryID        string // the AllergyIntolerance written
	RoleID         string // agent.who: PractitionerRole
	OrganizationID string // agent.onBehalfOf: Organization
	Activity       string
	AgentType      string
	Version        int32 // the target's version
}
