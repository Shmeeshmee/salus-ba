package terminology

import "errors"

// The errors of the terminology domain. Their text never carries the search text.
var (
	// ErrForbidden is returned when the caller is not staff of a clinic (a patient): the API's forbidden.
	ErrForbidden = errors.New("terminology: forbidden")
	// ErrInvalidSearch is returned for a search text that is blank or longer than 100 characters, an unknown
	// language or category, or a limit outside 1 to 25: the API's invalid_request.
	ErrInvalidSearch = errors.New("terminology: invalid search")
	// ErrTermNotFound is returned for an id that names no term.
	ErrTermNotFound = errors.New("terminology: no such term")
	// ErrDuplicateWikidata is returned when a term would take a Wikidata id another term has.
	ErrDuplicateWikidata = errors.New("terminology: the Wikidata id belongs to another term")
	// ErrInvalidStatus is returned for a review status that is not unreviewed, approved or needs_changes.
	ErrInvalidStatus = errors.New("terminology: invalid review status")
)
