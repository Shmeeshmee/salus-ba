package clinicstats

import "errors"

// Domain errors.
var (
	// ErrForbidden: the caller may not read these figures (not staff of a clinic, or a practitioner asking for
	// another role's figures or, without a role, for diagnoses).
	ErrForbidden = errors.New("clinicstats: forbidden")
	// ErrInvalidQuery: a range that is missing, reversed or longer than ten years, or an unknown breakdown.
	ErrInvalidQuery = errors.New("clinicstats: invalid query")
	// ErrUnknownClinic: a refresh of a clinic the builds do not know.
	ErrUnknownClinic = errors.New("clinicstats: unknown clinic")
)
