package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// dataRequestPorts are what the data subject's requests are built on in the API (docs/clinic-cms/WAVE5.md, D1).
type dataRequestPorts struct {
	pool     *pgxpool.Pool
	audit    *auditevent.Service
	tx       *postgres.Transactor
	accounts *auth.Service
	patients *patient.Service
	consents *consent.Service
	clinics  *organization.Service
}

// dataRequestRoutes are the patient's /me/data-requests (list, file, withdraw) and the clinic's /clinic/data-requests
// (list, read, answer, extend) and /clinic/patients/{id}/data-requests (S10).
func dataRequestRoutes(p dataRequestPorts, log *slog.Logger) func(*http.ServeMux) {
	svc := datarequest.NewService(datarequest.Ports{
		Repo: datarequest.NewPostgresRepository(p.pool), Clock: clock.System{}, Tx: p.tx, Audit: p.audit,
		Profiles: p.patients, Consents: p.consents, Clinics: p.clinics,
	})
	return datarequest.NewHandler(svc, dataRequestCaller(p.accounts), log).Routes
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
