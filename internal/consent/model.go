// Package consent is the patients' consents: FHIR R4 Consent, one per patient per clinic, "share my profile
// and appointments with this clinic for care" (docs/booking/SPEC.md "Consent", "Consent shape"). A patient
// grants and revokes their own; revocation ends the clinic's access when its transaction commits. The
// package knows nothing about appointments: they depend on it, never the other way.
package consent

import "time"

// Status is Consent.status: active, or inactive once revoked.
type Status string

// The statuses.
const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// Consent is one patient's consent with one clinic.
type Consent struct {
	DateTime       time.Time // Consent.dateTime: when it was granted
	RevokedAt      time.Time // zero while active
	LastUpdated    time.Time // meta.lastUpdated
	ID             string
	PatientID      string // Consent.patient and performer
	OrganizationID string // Consent.organization and provision.actor (IRCP)
	Status         Status
	VersionID      int32 // meta.versionId
}

// Caller is who asks: the signed-in account and, for staff, its clinic; for a patient, their profile's id,
// resolved by the composition root through the patient service (empty before the profile exists). A caller
// without a clinic is a patient (an account has a clinic exactly when it is staff).
type Caller struct {
	AccountID      string
	OrganizationID string // the caller's clinic; empty for a patient
	PatientID      string // the calling patient's profile; empty for staff and before the first save
}

func (c Caller) patient() bool { return c.OrganizationID == "" }

// RevokedArgs is the consent_revoked job on the default queue: cancel the patient's future appointments with
// the clinic after a revocation (the appointment domain does, docs/booking/SPEC.md "Revocation"). Opaque ids
// only.
type RevokedArgs struct {
	PatientID      string `json:"patient_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (RevokedArgs) Kind() string { return "consent_revoked" }
