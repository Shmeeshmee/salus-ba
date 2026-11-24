package clinicexport_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/clinicexport"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seed makes the two clinics and an admin of clinic A as the owner, and returns the admin's account.
func seed(t *testing.T, owner *pgxpool.Pool) string {
	t.Helper()
	ctx := t.Context()
	if _, err := owner.Exec(ctx, `INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic A'), ($2, 'Synthetic clinic B')`, clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	var account string
	if err := owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id)
		VALUES (gen_random_uuid()::text || '@example.test', 'h', 'admin', $1) RETURNING id`, clinicA).Scan(&account); err != nil {
		t.Fatal(err)
	}
	return account
}

// in runs fn in a transaction of the app scoped to the clinic org.
func in(t *testing.T, app *pgxpool.Pool, repo clinicexport.Repository, org string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// An export's life as salus_app: one queued per clinic, seen only by its clinic, ready once.
func TestPostgresRepository_OneQueuedPerClinicSeenOnlyThere(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	account := seed(t, owner)
	repo := clinicexport.NewPostgresRepository(app)
	x := clinicexport.Export{OrganizationID: clinicA, RequestedBy: account, RequestedAt: asked, ExpiresAt: asked.Add(clinicexport.Window)}

	var first clinicexport.Export
	err := in(t, app, repo, clinicA, func(ctx context.Context) error {
		var ok bool
		var err error
		if first, ok, err = repo.Insert(ctx, x); err != nil || !ok {
			t.Errorf("Insert() = %v, %v; want inserted", ok, err)
			return err
		}
		if _, ok, err := repo.Insert(ctx, x); err != nil || ok {
			t.Errorf("second Insert() = %v, %v; want none while one is queued", ok, err)
		}
		got, err := repo.Queued(ctx, clinicA)
		if err != nil || got.ID != first.ID || got.Status != clinicexport.StatusQueued || !got.ExpiresAt.Equal(x.ExpiresAt) {
			t.Errorf("Queued() = %+v, %v; want the first", got, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = in(t, app, repo, clinicB, func(ctx context.Context) error {
		if _, err := repo.ByID(ctx, first.ID); !errors.Is(err, clinicexport.ErrNotFound) {
			t.Errorf("ByID() in clinic B = %v, want ErrNotFound", err)
		}
		if ok, err := repo.MarkReady(ctx, clinicexport.Export{ID: first.ID, ReadyAt: asked, Size: 1, KEKID: "k", WrappedKey: []byte{1}}); err != nil || ok {
			t.Errorf("MarkReady() in clinic B = %v, %v; want nothing", ok, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = in(t, app, repo, clinicA, func(ctx context.Context) error {
		ready := clinicexport.Export{ID: first.ID, ReadyAt: asked.Add(time.Minute), Size: 42, KEKID: "k", WrappedKey: []byte{1, 2}}
		if ok, err := repo.MarkReady(ctx, ready); err != nil || !ok {
			t.Errorf("MarkReady() = %v, %v", ok, err)
		}
		if ok, err := repo.MarkFailed(ctx, first.ID); err != nil || ok {
			t.Errorf("MarkFailed() of a ready export = %v, %v; want nothing", ok, err)
		}
		got, err := repo.ByID(ctx, first.ID)
		if err != nil || got.Status != clinicexport.StatusReady || got.Size != 42 || got.KEKID != "k" || len(got.WrappedKey) != 2 {
			t.Errorf("ByID() = %+v, %v; want it ready", got, err)
		}
		if _, ok, err := repo.Insert(ctx, x); err != nil || !ok {
			t.Errorf("Insert() after the build = %v, %v; want a new one", ok, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// salus_app may not delete an export; salus_retention deletes those asked for before the cut-off.
func TestPostgresRepository_RetentionDeletesOldExports(t *testing.T) {
	t.Parallel()
	owner, retention := pgtest.NewRetentionDatabase(t)
	account := seed(t, owner)
	if _, err := owner.Exec(t.Context(), `INSERT INTO clinic_export (organization_id, requested_by_account_id, requested_at, expires_at)
		VALUES ($1, $2, $3, $4), ($1, $2, $5, $6)`, clinicA, account, asked, asked.Add(clinicexport.Window),
		asked.Add(time.Hour), asked.Add(time.Hour+clinicexport.Window)); err == nil {
		t.Fatal("two queued exports of one clinic were stored")
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO clinic_export (organization_id, requested_by_account_id, requested_at, expires_at, status)
		VALUES ($1, $2, $3, $4, 'failed'), ($1, $2, $5, $6, 'queued')`, clinicA, account, asked, asked.Add(clinicexport.Window),
		asked.Add(time.Hour), asked.Add(time.Hour+clinicexport.Window)); err != nil {
		t.Fatal(err)
	}
	repo := clinicexport.NewPostgresRepository(retention)
	ids, err := repo.RequestedBefore(t.Context(), asked.Add(time.Minute), 10)
	if err != nil || len(ids) != 1 {
		t.Fatalf("RequestedBefore() = %v, %v; want the older one", ids, err)
	}
	if ok, err := repo.Delete(t.Context(), ids[0]); err != nil || !ok {
		t.Errorf("Delete() = %v, %v", ok, err)
	}
	var left int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM clinic_export`).Scan(&left); err != nil || left != 1 {
		t.Errorf("left %d, %v; want 1", left, err)
	}
}
