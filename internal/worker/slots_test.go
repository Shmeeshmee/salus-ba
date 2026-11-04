package worker_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/schedule"
)

// seedSchedules creates, as the owner, the listed clinic clinicID with n practitioners named like seeded's,
// each with a schedule every day from 00:00 to 23:00 by the hour, so slots exist whatever the time; it
// returns the schedule ids.
func seedSchedules(t *testing.T, db logins, n int) []string {
	t.Helper()
	if _, err := db.owner.Exec(t.Context(), `INSERT INTO organization (id, name, listed) VALUES ($1, 'Synthetic clinic', true)`, clinicID); err != nil {
		t.Fatal(err)
	}
	rules := make([]string, 0, 7)
	for day := 1; day <= 7; day++ {
		rules = append(rules, `{"day": `+strconv.Itoa(day)+`, "start": "00:00", "end": "23:00"}`)
	}
	rows, err := db.owner.Query(t.Context(), `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
			SELECT $1, 'Bo', 'Example' FROM generate_series(1, $2) RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING id, organization_id)
		INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
		SELECT organization_id, id, 60, $3 FROM r RETURNING id`, clinicID, n, "["+strings.Join(rules, ",")+"]")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}

func countWhere(t *testing.T, db logins, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.owner.QueryRow(t.Context(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSlotsGenerate_GeneratesTheJobsScheduleAndLogsOnlyCounts(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	ids := seedSchedules(t, db, 2)

	ended, logs := runWorkerJob(t, db, schedule.SlotsGenerateArgs{ScheduleID: ids[0], OrganizationID: clinicID})

	if ended != river.EventKindJobCompleted {
		t.Fatalf("job %s, want completed; log:\n%s", ended, logs)
	}
	// 28 days of 23 hourly slots, give or take the current hour.
	first := countWhere(t, db, "SELECT count(*) FROM slot WHERE schedule_id = $1", ids[0])
	if first < 27*23 || first > 28*23 || countWhere(t, db, "SELECT count(*) FROM slot WHERE schedule_id = $1", ids[1]) != 0 {
		t.Errorf("%d slots of the job's schedule and some of the other, want 28 days' worth of the job's only", first)
	}
	if want := `"msg":"slots generated","schedules":1,"inserted":` + strconv.Itoa(first) + `,"deleted":0`; !strings.Contains(logs, want) {
		t.Errorf("log lacks %s:\n%s", want, logs)
	}
	assertNoPersonalData(t, logs)
}

func TestSlotsGenerate_WithoutIDsGeneratesEveryScheduleOfAListedClinic(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	ids := seedSchedules(t, db, 3)

	ended, logs := runWorkerJob(t, db, schedule.SlotsGenerateArgs{})

	if ended != river.EventKindJobCompleted || !strings.Contains(logs, `"schedules":3`) {
		t.Fatalf("job %s, want completed for 3 schedules; log:\n%s", ended, logs)
	}
	for _, id := range ids {
		if countWhere(t, db, "SELECT count(*) FROM slot WHERE schedule_id = $1", id) == 0 {
			t.Errorf("schedule %s has no slots", id)
		}
	}
	assertNoPersonalData(t, logs)
}

// A schedule that does not exist never will (schedules are not deleted): the job is cancelled, not retried.
func TestSlotsGenerate_CancelsTheJobOfAnUnknownSchedule(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	seedSchedules(t, db, 1)

	ended, logs := runWorkerJob(t, db, schedule.SlotsGenerateArgs{ScheduleID: "00000000-0000-4000-8000-000000000000", OrganizationID: clinicID})

	if ended != river.EventKindJobCancelled {
		t.Errorf("job %s, want cancelled; log:\n%s", ended, logs)
	}
}
