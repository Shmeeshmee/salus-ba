package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/diagnosticreport"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// labResultPorts are what the lab results are built on.
type labResultPorts struct {
	pool         *pgxpool.Pool
	audit        *auditevent.Service
	tx           *postgres.Transactor
	appointments *appointment.Service
	accounts     *auth.Service
	patients     *patient.Service
	files        *binary.Service // the PDFs: binary with a report as their parent
}

// labResultRoutes are the lab results' (docs/clinic-cms/WAVE6.md U5, package LB1): attached, changed and released by
// the clinic's admin and practitioners on a visit, read by the patient once released, with the lab's PDF scanned and
// encrypted as every Binary. The receptionist is refused by the service, audited (deskCaller names the receptionist
// rather than hiding them). It returns the routes and the PDF's streaming routes.
func labResultRoutes(p labResultPorts, log *slog.Logger) (routes, streams func(*http.ServeMux)) {
	svc := diagnosticreport.NewService(diagnosticreport.Ports{
		Repo: diagnosticreport.NewPostgresRepository(p.pool), Visits: p.appointments, Audit: p.audit, Tx: p.tx, Clock: clock.System{},
		Files: p.files,
	})
	h := diagnosticreport.NewHTTPHandler(svc, labResultSession(p.accounts, p.patients), clock.System{}, log)
	return h.Routes, h.StreamRoutes
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
