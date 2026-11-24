package datarequest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seedRequests makes, as the owner, a patient's account with its profile and two access requests to the account, and
// returns the account and the requests.
func seedRequests(t *testing.T, owner *pgxpool.Pool) (account string, requests [2]string) {
	t.Helper()
	ctx := t.Context()
	if err := owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role)
		VALUES (gen_random_uuid()::text || '@example.test', 'h', 'patient') RETURNING id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	var patient string
	if err := owner.QueryRow(ctx, `INSERT INTO patient (account_id, given_name, family_name, birth_date)
		VALUES ($1, 'Synthetic', 'Patient', '1980-02-03') RETURNING id`, account).Scan(&patient); err != nil {
		t.Fatal(err)
	}
	for i := range requests {
		if err := owner.QueryRow(ctx, `INSERT INTO data_request (account_id, patient_id, filed_by_account_id, kind, received_at, due_at)
			VALUES ($1, $2, $1, 'access', now(), now() + interval '1 month') RETURNING id`, account, patient).Scan(&requests[i]); err != nil {
			t.Fatal(err)
		}
	}
	return account, requests
}

// inAccount runs fn in a transaction of the app scoped to the account.
func inAccount(t *testing.T, app *pgxpool.Pool, repo datarequest.ExportRepository, account string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetAccount(ctx, account); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// An export's life as salus_app: one queued per patient, seen only in their scope, ready once, and its request
// fulfilled once it is ready.
func TestExportRepository_OneQueuedSeenByThePatientReadyAndFulfilled(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	account, req := seedRequests(t, owner)
	repo := datarequest.NewExportRepository(app)
	asked := now
	x := datarequest.StoredExport{RequestID: req[0], AccountID: account, RequestedAt: asked}

	err := inAccount(t, app, repo, account, func(ctx context.Context) error {
		if ok, err := repo.InsertExport(ctx, x); err != nil || !ok {
			t.Errorf("InsertExport = %v, %v; want inserted", ok, err)
		}
		second := x
		second.RequestID = req[1]
		if ok, err := repo.InsertExport(ctx, second); err != nil || ok {
			t.Errorf("a second queued: %v, %v; want not inserted", ok, err)
		}
		got, err := repo.ExportOf(ctx, req[0])
		if err != nil || got.Status != datarequest.ExportQueued || got.Kind != datarequest.KindAccess || got.OrganizationID != "" ||
			!got.RequestedAt.Equal(asked) || got.AccountID != account {
			t.Errorf("ExportOf = %+v, %v", got, err)
		}
		if done, err := repo.Fulfil(ctx, req[0]); err != nil || done {
			t.Errorf("Fulfil before ready = %v, %v; want false", done, err)
		}
		ready := got
		ready.ReadyAt, ready.ExpiresAt, ready.Size, ready.KEKID, ready.WrappedKey = asked, asked.Add(datarequest.ExportWindow), 10, "k", []byte{1}
		if ok, err := repo.MarkExportReady(ctx, ready); err != nil || !ok {
			t.Errorf("MarkExportReady = %v, %v", ok, err)
		}
		if ok, _ := repo.MarkExportReady(ctx, ready); ok {
			t.Error("marked ready twice")
		}
		if done, err := repo.Fulfil(ctx, req[0]); err != nil || !done {
			t.Errorf("Fulfil = %v, %v; want true", done, err)
		}
		if ok, err := repo.MarkExportFailed(ctx, req[0]); err != nil || ok {
			t.Errorf("failed a ready export: %v, %v", ok, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := inAccount(t, app, repo, "00000000-0000-4000-8000-0000000000ff", func(ctx context.Context) error {
		_, err := repo.ExportOf(ctx, req[0])
		return err
	}); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("another account's scope: %v, want ErrNotFound", err)
	}
	var status string
	if err := owner.QueryRow(t.Context(), `SELECT status FROM data_request WHERE id = $1`, req[0]).Scan(&status); err != nil || status != "fulfilled" {
		t.Errorf("request %s (%v), want fulfilled", status, err)
	}
}

// The retention role lists the exports ready or failed before the cut-off, in every account, and deletes them.
func TestExportRepository_RetentionDeletesDoneExports(t *testing.T) {
	t.Parallel()
	owner, retention := pgtest.NewRetentionDatabase(t)
	account, req := seedRequests(t, owner)
	if _, err := owner.Exec(t.Context(), `INSERT INTO data_request_export (data_request_id, account_id, requested_at, status, ready_at,
		expires_at, size, kek_id, wrapped_key) VALUES ($1, $3, now() - interval '9 days', 'ready', now() - interval '8 days',
		now() - interval '1 day', 1, 'k', '\x01'), ($2, $3, now() - interval '8 days', 'queued', NULL, NULL, NULL, NULL, NULL)`, req[0], req[1], account); err != nil {
		t.Fatal(err)
	}
	repo := datarequest.NewExportRepository(retention)
	var cutoff time.Time
	if err := owner.QueryRow(t.Context(), `SELECT now() - interval '7 days'`).Scan(&cutoff); err != nil {
		t.Fatal(err)
	}
	ids, err := repo.ExportsDoneBefore(t.Context(), cutoff, 10)
	if err != nil || len(ids) != 2 || ids[0] != req[0] || ids[1] != req[1] {
		t.Fatalf("ExportsDoneBefore = %v, %v; want the ready one and the one stuck queued", ids, err)
	}
	if ok, err := repo.DeleteExport(t.Context(), req[0]); err != nil || !ok {
		t.Errorf("DeleteExport = %v, %v", ok, err)
	}
	for _, bad := range []string{"not-a-uuid"} {
		if ok, err := repo.DeleteExport(t.Context(), bad); err != nil || ok {
			t.Errorf("DeleteExport(%q) = %v, %v", bad, ok, err)
		}
		if _, err := repo.ExportOf(t.Context(), bad); !errors.Is(err, datarequest.ErrNotFound) {
			t.Errorf("ExportOf(%q) = %v", bad, err)
		}
	}
}

// The build is enqueued in the transaction with opaque ids only.
func TestRiverExportJobs_EnqueuesTheBuildWithOnlyIDs(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := datarequest.NewRiverExportJobs(client)
	args := datarequest.ExportArgs{RequestID: request1, AccountID: accountA}
	if err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error { return jobs.EnqueueExport(ctx, args) }); err != nil {
		t.Fatal(err)
	}
	var kind, got string
	if err := owner.QueryRow(t.Context(), "SELECT kind, args::text FROM river_job").Scan(&kind, &got); err != nil {
		t.Fatal(err)
	}
	want := `{"account_id": "` + accountA + `", "data_request_id": "` + request1 + `"}`
	if kind != "data_request_export_build" || got != want {
		t.Errorf("job %s with %s, want data_request_export_build with %s", kind, got, want)
	}
	if err := jobs.EnqueueExport(t.Context(), args); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("outside a transaction: %v, want postgres.ErrNoTx", err)
	}
}

// The export of a request withdrawn while it was built is never made ready.
func TestExportRepository_NotReadyOnceTheRequestIsWithdrawn(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	account, req := seedRequests(t, owner)
	repo := datarequest.NewExportRepository(app)
	x := datarequest.StoredExport{RequestID: req[0], AccountID: account, RequestedAt: now}
	if err := inAccount(t, app, repo, account, func(ctx context.Context) error {
		_, err := repo.InsertExport(ctx, x)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE data_request SET status = 'withdrawn', version = 2 WHERE id = $1`, req[0]); err != nil {
		t.Fatal(err)
	}
	x.ReadyAt, x.ExpiresAt, x.Size, x.KEKID, x.WrappedKey = now, now.Add(datarequest.ExportWindow), 1, "k", []byte{1}
	if err := inAccount(t, app, repo, account, func(ctx context.Context) error {
		if ok, err := repo.MarkExportReady(ctx, x); err != nil || ok {
			t.Errorf("MarkExportReady = %v, %v; want false", ok, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
