package organization_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// inDirectory runs fn in a transaction on app in the public directory's scope.
func inDirectory(t *testing.T, app *pgxpool.Pool, repo organization.Repository, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := repo.SetDirectory(ctx); err != nil {
			return err
		}
		return fn(ctx)
	})
}

func TestPutProfile_StoresTheProfileAtTheNextVersionOrConflicts(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	repo := organization.NewPostgresRepository(app)
	at := time.Date(2026, 11, 23, 9, 30, 0, 0, time.UTC)

	in := organization.Profile{Languages: []string{"de", "bs"}, Description: "Synthetic", WelcomesInternational: true, VersionID: 1,
		Translations: []organization.Translation{{Language: "en", Description: "Synthetic, en"}, {Language: "de", Description: "Synthetisch"}}} // in the enum's order
	var saved, read organization.Profile
	err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		if saved, err = repo.PutProfile(ctx, clinicA, in, at); err != nil {
			return err
		}
		read, err = repo.ProfileOf(ctx, clinicA)
		return err
	})
	want := in
	want.VersionID = 2
	if err != nil || !reflect.DeepEqual(saved, want) || !reflect.DeepEqual(read, want) {
		t.Errorf("PutProfile() = %+v, then ProfileOf() = %+v, %v; want %+v", saved, read, err, want)
	}

	// A second save replaces the translations; a stale version conflicts.
	next := organization.Profile{Languages: []string{}, Translations: []organization.Translation{}, VersionID: 2}
	err = inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		if _, err = repo.PutProfile(ctx, clinicA, next, at); err != nil {
			return err
		}
		read, err = repo.ProfileOf(ctx, clinicA)
		return err
	})
	next.VersionID = 3
	if err != nil || !reflect.DeepEqual(read, next) {
		t.Errorf("ProfileOf() after the second save = %+v, %v; want %+v", read, err, next)
	}
	for _, id := range []string{clinicA, clinicB, "not-a-uuid"} {
		err := inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
			_, err := repo.PutProfile(ctx, id, organization.Profile{VersionID: 1}, at)
			return err
		})
		if !errors.Is(err, organization.ErrConflict) {
			t.Errorf("PutProfile(%q) at version 1 in clinic A: %v, want ErrConflict", id, err)
		}
	}
	err = inClinic(t, app, repo, clinicA, func(ctx context.Context) error {
		_, err := repo.ProfileOf(ctx, clinicB)
		return err
	})
	if !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("ProfileOf(clinic B) in clinic A: %v, want ErrNotFound", err)
	}
}

// The directory shows a listed clinic's profile and finds it by a language spoken; a clinic that is not listed, or
// whose membership is suspended (package I's policy), is not found there, by its profile either.
func TestProfile_TheDirectoryHidesAnUnlistedOrSuspendedClinic(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	repo := organization.NewPostgresRepository(app)
	svc := organization.NewService(repo, &fakeAuditor{}, postgres.NewTransactor(app), clock.Fixed{T: now}, nil)
	_, err := owner.Exec(t.Context(), `UPDATE organization SET listed = true, languages = '{bs,de}', profile_description = 'Synthetic'
		WHERE id IN ($1, $2)`, clinicA, clinicB)
	if err == nil {
		_, err = owner.Exec(t.Context(), `UPDATE organization SET listed = false WHERE id = $1`, clinicB)
	}
	if err != nil {
		t.Fatal(err)
	}
	if p, err := svc.Profile(t.Context(), clinicA, nil); err != nil || p.Description != "Synthetic" {
		t.Errorf("Profile(listed clinic A) = %+v, %v; want its profile", p, err)
	}
	if got, err := svc.ListListedSpeaking(t.Context(), "de"); err != nil || len(got) != 1 || got[0].ID != clinicA {
		t.Errorf("ListListedSpeaking(de) = %+v, %v; want clinic A", got, err)
	}
	if got, err := svc.ListListedSpeaking(t.Context(), "fr"); err != nil || len(got) != 0 {
		t.Errorf("ListListedSpeaking(fr) = %+v, %v; want none", got, err)
	}
	if _, err := svc.Profile(t.Context(), clinicB, nil); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Profile(unlisted clinic B) = %v, want ErrNotFound", err)
	}

	_, err = owner.Exec(t.Context(), `INSERT INTO subscription_plan (code, name, price_per_seat_minor) VALUES ('basic', 'Basic', 10000)`)
	if err == nil {
		_, err = owner.Exec(t.Context(), `INSERT INTO clinic_subscription (organization_id, plan_code, status, trial_ends_at)
		VALUES ($1, 'basic', 'suspended', now())`, clinicA)
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Profile(t.Context(), clinicA, nil); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Profile(suspended clinic A) = %v, want ErrNotFound, as the directory", err)
	}
	err = inDirectory(t, app, repo, func(ctx context.Context) error {
		_, err := repo.ProfileOf(ctx, clinicA)
		return err
	})
	if !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("ProfileOf(suspended clinic A) in the directory: %v, want ErrNotFound", err)
	}
	if got, err := svc.ListListedSpeaking(t.Context(), "de"); err != nil || len(got) != 0 {
		t.Errorf("ListListedSpeaking(de) with clinic A suspended = %+v, %v; want none", got, err)
	}
	// Its own staff and patients keep it, read in its own scope (readiness: suspended clinics).
	served, err := svc.WithPatients(fakePatients{"c1000000-0000-4000-8000-0000000000c1": {clinicA, clinicB}}).Served(t.Context(),
		organization.Caller{AccountID: "c1000000-0000-4000-8000-0000000000c1", PatientID: "c1000000-0000-4000-8000-0000000000c1"})
	if !slices.Equal(served, []string{clinicA}) || err != nil {
		t.Errorf("Served(a patient of A and of unlisted B) = %v, %v; want the suspended, listed A", served, err)
	}
	if p, err := svc.Profile(t.Context(), clinicA, servedAs(clinicA)); err != nil || p.Description != "Synthetic" {
		t.Errorf("Profile(suspended clinic A) to whom it serves = %+v, %v; want its profile", p, err)
	}
	if _, err := svc.Profile(t.Context(), clinicB, servedAs(clinicB)); !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Profile(unlisted clinic B) to whom it serves = %v, want ErrNotFound", err)
	}
}
