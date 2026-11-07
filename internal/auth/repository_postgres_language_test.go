package auth_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// docs/i18n/SPEC.md §3 "The account's language fields": the user's choice and the client's match.

func TestCreate_StoresTheChosenAndTheDetectedLanguage(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	ctx := t.Context()
	a := patient("lena@example.test")
	a.Language, a.DetectedLanguage = "de", "ar"
	id, err := repo.Create(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.ByID(ctx, id)
	if err != nil || got.Language != "de" || got.DetectedLanguage != "ar" {
		t.Errorf("ByID() = %+v, %v; want language de, detected ar", got, err)
	}
}

func TestSetLanguage_SetsAndClearsTheChoiceAndSetDetectedLanguageTheMatch(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auth.NewPostgresRepository(app)
	ctx := t.Context()
	id, err := repo.Create(ctx, patient("ivo@example.test"))
	if err != nil {
		t.Fatal(err)
	}
	steps := []struct {
		set                    func() error
		name                   string
		wantLang, wantDetected string
	}{
		{name: "choose sr-Cyrl", set: func() error { return repo.SetLanguage(ctx, id, "sr-Cyrl") }, wantLang: "sr-Cyrl"},
		{name: "detect tr", set: func() error { return repo.SetDetectedLanguage(ctx, id, "tr") }, wantLang: "sr-Cyrl", wantDetected: "tr"},
		{name: "clear the choice", set: func() error { return repo.SetLanguage(ctx, id, "") }, wantDetected: "tr"},
	}
	for _, s := range steps {
		if err := s.set(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		got, err := repo.ByEmail(ctx, "ivo@example.test")
		if err != nil || got.Language != s.wantLang || got.DetectedLanguage != s.wantDetected {
			t.Errorf("after %s: language %q, detected %q, %v; want %q and %q", s.name, got.Language, got.DetectedLanguage, err, s.wantLang, s.wantDetected)
		}
	}
	for _, unknown := range []string{accountID, "not-a-uuid"} {
		if err := repo.SetLanguage(ctx, unknown, "en"); !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("SetLanguage(%q) = %v, want ErrNotFound", unknown, err)
		}
		if err := repo.SetDetectedLanguage(ctx, unknown, "en"); !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("SetDetectedLanguage(%q) = %v, want ErrNotFound", unknown, err)
		}
	}
}
