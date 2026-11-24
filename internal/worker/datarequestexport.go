package worker

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// dataRequestExportWorker builds a patient's own export (docs/clinic-cms/WAVE5.md D2, S4). On its last attempt a
// failure marks the export failed; the request stays open for its answerer. It logs the request's id on a failure,
// nothing else.
type dataRequestExportWorker struct {
	river.WorkerDefaults[datarequest.ExportArgs]
	exports *datarequest.Service
	log     *slog.Logger
}

func (w *dataRequestExportWorker) Work(ctx context.Context, job *river.Job[datarequest.ExportArgs]) error {
	err := w.exports.BuildExport(ctx, job.Args)
	if err == nil || job.Attempt < job.MaxAttempts {
		return err
	}
	w.log.WarnContext(ctx, "data request export failed", "data_request_id", job.Args.RequestID)
	return w.exports.FailExport(ctx, job.Args)
}

// newDataRequestExports builds the export's service on pool (salus_app) for its build job: the patient's own profile,
// consents, appointments (the appointment service of the default queue) and receipts, and their account, each read
// audited by its own service as the patient's, and the object store and key-encryption key of the binary jobs.
func newDataRequestExports(pool *pgxpool.Pool, files Files, appointments *appointment.Service, receipts *invoice.Service, clk clock.Clock) *datarequest.Service {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	consents := consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clk, nil)
	return datarequest.NewService(datarequest.Ports{Clock: clk, Export: datarequest.ExportPorts{
		Repo: datarequest.NewExportRepository(pool), Audit: audit, Tx: tx, Store: files.Store, KEK: files.KEK,
		Records: clinicexport.NewOwnRecords(clinicexport.OwnPorts{
			Profile:  patient.NewService(patient.NewPostgresRepository(pool), audit, tx, clk, consents, nil),
			Consents: consents, Appointments: appointments, Receipts: receipts,
		}),
		Accounts: auth.NewService(auth.Ports{Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk}),
	}})
}
