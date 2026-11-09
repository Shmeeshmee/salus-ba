package suggestion

import "context"

// Repository stores the suggestions (suggestion). Row-level security shows a transaction the suggestions of
// the encounters its clinic's scope shows; the encounter's service sets the scope.
type Repository interface {
	// Insert stores the suggestion of e under its ID unless one with that id is stored already, and reports
	// whether this call stored it. What was suggested, and from what, never changes.
	Insert(ctx context.Context, e Event) (bool, error)
	// SetStatus sets the status of the encounter's suggestion id and returns it, and whether this call changed
	// it; ErrNotFound for no such suggestion of the encounter in the scope.
	SetStatus(ctx context.Context, encounterID, id string, status Status) (Event, bool, error)
	// ByEncounter returns the encounter's suggestions, oldest first.
	ByEncounter(ctx context.Context, encounterID string) ([]Event, error)
}
