package organization_test

import (
	"context"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/organization"
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
	for _, s := range []organization.AppointmentSettings{customSet, organization.DefaultAppointmentSettings()} {
		if err := in(clinicA, func(ctx context.Context) error { return repo.PutAppointmentSettings(ctx, clinicA, s, at) }); err != nil {
			t.Fatalf("PutAppointmentSettings() = %v", err)
		}
		if got, ok := read(clinicA, clinicA); !ok || !same(got, s) {
			t.Errorf("read back %+v, %t; want %+v", got, ok, s)
		}
	}
	if _, ok := read(clinicB, clinicA); ok {
		t.Error("clinic B reads clinic A's settings")
	}
	if _, ok := read(clinicA, "not a uuid"); ok {
		t.Error("an id that is not a UUID has settings")
	}
	if err := in(clinicB, func(ctx context.Context) error { return repo.PutAppointmentSettings(ctx, clinicA, customSet, at) }); err == nil {
		t.Error("clinic B stored clinic A's settings")
	}
	if err := in(clinicA, func(ctx context.Context) error { return repo.PutAppointmentSettings(ctx, "x", customSet, at) }); err == nil {
		t.Error("settings of an id that is not a UUID were stored")
	}
}
