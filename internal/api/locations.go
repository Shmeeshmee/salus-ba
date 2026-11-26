package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// locations are the clinic's sites (docs/clinic-cms/WAVE6.md LC1): their own routes and the location of an in-person
// availability rule.
func locations(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor, clinics *organization.Service) *location.Service {
	return location.NewService(location.Ports{
		Repo: location.NewPostgresRepository(pool), Clock: clock.System{}, Audit: audit, Tx: tx, Clinics: clinics,
		Places: terminology.NewCatalog(terminology.NewPostgresCatalogRepository(pool)), // a location's city in its address
	})
}

// locationRoutes are the clinic's locations and a listed clinic's for the patient booking in person, and for the
// address of a visit at a clinic whose membership stopped to those it serves.
func locationRoutes(svc *location.Service, accounts *auth.Service, served organization.Served, log *slog.Logger) func(*http.ServeMux) {
	clinicCaller := organizationCaller(accounts)
	caller := func(ctx context.Context) (location.Staff, bool, error) {
		c, ok, err := clinicCaller(ctx)
		return location.Staff{AccountID: c.AccountID, OrganizationID: c.OrganizationID, Admin: c.Admin}, ok, err
	}
	return location.NewHandler(svc, caller, log).WithServed(served).Routes
}
