package practitioner

import "errors"

// ErrNotFound is returned for a practitioner that does not exist or that the caller may not see: its
// existence is not disclosed. Its text carries no name.
var ErrNotFound = errors.New("practitioner: not found")
