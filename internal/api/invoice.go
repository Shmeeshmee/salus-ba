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
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// invoicePorts are what the receipts' routes are built from.
type invoicePorts struct {
	pool         *pgxpool.Pool
	audit        *auditevent.Service
	tx           *postgres.Transactor
	jobs         *river.Client[pgx.Tx]
	clinics      *organization.Service
	appointments *appointment.Service
	consents     *consent.Service
	care         *appointment.Facts
	accounts     *auth.Service
	patients     *patient.Service
}

// invoiceRoutes are the receipts' (docs/plans/clinic-cms.md, package C): issued, reversed, listed, printed and
// reported by the clinic, read by their patient. The api reports nothing to the tax authority: the worker's
// invoice_fiscalize job does, enqueued in the receipt's transaction.
func invoiceRoutes(p invoicePorts, log *slog.Logger) func(*http.ServeMux) {
	svc := invoice.NewService(invoice.Ports{
		Repo: invoice.NewPostgresRepository(p.pool), Jobs: invoice.NewRiverJobs(p.jobs), Audit: p.audit, Tx: p.tx, Clock: clock.System{},
		// The price list is package B's: until it merges, this service refuses to price (errors.ErrUnsupported) and no
		// receipt can be issued. The integration passes the price list's service built for its own routes.
		Prices:  chargeitemdefinition.NewService(nil, clock.System{}),
		Sellers: clinicSeller{p.clinics}, Clinics: p.clinics, Visits: p.appointments, Consents: p.consents, Care: p.care,
	})
	return invoice.NewHTTPHandler(svc, invoiceCaller(p.accounts, p.patients), log).Routes
}

// clinicSeller reads the seller a receipt names from the clinic: its legal name and tax id (the billing details,
// package D) and its address (package A). Neither is on organization.Organization before those packages merge, so
// until then every clinic's details are incomplete and issuing is refused with billing_details_missing.
type clinicSeller struct{ clinics *organization.Service }

func (s clinicSeller) Seller(ctx context.Context, org string) (invoice.Seller, error) {
	if _, err := s.clinics.Clinic(ctx, org); err != nil {
		return invoice.Seller{}, err
	}
	return invoice.Seller{}, nil
}

// invoiceCaller is the receipts' view of the signed-in session: for staff, whether the account is the clinic's admin
// and the practitioner it is; for a patient, their profile's id (empty before the profile exists).
func invoiceCaller(accounts *auth.Service, patients *patient.Service) func(ctx context.Context) (invoice.Caller, bool, error) {
	caller := appointmentCaller(accounts, patients)
	return func(ctx context.Context) (invoice.Caller, bool, error) {
		c, ok, err := caller(ctx)
		return invoice.Caller{
			AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID, PatientID: c.PatientID, Admin: c.Admin,
		}, ok, err
	}
}
