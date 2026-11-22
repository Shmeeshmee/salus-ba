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
	// ErrNoPlan is returned for a plan that does not exist or is not active (the owner sets plans in salusctl).
	ErrNoPlan = errors.New("subscription: no such plan")
	// ErrInvalidPlan is returned for a plan whose code, name, price or currency is malformed.
	ErrInvalidPlan = errors.New("subscription: invalid plan")
	// ErrNotBilled is returned for a month a clinic gets no invoice for: its membership is cancelled, or its trial has
	// not ended when the month starts.
	ErrNotBilled = errors.New("subscription: not billed")
)
