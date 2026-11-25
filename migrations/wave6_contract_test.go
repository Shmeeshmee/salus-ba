package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// wave6Contract is the migration of the wave 6 contract (docs/clinic-cms/WAVE6.md, 6.0).
const wave6Contract = 80

// seedAccount creates, as the owner, a staff account of org with the role and returns its id.
func seedAccount(t *testing.T, owner *pgxpool.Pool, org, role, email string) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `INSERT INTO account (email, password_hash, role, organization_id)
		VALUES ($1, 'h', $2, $3) RETURNING id`, email, role, org).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// firstLocation returns the location the migration gave org.
func firstLocation(t *testing.T, owner *pgxpool.Pool, org string) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `SELECT id FROM location WHERE organization_id = $1`, org).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// seedLocation returns the location of org: a clinic made after the migration gets its first one from 00081's
// trigger (LC1).
func seedLocation(t *testing.T, owner *pgxpool.Pool, org string) string {
	t.Helper()
	return firstLocation(t, owner, org)
}

// execIn runs stmt in the app's scope and returns its error.
func execIn(t *testing.T, app *pgxpool.Pool, scope map[string]string, stmt string, args ...any) error {
	t.Helper()
	return asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, stmt, args...)
		return err
	})
}

// Every clinic that exists gets its first location from its name and address, and every in-person rule, slot and
// visit is at it; video ones have none (U1, U2).
func TestWave6_TheClinicsAddressBecomesItsFirstLocation(t *testing.T) {
	t.Parallel()
	dbURL := pgtest.NewEmptyDatabaseURL(t)
	provider := gooseProvider(t, dbURL)
	if _, err := provider.UpTo(t.Context(), wave6Contract-1); err != nil {
		t.Fatal(err)
	}
	owner := open(t, dbURL)
	dir := seedDirectory(t, owner)
	ctx := t.Context()
	if _, err := owner.Exec(ctx, `UPDATE organization SET address_line = 'Synthetic street 1', postal_code = '71000' WHERE id = $1`, clinicA); err != nil {
		t.Fatal(err)
	}
	var schedule string
	if err := owner.QueryRow(ctx, `INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
		VALUES ($1, $2, 30, '[{"day": 1, "start": "08:00", "end": "12:00", "mode": "in-person"}, {"day": 3, "start": "14:00", "end": "16:00", "mode": "video"}]')
		RETURNING id`, clinicA, dir[clinicA].role).Scan(&schedule); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO slot (schedule_id, organization_id, period, visit_mode) VALUES
		($1, $2, tstzrange('2030-01-07T08:00:00Z', '2030-01-07T08:30:00Z'), 'in-person'),
		($1, $2, tstzrange('2030-01-09T14:00:00Z', '2030-01-09T14:30:00Z'), 'video')`, schedule, clinicA); err != nil {
		t.Fatal(err)
	}
	p := seedPatients(t, owner)
	var inPerson, video string
	const visit = `INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, visit_mode)
		VALUES ($1, $2, $3, gen_random_uuid(), tstzrange($4::timestamptz, $4::timestamptz + interval '30 minutes'), $5) RETURNING id`
	if err := owner.QueryRow(ctx, visit, p[0].patient, dir[clinicA].role, clinicA, "2030-01-07T08:00:00Z", "in-person").Scan(&inPerson); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, visit, p[0].patient, dir[clinicA].role, clinicA, "2030-01-09T14:00:00Z", "video").Scan(&video); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, wave6Contract); err != nil {
		t.Fatal(err)
	}

	var name, address, postal string
	if err := owner.QueryRow(ctx, `SELECT name, address_line, postal_code FROM location WHERE organization_id = $1`, clinicA).
		Scan(&name, &address, &postal); err != nil || name != "Synthetic clinic" || address != "Synthetic street 1" || postal != "71000" {
		t.Errorf("clinic A's first location = %q %q %q, %v", name, address, postal, err)
	}
	var count int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM location`).Scan(&count); err != nil || count != 2 {
		t.Errorf("%d locations, %v; want one per clinic", count, err)
	}
	loc := firstLocation(t, owner, clinicA)
	var rules []*string
	if err := owner.QueryRow(ctx, `SELECT array_agg(r->>'location_id' ORDER BY r->>'day') FROM schedule, jsonb_array_elements(availability) r
		WHERE id = $1`, schedule).Scan(&rules); err != nil || len(rules) != 2 || rules[0] == nil || *rules[0] != loc || rules[1] != nil {
		t.Errorf("rules' locations = %v, %v; want the first location on the in-person rule only", rules, err)
	}
	var slots []*string
	if err := owner.QueryRow(ctx, `SELECT array_agg(location_id::text ORDER BY lower(period)) FROM slot WHERE schedule_id = $1`, schedule).
		Scan(&slots); err != nil || len(slots) != 2 || slots[0] == nil || *slots[0] != loc || slots[1] != nil {
		t.Errorf("slots' locations = %v, %v; want the first location on the in-person slot only", slots, err)
	}
	var visits []*string
	if err := owner.QueryRow(ctx, `SELECT array_agg(location_id::text ORDER BY lower(period)) FROM appointment WHERE id IN ($1, $2)`, inPerson, video).
		Scan(&visits); err != nil || len(visits) != 2 || visits[0] == nil || *visits[0] != loc || visits[1] != nil {
		t.Errorf("visits' locations = %v, %v; want the first location on the in-person visit only", visits, err)
	}
}

// A clinic sees and writes its own locations, rooms and devices; the directory sees the active locations of listed
// clinics, a patient the location of their own visit (U1, U2).
func TestWave6_LocationsAreTheClinicsAndTheDirectorysAndTheVisitsPatients(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, dir, a := seedAppointments(t, owner)
	ctx := t.Context()
	locA, locB := seedLocation(t, owner, clinicA), seedLocation(t, owner, clinicB)
	if _, err := owner.Exec(ctx, `UPDATE organization SET listed = (id = $1)`, clinicB); err != nil {
		t.Fatal(err)
	}
	// The visit takes its slot's location (00081).
	if _, err := owner.Exec(ctx, `WITH s AS (INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) VALUES ($2, $3, 30) RETURNING id),
		sl AS (INSERT INTO slot (schedule_id, organization_id, period, visit_mode, location_id)
			SELECT id, $2, tstzrange('2031-01-06T08:00:00Z', '2031-01-06T08:30:00Z'), 'in-person', $4 FROM s RETURNING id)
		UPDATE appointment SET visit_mode = 'in-person', slot_id = (SELECT id FROM sl) WHERE id = $1`, a.aFuture, clinicA, dir[clinicA].role, locA); err != nil {
		t.Fatal(err)
	}
	const locations = "SELECT id::text FROM location"
	for name, tc := range map[string]struct {
		scope map[string]string
		want  []string
	}{
		"clinic A":  {map[string]string{"app.organization_id": clinicA}, []string{locA}},
		"directory": {map[string]string{"app.directory": "public"}, []string{locB}},
		"patient A": {map[string]string{"app.patient_id": p[0].patient}, []string{locA}},
		"patient B": {map[string]string{"app.patient_id": p[1].patient}, nil},
		"no scope":  {nil, nil},
	} {
		if got := idsIn(t, app, tc.scope, locations); !slices.Equal(got, tc.want) {
			t.Errorf("%s sees %v, want %v", name, got, tc.want)
		}
	}
	scopeA := map[string]string{"app.organization_id": clinicA}
	if err := execIn(t, app, scopeA, `INSERT INTO room (organization_id, location_id, name) VALUES ($1, $2, 'Room 1')`, clinicA, locA); err != nil {
		t.Errorf("clinic A adds a room: %v", err)
	}
	if err := execIn(t, app, scopeA, `INSERT INTO device (organization_id, location_id, name) VALUES ($1, $2, 'Ultrasound')`, clinicA, locB); err == nil {
		t.Error("clinic A adds a device at clinic B's location: no error")
	}
	if err := execIn(t, app, scopeA, `INSERT INTO location (organization_id, name) VALUES ($1, 'Second site')`, clinicB); err == nil {
		t.Error("clinic A adds a location to clinic B: no error")
	}
}

// Two in-person slots in one room never overlap, nor two visits that need one device; a video slot or visit has no
// location (U2 to U4).
func TestWave6_ARoomAndADeviceAreNeverDoubleBooked(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	_, dir, a := seedAppointments(t, owner)
	ctx := t.Context()
	loc := seedLocation(t, owner, clinicA)
	var room, device, schedule string
	if err := owner.QueryRow(ctx, `INSERT INTO room (organization_id, location_id, name) VALUES ($1, $2, 'Room 1') RETURNING id`, clinicA, loc).Scan(&room); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO device (organization_id, location_id, name) VALUES ($1, $2, 'Ultrasound') RETURNING id`, clinicA, loc).Scan(&device); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) VALUES ($1, $2, 30) RETURNING id`,
		clinicA, dir[clinicA].role).Scan(&schedule); err != nil {
		t.Fatal(err)
	}
	const slot = `INSERT INTO slot (schedule_id, organization_id, period, visit_mode, location_id, room_id)
		VALUES ($1, $2, tstzrange($3::timestamptz, $3::timestamptz + interval '30 minutes'), $4, $5, $6)`
	if _, err := owner.Exec(ctx, slot, schedule, clinicA, "2030-02-04T08:00:00Z", "in-person", loc, room); err != nil {
		t.Fatal(err)
	}
	// Another schedule's slot in the same room, overlapping: refused by the room, not the schedule.
	var other string
	if err := owner.QueryRow(ctx, `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Bea', 'Example') RETURNING id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT $1, id FROM p RETURNING id)
		INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) SELECT $1, id, 30 FROM r RETURNING id`, clinicA).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, slot, other, clinicA, "2030-02-04T08:15:00Z", "in-person", loc, room); sqlState(err) != "23P01" {
		t.Errorf("an overlapping slot in the same room = %v, want an exclusion violation", err)
	}
	if _, err := owner.Exec(ctx, slot, other, clinicA, "2030-02-05T08:00:00Z", "video", loc, nil); sqlState(err) != "23514" {
		t.Errorf("a video slot at a location = %v, want a check violation", err)
	}
	// A visit takes its service's device (00082's guard, RM1), so the visits need it through a service.
	var service string
	if err := owner.QueryRow(ctx, `INSERT INTO charge_item_definition (organization_id, name, price_minor, device_id) VALUES ($1, 'Synthetic scan', 5000, $2)
		RETURNING id`, clinicA, device).Scan(&service); err != nil {
		t.Fatal(err)
	}
	const needs = `UPDATE appointment SET visit_mode = 'in-person', service_id = $2 WHERE id = $1`
	if _, err := owner.Exec(ctx, needs, a.aFuture, service); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `UPDATE appointment SET period = (SELECT period FROM appointment WHERE id = $2),
		practitioner_role_id = (SELECT practitioner_role_id FROM schedule WHERE id = $3) WHERE id = $1`, a.other, a.aFuture, other); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, needs, a.other, service); sqlState(err) != "23P01" {
		t.Errorf("an overlapping visit needing the same device = %v, want an exclusion violation", err)
	}
}
