package worker

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// revocationWorker cancels a patient's future appointments with a clinic after they revoked their consent
// there (consent_revoked, enqueued in the revocation's transaction). It logs the count, nothing else.
type revocationWorker struct {
	river.WorkerDefaults[consent.RevokedArgs]
	appointments *appointment.Service
	log          *slog.Logger
}

func (w *revocationWorker) Work(ctx context.Context, job *river.Job[consent.RevokedArgs]) error {
	n, err := w.appointments.CancelForRevokedConsent(ctx, job.Args.PatientID, job.Args.OrganizationID)
	w.log.InfoContext(ctx, "appointments cancelled after a revocation", "cancelled", n)
	return err
}

// newRevocations builds the appointment service on pool (salus_app) at clk for CancelForRevokedConsent: the
// consent's state, the slots to free and the cancellation e-mails to enqueue on jobs (an insert-only client).
func newRevocations(pool *pgxpool.Pool, jobs *river.Client[pgx.Tx], clk clock.Clock) *appointment.Service {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	// Only State is asked of the consent service: it needs no jobs.
	consents := consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clk, nil)
	return appointment.NewService(appointment.Ports{
		Repo: appointment.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk,
		Consents: consents, Slots: newSlots(pool, clk), Jobs: appointment.NewRiverJobs(jobs),
	})
}
