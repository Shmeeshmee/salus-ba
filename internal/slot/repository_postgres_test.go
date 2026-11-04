package slot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// monday is a Monday morning, 09:00 in Sarajevo, far enough ahead for the database's own clock not to matter.
var monday = time.Date(2030, 1, 7, 8, 0, 0, 0, time.UTC)

// seed creates, as the owner, clinics A (listed) and B with a role each, and their schedules: A on Mondays
// 10:00 to 12:00 in hour slots, B on Tuesdays.
func seed(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	ctx := t.Context()
	if _, err := owner.Exec(ctx, `INSERT INTO organization (id, name, listed) VALUES ($1, 'Synthetic clinic A', true), ($2, 'Synthetic clinic B', false)`,
		clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	for _, r := range []struct{ role, org, schedule, rules string }{
		{roleA1, clinicA, scheduleA1, `[{"day": 1, "start": "10:00", "end": "12:00"}]`},
		{roleB, clinicB, scheduleB, `[{"day": 2, "start": "10:00", "end": "11:00"}]`},
	} {
		if _, err := owner.Exec(ctx, `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
				VALUES ($2, 'Ada', 'Example') RETURNING id, organization_id),
			r AS (INSERT INTO practitioner_role (id, organization_id, practitioner_id) SELECT $1, organization_id, id FROM p RETURNING id, organization_id)
			INSERT INTO schedule (id, organization_id, practitioner_role_id, slot_minutes, availability)
			SELECT $3, organization_id, id, 60, $4 FROM r`, r.role, r.org, r.schedule, r.rules); err != nil {
			t.Fatal(err)
		}
	}
}

// addSlot inserts, as the owner, a slot of schedule A1 from start for an hour, and returns its id.
func addSlot(t *testing.T, owner *pgxpool.Pool, scheduleID, org string, start time.Time, status slot.Status) string {
	t.Helper()
	var id string
	if err := owner.QueryRow(t.Context(), `INSERT INTO slot (schedule_id, organization_id, period, status)
		VALUES ($1, $2, tstzrange($3, $4), $5) RETURNING id`, scheduleID, org, start, start.Add(time.Hour), status).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

type slotRow struct {
	start  time.Time
	status slot.Status
}

// slots lists, as the owner, the slots of the schedule by start.
func slots(t *testing.T, owner *pgxpool.Pool, scheduleID string) []slotRow {
	t.Helper()
	rows, err := owner.Query(t.Context(), "SELECT lower(period), status FROM slot WHERE schedule_id = $1 ORDER BY lower(period)", scheduleID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []slotRow
	for rows.Next() {
		var r slotRow
		if err := rows.Scan(&r.start, &r.status); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// seeded is what seed stored: schedule A1 on Mondays 10:00 to 12:00 in hour slots in Sarajevo, and B on
// Tuesdays in Lisbon.
func seeded() []schedule.Schedule {
	return []schedule.Schedule{
		{ID: scheduleA1, OrganizationID: clinicA, PractitionerRoleID: roleA1, SlotMinutes: 60, Rules: []schedule.Rule{{Day: 1, Start: "10:00", End: "12:00"}}},
		{ID: scheduleB, OrganizationID: clinicB, PractitionerRoleID: roleB, SlotMinutes: 60, Rules: []schedule.Rule{{Day: 2, Start: "10:00", End: "11:00"}}},
	}
}

// newService builds the slot service on app at the time at, with the schedules and zones of the fakes (the
// schedule and organization services are tested in their domains).
func newService(app *pgxpool.Pool, at time.Time, schedules ...schedule.Schedule) *slot.Service {
	if schedules == nil {
		schedules = seeded()
	}
	l := &log{}
	return slot.NewService(slot.NewPostgresRepository(app), postgres.NewTransactor(app), clock.Fixed{T: at},
		&fakeSchedules{log: l, schedules: schedules},
		fakeClinics{log: l, zones: map[string]string{clinicA: "Europe/Sarajevo", clinicB: "Europe/Lisbon"}})
}

func TestGenerate_AddsTheSlotsOnceAndARerunChangesNothing(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	svc := newService(app, monday)

	first, err := svc.Generate(t.Context(), clinicA, scheduleA1)
	if err != nil || first.Inserted != 8 || first.Deleted != 0 { // four Mondays from today, 10:00 and 11:00 each
		t.Fatalf("Generate() = %+v, %v; want 8 inserted", first, err)
	}
	got := slots(t, owner, scheduleA1)
	if len(got) != 8 || !got[0].start.Equal(monday.Add(time.Hour)) || got[0].status != slot.StatusFree {
		t.Errorf("slots %v, want 8 free from 10:00 today", got)
	}

	again, err := svc.Generate(t.Context(), clinicA, scheduleA1)
	if err != nil || again.Inserted != 0 || again.Deleted != 0 || len(slots(t, owner, scheduleA1)) != 8 {
		t.Errorf("a re-run = %+v, %v; want nothing inserted or deleted", again, err)
	}
	if n := len(slots(t, owner, scheduleB)); n != 0 {
		t.Errorf("clinic B's schedule has %d slots, want none", n)
	}
}

func TestGenerate_ANarrowedRuleDeletesOnlyFutureFreeSlotsAndABusySlotSurvives(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	past := addSlot(t, owner, scheduleA1, clinicA, monday.Add(-7*24*time.Hour+time.Hour), slot.StatusFree) // last Monday 10:00
	if _, err := newService(app, monday).Generate(t.Context(), clinicA, scheduleA1); err != nil {
		t.Fatal(err)
	}
	// Next Monday 11:00 is booked; then the rule becomes 10:00 to 11:00 and the length half an hour.
	if _, err := owner.Exec(t.Context(), `UPDATE slot SET status = 'busy' WHERE schedule_id = $1 AND lower(period) = $2`,
		scheduleA1, monday.Add(7*24*time.Hour+2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	narrowed := seeded()
	narrowed[0].SlotMinutes, narrowed[0].Rules = 30, []schedule.Rule{{Day: 1, Start: "10:00", End: "11:30"}}

	got, err := newService(app, monday, narrowed...).Generate(t.Context(), clinicA, scheduleA1)

	// Every future free hour slot goes (7); each Monday gets 10:00, 10:30 and 11:00, except next Monday's
	// 11:00, which overlaps the busy slot (11).
	if err != nil || got.Deleted != 7 || got.Inserted != 11 {
		t.Errorf("Generate() = %+v, %v; want 7 deleted and 11 inserted", got, err)
	}
	var busy, pastKept int
	for _, s := range slots(t, owner, scheduleA1) {
		switch {
		case s.status == slot.StatusBusy && s.start.Equal(monday.Add(7*24*time.Hour+2*time.Hour)):
			busy++
		case s.start.Before(monday):
			pastKept++
		}
	}
	if busy != 1 || pastKept != 1 {
		t.Errorf("busy slot kept %d, past free slot %s kept %d; want both kept", busy, past, pastKept)
	}
}

// in runs fn in a transaction on app with the clinic org set.
func in(t *testing.T, app *pgxpool.Pool, org string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := postgres.SetLocal(ctx, "app.organization_id", org); err != nil {
			return err
		}
		return fn(ctx)
	})
}

func TestHold_JoinsTheCallersTransactionAndTakesOnlyAFreeSlotOfTheClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	free := addSlot(t, owner, scheduleA1, clinicA, monday.Add(time.Hour), slot.StatusFree)
	svc := newService(app, monday)

	// The booking fails after the hold: the slot stays free.
	errBooking := errors.New("the booking failed")
	err := in(t, app, clinicA, func(ctx context.Context) error {
		if _, err := svc.Hold(ctx, clinicA, free); err != nil {
			return err
		}
		return errBooking
	})
	if got := slots(t, owner, scheduleA1); !errors.Is(err, errBooking) || got[0].status != slot.StatusFree {
		t.Fatalf("after a rolled-back hold: %v, %v; want the slot free", got, err)
	}

	held, err := svc.Hold(t.Context(), clinicA, free)
	if err != nil || held.ID != free || held.Status != slot.StatusBusy || held.VersionID != 2 || held.ScheduleID != scheduleA1 ||
		!held.Start.Equal(monday.Add(time.Hour)) || !held.End.Equal(monday.Add(2*time.Hour)) || !held.LastUpdated.Equal(monday) {
		t.Errorf("Hold() = %+v, %v; want the slot busy at version 2", held, err)
	}
	for name, id := range map[string]string{"a busy slot": free, "another clinic's": free, "not a UUID": "x"} {
		org := clinicA
		if name == "another clinic's" {
			org = clinicB
		}
		if _, err := svc.Hold(t.Context(), org, id); !errors.Is(err, slot.ErrUnavailable) {
			t.Errorf("Hold(%s) = %v, want ErrUnavailable", name, err)
		}
	}
}

func TestRelease_FreesABusySlotOnlyBeforeItStarts(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	future := addSlot(t, owner, scheduleA1, clinicA, monday.Add(time.Hour), slot.StatusBusy)
	started := addSlot(t, owner, scheduleA1, clinicA, monday.Add(-30*time.Minute), slot.StatusBusy)
	svc := newService(app, monday)

	for _, tt := range []struct {
		name, id, org string
		want          bool
	}{
		{name: "another clinic's", id: future, org: clinicB},
		{name: "a started slot", id: started, org: clinicA},
		{name: "not a UUID", id: "x", org: clinicA},
		{name: "a future busy slot", id: future, org: clinicA, want: true},
		{name: "a free slot", id: future, org: clinicA},
	} {
		if freed, err := svc.Release(t.Context(), tt.org, tt.id); err != nil || freed != tt.want {
			t.Errorf("Release(%s) = %v, %v; want %v", tt.name, freed, err, tt.want)
		}
	}
	got := slots(t, owner, scheduleA1)
	if got[0].status != slot.StatusBusy || got[1].status != slot.StatusFree {
		t.Errorf("slots %v, want the started one busy and the future one free", got)
	}
}

func TestSearch_GivesTheFreeSlotsOfListedClinicsInTheRange(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	first := addSlot(t, owner, scheduleA1, clinicA, monday.Add(time.Hour), slot.StatusFree)
	addSlot(t, owner, scheduleA1, clinicA, monday.Add(2*time.Hour), slot.StatusBusy)
	addSlot(t, owner, scheduleA1, clinicA, monday.Add(3*time.Hour), slot.StatusFree) // at the end of the range
	addSlot(t, owner, scheduleB, clinicB, monday.Add(time.Hour), slot.StatusFree)    // unlisted
	svc := newService(app, monday)

	got, err := svc.Search(t.Context(), scheduleA1, monday, monday.Add(3*time.Hour))
	if err != nil || len(got) != 1 || got[0].ID != first || got[0].Status != slot.StatusFree || got[0].OrganizationID != clinicA {
		t.Errorf("Search(A1) = %+v, %v; want the free 10:00 slot only", got, err)
	}
	if got, err := svc.Search(t.Context(), scheduleB, monday, monday.Add(3*time.Hour)); err != nil || len(got) != 0 {
		t.Errorf("Search(B, unlisted) = %+v, %v; want none", got, err)
	}
	if got, err := svc.Search(t.Context(), "x", monday, monday.Add(3*time.Hour)); err != nil || len(got) != 0 {
		t.Errorf("Search(not a UUID) = %+v, %v; want none", got, err)
	}
}

func TestPurger_DeletesFreeSlotsOfEveryClinicThirtyDaysAfterTheyEnd(t *testing.T) {
	t.Parallel()
	owner, ret := pgtest.NewRetentionDatabase(t)
	seed(t, owner)
	cutOff := monday.Add(-30 * 24 * time.Hour)
	addSlot(t, owner, scheduleB, clinicB, cutOff.Add(-time.Hour-time.Second), slot.StatusFree) // ended a second before
	addSlot(t, owner, scheduleB, clinicB, cutOff.Add(-3*time.Hour), slot.StatusFree)
	addSlot(t, owner, scheduleA1, clinicA, cutOff.Add(-time.Hour+time.Second), slot.StatusFree) // ends a second after
	addSlot(t, owner, scheduleA1, clinicA, cutOff.Add(-10*time.Hour), slot.StatusBusy)

	n, err := slot.NewPurger(slot.NewPostgresRepository(ret)).Purge(t.Context(), cutOff)

	if err != nil || n != 2 {
		t.Errorf("Purge() = %d, %v; want the two free slots that ended before the cut-off", n, err)
	}
	if a, b := len(slots(t, owner, scheduleA1)), len(slots(t, owner, scheduleB)); a != 2 || b != 0 {
		t.Errorf("%d slots of A and %d of B left, want the busy and the recent one of A", a, b)
	}
}
