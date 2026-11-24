package worker

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The hourly run generates the free times of every clinic, listed or not (docs/clinic-cms WAVE4 R1), through the
// slot service newSlots builds: clinic A is listed, B is not, C is listed but suspended; B also has an inactive role.
const (
	unlistedA, unlistedB, unlistedC = "a0000000-0000-4000-8000-0000000000a1", "b0000000-0000-4000-8000-0000000000b1", "c0000000-0000-4000-8000-0000000000c1"
	scheduleOfA, scheduleOfB        = "a5000000-0000-4000-8000-0000000000a1", "b5000000-0000-4000-8000-0000000000b1"
	scheduleOfC, scheduleRetired    = "c5000000-0000-4000-8000-0000000000c1", "b5000000-0000-4000-8000-0000000000b2"
)

// genMonday is Monday 7 January 2030, 09:00 in Sarajevo.
var genMonday = time.Date(2030, 1, 7, 8, 0, 0, 0, time.UTC)

type genLogins struct{ owner, app, stats *pgxpool.Pool }

func newGenLogins(t testing.TB) genLogins {
	t.Helper()
	dbURL := pgtest.NewDatabaseURL(t)
	return genLogins{owner: open(t, dbURL), app: open(t, pgtest.AppURL(t, dbURL)), stats: open(t, pgtest.StatsURL(t, dbURL))}
}

// seedEveryClinic creates, as the owner, the three clinics with one schedule each, from 10:00 to 11:00 on Mondays (A),
// Tuesdays (B) and Wednesdays (C), and B's inactive role with a schedule on Thursdays.
func seedEveryClinic(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	for _, q := range []string{
		`INSERT INTO organization (id, name, listed) VALUES ('` + unlistedA + `', 'Synthetic clinic A', true),
			('` + unlistedB + `', 'Synthetic clinic B', false), ('` + unlistedC + `', 'Synthetic clinic C', true)`,
		`INSERT INTO subscription_plan (code, name, price_per_seat_minor) VALUES ('synthetic', 'Synthetic plan', 0) ON CONFLICT DO NOTHING`,
		`INSERT INTO clinic_subscription (organization_id, plan_code, status) VALUES ('` + unlistedC + `', 'synthetic', 'suspended')`,
	} {
		if _, err := owner.Exec(t.Context(), q); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range []struct {
		org, schedule string
		day           int
		active        bool
	}{{unlistedA, scheduleOfA, 1, true}, {unlistedB, scheduleOfB, 2, true}, {unlistedC, scheduleOfC, 3, true}, {unlistedB, scheduleRetired, 4, false}} {
		if _, err := owner.Exec(t.Context(), `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
				VALUES ($1, 'Ada', 'Example') RETURNING id, organization_id),
			r AS (INSERT INTO practitioner_role (organization_id, practitioner_id, active) SELECT organization_id, id, $4 FROM p RETURNING id, organization_id)
			INSERT INTO schedule (id, organization_id, practitioner_role_id, slot_minutes, availability)
			SELECT $2, organization_id, id, 60, jsonb_build_array(jsonb_build_object('day', $3::int, 'start', '10:00', 'end', '11:00')) FROM r`,
			r.org, r.schedule, r.day, r.active); err != nil {
			t.Fatal(err)
		}
	}
}

// slotStarts lists, as the owner, the starts of the schedule's slots in order.
func slotStarts(t *testing.T, owner *pgxpool.Pool, scheduleID string) []time.Time {
	t.Helper()
	rows, err := owner.Query(t.Context(), "SELECT lower(period) FROM slot WHERE schedule_id = $1 ORDER BY lower(period)", scheduleID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []time.Time
	for rows.Next() {
		var s time.Time
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGenerateAll_RollsAnUnlistedClinicsSlotsForwardOnTheHourlyRun(t *testing.T) {
	t.Parallel()
	db := newGenLogins(t)
	seedEveryClinic(t, db.owner)

	got, err := newSlots(db.app, db.stats, clock.Fixed{T: genMonday}).GenerateAll(t.Context())
	if err != nil || got.Schedules != 3 {
		t.Fatalf("GenerateAll() = %+v, %v; want A, B and C (not the inactive role's)", got, err)
	}
	if b := slotStarts(t, db.owner, scheduleOfB); len(b) != 4 { // the Tuesdays of the 28 days from Monday 7 January
		t.Fatalf("unlisted clinic B has %d slots, want 4", len(b))
	}

	week := genMonday.Add(7 * 24 * time.Hour)
	if _, err := newSlots(db.app, db.stats, clock.Fixed{T: week}).GenerateAll(t.Context()); err != nil {
		t.Fatal(err)
	}
	b := slotStarts(t, db.owner, scheduleOfB)
	if len(b) != 5 || !b[len(b)-1].After(genMonday.Add(28*24*time.Hour)) {
		t.Errorf("unlisted clinic B a week later has slots %v, want 5 rolled past the first run's 28 days", b)
	}
	if n := len(slotStarts(t, db.owner, scheduleOfC)); n != 5 {
		t.Errorf("suspended clinic C has %d slots, want 5 (its bookings stop elsewhere)", n)
	}
	if n := len(slotStarts(t, db.owner, scheduleRetired)); n != 0 {
		t.Errorf("the inactive role has %d slots, want none", n)
	}
}

func TestGenerateAll_LeavesTheDirectorysResultsAsTheyWere(t *testing.T) {
	t.Parallel()
	db := newGenLogins(t)
	seedEveryClinic(t, db.owner)
	svc := newSlots(db.app, db.stats, clock.Fixed{T: genMonday})
	if _, err := svc.GenerateAll(t.Context()); err != nil {
		t.Fatal(err)
	}

	to := genMonday.Add(28 * 24 * time.Hour)
	for _, c := range []struct {
		schedule string
		want     int
	}{{scheduleOfA, 4}, {scheduleOfB, 0}, {scheduleOfC, 0}} {
		got, err := svc.Search(t.Context(), c.schedule, genMonday, to)
		if err != nil || len(got) != c.want || (len(got) > 0 && got[0].Status != slot.StatusFree) {
			t.Errorf("directory Search(%s) = %d slots, %v; want %d free", c.schedule, len(got), err, c.want)
		}
	}
}

// BenchmarkGenerateAll1000Clinics is the hourly run over 1 000 clinics with one schedule each (weekdays 08:00 to
// 16:00 in hour slots), half of them unlisted: "first" from no slots, "rerun" on the slots already there (the hourly
// job's steady state). Numbers in docs/clinic-cms/PROGRESS.md, R1.
// go test -run '^$' -bench GenerateAll1000Clinics -benchtime 3x ./internal/worker
func BenchmarkGenerateAll1000Clinics(b *testing.B) {
	db := newGenLogins(b)
	if _, err := db.owner.Exec(b.Context(), `WITH o AS (INSERT INTO organization (name, listed)
			SELECT 'Synthetic clinic', i % 2 = 0 FROM generate_series(1, 1000) i RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Ada', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id)
		INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
		SELECT organization_id, id, 60, (SELECT jsonb_agg(jsonb_build_object('day', d, 'start', '08:00', 'end', '16:00'))
			FROM generate_series(1, 5) d) FROM r`); err != nil {
		b.Fatal(err)
	}
	svc := newSlots(db.app, db.stats, clock.Fixed{T: genMonday})
	b.Run("first", func(b *testing.B) {
		for range b.N {
			b.StopTimer()
			if _, err := db.owner.Exec(b.Context(), "DELETE FROM slot"); err != nil {
				b.Fatal(err)
			}
			b.StartTimer()
			if g, err := svc.GenerateAll(b.Context()); err != nil || g.Schedules != 1000 {
				b.Fatalf("GenerateAll() = %+v, %v; want 1000 schedules", g, err)
			}
		}
	})
	b.Run("rerun", func(b *testing.B) {
		for range b.N {
			if g, err := svc.GenerateAll(b.Context()); err != nil || g.Inserted != 0 || g.Deleted != 0 {
				b.Fatalf("GenerateAll() again = %+v, %v; want nothing changed", g, err)
			}
		}
	})
}
