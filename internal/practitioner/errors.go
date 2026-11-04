package practitioner

import "errors"

// The errors of the practitioner domain. Their text carries no name.
var (
	// ErrNotFound is returned for a practitioner that does not exist or that the caller may not see: its
	// existence is not disclosed.
	ErrNotFound = errors.New("practitioner: not found")
	// ErrForbidden is returned when a caller without a clinic tries to add a practitioner.
	ErrForbidden = errors.New("practitioner: forbidden")
)
