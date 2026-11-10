package consent

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

func (j *riverJobs) EnqueueConsentRevoked(ctx context.Context, patientID, organizationID string) error {
	return j.insert(ctx, RevokedArgs{PatientID: patientID, OrganizationID: organizationID})
}

func (j *riverJobs) EnqueueTranscriptionRevoked(ctx context.Context, patientID, organizationID string) error {
	return j.insert(ctx, TranscriptionRevokedArgs{PatientID: patientID, OrganizationID: organizationID})
}

// insert enqueues args in the transaction in ctx.
func (j *riverJobs) insert(ctx context.Context, args river.JobArgs) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	if _, err := j.client.InsertTx(ctx, tx, args, nil); err != nil {
		return fmt.Errorf("consent: enqueue %s: %w", args.Kind(), err)
	}
	return nil
}
