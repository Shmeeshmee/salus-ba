// Package medicationstatement is the patient's current medications: FHIR R4 MedicationStatement
// (docs/plans/patient-history.md, ADR 0013, SAL-41). Same sources and visibility as allergyintolerance: the patient's
// own entry is always in their profile, a clinic's belongs to its visit until the patient adds it to the profile;
// which clinic recorded or confirmed it is shown only to the patient and that clinic. Every read and write is
// audited.
package medicationstatement

import "time"

// Status is MedicationStatement.status, the three we use: active (a current medication), stopped, and
// entered-in-error (a mistaken entry kept with the record, shown to no one).
type Status string

// The statuses.
const (
	StatusActive         Status = "active"
	StatusStopped        Status = "stopped"
	StatusEnteredInError Status = "entered-in-error"
)

// DoseUnits returns the dose units, UCUM codes (http://unitsofmeasure.org): units, and the annotated counts a patient
// doses in, as the table's check (00042) lists them.
func DoseUnits() []string {
	return []string{"mg", "g", "ug", "mL", "[iU]", "{tablet}", "{capsule}", "{drop}", "{puff}", "{patch}"}
}

// PeriodUnits returns the period units, UCUM codes as FHIR Timing.repeat.periodUnit.
func PeriodUnits() []string { return []string{"h", "d", "wk", "mo"} }

// Code is the medication: a term of the terminology when picked from it (its id and ATC code), and the text shown.
type Code struct {
	TermID string
	ATC    string
	Text   string
}

// Source is where an entry comes from: a clinic's visit and the role that recorded it, or, all empty, the patient.
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

// Dosage is how the medication is taken: a dose (value and UCUM unit, both or neither) and how often (frequency
// times per period of period unit, all three or none), or as needed. Zero values are absent.
type Dosage struct {
	DoseUnit   string  `json:"doseUnit,omitempty"`
	PeriodUnit string  `json:"periodUnit,omitempty"`
	DoseValue  float64 `json:"doseValue,omitempty"`
	Period     float64 `json:"period,omitempty"`
	Frequency  int32   `json:"frequency,omitempty"`
	AsNeeded   bool    `json:"asNeeded,omitempty"`
}

// MedicationStatement is one medication of one patient.
type MedicationStatement struct {
	DateAsserted time.Time // MedicationStatement.dateAsserted
	LastUpdated  time.Time // meta.lastUpdated
	Confirmation Confirmation
	ID           string
	PatientID    string // MedicationStatement.subject
	Source       Source
	Code         Code
	Status       Status
	Dosage       Dosage
	VersionID    int32 // meta.versionId
	InProfile    bool  // seen as the profile is; always true for the patient's own
}

// Input is what the patient or a practitioner writes (api/openapi.yaml MedicationStatementInput): the medication from
// the terminology (termId) or typed (text), how it is taken, and whether it is current.
type Input struct {
	TermID string `json:"termId,omitempty"`
	Text   string `json:"text"`
	Status Status `json:"status"`
	Dosage Dosage `json:"dosage"`
}

// Seen is a medication as one caller may see it (access.ProfileEntry): Attributed (access.Attributed) when the
// caller may see which clinic recorded or confirmed it; without it nothing derived from Source or Confirmation but
// whether it is confirmed may reach the caller.
type Seen struct {
	MedicationStatement
	Attributed bool
}
