// Package practitioner is the directory's practitioners: FHIR R4 Practitioner (docs/accounts/SPEC.md
// "Directory model"). A practitioner belongs to one clinic; the service scopes every query to the caller's
// clinic and row-level security shows no other. The directory knows nothing about accounts.
package practitioner

import "time"

// Practitioner is a person who gives care in one clinic: one HumanName (given, family) and whether they are
// active.
type Practitioner struct {
	LastUpdated    time.Time // meta.lastUpdated
	ID             string
	OrganizationID string // the clinic
	GivenName      string
	FamilyName     string
	VersionID      int32 // meta.versionId
	Active         bool
}

// Caller is who asks: the signed-in account and, for staff, its clinic.
type Caller struct {
	AccountID      string
	OrganizationID string // the caller's clinic; empty for a patient
}
