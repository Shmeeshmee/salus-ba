package invoice

import "errors"

// The errors of receipts. Their text carries no name, amount or number.
var (
	// ErrNotFound is returned for a receipt that does not exist or that the caller may not see.
	ErrNotFound = errors.New("invoice: not found")
	// ErrForbidden is returned when someone who may not issue or reverse receipts of the clinic tries to.
	ErrForbidden = errors.New("invoice: forbidden")
	// ErrInvalid is returned for a receipt without lines, with both a patient and a company, an unknown payment method,
	// or a line the price list does not price.
	ErrInvalid = errors.New("invoice: invalid receipt")
	// ErrAlreadyReversed is returned for a reversal of a receipt that was already reversed, or of a reversal.
	ErrAlreadyReversed = errors.New("invoice: already reversed")
	// ErrReportFailed is returned for a reversal of a receipt whose fiscal report failed: the reversal names the
	// receipt's fiscal number, so it could never be reported either.
	ErrReportFailed = errors.New("invoice: the receipt's fiscal report failed")
	// ErrSellerIncomplete is returned for a receipt of a clinic whose legal name, tax id or address is missing: a
	// receipt names its seller.
	ErrSellerIncomplete = errors.New("invoice: the clinic's billing details are incomplete")
	// ErrOriginalPending is returned by the fiscal job for a reversal whose receipt is not fiscalized yet: the job
	// is retried.
	ErrOriginalPending = errors.New("invoice: the reversed receipt is not fiscalized yet")
)
