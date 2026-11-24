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
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// dataRequestPorts are what the data subject's requests are built on in the API (docs/clinic-cms/WAVE5.md D1 to D3).
type dataRequestPorts struct {
	pool     *pgxpool.Pool
	jobs     *river.Client[pgx.Tx]
	audit    *auditevent.Service
	tx       *postgres.Transactor
	kek      *envelope.KEK
	accounts *auth.Service
	patients *patient.Service
	consents *consent.Service
	clinics  *organization.Service
	files    config.ObjectStore
}

// dataRequestRoutes are the patient's /me/data-requests (list, file, withdraw), their export's status and download
// (D2) and /me/account/close (D3), and the clinic's /clinic/data-requests (list, read, answer, extend) and
// /clinic/patients/{id}/data-requests (S10). One request service serves them, with the exports on: filing an access or
// portability request queues its export in the filing's transaction.
func dataRequestRoutes(p dataRequestPorts, log *slog.Logger) func(*http.ServeMux) {
	repo := datarequest.NewPostgresRepository(p.pool)
	svc := datarequest.NewService(datarequest.Ports{
		Repo: repo, Clock: clock.System{}, Tx: p.tx, Audit: p.audit,
		Profiles: p.patients, Consents: p.consents, Clinics: p.clinics, Export: dataRequestExportsOn(p),
	})
	closure := datarequest.NewClosure(datarequest.ClosurePorts{
		Tx: p.tx, Repo: repo, Scope: datarequest.NewErasureScope(), Accounts: p.accounts, Profiles: p.patients,
		Consents: p.consents, Clock: clock.System{},
		Devices: notification.NewChat(notification.ChatPorts{
			Repo: notification.NewPostgresRepository(p.pool), Audit: p.audit, Tx: p.tx, Clock: clock.System{},
		}),
	})
	requests := datarequest.NewHandler(svc, dataRequestCaller(p.accounts), log)
	exports := datarequest.NewExportHandler(svc, dataRequestPatient, log)
	closing := datarequest.NewCloseHandler(closure, dataRequestCloser, log)
	return func(mux *http.ServeMux) {
		requests.Routes(mux)
		exports.Routes(mux)
		closing.Routes(mux)
	}
}

// dataRequestCaller is the requests' view of the signed-in session: staff with their clinic and whether they are its
// admin or a receptionist, as read from the database; a patient with no clinic.
func dataRequestCaller(accounts *auth.Service) func(ctx context.Context) (datarequest.Staff, bool, error) {
	return func(ctx context.Context) (datarequest.Staff, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return datarequest.Staff{}, false, nil
		}
		st := datarequest.Staff{AccountID: s.AccountID, OrganizationID: s.OrganizationID}
		if st.OrganizationID == "" {
			return st, true, nil
		}
		m, err := accounts.ClinicMember(ctx, s)
		if err != nil {
			return datarequest.Staff{}, false, err
		}
		st.Admin, st.Reception = m.Admin, m.Reception
		return st, true, nil
	}
}

// dataRequestExportsOn are the export's ports in the API: its status and download, and its queueing when an access or
// portability request is filed (Service.QueueExport). The archive is built by cmd/worker's data_request_export_build.
func dataRequestExportsOn(p dataRequestPorts) datarequest.ExportPorts {
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

// dataRequestCloser is the signed-in account as the patient closing it; the closure refuses an account that is not an
// open patient account (ErrNotFound).
func dataRequestCloser(ctx context.Context) (datarequest.Patient, bool, error) {
	p, ok := dataRequestPatient(ctx)
	return p, ok, nil
}
