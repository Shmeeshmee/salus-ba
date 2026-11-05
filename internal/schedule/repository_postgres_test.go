package schedule_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seed creates, as the owner, clinics A (listed) and B with roles A1, A2 and B, and a schedule for A2 and B.
func seed(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	ctx := t.Context()
	if _, err := owner.Exec(ctx, `INSERT INTO organization (id, name, listed) VALUES ($1, 'Synthetic clinic A', true), ($2, 'Synthetic clinic B', false)`,
		clinicA, clinicB); err != nil {
		t.Fatal(err)
	}
	for _, r := range []struct{ id, org string }{{roleA1, clinicA}, {roleA2, clinicA}, {roleB, clinicB}} {
		if _, err := owner.Exec(ctx, `WITH p AS (INSERT INTO practitioner (organization_id, given_name, family_name)
			VALUES ($2, 'Ada', 'Example') RETURNING id, organization_id)
			INSERT INTO practitioner_role (id, organization_id, practitioner_id) SELECT $1, organization_id, id FROM p`, r.id, r.org); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := owner.Exec(ctx, `INSERT INTO schedule (id, organization_id, practitioner_role_id, slot_minutes, availability, updated_at)
		VALUES ($1, $2, $3, 20, '[{"day": 2, "start": "08:00", "end": "10:00"}]', '2026-11-01T08:00:00Z'),
		       ($4, $5, $6, 15, '[]', '2026-11-01T08:00:00Z')`, scheduleA2, clinicA, roleA2, scheduleB, clinicB, roleB); err != nil {
		t.Fatal(err)
	}
}

// in runs fn in a transaction on app in the scope set by scope.
func in(t *testing.T, app *pgxpool.Pool, scope func(ctx context.Context) error, fn func(ctx context.Context) error) error {
	t.Helper()
	return postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		if err := scope(ctx); err != nil {
			return err
		}
		return fn(ctx)
	})
}

var scheduleA2Row = schedule.Schedule{
	ID: scheduleA2, OrganizationID: clinicA, PractitionerRoleID: roleA2, SlotMinutes: 20, VersionID: 1,
	Rules: []schedule.Rule{{Day: 2, Start: "08:00", End: "10:00"}}, LastUpdated: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC),
}

func equal(a, b schedule.Schedule) bool {
	return a.ID == b.ID && a.OrganizationID == b.OrganizationID && a.PractitionerRoleID == b.PractitionerRoleID &&
		a.SlotMinutes == b.SlotMinutes && a.VersionID == b.VersionID && slices.Equal(a.Rules, b.Rules) && a.LastUpdated.Equal(b.LastUpdated)
}

func TestPostgres_ReadsSeeOnlyTheClinicSet(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := schedule.NewPostgresRepository(app)
	clinic := func(ctx context.Context) error { return repo.SetOrganization(ctx, clinicA) }

	reads := map[string]func(ctx context.Context) (schedule.Schedule, error){
		"ByID":   func(ctx context.Context) (schedule.Schedule, error) { return repo.ByID(ctx, scheduleA2) },
		"ByRole": func(ctx context.Context) (schedule.Schedule, error) { return repo.ByRole(ctx, roleA2) },
		"Lock":   func(ctx context.Context) (schedule.Schedule, error) { return repo.Lock(ctx, scheduleA2) },
	}
	for name, read := range reads {
		var got schedule.Schedule
		err := in(t, app, clinic, func(ctx context.Context) error {
			var err error
			got, err = read(ctx)
			return err
		})
		if err != nil || !equal(got, scheduleA2Row) {
			t.Errorf("%s in clinic A = %+v, %v;\nwant %+v", name, got, err, scheduleA2Row)
		}
	}
	for name, read := range map[string]func(ctx context.Context) (schedule.Schedule, error){
		"ByID of B":       func(ctx context.Context) (schedule.Schedule, error) { return repo.ByID(ctx, scheduleB) },
		"ByRole of B":     func(ctx context.Context) (schedule.Schedule, error) { return repo.ByRole(ctx, roleB) },
		"Lock of B":       func(ctx context.Context) (schedule.Schedule, error) { return repo.Lock(ctx, scheduleB) },
		"ByRole of A1":    func(ctx context.Context) (schedule.Schedule, error) { return repo.ByRole(ctx, roleA1) },
		"ByID not a UUID": func(ctx context.Context) (schedule.Schedule, error) { return repo.ByID(ctx, "x") },
		"ByRole not UUID": func(ctx context.Context) (schedule.Schedule, error) { return repo.ByRole(ctx, "x") },
		"Lock not a UUID": func(ctx context.Context) (schedule.Schedule, error) { return repo.Lock(ctx, "x") },
	} {
		err := in(t, app, clinic, func(ctx context.Context) error {
			_, err := read(ctx)
			return err
		})
		if !errors.Is(err, schedule.ErrNotFound) {
			t.Errorf("%s in clinic A = %v, want ErrNotFound", name, err)
		}
	}
}

func TestPostgres_TheDirectoryListsTheSchedulesOfListedClinicsOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := schedule.NewPostgresRepository(app)

	var got []schedule.Schedule
	err := in(t, app, repo.SetDirectory, func(ctx context.Context) error {
		var err error
		got, err = repo.Listed(ctx)
		return err
	})
	if err != nil || len(got) != 1 || !equal(got[0], scheduleA2Row) {
		t.Errorf("Listed() in the directory = %+v, %v; want schedule A2 only", got, err)
	}
}

func TestPostgres_CreatesOncePerRoleAndUpdatesTheCurrentVersionOnly(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := schedule.NewPostgresRepository(app)
	clinic := func(ctx context.Context) error { return repo.SetOrganization(ctx, clinicA) }
	rules := []schedule.Rule{{Day: 1, Start: "09:00", End: "12:00"}, {Day: 5, Start: "14:00", End: "15:30"}}
	at := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)

	var created, updated schedule.Schedule
	err := in(t, app, clinic, func(ctx context.Context) error {
		var err error
		if created, err = repo.Create(ctx, schedule.Schedule{OrganizationID: clinicA, PractitionerRoleID: roleA1, SlotMinutes: 30, Rules: rules}); err != nil {
			return err
		}
		updated, err = repo.Update(ctx, schedule.Schedule{ID: created.ID, SlotMinutes: 45, Rules: rules[:1], VersionID: 1}, at)
		return err
	})
	if err != nil || created.ID == "" || created.VersionID != 1 || created.SlotMinutes != 30 || !slices.Equal(created.Rules, rules) {
		t.Fatalf("Create() = %+v, %v; want role A1's schedule at version 1 with its rules", created, err)
	}
	if updated.ID != created.ID || updated.VersionID != 2 || updated.SlotMinutes != 45 || !slices.Equal(updated.Rules, rules[:1]) || !updated.LastUpdated.Equal(at) {
		t.Errorf("Update() = %+v; want version 2 with 45 minutes and one rule, updated at %v", updated, at)
	}

	for name, write := range map[string]func(ctx context.Context) error{
		"a second create": func(ctx context.Context) error {
			_, err := repo.Create(ctx, schedule.Schedule{OrganizationID: clinicA, PractitionerRoleID: roleA1, SlotMinutes: 30})
			return err
		},
		"a stale update": func(ctx context.Context) error {
			_, err := repo.Update(ctx, schedule.Schedule{ID: created.ID, SlotMinutes: 30, VersionID: 1}, at)
			return err
		},
		"an update of another clinic's": func(ctx context.Context) error {
			_, err := repo.Update(ctx, schedule.Schedule{ID: scheduleB, SlotMinutes: 30, VersionID: 1}, at)
			return err
		},
		"an update of not a UUID": func(ctx context.Context) error {
			_, err := repo.Update(ctx, schedule.Schedule{ID: "x", SlotMinutes: 30, VersionID: 1}, at)
			return err
		},
	} {
		if err := in(t, app, clinic, write); !errors.Is(err, schedule.ErrConflict) {
			t.Errorf("%s = %v, want ErrConflict", name, err)
		}
	}
}
