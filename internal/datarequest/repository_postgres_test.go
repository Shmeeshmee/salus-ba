package datarequest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seeded are the synthetic rows of the repository's tests: patient A's account and profile, clinic A's admin and a
// profile it registered.
type seeded struct{ account, profile, admin, registered string }

func seed(t *testing.T, owner *pgxpool.Pool) seeded {
	t.Helper()
	ctx := t.Context()
	if _, err := owner.Exec(ctx, `INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic A'), ($2, 'Synthetic clinic B')`, clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	var s seeded
	if err := owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role) VALUES ('pat@example.test', 'h', 'patient') RETURNING id`).Scan(&s.account); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO patient (account_id, given_name, family_name, birth_date) VALUES ($1, 'Ana', 'Synthetic', '1980-01-01') RETURNING id`, s.account).Scan(&s.profile); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id) VALUES ('adm@example.test', 'h', 'admin', $1) RETURNING id`, clinicA).Scan(&s.admin); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO patient (registered_by_organization_id, given_name, family_name, birth_date) VALUES ($1, 'Bo', 'Synthetic', '1970-02-02') RETURNING id`, clinicA).Scan(&s.registered); err != nil {
		t.Fatal(err)
	}
	return s
}

// dbNow is the database's time: a request is filed when it is received (data_request_guard).
func dbNow(t *testing.T, owner *pgxpool.Pool) time.Time {
	t.Helper()
	var now time.Time
	if err := owner.QueryRow(t.Context(), "SELECT now()").Scan(&now); err != nil {
		t.Fatal(err)
	}
	return now.UTC()
}

// within runs fn in a transaction of the app with scope set.
func within(t *testing.T, app *pgxpool.Pool, scope func(ctx context.Context) error, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := scope(ctx); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// A request's life as salus_app: the patient files and withdraws theirs, the clinic files for its registered patient
// and answers, each status change in the history; each scope sees its own; the deadline job sees the open ones.
func TestPostgresRepository_ARequestsLife(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seed(t, owner)
	repo := datarequest.NewPostgresRepository(app)
	now := dbNow(t, owner)
	asPatient := func(ctx context.Context) error { return repo.SetAccount(ctx, s.account) }
	asClinic := func(ctx context.Context) error { return repo.SetOrganization(ctx, clinicA) }

	var own, onBehalf datarequest.Request
	err := within(t, app, asPatient, func(ctx context.Context) error {
		var err error
		own, err = repo.Insert(ctx, datarequest.Request{AccountID: s.account, PatientID: s.profile, FiledBy: s.account,
			Kind: datarequest.KindAccess, ReceivedAt: now, DueAt: now.AddDate(0, 1, 0)})
		return err
	})
	if err != nil || own.ID == "" || own.Status != datarequest.StatusReceived || own.Version != 1 || own.Target() != datarequest.TargetAccount {
		t.Fatalf("the patient files: %+v, %v", own, err)
	}
	err = within(t, app, asClinic, func(ctx context.Context) error {
		var err error
		onBehalf, err = repo.Insert(ctx, datarequest.Request{PatientID: s.registered, OrganizationID: clinicA, FiledBy: s.admin,
			Kind: datarequest.KindErasure, ReceivedAt: now, DueAt: now.AddDate(0, 1, 0)})
		return err
	})
	if err != nil || onBehalf.AccountID != "" || onBehalf.OrganizationID != clinicA {
		t.Fatalf("the clinic files on behalf: %+v, %v", onBehalf, err)
	}
	err = within(t, app, asClinic, func(ctx context.Context) error {
		_, err := repo.Insert(ctx, datarequest.Request{PatientID: s.profile, OrganizationID: clinicA, FiledBy: s.admin,
			Kind: datarequest.KindErasure, ReceivedAt: now, DueAt: now.AddDate(0, 1, 0)})
		return err
	})
	if !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("the clinic files for a patient it did not register: %v, want ErrNotFound", err)
	}

	changeAsThePatient(t, app, repo, s, own, onBehalf.ID)
	changeAsTheClinic(t, app, repo, s, onBehalf, now)
}

// The deadline job sees the open requests of every target and marks their warnings (D1, S3).
func TestPostgresRepository_TheDeadlineJob(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	s := seed(t, owner)
	repo := datarequest.NewPostgresRepository(app)
	now := dbNow(t, owner)
	var soon, later string
	if err := owner.QueryRow(t.Context(), `INSERT INTO data_request (account_id, patient_id, filed_by_account_id, kind, received_at, due_at)
		VALUES ($1, $2, $1, 'access', $3, $4) RETURNING id`, s.account, s.profile, now.AddDate(0, -1, 0), now.Add(72*time.Hour)).Scan(&soon); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(t.Context(), `INSERT INTO data_request (patient_id, organization_id, filed_by_account_id, kind, received_at, due_at)
		VALUES ($1, $2, $3, 'erasure', $4, $5) RETURNING id`, s.registered, clinicA, s.admin, now, now.AddDate(0, 1, 0)).Scan(&later); err != nil {
		t.Fatal(err)
	}
	err := within(t, app, repo.SetDeadlines, func(ctx context.Context) error {
		open, err := repo.OpenDeadlines(ctx, now.Add(7*24*time.Hour))
		if err != nil || len(open) != 1 || open[0].Request.ID != soon || !open[0].WarnedAt.IsZero() {
			t.Errorf("OpenDeadlines = %+v, %v; want the account's one due in 3 days", open, err)
		}
		if ok, err := repo.MarkWarned(ctx, soon, time.Time{}, now); err != nil || !ok {
			t.Errorf("MarkWarned = %v, %v", ok, err)
		}
		if ok, err := repo.MarkWarned(ctx, soon, time.Time{}, now.Add(time.Minute)); err != nil || ok {
			t.Errorf("MarkWarned by an overlapping run = %v, %v; want false", ok, err)
		}
		open, err = repo.OpenDeadlines(ctx, now.AddDate(0, 2, 0))
		if err != nil || len(open) != 2 || !open[0].WarnedAt.Equal(now) || open[1].Request.ID != later {
			t.Errorf("OpenDeadlines = %+v, %v; want both, the first warned", open, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// changeAsThePatient: the patient sees their own request only, withdraws it once, and its history shows it.
func changeAsThePatient(t *testing.T, app *pgxpool.Pool, repo datarequest.Repository, s seeded, own datarequest.Request, other string) {
	t.Helper()
	asPatient := func(ctx context.Context) error { return repo.SetAccount(ctx, s.account) }
	onBehalf := datarequest.Request{ID: other}
	err := within(t, app, asPatient, func(ctx context.Context) error {
		mine, err := repo.ForAccount(ctx, s.account)
		if err != nil || len(mine) != 1 || mine[0].ID != own.ID {
			t.Errorf("ForAccount = %+v, %v", mine, err)
		}
		if _, err := repo.ByID(ctx, onBehalf.ID); !errors.Is(err, datarequest.ErrNotFound) {
			t.Errorf("the patient reads the clinic's: %v, want ErrNotFound", err)
		}
		w := own
		w.Status, w.Version = datarequest.StatusWithdrawn, 2
		if ok, err := repo.Update(ctx, w, s.account); err != nil || !ok {
			t.Errorf("withdraw: %v, %v", ok, err)
		}
		if ok, err := repo.Update(ctx, w, s.account); err != nil || ok {
			t.Errorf("withdraw at a stale version: %v, %v; want false", ok, err)
		}
		h, err := repo.History(ctx, own.ID)
		if err != nil || len(h) != 2 || h[0].Status != datarequest.StatusReceived || h[1].Status != datarequest.StatusWithdrawn || h[1].ChangedBy != s.account {
			t.Errorf("History = %+v, %v", h, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

}

// changeAsTheClinic: the clinic lists its request, extends and answers it; the extension is no status change.
func changeAsTheClinic(t *testing.T, app *pgxpool.Pool, repo datarequest.Repository, s seeded, onBehalf datarequest.Request, now time.Time) {
	t.Helper()
	asClinic := func(ctx context.Context) error { return repo.SetOrganization(ctx, clinicA) }
	err := within(t, app, asClinic, func(ctx context.Context) error {
		list, err := repo.ForClinic(ctx, clinicA)
		if err != nil || len(list) != 1 || list[0].ID != onBehalf.ID {
			t.Errorf("ForClinic = %+v, %v", list, err)
		}
		x := onBehalf
		x.ExtendedUntil, x.ExtensionReason, x.Version = x.DueAt.AddDate(0, 1, 0), datarequest.ExtensionComplex, 2
		if ok, err := repo.Update(ctx, x, s.admin); err != nil || !ok {
			t.Errorf("extend: %v, %v", ok, err)
		}
		x.Status, x.AnswerCode, x.AnsweredAt, x.Version = datarequest.StatusFulfilled, datarequest.AnswerDonePartlyKept, now, 3
		if ok, err := repo.Update(ctx, x, s.admin); err != nil || !ok {
			t.Errorf("answer: %v, %v", ok, err)
		}
		got, err := repo.ByID(ctx, onBehalf.ID)
		if err != nil || got.Status != datarequest.StatusFulfilled || got.AnswerCode != datarequest.AnswerDonePartlyKept ||
			!got.AnsweredAt.Equal(now) || got.ExtensionReason != datarequest.ExtensionComplex || got.Version != 3 {
			t.Errorf("ByID = %+v, %v", got, err)
		}
		h, err := repo.History(ctx, onBehalf.ID)
		if err != nil || len(h) != 2 || h[1].Status != datarequest.StatusFulfilled || !h[1].ChangedAt.Equal(now) {
			t.Errorf("History = %+v, %v (the extension is no status change)", h, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
