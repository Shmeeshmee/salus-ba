package videosession

import (
	"context"
	"fmt"
	"time"

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

func (j *riverJobs) EnqueueWindowClose(ctx context.Context, sessionID, org string, at time.Time) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	args := WindowCloseArgs{SessionID: sessionID, OrganizationID: org}
	if _, err := j.client.InsertTx(ctx, tx, args, &river.InsertOpts{Queue: river.QueueDefault, ScheduledAt: at}); err != nil {
		return fmt.Errorf("videosession: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}

func (j *riverJobs) EnqueueAgentJoin(ctx context.Context, sessionID, org string) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	args := AgentJoinArgs{SessionID: sessionID, OrganizationID: org}
	if _, err := j.client.InsertTx(ctx, tx, args, &river.InsertOpts{Queue: QueueTranscription}); err != nil {
		return fmt.Errorf("videosession: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}
