package organization_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A clinic stores and replaces its own chat settings, and sees only its own.
func TestChatSettings_StoresAndReplacesTheClinicsOwn(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedClinics(t, owner)
	repo := organization.NewPostgresChatRepository(app)
	tx := postgres.NewTransactor(app)
	in := func(org string, fn func(ctx context.Context) error) error {
		return tx.WithTx(t.Context(), func(ctx context.Context) error {
			if err := repo.SetOrganization(ctx, org); err != nil {
				return err
			}
			return fn(ctx)
		})
	}
	read := func(org, of string) (organization.ChatSettings, bool) {
		var s organization.ChatSettings
		var ok bool
		if err := in(org, func(ctx context.Context) error {
			var err error
			s, ok, err = repo.ChatSettings(ctx, of)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return s, ok
	}
	if _, ok := read(clinicA, clinicA); ok {
		t.Fatal("clinic A has settings before it stored any")
	}
	at := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	for _, s := range []organization.ChatSettings{
		{Hours: []organization.ChatHour{{Day: "mon", Start: "08:00", End: "16:00"}}, ResponseTimeHours: 8, EmailAfterMinutes: 30},
		{Hours: []organization.ChatHour{}, ResponseTimeHours: 48, EmailAfterMinutes: 120},
	} {
		if err := in(clinicA, func(ctx context.Context) error { return repo.PutChatSettings(ctx, clinicA, s, at) }); err != nil {
			t.Fatalf("PutChatSettings() = %v", err)
		}
		got, ok := read(clinicA, clinicA)
		if !ok || !slices.Equal(got.Hours, s.Hours) || got.ResponseTimeHours != s.ResponseTimeHours || got.EmailAfterMinutes != s.EmailAfterMinutes {
			t.Errorf("ChatSettings() = %+v, %v; want %+v", got, ok, s)
		}
	}
	if _, ok := read(clinicB, clinicA); ok {
		t.Error("clinic B sees clinic A's settings")
	}
	if err := in(clinicB, func(ctx context.Context) error {
		return repo.PutChatSettings(ctx, clinicA, organization.ChatSettings{ResponseTimeHours: 1, EmailAfterMinutes: 5}, at)
	}); err == nil {
		t.Error("clinic B wrote clinic A's settings")
	}
}
