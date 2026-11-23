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
	prices       *chargeitemdefinition.Service
	accounts     *auth.Service
	patients     *patient.Service
}

// invoiceRoutes are the receipts' (docs/plans/clinic-cms.md, package C): issued, reversed, listed, printed and
// reported by the clinic, read by their patient. The api reports nothing to the tax authority: the worker's
// invoice_fiscalize job does, enqueued in the receipt's transaction.
func invoiceRoutes(p invoicePorts, log *slog.Logger) func(*http.ServeMux) {
	svc := invoice.NewService(invoice.Ports{
		Repo: invoice.NewPostgresRepository(p.pool), Jobs: invoice.NewRiverJobs(p.jobs), Audit: p.audit, Tx: p.tx, Clock: clock.System{},
		Prices: p.prices, // the price list's own service (package B)
		Sellers: clinicSeller{clinics: p.clinics, billing: organization.NewBilling(organization.BillingPorts{
			Repo: organization.NewPostgresBillingRepository(p.pool), Audit: p.audit, Tx: p.tx,
		})},
		Clinics: p.clinics, Visits: p.appointments, Consents: p.consents, Care: p.care,
	})
	return invoice.NewHTTPHandler(svc, invoiceCaller(p.accounts, p.patients), log).Routes
}

// clinicSeller reads the seller a receipt names from the clinic as it is now: its legal name and tax id from the
// billing details (package D) and its address on one line, with the city's name (package A). A clinic that has not
// stated them gives an incomplete seller, and the receipt is refused with billing_details_missing.
type clinicSeller struct {
	clinics *organization.Service
	billing *organization.Billing
}

func (s clinicSeller) Seller(ctx context.Context, org string) (invoice.Seller, error) {
	details, err := s.billing.ForClinic(ctx, org)
	if err != nil {
		return invoice.Seller{}, err
	}
	clinic, err := s.clinics.Read(ctx, organization.Caller{OrganizationID: org}, org) // Read names the city
	if err != nil {
		return invoice.Seller{}, err
	}
	return invoice.Seller{LegalName: details.LegalName, TaxID: details.TaxID, Address: clinic.Address()}, nil
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
