package practitionerrole

import "errors"

// ErrNotFound is returned for a role, or a clinic's roles, that does not exist or that the caller may not
// see: its existence is not disclosed.
var ErrNotFound = errors.New("practitionerrole: not found")
