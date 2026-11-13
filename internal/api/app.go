// Package api builds the API binary's dependencies and handler: the shared composition root.
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/api/router"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/history"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/internal/suggestion"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/internal/transcript"
	"github.com/Shmeeshmee/salus/internal/trends"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/llm"
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
	chat    *redis.Hub
}

// minRateLimitKey is the shortest RATE_LIMIT_KEY accepted: as long as the HMAC-SHA256 output.
const minRateLimitKey = 32

// New connects to the dependencies and builds the handler; lk is the video server (config.LoadLiveKit), files
// the object store and the key-encryption key of the questionnaire attachments (config.LoadObjectStore), drafter
// the language model that drafts documents (config.LoadLLM's engine), summarizer the one that summarises a long chat
// (config.LoadChatSummary's: the fake).
func New(ctx context.Context, cfg config.Config, lk config.LiveKit, files config.ObjectStore, drafter llm.Drafter, summarizer llm.Summarizer, log *slog.Logger) (*App, error) {
	k, err := checkKeys(cfg, files)
	if err != nil {
		return nil, err
	}
	box, kek, fhirBase := k.box, k.kek, k.fhirBase
	pool, rdb, jobs, err := connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{})
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clock.System{},
		terminology.NewCatalog(terminology.NewPostgresCatalogRepository(pool))) // the location's check and city name
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
		Jobs: appointment.NewRiverJobs(jobs), Stats: clinicstats.NewRiverQueue(jobs), // in the write's transaction
	})
	accounts := auth.NewService(auth.Ports{
		Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{},
		Challenges: auth.NewRedisChallengeStore(rdb), Counters: auth.NewRedisCounters(rdb), Box: box,
		Jobs: auth.NewRiverJobs(jobs), Practitioners: practitioners, PractitionerRoles: roles,
		RateLimitKey: cfg.RateLimitKey, Hasher: auth.NewArgon2Hasher(auth.DefaultArgon2Params(), auth.MaxConcurrentHashes),
	})
	authHandler := auth.NewHandler(accounts, log, cfg.TrustedProxies).WithFeatures(features(cfg.Analytics))
	encounters, sessions := newConsultation(pool, lk, jobs, audit, tx, appointments, consents, patients, practitioners)
	consultAPI, consultStream, documents := newConsultRecord(consultPorts{pool: pool, rdb: rdb, audit: audit, tx: tx, encounters: encounters, consents: consents, clinics: clinics, jobs: jobs, files: files, kek: kek, drafter: drafter}, appointmentCaller(accounts, patients), fhirBase, log)
	questionnaireAPI, streams, timeline := questionnaireRoutes(questionnairePorts{
		pool: pool, audit: audit, tx: tx, jobs: jobs, roles: roles, clinics: clinics, accounts: accounts, appointments: appointments, patients: patients,
		store: binary.NewS3Store(&http.Client{Timeout: binary.UploadDeadline}, files.Endpoint, files.Region, files.Bucket, files.AccessKeyID, files.SecretAccessKey, s3PageSize),
		kek:   kek,
	}, fhirBase, log)
	chatAPI, chatStream, chatHub := newChat(chatPorts{db: pool, rdb: rdb, audit: audit, tx: tx, consents: consents, jobs: jobs, files: newChatFiles(pool, files, kek, jobs, tx), language: uiLanguage(accounts),
		visits: appointment.NewFacts(appointmentRepo), roles: roles, encounters: encounters, caller: appointmentCaller(accounts, patients), active: authHandler.Active, summarizer: summarizer}, log)
	entries := newProfileEntries(profilePorts{pool, audit, tx, consents, appointmentRepo, roles, encounters, accounts, patients}, fhirBase, log)
	h := router.New(router.Config{
		Logger:  log,
		Checks:  readiness(pool, rdb),
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
			history.NewHTTPHandler(history.NewService(history.Ports{Visits: encounters, Documents: documents, Responses: timeline.Responses, Files: timeline.Files, Clinics: clinics, Roles: roles, Audit: audit, Tx: tx}), appointmentCaller(accounts, patients), log).Routes,
		}, slices.Concat(questionnaireAPI, consultAPI, terminologyRoutes(pool, audit, tx, accounts, log), []func(*http.ServeMux){chatAPI, clinicStatsRoutes(pool, audit, tx, clinicStatsCaller(accounts, roles), log)}, entries.routes, chatNotificationRoutes(chatNotificationPorts{pool: pool, audit: audit, tx: tx, consents: consents, accounts: accounts, patients: patients, box: k.webhooks}, log), platformRoutes(cfg.Analytics, pool, audit, tx, clinics, accounts, log))...),
		Streams: append(streams, consultStream, chatStream),
		Timeout: RequestTimeout,
	})
	return &App{Handler: h, pool: pool, redis: rdb, chat: chatHub}, nil
}

// connect opens the database, Redis and the job inserter (invites and bookings enqueue their e-mail in their
// transaction); on an error nothing stays open.
func connect(ctx context.Context, cfg config.Config) (*pgxpool.Pool, *goredis.Client, *river.Client[pgx.Tx], error) {
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, nil, err
	}
	rdb, err := redis.Open(ctx, cfg.RedisURL)
	if err != nil {
		pool.Close()
		return nil, nil, nil, err
	}
	jobs, err := worker.NewInserter(pool)
	if err != nil {
		pool.Close()
		_ = rdb.Close()
		return nil, nil, nil, err
	}
	return pool, rdb, jobs, nil
}

// profilePorts are what the allergies and the medications of the patient's profile are built on.
type profilePorts struct {
	pool         *pgxpool.Pool
	audit        *auditevent.Service
	tx           *postgres.Transactor
	consents     *consent.Service
	appointments appointment.Repository
	roles        *practitionerrole.Service
	encounters   *encounter.Service
	accounts     *auth.Service
	patients     *patient.Service
}

// profileEntries are the allergies and the current medications of the patient's profile (SAL-41,
// docs/plans/patient-history.md), which the history reads through their Seen, and their routes.
type profileEntries struct {
	allergies   *allergyintolerance.Service
	medications *medicationstatement.Service
	routes      []func(*http.ServeMux)
}

// newProfileEntries builds the profileEntries.
func newProfileEntries(p profilePorts, fhirBase string, log *slog.Logger) profileEntries {
	visits, caller := appointment.NewFacts(p.appointments), appointmentCaller(p.accounts, p.patients)
	terms := terminology.NewService(terminology.NewPostgresRepository(p.pool), uiLanguage(p.accounts))
	allergies := allergyintolerance.NewService(allergyintolerance.Ports{
		Repo: allergyintolerance.NewPostgresRepository(p.pool), Consents: p.consents, Visits: visits, Roles: p.roles,
		Encounters: p.encounters, Terms: terms, Audit: p.audit, Tx: p.tx, Clock: clock.System{},
	})
	medications := medicationstatement.NewService(medicationstatement.Ports{
		Repo: medicationstatement.NewPostgresRepository(p.pool), Consents: p.consents, Visits: visits, Roles: p.roles,
		Encounters: p.encounters, Terms: terms, Audit: p.audit, Tx: p.tx, Clock: clock.System{},
	})
	return profileEntries{allergies: allergies, medications: medications, routes: []func(*http.ServeMux){
		allergyintolerance.NewHTTPHandler(allergies, caller, log).Routes,
		allergyintolerance.NewFHIRHandler(allergies, caller, fhirBase, log).Routes,
		medicationstatement.NewHTTPHandler(medications, caller, log).Routes,
		medicationstatement.NewFHIRHandler(medications, caller, fhirBase, log).Routes,
	}}
}

// readiness are the dependencies /readyz pings.
func readiness(pool *pgxpool.Pool, rdb *goredis.Client) []router.Check {
	return []router.Check{
		{Name: "postgres", Ping: pool.Ping},
		{Name: "redis", Ping: func(ctx context.Context) error { return rdb.Ping(ctx).Err() }},
	}
}

// keys are the settings New checks before it connects to anything.
type keys struct {
	box      *auth.AESSecretBox
	webhooks *auth.AESSecretBox // WEBHOOK_KEY: a key of its own for the clinics' webhook secrets
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
	webhooks, err := auth.NewAESSecretBox(cfg.WebhookKey)
	if err != nil {
		return keys{}, errors.New("api: WEBHOOK_KEY must be 32 bytes")
	}
	if len(cfg.RateLimitKey) < minRateLimitKey {
		return keys{}, errors.New("api: RATE_LIMIT_KEY must be at least 32 bytes")
	}
	fhirBase, err := absoluteURL(cfg.FHIRBaseURL)
	if err != nil {
		return keys{}, err
	}
	return keys{box: box, webhooks: webhooks, kek: kek, fhirBase: fhirBase}, nil
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

// consultPorts are what the consultation's record is built on: its transcript and its documents.
type consultPorts struct {
	drafter    llm.Drafter // config.LoadLLM's engine (cmd/api)
	pool       *pgxpool.Pool
	rdb        *goredis.Client
	audit      *auditevent.Service
	tx         *postgres.Transactor
	encounters *encounter.Service
	consents   *consent.Service
	clinics    *organization.Service
	jobs       *river.Client[pgx.Tx]
	kek        *envelope.KEK
	files      config.ObjectStore
}

// newConsultRecord builds the encounter's live events (its Redis Stream), its stored transcript, and its
// documents (templates, drafting, signing, the signed PDF and their FHIR reads), for its practitioner; it
// returns their routes and the stream's, with the suggestions' statuses the practitioner sets on the stream, and the
// documents (a source of the patient's timeline).
func newConsultRecord(p consultPorts, caller func(ctx context.Context) (appointment.Caller, bool, error), fhirBase string, log *slog.Logger) (routes []func(*http.ServeMux), stream func(*http.ServeMux), documents *composition.Service) {
	terms := terminology.NewService(terminology.NewPostgresRepository(p.pool), nil) // the tagging only (SAL-13)
	transcripts := transcript.NewService(transcript.Ports{
		Repo: transcript.NewPostgresRepository(p.pool), Stream: transcript.NewRedisStream(p.rdb), Encounters: p.encounters, Consents: p.consents,
		Sharing: p.clinics, Audit: p.audit, Tx: p.tx, Terms: terms,
	})
	documents = composition.NewService(composition.Ports{
		Repo: composition.NewPostgresRepository(p.pool), Encounters: p.encounters, Transcripts: transcripts, Drafter: p.drafter,
		Store:     binary.NewS3Store(&http.Client{Timeout: binary.UploadDeadline}, p.files.Endpoint, p.files.Region, p.files.Bucket, p.files.AccessKeyID, p.files.SecretAccessKey, s3PageSize),
		Publisher: transcript.NewRedisPublisher(p.rdb), Jobs: composition.NewRiverJobs(p.jobs), // a signature enqueues the patient's e-mail
		Stats:      clinicstats.NewRiverQueue(p.jobs),                             // and, with diagnoses, its clinic-day's statistics (SAL-20)
		Conditions: condition.NewService(condition.NewPostgresRepository(p.pool)), // the diagnoses of a signature (SAL-19)
		Audit:      p.audit, Tx: p.tx, Clock: clock.System{}, KEK: p.kek, Log: log, Tagger: terms,
	})
	suggestions := suggestion.NewService(suggestion.Ports{
		Repo: suggestion.NewPostgresRepository(p.pool), Encounters: p.encounters, Consents: p.consents, Records: transcripts,
		Publisher: suggestion.NewRedisPublisher(p.rdb), Audit: p.audit, Tx: p.tx,
	})
	transcriptHandler := transcript.NewHTTPHandler(transcripts, suggestion.NewCommandHandler(suggestions, log), caller, clock.System{}, log)
	return []func(*http.ServeMux){
		transcriptHandler.Routes,
		composition.NewHTTPHandler(documents, caller, log).Routes,
		composition.NewFHIRHandler(documents, caller, fhirBase, log).Routes,
		condition.NewFHIRHandler(condition.NewReads(condition.ReadPorts{
			Repo: condition.NewPostgresReadRepository(p.pool), Encounters: p.encounters, Audit: p.audit, Tx: p.tx,
		}), caller, log).Routes,
	}, transcriptHandler.StreamRoutes, documents
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
	clinics      *organization.Service
	accounts     *auth.Service
	appointments *appointment.Service
	patients     *patient.Service
	store        binary.Store
	kek          *envelope.KEK
}

// questionnaireRoutes builds the clinic's questionnaire library, the patients' responses and the files
// attached to them, and returns their JSON and FHIR routes, and the files' streaming routes. The responses are
// built first: they are the files' holder (docs/attachments/SPEC.md §2 "Dependency direction"). The responses and
// the files are sources of the patient's timeline too (timeline).
func questionnaireRoutes(p questionnairePorts, fhirBase string, log *slog.Logger) (routes, streams []func(*http.ServeMux), timeline history.Ports) {
	questionnaires := questionnaire.NewService(questionnaire.Ports{
		Repo: questionnaire.NewPostgresRepository(p.pool), Audit: p.audit, Tx: p.tx, Clock: clock.System{}, Roles: p.roles,
		Clinics: p.clinics,
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
	}, []func(*http.ServeMux){filesHandler.StreamRoutes}, history.Ports{Responses: responses, Files: files}
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

// clinicStatsRoutes are the clinic's own statistics' (SAL-20), read on pool (salus_app) in the caller's clinic.
func clinicStatsRoutes(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor,
	caller func(ctx context.Context) (clinicstats.Caller, bool, error), log *slog.Logger,
) func(*http.ServeMux) {
	svc := clinicstats.NewService(clinicstats.Ports{Repo: clinicstats.NewPostgresRepository(pool), Audit: audit, Tx: tx})
	return clinicstats.NewHTTPHandler(svc, caller, log).Routes
}

// clinicStatsCaller is the clinic statistics' view of the signed-in session: for staff, whether the account is the
// clinic's admin and whether it is a practitioner there with their role, as read from the database (the session
// holds neither).
func clinicStatsCaller(accounts *auth.Service, roles *practitionerrole.Service) func(ctx context.Context) (clinicstats.Caller, bool, error) {
	return func(ctx context.Context) (clinicstats.Caller, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return clinicstats.Caller{}, false, nil
		}
		c := clinicstats.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID}
		if c.OrganizationID == "" {
			return c, true, nil
		}
		m, err := accounts.ClinicMember(ctx, s)
		if err != nil {
			return clinicstats.Caller{}, false, err
		}
		c.Admin, c.Practitioner = m.Admin, m.PractitionerID != ""
		if !c.Practitioner {
			return c, true, nil
		}
		role, err := roles.OfPractitioner(ctx, c.OrganizationID, m.PractitionerID)
		switch {
		case errors.Is(err, practitionerrole.ErrNotFound):
		case err != nil:
			return clinicstats.Caller{}, false, err
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

// terminologyRoutes are the term list's: the cheatsheet's search, the review panel, and the diagnosis groups and
// places of the analytics (docs/plans/analytics.md §3).
func terminologyRoutes(pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor, accounts *auth.Service, log *slog.Logger) []func(*http.ServeMux) {
	return []func(*http.ServeMux){
		terminology.NewHTTPHandler(terminology.NewService(terminology.NewPostgresRepository(pool), uiLanguage(accounts)), terminologyCaller(accounts), log).Routes,
		terminology.NewReviewHandler(terminology.NewReviews(terminology.ReviewPorts{
			Repo: terminology.NewPostgresReviewRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{},
		}), terminologyCaller(accounts), log).Routes,
		terminology.NewCatalogHandler(terminology.NewCatalog(terminology.NewPostgresCatalogRepository(pool)), terminologyCaller(accounts), log).Routes,
		terminology.NewGroupReviewHandler(terminology.NewGroupReviews(terminology.GroupReviewPorts{
			Repo: terminology.NewPostgresGroupReviewRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{},
		}), terminologyCaller(accounts), log).Routes,
	}
}

// terminologyCaller is the terminology's view of the signed-in session: whether the account is staff of the
// session's clinic (its admin or a practitioner) and a reviewer of the term list, as read from the database. A
// patient's session has no clinic and needs no read.
func terminologyCaller(accounts *auth.Service) func(ctx context.Context) (terminology.Caller, bool, error) {
	return func(ctx context.Context) (terminology.Caller, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return terminology.Caller{}, false, nil
		}
		c := terminology.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID}
		if s.OrganizationID == "" {
			return c, true, nil
		}
		m, err := accounts.ClinicMember(ctx, s)
		if err != nil {
			return terminology.Caller{}, false, err
		}
		c.Staff = m.Admin || m.PractitionerID != ""
		c.Reviewer = c.Staff && m.TermReviewer
		return c, true, nil
	}
}

// uiLanguage is an account's UI language for the terminology's default languages: its own choice, else the one
// its client detected, else none. Read (and audited) through the account service.
func uiLanguage(accounts *auth.Service) func(ctx context.Context, accountID string) (string, error) {
	return func(ctx context.Context, accountID string) (string, error) {
		a, err := accounts.Me(ctx, accountID)
		if err != nil {
			return "", err
		}
		if a.Language != "" {
			return a.Language, nil
		}
		return a.DetectedLanguage, nil
	}
}

// features are the codes GET /auth/me lists: analytics_platform when the platform view is on (§2a).
func features(a config.Analytics) []string {
	if a.PlatformEnabled {
		return []string{"analytics_platform"}
	}
	return nil
}

// platformRoutes are the All of Salus view's routes (docs/plans/analytics.md §5, §8), none while
// ANALYTICS_PLATFORM_ENABLED is off (§2a): its paths then answer 404.
func platformRoutes(a config.Analytics, pool *pgxpool.Pool, audit *auditevent.Service, tx *postgres.Transactor,
	clinics *organization.Service, accounts *auth.Service, log *slog.Logger,
) []func(*http.ServeMux) {
	if !a.PlatformEnabled {
		return nil
	}
	codes := make([]string, 0, len(practitionerrole.Specialties()))
	for _, s := range practitionerrole.Specialties() {
		codes = append(codes, s.Code)
	}
	view := trends.NewService(trends.Ports{
		Repo: trends.NewPostgresRepository(pool), Clinics: clinics, Catalog: terminology.NewCatalog(terminology.NewPostgresCatalogRepository(pool)),
		Audit: audit, Tx: tx, Specialties: codes, K: a.MinCount,
	})
	return []func(*http.ServeMux){trends.NewHandler(view, trendsCaller(accounts), log).Routes}
}

// trendsCaller is the platform view's view of the signed-in session: whether the account is staff of the session's
// clinic (its admin or a practitioner), as read from the database.
func trendsCaller(accounts *auth.Service) func(ctx context.Context) (trends.Caller, bool, error) {
	return func(ctx context.Context) (trends.Caller, bool, error) {
		s, ok := auth.SessionFrom(ctx)
		if !ok {
			return trends.Caller{}, false, nil
		}
		c := trends.Caller{AccountID: s.AccountID, OrganizationID: s.OrganizationID}
		if s.OrganizationID == "" {
			return c, true, nil
		}
		m, err := accounts.ClinicMember(ctx, s)
		if err != nil {
			return trends.Caller{}, false, err
		}
		c.Staff = m.Admin || m.PractitionerID != ""
		return c, true, nil
	}
}

// Close releases the dependencies.
func (a *App) Close() {
	_ = a.chat.Close()
	_ = a.redis.Close()
	a.pool.Close()
}
