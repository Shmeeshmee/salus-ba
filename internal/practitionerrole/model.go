// Package practitionerrole is a practitioner's membership of a clinic: FHIR R4 PractitionerRole
// (docs/accounts/SPEC.md "Directory model"). A practitioner has one role, in their clinic; no code or
// specialty yet. The service scopes every query to the caller's clinic and row-level security shows no
// other.
package practitionerrole

import "time"

// PractitionerRole links a practitioner to their clinic.
type PractitionerRole struct {
	LastUpdated    time.Time // meta.lastUpdated
	ID             string
	OrganizationID string // PractitionerRole.organization
	PractitionerID string // PractitionerRole.practitioner
	VersionID      int32  // meta.versionId
	Active         bool
}

// Caller is who asks: the signed-in account and, for staff, its clinic.
type Caller struct {
	AccountID      string
	OrganizationID string // the caller's clinic; empty for a patient
}
