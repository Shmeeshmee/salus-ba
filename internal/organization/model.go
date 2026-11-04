// Package organization is the directory's clinic: FHIR R4 Organization (docs/accounts/SPEC.md "Directory
// model"), with whether it is listed in the public directory and its time zone (docs/booking/SPEC.md
// "Directory"). A clinic is its own row-level security scope: the service sets it on the transaction, and the
// database shows no other clinic; the public directory shows the listed clinics.
package organization

import "time"

// Organization is a clinic. Nothing in it is personal data.
type Organization struct {
	LastUpdated time.Time // meta.lastUpdated
	ID          string
	Name        string
	TimeZone    string // IANA, e.g. Europe/Sarajevo: appointment times are shown in it
	VersionID   int32  // meta.versionId; an update names the version it changes
	Active      bool
	Listed      bool // anyone may find the clinic and its practitioners
}

// Changes are what an update sets: the name, and the listing and the time zone when given (nil keeps them).
type Changes struct {
	Listed   *bool
	TimeZone *string
	Name     string
}

// Caller is who asks: the signed-in account and, for staff, its clinic.
type Caller struct {
	AccountID      string
	OrganizationID string // the caller's clinic; empty for a patient
	Admin          bool   // the caller is an admin of that clinic
}
