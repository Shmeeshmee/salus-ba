package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The deadline job's scope (docs/clinic-cms WAVE5.md, D1, S3) sees the open requests of every target, the account's
// included, and may only mark that their warning was queued: nothing else of a request, and nothing of a closed one.
func TestDataRequestDeadlines_TheJobSeesOpenRequestsAndOnlyMarksTheirWarning(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	toAccount := fileRequest(t, owner, p[0].account, p[0].patient, nil, p[0].account)
	toClinic := fileRequest(t, owner, p[1].account, p[1].patient, clinicB, p[1].account)
	withdrawn := fileRequest(t, owner, p[0].account, p[0].patient, clinicA, p[0].account)
	if _, err := owner.Exec(t.Context(), `UPDATE data_request SET status = 'withdrawn', version = 2 WHERE id = $1`, withdrawn); err != nil {
		t.Fatal(err)
	}
	job := map[string]string{"app.data_request_deadlines": "all"}
	if got, want := idsIn(t, app, job, "SELECT id::text FROM data_request"), sorted(toAccount, toClinic); !slices.Equal(got, want) {
		t.Errorf("the job sees %v, want the open ones %v", got, want)
	}
	exec := func(stmt string, args ...any) (int64, error) {
		var n int64
		err := asApp(t, app, job, func(ctx context.Context, db postgres.Querier) error {
			tag, err := db.Exec(ctx, stmt, args...)
			n = tag.RowsAffected()
			return err
		})
		return n, err
	}
	if n, err := exec(`UPDATE data_request SET deadline_warned_at = now() WHERE id = $1`, toAccount); err != nil || n != 1 {
		t.Errorf("the job marks a warning: %d rows, %v; want 1", n, err)
	}
	if n, err := exec(`UPDATE data_request SET deadline_warned_at = now() WHERE id = $1`, withdrawn); err != nil || n != 0 {
		t.Errorf("the job marks a withdrawn request: %d rows, %v; want none", n, err)
	}
	for name, stmt := range map[string]string{
		"status":  `UPDATE data_request SET status = 'in_progress' WHERE id = $1`,
		"version": `UPDATE data_request SET version = version + 1 WHERE id = $1`,
		"answer":  `UPDATE data_request SET status = 'refused', answered_at = now(), answer_code = 'excessive' WHERE id = $1`,
	} {
		if _, err := exec(stmt, toClinic); err == nil {
			t.Errorf("the job changes the %s: no error, want a refusal", name)
		}
	}
	if got := idsIn(t, app, map[string]string{"app.data_request_deadlines": "other"}, "SELECT id::text FROM data_request"); len(got) != 0 {
		t.Errorf("another value of the setting sees %v, want none", got)
	}
}
