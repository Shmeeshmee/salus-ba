package organization_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A clinic stores and replaces its own appointment settings, and sees only its own.
func TestAppointmentSettings_StoresAndReplacesTheClinicsOwn(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	repo := organization.NewPostgresAppointmentsRepository(app)
	tx := postgres.NewTransactor(app)
	in := func(org string, fn func(ctx context.Context) error) error {
		return tx.WithTx(t.Context(), func(ctx context.Context) error {
			if err := repo.SetOrganization(ctx, org); err != nil {
				return err
			}
			return fn(ctx)
		})
	}
	read := func(org, of string) (organization.AppointmentSettings, bool) {
		var s organization.AppointmentSettings
		var ok bool
		if err := in(org, func(ctx context.Context) error {
			var err error
			s, ok, err = repo.AppointmentSettings(ctx, of)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return s, ok
	}
	if _, ok := read(clinicA, clinicA); ok {
		t.Fatal("clinic A has settings before it stored any")
	}
	at := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	for i, s := range []organization.AppointmentSettings{customSet, organization.DefaultAppointmentSettings()} {
		s.Version = i
		var version int
		if err := in(clinicA, func(ctx context.Context) error {
			var err error
			version, err = repo.PutAppointmentSettings(ctx, clinicA, s, at)
			return err
		}); err != nil || version != i+1 {
			t.Fatalf("PutAppointmentSettings(version %d) = %d, %v; want %d", i, version, err, i+1)
		}
		if got, ok := read(clinicA, clinicA); !ok || !same(got, s) || got.Version != i+1 {
			t.Errorf("read back %+v, %t; want %+v at version %d", got, ok, s, i+1)
		}
	}
	for _, stale := range []int{0, 1, 3} {
		s := customSet
		s.Version = stale
		if err := in(clinicA, func(ctx context.Context) error {
			_, err := repo.PutAppointmentSettings(ctx, clinicA, s, at)
			return err
		}); !errors.Is(err, organization.ErrSettingsChanged) {
			t.Errorf("PutAppointmentSettings(version %d) at version 2 = %v, want ErrSettingsChanged", stale, err)
		}
	}
	if _, ok := read(clinicB, clinicA); ok {
		t.Error("clinic B reads clinic A's settings")
	}
	if _, ok := read(clinicA, "not a uuid"); ok {
		t.Error("an id that is not a UUID has settings")
	}
	if err := in(clinicB, func(ctx context.Context) error {
		_, err := repo.PutAppointmentSettings(ctx, clinicA, customSet, at)
		return err
	}); err == nil {
		t.Error("clinic B stored clinic A's settings")
	}
	if err := in(clinicA, func(ctx context.Context) error {
		_, err := repo.PutAppointmentSettings(ctx, "x", customSet, at)
		return err
	}); err == nil {
		t.Error("settings of an id that is not a UUID were stored")
	}
}

// Two saves read at the same version, at the same time, through the service on the real database: exactly one is
// stored and the other is refused with ErrSettingsChanged (the handler's 412), whether a row existed before or not.
func TestAppointmentSettings_ConcurrentSavesLeaveOneWinner(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	tx := postgres.NewTransactor(app)
	svc := organization.NewAppointments(organization.AppointmentsPorts{
		Repo: organization.NewPostgresAppointmentsRepository(app), Tx: tx, Clock: clock.Fixed{T: now}, Audit: &lockedAuditor{},
	})
	for _, version := range []int{0, 1} {
		var wg sync.WaitGroup
		errs := make([]error, 8)
		start := make(chan struct{})
		for i := range errs {
			wg.Go(func() {
				in := organization.DefaultAppointmentSettings()
				in.Version, in.RescheduleMax = version, i%(5+1)
				<-start
				_, errs[i] = svc.SetSettings(t.Context(), adminOfA, in)
			})
		}
		close(start)
		wg.Wait()
		won, lost := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				won++
			case errors.Is(err, organization.ErrSettingsChanged):
				lost++
			default:
				t.Fatalf("SetSettings() = %v", err)
			}
		}
		if won != 1 || lost != len(errs)-1 {
			t.Errorf("read at version %d: %d stored, %d refused; want exactly one stored", version, won, lost)
		}
		if got, err := svc.Settings(t.Context(), adminOfA); err != nil || got.Version != version+1 {
			t.Errorf("Settings() = version %d, %v; want %d", got.Version, err, version+1)
		}
	}
}

// lockedAuditor is fakeAuditor for concurrent saves.
type lockedAuditor struct {
	fakeAuditor
	mu sync.Mutex
}

func (a *lockedAuditor) Record(ctx context.Context, e auditevent.AuditEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.fakeAuditor.Record(ctx, e)
}
