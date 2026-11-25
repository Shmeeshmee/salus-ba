// Package careplan is the treatment plans a practitioner or admin lays out for a patient (FHIR CarePlan and its
// activities; docs/clinic-cms/WAVE6.md U6). Health data (GDPR Art. 9): every read and write is audited.
package careplan

import "time"

// Status is a plan's CarePlan.status (http://hl7.org/fhir/request-status), the four a plan here takes.
type Status string

// A plan is drafted, booked (every step at once, or none), and ends completed or revoked (cancelled).
const (
	StatusDraft     Status = "draft"
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusRevoked   Status = "revoked"
)

// MaxDayOffset is the latest day a step may fall on, counted from the plan's start: two years.
const MaxDayOffset = 730

// Step is one activity of a plan: a service of the price list, its day counted from the plan's start, and the visit
// mode; the visit once booked.
type Step struct {
	// VisitStart and VisitEnd are the visit's times once booked, VisitStatus its status (in a response only; the
	// patient's app and the export show the visit's day, which a move changes).
	VisitStart    *time.Time `json:"visitStart,omitempty"`
	VisitEnd      *time.Time `json:"-"`
	ID            string     `json:"id,omitempty"` // in a response only
	ServiceID     string     `json:"serviceId"`
	ServiceName   string     `json:"serviceName,omitempty"` // the price list's name, in a response only
	VisitStatus   string     `json:"-"`
	VisitMode     string     `json:"visitMode"` // in-person or video, as appointment's
	AppointmentID string     `json:"appointmentId,omitempty"`
	Position      int        `json:"position,omitempty"` // in a response only
	DayOffset     int        `json:"dayOffset"`
}

// Plan is a treatment plan for a patient of the clinic. It carries its quote (X1).
type Plan struct {
	StartsOn       *time.Time `json:"startsOn,omitempty"` // day 0; set when booked
	BookedAt       *time.Time `json:"bookedAt,omitempty"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	PatientID      string     `json:"patientId"`
	Status         Status     `json:"status"`
	Title          string     `json:"title"`
	QuoteID        string     `json:"quoteId,omitempty"`
	Steps          []Step     `json:"steps"`
	VersionID      int        `json:"versionId"`
}

// LastUpdated is the plan's latest change: written, booked or cancelled (a booked plan's steps change only through
// its visits).
func (p Plan) LastUpdated() time.Time {
	at := p.CreatedAt
	for _, t := range []*time.Time{p.BookedAt, p.RevokedAt} {
		if t != nil && t.After(at) {
			at = *t
		}
	}
	return at
}

// Draft is what staff write while a plan is a draft.
type Draft struct {
	PatientID string
	Title     string
	QuoteID   string
	Steps     []Step
	Version   int // 0 when writing a new plan
}

// Booking books every step of a plan: its start, and for each step the slot of its visit, in the steps' order.
type Booking struct {
	StartsOn time.Time
	SlotIDs  []string
	Version  int
}

// Staff is the caller of the clinic's routes: its admin or a practitioner; a receptionist is refused (no clinical
// content).
type Staff struct {
	AccountID      string
	OrganizationID string
	PractitionerID string // the practitioner the account is; empty for an admin
	Admin          bool
	Practitioner   bool
	Reception      bool
}

// Patient is the caller of the patient's routes.
type Patient struct {
	AccountID string
	PatientID string
}
