package worker

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

type fakeEraser struct {
	err  error
	runs int
}

func (f *fakeEraser) Run(context.Context) ([]datarequest.Erased, error) {
	f.runs++
	return []datarequest.Erased{{AccountID: "00000000-0000-4000-8000-0000000000a1", RequestID: "00000000-0000-4000-8000-0000000000d1"}}, f.err
}

// The daily account_erasure job runs the erasure once, and fails (so River retries and alerts) when it does; it is
// cancelled, not retried, while the erasure is not built in this binary.
func TestAccountErasureWorker_RunsTheErasure(t *testing.T) {
	t.Parallel()
	f := &fakeEraser{}
	w := &accountErasureWorker{erasure: f, log: slog.New(slog.DiscardHandler)}
	if err := w.Work(t.Context(), &river.Job[AccountErasureArgs]{}); err != nil || f.runs != 1 {
		t.Errorf("Work() = %v after %d runs, want nil after 1", err, f.runs)
	}
	f.err = errors.New("synthetic failure")
	if err := w.Work(t.Context(), &river.Job[AccountErasureArgs]{}); !errors.Is(err, f.err) {
		t.Errorf("Work() = %v, want the run's error", err)
	}
	var cancel *river.JobCancelError
	if err := (&accountErasureWorker{}).Work(t.Context(), &river.Job[AccountErasureArgs]{}); !errors.As(err, &cancel) {
		t.Errorf("Work() without the erasure = %v, want a cancellation", err)
	}
	if (AccountErasureArgs{}).Kind() != "account_erasure" {
		t.Errorf("kind %q", AccountErasureArgs{}.Kind())
	}
}

// The worker's erasure on Postgres as the app role: a closed account past its erasure date is blanked and its erasure
// request fulfilled (done: no clinic keeps a record); a second run changes nothing (D3 wiring).
func TestAccountErasureWorker_ErasesADueAccountOnce(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	var account, request string
	err := owner.QueryRow(ctx, `WITH a AS (INSERT INTO account (email, password_hash, role, closed_at, erase_after)
			VALUES ('closed@example.test', 'h', 'patient', now() - interval '22 days', now() - interval '1 day') RETURNING id),
		p AS (INSERT INTO patient (account_id, given_name, family_name, birth_date) SELECT id, 'Ida', 'Example', '1980-01-01' FROM a)
		INSERT INTO data_request (received_at, due_at, account_id, filed_by_account_id, kind)
		SELECT now(), now() + interval '9 days', id, id, 'erasure' FROM a RETURNING account_id, id`).Scan(&account, &request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `UPDATE data_request SET status = 'in_progress', version = 2 WHERE id = $1`, request); err != nil {
		t.Fatal(err)
	}
	w := &accountErasureWorker{erasure: newErasure(app, clock.System{}), log: slog.New(slog.DiscardHandler)}
	for range 2 {
		if err := w.Work(ctx, &river.Job[AccountErasureArgs]{}); err != nil {
			t.Fatalf("Work() = %v", err)
		}
	}
	var email, hash, status, code string
	if err := owner.QueryRow(ctx, `SELECT a.email, a.password_hash, r.status, r.answer_code FROM account a
		JOIN data_request r ON r.account_id = a.id WHERE a.id = $1`, account).Scan(&email, &hash, &status, &code); err != nil {
		t.Fatal(err)
	}
	if email != account+"@erased.invalid" || hash != "" || status != "fulfilled" || code != "done" {
		t.Errorf("after erasure: %q %q %s %s, want blanked and the request fulfilled done", email, hash, status, code)
	}
	var changes int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM data_request_change WHERE data_request_id = $1 AND status = 'fulfilled'`, request).Scan(&changes); err != nil || changes != 1 {
		t.Errorf("%d fulfilments in the history, %v; want one: a second run adds none", changes, err)
	}
}
