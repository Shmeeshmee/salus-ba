package worker_test

import (
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// seedStatsClinic creates, as the owner, a clinic with a role and a fulfilled appointment of a synthetic patient
// starting at; it returns the clinic.
func seedStatsClinic(t *testing.T, db logins, at time.Time) string {
	t.Helper()
	var org string
	err := db.owner.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id),
			pr AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
			r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM pr RETURNING id, organization_id),
			a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
			p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id)
		INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
			SELECT p.id, r.id, r.organization_id, gen_random_uuid(), tstzrange($1::timestamptz, $1::timestamptz + interval '30 minutes'), 'fulfilled' FROM p, r
		RETURNING organization_id::text`, at).Scan(&org)
	if err != nil {
		t.Fatal(err)
	}
	return org
}

func appointmentStats(t *testing.T, db logins) int {
	t.Helper()
	var n int
	if err := db.owner.QueryRow(t.Context(), "SELECT coalesce(sum(count), 0) FROM clinic_appointment_stat").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// clinicstats_refresh recomputes the clinic-day of its instant on the salus_stats pool.
func TestWorker_RefreshesAClinicDay(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	at := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	org := seedStatsClinic(t, db, at)
	if kind, _ := runJobOn(t, db, liveKit(t), testFiles(t), clinicstats.RefreshArgs{OrganizationID: org, Day: at}, nil); kind != river.EventKindJobCompleted {
		t.Fatalf("job ended %s", kind)
	}
	if n := appointmentStats(t, db); n != 1 {
		t.Errorf("%d appointments counted, want 1", n)
	}
}

// clinicstats_nightly recomputes the last 35 days of every clinic.
func TestWorker_RefreshesTheRecentDaysNightly(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	seedStatsClinic(t, db, clock.System{}.Now().Add(-48*time.Hour).Truncate(time.Hour))
	if kind, _ := runJobOn(t, db, liveKit(t), testFiles(t), clinicstats.NightlyArgs{}, nil); kind != river.EventKindJobCompleted {
		t.Fatalf("job ended %s", kind)
	}
	if n := appointmentStats(t, db); n != 1 {
		t.Errorf("%d appointments counted, want 1", n)
	}
}

func TestNew_RefusesToStartWithoutTheStatsPool(t *testing.T) {
	_, app := pgtest.NewAppDatabase(t)
	if _, err := worker.New(app, app, nil, liveKit(t), testFiles(t), webhookKey, slog.New(slog.DiscardHandler)); err == nil {
		t.Fatal("New() error = nil without the stats pool")
	}
}

// The stats pool reaches one place: New hands it to newClinicStats, which builds the clinic statistics' Builder,
// the sources' stats readers and its transactions on it. Every other job, and River itself, runs on salus_app.
func TestNew_HandsTheStatsPoolToTheClinicStatisticsOnly(t *testing.T) {
	want := []string{"appointment.NewPostgresStatsRepository", "clinicstats.NewPostgresBuildRepository", "condition.NewPostgresRepository",
		"newClinicStats", "organization.NewPostgresStatsRepository", "postgres.NewTransactor",
		"practitionerrole.NewPostgresStatsRepository", "terminology.NewPostgresCatalogRepository"}
	if uses := poolUses(t, "statsPool"); !slices.Equal(uses, want) {
		t.Errorf("statsPool is passed to %v, want only %v", uses, want)
	}
}
