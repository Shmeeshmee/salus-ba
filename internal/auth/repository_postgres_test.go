package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/clock"
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
	failures := map[string]error{}
	_, failures["TOTPSecret"] = repo.TOTPSecret(ctx, accountID)
	failures["SetTOTPSecret"] = repo.SetTOTPSecret(ctx, accountID, []byte("s"))
	failures["ConfirmMFA"] = repo.ConfirmMFA(ctx, accountID, 1, now)
	_, failures["UseTOTPStep"] = repo.UseTOTPStep(ctx, accountID, 1)
	_, failures["CreateSession"] = repo.CreateSession(ctx, auth.Session{AccountID: accountID}, []byte("h"), now)
	_, failures["SessionByTokenHash"] = repo.SessionByTokenHash(ctx, []byte("h"), now)
	failures["RevokeSession"] = repo.RevokeSession(ctx, sessionID, now)
	for name, err := range failures {
		if err == nil || errors.Is(err, auth.ErrNotFound) || errors.Is(err, auth.ErrAlreadyEnrolled) {
			t.Errorf("%s() on a closed pool = %v, want the failure", name, err)
		}
	}
}

const sessionID = "5e550000-0000-4000-8000-000000000001"

func TestSessions_AreFoundByTokenHashWhileOpen(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	ctx := t.Context()
	var staffID string
	err := owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id, mfa_enrolled_at)
		VALUES ('ben@example.test', 'h', 'admin', $1, now()) RETURNING id`, clinicID).Scan(&staffID)
	if err != nil {
		t.Fatal(err)
	}
	want := auth.Session{AccountID: staffID, OrganizationID: clinicID, MFAVerified: true, ExpiresAt: now.Add(12 * time.Hour)}

	id, err := repo.CreateSession(ctx, want, []byte("token hash"), now)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	want.ID = id
	got, err := repo.SessionByTokenHash(ctx, []byte("token hash"), now)
	if err != nil || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Fatalf("SessionByTokenHash() = %+v, %v;\nwant %+v", got, err, want)
	}
	got.ExpiresAt = want.ExpiresAt // same instant, possibly another zone
	if got != want {
		t.Errorf("SessionByTokenHash() = %+v,\nwant %+v", got, want)
	}
	var created time.Time
	if err := owner.QueryRow(ctx, "SELECT created_at FROM session").Scan(&created); err != nil || !created.Equal(now) {
		t.Errorf("created_at = %v, %v; want the service's now", created, err)
	}

	for name, find := range map[string]func() error{
		"another token": func() error { _, err := repo.SessionByTokenHash(ctx, []byte("other"), now); return err },
		"at expiry":     func() error { _, err := repo.SessionByTokenHash(ctx, []byte("token hash"), want.ExpiresAt); return err },
	} {
		if err := find(); !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("%s: SessionByTokenHash() = %v, want ErrNotFound", name, err)
		}
	}

	if err := repo.RevokeSession(ctx, id, now.Add(time.Minute)); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	if _, err := repo.SessionByTokenHash(ctx, []byte("token hash"), now.Add(2*time.Minute)); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("SessionByTokenHash() after revocation = %v, want ErrNotFound", err)
	}
	for _, id := range []string{id, sessionID, "not-a-uuid"} {
		if err := repo.RevokeSession(ctx, id, now); !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("RevokeSession(%q) of no open session = %v, want ErrNotFound", id, err)
		}
	}
}

func TestSessions_OfAPatientHaveNoClinic(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	ctx := t.Context()
	accountID, _ := repo.Create(ctx, patient("ana@example.test"))
	if _, err := repo.CreateSession(ctx, auth.Session{AccountID: accountID, ExpiresAt: now.Add(time.Hour)}, []byte("h"), now); err != nil {
		t.Fatal(err)
	}
	got, err := repo.SessionByTokenHash(ctx, []byte("h"), now)
	if err != nil || got.OrganizationID != "" || got.MFAVerified {
		t.Errorf("SessionByTokenHash() = %+v, %v; want a patient's session with no clinic", got, err)
	}
	if _, err := repo.CreateSession(ctx, auth.Session{AccountID: "not-a-uuid"}, []byte("h2"), now); err == nil {
		t.Error("CreateSession() for an account id that is not a UUID: no error")
	}
}

func TestTOTP_SecretEnrolmentAndUsedSteps(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	ctx := t.Context()
	id, _ := repo.Create(ctx, patient("ana@example.test"))

	if got, err := repo.TOTPSecret(ctx, id); err != nil || got != nil {
		t.Errorf("TOTPSecret() before enrolment = %q, %v; want none", got, err)
	}
	for _, sealed := range []string{"first", "second"} { // a repeated enrol replaces the pending secret
		if err := repo.SetTOTPSecret(ctx, id, []byte(sealed)); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := repo.TOTPSecret(ctx, id); err != nil || string(got) != "second" {
		t.Errorf("TOTPSecret() = %q, %v; want the last one stored", got, err)
	}
	if err := repo.ConfirmMFA(ctx, id, 100, now); err != nil {
		t.Fatalf("ConfirmMFA() error = %v", err)
	}
	if a, _ := repo.ByID(ctx, id); !a.MFAEnrolledAt.Equal(now) {
		t.Errorf("enrolled at %v, want %v", a.MFAEnrolledAt, now)
	}
	if err := repo.ConfirmMFA(ctx, id, 101, now); !errors.Is(err, auth.ErrAlreadyEnrolled) {
		t.Errorf("a second ConfirmMFA() = %v, want ErrAlreadyEnrolled", err)
	}
	for _, tt := range []struct {
		step int64
		want bool
	}{{100, false}, {99, false}, {101, true}, {101, false}, {103, true}} {
		if got, err := repo.UseTOTPStep(ctx, id, tt.step); err != nil || got != tt.want {
			t.Errorf("UseTOTPStep(%d) = %v, %v; want %v", tt.step, got, err, tt.want)
		}
	}
}

func TestTOTP_ReportsAnUnknownAccount(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	ctx := t.Context()
	for _, id := range []string{accountID, "not-a-uuid"} {
		if _, err := repo.TOTPSecret(ctx, id); !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("TOTPSecret(%q) = %v, want ErrNotFound", id, err)
		}
		if err := repo.SetTOTPSecret(ctx, id, []byte("s")); !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("SetTOTPSecret(%q) = %v, want ErrNotFound", id, err)
		}
		if err := repo.ConfirmMFA(ctx, id, 1, now); !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("ConfirmMFA(%q) = %v, want ErrNotFound", id, err)
		}
		if ok, err := repo.UseTOTPStep(ctx, id, 1); ok || (err != nil && !errors.Is(err, auth.ErrNotFound)) {
			t.Errorf("UseTOTPStep(%q) = %v, %v; want false", id, ok, err)
		}
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
	return auth.NewService(postgresPorts(app, auth.NewArgon2Hasher(p, 1))), owner
}

// postgresPorts are the Postgres repository and transactor on db, with fakes for the rest.
func postgresPorts(db *pgxpool.Pool, h auth.Hasher) auth.Ports {
	return auth.Ports{
		Repo: auth.NewPostgresRepository(db), Hasher: h, Audit: &fakeAuditor{}, Tx: postgres.NewTransactor(db),
		Clock: clock.Fixed{T: now}, Challenges: &fakeChallenges{challenges: map[string]*fakeChallenge{}},
		Counters: &fakeCounters{counts: map[string]int64{}, window: map[string]time.Duration{}}, Box: &fakeBox{},
		RateLimitKey: []byte("k"),
	}
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
	if _, err := svc.Register(t.Context(), "Ana@Example.test", password, "", ""); err != nil {
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

func TestLogin_ReHashesOlderParametersInTheDatabase(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	older := auth.NewService(postgresPorts(app, auth.NewArgon2Hasher(cheap, 1)))
	newer := auth.NewService(postgresPorts(app, auth.NewArgon2Hasher(auth.Argon2Params{Memory: 128, Time: 2, Threads: 1}, 1)))
	if _, err := older.Register(t.Context(), "ana@example.test", password, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := newer.Login(t.Context(), "ana@example.test", password, clientIP, "", ""); err != nil {
		t.Fatalf("Login() = %v", err)
	}
	if n := count(t, owner, "SELECT count(*) FROM account WHERE password_hash LIKE '$argon2id$v=19$m=128,t=2,p=1$%'"); n != 1 {
		t.Error("the hash was not replaced with the newer parameters")
	}
}

func TestLanguages_AreSavedByRegistrationSignInAndTheChoiceInTheDatabase(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	svc := auth.NewService(postgresPorts(app, auth.NewArgon2Hasher(cheap, 1)))
	a, err := svc.Register(t.Context(), "ana@example.test", password, "ar", "de")
	if err != nil {
		t.Fatal(err)
	}
	languages := func() (chosen, detected *string) {
		t.Helper()
		if err := owner.QueryRow(t.Context(), "SELECT language::text, detected_language::text FROM account").Scan(&chosen, &detected); err != nil {
			t.Fatal(err)
		}
		return chosen, detected
	}
	if chosen, detected := languages(); chosen == nil || *chosen != "ar" || detected == nil || *detected != "de" {
		t.Errorf("after registration: %v, %v; want ar, de", chosen, detected)
	}
	if _, err := svc.Login(t.Context(), "ana@example.test", password, clientIP, "", "tr"); err != nil {
		t.Fatal(err)
	}
	if chosen, detected := languages(); *chosen != "ar" || *detected != "tr" {
		t.Errorf("after sign-in: %s, %s; want ar kept, tr detected", *chosen, *detected)
	}
	if _, err := svc.SetLanguage(t.Context(), a.ID, ""); err != nil {
		t.Fatal(err)
	}
	if chosen, detected := languages(); chosen != nil || *detected != "tr" {
		t.Errorf("after clearing: %v, %s; want no choice, tr kept", chosen, *detected)
	}
}

func TestByID_ReadsAPractitionersPractitioner(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	const practitionerID = "a3000000-0000-4000-8000-0000000000a1"
	var id string
	if _, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic')`, clinicID); err != nil {
		t.Fatal(err)
	}
	err := owner.QueryRow(t.Context(), `WITH p AS (INSERT INTO practitioner (id, organization_id, given_name, family_name)
			VALUES ($2, $1, 'Ada', 'Example') RETURNING id, organization_id)
		INSERT INTO account (email, password_hash, role, organization_id, practitioner_id)
		SELECT 'ada@example.test', 'h', 'practitioner', organization_id, id FROM p RETURNING id`, clinicID, practitionerID).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	repo := auth.NewPostgresRepository(app)
	got, err := repo.ByID(t.Context(), id)
	if err != nil || got.Role != auth.RolePractitioner || got.PractitionerID != practitionerID {
		t.Errorf("ByID() = %+v, %v; want the practitioner %s", got, err, practitionerID)
	}
	if got, err := repo.ByEmail(t.Context(), "ada@example.test"); err != nil || got.PractitionerID != practitionerID {
		t.Errorf("ByEmail() = %+v, %v; want the practitioner %s", got, err, practitionerID)
	}
}
