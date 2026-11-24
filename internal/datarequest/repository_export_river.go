package datarequest

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// ExportAttempts is how often the build is tried before the export is marked failed.
const ExportAttempts = 3

type riverExportJobs struct {
	client *river.Client[pgx.Tx]
}

// NewRiverExportJobs returns the ExportJobs port on a River client (an insert-only one will do).
func NewRiverExportJobs(client *river.Client[pgx.Tx]) ExportJobs {
	return &riverExportJobs{client: client}
}

func (j *riverExportJobs) EnqueueExport(ctx context.Context, args ExportArgs) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	if _, err := j.client.InsertTx(ctx, tx, args, &river.InsertOpts{MaxAttempts: ExportAttempts}); err != nil {
		return fmt.Errorf("datarequest: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}
