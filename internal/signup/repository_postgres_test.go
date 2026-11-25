package signup_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/signup"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

const (
	clinicA = "c0000000-0000-4000-8000-00000000000a"
	clinicB = "c0000000-0000-4000-8000-00000000000b"
)

func hashOf(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

// A signup's life as salus_app (insert, a new token, verification) and the owner (list, approval).
func TestPostgresRepository_SignupFromInsertToApproval(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	if _, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic A')`, clinicA); err != nil {
		t.Fatal(err)
	}
	repo, ownerRepo := signup.NewPostgresRepository(app), signup.NewPostgresRepository(owner)
	ctx := t.Context()

	id, err := repo.Insert(ctx, signup.Request{Email: "admin@example.test", ClinicName: "Synthetic clinic A", Language: "en"}, hashOf("one"))
	if err != nil {
		t.Fatalf("Insert() = %v", err)
	}
	issued := now.Add(time.Hour)
	if ok, err := repo.SetToken(ctx, id, hashOf("two"), issued); err != nil || !ok {
		t.Fatalf("SetToken() = %v, %v", ok, err)
	}
	if _, err := repo.ByTokenHash(ctx, hashOf("one")); !errors.Is(err, signup.ErrNotFound) {
		t.Errorf("ByTokenHash(the replaced token) = %v, want ErrNotFound", err)
	}
	var got signup.Signup
	err = postgres.NewTransactor(app).WithTx(ctx, func(ctx context.Context) error {
		if got, err = repo.ByTokenHash(ctx, hashOf("two")); err != nil {
			return err
		}
		ok, err := repo.MarkVerified(ctx, id, clinicA, now)
		if err == nil && !ok {
			t.Error("MarkVerified() = false, want true")
		}
		return err
	})
	if err != nil || got.ID != id || got.Email != "admin@example.test" || got.Language != "en" || got.CreatedAt.IsZero() || !got.TokenIssuedAt.Equal(issued) {
		t.Fatalf("ByTokenHash() = %+v, %v", got, err)
	}
	if ok, err := repo.MarkVerified(ctx, id, clinicA, now); err != nil || ok {
		t.Errorf("MarkVerified() twice = %v, %v; want false", ok, err)
	}
	if ok, err := repo.SetToken(ctx, id, hashOf("three"), issued); err != nil || ok {
		t.Errorf("SetToken() of a verified signup = %v, %v; want false", ok, err)
	}
	if got, err := repo.ByID(ctx, id); err != nil || got.OrganizationID != clinicA || !got.VerifiedAt.Equal(now) {
		t.Errorf("ByID() = %+v, %v; want it verified with clinic A", got, err)
	}
	if _, err := repo.ByID(ctx, "not-a-uuid"); !errors.Is(err, signup.ErrNotFound) {
		t.Errorf("ByID(not a uuid) = %v, want ErrNotFound", err)
	}
	if ok, err := ownerRepo.Approve(ctx, clinicA, now); err != nil || !ok {
		t.Fatalf("Approve() = %v, %v", ok, err)
	}
	if ok, err := ownerRepo.Approve(ctx, clinicA, now); err != nil || ok {
		t.Errorf("Approve() twice = %v, %v; want false", ok, err)
	}
	list, err := ownerRepo.Verified(ctx)
	if err != nil || len(list) != 1 || list[0].ID != id || list[0].ApprovedAt.IsZero() {
		t.Errorf("Verified() = %+v, %v", list, err)
	}
	// U8: once the admin joined, the verified signup forgets the e-mail; the rest of the row stays.
	if err := repo.ForgetEmail(ctx, clinicA); err != nil {
		t.Fatalf("ForgetEmail() = %v", err)
	}
	if got, err := repo.ByID(ctx, id); err != nil || got.Email != "" || got.ClinicName != "Synthetic clinic A" {
		t.Errorf("ByID() after ForgetEmail = %+v, %v; want no e-mail", got, err)
	}
}

// U8: an unverified signup keeps its e-mail (its link is still to be sent); only a verified one may lose it.
func TestPostgresRepository_AnUnverifiedSignupKeepsItsEmail(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	_, err := owner.Exec(t.Context(), `INSERT INTO clinic_signup (email, clinic_name, language, token_hash)
		VALUES ('admin@example.test', 'Synthetic clinic', 'en', sha256('a'));
		UPDATE clinic_signup SET email = NULL`)
	if err == nil || !strings.Contains(err.Error(), "clinic_signup_email_kept") {
		t.Errorf("blanking an unverified signup's e-mail: %v, want clinic_signup_email_kept", err)
	}
}

// The wizard's progress is the clinic's own: row-level security shows one clinic's steps only.
func TestPostgresRepository_SetupStepsInTheClinicsScope(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	if _, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic A'), ($2, 'Synthetic clinic B')`, clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	repo := signup.NewPostgresRepository(app)
	tx := postgres.NewTransactor(app)
	in := func(org string, fn func(ctx context.Context) error) error {
		return tx.WithTx(t.Context(), func(ctx context.Context) error {
			if err := repo.SetOrganization(ctx, org); err != nil {
				return err
			}
			return fn(ctx)
		})
	}
	err := in(clinicA, func(ctx context.Context) error {
		for step, state := range map[signup.Step]signup.StepState{signup.StepDetails: signup.StepDone, signup.StepBilling: signup.StepSkipped, signup.StepHours: signup.StepDone} {
			if err := repo.PutStep(ctx, clinicA, step, state, now); err != nil {
				return err
			}
		}
		return repo.PutStep(ctx, clinicA, signup.StepHours, signup.StepOpen, now)
	})
	if err != nil {
		t.Fatalf("PutStep() = %v", err)
	}
	var got map[signup.Step]signup.StepState
	if err := in(clinicA, func(ctx context.Context) (err error) { got, err = repo.Steps(ctx, clinicA); return err }); err != nil ||
		!maps.Equal(got, map[signup.Step]signup.StepState{signup.StepDetails: signup.StepDone, signup.StepBilling: signup.StepSkipped}) {
		t.Errorf("Steps(clinic A) = %v, %v", got, err)
	}
	if err := in(clinicB, func(ctx context.Context) (err error) { got, err = repo.Steps(ctx, clinicA); return err }); err != nil || len(got) != 0 {
		t.Errorf("Steps(clinic A) in clinic B's scope = %v, %v; want none", got, err)
	}
	if err := in(clinicB, func(ctx context.Context) error {
		return repo.PutStep(ctx, clinicA, signup.StepDetails, signup.StepDone, now)
	}); err == nil {
		t.Error("PutStep(clinic A) in clinic B's scope succeeded, want refused")
	}
}

// The retention job deletes the unverified signups made before its cut-off, as salus_retention, and nothing else.
func TestPurger_DeletesOnlyUnverifiedSignupsBeforeTheCutOff(t *testing.T) {
	t.Parallel()
	owner, retention := pgtest.NewRetentionDatabase(t)
	_, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name) VALUES ('`+clinicA+`', 'Synthetic clinic A');
		INSERT INTO clinic_signup (email, clinic_name, language, token_hash, created_at) VALUES
			('old@example.test', 'Old', 'en', sha256('old'), '2026-10-01T00:00:00Z'),
			('new@example.test', 'New', 'en', sha256('new'), '2026-11-20T00:00:00Z');
		INSERT INTO clinic_signup (email, clinic_name, language, token_hash, created_at, verified_at, organization_id) VALUES
			('kept@example.test', 'Kept', 'en', sha256('kept'), '2026-10-01T00:00:00Z', '2026-10-01T01:00:00Z', '`+clinicA+`')`)
	if err != nil {
		t.Fatal(err)
	}
	n, err := signup.NewPurger(signup.NewPostgresRepository(retention)).Purge(t.Context(), now.Add(-30*24*time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("Purge() = %d, %v; want the one old unverified signup", n, err)
	}
	var left []string
	rows, err := owner.Query(t.Context(), "SELECT clinic_name FROM clinic_signup ORDER BY clinic_name")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		left = append(left, s)
	}
	if strings.Join(left, ",") != "Kept,New" {
		t.Errorf("left %v, want Kept and New", left)
	}
}

// A self-signed-up clinic is listed only once the operator approves it: its own admin cannot list it through the
// clinic settings (salus_app may update organization.listed) until approved_at is set.
func TestListing_ASignedUpClinicWaitsForTheOperator(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name) VALUES ('`+clinicA+`', 'Synthetic clinic A'), ('`+clinicB+`', 'Synthetic clinic B');
		INSERT INTO clinic_signup (email, clinic_name, language, token_hash, verified_at, organization_id)
		VALUES ('admin@example.test', 'Synthetic clinic A', 'en', sha256('a'), now(), '`+clinicA+`')`)
	if err != nil {
		t.Fatal(err)
	}
	list := func(org string) error {
		return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			if err := postgres.SetLocal(ctx, "app.organization_id", org); err != nil {
				return err
			}
			_, err := postgres.Conn(ctx, app).Exec(ctx, "UPDATE organization SET listed = true WHERE id = $1", org)
			return err
		})
	}
	if err := list(clinicA); err == nil {
		t.Error("listing an unapproved signed-up clinic as salus_app succeeded, want refused")
	}
	if err := list(clinicB); err != nil {
		t.Errorf("listing a clinic made by salusctl create-clinic: %v", err)
	}
	if _, err := owner.Exec(t.Context(), "UPDATE clinic_signup SET approved_at = now() WHERE organization_id = $1", clinicA); err != nil {
		t.Fatal(err)
	}
	if err := list(clinicA); err != nil {
		t.Errorf("listing an approved clinic: %v", err)
	}
}
