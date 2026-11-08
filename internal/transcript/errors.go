package transcript

import (
	"errors"
	"fmt"
)

// ErrInvalid is returned for an event the contract does not allow. The error names the field, never its value.
var ErrInvalid = errors.New("transcript: invalid event")

func invalid(field string) error { return fmt.Errorf("%w: %s", ErrInvalid, field) }
