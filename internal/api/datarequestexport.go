package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// dataRequestExportPorts are what the patient's export is built on in the API (docs/clinic-cms/WAVE5.md D2): its
// status and its download. The archive is built by cmd/worker's data_request_export_build job.
type dataRequestExportPorts struct {
	pool  *pgxpool.Pool
	jobs  *river.Client[pgx.Tx]
	audit *auditevent.Service
	tx    *postgres.Transactor
	kek   *envelope.KEK
	files config.ObjectStore
}

// dataRequestExportRoutes are GET /me/data-requests/{id}/export and its /archive, for the patient who filed the
// request.
func dataRequestExportRoutes(p dataRequestExportPorts, log *slog.Logger) func(*http.ServeMux) {
	svc := datarequest.NewService(datarequest.Ports{Clock: clock.System{}, Export: dataRequestExportsOn(p)})
	return datarequest.NewExportHandler(svc, dataRequestPatient, log).Routes
}

// dataRequestExportsOn are the export's ports in the API, which D1's request service is built with too: filing an
// access or portability request queues its export (Service.QueueExport).
func dataRequestExportsOn(p dataRequestExportPorts) datarequest.ExportPorts {
	return datarequest.ExportPorts{
		Repo: datarequest.NewExportRepository(p.pool), Audit: p.audit, Tx: p.tx, Jobs: datarequest.NewRiverExportJobs(p.jobs),
		KEK: p.kek, Store: binary.NewS3Store(&http.Client{Timeout: binary.UploadDeadline}, p.files.Endpoint, p.files.Region,
			p.files.Bucket, p.files.AccessKeyID, p.files.SecretAccessKey, s3PageSize),
	}
}

// dataRequestPatient is the signed-in account as the patient of their requests. Staff have no export of their own:
// their account's scope finds none.
func dataRequestPatient(ctx context.Context) (datarequest.Patient, bool) {
	s, ok := auth.SessionFrom(ctx)
	return datarequest.Patient{AccountID: s.AccountID}, ok
}
