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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/api/router"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
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

// New connects to the dependencies and builds the handler; lk is the video server (config.LoadLiveKit), files
// the object store and the key-encryption key of the questionnaire attachments (config.LoadObjectStore).
func New(ctx context.Context, cfg config.Config, lk config.LiveKit, files config.ObjectStore, log *slog.Logger) (*App, error) {
	k, err := checkKeys(cfg, files)
	if err != nil {
		return nil, err
	}
	box, kek, fhirBase := k.box, k.kek, k.fhirBase
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
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clock.System{})
	practitioners := practitioner.NewService(practitioner.NewPostgresRepository(pool), audit, tx)
	roles := practitionerrole.NewService(practitionerrole.Ports{
		Repo: practitionerrole.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}, Clinics: clinics, People: practitioners,
	})
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
		Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{},
		Challenges: auth.NewRedisChallengeStore(rdb), Counters: auth.NewRedisCounters(rdb), Box: box,
		Jobs: auth.NewRiverJobs(jobs), Practitioners: practitioners, PractitionerRoles: roles,
		RateLimitKey: cfg.RateLimitKey, Hasher: auth.NewArgon2Hasher(auth.DefaultArgon2Params(), auth.MaxConcurrentHashes),
	})
	authHandler := auth.NewHandler(accounts, log, cfg.TrustedProxies)
	encounters, sessions := newConsultation(pool, lk, jobs, audit, tx, appointments, consents, patients, practitioners)
	questionnaireAPI, streams := questionnaireRoutes(questionnairePorts{
		pool: pool, audit: audit, tx: tx, jobs: jobs, roles: roles, accounts: accounts, appointments: appointments, patients: patients,
		store: binary.NewS3Store(&http.Client{Timeout: binary.UploadDeadline}, files.Endpoint, files.Region, files.Bucket, files.AccessKeyID, files.SecretAccessKey, s3PageSize),
		kek:   kek,
	}, fhirBase, log)

	h := router.New(router.Config{
		Logger: log,
		Checks: []router.Check{
			{Name: "postgres", Ping: pool.Ping},
			{Name: "redis", Ping: func(ctx context.Context) error { return rdb.Ping(ctx).Err() }},
		},
		Session: authHandler.Session,
		Routes: append([]func(*http.ServeMux){
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
			videosession.NewHandler(sessions, appointmentCaller(accounts, patients), log).Routes,
			encounter.NewHandler(encounters, appointmentCaller(accounts, patients), fhirBase, log).Routes,
		}, questionnaireAPI...),
		Streams: streams,
		Timeout: RequestTimeout,
	})
	return &App{Handler: h, pool: pool, redis: rdb}, nil
}

// keys are the settings New checks before it connects to anything.
type keys struct {
	box      *auth.AESSecretBox
	kek      *envelope.KEK
	fhirBase string
}

// checkKeys checks the keys and the FHIR base of cfg and files; an error names the variable, never a value.
func checkKeys(cfg config.Config, files config.ObjectStore) (keys, error) {
	box, err := auth.NewAESSecretBox(cfg.TOTPKey)
	if err != nil {
		return keys{}, err
	}
	kek, err := envelope.NewKEK(files.KEKID, files.KEK)
	if err != nil {
		return keys{}, errors.New("api: BINARY_KEK must be 32 bytes")
	}
	if len(cfg.RateLimitKey) < minRateLimitKey {
		return keys{}, errors.New("api: RATE_LIMIT_KEY must be at least 32 bytes")
	}
	fhirBase, err := absoluteURL(cfg.FHIRBaseURL)
	if err != nil {
		return keys{}, err
	}
	return keys{box: box, kek: kek, fhirBase: fhirBase}, nil
}

// newConsultation builds the consultation's Encounter and its video sessions on LiveKit (lk), which start
// and finish it; the first join enqueues the window's close in its transaction.
func newConsultation(pool *pgxpool.Pool, lk config.LiveKit, jobs *river.Client[pgx.Tx], audit *auditevent.Service, tx *postgres.Transactor,
	appointments *appointment.Service, consents *consent.Service, patients *patient.Service, practitioners *practitioner.Service,
) (*encounter.Service, *videosession.Service) {
	encounters := encounter.NewService(encounter.Ports{
		Repo: encounter.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}, Appointments: appointments,
	})
	return encounters, videosession.NewService(videosession.Ports{
		Repo: videosession.NewPostgresRepository(pool), Provider: videosession.NewLiveKitProvider(lk.URL, lk.APIKey, lk.APISecret),
		Jobs: videosession.NewRiverJobs(jobs), Encounters: encounters, Audit: audit, Tx: tx, Clock: clock.System{},
		Appointments: appointments, Consents: consents, Patients: patients, Practitioners: practitioners, PublicURL: lk.PublicURL,
	})
}

// s3PageSize is the object store listing's page, S3's largest.
const s3PageSize = 1000

// questionnairePorts are what the questionnaires, their responses and the responses' files are built on.
type questionnairePorts struct {
	pool         *pgxpool.Pool
	audit        *auditevent.Service
	tx           *postgres.Transactor
	jobs         *river.Client[pgx.Tx]
	roles        *practitionerrole.Service
	accounts     *auth.Service
	appointments *appointment.Service
	patients     *patient.Service
	store        binary.Store
	kek          *envelope.KEK
}

// questionnaireRoutes builds the clinic's questionnaire library, the patients' responses and the files
// attached to them, and returns their JSON and FHIR routes, and the files' streaming routes. The responses are
// built first: they are the files' holder (docs/attachments/SPEC.md §2 "Dependency direction").
func questionnaireRoutes(p questionnairePorts, fhirBase string, log *slog.Logger) (routes, streams []func(*http.ServeMux)) {
	questionnaires := questionnaire.NewService(questionnaire.Ports{
		Repo: questionnaire.NewPostgresRepository(p.pool), Audit: p.audit, Tx: p.tx, Clock: clock.System{}, Roles: p.roles,
	})
	responses := questionnaireresponse.NewService(questionnaireresponse.Ports{
		Repo: questionnaireresponse.NewPostgresRepository(p.pool), Audit: p.audit, Tx: p.tx, Clock: clock.System{},
		Appointments: p.appointments, Questionnaires: questionnaires,
		Jobs: questionnaireresponse.NewRiverJobs(p.jobs), // a withdrawal or a submit enqueues its files' discard in its transaction
	})
	files := binary.NewService(binary.Ports{
		Repo: binary.NewPostgresRepository(p.pool), Store: p.store, Holder: responses, Audit: p.audit, Tx: p.tx,
		Jobs: binary.NewRiverJobs(p.jobs), Clock: clock.System{}, KEK: p.kek, // an upload enqueues its scan in its transaction
	})
	caller, responseCaller := questionnaireCaller(p.accounts, p.patients), appointmentCaller(p.accounts, p.patients)
	filesHandler := binary.NewHTTPHandler(files, binaryCaller(responseCaller), clock.System{}, log)
	return []func(*http.ServeMux){
		questionnaire.NewHTTPHandler(questionnaires, caller, log).Routes,
		questionnaire.NewHandler(questionnaires, caller, fhirBase, log).Routes,
		questionnaireresponse.NewHTTPHandler(responses, responseCaller, log).Routes,
		questionnaireresponse.NewHandler(responses, questionnaires, responseCaller, fhirBase, log).Routes,
		filesHandler.Routes,
	}, []func(*http.ServeMux){filesHandler.StreamRoutes}
}

// binaryCaller is the files' view of the signed-in session: the appointment's (who is staff of which clinic
// and which practitioner, or which patient).
func binaryCaller(of func(ctx context.Context) (appointment.Caller, bool, error)) func(ctx context.Context) (binary.Caller, bool, error) {
	return func(ctx context.Context) (binary.Caller, bool, error) {
		c, ok, err := of(ctx)
		return binary.Caller{AccountID: c.AccountID, OrganizationID: c.OrganizationID, PractitionerID: c.PractitionerID, PatientID: c.PatientID, Admin: c.Admin}, ok, err
	}
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

// questionnaireCaller is the questionnaire domain's view of the signed-in session: for staff of the
// session's clinic, its clinic, whether it is the admin and which practitioner it is, as read from the
// database; for a patient, no clinic and their profile's id, read (and audited) through the patient service,
// empty before the profile exists.
func questionnaireCaller(accounts *auth.Service, patients *patient.Service) func(ctx context.Context) (questionnaire.Caller, bool, error) {
	return func(ctx context.Context) (questionnaire.Caller, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return questionnaire.Caller{}, false, nil
		}
		c := questionnaire.Caller{AccountID: s.AccountID}
		if s.OrganizationID == "" {
			p, err := patients.Me(ctx, patient.Caller{AccountID: s.AccountID})
			switch {
			case errors.Is(err, patient.ErrNotFound):
			case err != nil:
				return questionnaire.Caller{}, false, err
			default:
				c.PatientID = p.ID
			}
			return c, true, nil
		}
		m, err := accounts.ClinicMember(ctx, s)
		if err != nil {
			return questionnaire.Caller{}, false, err
		}
		c.PractitionerID, c.Admin = m.PractitionerID, m.Admin
		if m.Admin || m.PractitionerID != "" {
			c.OrganizationID = s.OrganizationID
		}
		return c, true, nil
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
