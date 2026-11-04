package slot

import (
	"context"
	"time"
)

// Repository stores slots. Every read and write sees only the clinic set by SetOrganization, or the free
// slots of listed clinics after SetDirectory (row-level security); the retention role's sees every clinic.
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// SetDirectory scopes the rest of the transaction in ctx to the public directory.
	SetDirectory(ctx context.Context) error
	// DeleteFreeExcept deletes the free slots of the schedule that start after after and are not one of keep,
	// and returns how many.
	DeleteFreeExcept(ctx context.Context, scheduleID string, after time.Time, keep []Period) (int64, error)
	// Insert adds a free slot of the schedule of the clinic org for each period that overlaps none of the
	// schedule's slots, and returns how many.
	Insert(ctx context.Context, scheduleID, org string, periods []Period) (int64, error)
	// Search returns the free slots of the schedule that start in [from, to), by start.
	Search(ctx context.Context, scheduleID string, from, to time.Time) ([]Slot, error)
	// Hold marks the free slot id busy, last updated at at, and returns it; or ErrUnavailable.
	Hold(ctx context.Context, id string, at time.Time) (Slot, error)
	// Release marks the busy slot id free, last updated at at, when it starts after at, and reports whether
	// it did.
	Release(ctx context.Context, id string, at time.Time) (bool, error)
	// DeleteFreeEndedBefore deletes the free slots that ended before before, in every clinic the login may
	// see, and returns how many.
	DeleteFreeEndedBefore(ctx context.Context, before time.Time) (int64, error)
}
