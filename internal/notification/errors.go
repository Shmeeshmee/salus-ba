package notification

import "errors"

// ErrRecipientNotASCII is a recipient address with a UTF-8 local part or an IDN domain: SMTP without SMTPUTF8
// cannot carry it, so no retry can send the e-mail. It names no address.
var ErrRecipientNotASCII = errors.New("notification: the recipient's address is not ASCII, which SMTP cannot carry")
