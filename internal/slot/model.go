// Package slot is the bookable time of a schedule: FHIR R4 Slot (docs/booking/SPEC.md "Slot generation").
// Slots are generated from a schedule's rules 28 days ahead in the clinic's time zone, free until an
// appointment holds one (busy) in the appointment's own transaction. Staff of the clinic see its slots; the
// public directory sees the free slots of listed clinics.
package slot

import "time"

// Status is FHIR SlotStatus, the two the product uses.
type Status string

// The statuses.
const (
	StatusFree Status = "free"
	StatusBusy Status = "busy"
)

// Period is a slot's start and end, half-open [Start, End).
type Period struct {
	Start time.Time
	End   time.Time
}

// Slot is a bookable time of one schedule.
type Slot struct {
	Period
	LastUpdated    time.Time // meta.lastUpdated
	ID             string
	ScheduleID     string
	OrganizationID string
	Status         Status
	VersionID      int32 // meta.versionId
}

// Bookable is a free slot of the directory with the practitioner role of its schedule, which a booking is
// made with.
type Bookable struct {
	PractitionerRoleID string
	Slot
}

// Generated is what a generation did: the schedules it went through and the slots it added and removed.
type Generated struct {
	Schedules int
	Inserted  int64
	Deleted   int64
}
