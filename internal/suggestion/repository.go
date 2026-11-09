package suggestion

import "context"

// Saved is what a Save did.
type Saved int

// The outcomes of a Save.
const (
	// SavedUnchanged: the suggestion is stored already with that status (an event delivered again).
	SavedUnchanged Saved = iota
	// SavedCreated: the suggestion was new and is stored.
	SavedCreated
	// SavedUpdated: the stored suggestion took the event's status.
	SavedUpdated
)

// Repository stores the suggestions (suggestion). Row-level security shows a transaction the suggestions of
// the encounters its clinic's scope shows; the encounter's service sets the scope.
type Repository interface {
	// Save stores the suggestion of e under its ID when it is new; when it is stored already for e's
	// encounter, it takes e's status and nothing else (what was suggested, and from what, never changes).
	Save(ctx context.Context, e Event) (Saved, error)
}
