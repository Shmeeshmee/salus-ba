// Package schedule is a practitioner role's weekly availability: FHIR R4 Schedule (docs/booking/SPEC.md
// "Schedule shape"). One per role, created with the role's first availability edit: rules {day, start, end}
// in the clinic's time zone and a slot length. Slots are generated from it by internal/slot. Staff of the
// clinic see and change it through the service (the practitioner their own role's, the admin any role's);
// the public directory reads the schedules of listed clinics.
package schedule

import "time"

// Schedule is a role's availability.
type Schedule struct {
	LastUpdated        time.Time // meta.lastUpdated
	ID                 string    // empty before the role's first availability edit
	OrganizationID     string
	PractitionerRoleID string // Schedule.actor
	Rules              []Rule // sorted by day and start
	SlotMinutes        int32  // 5 to 240, a multiple of 5; zero before the first edit
	VersionID          int32  // meta.versionId; zero before the first edit
}

// Rule is one weekly window of availability: an ISO weekday (1 Monday to 7 Sunday) and a start and end
// time of day, HH:MM in the clinic's time zone, start before end.
type Rule struct {
	Start string
	End   string
	Day   int
}

// Availability is what an edit sets: the slot length and the rules.
type Availability struct {
	Rules       []Rule
	SlotMinutes int32
}

// Caller is who asks: the signed-in account and, for staff, its clinic, whether they are its admin and the
// practitioner they are (empty for an admin who is not one).
type Caller struct {
	AccountID      string
	OrganizationID string // empty for a patient
	PractitionerID string
	Admin          bool
}

// SlotsGenerateArgs is the slots_generate job on the default queue: generate the slots of one schedule, or
// of every schedule of a listed clinic when both ids are empty (the scheduler's hourly run). Opaque ids only.
type SlotsGenerateArgs struct {
	ScheduleID     string `json:"schedule_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
}

// Kind names the job in river_job.
func (SlotsGenerateArgs) Kind() string { return "slots_generate" }
