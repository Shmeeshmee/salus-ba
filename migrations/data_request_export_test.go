package migrations_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The patient's export (docs/clinic-cms WAVE5.md D2, S4) is seen and queued in their own scope only, for an access or
// portability request of theirs, one queued at a time; once ready, data_request_export_fulfil answers the request done
// with a history entry by the job, and only in the scope of its account.
func TestDataRequestExport_OwnScopeOneQueuedAndFulfil(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	mine := fileRequest(t, owner, p[0].account, p[0].patient, nil, p[0].account)
	second := fileRequest(t, owner, p[0].account, p[0].patient, clinicA, p[0].account)
	theirs := fileRequest(t, owner, p[1].account, p[1].patient, nil, p[1].account)
	a := map[string]string{"app.account_id": p[0].account}
	const queue = `INSERT INTO data_request_export (data_request_id, account_id, requested_at) VALUES ($1, $2, now())`

	if err := asApp(t, app, a, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, queue, theirs, p[0].account)
		return err
	}); err == nil {
		t.Fatal("queued an export of another patient's request")
	}
	if err := asApp(t, app, a, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, queue, mine, p[0].account)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := asApp(t, app, a, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, queue, second, p[0].account)
		return err
	}); err == nil {
		t.Fatal("queued a second export while one is queued")
	}
	var seen int
	if err := asApp(t, app, map[string]string{"app.account_id": p[1].account}, func(ctx context.Context, db postgres.Querier) error {
		return db.QueryRow(ctx, `SELECT count(*) FROM data_request_export`).Scan(&seen)
	}); err != nil || seen != 0 {
		t.Fatalf("another patient sees %d exports (%v)", seen, err)
	}

	fulfil := func(scope map[string]string) bool {
		t.Helper()
		var done bool
		if err := asApp(t, app, scope, func(ctx context.Context, db postgres.Querier) error {
			return db.QueryRow(ctx, `SELECT data_request_export_fulfil($1)`, mine).Scan(&done)
		}); err != nil {
			t.Fatal(err)
		}
		return done
	}
	if fulfil(a) {
		t.Fatal("fulfilled a request whose export is only queued")
	}
	if err := asApp(t, app, a, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE data_request_export SET status = 'ready', ready_at = now(),
			expires_at = now() + interval '7 days', size = 1, kek_id = 'k', wrapped_key = '\x00' WHERE data_request_id = $1`, mine)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if fulfil(map[string]string{"app.account_id": p[1].account}) {
		t.Fatal("another patient's scope fulfilled the request")
	}
	if !fulfil(a) {
		t.Fatal("the ready export did not fulfil its request")
	}
	if fulfil(a) {
		t.Fatal("fulfilled the request twice")
	}
	var status, code string
	var changes int
	if err := owner.QueryRow(t.Context(), `SELECT r.status, r.answer_code, (SELECT count(*) FROM data_request_change c
		WHERE c.data_request_id = r.id AND c.status = 'fulfilled' AND c.changed_by_account_id IS NULL)
		FROM data_request r WHERE r.id = $1`, mine).Scan(&status, &code, &changes); err != nil {
		t.Fatal(err)
	}
	if status != "fulfilled" || code != "done" || changes != 1 {
		t.Fatalf("request %s %s with %d history entries", status, code, changes)
	}
}
