package schedule

import "errors"

// The errors of the schedule domain.
var (
	// ErrNotFound is returned for a role or schedule that does not exist or that the caller may not see.
	ErrNotFound = errors.New("schedule: not found")
	// ErrForbidden is returned when a practitioner of the clinic asks for another practitioner's role.
	ErrForbidden = errors.New("schedule: forbidden")
	// ErrConflict is returned for an edit of a version that is not the current one.
	ErrConflict = errors.New("schedule: version conflict")
	// ErrInvalidSlotMinutes is returned for a slot length outside 5 to 240 or not a multiple of 5.
	ErrInvalidSlotMinutes = errors.New("schedule: invalid slot length")
	// ErrInvalidAvailability is returned for a rule with a day outside 1 to 7, a time that is not HH:MM, an
	// end not after its start, or rules that overlap on a day.
	ErrInvalidAvailability = errors.New("schedule: invalid availability")
)
