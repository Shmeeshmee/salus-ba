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
	// ErrInvalidReview is returned for a review panel's request the term list does not take: an unknown language
	// or status, a limit outside 1 to 100, an unreadable page token, labels or an explanation outside their
	// bounds, a decision other than approved or needs_changes, or a note over 500 characters.
	ErrInvalidReview = errors.New("terminology: invalid review request")
	// ErrVersionConflict is returned when the entry changed since the version the reviewer read.
	ErrVersionConflict = errors.New("terminology: the entry changed since it was read")
	// ErrNoteRequired is returned for needs_changes without a note.
	ErrNoteRequired = errors.New("terminology: needs changes takes a note")
	// ErrNoTranslation is returned for a review of a term's labels in a language it has none in.
	ErrNoTranslation = errors.New("terminology: no labels in the language")
	// ErrNoExplanation is returned for a review of an explanation that does not exist.
	ErrNoExplanation = errors.New("terminology: no explanation in the language")
	// ErrNoGroupName is returned for a review of a diagnosis group or language that has no name.
	ErrNoGroupName = errors.New("terminology: no such diagnosis group name")
	// ErrInvalidGroups is returned when the diagnosis groups do not form one grouping: a range of an unknown group
	// or of "other", reversed or with ends of different lengths, two ranges of one level that share a code, a
	// subgroup's range outside its group, a subgroup of a subgroup, or no group "other".
	ErrInvalidGroups = errors.New("terminology: invalid diagnosis groups")
)
