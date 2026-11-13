// Package organization is the directory's clinic: FHIR R4 Organization (docs/accounts/SPEC.md "Directory
// model"), with whether it is listed in the public directory and its time zone (docs/booking/SPEC.md
// "Directory"). A clinic is its own row-level security scope: the service sets it on the transaction, and the
// database shows no other clinic; the public directory shows the listed clinics.
package organization

import "time"

// Organization is a clinic. Nothing in it is personal data.
type Organization struct {
	LastUpdated time.Time // meta.lastUpdated
	// LocationSince is when Region and City were last set (zero without a location), ContributesSince when
	// AnalyticsContributes last changed: a clinic counts in a period of the platform figures only when neither
	// changed during it (docs/plans/analytics.md §5 "Which clinics count").
	LocationSince    time.Time
	ContributesSince time.Time
	ID               string
	Name             string
	TimeZone         string // IANA, e.g. Europe/Sarajevo: appointment times are shown in it
	// DefaultLanguage is a staff language (docs/i18n/SPEC.md §3 "The clinic's default language"), bs unless
	// an admin changes it: for e-mails to staff with no language of their own, and new questionnaires.
	DefaultLanguage string
	// Region and City are the clinic's location (docs/plans/analytics.md O1): a leaf region (an ISO 3166-2:BA
	// code) and a city under it (a Wikidata id), both of terminology's places, or both empty. CityName is the
	// city's name in Latin script, filled by Read for Organization.address.city.
	Region    string
	City      string
	CityName  string
	VersionID int32 // meta.versionId; an update names the version it changes
	Active    bool
	Listed    bool // anyone may find the clinic and its practitioners
	// PatientTranscriptCopy: the clinic gives its patients a copy of their consultations' stored transcripts in
	// the app (docs/consultation/SPEC.md "Patient documents"); off unless its admin turns it on.
	PatientTranscriptCopy bool
	// AnalyticsContributes: the clinic's figures count in the platform figures and its staff see them (O6); on
	// unless its admin opts the clinic out.
	AnalyticsContributes bool
}

// Changes are what an update sets: the name, and the listing, the time zone, the default language, the
// patients' transcript copy, the location (Region and City together, both empty to clear it) and the
// contribution to the platform figures when given (nil keeps them).
type Changes struct {
	Listed                *bool
	TimeZone              *string
	DefaultLanguage       *string
	PatientTranscriptCopy *bool
	Region                *string
	City                  *string
	AnalyticsContributes  *bool
	Name                  string
}

// Caller is who asks: the signed-in account and, for staff, its clinic.
type Caller struct {
	AccountID      string
	OrganizationID string // the caller's clinic; empty for a patient
	// PatientID is a patient's profile, read through the patient service; empty for staff and before the profile
	// exists. Only the chat hours read it.
	PatientID string
	Admin     bool // the caller is an admin of that clinic
}

// ChatHour is one stretch of a weekday in which the clinic answers chat messages, in its time zone: Day is the
// FHIR days-of-week code (mon … sun), Start and End are HH:MM, End after Start (24:00 for midnight).
type ChatHour struct {
	Day   string `json:"day"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// ChatSettings are the clinic's chat settings (docs/plans/chat.md, SAL-29): its chat hours, the response time it
// tells patients to expect, and how long a message stays unread before its reader gets an e-mail. Nothing in them
// is personal data.
type ChatSettings struct {
	Hours             []ChatHour `json:"hours"`
	ResponseTimeHours int        `json:"response_time_hours"`
	EmailAfterMinutes int        `json:"email_after_minutes"`
}

// ChatHours is what the clinic's patients and staff see of the settings: the hours in the clinic's time zone, the
// response time, and whether the clinic is within its hours now (the app's out-of-hours notice).
type ChatHours struct {
	TimeZone          string     `json:"time_zone"`
	Hours             []ChatHour `json:"hours"`
	ResponseTimeHours int        `json:"response_time_hours"`
	OpenNow           bool       `json:"open_now"`
}

// Location is a clinic as the aggregate builds need it (docs/plans/analytics.md §3): its id and time zone (the
// local day and hour of its figures), its region and city (empty without a location) and whether it contributes
// to the platform figures, each with the time it last changed (§5). No name.
type Location struct {
	LocationSince    time.Time
	ContributesSince time.Time
	ID               string
	TimeZone         string
	Region           string
	City             string
	Contributes      bool
}

// AppointmentSettings are the clinic's appointment rules (docs/plans/appointments.md, ADR 0015 "Clinic rules"): when
// reminders go out (minutes before the start, at most three, 15 minutes to 7 days), whether the paid SMS option is
// on, how late and how often a patient may reschedule (0 turns self-service rescheduling off), whether freed slots
// go to the waitlist and how long an offer is held, and from how many no-shows in how many days staff see a flag (0
// for off). Version is the stored row's (0 while the clinic has stored none): a save names the version it read, sent
// as the ETag, not in the JSON. Nothing in them is personal data.
type AppointmentSettings struct {
	ReminderOffsetsMinutes []int `json:"reminder_offsets_minutes"`
	Version                int   `json:"-"`
	RescheduleNoticeHours  int   `json:"reschedule_notice_hours"`
	RescheduleMax          int   `json:"reschedule_max"`
	WaitlistHoldMinutes    int   `json:"waitlist_hold_minutes"`
	NoShowFlagThreshold    int   `json:"noshow_flag_threshold"`
	NoShowFlagWindowDays   int   `json:"noshow_flag_window_days"`
	SMSReminders           bool  `json:"sms_reminders"`
	WaitlistEnabled        bool  `json:"waitlist_enabled"`
}
