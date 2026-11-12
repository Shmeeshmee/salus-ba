package organization

import "errors"

// The errors of the organization domain. Their text carries no name.
var (
	// ErrNotFound is returned for a clinic that does not exist or that the caller may not see: its existence
	// is not disclosed.
	ErrNotFound = errors.New("organization: not found")
	// ErrForbidden is returned when a member of the clinic who is not its admin tries to change it.
	ErrForbidden = errors.New("organization: forbidden")
	// ErrConflict is returned for an update of a version that is not the current one.
	ErrConflict = errors.New("organization: version conflict")
	// ErrInvalidName is returned for a name that is blank or longer than 200 characters.
	ErrInvalidName = errors.New("organization: invalid name")
	// ErrInvalidTimeZone is returned for a time zone that is not an IANA zone name.
	ErrInvalidTimeZone = errors.New("organization: invalid time zone")
	// ErrInvalidDefaultLanguage is returned for a default language that is not a staff language (bs, hr,
	// sr-Latn, sr-Cyrl, en).
	ErrInvalidDefaultLanguage = errors.New("organization: invalid default language")
	// ErrInvalidChatSettings is returned for chat settings out of their ranges, or hours that are not valid
	// stretches of a weekday.
	ErrInvalidChatSettings = errors.New("organization: invalid chat settings")
)
