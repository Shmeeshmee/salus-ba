// Package encounter is the FHIR R4 Encounter of an appointment's video consultation
// (docs/consultation/SPEC.md §2 "Encounter"): class VR (virtual), started when the practitioner first joins
// and finished when the consultation ends. It holds no clinical content, only that a consultation took place
// and when. The lifecycle (videosession) drives it; there is no FHIR create or update. A finished encounter
// never changes again. Who sees it follows its appointment (access.Appointment); every read and write is
// audited.
package encounter

import "time"

// Status is Encounter.status, the two the consultation uses.
type Status string

// The statuses.
const (
	StatusInProgress Status = "in-progress"
	StatusFinished   Status = "finished"
)

// Encounter is one consultation. PatientID and PractitionerRoleID are its subject and participant, copied
// once from the appointment when it starts.
type Encounter struct {
	PeriodStart        time.Time // Encounter.period.start: the practitioner's first join
	PeriodEnd          time.Time // Encounter.period.end; zero while in progress
	LastUpdated        time.Time // meta.lastUpdated
	ID                 string
	AppointmentID      string // Encounter.appointment
	OrganizationID     string // Encounter.serviceProvider
	PatientID          string // Encounter.subject
	PractitionerRoleID string // Encounter.participant.individual
	Status             Status
	VersionID          int32 // meta.versionId
}

// Filter narrows a search: the appointment, the patient and the practitioner behind the role, when set.
type Filter struct {
	AppointmentID  string
	PatientID      string
	PractitionerID string
}
