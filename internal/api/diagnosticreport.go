package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// labResultRoutes are the lab results' (docs/clinic-cms/WAVE6.md U5, package LB1): attached, changed and released by
// the clinic's admin and practitioners on a visit, read by the patient once released. The receptionist is refused by
// the service, audited (deskCaller names the receptionist rather than hiding them).
func labResultRoutes(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor, appointments *appointment.Service,
	accounts *auth.Service, patients *patient.Service, log *slog.Logger,
) func(*http.ServeMux) {
	svc := diagnosticreport.NewService(diagnosticreport.Ports{
		Repo: diagnosticreport.NewPostgresRepository(pool), Visits: appointments, Audit: audit, Tx: tx, Clock: clock.System{},
	})
	return diagnosticreport.NewHTTPHandler(svc, labResultSession(accounts, patients), log).Routes
}

// labResultSession is the lab results' view of the signed-in session.
func labResultSession(accounts *auth.Service, patients *patient.Service) func(ctx context.Context) (diagnosticreport.Session, bool, error) {
	caller := deskCaller(accounts, patients)
	return func(ctx context.Context) (diagnosticreport.Session, bool, error) {
		c, ok, err := caller(ctx)
		if c.OrganizationID == "" {
			return diagnosticreport.Session{Patient: diagnosticreport.Patient{AccountID: c.AccountID, PatientID: c.PatientID}}, ok, err
		}
		return diagnosticreport.Session{Staff: diagnosticreport.Staff{
			AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID, Admin: c.Admin, Reception: c.Reception,
		}}, ok, err
	}
}
