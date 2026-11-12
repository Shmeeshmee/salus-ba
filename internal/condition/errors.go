package condition

import "errors"

// ErrInvalid is returned for a Condition the model does not take: an id that is not a UUID, a code that is not a
// well-formed ICD-10 code, an unknown verification status, or no field key or recording time.
var ErrInvalid = errors.New("condition: invalid condition")
