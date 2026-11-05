package auth_test

import (
	"context"
	"time"
)

// purge is one call of a retention delete: the table and the cut-off.
type purge struct {
	before time.Time
	table  string
}

func (r *fakeRepository) DeleteSessionsEndedBefore(ctx context.Context, before time.Time) (int64, error) {
	r.seen(ctx, "DeleteSessionsEndedBefore")
	r.purges = append(r.purges, purge{table: "session", before: before})
	return r.purged, r.err
}

func (r *fakeRepository) DeleteInvitesEndedBefore(ctx context.Context, before time.Time) (int64, error) {
	r.seen(ctx, "DeleteInvitesEndedBefore")
	r.purges = append(r.purges, purge{table: "invite", before: before})
	return r.purged, r.err
}
