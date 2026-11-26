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

// Visit is an encounter in a practitioner's view of the patient's history (ADR 0014): Shared marks another
// clinic's visit the patient shared with the practitioner's clinic.
type Visit struct {
	Encounter
	Shared bool
}

// Filter narrows a search: the appointment, the patient and the practitioner behind the role, when set. A page of it
// starts after the position (AfterStart, AfterID) in the search's order, by period start then id, when AfterID is set,
// and holds Limit encounters at most when Limit is (SearchPage).
type Filter struct {
	AfterStart     time.Time
	AppointmentID  string
	PatientID      string
	PractitionerID string
	AfterID        string
	Limit          int // set by the service; 0 for every match
}

// Page is one page of a search: its encounters, and whether more may follow the last (the next page starts after it).
type Page struct {
	Encounters []Encounter
	More       bool
}
