package worker

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The clinic statistics end to end on the database (docs/plans/analytics.md §4): the Builder as newClinicStats
// builds it, on every real source as salus_stats, read back through the Service as salus_app. Synthetic data.

// statsPools are the owner's, salus_app's and salus_stats' logins on one throwaway database.
type statsPools struct {
	owner, app, stats *pgxpool.Pool
}

func newStatsPools(t testing.TB) statsPools {
	t.Helper()
	url := pgtest.NewDatabaseURL(t)
	open := func(u string) *pgxpool.Pool {
		pool, err := postgres.Open(context.Background(), u)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(pool.Close)
		return pool
	}
	return statsPools{owner: open(url), app: open(pgtest.AppURL(t, url)), stats: open(pgtest.StatsURL(t, url))}
}

// statsZone is the synthetic clinics' time zone, the default one.
var statsZone = func() *time.Location {
	l, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		panic(err)
	}
	return l
}()

func localAt(y int, m time.Month, d, h, minute int) time.Time {
	return time.Date(y, m, d, h, minute, 0, 0, statsZone)
}

// date is y-m-d as the aggregates hold it: midnight UTC.
func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// seedStatsRoles creates, as the owner, a clinic in Sarajevo with n practitioner roles and returns its id and the
// roles' ids.
func seedStatsRoles(t testing.TB, owner *pgxpool.Pool, n int) (string, []string) {
	t.Helper()
	ctx := context.Background()
	var org string
	var roles []string
	err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id),
			p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
				SELECT o.id, 'Cy', 'Example ' || i FROM o, generate_series(1, $1) i RETURNING id, organization_id),
			r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id)
		SELECT min(organization_id::text), array_agg(id::text ORDER BY id) FROM r`, n).Scan(&org, &roles)
	if err != nil {
		t.Fatal(err)
	}
	return org, roles
}

// recordNothing is an Auditor that keeps nothing.
type recordNothing struct{}

func (recordNothing) Record(context.Context, auditevent.AuditEvent) error { return nil }

// synthetic generates, as the owner, a clinic's synthetic appointments and Conditions on every local day from
// first to last (dates): perDay appointments per role and day at distinct half hours spread over the whole day
// (none at 02:xx, which a spring-forward day lacks), every status, 20 minutes long; perDay Conditions per role and
// day at times spread over the day, of seven codes, one in seven unconfirmed. Each role's Conditions hang on one
// anchor encounter and signed document in 2000, outside any range.
func synthetic(t testing.TB, owner *pgxpool.Pool, org string, roles []string, first, last time.Time, perDay int) {
	t.Helper()
	ctx := context.Background()
	_, err := owner.Exec(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role) VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id),
			p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ada', 'Example', '1990-01-01' FROM a RETURNING id),
			ap AS (INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status)
				SELECT p.id, r, $1, gen_random_uuid(), tstzrange('2000-01-01T09:00:00Z', '2000-01-01T09:30:00Z'), 'fulfilled'
				FROM p, unnest($2::uuid[]) r RETURNING *),
			e AS (INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start)
				SELECT id, organization_id, patient_id, practitioner_role_id, 'in-progress', lower(period) FROM ap RETURNING id),
			tm AS (INSERT INTO document_template (organization_id, title, fields)
				VALUES ($1, 'Placeholder template', '[{"key":"diagnosis","type":"coded","label":"Placeholder"}]') RETURNING id)
		INSERT INTO document_draft (encounter_id, template_id, status, fields)
			SELECT e.id, tm.id, 'entered-in-error', '[{"key":"diagnosis","missing":true}]' FROM e, tm`, org, roles)
	if err != nil {
		t.Fatal(err)
	}
	_, err = owner.Exec(ctx, `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, status,
			cancelled_at, cancellation_reason)
		SELECT x.patient, x.role, $1, gen_random_uuid(), tstzrange(x.start, x.start + interval '20 minutes'), x.status,
			CASE WHEN x.status = 'cancelled' THEN x.start END, CASE WHEN x.status = 'cancelled' THEN 'pat' END
		FROM (SELECT (SELECT patient_id FROM appointment WHERE organization_id = $1 LIMIT 1) AS patient, r.role,
				(d::date + make_interval(mins => (s.half * 30)::int)) AT TIME ZONE 'Europe/Sarajevo' AS start,
				(ARRAY['cancelled', 'cancelled', 'noshow', 'booked', 'fulfilled', 'fulfilled', 'fulfilled', 'fulfilled',
				       'fulfilled', 'fulfilled'])[1 + (k + r.n + extract(doy FROM d)::int) % 10] AS status
			FROM unnest($2::uuid[]) WITH ORDINALITY r (role, n), generate_series($3::date, $4::date, interval '1 day') d,
				generate_series(0, $5 - 1) k,
				LATERAL (SELECT (k * 31 + r.n * 5 + extract(doy FROM d)::int) % 48 AS half) s
			WHERE s.half / 2 <> 2) x`, org, roles, first, last, perDay)
	if err != nil {
		t.Fatal(err)
	}
	_, err = owner.Exec(ctx, `INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id, document_draft_id,
			practitioner_role_id, field_key, code, verification_status)
		SELECT (d::date + make_interval(mins => ((k * 277 + r.n * 61 + extract(doy FROM d)::int * 7) % 1440)::int)) AT TIME ZONE 'Europe/Sarajevo',
			$1, e.patient_id, e.id, dd.id, r.role, 'f' || to_char(d, 'YYYYMMDD') || '-' || k,
			(ARRAY['J00', 'J06.9', 'J18.9', 'I10', 'E11.9', 'U07.1', 'A09'])[1 + (k * 3 + r.n + extract(doy FROM d)::int) % 7],
			CASE WHEN (k + extract(doy FROM d)::int) % 7 = 0 THEN 'unconfirmed' ELSE 'confirmed' END
		FROM unnest($2::uuid[]) WITH ORDINALITY r (role, n)
			JOIN encounter e ON e.practitioner_role_id = r.role JOIN document_draft dd ON dd.encounter_id = e.id,
			generate_series($3::date, $4::date, interval '1 day') d, generate_series(0, $5 - 1) k`, org, roles, first, last, perDay)
	if err != nil {
		t.Fatal(err)
	}
}

// rawQuery runs q on the owner with the clinic and the range and reads each row with scan.
func rawQuery(t *testing.T, owner *pgxpool.Pool, q string, scan func(rows interface{ Scan(...any) error }) error, args ...any) {
	t.Helper()
	rows, err := owner.Query(t.Context(), q, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			t.Fatal(err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

// local is the source rows' local date and hour in the clinic's zone.
const localStart = `(lower(period) AT TIME ZONE 'Europe/Sarajevo')`

// The aggregates equal the raw counts: a synthetic clinic of three roles over three months (every status, starts
// and signings at every local hour, so on both sides of the local midnight, and the spring-forward day of 29 March),
// refreshed day by day, read back through the Service as its admin, against the same counts taken from the
// appointment and condition tables directly.
func TestClinicStats_AggregatesEqualTheRawCountsOfASyntheticClinic(t *testing.T) {
	t.Parallel()
	p := newStatsPools(t)
	org, roles := seedStatsRoles(t, p.owner, 3)
	first, last := date(2026, 2, 1), date(2026, 4, 30)
	synthetic(t, p.owner, org, roles, first, last, 6)
	b := newClinicStats(p.stats, clock.Fixed{T: localAt(2026, 5, 1, 9, 0)})
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		if err := b.Refresh(t.Context(), org, localAt(d.Year(), d.Month(), d.Day(), 12, 0)); err != nil {
			t.Fatal(err)
		}
	}
	svc := clinicstats.NewService(clinicstats.Ports{Repo: clinicstats.NewPostgresRepository(p.app), Audit: recordNothing{}, Tx: postgres.NewTransactor(p.app)})
	c := clinicstats.Caller{AccountID: "acc", OrganizationID: org, Admin: true}
	checkAppointments(t, p, svc, c, first, last)
	checkDiagnoses(t, p, svc, c, first, last)

	// The spring-forward day on its own: its 23 hours, both midnights excluded.
	dst := date(2026, 3, 29)
	got, err := svc.Appointments(t.Context(), c, clinicstats.AppointmentQuery{From: dst, To: dst})
	var n int64
	if err := p.owner.QueryRow(t.Context(), `SELECT count(*) FROM appointment WHERE organization_id = $1
		AND lower(period) >= '2026-03-28T23:00:00Z' AND lower(period) < '2026-03-29T22:00:00Z'`, org).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if err != nil || len(got.Series) != 1 || got.Series[0].Booked != n || n == 0 {
		t.Errorf("29 March: %v, %v; want %d appointments", got.Series, err, n)
	}
}

// checkAppointments compares the admin's appointment figures of the range with the appointment table's counts.
func checkAppointments(t *testing.T, p statsPools, svc *clinicstats.Service, c clinicstats.Caller, first, last time.Time) {
	t.Helper()
	org := c.OrganizationID
	figures, err := svc.Appointments(t.Context(), c, clinicstats.AppointmentQuery{From: first, To: last})
	if err != nil {
		t.Fatal(err)
	}

	var series []clinicstats.Period
	rawQuery(t, p.owner, `SELECT to_char(`+localStart+`::date, 'YYYY-MM'), count(*), count(*) FILTER (WHERE status = 'fulfilled'),
			count(*) FILTER (WHERE status = 'noshow'), count(*) FILTER (WHERE status = 'cancelled')
		FROM appointment WHERE organization_id = $1 AND `+localStart+`::date BETWEEN $2 AND $3 GROUP BY 1 ORDER BY 1`,
		func(r interface{ Scan(...any) error }) error {
			var x clinicstats.Period
			err := r.Scan(&x.Period, &x.Booked, &x.Fulfilled, &x.NoShow, &x.Cancelled)
			series = append(series, x)
			return err
		}, org, first, last)
	if len(series) != 3 || !slices.Equal(figures.Series, series) {
		t.Errorf("series\n got %v\nwant %v", figures.Series, series)
	}
	var heat []clinicstats.HeatCell
	rawQuery(t, p.owner, `SELECT extract(isodow FROM `+localStart+`)::int, extract(hour FROM `+localStart+`)::int, count(*)
		FROM appointment WHERE organization_id = $1 AND `+localStart+`::date BETWEEN $2 AND $3 AND status <> 'cancelled'
		GROUP BY 1, 2 ORDER BY 1, 2`,
		func(r interface{ Scan(...any) error }) error {
			var x clinicstats.HeatCell
			err := r.Scan(&x.Weekday, &x.Hour, &x.Count)
			heat = append(heat, x)
			return err
		}, org, first, last)
	if len(heat) < 7*20 || !slices.Equal(figures.Heatmap, heat) {
		t.Errorf("heatmap\n got %v\nwant %v", figures.Heatmap, heat)
	}
	var totals []clinicstats.PractitionerTotal
	rawQuery(t, p.owner, `SELECT practitioner_role_id::text, count(*) FILTER (WHERE status = 'fulfilled')
		FROM appointment WHERE organization_id = $1 AND `+localStart+`::date BETWEEN $2 AND $3 GROUP BY 1 ORDER BY 2 DESC, 1`,
		func(r interface{ Scan(...any) error }) error {
			var x clinicstats.PractitionerTotal
			err := r.Scan(&x.PractitionerRoleID, &x.Fulfilled)
			totals = append(totals, x)
			return err
		}, org, first, last)
	if !slices.Equal(figures.Practitioners, totals) {
		t.Errorf("practitioners\n got %v\nwant %v", figures.Practitioners, totals)
	}
}

// checkDiagnoses compares the admin's diagnosis figures of the range with the condition table's confirmed counts.
func checkDiagnoses(t *testing.T, p statsPools, svc *clinicstats.Service, c clinicstats.Caller, first, last time.Time) {
	t.Helper()
	const localRecorded = `(recorded_at AT TIME ZONE 'Europe/Sarajevo')`
	for by, key := range map[clinicstats.Breakdown]string{
		clinicstats.ByCode:             "code",
		clinicstats.ByMonth:            "to_char(" + localRecorded + "::date, 'YYYY-MM')",
		clinicstats.ByPractitionerRole: "practitioner_role_id::text",
	} {
		got, err := svc.Diagnoses(t.Context(), c, clinicstats.DiagnosisQuery{From: first, To: last, By: by})
		if err != nil {
			t.Fatal(err)
		}
		order := "2 DESC, 1"
		if by == clinicstats.ByMonth {
			order = "1"
		}
		var want []clinicstats.Count
		rawQuery(t, p.owner, `SELECT `+key+`, count(*) FROM condition WHERE organization_id = $1 AND verification_status = 'confirmed'
			AND `+localRecorded+`::date BETWEEN $2 AND $3 GROUP BY 1 ORDER BY `+order,
			func(r interface{ Scan(...any) error }) error {
				var x clinicstats.Count
				err := r.Scan(&x.Key, &x.Count)
				want = append(want, x)
				return err
			}, c.OrganizationID, first, last)
		if len(want) < 3 || !slices.Equal(got, want) {
			t.Errorf("diagnoses by %s\n got %v\nwant %v", by, got, want)
		}
	}
}
