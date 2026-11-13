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
	ConfirmedAt        time.Time // when the patient confirmed it; zero while unconfirmed (docs/plans/appointments.md)
	ID                 string
	PatientID          string // the patient participant
	PractitionerRoleID string // the practitioner role participant
	OrganizationID     string // the clinic
	SlotID             string // Appointment.slot
	CancelledBy        string // the account that cancelled; empty for none or a job
	Status             Status
	Reason             Reason    // Appointment.cancelationReason; empty unless cancelled
	ConfirmedVia       Via       // ViaApp or ViaLink once confirmed; empty while unconfirmed
	BookedVia          BookedVia // how it was booked: directly or through a waitlist offer
	VersionID          int32     // meta.versionId
	RescheduleCount    int32     // how often the patient moved it
}

// Via is how a change of an appointment was made: in the patient's app, through a one-time link of a reminder, by
// the clinic's staff, or by the system (a job, the consultation's end).
type Via string

// The ways.
const (
	ViaApp    Via = "app"
	ViaLink   Via = "link"
	ViaStaff  Via = "staff"
	ViaSystem Via = "system"
)

// BookedVia is how an appointment was booked.
type BookedVia string

// The ways an appointment is booked: a free slot of the directory, or an offer of the waitlist (SAL-35).
const (
	BookedDirect   BookedVia = "direct"
	BookedWaitlist BookedVia = "waitlist"
)

// ChangeKind is what an appointment_change records.
type ChangeKind string

// The kinds of change.
const (
	ChangeConfirmed       ChangeKind = "confirmed"        // the patient confirmed (SAL-34)
	ChangeRescheduled     ChangeKind = "rescheduled"      // the patient moved it to another slot of its role (SAL-34)
	ChangeStatusSet       ChangeKind = "status_set"       // booked to fulfilled or noshow, by the practitioner or the system
	ChangeStatusCorrected ChangeKind = "status_corrected" // noshow and fulfilled corrected into each other (SAL-36)
)

// Change is one row of an appointment's history: append-only, kept with the appointment. A reschedule names both
// periods and slots, a status change both statuses. ByAccountID is the account that made it (empty for the system,
// whose job is Agent); neither is shown to the patient.
type Change struct {
	At            time.Time
	FromStart     time.Time
	FromEnd       time.Time
	ToStart       time.Time
	ToEnd         time.Time
	ID            string
	AppointmentID string
	ByAccountID   string
	Agent         string
	FromSlotID    string
	ToSlotID      string
	Kind          ChangeKind
	Via           Via
	FromStatus    Status
	ToStatus      Status
}

// Channel is the message a one-time link was sent in.
type Channel string

// The channels that carry a link.
const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
)

// Link is a one-time link to issue for a reminder of an appointment (ADR 0015 "One-tap links"): the appointment and
// its clinic, the reminder and the channel it goes out in, and the language the page answers in.
type Link struct {
	AppointmentID  string
	OrganizationID string
	ReminderID     string // empty until SAL-33's reminders exist
	Language       string
	Channel        Channel
}

// Token is a stored one-time link: only the SHA-256 of its token.
type Token struct {
	CreatedAt      time.Time
	ExpiresAt      time.Time
	AppointmentID  string
	OrganizationID string
	PatientID      string
	AccountID      string
	ReminderID     string
	Language       string
	Channel        Channel
	Hash           [32]byte
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

// AccessEndArgs is the consultation_access_end job on the default queue, enqueued with every cancellation: the
// consultation (docs/consultation/SPEC.md §2 "Removal") removes the patient from the appointment's video room
// and ends its session. Opaque ids only.
type AccessEndArgs struct {
	AppointmentID  string `json:"appointment_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (AccessEndArgs) Kind() string { return "consultation_access_end" }

// QuestionnaireRequestArgs is the questionnaire_default_request job on the default queue, enqueued with every
// booking: the questionnaire responses (docs/questionnaire/SPEC.md §2 "Requests") request the default
// questionnaire of the appointment's role, if it has one. Opaque ids only.
type QuestionnaireRequestArgs struct {
	AppointmentID  string `json:"appointment_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (QuestionnaireRequestArgs) Kind() string { return "questionnaire_default_request" }

// QuestionnaireDiscardArgs is the questionnaire_responses_discard job on the default queue, enqueued with
// every cancellation: the questionnaire responses (docs/questionnaire/SPEC.md §2 "Cancellation and
// revocation") delete every response of the cancelled appointment. Opaque ids only.
type QuestionnaireDiscardArgs struct {
	AppointmentID  string `json:"appointment_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (QuestionnaireDiscardArgs) Kind() string { return "questionnaire_responses_discard" }

// ReminderEmailArgs is the appointment_reminder_email job, scheduled 24 hours before the start. Opaque ids
// only.
type ReminderEmailArgs struct {
	AppointmentID  string `json:"appointment_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (ReminderEmailArgs) Kind() string { return "appointment_reminder_email" }

// Booked is an appointment as the aggregate builds read it from the store (salus_stats): when it starts, whose,
// with whom, and its status. No slot, no cancellation details.
type Booked struct {
	Start              time.Time
	PatientID          string
	PractitionerRoleID string
	Status             Status
}

// Start is an appointment as a build gets it: the patient only as a pseudonym under the build's key
// (docs/plans/analytics.md §3).
type Start struct {
	Start              time.Time
	PractitionerRoleID string
	Status             Status
	Patient            [32]byte
}

// SlotFreedArgs is the waitlist_slot_freed job on the default queue, enqueued with every cancellation that frees a
// slot still ahead (and every reschedule, SAL-34): the waitlist offers it (SAL-35). Opaque ids only.
type SlotFreedArgs struct {
	SlotID         string `json:"slot_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (SlotFreedArgs) Kind() string { return "waitlist_slot_freed" }
