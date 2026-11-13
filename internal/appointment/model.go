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
	ChangeStatusSet       ChangeKind = "status_set"       // booked to fulfilled, noshow or cancelled, by whoever did it
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

// Token is a stored one-time link: only the SHA-256 of its token. StartAt is the appointment's start it was issued
// for, and when it expires: a link whose StartAt is not the appointment's start now is refused (SAL-34), so a
// reschedule leaves the earlier reminders' links useless. A link is used once, by a conditional UPDATE (WHERE used_at
// IS NULL) in the transaction that confirms or cancels; the database refuses any change to a used one.
type Token struct {
	CreatedAt time.Time
	StartAt   time.Time
	// UsedAt, ID and UsedAction are read back from the store (SAL-34): when and for what it was used (zero and empty
	// while unused), and its id.
	UsedAt         time.Time
	AppointmentID  string
	OrganizationID string
	PatientID      string
	AccountID      string
	ReminderID     string
	Language       string
	ID             string
	Channel        Channel
	UsedAction     LinkAction
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
	// Via is how a patient acts when not in their app: ViaLink for a reminder's one-time link (SAL-34), whose
	// changes are recorded and audited as the link's; empty for the app, or staff.
	Via   Via
	Admin bool
}

func (c Caller) patient() bool { return c.OrganizationID == "" }

// via is how c's change is recorded in the history: the call's own Via, else the patient's app or the staff's.
func (c Caller) via() Via {
	switch {
	case c.Via != "":
		return c.Via
	case c.patient():
		return ViaApp
	}
	return ViaStaff
}

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
// with whom, its status and how it was booked. No slot, no cancellation details.
type Booked struct {
	Start              time.Time
	PatientID          string
	PractitionerRoleID string
	Status             Status
	BookedVia          BookedVia
}

// Start is an appointment as a build gets it: the patient only as a pseudonym under the build's key
// (docs/plans/analytics.md §3).
type Start struct {
	Start              time.Time
	PractitionerRoleID string
	Status             Status
	BookedVia          BookedVia // how it was booked (SAL-36: the slots refilled from the waitlist)
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

// SAL-33: reminders (docs/plans/appointments.md, ADR 0015 "Reminders").

// ReminderState is where a reminder is: scheduled until its dispatch job runs; dispatched once its deliveries are
// stored; cancelled with its appointment (a cancellation or a reschedule); skipped when its job found the appointment
// no longer as it was scheduled for.
type ReminderState string

// The reminder states (appointment_reminder.state).
const (
	ReminderScheduled  ReminderState = "scheduled"
	ReminderDispatched ReminderState = "dispatched"
	ReminderCancelled  ReminderState = "cancelled"
	ReminderSkipped    ReminderState = "skipped"
)

// Reminder is one reminder of an appointment: due its offset before the start it was scheduled for (StartAt), an
// absolute instant whatever the zones.
type Reminder struct {
	StartAt        time.Time
	DueAt          time.Time
	DispatchedAt   time.Time
	ID             string
	AppointmentID  string
	OrganizationID string
	PatientID      string
	State          ReminderState
	OffsetMinutes  int
}

// ReminderNotice is what the notifier needs of a reminder to send it: the appointment's start and whose it is (the
// patient and their account). Nothing about the practitioner, the specialty or the visit.
type ReminderNotice struct {
	Start          time.Time
	ReminderID     string
	AppointmentID  string
	OrganizationID string
	PatientID      string
	AccountID      string
}

// ReminderArgs is the appointment_reminder job on the notifier's push queue, scheduled at the reminder's due time and
// unique by it: the dispatch. Opaque ids only.
type ReminderArgs struct {
	ReminderID     string `json:"reminder_id" river:"unique"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (ReminderArgs) Kind() string { return "appointment_reminder" }

// RemindersSweepArgs is the appointment_reminders_sweep job, hourly on the default queue: it enqueues again the
// dispatch of every reminder still scheduled past its due time while its start is ahead (a lost job).
type RemindersSweepArgs struct{}

// Kind names the job in river_job.
func (RemindersSweepArgs) Kind() string { return "appointment_reminders_sweep" }

// RescheduledEmailArgs is the appointment_rescheduled_email job on the notifier's email queue (SAL-34): the patient is
// told the appointment's new time. Opaque ids only.
type RescheduledEmailArgs struct {
	AppointmentID  string `json:"appointment_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (RescheduledEmailArgs) Kind() string { return "appointment_rescheduled_email" }

// LinkAction is what a one-time link was used for (appointment_action_token.used_action).
type LinkAction string

// The actions of a one-time link.
const (
	LinkConfirm LinkAction = "confirm"
	LinkCancel  LinkAction = "cancel"
)

// Move is a reschedule: the new slot and its period, and the reschedule count it was decided on (a concurrent
// reschedule that committed first leaves the appointment at another count, and this one changes nothing).
type Move struct {
	Start  time.Time
	End    time.Time
	SlotID string
	From   int32
}

// Actions is what the patient may do with their appointment now (GET /appointments/{id}/actions): confirm while
// unconfirmed, cancel until 24 hours before the start, reschedule within the clinic's notice and limit. The times are
// the last moments each is possible; zero when it is not.
type Actions struct {
	CancelUntil     time.Time
	RescheduleUntil time.Time
	ReschedulesLeft int
	Confirm         bool
	Cancel          bool
	Reschedule      bool
}

// LinkPage is what the page of a one-time link shows (ADR 0015 "One-tap links"): the clinic, the start in the
// clinic's zone and the possible actions; for a used link only its outcome, with the start it was issued for. Never a
// name, practitioner or specialty.
type LinkPage struct {
	Start    time.Time // in the clinic's zone
	Clinic   string
	TimeZone string // the clinic's IANA zone
	Language string // the language the link was issued in
	// Used is what the link was used for; empty while unused.
	Used LinkAction
	// Confirmed: the patient has confirmed it (in the app or a link). CanConfirm and CanCancel are the actions the
	// page offers; CancelClosed says the 24 hours before the start have begun (contact the clinic).
	Confirmed    bool
	CanConfirm   bool
	CanCancel    bool
	CancelClosed bool
}

// SAL-35: the waitlist (docs/plans/appointments.md, ADR 0015 "The waitlist hold lives in Postgres").

// WaitlistStatus is a waitlist entry's: active until booked through an offer, withdrawn (by the patient, staff or
// the revocation of the care consent), or expired once its latest day is over.
type WaitlistStatus string

// The statuses of an entry.
const (
	WaitlistActive    WaitlistStatus = "active"
	WaitlistBooked    WaitlistStatus = "booked"
	WaitlistWithdrawn WaitlistStatus = "withdrawn"
	WaitlistExpired   WaitlistStatus = "expired" // an active entry past EndsAt; never stored
)

// OfferState is a waitlist offer's.
type OfferState string

// The states of an offer.
const (
	OfferOpen      OfferState = "open"
	OfferAccepted  OfferState = "accepted"
	OfferDeclined  OfferState = "declined"
	OfferExpired   OfferState = "expired"
	OfferWithdrawn OfferState = "withdrawn" // its entry was withdrawn, or the care consent revoked
)

// Times is a time of day in the clinic's zone, [Start, End) in minutes after midnight (End up to 1440); zero for any.
type Times struct {
	Start int
	End   int
}

// Any reports whether t is no time of day: every time matches.
func (t Times) Any() bool { return t.End == 0 }

// Offer is a freed slot held for one waitlist entry until ExpiresAt; Start and End are the slot's.
type Offer struct {
	Start              time.Time
	End                time.Time
	ExpiresAt          time.Time
	Created            time.Time
	ID                 string
	EntryID            string
	SlotID             string
	OrganizationID     string
	PatientID          string
	PractitionerRoleID string // the slot's
	AppointmentID      string // the appointment its acceptance booked
	State              OfferState
}

// WaitlistEntry is a patient's wish for a practitioner role or a specialty of a clinic (one of the two), between two
// dates of the clinic's zone (Earliest and Latest at midnight UTC: dates, not instants) and, optionally, a time of day.
// EndsAt is the end of Latest in the clinic's zone. Offer is its open offer, if any (ID empty for none).
type WaitlistEntry struct {
	Earliest           time.Time
	Latest             time.Time
	Created            time.Time
	EndsAt             time.Time
	ClosedAt           time.Time // zero while active
	Offer              Offer
	ID                 string
	PatientID          string
	OrganizationID     string
	PractitionerRoleID string
	Specialty          string // a code of practitionerrole.Specialties
	Status             WaitlistStatus
	Times              Times
	Priority           int // 0 to 100: staff move an entry up
}

// NewEntry is a patient's request to join a clinic's waitlist.
type NewEntry struct {
	Earliest           time.Time
	Latest             time.Time
	OrganizationID     string
	PractitionerRoleID string
	Specialty          string
	Times              Times
}

// ClinicEntry is an entry as the clinic's staff see it: with the patient's name where access.Profile gives it.
type ClinicEntry struct {
	Patient Name // zero where the care consent does not give it
	Entry   WaitlistEntry
	Named   bool
}

// WaitlistFilter narrows the clinic's waitlist: the entries a role could be offered (its own and its specialty's), a
// status (active when empty), and a page: Limit entries (a default when 0) after the first Offset.
type WaitlistFilter struct {
	PractitionerRoleID string
	Status             WaitlistStatus
	Limit              int
	Offset             int
}

// ClinicQuery is a page of a clinic's waitlist as the repository reads it: the filter with the role's specialty, and
// the instant that tells active entries from expired ones.
type ClinicQuery struct {
	Now                time.Time
	OrganizationID     string
	PractitionerRoleID string
	Specialty          string
	Status             WaitlistStatus
	Limit              int
	Offset             int
}

// Match is what a freed slot asks of an entry: its clinic, role and specialty, its day and times in the clinic's zone,
// its period, and the instant (an entry past its EndsAt matches nothing).
type Match struct {
	Start              time.Time
	End                time.Time
	Day                time.Time // the slot's date in the clinic's zone, at midnight UTC
	Now                time.Time
	OrganizationID     string
	SlotID             string
	PractitionerRoleID string
	Specialty          string   // the role's; empty for none
	Skip               []string // entries passed over already
	Times              Times    // the slot's, in the clinic's zone
}

// Reach is whether an offer reaches a patient now: Reachable when some channel of theirs carries it (e-mail on, push on
// with a device, SMS opted into at a clinic that offers it), Quiet while inside their quiet hours, until QuietUntil.
type Reach struct {
	QuietUntil time.Time
	Reachable  bool
	Quiet      bool
}

// OfferNotice is what the notifier tells a patient of an offer: the offer and the patient's account.
type OfferNotice struct {
	AccountID string
	Offer     Offer
}

// OfferExpireArgs is the waitlist_offer_expire job on the default queue, at the offer's expires_at: the slot goes to
// the next entry. Opaque ids only.
type OfferExpireArgs struct {
	OfferID        string `json:"offer_id"`
	OrganizationID string `json:"organization_id"`
}

// Kind names the job in river_job.
func (OfferExpireArgs) Kind() string { return "waitlist_offer_expire" }

// NoShowFlag is the repeat no-show flag beside an appointment, for staff only (SAL-36, Q7): its patient missed
// NoShows appointments at the clinic in the last WindowDays days. Computed on read, never stored, never shown to
// the patient and never a reason to refuse a booking.
type NoShowFlag struct {
	AppointmentID string
	NoShows       int
	WindowDays    int
}
