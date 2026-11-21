package subscription

import "errors"

// The errors of the membership. Their text carries no amount.
var (
	// ErrNotFound is returned for a clinic without a membership, or one the caller may not see.
	ErrNotFound = errors.New("subscription: not found")
	// ErrForbidden is returned when staff who are not the clinic's admin ask for its membership.
	ErrForbidden = errors.New("subscription: forbidden")
	// ErrStatus is returned for a status move that is not forward.
	ErrStatus = errors.New("subscription: invalid status change")
)
