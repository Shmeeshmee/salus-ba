package schedule

import (
	"context"
	"time"
)

// Repository stores schedules. Every read sees only the clinic set by SetOrganization, or the schedules of
// active roles of listed clinics after SetDirectory (row-level security).
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// SetDirectory scopes the rest of the transaction in ctx to the public directory.
	SetDirectory(ctx context.Context) error
	// ByID returns the schedule, or ErrNotFound (also outside the scope, or for an id that is not a UUID).
	ByID(ctx context.Context, id string) (Schedule, error)
	// ByRole returns the schedule of the practitioner role, or ErrNotFound.
	ByRole(ctx context.Context, roleID string) (Schedule, error)
	// Lock returns the schedule, locked until the transaction ends so generations of one schedule and its
	// edits run one after the other, or ErrNotFound.
	Lock(ctx context.Context, id string) (Schedule, error)
	// Listed returns every schedule the scope shows, oldest first.
	Listed(ctx context.Context) ([]Schedule, error)
	// Create stores the first schedule of s's role at version 1, or returns ErrConflict when the role has one.
	Create(ctx context.Context, s Schedule) (Schedule, error)
	// Update sets s's slot length and rules when s.VersionID is the current version, moving it to the next,
	// last updated at at; or returns ErrConflict.
	Update(ctx context.Context, s Schedule, at time.Time) (Schedule, error)
}
