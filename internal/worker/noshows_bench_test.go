package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// benchClinic and benchAdmin are the benchmark's synthetic clinic and its admin's account.
const (
	benchClinic = "b3600000-0000-4000-8000-00000000000a"
	benchAdmin  = "b3600000-0000-4000-8000-0000000000ad"
)

// rulesOf is the clinic's appointment settings, fixed.
type rulesOf organization.AppointmentSettings

func (r rulesOf) For(context.Context, string) (organization.AppointmentSettings, error) {
	return organization.AppointmentSettings(r), nil
}

// The repeat no-show flags of 62 days of a busy synthetic clinic (docs/plans/appointments.PROGRESS.md, SAL-36): one
// role, 600 patients with an active consent, each with 6 past appointments (one in three a no-show), and 25
// appointments a day in the range: 1 550 appointments of 600 patients, read by the admin as the api does, every one
// audited. go test -run '^$' -bench BenchmarkNoShowFlags ./internal/worker
func BenchmarkNoShowFlags(b *testing.B) {
	owner, app := pgtest.NewAppDatabase(b)
	var role string
	err := owner.QueryRow(b.Context(), `WITH o AS (INSERT INTO organization (id, name, listed) VALUES ($1, 'Synthetic clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id)
		INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id::text`, benchClinic).Scan(&role)
	if err != nil {
		b.Fatal(err)
	}
	now := time.Date(2026, 11, 13, 6, 0, 0, 0, time.UTC)
	for i, q := range []string{
		`WITH a AS (INSERT INTO account (email, password_hash, role) SELECT 'p' || i || '@example.test', 'h', 'patient' FROM generate_series(1, 600) i RETURNING id)
			INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a`,
		`INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT id, $1, 'active', $2 FROM patient`,
		// The history: 6 past appointments per patient, a quarter of an hour apart per patient and day.
		`INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
			SELECT p.id, $2, $1, gen_random_uuid(), tstzrange(t, t + interval '10 minutes'), CASE WHEN k % 3 = 0 THEN 'noshow' ELSE 'fulfilled' END
			FROM (SELECT id, row_number() OVER (ORDER BY id) n FROM patient) p, generate_series(1, 6) k,
				LATERAL (SELECT $3::timestamptz - make_interval(days => 3 + k * 25) + make_interval(mins => (p.n * 11)::int) AS t) x`,
		// The range: 25 a day for 62 days, the patients in turn.
		`INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
			SELECT p.id, $2, $1, gen_random_uuid(), tstzrange(t, t + interval '10 minutes'), 'booked'
			FROM generate_series(0, 62 * 25 - 1) i
				JOIN (SELECT id, row_number() OVER (ORDER BY id) - 1 n FROM patient) p ON p.n = i % 600,
				LATERAL (SELECT $3::timestamptz + make_interval(days => (i / 25)::int, mins => ((i % 25) * 20)::int) AS t) x`,
	} {
		args := [][]any{nil, {benchClinic, now}, {benchClinic, role, now}, {benchClinic, role, now}}[i]
		if _, err := owner.Exec(b.Context(), q, args...); err != nil {
			b.Fatal(err)
		}
	}
	tx := postgres.NewTransactor(app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, clock.Fixed{T: now})
	set := organization.DefaultAppointmentSettings()
	set.NoShowFlagThreshold = 2
	svc := appointment.NewService(appointment.Ports{
		Repo: appointment.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: clock.Fixed{T: now},
		Consents: consent.NewService(consent.NewPostgresRepository(app), audit, tx, clock.Fixed{T: now}, nil),
		NoShows:  appointment.NewPostgresNoShowRepository(app), Rules: rulesOf(set),
	})
	c := appointment.Caller{AccountID: benchAdmin, OrganizationID: benchClinic, Admin: true}
	if _, err := owner.Exec(b.Context(), `INSERT INTO account (id, email, password_hash, role, organization_id) VALUES ($1, 'admin@example.test', 'h', 'admin', $2)`, benchAdmin, benchClinic); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		flags, err := svc.NoShowFlags(b.Context(), c, now, now.Add(62*24*time.Hour), "")
		if err != nil || len(flags) != 62*25 {
			b.Fatalf("%d flags, %v; want every appointment of the range", len(flags), err)
		}
	}
}
