package migrations_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A clinic made after the wave 6 contract gets its first location from its name and address (00081, LC1).
func TestLocations_ANewClinicGetsItsFirstLocation(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	const org = "00000000-0000-4000-8000-0000000000c1"
	if _, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name, address_line, postal_code, time_zone)
		VALUES ($1, 'Synthetic clinic', 'Synthetic street 2', '71000', 'Europe/Belgrade')`, org); err != nil {
		t.Fatal(err)
	}
	var name, address, postal, zone string
	var count int
	if err := owner.QueryRow(t.Context(), `SELECT min(name), min(address_line), min(postal_code), min(time_zone), count(*)
		FROM location WHERE organization_id = $1`, org).Scan(&name, &address, &postal, &zone, &count); err != nil {
		t.Fatal(err)
	}
	if count != 1 || name != "Synthetic clinic" || address != "Synthetic street 2" || postal != "71000" || zone != "Europe/Belgrade" {
		t.Errorf("first location = %q %q %q %q (%d), want the clinic's name, address and time zone, one", name, address, postal, zone, count)
	}
}

// A visit is where its slot is, whatever location the writer names; a video visit is nowhere; a move takes the new
// slot's location (00081, U2).
func TestLocations_AVisitIsWhereItsSlotIs(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	_, dir, a := seedAppointments(t, owner)
	ctx := t.Context()
	first := firstLocation(t, owner, clinicA)
	var second, schedule, slotFirst, slotSecond string
	if err := owner.QueryRow(ctx, `INSERT INTO location (organization_id, name) VALUES ($1, 'Second site') RETURNING id`, clinicA).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes) VALUES ($1, $2, 30) RETURNING id`,
		clinicA, dir[clinicA].role).Scan(&schedule); err != nil {
		t.Fatal(err)
	}
	const slot = `INSERT INTO slot (schedule_id, organization_id, period, visit_mode, location_id)
		VALUES ($1, $2, tstzrange($3::timestamptz, $3::timestamptz + interval '30 minutes'), 'in-person', $4) RETURNING id`
	if err := owner.QueryRow(ctx, slot, schedule, clinicA, "2031-02-03T08:00:00Z", first).Scan(&slotFirst); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, slot, schedule, clinicA, "2031-02-03T09:00:00Z", second).Scan(&slotSecond); err != nil {
		t.Fatal(err)
	}
	locationOf := func() *string {
		t.Helper()
		var l *string
		if err := owner.QueryRow(ctx, `SELECT location_id::text FROM appointment WHERE id = $1`, a.aFuture).Scan(&l); err != nil {
			t.Fatal(err)
		}
		return l
	}
	steps := []struct {
		want *string
		name string
		stmt string
		args []any
	}{
		{&first, "booked at the first site, naming the second", `UPDATE appointment SET visit_mode = 'in-person', slot_id = $2, location_id = $3 WHERE id = $1`, []any{a.aFuture, slotFirst, second}},
		{&first, "a location changed alone", `UPDATE appointment SET location_id = $2 WHERE id = $1`, []any{a.aFuture, second}},
		{&second, "moved to the second site's slot", `UPDATE appointment SET slot_id = $2 WHERE id = $1`, []any{a.aFuture, slotSecond}},
		{nil, "by video", `UPDATE appointment SET visit_mode = 'video' WHERE id = $1`, []any{a.aFuture}},
	}
	for _, s := range steps {
		if _, err := owner.Exec(ctx, s.stmt, s.args...); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if got := locationOf(); (got == nil) != (s.want == nil) || (got != nil && *got != *s.want) {
			t.Errorf("%s: location %v, want %v", s.name, got, s.want)
		}
	}
}
