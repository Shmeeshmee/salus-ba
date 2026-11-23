package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/signup"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// signupPorts are what a clinic's self-signup and its setup wizard are built on (docs/clinic-cms, O1).
type signupPorts struct {
	pool           *pgxpool.Pool
	jobs           *river.Client[pgx.Tx]
	counters       auth.Counters
	audit          *auditevent.Service
	tx             *postgres.Transactor
	clinics        *organization.Service
	accounts       *auth.Service
	rateLimitKey   []byte
	trustedProxies []netip.Prefix
}

// signupRoutes are POST /signup, POST /signup/verify and the wizard's /clinic/setup. A verified signup creates its
// clinic as salusctl create-clinic does: signup.Service.CreateClinic.
func signupRoutes(p signupPorts, log *slog.Logger) func(*http.ServeMux) {
	svc := signup.NewService(signup.Ports{
		Repo: signup.NewPostgresRepository(p.pool), Jobs: signup.NewRiverJobs(p.jobs), Counters: p.counters, Tx: p.tx,
		Clock: clock.System{}, Clinics: p.clinics, Admins: p.accounts, Trials: memberships(p.pool, p.audit, p.tx),
		RateLimitKey: p.rateLimitKey,
	})
	clinicCaller := organizationCaller(p.accounts)
	caller := func(ctx context.Context) (signup.Caller, bool, error) {
		c, ok, err := clinicCaller(ctx)
		return signup.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, Admin: c.Admin}, ok, err
	}
	return signup.NewHandler(svc, caller, log, p.trustedProxies).Routes
}
