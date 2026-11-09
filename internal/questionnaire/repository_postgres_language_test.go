package questionnaire_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// docs/i18n/SPEC.md §3 "Questionnaires": the original language and the translations of a version.

func TestPostgres_StoresTheOriginalLanguage(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := questionnaire.NewPostgresRepository(app)
	at := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)

	var none, english, kept, croatian, revised questionnaire.Questionnaire
	err := in(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		if none, err = repo.Create(ctx, questionnaire.Questionnaire{OrganizationID: clinicA, Title: "Placeholder"}, at); err != nil {
			return err
		}
		if english, err = repo.Create(ctx, questionnaire.Questionnaire{OrganizationID: clinicA, Title: "Placeholder", Language: "en"}, at); err != nil {
			return err
		}
		if kept, err = repo.UpdateDraft(ctx, questionnaire.Questionnaire{ID: english.ID, VersionID: 1, Title: "Placeholder"}, at); err != nil {
			return err
		}
		if croatian, err = repo.UpdateDraft(ctx, questionnaire.Questionnaire{ID: english.ID, VersionID: 2, Title: "Placeholder", Language: "hr"}, at); err != nil {
			return err
		}
		if _, err = repo.Publish(ctx, english.ID, at); err != nil {
			return err
		}
		revised, err = repo.CreateVersion(ctx, croatian, at)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, tt := range map[string]struct{ got, want string }{
		"Create() with no language":           {got: none.Language, want: "bs"},
		"Create() in English":                 {got: english.Language, want: "en"},
		"UpdateDraft() with no language":      {got: kept.Language, want: "en"},
		"UpdateDraft() in Croatian":           {got: croatian.Language, want: "hr"},
		"CreateVersion() of the Croatian one": {got: revised.Language, want: "hr"},
	} {
		if tt.got != tt.want {
			t.Errorf("%s: language %q, want %q", name, tt.got, tt.want)
		}
	}
	var read questionnaire.Questionnaire
	if err := in(t, app, repo, clinicA, func(ctx context.Context) error { var err error; read, err = repo.ByID(ctx, activeA); return err }); err != nil || read.Language != "bs" {
		t.Errorf("ByID() of a seeded version: language %q, %v; want bs", read.Language, err)
	}
}

func TestPostgres_ReplacesADraftsTranslationsAndANewVersionCopiesThem(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	repo := questionnaire.NewPostgresRepository(app)
	at := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	german := questionnaire.Translation{Language: "de", Title: "Platzhalter", Description: "Platzhalter Beschreibung",
		Items: map[string]questionnaire.ItemTranslation{"q1": {Text: "Platzhalterfrage", Help: "Platzhalterhilfe", Options: []string{"Eins", "Zwei"}}}}
	english := questionnaire.Translation{Language: "en", Title: "Placeholder", Items: map[string]questionnaire.ItemTranslation{"q1": {Text: "Placeholder question"}}}

	var first, replaced, unknown []questionnaire.Translation
	err := in(t, app, repo, clinicA, func(ctx context.Context) error {
		var err error
		if err = repo.PutTranslations(ctx, draftA, []questionnaire.Translation{german, english}, at); err != nil {
			return err
		}
		if first, err = repo.Translations(ctx, draftA); err != nil {
			return err
		}
		if err = repo.PutTranslations(ctx, draftA, []questionnaire.Translation{english}, at); err != nil {
			return err
		}
		if replaced, err = repo.Translations(ctx, draftA); err != nil {
			return err
		}
		unknown, err = repo.Translations(ctx, "not-a-uuid")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []questionnaire.Translation{english, german}; !reflect.DeepEqual(first, want) {
		t.Errorf("Translations() = %+v;\nwant %+v (in the enum's order)", first, want)
	}
	if want := []questionnaire.Translation{english}; !reflect.DeepEqual(replaced, want) {
		t.Errorf("Translations() after a second put = %+v; want only %+v", replaced, want)
	}
	if len(unknown) != 0 {
		t.Errorf("Translations() of an id that is not a UUID = %+v; want none", unknown)
	}

	// A published version's translations stay as they are.
	err = in(t, app, repo, clinicA, func(ctx context.Context) error {
		return repo.PutTranslations(ctx, activeA, []questionnaire.Translation{english}, at)
	})
	if err == nil {
		t.Error("PutTranslations() of an active version: accepted, want refused")
	}

	if _, err := owner.Exec(t.Context(), `INSERT INTO questionnaire_translation (questionnaire_id, language, texts)
		VALUES ($1, 'de', '{"title": "Platzhalter B", "items": {}}')`, activeB); err != nil {
		t.Fatal(err)
	}
	var copied []questionnaire.Translation
	err = in(t, app, repo, clinicB, func(ctx context.Context) error {
		from, err := repo.ByID(ctx, activeB)
		if err != nil {
			return err
		}
		next, err := repo.CreateVersion(ctx, from, at)
		if err != nil {
			return err
		}
		copied, err = repo.Translations(ctx, next.ID)
		return err
	})
	want := []questionnaire.Translation{{Language: "de", Title: "Platzhalter B", Items: map[string]questionnaire.ItemTranslation{}}}
	if err != nil || !reflect.DeepEqual(copied, want) {
		t.Errorf("the new version's translations = %+v, %v;\nwant %+v", copied, err, want)
	}
}
