//go:build loadtest

package load_test

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// seedPrefix starts every seeded clinic's name; guard refuses a database with any other clinic.
const seedPrefix = "Load clinic "

// zone is the seeded clinics' time zone, the default one.
const zone = "Europe/Sarajevo"

// scale is the size of a seed. Every name, address, e-mail and number in it is synthetic.
type scale struct {
	clinics    int // each with one location, an admin and a receptionist
	roles      int // practitioners per clinic, each with an account and a weekday schedule
	patients   int // spread over the clinics, each with an account and a consent at one clinic (one in ten at two)
	perDay     int // visits per practitioner and weekday, of the schedule's 16 half-hour slots
	pastDays   int // visits from today minus pastDays
	futureDays int // to today plus futureDays
	services   int // price list entries per clinic
}

// busyYear is WAVE7.md RD2's size: 50 clinics, 500 practitioners, 200 000 patients and about 1 000 000 visits over
// a year (261 weekdays x 500 practitioners x 8), with their receipts and reminders.
var busyYear = scale{clinics: 50, roles: 10, patients: 200_000, perDay: 8, pastDays: 300, futureDays: 64, services: 20}

// small is the budget test's seed (budget_test.go): the same shape, about 25 000 visits.
var small = scale{clinics: 3, roles: 10, patients: 6_000, perDay: 8, pastDays: 120, futureDays: 30, services: 20}

// seeded is what the driver needs of a seed: ids of who signs in and what they ask for.
type seeded struct {
	clinics []seededClinic
	// patients are, per clinic (same order), a sample of its patients' ids and account ids.
	patients [][]seededPatient
}

type seededClinic struct {
	id, admin, reception string
	roles                []seededRole
}

type seededRole struct{ id, practitioner, account, schedule string }

type seededPatient struct{ id, account string }

// seedSteps are the seed's statements in order, on one connection (the temporary tables are its own). $1 to $7 stand
// for the scale's fields in order, $8 for the time zone (seed writes them in).
var seedSteps = []struct{ name, sql string }{
	{"clinics", `CREATE TEMP TABLE seed_org AS
		SELECT c AS n, gen_random_uuid() AS id, gen_random_uuid() AS admin, gen_random_uuid() AS reception
		FROM generate_series(1, $1::int) c`},
	{"organizations", `INSERT INTO organization (id, name, listed, legal_name, tax_id, address_line, postal_code, billing_email, languages, time_zone)
		SELECT id, 'Load clinic ' || lpad(n::text, 3, '0'), true, 'Load clinic ' || n || ' d.o.o.', '42000000' || lpad(n::text, 5, '0'),
			'Synthetic street ' || n, '71000', 'load-clinic-' || n || '@example.test', '{bs,en}', $8
		FROM seed_org`},
	{"staff", `INSERT INTO account (id, email, password_hash, role, organization_id)
		SELECT admin, 'load-admin-' || n || '@example.test', 'synthetic-no-login', 'admin', id FROM seed_org
		UNION ALL SELECT reception, 'load-reception-' || n || '@example.test', 'synthetic-no-login', 'reception', id FROM seed_org`},
	{"roles", `CREATE TEMP TABLE seed_role AS
		SELECT o.n AS org_n, r, o.id AS org, (SELECT l.id FROM location l WHERE l.organization_id = o.id) AS location,
			gen_random_uuid() AS practitioner, gen_random_uuid() AS role, gen_random_uuid() AS account, gen_random_uuid() AS schedule
		FROM seed_org o, generate_series(1, $2::int) r`},
	{"practitioners", `INSERT INTO practitioner (id, organization_id, given_name, family_name)
		SELECT practitioner, org, 'Load', 'Practitioner ' || org_n || '-' || r FROM seed_role`},
	{"practitioner roles", `INSERT INTO practitioner_role (id, organization_id, practitioner_id) SELECT role, org, practitioner FROM seed_role`},
	{"practitioner accounts", `INSERT INTO account (id, email, password_hash, role, organization_id, practitioner_id)
		SELECT account, 'load-practitioner-' || org_n || '-' || r || '@example.test', 'synthetic-no-login', 'practitioner', org, practitioner FROM seed_role`},
	// Monday to Wednesday at the clinic, Thursday and Friday by video, 08:00 to 16:00 in half hours.
	{"schedules", `INSERT INTO schedule (id, organization_id, practitioner_role_id, slot_minutes, availability)
		SELECT schedule, org, role, 30, jsonb_build_array(
			jsonb_build_object('day', 1, 'start', '08:00', 'end', '16:00', 'mode', 'in-person', 'location_id', location),
			jsonb_build_object('day', 2, 'start', '08:00', 'end', '16:00', 'mode', 'in-person', 'location_id', location),
			jsonb_build_object('day', 3, 'start', '08:00', 'end', '16:00', 'mode', 'in-person', 'location_id', location),
			jsonb_build_object('day', 4, 'start', '08:00', 'end', '16:00', 'mode', 'video'),
			jsonb_build_object('day', 5, 'start', '08:00', 'end', '16:00', 'mode', 'video'))
		FROM seed_role`},
	{"services", `CREATE TEMP TABLE seed_service AS
		SELECT o.n AS org_n, s, o.id AS org, gen_random_uuid() AS id, (2000 + s * 500)::bigint AS price FROM seed_org o, generate_series(0, $7::int - 1) s`},
	{"price list", `INSERT INTO charge_item_definition (id, organization_id, name, code, price_minor, duration_minutes, vat_rate_bp)
		SELECT id, org, 'Synthetic service ' || s, 'S' || s, price, 30, 0 FROM seed_service`},
	{"patients", `CREATE TEMP TABLE seed_patient AS
		SELECT i AS n, 1 + (i - 1) % $1::int AS org_n, (i - 1) / $1::int AS k, gen_random_uuid() AS id, gen_random_uuid() AS account
		FROM generate_series(1, $3::int) i`},
	{"patient index", `CREATE INDEX ON seed_patient (org_n, k)`},
	{"patient accounts", `INSERT INTO account (id, email, password_hash, role, language)
		SELECT account, 'load-patient-' || n || '@example.test', 'synthetic-no-login', 'patient', 'bs' FROM seed_patient`},
	{"patient profiles", `INSERT INTO patient (id, account_id, birth_date, given_name, family_name, phone)
		SELECT id, account, date '1940-01-01' + (n * 7919) % 25000, 'Load', 'Patient ' || n, '+3876' || lpad(n::text, 7, '0') FROM seed_patient`},
	{"consents", `INSERT INTO consent (patient_id, organization_id, date_time, status, given_in_person)
		SELECT p.id, o.id, now() - interval '400 days', 'active', false FROM seed_patient p JOIN seed_org o ON o.n = p.org_n
		UNION ALL
		SELECT p.id, o.id, now() - interval '400 days', 'active', false FROM seed_patient p JOIN seed_org o ON o.n = p.org_n % $1::int + 1
		WHERE p.n % 10 = 0 AND $1::int > 1`},
	// Each practitioner's weekday holds perDay visits on every other half hour from 08:00 (alternating by day), each
	// with its slot. Past visits: 80% fulfilled, 5% no-show, 15% cancelled; future: 85% booked, 15% cancelled.
	{"visits", `CREATE TEMP TABLE seed_visit AS
		SELECT gen_random_uuid() AS id, gen_random_uuid() AS slot, r.org, r.org_n, r.role, r.schedule, r.r,
			CASE WHEN v.mode = 'in-person' THEN r.location END AS location, p.id AS patient, v.start, v.mode,
			CASE WHEN v.pick BETWEEN 15 AND 17 THEN 'cancelled' WHEN v.start > now() THEN 'booked'
				WHEN v.pick = 14 THEN 'noshow' ELSE 'fulfilled' END AS status,
			sv.id AS service, sv.price
		FROM seed_role r,
			generate_series(current_date - $5::int, current_date + $6::int, interval '1 day') d,
			generate_series(0, $4::int - 1) j,
			LATERAL (SELECT (d::date - date '2000-01-01') AS day) dn,
			LATERAL (SELECT ((d::date + time '08:00' + make_interval(mins => 30 * (2 * j + (r.r + dn.day) % 2)))
					AT TIME ZONE $8) AS start,
				CASE WHEN extract(isodow FROM d) <= 3 THEN 'in-person' ELSE 'video' END AS mode,
				(j * 7 + r.r * 3 + dn.day) % 20 AS pick) v,
			LATERAL (SELECT sp.id FROM seed_patient sp WHERE sp.org_n = r.org_n
				AND sp.k = (r.r::bigint * 7919 + dn.day::bigint * 104729 + j::bigint * 1299709) % ($3::int / $1::int)) p,
			seed_service sv
		WHERE extract(isodow FROM d) <= 5 AND sv.org_n = r.org_n AND sv.s = (j + r.r) % $7::int`},
	{"slots", `INSERT INTO slot (id, schedule_id, organization_id, status, period, visit_mode, location_id, created_at)
		SELECT slot, schedule, org, CASE WHEN status = 'cancelled' THEN 'free' ELSE 'busy' END,
			tstzrange(start, start + interval '30 minutes'), mode, location, start - interval '30 days' FROM seed_visit`},
	{"appointments", `INSERT INTO appointment (id, patient_id, practitioner_role_id, organization_id, slot_id, period, status, cancelled_at,
			cancellation_reason, created_at, updated_at, visit_mode, service_id)
		SELECT id, patient, role, org, slot, tstzrange(start, start + interval '30 minutes'), status,
			CASE WHEN status = 'cancelled' THEN start - interval '2 days' END, CASE WHEN status = 'cancelled' THEN 'pat' END,
			start - interval '14 days', start, mode, service FROM seed_visit`},
	{"reminders", `INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at, state, dispatched_at, created_at)
		SELECT v.id, v.org, v.patient, o.m, v.start, v.start - make_interval(mins => o.m),
			CASE WHEN v.status = 'cancelled' THEN 'cancelled' WHEN v.start - make_interval(mins => o.m) > now() THEN 'scheduled' ELSE 'dispatched' END,
			CASE WHEN v.status <> 'cancelled' AND v.start - make_interval(mins => o.m) <= now() THEN v.start - make_interval(mins => o.m) END,
			v.start - interval '14 days'
		FROM seed_visit v, (VALUES (1440), (60)) o (m)
		WHERE v.start > now() - interval '90 days'`}, // the retention rule keeps 90 days after the due time
	{"encounters", `INSERT INTO encounter (id, appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, period_end, created_at)
		SELECT gen_random_uuid(), id, org, patient, role, 'finished', start, start + interval '25 minutes', start FROM seed_visit WHERE status = 'fulfilled'`},
	{"templates", `INSERT INTO document_template (organization_id, title, fields)
		SELECT id, 'Placeholder template', '[{"key":"diagnosis","type":"coded","label":"Placeholder"}]' FROM seed_org`},
	{"drafts", `INSERT INTO document_draft (encounter_id, template_id, fields, created_at)
		SELECT e.id, t.id, '[{"key":"diagnosis","missing":true}]', e.period_start FROM encounter e
		JOIN document_template t ON t.organization_id = e.organization_id JOIN seed_org o ON o.id = e.organization_id`},
	// One coded diagnosis a fulfilled visit, of seven ICD-10 codes, one in seven unconfirmed.
	{"conditions", `INSERT INTO condition (recorded_at, organization_id, patient_id, encounter_id, document_draft_id, practitioner_role_id,
			field_key, code, verification_status)
		SELECT e.period_end, e.organization_id, e.patient_id, e.id, dd.id, e.practitioner_role_id, 'diagnosis',
			(ARRAY['J00', 'J06.9', 'J18.9', 'I10', 'E11.9', 'M54.5', 'A09'])[1 + abs(hashtext(e.id::text)) % 7],
			CASE WHEN abs(hashtext(e.id::text)) % 7 = 0 THEN 'unconfirmed' ELSE 'confirmed' END
		FROM encounter e JOIN document_draft dd ON dd.encounter_id = e.id JOIN seed_org o ON o.id = e.organization_id`},
}

// receiptSteps issue a fiscalised receipt with one line for every fulfilled visit, numbered per clinic in order,
// 40 minutes after its start, by the clinic's receptionist: one transaction (a line is added only with its receipt).
var receiptSteps = []string{
	`CREATE TEMP TABLE seed_receipt AS
		SELECT gen_random_uuid() AS receipt, v.*, o.reception, row_number() OVER (PARTITION BY v.org ORDER BY v.start, v.id) AS number
		FROM seed_visit v JOIN seed_org o ON o.id = v.org WHERE v.status = 'fulfilled'`,
	`INSERT INTO invoice (id, organization_id, seller_legal_name, seller_tax_id, seller_address, patient_id, appointment_id, kind, number,
			issued_at, issued_by_account_id, payment_method, total_minor, fiscal_status, fiscal_id, fiscal_code, fiscal_qr, fiscalized_at,
			place_of_service)
		SELECT receipt, org, 'Load clinic ' || org_n || ' d.o.o.', '42000000' || lpad(org_n::text, 5, '0'), 'Synthetic street ' || org_n,
			patient, id, 'receipt', number, start + interval '40 minutes', reception,
			(ARRAY['cash', 'card'])[1 + number % 2], price, 'fiscalized', 'SYN-' || number, 'SYNCODE', 'https://example.test/qr',
			start + interval '41 minutes', CASE WHEN location IS NOT NULL THEN 'Synthetic street ' || org_n END
		FROM seed_receipt`,
	`INSERT INTO invoice_line (invoice_id, charge_item_definition_id, name, quantity, unit_price_minor, total_minor, position, vat_rate_bp)
		SELECT receipt, service, 'Synthetic service', 1, price, price, 0, 0 FROM seed_receipt`,
}

// seed writes s's synthetic data with owner (the schema's owner, so row-level security does not apply), then builds
// the clinic statistics of every seeded day as statsPool (salus_stats, as the worker does) and analyses the tables.
// Only budget_test.go's throwaway database and a database guard has passed come here.
func seed(ctx context.Context, owner, statsPool *pgxpool.Pool, s scale) error {
	conn, err := owner.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	// The scale's numbers and the zone are written into the statements: each uses only some of them, and Postgres
	// cannot type a parameter a statement does not use.
	args := strings.NewReplacer("$1", strconv.Itoa(s.clinics), "$2", strconv.Itoa(s.roles), "$3", strconv.Itoa(s.patients),
		"$4", strconv.Itoa(s.perDay), "$5", strconv.Itoa(s.pastDays), "$6", strconv.Itoa(s.futureDays), "$7", strconv.Itoa(s.services),
		"$8", "'"+zone+"'")
	for _, st := range seedSteps {
		began := wall.Now()
		n, err := conn.Exec(ctx, args.Replace(st.sql))
		if err != nil {
			return fmt.Errorf("seed %s: %w", st.name, err)
		}
		log.Printf("seed: %s, %d rows, %s", st.name, n.RowsAffected(), wall.Now().Sub(began).Round(time.Millisecond))
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	began := wall.Now()
	for _, q := range receiptSteps {
		if _, err := tx.Exec(ctx, q); err != nil {
			return fmt.Errorf("seed receipts: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	log.Printf("seed: receipts, %s", wall.Now().Sub(began).Round(time.Millisecond))
	if err := refreshStats(ctx, owner, statsPool, s); err != nil {
		return err
	}
	_, err = conn.Exec(ctx, `VACUUM (ANALYZE)`)
	return err
}

// refreshStats runs the clinic statistics' refresh of every clinic and seeded day, as the worker's clinic_stats job
// does after the day's changes, on runtime.NumCPU() workers.
func refreshStats(ctx context.Context, owner, statsPool *pgxpool.Pool, s scale) error {
	began := wall.Now()
	builder := clinicstats.NewBuilder(clinicstats.BuildPorts{
		Repo: clinicstats.NewPostgresBuildRepository(statsPool), Tx: postgres.NewTransactor(statsPool), Clock: clock.System{},
		Clinics:      organization.NewStats(organization.NewPostgresStatsRepository(statsPool)),
		Appointments: appointment.NewStats(appointment.NewPostgresStatsRepository(statsPool)),
		Conditions:   condition.NewService(condition.NewPostgresRepository(statsPool)),
		Specialties:  practitionerrole.NewStats(practitionerrole.NewPostgresStatsRepository(statsPool)),
		Groups:       terminology.NewCatalog(terminology.NewPostgresCatalogRepository(statsPool)),
	})
	rows, err := owner.Query(ctx, `SELECT id::text FROM organization WHERE name LIKE $1 || '%'`, seedPrefix)
	if err != nil {
		return err
	}
	var orgs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		orgs = append(orgs, id)
	}
	rows.Close()
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return err
	}
	today := wall.Now().In(loc)
	type job struct {
		at  time.Time
		org string
	}
	work := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var first error
	for range runtime.NumCPU() {
		wg.Go(func() {
			for j := range work {
				if err := builder.Refresh(ctx, j.org, j.at); err != nil {
					mu.Lock()
					first = err
					mu.Unlock()
				}
			}
		})
	}
	for _, org := range orgs {
		for d := -s.pastDays; d <= s.futureDays; d++ {
			day := today.AddDate(0, 0, d)
			work <- job{org: org, at: time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, loc)}
		}
	}
	close(work)
	wg.Wait()
	if first != nil {
		return fmt.Errorf("seed clinic statistics: %w", first)
	}
	log.Printf("seed: clinic statistics of %d clinic-days, %s", len(orgs)*(s.pastDays+s.futureDays+1), wall.Now().Sub(began).Round(time.Millisecond))
	return nil
}
