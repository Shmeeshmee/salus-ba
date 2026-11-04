package auditevent

import "errors"

// ErrInvalid is returned for an event with an unknown action or outcome, or a missing code.
var ErrInvalid = errors.New("auditevent: invalid event")
