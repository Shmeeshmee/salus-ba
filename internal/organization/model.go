// Package organization is the directory's clinic: FHIR R4 Organization (docs/accounts/SPEC.md "Directory
// model"). A clinic is its own row-level security scope: the service sets it on the transaction, and the
// database shows no other clinic.
package organization

import "time"

// Organization is a clinic. Its name is not personal data.
type Organization struct {
	LastUpdated time.Time // meta.lastUpdated
	ID          string
	Name        string
	VersionID   int32 // meta.versionId; an update names the version it changes
	Active      bool
}

// Caller is who asks: the signed-in account and, for staff, its clinic.
type Caller struct {
	AccountID      string
	OrganizationID string // the caller's clinic; empty for a patient
	Admin          bool   // the caller is an admin of that clinic
}
