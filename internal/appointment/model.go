// Package appointment is a patient's booking of a slot with a practitioner: FHIR R4 Appointment
// (docs/booking/SPEC.md "Booking rules", "Who may do what with an appointment"). Booked at once from a free
// slot of a listed clinic, with the patient's active consent; cancelled by the patient up to 24 hours before
// the start and by staff at any time; marked fulfilled or no-show by its practitioner once it has started.
// The slot's status changes in the same transaction as the appointment's. Who sees what is decided by
// access; every read and write is audited, refusals included.
package appointment

import (
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
)

// Status is Appointment.status, the four the product uses.
type Status string

// The statuses.
const (
	StatusBooked    Status = "booked"
	StatusCancelled Status = "cancelled"
	StatusFulfilled Status = "fulfilled"
	StatusNoShow    Status = "noshow"
)

// SystemCancellationReason is the code system of Appointment.cancelationReason.
const SystemCancellationReason = "http://terminology.hl7.org/CodeSystem/appointment-cancellation-reason"

// Reason is a cancellation reason: who cancelled, never why (no free text).
type Reason string

// The reasons.
const (
	ReasonPatient  Reason = "pat"  // the patient, also through the revocation of their consent
	ReasonProvider Reason = "prov" // staff
)

// Appointment is one booking.
type Appointment struct {
	Start              time.Time // Appointment.start
	End                time.Time // Appointment.end
	Created            time.Time // Appointment.created
	CancelledAt        time.Time // zero unless cancelled
	LastUpdated        time.Time // meta.lastUpdated
	ID                 string
	PatientID          string // the patient participant
	PractitionerRoleID string // the practitioner role participant
	OrganizationID     string // the clinic
	SlotID             string // Appointment.slot
	CancelledBy        string // the account that cancelled; empty for none or a job
	Status             Status
	Reason             Reason // Appointment.cancelationReason; empty unless cancelled
	VersionID          int32  // meta.versionId
}

// Name is the patient's name as an appointment shows it.
type Name struct {
	Given  string
	Family string
}

// Seen is an appointment as a viewer may see it: at Full everything; at Name (a past appointment after a
// revocation) only the times, the status and the participants. Patient is the patient's name, set only when
// the viewer may see it (and its read was audited). PractitionerID is the practitioner of the appointment's
// role, a participant beside the role.
type Seen struct {
	Patient        Name
	PractitionerID string
	Appointment    Appointment
	Level          access.Level
}

// Filter narrows a search: the start in [From, To) (zero for open), and the patient, the practitioner (or
// their role) and the status when set.
type Filter struct {
	From               time.Time
	To                 time.Time
	PatientID          string
	PractitionerID     string // the practitioner participant (FHIR practitioner)
	PractitionerRoleID string
	Status             Status
	OrganizationID     string // set by the service, never by a caller
}

// Notice is what an e-mail job needs: the appointment and the account of its patient.
type Notice struct {
	AccountID   string
	Appointment Appointment
}

// Caller is who asks: the signed-in account and, for staff, its clinic, whether they are its admin and the
// practitioner they are; for a patient, their profile's id, resolved by the composition root (empty before
// the profile exists). A caller without a clinic is a patient.
type Caller struct {
	AccountID      string
	OrganizationID string // the caller's clinic; empty for a patient
	PatientID      string // the calling patient's profile
	PractitionerID string // the practitioner a staff account is; empty for an admin
	Admin          bool
}

func (c Caller) patient() bool { return c.OrganizationID == "" }

// BookedEmailArgs is the appointment_booked_email job on the notifier's email queue. Opaque ids only.
type BookedEmailArgs struct {
	AppointmentID  string `json:"appointment_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (BookedEmailArgs) Kind() string { return "appointment_booked_email" }

// CancelledEmailArgs is the appointment_cancelled_email job. Opaque ids only.
type CancelledEmailArgs struct {
	AppointmentID  string `json:"appointment_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (CancelledEmailArgs) Kind() string { return "appointment_cancelled_email" }

// ReminderEmailArgs is the appointment_reminder_email job, scheduled 24 hours before the start. Opaque ids
// only.
type ReminderEmailArgs struct {
	AppointmentID  string `json:"appointment_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (ReminderEmailArgs) Kind() string { return "appointment_reminder_email" }
