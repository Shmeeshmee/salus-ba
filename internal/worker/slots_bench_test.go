package worker

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// benchSchedules is the plan's load: 50 practitioners of one listed clinic.
const benchSchedules = 50

// seedBench creates, as the owner, a listed clinic in Sarajevo with benchSchedules roles, each with a
// schedule on weekdays 08:00 to 16:00 in quarter-hour slots: 32 slots a day; from 09:00 on a Monday to 28
// days later, 639 a schedule (that day's first five are past, the last morning adds four), 31 950 in all.
func seedBench(b *testing.B, owner *pgxpool.Pool) {
	b.Helper()
	rules := make([]string, 0, 5)
	for day := 1; day <= 5; day++ {
		rules = append(rules, fmt.Sprintf(`{"day": %d, "start": "08:00", "end": "16:00"}`, day))
	}
	if _, err := owner.Exec(b.Context(), `WITH o AS (INSERT INTO organization (name, listed) VALUES ('Synthetic clinic', true) RETURNING id),
			p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
				SELECT o.id, 'Ada', 'Example' FROM o, generate_series(1, $1) RETURNING id, organization_id),
			r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id)
		INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
		SELECT organization_id, id, 15, $2 FROM r`, benchSchedules, "["+strings.Join(rules, ",")+"]"); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkGenerate is the hourly slots_generate run over 50 schedules, 28 days ahead, through the services
// the worker builds (salus_app, row-level security on): "first" from no slots (each iteration starts from an
// empty slot table), "rerun" on the slots already there, which changes nothing. make bench-slots; the
// numbers are in docs/booking/PROGRESS.md, package 5.
func BenchmarkGenerate(b *testing.B) {
	owner, app := pgtest.NewAppDatabase(b)
	seedBench(b, owner)
	monday := time.Date(2030, 1, 7, 8, 0, 0, 0, time.UTC) // 09:00 in Sarajevo
	slots := newSlots(app, clock.Fixed{T: monday})
	const want = 639 * benchSchedules

	b.Run("first", func(b *testing.B) {
		for range b.N {
			b.StopTimer()
			if _, err := owner.Exec(b.Context(), "DELETE FROM slot"); err != nil {
				b.Fatal(err)
			}
			b.StartTimer()
			g, err := slots.GenerateAll(b.Context())
			if err != nil || g.Schedules != benchSchedules || g.Inserted != want {
				b.Fatalf("GenerateAll() = %+v, %v; want %d slots of %d schedules", g, err, want, benchSchedules)
			}
		}
	})
	b.Run("rerun", func(b *testing.B) {
		for range b.N {
			g, err := slots.GenerateAll(b.Context())
			if err != nil || g.Schedules != benchSchedules || g.Inserted != 0 || g.Deleted != 0 {
				b.Fatalf("GenerateAll() again = %+v, %v; want nothing changed", g, err)
			}
		}
	})
}
