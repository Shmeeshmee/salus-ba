package worker

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// clinicExportWorker builds a clinic's data export (docs/clinic-cms M2). On its last attempt a failure marks the
// export failed, so the admin sees it and may ask again. It logs the export's id on a failure, nothing else.
type clinicExportWorker struct {
	river.WorkerDefaults[clinicexport.BuildArgs]
	exports *clinicexport.Service
	log     *slog.Logger
}

func (w *clinicExportWorker) Work(ctx context.Context, job *river.Job[clinicexport.BuildArgs]) error {
	err := w.exports.Build(ctx, job.Args)
	if err == nil || job.Attempt < job.MaxAttempts {
		return err
	}
	w.log.WarnContext(ctx, "clinic export failed", "export_id", job.Args.ExportID)
	return w.exports.Fail(ctx, job.Args)
}

// newClinicExports builds the export's service on pool (salus_app) for its build job: the clinic's price list, staff,
// appointments (the appointment service of the default queue), receipts and quotes, and the records of the patients it
// registered (profile and consents), each read audited by its own service, and the object store and key-encryption
// key of the binary jobs.
func newClinicExports(pool *pgxpool.Pool, files Files, appointments *appointment.Service, receipts *invoice.Service, clk clock.Clock) *clinicexport.Service {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	return clinicexport.NewService(clinicexport.Ports{
		Repo: clinicexport.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk, Store: files.Store, KEK: files.KEK,
		PriceList:     chargeitemdefinition.NewService(chargeitemdefinition.Ports{Repo: chargeitemdefinition.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk}),
		Roles:         practitionerrole.NewService(practitionerrole.Ports{Repo: practitionerrole.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk}),
		Practitioners: practitioner.NewService(practitioner.NewPostgresRepository(pool), audit, tx),
		Appointments:  appointments, Receipts: receipts, Quotes: receipts,
		Admins: auth.NewService(auth.Ports{Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk}),
		Patients: patient.NewService(patient.NewPostgresRepository(pool), audit, tx, clk,
			consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clk, nil), nil),
	})
}
