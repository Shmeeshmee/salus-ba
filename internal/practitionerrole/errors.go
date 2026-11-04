package practitionerrole

import "errors"

// The errors of the practitionerrole domain.
var (
	// ErrNotFound is returned for a role, or a clinic's roles, that does not exist or that the caller may not
	// see: its existence is not disclosed.
	ErrNotFound = errors.New("practitionerrole: not found")
	// ErrForbidden is returned when a caller without a clinic tries to add a role, or a member of the clinic
	// who is not its admin tries to change one.
	ErrForbidden = errors.New("practitionerrole: forbidden")
	// ErrConflict is returned for an update of a version that is not the current one.
	ErrConflict = errors.New("practitionerrole: version conflict")
	// ErrInvalidSpecialty is returned for a code that is not one of Specialties.
	ErrInvalidSpecialty = errors.New("practitionerrole: invalid specialty")
)
