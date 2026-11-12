package api_test

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The consultation's join on the real database, through the services as New builds them
// (docs/consultation, package 1).

// consultationSeed is a listed clinic with one practitioner role and its practitioner's account, and a patient with an
// active consent and an appointment booked from at for half an hour. Synthetic.
type consultationSeed struct {
	clinic, role, practitioner, practitionerAccount, patient, patientAccount, appointment string
}

func seedConsultation(t testing.TB, owner *pgxpool.Pool, at time.Time) consultationSeed {
	t.Helper()
	ctx := context.Background()
	var s consultationSeed
	err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (name, listed) VALUES ('Synthetic clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id, practitioner_id),
		a AS (INSERT INTO account (email, password_hash, role, organization_id, practitioner_id)
			SELECT gen_random_uuid()::text || '@example.test', 'h', 'practitioner', organization_id, practitioner_id FROM r RETURNING id)
		SELECT r.organization_id::text, r.id::text, r.practitioner_id::text, a.id::text FROM r, a`).
		Scan(&s.clinic, &s.role, &s.practitioner, &s.practitionerAccount)
	if err != nil {
		t.Fatal(err)
	}
	err = owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
		p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id, account_id),
		c AS (INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT id, $1, 'active', $2::timestamptz - interval '30 days' FROM p),
		tc AS (INSERT INTO transcription_consent (patient_id, organization_id, status, date_time) SELECT id, $1, 'active', $2::timestamptz - interval '30 days' FROM p),
		b AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period)
			SELECT id, $3, $1, gen_random_uuid(), tstzrange($2::timestamptz, $2::timestamptz + interval '30 minutes') FROM p RETURNING id)
		SELECT p.id::text, p.account_id::text, b.id::text FROM p, b`, s.clinic, at, s.role).
		Scan(&s.patient, &s.patientAccount, &s.appointment)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// realService builds the service as internal/api would, on the salus_app pool app, at the time at, with
// the provider p.
func realService(tb testing.TB, app *pgxpool.Pool, at time.Time, p videosession.Provider) *videosession.Service {
	tb.Helper()
	clk := clock.Fixed{T: at}
	tx := postgres.NewTransactor(app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, clk)
	clinics := organization.NewService(organization.NewPostgresRepository(app), audit, tx, clk, nil)
	practitioners := practitioner.NewService(practitioner.NewPostgresRepository(app), audit, tx)
	roles := practitionerrole.NewService(practitionerrole.Ports{
		Repo: practitionerrole.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clk, Clinics: clinics, People: practitioners,
	})
	consents := consent.NewService(consent.NewPostgresRepository(app), audit, tx, clk, nil)
	appointmentRepo := appointment.NewPostgresRepository(app)
	patients := patient.NewService(patient.NewPostgresRepository(app), audit, tx, clk, consents, appointment.NewFacts(appointmentRepo))
	appointments := appointment.NewService(appointment.Ports{
		Repo: appointmentRepo, Audit: audit, Tx: tx, Clock: clk, Consents: consents, Patients: patients, Roles: roles,
	})
	encounters := encounter.NewService(encounter.Ports{
		Repo: encounter.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clk, Appointments: appointments,
	})
	jobs, err := worker.NewInserter(app)
	if err != nil {
		tb.Fatal(err)
	}
	return videosession.NewService(videosession.Ports{
		Repo: videosession.NewPostgresRepository(app), Provider: p, Audit: audit, Tx: tx, Clock: clk,
		Appointments: appointments, Patients: patients, Practitioners: practitioners, PublicURL: "ws://localhost:7880",
		Jobs: videosession.NewRiverJobs(jobs), Encounters: encounters, Consents: consents,
	})
}

// recordingProvider is a provider that records the rooms created and the participants removed, safe for
// concurrent calls; it calls no LiveKit. Its webhooks are the domain's events as JSON, "signed" when the
// authorization says so; its rooms are empty.
type recordingProvider struct {
	rooms   []string
	removed []string
	mu      sync.Mutex
}

func (p *recordingProvider) CreateRoom(_ context.Context, r videosession.Room) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rooms = append(p.rooms, r.Name)
	return nil
}

func (p *recordingProvider) Token(g videosession.Grant) (string, error) {
	return "token:" + g.Identity, nil
}

func (p *recordingProvider) RemoveParticipant(_ context.Context, room, identity string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.removed = append(p.removed, room+" "+identity)
	return nil
}

func (p *recordingProvider) DeleteRoom(context.Context, string) error { return nil }

func (p *recordingProvider) Participants(context.Context, string) ([]string, error) { return nil, nil }

func (p *recordingProvider) ParseWebhook(body []byte, authorization string) (videosession.Event, error) {
	var e videosession.Event
	if authorization != "signed" || json.Unmarshal(body, &e) != nil {
		return videosession.Event{}, videosession.ErrInvalidSignature
	}
	return e, nil
}

// composeLiveKit is the provider on the compose LiveKit (TEST_LIVEKIT_*); without it the benchmark fails.
func composeLiveKit(b *testing.B) videosession.Provider {
	b.Helper()
	url, key, secret := os.Getenv("TEST_LIVEKIT_URL"), os.Getenv("TEST_LIVEKIT_API_KEY"), os.Getenv("TEST_LIVEKIT_API_SECRET")
	if url == "" || key == "" || secret == "" {
		b.Fatal("TEST_LIVEKIT_URL, TEST_LIVEKIT_API_KEY and TEST_LIVEKIT_API_SECRET are not set; run it with `make bench-join`")
	}
	return videosession.NewLiveKitProvider(url, key, secret)
}

func callers(s consultationSeed) (patientCaller, practitionerCaller appointment.Caller) {
	return appointment.Caller{AccountID: s.patientAccount, PatientID: s.patient},
		appointment.Caller{AccountID: s.practitionerAccount, OrganizationID: s.clinic, PractitionerID: s.practitioner}
}

// The patient and the practitioner ask at the same moment for the first time: one session, one room, and
// the names each gets, with every join audited and the practitioner's read of the patient's name too.
func TestJoin_TwoFirstRequestsAtOnceShareOneSessionAndRoom(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	s := seedConsultation(t, owner, at)
	p := &recordingProvider{}
	svc := realService(t, app, at.Add(-5*time.Minute), p)
	pat, pra := callers(s)

	var wg sync.WaitGroup
	got := make([]videosession.Join, 2)
	errs := make([]error, 2)
	for i, c := range []appointment.Caller{pat, pra} {
		wg.Go(func() { got[i], errs[i] = svc.Join(context.Background(), c, s.appointment) })
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("Join() at once: %v, %v", errs[0], errs[1])
	}
	if got[0].PeerName != "Cy Example" || got[1].PeerName != "Ada Example" || got[0].Token == got[1].Token {
		t.Errorf("Join() = %+v and %+v; want each the other's name and their own identity", got[0], got[1])
	}
	var n int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM video_session").Scan(&n); err != nil || n != 1 {
		t.Errorf("%d sessions, %v; want 1", n, err)
	}
	if len(p.rooms) != 2 || p.rooms[0] != p.rooms[1] {
		t.Errorf("created rooms %v, want the one room twice", p.rooms)
	}
	var jobs int
	var scheduled time.Time
	if err := owner.QueryRow(t.Context(), `SELECT count(*), min(scheduled_at) FROM river_job WHERE kind = 'consultation_window_close'`).
		Scan(&jobs, &scheduled); err != nil || jobs != 1 || !scheduled.Equal(at.Add(time.Hour)) {
		t.Errorf("%d window-close jobs at %v, %v; want one at the window's close", jobs, scheduled, err)
	}
	var events []string
	rows, err := owner.Query(t.Context(), `SELECT subtype_code || ' ' || entity_reference || ' ' || purpose_code FROM audit_event ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			t.Fatal(err)
		}
		events = append(events, e)
	}
	want := []string{
		"operation Appointment/" + s.appointment + " PATRQT", "operation Appointment/" + s.appointment + " TREAT",
		"read Patient/" + s.patient + " TREAT",
		"read PractitionerRole/" + s.role + " HOPERAT", // the practitioner's own role, read by appointment's access check
	}
	if !slices.Equal(events, want) {
		t.Errorf("audited %v, want %v", events, want)
	}
}

// BenchmarkJoin is the token request through the service on the real database (an already open session,
// the steady state), with the compose LiveKit and with a fake provider, to separate LiveKit's share
// (docs/consultation/PROGRESS.md, package 1; `make bench-join`). It sits with the composition root because it
// builds the services the api does.
func BenchmarkJoin(b *testing.B) {
	for _, bc := range []struct {
		provider func(b *testing.B) videosession.Provider
		name     string
	}{
		{name: "livekit", provider: composeLiveKit},
		{name: "fake-provider", provider: func(*testing.B) videosession.Provider { return &recordingProvider{} }},
	} {
		b.Run(bc.name, func(b *testing.B) {
			owner, app := pgtest.NewAppDatabase(b)
			at := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
			s := seedConsultation(b, owner, at)
			p := bc.provider(b)
			svc := realService(b, app, at.Add(-5*time.Minute), p)
			pat, _ := callers(s)
			if _, err := svc.Join(context.Background(), pat, s.appointment); err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() {
				var room string
				if owner.QueryRow(context.Background(), "SELECT room FROM video_session").Scan(&room) == nil {
					_ = p.DeleteRoom(context.Background(), room)
				}
			})
			for b.Loop() {
				if _, err := svc.Join(context.Background(), pat, s.appointment); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
