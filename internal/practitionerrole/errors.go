package practitionerrole

import "errors"

// The errors of the practitionerrole domain.
var (
	// ErrNotFound is returned for a role, or a clinic's roles, that does not exist or that the caller may not
	// see: its existence is not disclosed.
	ErrNotFound = errors.New("practitionerrole: not found")
	// ErrForbidden is returned when a caller without a clinic tries to add a role.
	ErrForbidden = errors.New("practitionerrole: forbidden")
)
