package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

func patient(email string) auth.Account {
	return auth.Account{Email: email, Role: auth.RolePatient, PasswordHash: "$argon2id$synthetic"}
}

func TestCreate_StoresTheAccountAndFindsItByEmailAndID(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	ctx := t.Context()

	id, err := repo.Create(ctx, patient("ana@example.test"))
	if err != nil || id == "" {
		t.Fatalf("Create() = %q, %v", id, err)
	}
	want := patient("ana@example.test")
	want.ID = id
	for name, find := range map[string]func() (auth.Account, error){
		"ByEmail": func() (auth.Account, error) { return repo.ByEmail(ctx, "ana@example.test") },
		"ByID":    func() (auth.Account, error) { return repo.ByID(ctx, id) },
	} {
		got, err := find()
		if err != nil || got != want {
			t.Errorf("%s() = %+v, %v;\nwant %+v", name, got, err, want)
		}
	}
}

func TestByID_ReadsAStaffAccountsClinicAndEnrolment(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	var id string
	err := owner.QueryRow(t.Context(), `INSERT INTO account (email, password_hash, role, organization_id, mfa_enrolled_at)
		VALUES ('ben@example.test', 'h', 'admin', $1, '2026-11-04T09:30:00Z') RETURNING id`, clinicID).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	got, err := auth.NewPostgresRepository(app).ByID(t.Context(), id)
	if err != nil || got.Role != auth.RoleAdmin || got.OrganizationID != clinicID || !got.MFAEnrolled() {
		t.Errorf("ByID() = %+v, %v; want an enrolled admin of the clinic", got, err)
	}
}

func TestCreate_ReportsATakenEmailAndKeepsTheTransactionUsable(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	if _, err := repo.Create(t.Context(), patient("ana@example.test")); err != nil {
		t.Fatal(err)
	}
	err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if _, err := repo.Create(ctx, patient("ana@example.test")); !errors.Is(err, auth.ErrEmailTaken) {
			t.Errorf("Create() with a taken e-mail = %v, want ErrEmailTaken", err)
		}
		_, err := repo.ByEmail(ctx, "ana@example.test")
		return err
	})
	if err != nil {
		t.Errorf("the transaction after a taken e-mail: %v, want it usable", err)
	}
}

func TestFind_ReportsAnUnknownAccount(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	ctx := t.Context()
	if _, err := repo.ByEmail(ctx, "nobody@example.test"); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("ByEmail() = %v, want ErrNotFound", err)
	}
	for _, id := range []string{accountID, "not-a-uuid"} {
		if _, err := repo.ByID(ctx, id); !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("ByID(%q) = %v, want ErrNotFound", id, err)
		}
	}
}

func TestRepository_PassesDatabaseFailuresOn(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	app.Close()
	ctx := t.Context()
	if _, err := repo.Create(ctx, patient("ana@example.test")); err == nil || errors.Is(err, auth.ErrEmailTaken) {
		t.Errorf("Create() on a closed pool = %v, want the failure", err)
	}
	if _, err := repo.ByEmail(ctx, "ana@example.test"); err == nil || errors.Is(err, auth.ErrNotFound) {
		t.Errorf("ByEmail() on a closed pool = %v, want the failure", err)
	}
	if _, err := repo.ByID(ctx, accountID); err == nil || errors.Is(err, auth.ErrNotFound) {
		t.Errorf("ByID() on a closed pool = %v, want the failure", err)
	}
	if err := repo.SetPasswordHash(ctx, accountID, "h"); err == nil {
		t.Error("SetPasswordHash() on a closed pool: no error")
	}
}

func TestSetPasswordHash_ReplacesTheHash(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	ctx := t.Context()
	id, _ := repo.Create(ctx, patient("ana@example.test"))
	if err := repo.SetPasswordHash(ctx, id, "$argon2id$newer"); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.ByID(ctx, id); got.PasswordHash != "$argon2id$newer" {
		t.Errorf("hash = %q after SetPasswordHash", got.PasswordHash)
	}
	if err := repo.SetPasswordHash(ctx, "not-a-uuid", "h"); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("SetPasswordHash(not a UUID) = %v, want ErrNotFound", err)
	}
}

// newPostgresService is the service on the Postgres repository, logged in as the application role, with
// cheap argon2 parameters. Audit events go to the fake auditor; internal/api's tests check them in the
// database.
func newPostgresService(t *testing.T, p auth.Argon2Params) (*auth.Service, *pgxpool.Pool) {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	return auth.NewService(auth.NewPostgresRepository(app), auth.NewArgon2Hasher(p, 1), &fakeAuditor{}, postgres.NewTransactor(app)), owner
}

func count(t *testing.T, db *pgxpool.Pool, query string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(t.Context(), query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRegister_StoresTheNormalisedEmailAndOnlyTheArgon2Hash(t *testing.T) {
	t.Parallel()
	svc, owner := newPostgresService(t, cheap)
	if _, err := svc.Register(t.Context(), "Ana@Example.test", password); err != nil {
		t.Fatal(err)
	}
	var email, hash string
	if err := owner.QueryRow(t.Context(), "SELECT email, password_hash FROM account").Scan(&email, &hash); err != nil {
		t.Fatal(err)
	}
	if email != "ana@example.test" || !strings.HasPrefix(hash, "$argon2id$v=19$") || strings.Contains(hash, password) {
		t.Errorf("stored %q, %q; want the normalised e-mail and an argon2id hash only", email, hash)
	}
}

func TestCheckPassword_ReHashesOlderParametersInTheDatabase(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	tx, repo := postgres.NewTransactor(app), auth.NewPostgresRepository(app)
	older := auth.NewService(repo, auth.NewArgon2Hasher(cheap, 1), &fakeAuditor{}, tx)
	newer := auth.NewService(repo, auth.NewArgon2Hasher(auth.Argon2Params{Memory: 128, Time: 2, Threads: 1}, 1), &fakeAuditor{}, tx)
	if _, err := older.Register(t.Context(), "ana@example.test", password); err != nil {
		t.Fatal(err)
	}
	if _, err := newer.CheckPassword(t.Context(), "ana@example.test", password); err != nil {
		t.Fatalf("CheckPassword() = %v", err)
	}
	if n := count(t, owner, "SELECT count(*) FROM account WHERE password_hash LIKE '$argon2id$v=19$m=128,t=2,p=1$%'"); n != 1 {
		t.Error("the hash was not replaced with the newer parameters")
	}
}
