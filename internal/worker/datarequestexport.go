package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// dataRequestExportWorker builds a patient's own export (docs/clinic-cms/WAVE5.md D2, D7, S4). It is tried twice
// (datarequest.ExportAttempts): on the last attempt a failure marks the export failed; the request stays open for its
// answerer. It logs the request's id on a failure,
// nothing else.
type dataRequestExportWorker struct {
	river.WorkerDefaults[datarequest.ExportArgs]
	exports *datarequest.Service
	log     *slog.Logger
}

// exportTimeout bounds one build attempt: River's default minute is short for an archive with every signed PDF and
// uploaded file of the patient's (D7). Not measured yet (docs/clinic-cms/PROGRESS.md D7).
const exportTimeout = 15 * time.Minute

func (w *dataRequestExportWorker) Timeout(*river.Job[datarequest.ExportArgs]) time.Duration {
	return exportTimeout
}

func (w *dataRequestExportWorker) Work(ctx context.Context, job *river.Job[datarequest.ExportArgs]) error {
	err := w.exports.BuildExport(ctx, job.Args)
	if err == nil || job.Attempt < job.MaxAttempts {
		return err
	}
	w.log.WarnContext(ctx, "data request export failed", "data_request_id", job.Args.RequestID)
	return w.exports.FailExport(ctx, job.Args)
}

// newDataRequestExports builds the export's service on pool (salus_app) for its build job: what the patient reads in
// the app, through the services the app reads (profile, consents, appointments, encounters, signed documents and their
// PDFs, submitted answers, their questionnaires and files, receipts and quotes), and their account, each read audited
// by its own service as the patient's, and the object store and key-encryption key of the binary jobs.
func newDataRequestExports(pool *pgxpool.Pool, files Files, appointments *appointment.Service, receipts *invoice.Service, clk clock.Clock) *datarequest.Service {
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clk)
	consents := consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clk, nil)
	roles := practitionerrole.NewService(practitionerrole.Ports{Repo: practitionerrole.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk})
	encounters := encounter.NewService(encounter.Ports{
		Repo: encounter.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk, Appointments: appointments, Consents: consents, Roles: roles,
	})
	questionnaires := questionnaire.NewService(questionnaire.Ports{Repo: questionnaire.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk, Roles: roles})
	answers := questionnaireresponse.NewService(questionnaireresponse.Ports{
		Repo: questionnaireresponse.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk,
		Appointments: appointments, Questionnaires: questionnaires, Encounters: encounters,
	})
	return datarequest.NewService(datarequest.Ports{Clock: clk, Export: datarequest.ExportPorts{
		Repo: datarequest.NewExportRepository(pool), Audit: audit, Tx: tx, Store: files.Store, KEK: files.KEK,
		Records: clinicexport.NewOwnRecords(clinicexport.OwnPorts{
			Profile:  patient.NewService(patient.NewPostgresRepository(pool), audit, tx, clk, consents, nil),
			Consents: consents, Appointments: appointments, Receipts: receipts, Quotes: receipts, Encounters: encounters,
			Documents: composition.NewService(composition.Ports{
				Repo: composition.NewPostgresRepository(pool), Encounters: encounters, Store: files.Store, Audit: audit, Tx: tx, Clock: clk, KEK: files.KEK,
			}),
			Answers: answers, Questionnaires: questionnaires,
			Files: binary.NewService(binary.Ports{Repo: binary.NewPostgresRepository(pool), Store: files.Store, Holder: answers, Audit: audit, Tx: tx, Clock: clk, KEK: files.KEK}),
			Chat: communication.NewService(communication.Ports{ // the patient's own reads, as the app's chat makes them
				Repo: communication.NewPostgresRepository(pool), Consents: consents, ChatConsents: consents, Visits: appointment.NewFacts(appointment.NewPostgresRepository(pool)),
				Roles: roles, Directory: roles, Encounters: encounters, Audit: audit, Tx: tx, Clock: clk,
				Files: binary.NewService(binary.Ports{Repo: binary.NewPostgresRepository(pool), Store: files.Store, Audit: audit, Tx: tx, Clock: clk, KEK: files.KEK}),
			}),
		}),
		Accounts: auth.NewService(auth.Ports{Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clk}),
	}})
}
