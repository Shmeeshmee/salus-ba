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
	AccountID   string // the patient's account; empty for a profile a clinic registered
	// RegisteredBy is the clinic that registered the profile of a patient without an account (docs/clinic-cms F2,
	// W1): only that clinic sees it. Empty for a patient's own profile.
	RegisteredBy string
	GivenName    string
	FamilyName   string
	Phone        string // E.164; empty for none
	VersionID    int32  // meta.versionId
	// ConsentActive is read for the clinic's staff, never stored: the patient has an active care consent with the
	// clinic whose scope read the profile (the desk's search, a registration and a correction, docs/clinic-cms wave 4).
	ConsentActive bool
}

// Restriction is the restriction of a patient's record at one clinic (docs/clinic-cms/WAVE5.md S6, S7), as its admin
// reads it: since when, and whether the erasure made it (which the clinic cannot lift) rather than the clinic's staff.
// A zero At is a record that is not restricted.
type Restriction struct {
	At        time.Time
	ByErasure bool
}

// Restricted reports whether the record is restricted.
func (r Restriction) Restricted() bool { return !r.At.IsZero() }

// Caller is who asks: the signed-in account and, for staff, its clinic, whether they are its admin and a
// practitioner's role there (set by the composition root where a route needs staff access). A caller without
// a clinic is a patient (an account has a clinic exactly when it is staff).
type Caller struct {
	AccountID          string
	OrganizationID     string // the caller's clinic; empty for a patient
	PractitionerRoleID string // a practitioner's role in that clinic
	Admin              bool   // the clinic's admin
	Reception          bool   // the clinic's receptionist (docs/clinic-cms F1)
}

func (c Caller) patient() bool { return c.OrganizationID == "" }
