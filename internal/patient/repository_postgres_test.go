package patient_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seedAccounts creates patient accounts A and B as the owner, with a profile for A only.
func seedAccounts(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	_, err := owner.Exec(t.Context(), `INSERT INTO account (id, email, password_hash, role) VALUES
		($1, 'pat.a@example.test', 'h', 'patient'), ($2, 'pat.b@example.test', 'h', 'patient')`, accountA, accountB)
	if err != nil {
		t.Fatal(err)
	}
	_, err = owner.Exec(t.Context(), `INSERT INTO patient (id, account_id, given_name, family_name, birth_date, phone, updated_at)
		VALUES ($1, $2, 'Ada', 'Example', '1990-01-01', '+38760000001', '2026-11-01T08:00:00Z')`, patientA, accountA)
	if err != nil {
		t.Fatal(err)
	}
}

// inAccount runs fn in a transaction on app scoped to the account.
func inAccount(t *testing.T, app *pgxpool.Pool, repo patient.Repository, account string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetAccount(ctx, account); err != nil {
			return err
		}
		return fn(ctx)
	})
}

var profileA = patient.Patient{ID: patientA, AccountID: accountA, GivenName: "Ada", FamilyName: "Example",
	BirthDate: date(1990, 1, 1), Phone: "+38760000001", VersionID: 1, LastUpdated: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)}

func samePatient(a, b patient.Patient) bool {
	return a.ID == b.ID && a.AccountID == b.AccountID && a.GivenName == b.GivenName && a.FamilyName == b.FamilyName &&
		a.BirthDate.Equal(b.BirthDate) && a.Phone == b.Phone && a.VersionID == b.VersionID && a.LastUpdated.Equal(b.LastUpdated)
}

func TestByAccountAndByID_SeeOnlyTheAccountSet(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	repo := patient.NewPostgresRepository(app)

	var byAccount, byID patient.Patient
	err := inAccount(t, app, repo, accountA, func(ctx context.Context) error {
		var err error
		if byAccount, err = repo.ByAccount(ctx, accountA); err != nil {
			return err
		}
		byID, err = repo.ByID(ctx, patientA)
		return err
	})
	if err != nil || !samePatient(byAccount, profileA) || !samePatient(byID, profileA) {
		t.Errorf("in account A: ByAccount = %+v, ByID = %+v, %v;\nwant %+v", byAccount, byID, err, profileA)
	}
	for _, lookup := range []func(ctx context.Context) error{
		func(ctx context.Context) error { _, err := repo.ByAccount(ctx, accountA); return err },
		func(ctx context.Context) error { _, err := repo.ByID(ctx, patientA); return err },
		func(ctx context.Context) error { _, err := repo.ByID(ctx, "not-a-uuid"); return err },
		func(ctx context.Context) error { _, err := repo.ByAccount(ctx, "not-a-uuid"); return err },
	} {
		if err := inAccount(t, app, repo, accountB, lookup); !errors.Is(err, patient.ErrNotFound) {
			t.Errorf("lookup of profile A in account B: %v, want ErrNotFound", err)
		}
	}
}

func TestCreate_StoresTheProfileOnceAndWithoutAPhoneAsNone(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	repo := patient.NewPostgresRepository(app)
	at := time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

	in := patient.Patient{AccountID: accountB, GivenName: "Bo", FamilyName: "Example", BirthDate: date(1991, 2, 2)}
	var got patient.Patient
	err := inAccount(t, app, repo, accountB, func(ctx context.Context) error {
		var err error
		got, err = repo.Create(ctx, in, at)
		return err
	})
	want := in
	want.ID, want.VersionID, want.LastUpdated = got.ID, 1, at
	if err != nil || got.ID == "" || !samePatient(got, want) {
		t.Fatalf("Create() = %+v, %v;\nwant %+v", got, err, want)
	}
	var phoneIsNull bool
	if err := owner.QueryRow(t.Context(), "SELECT phone IS NULL FROM patient WHERE id = $1", got.ID).Scan(&phoneIsNull); err != nil || !phoneIsNull {
		t.Errorf("phone stored as NULL: %v, %v; want true", phoneIsNull, err)
	}

	// A second create, in the same transaction as a later statement, gives ErrConflict and leaves the
	// transaction usable.
	err = inAccount(t, app, repo, accountB, func(ctx context.Context) error {
		if _, err := repo.Create(ctx, in, at); !errors.Is(err, patient.ErrConflict) {
			t.Errorf("second Create() = %v, want ErrConflict", err)
		}
		_, err := repo.ByAccount(ctx, accountB)
		return err
	})
	if err != nil {
		t.Errorf("a read after the conflict: %v", err)
	}
	// Another account's profile is refused by row-level security.
	err = inAccount(t, app, repo, accountB, func(ctx context.Context) error {
		_, err := repo.Create(ctx, patient.Patient{AccountID: accountA, GivenName: "X", FamilyName: "Y", BirthDate: date(1990, 1, 1)}, at)
		return err
	})
	if err == nil || errors.Is(err, patient.ErrConflict) {
		t.Errorf("Create() for account A in account B's scope = %v, want the row-level security refusal", err)
	}
	if err := inAccount(t, app, repo, accountB, func(ctx context.Context) error {
		_, err := repo.Create(ctx, patient.Patient{AccountID: "not-a-uuid"}, at)
		return err
	}); err == nil {
		t.Error("Create() with an account id that is not a UUID succeeded")
	}
}

func TestUpdate_ChangesOnlyTheCurrentVersion(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedAccounts(t, owner)
	repo := patient.NewPostgresRepository(app)
	at := time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

	in := patient.Patient{AccountID: accountA, GivenName: "Ada", FamilyName: "Renamed", BirthDate: date(1990, 1, 2), VersionID: 1}
	var got patient.Patient
	err := inAccount(t, app, repo, accountA, func(ctx context.Context) error {
		var err error
		got, err = repo.Update(ctx, in, at)
		return err
	})
	want := in
	want.ID, want.VersionID, want.LastUpdated = patientA, 2, at
	if err != nil || !samePatient(got, want) {
		t.Fatalf("Update() = %+v, %v;\nwant %+v (the phone removed)", got, err, want)
	}
	for name, scope := range map[string]string{"stale version": accountA, "another account's scope": accountB} {
		err := inAccount(t, app, repo, scope, func(ctx context.Context) error {
			in.VersionID = 1
			if name == "another account's scope" {
				in.VersionID = 2
			}
			_, err := repo.Update(ctx, in, at)
			return err
		})
		if !errors.Is(err, patient.ErrConflict) {
			t.Errorf("%s: Update() = %v, want ErrConflict", name, err)
		}
	}
	if err := inAccount(t, app, repo, accountA, func(ctx context.Context) error {
		_, err := repo.Update(ctx, patient.Patient{AccountID: "not-a-uuid"}, at)
		return err
	}); !errors.Is(err, patient.ErrConflict) {
		t.Errorf("Update() with an account id that is not a UUID = %v, want ErrConflict", err)
	}
}

func TestSetAccount_NeedsATransaction(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	if err := patient.NewPostgresRepository(app).SetAccount(t.Context(), accountA); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("SetAccount() outside a transaction = %v, want ErrNoTx", err)
	}
}
