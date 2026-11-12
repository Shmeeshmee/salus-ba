// Package condition is the coded diagnoses of the signed documents (docs/plans/analytics.md §3, ADR 0012): one
// FHIR Condition (category encounter-diagnosis) per filled coded field of a signed Composition, written in the
// signing transaction and never changed, the source of the analytics. Owns the condition table.
package condition

import "time"

// SystemICD10 is the code system of a Condition's code.
const SystemICD10 = "http://hl7.org/fhir/sid/icd-10"

// The verification statuses a Condition takes (http://terminology.hl7.org/CodeSystem/condition-ver-status):
// confirmed by the signing practitioner, or unconfirmed (backfilled from a document signed before the
// confirmation existed, not counted by the analytics).
const (
	VerificationConfirmed   = "confirmed"
	VerificationUnconfirmed = "unconfirmed"
)

// Condition is a diagnosis of a signed document: Condition.subject (the patient), .encounter, .recorder (the
// signing practitioner role), .recordedDate (the signing time), .code (ICD-10), .verificationStatus, and the
// document and field it came from (.evidence.detail). Health data (GDPR Art. 9).
type Condition struct {
	RecordedAt         time.Time
	LastUpdated        time.Time // meta.lastUpdated; set when read
	ID                 string
	OrganizationID     string
	PatientID          string
	EncounterID        string
	DocumentID         string
	PractitionerRoleID string
	FieldKey           string
	Code               string
	VerificationStatus string
	VersionID          int32 // meta.versionId; set when read
}

// Diagnosis is a confirmed Condition as the aggregate builds read it from the store: no document, no field.
type Diagnosis struct {
	RecordedAt         time.Time
	PatientID          string
	PractitionerRoleID string
	Code               string
}

// Recorded is a confirmed Condition as a build gets it: the patient only as a pseudonym under the build's key.
type Recorded struct {
	RecordedAt         time.Time
	PractitionerRoleID string
	Code               string
	Patient            [32]byte
}
