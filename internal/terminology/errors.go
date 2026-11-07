package terminology

import "errors"

// The errors of the terminology domain. Their text never carries the search text.
var (
	// ErrForbidden is returned when the caller is not staff of a clinic (a patient): the API's forbidden.
	ErrForbidden = errors.New("terminology: forbidden")
	// ErrInvalidSearch is returned for a search text that is blank or longer than 100 characters, an unknown
	// language or category, or a limit outside 1 to 25: the API's invalid_request.
	ErrInvalidSearch = errors.New("terminology: invalid search")
)
