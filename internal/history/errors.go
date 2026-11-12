package history

import "errors"

// The errors of the history. Their text carries no id or personal data (ErrCursor is in model.go).
var (
	// ErrForbidden is a staff account asking for a patient's own timeline.
	ErrForbidden = errors.New("history: forbidden")
	// ErrLimit is a page size outside 1 to MaxLimit.
	ErrLimit = errors.New("history: limit out of range")
)
