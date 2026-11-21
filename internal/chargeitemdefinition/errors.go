package chargeitemdefinition

import "errors"

// The errors of the price list. Their text carries no name or price.
var (
	// ErrNotFound is returned for a service that does not exist or that the caller may not see.
	ErrNotFound = errors.New("chargeitemdefinition: not found")
	// ErrForbidden is returned when staff who are not the clinic's admin change its price list.
	ErrForbidden = errors.New("chargeitemdefinition: forbidden")
	// ErrConflict is returned for an update of a version that is not the current one.
	ErrConflict = errors.New("chargeitemdefinition: version conflict")
	// ErrInvalid is returned for a service with a blank name, a negative price, an unknown currency or mode, or a VAT
	// rate out of range.
	ErrInvalid = errors.New("chargeitemdefinition: invalid service")
)
