package encounter

import "errors"

// The errors of the encounter domain. Their text carries no id or personal data.
var (
	// ErrNotFound is returned for an encounter, or an appointment, that does not exist or that the caller
	// may not see: its existence is not disclosed.
	ErrNotFound = errors.New("encounter: not found")
	// ErrForbidden is returned when a practitioner searches for another practitioner's encounters.
	ErrForbidden = errors.New("encounter: forbidden")
)
