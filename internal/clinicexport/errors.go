package clinicexport

import "errors"

// The service's errors.
var (
	// ErrForbidden: the caller is not the clinic's admin.
	ErrForbidden = errors.New("clinicexport: forbidden")
	// ErrNotFound: no such export of the caller's clinic, or (for the download) not ready or expired.
	ErrNotFound = errors.New("clinicexport: not found")
)
