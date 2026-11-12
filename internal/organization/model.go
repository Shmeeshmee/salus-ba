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
	ID          string
	Name        string
	TimeZone    string // IANA, e.g. Europe/Sarajevo: appointment times are shown in it
	// DefaultLanguage is a staff language (docs/i18n/SPEC.md §3 "The clinic's default language"), bs unless
	// an admin changes it: for e-mails to staff with no language of their own, and new questionnaires.
	DefaultLanguage string
	VersionID       int32 // meta.versionId; an update names the version it changes
	Active          bool
	Listed          bool // anyone may find the clinic and its practitioners
	// Region and City are the clinic's location (docs/plans/analytics.md O1): a leaf region (an ISO 3166-2:BA
	// code) and a city under it (a Wikidata id), both of terminology's places, or both empty. CityName is the
	// city's name in Latin script, filled by Read for Organization.address.city.
	Region   string
	City     string
	CityName string
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
	Admin          bool   // the caller is an admin of that clinic
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
