package clinicexport

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// BuildAttempts is how often the build is tried before the export is marked failed.
const BuildAttempts = 3

type riverJobs struct {
	client *river.Client[pgx.Tx]
}

// NewRiverJobs returns the Jobs port on a River client (an insert-only one will do).
func NewRiverJobs(client *river.Client[pgx.Tx]) Jobs {
	return &riverJobs{client: client}
}

func (j *riverJobs) EnqueueBuild(ctx context.Context, args BuildArgs) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	if _, err := j.client.InsertTx(ctx, tx, args, &river.InsertOpts{MaxAttempts: BuildAttempts}); err != nil {
		return fmt.Errorf("clinicexport: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}
