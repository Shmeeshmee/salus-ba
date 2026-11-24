package worker_test

import (
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/datarequest"
)

// The deadline job, as the app role: an open request 3 days before its deadline gets its 7-day warning queued once,
// with the request's id only, audited as the job's; a request due in a month gets none yet (docs/clinic-cms WAVE5.md
// D1, S3).
func TestWorker_QueuesEachDeadlineWarningOnce(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	var account, soon string
	if err := db.owner.QueryRow(t.Context(), `INSERT INTO account (email, password_hash, role) VALUES ('dr@example.test', 'h', 'patient') RETURNING id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	var now time.Time
	if err := db.owner.QueryRow(t.Context(), "SELECT now()").Scan(&now); err != nil {
		t.Fatal(err)
	}
	if err := db.owner.QueryRow(t.Context(), `INSERT INTO data_request (account_id, filed_by_account_id, kind, received_at, due_at)
		VALUES ($1, $1, 'access', $2, $3) RETURNING id`, account, now.AddDate(0, -1, 0), now.Add(72*time.Hour)).Scan(&soon); err != nil {
		t.Fatal(err)
	}
	if _, err := db.owner.Exec(t.Context(), `INSERT INTO data_request (account_id, filed_by_account_id, kind, received_at, due_at)
		VALUES ($1, $1, 'erasure', $2, $3)`, account, now, now.AddDate(0, 1, 0)); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if kind, logs := runJobOn(t, db, liveKit(t), testFiles(t), datarequest.DeadlinesArgs{}, nil); kind != river.EventKindJobCompleted {
			t.Fatalf("job ended %s: %s", kind, logs)
		}
	}
	var n int
	var args string
	if err := db.owner.QueryRow(t.Context(), `SELECT count(*), min(args::text) FROM river_job WHERE kind = 'data_request_deadline_warning'`).Scan(&n, &args); err != nil {
		t.Fatal(err)
	}
	if want := `{"days": 7, "request_id": "` + soon + `"}`; n != 1 || args != want {
		t.Errorf("%d warnings, %s; want one: %s", n, args, want)
	}
	if err := db.owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event WHERE agent_name = 'data_request_deadlines'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("%d audit events of the job, %v; want one", n, err)
	}
}
