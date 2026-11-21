package chargeitemdefinition_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// seedClinics stores clinic A (listed) and clinic B (not listed).
func seedClinics(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	_, err := owner.Exec(t.Context(), `INSERT INTO organization (id, name, listed, updated_at, contributes_since) VALUES
		($1, 'Synthetic clinic A', true, '2026-11-01T08:00:00Z', '2026-11-01T08:00:00Z'),
		($2, 'Synthetic clinic B', false, '2026-11-01T08:00:00Z', '2026-11-01T08:00:00Z')`, clinicA, clinicB)
	if err != nil {
		t.Fatal(err)
	}
}

// scoped runs fn in a transaction on app in the clinic org's scope, or the directory's for no clinic.
func scoped(t *testing.T, app *pgxpool.Pool, repo chargeitemdefinition.Repository, org string, fn func(ctx context.Context) error) error {
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

// store is the repository on a throwaway database with clinic A (listed) and clinic B (not listed).
type store struct {
	t    *testing.T
	app  *pgxpool.Pool
	repo chargeitemdefinition.Repository
}

func newStore(t *testing.T) store {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	return store{t: t, app: app, repo: chargeitemdefinition.NewPostgresRepository(app)}
}

func (s store) create(org string, d chargeitemdefinition.Definition) chargeitemdefinition.Definition {
	s.t.Helper()
	d.OrganizationID, d.Currency = org, "BAM"
	var out chargeitemdefinition.Definition
	if err := scoped(s.t, s.app, s.repo, org, func(ctx context.Context) (err error) {
		out, err = s.repo.Create(ctx, d, now)
		return err
	}); err != nil {
		s.t.Fatal(err)
	}
	return out
}

// list is the services of org as the scope shows them (the directory's for no scope).
func (s store) list(scope, org string) []chargeitemdefinition.Definition {
	s.t.Helper()
	var out []chargeitemdefinition.Definition
	if err := scoped(s.t, s.app, s.repo, scope, func(ctx context.Context) (err error) {
		out, err = s.repo.List(ctx, org)
		return err
	}); err != nil {
		s.t.Fatal(err)
	}
	return out
}

func (s store) update(scope string, d chargeitemdefinition.Definition) (chargeitemdefinition.Definition, error) {
	var out chargeitemdefinition.Definition
	err := scoped(s.t, s.app, s.repo, scope, func(ctx context.Context) (err error) {
		out, err = s.repo.Update(ctx, d, now.Add(time.Minute))
		return err
	})
	return out, err
}

// fullExam is exam with a VAT rate and a second translation.
func fullExam() chargeitemdefinition.Definition {
	full := exam()
	full.VATRateBP = ptr(1700)
	full.Translations = append(full.Translations, chargeitemdefinition.Translation{Language: "ar", Name: "Placeholder"})
	return full
}

func TestRepository_StoresAndScopesTheClinicsServices(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	full := fullExam()
	made := s.create(clinicA, full)
	bare := s.create(clinicA, chargeitemdefinition.Definition{Name: "A placeholder", Modes: []schedule.VisitMode{schedule.ModeVideo}})
	s.create(clinicB, exam())

	got := s.list(clinicA, clinicA)
	if len(got) != 2 || got[0].ID != bare.ID || got[1].ID != made.ID {
		t.Fatalf("clinic A lists %+v, want its two by name", got)
	}
	if d := got[1]; d.Name != full.Name || d.Code != "EX-1" || d.PriceMinor != 5000 || d.Currency != "BAM" || d.DurationMinutes != 30 ||
		d.VATRateBP == nil || *d.VATRateBP != 1700 || len(d.Modes) != 2 || !d.Active || d.VersionID != 1 || !d.LastUpdated.Equal(now) ||
		d.OrganizationID != clinicA || len(d.Translations) != 2 || d.Translations[0].Name != full.Translations[0].Name || d.Translations[1].Language != "ar" {
		t.Errorf("read back %+v, want what was stored", d)
	}
	if d := got[0]; d.Code != "" || d.DurationMinutes != 0 || d.VATRateBP != nil || d.Active || len(d.Translations) != 0 {
		t.Errorf("read back %+v, want no code, duration, VAT rate or translation, inactive", d)
	}
	if other := s.list(clinicA, clinicB); len(other) != 0 {
		t.Errorf("clinic A sees clinic B's services: %+v", other)
	}
	if none := s.list(clinicA, "not-a-uuid"); len(none) != 0 {
		t.Errorf("a malformed clinic id lists %+v", none)
	}

	// The directory: the active services of listed clinics only.
	if dir := s.list("", clinicA); len(dir) != 1 || dir[0].ID != made.ID || len(dir[0].Translations) != 2 {
		t.Errorf("the directory shows %+v of clinic A, want its active service with its translations", dir)
	}
	if dir := s.list("", clinicB); len(dir) != 0 {
		t.Errorf("the directory shows an unlisted clinic's services: %+v", dir)
	}
	if err := scoped(t, s.app, s.repo, clinicA, func(ctx context.Context) error {
		_, err := s.repo.Create(ctx, chargeitemdefinition.Definition{OrganizationID: clinicB, Name: "x", Currency: "BAM", Modes: bare.Modes}, now)
		return err
	}); err == nil {
		t.Error("clinic A stored a service in clinic B")
	}
}

func TestRepository_ReplacesAServiceAtItsVersionInItsClinicOnly(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	made := s.create(clinicA, fullExam())
	next := made
	next.Name, next.PriceMinor, next.Code, next.VATRateBP, next.DurationMinutes, next.Active = "Renamed placeholder", 6000, "", nil, 0, false
	next.Modes = []schedule.VisitMode{schedule.ModeInPerson}
	next.Translations = []chargeitemdefinition.Translation{{Language: "en", Name: "Placeholder"}}
	if _, err := s.update(clinicB, next); !errors.Is(err, chargeitemdefinition.ErrNotFound) {
		t.Errorf("Update() from clinic B = %v, want ErrNotFound", err)
	}
	if _, err := s.update("", next); err == nil {
		t.Error("Update() from the directory's scope succeeded")
	}
	saved, err := s.update(clinicA, next)
	if err != nil || saved.VersionID != 2 {
		t.Fatalf("Update() = %+v, %v; want version 2", saved, err)
	}
	if _, err := s.update(clinicA, next); !errors.Is(err, chargeitemdefinition.ErrConflict) {
		t.Errorf("Update() at the old version = %v, want ErrConflict", err)
	}
	next.ID = "not-a-uuid"
	if _, err := s.update(clinicA, next); !errors.Is(err, chargeitemdefinition.ErrNotFound) {
		t.Errorf("Update() of a malformed id = %v, want ErrNotFound", err)
	}
	got := s.list(clinicA, clinicA)
	if d := got[0]; len(got) != 1 || d.ID != made.ID || d.Name != "Renamed placeholder" || d.PriceMinor != 6000 || d.Code != "" || d.VATRateBP != nil ||
		d.DurationMinutes != 0 || d.Active || d.VersionID != 2 || !d.LastUpdated.Equal(now.Add(time.Minute)) || len(d.Modes) != 1 ||
		len(d.Translations) != 1 || d.Translations[0].Language != "en" {
		t.Errorf("after the update read back %+v", d)
	}
}
