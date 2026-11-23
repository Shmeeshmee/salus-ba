package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// memberships is the clinics' membership service as the api needs it: the admin's reads and whether a clinic's new
// bookings are stopped. The api issues no invoice (the worker does), so it needs neither the clinics nor the seats.
func memberships(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor) *subscription.Service {
	return subscription.NewService(subscription.Ports{Repo: subscription.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}})
}

// bookableClinics is the clinics as a booking reads them (appointment.Clinics): a clinic whose membership is
// suspended or cancelled reads as not listed, so new bookings and waitlist entries stop (docs/plans/clinic-cms.md Q4).
// Nothing else asks here: existing appointments, records, documents, exports and subject rights keep working.
type bookableClinics struct {
	*organization.Service
	memberships *subscription.Service
}

func (b bookableClinics) Read(ctx context.Context, c organization.Caller, id string) (organization.Organization, error) {
	o, err := b.Service.Read(ctx, c, id)
	if err != nil || !o.Listed {
		return o, err
	}
	stopped, err := b.memberships.BookingsStopped(ctx, id)
	if err != nil {
		return organization.Organization{}, err
	}
	o.Listed = !stopped
	return o, nil
}

// subscriptionRoutes are the routes of package D of docs/plans/clinic-cms.md: the clinic's membership and Salus's
// invoices to it (subscription), and its billing details (organization). For the clinic's admin.
// The admin's PDF of an invoice reads the clinic, its billing details and the seller (package I).
func subscriptionRoutes(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor, accounts *auth.Service, clinics *organization.Service, seller config.Seller, log *slog.Logger) []func(*http.ServeMux) {
	clinicCaller := organizationCaller(accounts)
	caller := func(ctx context.Context) (subscription.Caller, bool, error) {
		c, ok, err := clinicCaller(ctx)
		return subscription.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, Admin: c.Admin}, ok, err
	}
	billing := organization.NewBilling(organization.BillingPorts{Repo: organization.NewPostgresBillingRepository(pool), Audit: audit, Tx: tx})
	return []func(*http.ServeMux){
		subscription.NewHTTPHandler(subscription.NewService(subscription.Ports{
			Repo: subscription.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}, Clinics: clinics, Billing: billing,
			Seller: subscription.Seller(seller),
		}), caller, log).Routes,
		organization.NewBillingHandler(billing, clinicCaller, log).Routes,
	}
}
