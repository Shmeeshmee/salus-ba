// Package api builds the API binary's dependencies and handler: the shared composition root.
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Shmeeshmee/salus/internal/api/router"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/redis"
)

// RequestTimeout bounds every request's handler.
const RequestTimeout = 30 * time.Second

// App is the built API.
type App struct {
	Handler http.Handler
	pool    *pgxpool.Pool
	redis   *goredis.Client
}

// minRateLimitKey is the shortest RATE_LIMIT_KEY accepted: as long as the HMAC-SHA256 output.
const minRateLimitKey = 32

// New connects to the dependencies and builds the handler.
func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	box, err := auth.NewAESSecretBox(cfg.TOTPKey)
	if err != nil {
		return nil, err
	}
	if len(cfg.RateLimitKey) < minRateLimitKey {
		return nil, errors.New("api: RATE_LIMIT_KEY must be at least 32 bytes")
	}
	fhirBase, err := absoluteURL(cfg.FHIRBaseURL)
	if err != nil {
		return nil, err
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	rdb, err := redis.Open(ctx, cfg.RedisURL)
	if err != nil {
		pool.Close()
		return nil, err
	}
	jobs, err := worker.NewInserter(pool) // invites enqueue their e-mail in their transaction
	if err != nil {
		pool.Close()
		_ = rdb.Close()
		return nil, err
	}
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{})
	hasher := auth.NewArgon2Hasher(auth.DefaultArgon2Params(), auth.MaxConcurrentHashes)
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clock.System{})
	practitioners := practitioner.NewService(practitioner.NewPostgresRepository(pool), audit, tx)
	roles := practitionerrole.NewService(practitionerrole.NewPostgresRepository(pool), audit, tx, clock.System{})
	consents := consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clock.System{}, consent.NewRiverJobs(jobs))
	appointmentRepo := appointment.NewPostgresRepository(pool)
	// The patient's appointment facts come from the appointment repository alone, so the patient service can
	// be built before the appointment service, which reads profiles through it.
	patients := patient.NewService(patient.NewPostgresRepository(pool), audit, tx, clock.System{}, consents, appointment.NewFacts(appointmentRepo))
	schedules := schedule.NewService(schedule.Ports{
		Repo: schedule.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{},
		Roles: roles, Jobs: schedule.NewRiverJobs(jobs), // an edit enqueues its slot generation in its transaction
	})
	slots := slot.NewService(slot.NewPostgresRepository(pool), tx, clock.System{}, schedules, clinics)
	appointments := appointment.NewService(appointment.Ports{
		Repo: appointmentRepo, Audit: audit, Tx: tx, Clock: clock.System{},
		Consents: consents, Slots: slots, Patients: patients, Clinics: clinics, Roles: roles,
		Jobs: appointment.NewRiverJobs(jobs), // a booking or cancellation enqueues its e-mails in its transaction
	})
	accounts := auth.NewService(auth.Ports{
		Repo: auth.NewPostgresRepository(pool), Hasher: hasher, Audit: audit, Tx: tx, Clock: clock.System{},
		Challenges: auth.NewRedisChallengeStore(rdb), Counters: auth.NewRedisCounters(rdb), Box: box,
		Jobs: auth.NewRiverJobs(jobs), Practitioners: practitioners, PractitionerRoles: roles,
		RateLimitKey: cfg.RateLimitKey,
	})
	authHandler := auth.NewHandler(accounts, log, cfg.TrustedProxies)

	h := router.New(router.Config{
		Logger: log,
		Checks: []router.Check{
			{Name: "postgres", Ping: pool.Ping},
			{Name: "redis", Ping: func(ctx context.Context) error { return rdb.Ping(ctx).Err() }},
		},
		Session: authHandler.Session,
		Routes: []func(*http.ServeMux){
			authHandler.Routes,
			organization.NewHandler(clinics, organizationCaller(accounts), fhirBase, log).Routes,
			practitioner.NewHandler(practitioners, practitionerCaller, log).Routes,
			practitionerrole.NewHandler(roles, practitioners, roleCaller, fhirBase, log).Routes,
			practitionerrole.NewHTTPHandler(roles, roleAdminCaller(accounts), log).Routes,
			patient.NewHandler(patients, patientCaller, log).Routes,
			patient.NewFHIRHandler(patients, patientReadCaller(accounts, roles), log).Routes,
			consent.NewHandler(consents, consentCaller(patients), log).Routes,
			consent.NewFHIRHandler(consents, consentCaller(patients), log).Routes,
			schedule.NewHTTPHandler(schedules, scheduleCaller(accounts), log).Routes,
			schedule.NewHandler(schedules, fhirBase, log).Routes,
			slot.NewHandler(slots, fhirBase, log).Routes,
			appointment.NewHTTPHandler(appointments, appointmentCaller(accounts, patients), log).Routes,
			appointment.NewHandler(appointments, appointmentCaller(accounts, patients), fhirBase, log).Routes,
		},
		Timeout: RequestTimeout,
	})
	return &App{Handler: h, pool: pool, redis: rdb}, nil
}

// absoluteURL is FHIR_BASE_URL without a trailing slash, or an error when it is not an absolute http(s) URL.
func absoluteURL(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("api: FHIR_BASE_URL must be an absolute http(s) URL")
	}
	return strings.TrimSuffix(s, "/"), nil
}

// organizationCaller is the directory's view of the signed-in session, with whether the account is the
// clinic's admin as read from the database (the session does not hold the role).
func organizationCaller(accounts *auth.Service) func(ctx context.Context) (organization.Caller, bool, error) {
	return func(ctx context.Context) (organization.Caller, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return organization.Caller{}, false, nil
		}
		admin, err := accounts.IsClinicAdmin(ctx, s)
		if err != nil {
			return organization.Caller{}, false, err
		}
		return organization.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID, Admin: admin}, true, nil
	}
}

func practitionerCaller(ctx context.Context) (practitioner.Caller, bool) {
	s, ok := auth.SessionFrom(ctx)
	return practitioner.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID}, ok
}

func roleCaller(ctx context.Context) (practitionerrole.Caller, bool) {
	s, ok := auth.SessionFrom(ctx)
	return practitionerrole.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID}, ok
}

// roleAdminCaller is the role domain's view of the signed-in session, with whether the account is the clinic's
// admin as read from the database, for the actions only an admin may take.
func roleAdminCaller(accounts *auth.Service) func(ctx context.Context) (practitionerrole.Caller, bool, error) {
	return func(ctx context.Context) (practitionerrole.Caller, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return practitionerrole.Caller{}, false, nil
		}
		admin, err := accounts.IsClinicAdmin(ctx, s)
		if err != nil {
			return practitionerrole.Caller{}, false, err
		}
		return practitionerrole.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID, Admin: admin}, true, nil
	}
}

// scheduleCaller is the schedule domain's view of the signed-in session, with whether the account is the
// clinic's admin and which practitioner it is, as read from the database (the session holds neither).
func scheduleCaller(accounts *auth.Service) func(ctx context.Context) (schedule.Caller, bool, error) {
	return func(ctx context.Context) (schedule.Caller, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return schedule.Caller{}, false, nil
		}
		m, err := accounts.ClinicMember(ctx, s)
		if err != nil {
			return schedule.Caller{}, false, err
		}
		return schedule.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID, PractitionerID: m.PractitionerID, Admin: m.Admin}, true, nil
	}
}

func patientCaller(ctx context.Context) (patient.Caller, bool) {
	s, ok := auth.SessionFrom(ctx)
	return patient.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID}, ok
}

// patientReadCaller is the patient domain's view of the signed-in session for a FHIR read: for staff, with
// whether the account is the clinic's admin and, for a practitioner, their role in the clinic, as read from
// the database (the session holds neither; access compares the role with the patient's appointments).
func patientReadCaller(accounts *auth.Service, roles *practitionerrole.Service) func(ctx context.Context) (patient.Caller, bool, error) {
	return func(ctx context.Context) (patient.Caller, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return patient.Caller{}, false, nil
		}
		c := patient.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID}
		if c.OrganizationID == "" {
			return c, true, nil
		}
		m, err := accounts.ClinicMember(ctx, s)
		if err != nil {
			return patient.Caller{}, false, err
		}
		c.Admin = m.Admin
		if m.PractitionerID == "" {
			return c, true, nil
		}
		role, err := roles.OfPractitioner(ctx, c.OrganizationID, m.PractitionerID)
		switch {
		case errors.Is(err, practitionerrole.ErrNotFound):
		case err != nil:
			return patient.Caller{}, false, err
		default:
			c.PractitionerRoleID = role.ID
		}
		return c, true, nil
	}
}

// consentCaller is the consent domain's view of the signed-in session: for a patient, with their profile's
// id, read (and audited) through the patient service; empty before the profile exists.
func consentCaller(patients *patient.Service) func(ctx context.Context) (consent.Caller, bool, error) {
	return func(ctx context.Context) (consent.Caller, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return consent.Caller{}, false, nil
		}
		c := consent.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID}
		if c.OrganizationID != "" {
			return c, true, nil
		}
		p, err := patients.Me(ctx, patient.Caller{AccountID: s.AccountID})
		switch {
		case errors.Is(err, patient.ErrNotFound):
			return c, true, nil
		case err != nil:
			return consent.Caller{}, false, err
		}
		c.PatientID = p.ID
		return c, true, nil
	}
}

// appointmentCaller is the appointment domain's view of the signed-in session: for staff, with whether the
// account is the clinic's admin and which practitioner it is, as read from the database; for a patient, with
// their profile's id, read (and audited) through the patient service, empty before the profile exists.
func appointmentCaller(accounts *auth.Service, patients *patient.Service) func(ctx context.Context) (appointment.Caller, bool, error) {
	return func(ctx context.Context) (appointment.Caller, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return appointment.Caller{}, false, nil
		}
		c := appointment.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID}
		if c.OrganizationID != "" {
			m, err := accounts.ClinicMember(ctx, s)
			if err != nil {
				return appointment.Caller{}, false, err
			}
			c.PractitionerID, c.Admin = m.PractitionerID, m.Admin
			return c, true, nil
		}
		p, err := patients.Me(ctx, patient.Caller{AccountID: s.AccountID})
		switch {
		case errors.Is(err, patient.ErrNotFound):
			return c, true, nil
		case err != nil:
			return appointment.Caller{}, false, err
		}
		c.PatientID = p.ID
		return c, true, nil
	}
}

// Close releases the dependencies.
func (a *App) Close() {
	_ = a.redis.Close()
	a.pool.Close()
}
