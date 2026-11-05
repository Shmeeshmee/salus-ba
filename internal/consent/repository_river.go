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
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return postgres.ErrNoTx
	}
	args := RevokedArgs{PatientID: patientID, OrganizationID: organizationID}
	if _, err := j.client.InsertTx(ctx, tx, args, nil); err != nil {
		return fmt.Errorf("consent: enqueue consent_revoked: %w", err)
	}
	return nil
}
