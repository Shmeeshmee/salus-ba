// Package patient is the patients' profiles: FHIR R4 Patient (docs/booking/SPEC.md "Patient", "Profile").
// One profile per patient account, not clinic-scoped, created by the patient's first save with every
// required field. Row-level security shows a patient only their own profile.
package patient

import "time"

// Patient is a patient's profile: one HumanName (use official), the birth date and an optional mobile phone.
// Nothing else is kept (SPEC.md section 4).
type Patient struct {
	BirthDate   time.Time // a date: midnight UTC
	LastUpdated time.Time // meta.lastUpdated
	ID          string
	AccountID   string // the patient's account
	GivenName   string
	FamilyName  string
	Phone       string // E.164; empty for none
	VersionID   int32  // meta.versionId
}

// Caller is who asks: the signed-in account and, for staff, its clinic, whether they are its admin and a
// practitioner's role there (set by the composition root where a route needs staff access). A caller without
// a clinic is a patient (an account has a clinic exactly when it is staff).
type Caller struct {
	AccountID          string
	OrganizationID     string // the caller's clinic; empty for a patient
	PractitionerRoleID string // a practitioner's role in that clinic
	Admin              bool   // the clinic's admin
}

func (c Caller) patient() bool { return c.OrganizationID == "" }
