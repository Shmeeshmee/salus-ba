package location_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/location"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// lcScoped runs fn in a transaction on app in the clinic org's scope, or the directory's for no clinic.
func lcScoped(t *testing.T, app *pgxpool.Pool, repo location.Repository, org string, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		set := repo.SetDirectory
		if org != "" {
			set = func(ctx context.Context) error { return repo.SetOrganization(ctx, org) }
		}
		if err := set(ctx); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// Clinic A (listed, in Belgrade's time zone) and B (not listed) each have the first location 00081 gives a new clinic.
func TestPostgresRepository_LocationsOfTheClinicAndTheDirectory(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	if _, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name, listed, time_zone) VALUES
		($1, 'Synthetic clinic A', true, 'Europe/Belgrade'), ($2, 'Synthetic clinic B', false, 'Europe/Sarajevo')`, lcClinicA, lcClinicB); err != nil {
		t.Fatal(err)
	}
	repo := location.NewPostgresRepository(app)
	var added, changed location.Location
	err := lcScoped(t, app, repo, lcClinicA, func(ctx context.Context) error {
		var err error
		if added, _, err = repo.SaveLocation(ctx, location.Location{OrganizationID: lcClinicA, Name: "Annex", Phone: "+38733000000", Active: true}); err != nil {
			return err
		}
		next := added
		next.Name, next.Active, next.VersionID = "Annex, closed", false, 2
		var current bool
		if changed, current, err = repo.SaveLocation(ctx, next); err != nil || !current {
			return errors.Join(errors.New("change not current"), err)
		}
		if _, current, err := repo.SaveLocation(ctx, next); err != nil || current {
			t.Errorf("a change at an old version: current %v, %v; want not current", current, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if added.TimeZone != "Europe/Belgrade" || added.VersionID != 1 || changed.Name != "Annex, closed" || changed.VersionID != 2 ||
		changed.TimeZone != "Europe/Belgrade" || changed.Phone != "+38733000000" {
		t.Errorf("added %+v, changed %+v; want the clinic's time zone kept and the version moved on", added, changed)
	}
	lists := map[string]struct {
		org, scope string
		want       []string
	}{
		"clinic A":                {lcClinicA, lcClinicA, []string{"Synthetic clinic A", "Annex, closed"}},
		"clinic A from B's scope": {lcClinicA, lcClinicB, nil},
		"directory, A":            {lcClinicA, "", []string{"Synthetic clinic A"}},
		"directory, B (unlisted)": {lcClinicB, "", nil},
	}
	for name, tc := range lists {
		var got []string
		if err := lcScoped(t, app, repo, tc.scope, func(ctx context.Context) error {
			ls, err := repo.Locations(ctx, tc.org)
			for _, l := range ls {
				got = append(got, l.Name)
			}
			return err
		}); err != nil || len(got) != len(tc.want) || (len(got) > 0 && (got[0] != tc.want[0] || got[len(got)-1] != tc.want[len(tc.want)-1])) {
			t.Errorf("%s: %v, %v; want %v", name, got, err, tc.want)
		}
	}
	refusals := map[string]struct {
		want error
		l    location.Location
	}{
		"B's location from A's scope": {location.ErrNotFound, location.Location{ID: added.ID, OrganizationID: lcClinicB, Name: "X", VersionID: 3}},
		"an unknown id":               {location.ErrNotFound, location.Location{ID: "not-a-uuid", OrganizationID: lcClinicA, Name: "X", VersionID: 2}},
		"an unknown city":             {location.ErrInvalid, location.Location{OrganizationID: lcClinicA, Name: "X", CityID: "Q0"}},
	}
	for name, tc := range refusals {
		err := lcScoped(t, app, repo, lcClinicA, func(ctx context.Context) error {
			_, _, err := repo.SaveLocation(ctx, tc.l)
			return err
		})
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", name, err, tc.want)
		}
	}
}
