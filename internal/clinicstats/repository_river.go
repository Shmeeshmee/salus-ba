package clinicstats

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type riverQueue struct {
	client *river.Client[pgx.Tx]
}

// NewRiverQueue returns the Queue on a River client (an insert-only one will do): clinicstats_refresh on
// QueueStats, which the worker works.
func NewRiverQueue(client *river.Client[pgx.Tx]) Queue {
	return &riverQueue{client: client}
}

func (q *riverQueue) EnqueueStatsRefresh(ctx context.Context, orgID string, day time.Time) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	args := RefreshArgs{OrganizationID: orgID, Day: day}
	if _, err := q.client.InsertTx(ctx, tx, args, nil); err != nil {
		return fmt.Errorf("clinicstats: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}
