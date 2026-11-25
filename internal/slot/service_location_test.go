package slot_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const siteA = "a5000000-0000-4000-8000-0000000000a1"

// An in-person slot is at its rule's location; a video slot at none (docs/clinic-cms/WAVE6.md U2).
func TestGenerate_GivesEachInPersonSlotItsRulesLocation(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.schedules.schedules[0].SlotMinutes, f.schedules.schedules[0].Rules = 60, []schedule.Rule{
		{Day: 1, Start: "08:00", End: "09:00", Mode: schedule.ModeInPerson, LocationID: siteA},
		{Day: 1, Start: "09:00", End: "10:00", Mode: schedule.ModeVideo},
	}
	if _, err := f.service(now).Generate(t.Context(), clinicA, scheduleA1); err != nil {
		t.Fatal(err)
	}
	for _, p := range f.repo.inserted {
		if want := map[schedule.VisitMode]string{schedule.ModeInPerson: siteA}[p.Mode]; p.LocationID != want {
			t.Errorf("the %s slot at %v is at %q, want %q", p.Mode, p.Start, p.LocationID, want)
		}
	}
}

// A rule moved to another location replaces its free slots there and keeps a busy one where it was booked.
func TestGenerate_ARuleThatMovesReplacesItsFreeSlotsAtTheNewLocation(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	var first, second string
	if err := owner.QueryRow(t.Context(), `SELECT id FROM location WHERE organization_id = $1`, clinicA).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(t.Context(), `INSERT INTO location (organization_id, name) VALUES ($1, 'Annex') RETURNING id`, clinicA).Scan(&second); err != nil {
		t.Fatal(err)
	}
	at := func(loc string) []schedule.Schedule {
		s := seeded()
		s[0].Rules = []schedule.Rule{{Day: 1, Start: "10:00", End: "12:00", Mode: schedule.ModeInPerson, LocationID: loc}}
		return s
	}
	if got, err := newService(app, monday, at(first)...).Generate(t.Context(), clinicA, scheduleA1); err != nil || got.Inserted != 8 {
		t.Fatalf("Generate() at the first site = %+v, %v; want 8 slots", got, err)
	}
	booked := monday.Add(7*24*time.Hour + 2*time.Hour) // next Monday 11:00
	if _, err := owner.Exec(t.Context(), `UPDATE slot SET status = 'busy' WHERE schedule_id = $1 AND lower(period) = $2`, scheduleA1, booked); err != nil {
		t.Fatal(err)
	}
	got, err := newService(app, monday, at(second)...).Generate(t.Context(), clinicA, scheduleA1)
	if err != nil || got.Deleted != 7 || got.Inserted != 7 {
		t.Fatalf("Generate() at the second site = %+v, %v; want the 7 free slots replaced", got, err)
	}
	var atFirst, atSecond int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE location_id = $2 AND status = 'busy' AND lower(period) = $4),
		count(*) FILTER (WHERE location_id = $3 AND status = 'free') FROM slot WHERE schedule_id = $1`, scheduleA1, first, second, booked).
		Scan(&atFirst, &atSecond); err != nil || atFirst != 1 || atSecond != 7 {
		t.Errorf("%d busy at the first site, %d free at the second, %v; want 1 and 7", atFirst, atSecond, err)
	}
	if again, err := newService(app, monday, at(second)...).Generate(t.Context(), clinicA, scheduleA1); err != nil || again.Inserted != 0 || again.Deleted != 0 {
		t.Errorf("a re-run = %+v, %v; want nothing inserted or deleted", again, err)
	}
}
