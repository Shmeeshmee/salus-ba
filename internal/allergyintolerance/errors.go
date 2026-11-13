package allergyintolerance

import "errors"

// The refusals: no such entry or visit for the caller (its existence is not disclosed), a caller who sees the entry
// but may not make the change, an input outside its value sets or lengths, and an entry that changed since the
// version the caller read.
var (
	ErrNotFound        = errors.New("allergyintolerance: not found")
	ErrForbidden       = errors.New("allergyintolerance: forbidden")
	ErrInvalid         = errors.New("allergyintolerance: invalid")
	ErrVersionConflict = errors.New("allergyintolerance: version conflict")
)
