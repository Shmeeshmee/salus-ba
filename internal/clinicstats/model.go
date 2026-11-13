// Package clinicstats is a clinic's own statistics (SAL-20, docs/plans/analytics.md §4, ADR 0013): its
// appointments and confirmed diagnoses counted per local day and hour, practitioner role and specialty, and
// diagnosis group, subgroup and code. The counts are pre-computed by the clinicstats_refresh job (a clinic-day,
// enqueued by every appointment write and every signing with a diagnosis) and the clinicstats_nightly job (the
// last 35 days of every clinic), on the salus_stats role; the charts read only these counts. No patient
// identifier is stored: the sources are asked without a pseudonym key and drop the one their shared query reads.
//
// Who sees what (C1 to C3): only staff of the clinic; an admin sees everything of their clinic, a practitioner
// who is not an admin the clinic's appointment figures and the diagnoses of their own role only; every read is
// audited.
package clinicstats

import (
	"time"

	"github.com/riverqueue/river"
)

// QueueStats is the River queue of the clinic statistics' builds: cmd/worker works it two jobs at a time, so the
// builds hold at most two salus_stats connections (each waits on its clinic's lock, the nightly pass runs for minutes)
// and never take the default queue's workers.
const QueueStats = "clinicstats"

// RefreshArgs is the clinicstats_refresh job: recompute the clinic's local day that holds the instant Day (an
// appointment's start or a signature's time). Opaque ids and an instant only.
type RefreshArgs struct {
	Day            time.Time `json:"day"`
	OrganizationID string    `json:"organization_id"`
}

// Kind names the job in river_job.
func (RefreshArgs) Kind() string { return "clinicstats_refresh" }

// InsertOpts puts the job on QueueStats, whoever enqueues it.
func (RefreshArgs) InsertOpts() river.InsertOpts { return river.InsertOpts{Queue: QueueStats} }

// NightlyArgs is the clinicstats_nightly job: recompute the last 35 local days of every clinic.
type NightlyArgs struct{}

// Kind names the job in river_job.
func (NightlyArgs) Kind() string { return "clinicstats_nightly" }

// InsertOpts puts the job on QueueStats, as the scheduler enqueues it.
func (NightlyArgs) InsertOpts() river.InsertOpts { return river.InsertOpts{Queue: QueueStats} }

// AppointmentCount is one row of clinic_appointment_stat: how many appointments of a role start in a local hour of
// a day (midnight UTC of the local date) and have a status now. SpecialtyCode is the role's SNOMED CT code at
// refresh time, empty for none; BookedVia how they were booked (direct or waitlist, SAL-36).
type AppointmentCount struct {
	Day                time.Time
	PractitionerRoleID string
	SpecialtyCode      string
	Status             string
	BookedVia          string
	Hour               int32
	Count              int32
}

// DiagnosisCount is one row of clinic_diagnosis_stat: how many confirmed Conditions of a code a role recorded in
// a local hour of a day, with the code's diagnosis group and subgroup (empty for none).
type DiagnosisCount struct {
	Day                time.Time
	PractitionerRoleID string
	SpecialtyCode      string
	Group              string
	Subgroup           string
	Code               string
	Hour               int32
	Count              int32
}

// Caller is who asks: the signed-in account, its clinic (empty for a patient), whether it is the clinic's admin
// and whether it is a practitioner there, with the practitioner's role in the clinic (empty for none).
type Caller struct {
	AccountID          string
	OrganizationID     string
	PractitionerRoleID string
	Admin              bool
	Practitioner       bool
}

// AppointmentQuery narrows the appointment figures: the local days From to To (dates, inclusive), and the role
// and the specialty when set.
type AppointmentQuery struct {
	From               time.Time
	To                 time.Time
	PractitionerRoleID string
	Specialty          string
}

// Breakdown is what the diagnosis figures are counted by.
type Breakdown string

// The breakdowns.
const (
	ByGroup            Breakdown = "group"
	BySubgroup         Breakdown = "subgroup"
	ByCode             Breakdown = "code"
	ByPractitionerRole Breakdown = "practitionerRole"
	BySpecialty        Breakdown = "specialty"
	ByMonth            Breakdown = "month"
)

// DiagnosisQuery narrows the diagnosis figures like AppointmentQuery, and by group and subgroup when set; By is
// the breakdown.
type DiagnosisQuery struct {
	From               time.Time
	To                 time.Time
	Group              string
	Subgroup           string
	Specialty          string
	PractitionerRoleID string
	By                 Breakdown
}

// Period is a month's appointments (YYYY-MM): Booked counts every appointment starting in it, whatever its status
// now; Fulfilled, NoShow and Cancelled those with that status; Refilled those booked through the waitlist, whatever
// their status (SAL-36: the slots refilled).
type Period struct {
	Period    string
	Booked    int64
	Fulfilled int64
	NoShow    int64
	Cancelled int64
	Refilled  int64
}

// HeatCell is how many appointments that were not cancelled start on an ISO weekday (1 is Monday) in a local hour.
type HeatCell struct {
	Count   int64
	Weekday int32
	Hour    int32
}

// PractitionerTotal is a role's appointments: every one (Booked), and the fulfilled and no-show ones.
type PractitionerTotal struct {
	PractitionerRoleID string
	Fulfilled          int64
	NoShow             int64
	Booked             int64
}

// SpecialtyTotal is a specialty's appointments (its SNOMED CT code, empty for roles without one), as Period counts
// them.
type SpecialtyTotal struct {
	Specialty string
	Booked    int64
	Fulfilled int64
	NoShow    int64
	Cancelled int64
}

// AppointmentFigures are the appointment figures of a query. Practitioners and Specialties are set only when
// PerPractitioner, for an admin.
type AppointmentFigures struct {
	Series          []Period
	Heatmap         []HeatCell
	Practitioners   []PractitionerTotal
	Specialties     []SpecialtyTotal
	PerPractitioner bool
}

// Count is a diagnosis count by its breakdown's key: a group, subgroup or ICD-10 code, a role's id, a SNOMED CT
// specialty or a month (YYYY-MM); empty for no subgroup or no specialty.
type Count struct {
	Key   string
	Count int64
}
