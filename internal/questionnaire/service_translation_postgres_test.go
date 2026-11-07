package questionnaire_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The translations' life on the real database, as salus_app under row-level security: saved with a draft,
// refused at publish while incomplete, published with the version, copied into the next one.
func TestService_TranslationsOnTheDatabase(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seed(t, owner)
	svc, _ := realService(app)
	ctx := t.Context()

	partial := english()
	partial.Items = map[string]questionnaire.ItemTranslation{"q1": {Text: "Placeholder question in English"}}
	q, err := svc.Create(ctx, adminA, withTranslations(partial))
	if err != nil || q.Language != "hr" || !reflect.DeepEqual(q.Translations, []questionnaire.Translation{partial}) {
		t.Fatalf("Create() = %q %+v, %v; want clinic A's hr and the partial translation", q.Language, q.Translations, err)
	}
	read, err := svc.Read(ctx, adminA, q.ID)
	if err != nil || !reflect.DeepEqual(read.Translations, []questionnaire.Translation{partial}) {
		t.Fatalf("Read() = %+v, %v; want the partial translation", read.Translations, err)
	}
	_, err = svc.Publish(ctx, adminA, q.ID, q.VersionID)
	if incomplete, ok := errors.AsType[*questionnaire.TranslationIncompleteError](err); !ok || incomplete.Language != "en" || !slices.Equal(incomplete.LinkIDs, []string{"q2"}) {
		t.Fatalf("Publish() of the partial translation = %v, want translation_incomplete for en naming q2", err)
	}

	q, err = svc.UpdateDraft(ctx, adminA, q.ID, withTranslations(german(), english()), q.VersionID)
	if err != nil || !slices.Equal(languagesOf(q.Translations), []string{"en", "de"}) {
		t.Fatalf("UpdateDraft() = %v, %v; want en and de", languagesOf(q.Translations), err)
	}
	published, err := svc.Publish(ctx, adminA, q.ID, q.VersionID)
	if err != nil || published.Status != questionnaire.StatusActive || len(published.Translations) != 2 {
		t.Fatalf("Publish() = %s %v, %v; want active with both", published.Status, languagesOf(published.Translations), err)
	}
	next, err := svc.Revise(ctx, adminA, published.ID)
	if err != nil || next.Language != "hr" || !reflect.DeepEqual(next.Translations, []questionnaire.Translation{english(), german()}) {
		t.Errorf("Revise() = %q %+v, %v; want hr and both translations copied", next.Language, next.Translations, err)
	}
	found, err := svc.Search(ctx, adminA, questionnaire.Filter{SeriesID: published.SeriesID})
	if err != nil || len(found) != 2 || len(found[0].Translations) != 2 || len(found[1].Translations) != 2 {
		t.Errorf("Search() = %d versions, %v; want both with their translations", len(found), err)
	}
}
