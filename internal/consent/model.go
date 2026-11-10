// Package consent is the patients' consents: FHIR R4 Consent, per patient and clinic, of two kinds. The care
// consent, "share my profile and appointments with this clinic for care" (docs/booking/SPEC.md "Consent",
// "Consent shape"), and the transcription consent, "transcribe and translate my consultations with this clinic,
// with AI assistance for the doctor" (SAL-12, docs/plans/v1-finish.md "Decisions"). A patient grants and revokes
// their own; a revocation takes effect when its transaction commits. The package knows nothing about
// appointments or consultations: they depend on it, never the other way.
package consent

import "time"

// Status is Consent.status: active, or inactive once revoked.
type Status string

// The statuses.
const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// Kind is what a consent is for.
type Kind string

// The kinds. A patient has at most one active consent of each kind with a clinic.
const (
	// KindCare shares the patient's profile and appointments with the clinic for care.
	KindCare Kind = "care"
	// KindTranscription lets the clinic's consultations with the patient be transcribed and translated, with AI
	// assistance for the practitioner (suggestions, document drafts). It is used only with an active care consent.
	KindTranscription Kind = "transcription"
)

// Valid reports whether k is one of the kinds.
func (k Kind) Valid() bool { return k == KindCare || k == KindTranscription }

// Consent is one patient's consent of one kind with one clinic.
type Consent struct {
	DateTime       time.Time // Consent.dateTime: when it was granted
	RevokedAt      time.Time // zero while active
	LastUpdated    time.Time // meta.lastUpdated
	ID             string
	PatientID      string // Consent.patient and performer
	OrganizationID string // Consent.organization and provision.actor (IRCP)
	Kind           Kind
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

// TranscriptionRevokedArgs is the transcription_consent_revoked job on the default queue: take the transcription
// agent out of the patient's consultations with the clinic after they withdrew the transcription consent (the
// consultation domain does; the call goes on). Opaque ids only.
type TranscriptionRevokedArgs struct {
	PatientID      string `json:"patient_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (TranscriptionRevokedArgs) Kind() string { return "transcription_consent_revoked" }
