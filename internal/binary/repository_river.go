package binary

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

type riverJobs struct {
	client *river.Client[pgx.Tx]
}

// NewRiverJobs returns the Jobs port on a River client (an insert-only one will do).
func NewRiverJobs(client *river.Client[pgx.Tx]) Jobs {
	return &riverJobs{client: client}
}

func (j *riverJobs) EnqueueScan(ctx context.Context, args ScanArgs) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	opts := &river.InsertOpts{Queue: QueueScan}
	if args.ConversationID != "" {
		opts.MaxAttempts = ChatScanAttempts
	}
	if _, err := j.client.InsertTx(ctx, tx, args, opts); err != nil {
		return fmt.Errorf("binary: enqueue binary_scan: %w", err)
	}
	return nil
}
