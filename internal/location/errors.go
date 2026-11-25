package location

import "errors"

// The errors of a location, room or device. Their text carries no personal data.
var (
	// ErrInvalid is returned for an empty name, an unknown time zone, city or a malformed phone.
	ErrInvalid = errors.New("location: invalid")
	// ErrNotFound is returned for a location, room or device the caller's clinic does not have.
	ErrNotFound = errors.New("location: not found")
	// ErrForbidden is returned when staff who are not the clinic's admin write.
	ErrForbidden = errors.New("location: forbidden")
	// ErrVersionConflict is returned when the row changed since the caller read it.
	ErrVersionConflict = errors.New("location: version conflict")
)
