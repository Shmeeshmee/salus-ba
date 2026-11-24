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
	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/internal/config"
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
	files    config.ObjectStore
}

// clinicExportRoutes are POST /clinic/export, GET /clinic/export/{id} and its /archive, for the clinic's admin.
func clinicExportRoutes(p clinicExportPorts, log *slog.Logger) func(*http.ServeMux) {
	svc := clinicexport.NewService(clinicexport.Ports{
		Repo: clinicexport.NewPostgresRepository(p.pool), Audit: p.audit, Tx: p.tx, Clock: clock.System{},
		Jobs: clinicexport.NewRiverJobs(p.jobs), KEK: p.kek,
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
