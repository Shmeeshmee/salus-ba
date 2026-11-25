package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// locations is the clinic's sites with their rooms and devices (docs/clinic-cms/WAVE6.md U1, U3, U4).
func locations(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor) *location.Service {
	return location.NewService(location.Ports{Repo: location.NewPostgresRepository(pool), Tx: tx, Audit: audit, Clock: clock.System{}})
}

// roomRoutes are a location's rooms and devices (package RM1): the clinic's staff read them, the desk included; only
// its admin writes.
func roomRoutes(svc *location.Service, accounts *auth.Service, log *slog.Logger) func(*http.ServeMux) {
	clinicCaller := organizationCaller(accounts)
	caller := func(ctx context.Context) (location.Staff, bool, error) {
		c, ok, err := clinicCaller(ctx)
		return location.Staff{AccountID: c.AccountID, OrganizationID: c.OrganizationID, Admin: c.Admin}, ok, err
	}
	return location.NewRoomHandler(svc, caller, log).Routes
}
