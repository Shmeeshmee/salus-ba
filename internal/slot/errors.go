package slot

import "errors"

// The errors of the slot domain.
var (
	// ErrUnavailable is returned when a slot to hold is not free, does not exist or is not in the clinic.
	ErrUnavailable = errors.New("slot: not available")
)
