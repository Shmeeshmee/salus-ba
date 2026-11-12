package notification

import "errors"

// ErrRecipientNotASCII is a recipient address with a UTF-8 local part or an IDN domain: SMTP without SMTPUTF8
// cannot carry it, so no retry can send the e-mail. It names no address.
var ErrRecipientNotASCII = errors.New("notification: the recipient's address is not ASCII, which SMTP cannot carry")

// The errors of the chat's notifications. Their text carries no token, address, URL or secret.
var (
	// ErrNotFound is returned for a device, an endpoint or a clinic the caller may not see.
	ErrNotFound = errors.New("notification: not found")
	// ErrForbidden is returned when someone other than the clinic's admin manages its webhooks, or staff register a
	// device (the app is the patients').
	ErrForbidden = errors.New("notification: forbidden")
	// ErrInvalid is returned for a request the contract refuses: a platform or token, a webhook URL that is not
	// https to a public host, an unknown event, or one endpoint too many.
	ErrInvalid = errors.New("notification: invalid request")
)
