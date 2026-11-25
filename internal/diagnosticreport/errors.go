package diagnosticreport

import "errors"

// The errors of a report. Their text carries no personal data.
var (
	// ErrInvalid is returned for a report without a title or values that do not fit (a unit without a number, both a
	// number and a text, an unknown flag), or a file that is not a PDF.
	ErrInvalid = errors.New("diagnosticreport: invalid")
	// ErrNotFound is returned for a report or visit the caller may not see.
	ErrNotFound = errors.New("diagnosticreport: not found")
	// ErrForbidden is returned when anyone but a practitioner of the clinic releases a report.
	ErrForbidden = errors.New("diagnosticreport: forbidden")
	// ErrReleased is returned for a change to a released report.
	ErrReleased = errors.New("diagnosticreport: released")
	// ErrVersionConflict is returned when the report changed since the caller read it.
	ErrVersionConflict = errors.New("diagnosticreport: version conflict")
)
