package trends

import "errors"

// The errors of the trends domain. Their text carries no count, code or id.
var (
	// ErrForbidden: the caller is not staff of a clinic.
	ErrForbidden = errors.New("trends: forbidden")
	// ErrNotContributing: the caller's clinic does not contribute to the platform figures, so its staff do not see
	// them (docs/plans/analytics.md O6).
	ErrNotContributing = errors.New("trends: the clinic does not contribute")
	// ErrInvalidQuery: a query outside the tables' combinations (§5, §8).
	ErrInvalidQuery = errors.New("trends: invalid query")
	// ErrInvalidPeriod: a period that is not a month, a quarter or an ISO week, or one that is not closed yet (A3).
	ErrInvalidPeriod = errors.New("trends: invalid or open period")
	// ErrFrozen: the period was built already and is never built again (A3).
	ErrFrozen = errors.New("trends: period already built")
	// ErrShortKey: the build's pseudonym key could not be made.
	ErrShortKey = errors.New("trends: no pseudonym key")
)
