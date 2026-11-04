package schedule

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

func (j *riverJobs) EnqueueSlotsGenerate(ctx context.Context, scheduleID, org string) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	args := SlotsGenerateArgs{ScheduleID: scheduleID, OrganizationID: org}
	if _, err := j.client.InsertTx(ctx, tx, args, nil); err != nil {
		return fmt.Errorf("schedule: enqueue slots_generate: %w", err)
	}
	return nil
}
