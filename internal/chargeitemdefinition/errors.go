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

// ErrInvalidImport is returned for a price-list file that is not a CSV (UTF-8 or Windows-1250) or XLSX file, has no
// header row naming at least the name and price columns, or has no rows or more than 1000.
var ErrInvalidImport = errors.New("chargeitemdefinition: invalid import")
