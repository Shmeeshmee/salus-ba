package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// priceListRoutes are the clinic's price list and the prices shown at booking (docs/plans/clinic-cms.md, package B).
// Prices are never charged in the app: appointment.Prices stays nil.
func priceListRoutes(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor, clinics *organization.Service,
	accounts *auth.Service, log *slog.Logger,
) func(*http.ServeMux) {
	svc := chargeitemdefinition.NewService(chargeitemdefinition.Ports{
		Repo: chargeitemdefinition.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}, Clinics: clinics,
	})
	clinicCaller := organizationCaller(accounts)
	caller := func(ctx context.Context) (chargeitemdefinition.Caller, bool, error) {
		c, ok, err := clinicCaller(ctx)
		return chargeitemdefinition.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, Admin: c.Admin}, ok, err
	}
	return chargeitemdefinition.NewHandler(svc, caller, log).Routes
}
