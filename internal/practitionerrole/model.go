// Package practitionerrole is a practitioner's membership of a clinic: FHIR R4 PractitionerRole
// (docs/accounts/SPEC.md "Directory model") with its specialty (docs/booking/SPEC.md "Specialty"). A
// practitioner has one role, in their clinic. The service scopes every staff query to the caller's clinic and
// row-level security shows no other; the public directory shows the active roles of listed clinics.
package practitionerrole

import "time"

// SystemSNOMED is the code system of every specialty.
const SystemSNOMED = "http://snomed.info/sct"

// PractitionerRole links a practitioner to their clinic.
type PractitionerRole struct {
	LastUpdated    time.Time // meta.lastUpdated
	ID             string
	OrganizationID string // PractitionerRole.organization
	PractitionerID string // PractitionerRole.practitioner
	SpecialtyCode  string // PractitionerRole.specialty, one of Specialties; empty for none
	VersionID      int32  // meta.versionId
	Active         bool
}

// Caller is who asks: the signed-in account and, for staff, its clinic.
type Caller struct {
	AccountID      string
	OrganizationID string // the caller's clinic; empty for a patient
	Admin          bool   // the caller is an admin of that clinic
}

// Specialty is a SNOMED CT concept, with its display, and
// Text, the owner's name for it (CodeableConcept.text), which the apps show.
type Specialty struct {
	Code    string
	Display string
	Text    string
}

// Specialties returns the specialties an admin may give a role: the owner's twelve, each code and display
// copied from c80-practice-codes in hl7.fhir.r4.core 4.0.1 (docs/booking/PROGRESS.md, package 4). That value
// set has no paediatrics concept, so paediatrics is SNOMED CT 394537008 "Pediatric specialty", which the
// preferred binding on PractitionerRole.specialty allows (docs/booking/SPEC.md §2).
func Specialties() []Specialty {
	return []Specialty{
		{Code: "394814009", Display: "General practice", Text: "General practice"},
		{Code: "419192003", Display: "Internal medicine", Text: "Internal medicine"},
		{Code: "394579002", Display: "Cardiology", Text: "Cardiology"},
		{Code: "394582007", Display: "Dermatology", Text: "Dermatology"},
		{Code: "394586005", Display: "Gynecology", Text: "Gynaecology"},
		{Code: "394537008", Display: "Pediatric specialty", Text: "Paediatrics"},
		{Code: "394587001", Display: "Psychiatry", Text: "Psychiatry"},
		{Code: "394591006", Display: "Neurology", Text: "Neurology"},
		{Code: "418960008", Display: "Otolaryngology", Text: "ENT"},
		{Code: "394594003", Display: "Ophthalmology", Text: "Ophthalmology"},
		{Code: "394801008", Display: "Surgery-Trauma and orthopedics", Text: "Orthopaedics"},
		{Code: "394583002", Display: "Endocrinology", Text: "Endocrinology"},
	}
}

// SpecialtyByCode returns the specialty with code, and false when it is not one of Specialties.
func SpecialtyByCode(code string) (Specialty, bool) {
	for _, s := range Specialties() {
		if s.Code == code {
			return s, true
		}
	}
	return Specialty{}, false
}
