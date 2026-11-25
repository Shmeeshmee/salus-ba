package migrations_test

import (
	"context"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A build attempt claims the export's object key in the patient's scope (docs/clinic-cms WAVE5.md D7); an export is
// never ready without the key of its archive.
func TestDataRequestExportObjectKey_ClaimedAndRequiredWhenReady(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _ := seedConsents(t, owner)
	mine := fileRequest(t, owner, p[0].account, p[0].patient, nil, p[0].account)
	a := map[string]string{"app.account_id": p[0].account}
	if err := asApp(t, app, a, func(ctx context.Context, db postgres.Querier) error {
		if _, err := db.Exec(ctx, `INSERT INTO data_request_export (data_request_id, account_id, requested_at) VALUES ($1, $2, now())`,
			mine, p[0].account); err != nil {
			return err
		}
		_, err := db.Exec(ctx, `UPDATE data_request_export SET object_key = 'datarequest/x/1' WHERE data_request_id = $1`, mine)
		return err
	}); err != nil {
		t.Fatalf("the patient's scope could not claim the key: %v", err)
	}
	if err := asApp(t, app, a, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, `UPDATE data_request_export SET status = 'ready', ready_at = now(), expires_at = now() + interval '7 days',
			size = 1, kek_id = 'k', wrapped_key = '\x00', object_key = NULL WHERE data_request_id = $1`, mine)
		return err
	}); err == nil {
		t.Fatal("an export was ready without its object key")
	}
}
