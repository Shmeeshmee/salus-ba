package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// clinicExportPorts are what the clinic's data export is built on in the API (docs/clinic-cms M2): asking, reading
// and downloading. The archive is built by cmd/worker's clinic_export_build job.
type clinicExportPorts struct {
	pool     *pgxpool.Pool
	jobs     *river.Client[pgx.Tx]
	audit    *auditevent.Service
	tx       *postgres.Transactor
	kek      *envelope.KEK
	accounts *auth.Service
	// The sources of a registered patient's record, for its hand-over (GET /clinic/patients/{id}/record).
	patients     *patient.Service
	appointments *appointment.Service
	files        config.ObjectStore
}

// clinicExportRoutes are POST /clinic/export, GET /clinic/export/{id} and its /archive, and the hand-over of a
// registered patient's record (GET /clinic/patients/{id}/record), for the clinic's admin.
func clinicExportRoutes(p clinicExportPorts, log *slog.Logger) func(*http.ServeMux) {
	svc := clinicexport.NewService(clinicexport.Ports{
		Repo: clinicexport.NewPostgresRepository(p.pool), Audit: p.audit, Tx: p.tx, Clock: clock.System{},
		Jobs: clinicexport.NewRiverJobs(p.jobs), KEK: p.kek, Patients: p.patients, Appointments: p.appointments,
		Receipts: invoice.NewService(invoice.Ports{Repo: invoice.NewPostgresRepository(p.pool), Audit: p.audit, Tx: p.tx, Clock: clock.System{}}),
		Store: binary.NewS3Store(&http.Client{Timeout: binary.UploadDeadline}, p.files.Endpoint, p.files.Region, p.files.Bucket,
			p.files.AccessKeyID, p.files.SecretAccessKey, s3PageSize),
	})
	clinicCaller := organizationCaller(p.accounts)
	caller := func(ctx context.Context) (clinicexport.Caller, bool, error) {
		c, ok, err := clinicCaller(ctx)
		return clinicexport.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, Admin: c.Admin}, ok, err
	}
	return clinicexport.NewHandler(svc, caller, log).Routes
}
