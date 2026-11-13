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
	// KindChat lets a practitioner role of another clinic, invited into one of the patient's conversations with a
	// clinic, read and write in that conversation (SAL-28, docs/plans/chat.md): one conversation and one role. It is
	// granted and revoked through GrantChat and RevokeChat only, and is not in Mine or the FHIR reads.
	KindChat Kind = "chat"
	// KindHistory shares chosen visits of the patient's at other clinics with the clinic (ADR 0014,
	// docs/plans/patient-history.md): only the visits it lists, while it and the care consent are active. Off unless
	// the patient grants it at booking or in their settings.
	KindHistory Kind = "history"
)

// Valid reports whether k is one of the kinds a patient grants with a clinic (care, transcription, history); a chat
// consent is granted through GrantChat only.
func (k Kind) Valid() bool { return k == KindCare || k == KindTranscription || k == KindHistory }

// Consent is one patient's consent of one kind with one clinic.
type Consent struct {
	DateTime       time.Time // Consent.dateTime: when it was granted
	RevokedAt      time.Time // zero while active
	LastUpdated    time.Time // meta.lastUpdated
	ID             string
	PatientID      string // Consent.patient and performer
	OrganizationID string // Consent.organization and provision.actor (IRCP)
	// ConversationID and PractitionerRoleID are a chat consent's provision.data and provision.actor; empty for the
	// other kinds.
	ConversationID             string
	PractitionerRoleID         string
	PractitionerOrganizationID string // the invited role's clinic
	Kind                       Kind
	Status                     Status
	// Shared is what a history consent lists now (provision.data), the oldest selection first; nil for the other
	// kinds and for a revoked history consent, which permits nothing.
	Shared    []SharedVisit
	VersionID int32 // meta.versionId
}

// ChatGrant is what a chat consent is for: the patient's conversation with the clinic OrganizationID, and the
// practitioner role of another clinic invited into it, with that role's clinic.
type ChatGrant struct {
	ConversationID             string
	OrganizationID             string
	PractitionerRoleID         string
	PractitionerOrganizationID string
}

// SharedVisit is one visit a history consent lists: the encounter and when the patient added it. Removing it and
// adding it again is a new selection; the record of each is kept (history_consent_encounter, 00043).
type SharedVisit struct {
	AddedAt     time.Time
	EncounterID string
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

// Candidate is one visit the patient may share with a clinic (api/openapi.yaml SharingCandidate, SAL-42): a finished
// visit of theirs at another clinic, as the preview shows it before they send, and exactly as the receiving clinic
// will see it marked once shared: its date, clinic, the specialty (SNOMED CT) of its practitioner role and its title
// (the title of its first signed document; empty when it has none). Selected when the clinic's active history
// consent lists it now.
type Candidate struct {
	Date             time.Time `json:"date"`
	EncounterID      string    `json:"encounterId"`
	OrganizationID   string    `json:"organizationId"`
	OrganizationName string    `json:"organizationName"`
	TimeZone         string    `json:"timeZone"` // the visit's clinic's (IANA): its date is shown in it
	Specialty        string    `json:"specialty,omitempty"`
	Title            string    `json:"title,omitempty"`
	Selected         bool      `json:"selected"`
}

// Preview is the visits a patient may share with one clinic (api/openapi.yaml SharingCandidateList), and the version
// of what they share with it now, to change it at (If-Match); VersionID empty when nothing is shared.
type Preview struct {
	VersionID  string      `json:"versionId,omitempty"`
	Candidates []Candidate `json:"candidates"`
}

// Sharing is what the patient shares with one clinic now (api/openapi.yaml HistorySharing): the history consent, its
// version, and the encounters it lists; ConsentID and VersionID empty when nothing is shared. VersionID is opaque to
// clients and names the consent with its meta.versionId ("<consent id>.<n>"), so a version read of one consent never
// matches another granted after it.
type Sharing struct {
	OrganizationID string   `json:"organizationId"`
	ConsentID      string   `json:"consentId,omitempty"`
	VersionID      string   `json:"versionId,omitempty"`
	Shared         []string `json:"shared"`
}

// SetSharing is the patient's choice of visits to share with a clinic (api/openapi.yaml SetHistorySharingRequest); an
// empty list revokes the sharing.
type SetSharing struct {
	EncounterIDs []string `json:"encounterIds"`
}
