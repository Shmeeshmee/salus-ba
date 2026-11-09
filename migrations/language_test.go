package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// Languages (docs/i18n/SPEC.md §3 "Languages", "The account's language fields", "The clinic's default language",
// "Questionnaires").

func TestLanguage_TheEnumHoldsTheTenTagsStaffLanguagesFirst(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	var tags []string
	if err := owner.QueryRow(t.Context(), "SELECT enum_range(NULL::language)::text[]").Scan(&tags); err != nil {
		t.Fatal(err)
	}
	want := []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}
	if !slices.Equal(tags, want) {
		t.Errorf("language: %v, want %v", tags, want)
	}
}

func TestLanguage_AnAccountHasNoLanguageUntilOneIsChosenOrDetected(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	var id string
	var lang, detected *string
	if err := app.QueryRow(ctx, insertAccount+" RETURNING id, language, detected_language", "lang@example.test", "patient", nil).
		Scan(&id, &lang, &detected); err != nil {
		t.Fatal(err)
	}
	if lang != nil || detected != nil {
		t.Errorf("a new account: language %v, detected %v; want neither", lang, detected)
	}
	if _, err := app.Exec(ctx, "UPDATE account SET language = 'ar', detected_language = 'de' WHERE id = $1", id); err != nil {
		t.Errorf("the app role sets both languages: %v", err)
	}
	if _, err := app.Exec(ctx, "UPDATE account SET language = NULL WHERE id = $1", id); err != nil {
		t.Errorf("the app role clears the chosen language: %v", err)
	}
	if _, err := app.Exec(ctx, "UPDATE account SET language = 'sr' WHERE id = $1", id); sqlState(err) != "22P02" {
		t.Errorf("a tag outside the enum: %v, want invalid_text_representation", err)
	}
}

func TestLanguage_AClinicDefaultsToBosnianAndTakesOnlyAStaffLanguage(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	seedDirectory(t, owner)
	var def string
	if err := owner.QueryRow(t.Context(), "SELECT default_language FROM organization WHERE id = $1", clinicA).Scan(&def); err != nil || def != "bs" {
		t.Errorf("a new clinic's default language: %q, %v; want bs", def, err)
	}
	for lang, want := range map[string]string{"en": "", "sr-Cyrl": "", "de": "23514", "ar": "23514"} {
		err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
			_, err := db.Exec(ctx, "UPDATE organization SET default_language = $1", lang)
			return err
		})
		if got := sqlState(err); got != want {
			t.Errorf("default language %s as %s: %v, want SQLSTATE %q", lang, pgtest.AppRole, err, want)
		}
	}
}

func TestLanguage_AQuestionnaireIsInBosnianUnlessSetAndADraftChangesIt(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	q := seedQuestionnaires(t, owner)[clinicA]
	var lang string
	if err := owner.QueryRow(t.Context(), "SELECT language FROM questionnaire WHERE id = $1", q.active).Scan(&lang); err != nil || lang != "bs" {
		t.Errorf("a questionnaire with no language given: %q, %v; want bs", lang, err)
	}
	if err := inClinic(t, app, clinicA, func(ctx context.Context, db postgres.Querier) error {
		_, err := db.Exec(ctx, "UPDATE questionnaire SET language = 'en' WHERE id = $1", q.draft)
		return err
	}); err != nil {
		t.Errorf("the app role sets a draft's language: %v", err)
	}
}

// seedTranslations stores, as the owner, an English translation of every questionnaire seedResponses made.
func seedTranslations(t *testing.T, owner *pgxpool.Pool, r responseRows) {
	t.Helper()
	for _, org := range []string{clinicA, clinicB} {
		for _, id := range append(slices.Clone(r.versions[org]), r.draft[org]) {
			if _, err := owner.Exec(t.Context(), `INSERT INTO questionnaire_translation (questionnaire_id, language, texts)
				VALUES ($1, 'en', '{"title": "Placeholder questionnaire", "items": {}}')`, id); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestQuestionnaireTranslation_EachScopeSeesTheTranslationsOfTheQuestionnairesItSees(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, r := seedResponses(t, owner)
	seedTranslations(t, owner, r)
	for _, tt := range []struct {
		scope map[string]string
		name  string
		want  []string
	}{
		{name: "no scope", scope: map[string]string{}},
		{name: "clinic A", scope: map[string]string{"app.organization_id": clinicA}, want: sorted(append(slices.Clone(r.versions[clinicA]), r.draft[clinicA])...)},
		{name: "patient A", scope: map[string]string{"app.patient_id": p[0].patient}, want: sorted(append(slices.Clone(r.versions[clinicA]), r.versions[clinicB]...)...)},
		{name: "patient B", scope: map[string]string{"app.patient_id": p[1].patient}, want: sorted(r.versions[clinicA]...)},
	} {
		got := idsWith(t, app, tt.scope, "SELECT questionnaire_id::text FROM questionnaire_translation")
		if !slices.Equal(sorted(got...), tt.want) {
			t.Errorf("%s sees the translations of %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestQuestionnaireTranslation_OnlyTheClinicWritesAndOnlyADraftsTranslations(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, _, r := seedResponses(t, owner)
	clinic := map[string]string{"app.organization_id": clinicA}
	patient := map[string]string{"app.patient_id": p[0].patient}

	insert := "INSERT INTO questionnaire_translation (questionnaire_id, language, texts) VALUES ($1, 'de', '{}')"
	if _, err := exec(t, app, clinic, insert, r.draft[clinicA]); err != nil {
		t.Errorf("clinic A translates its draft: %v", err)
	}
	for name, tt := range map[string]struct {
		scope map[string]string
		id    string
	}{
		"clinic A, its active version":  {scope: clinic, id: r.versions[clinicA][0]},
		"clinic A, clinic B's draft":    {scope: clinic, id: r.draft[clinicB]},
		"patient A, a version they see": {scope: patient, id: r.versions[clinicA][0]},
	} {
		if _, err := exec(t, app, tt.scope, insert, tt.id); sqlState(err) != "42501" {
			t.Errorf("%s: %v, want insufficient_privilege", name, err)
		}
	}

	seedTranslations(t, owner, r)
	for _, stmt := range []string{ // in order: the update first
		"UPDATE questionnaire_translation SET texts = '{\"title\": \"Renamed placeholder\"}', updated_at = now() WHERE language = 'en'",
		"DELETE FROM questionnaire_translation WHERE language = 'en'",
	} {
		if n, err := exec(t, app, clinic, stmt); err != nil || n != 1 {
			t.Errorf("%s in clinic A: %d rows, %v; want only the draft's", stmt, n, err)
		}
		if n, err := exec(t, app, patient, stmt); err != nil || n != 0 {
			t.Errorf("%s as patient A: %d rows, %v; want none", stmt, n, err)
		}
	}
	for stmt, want := range map[string]string{
		"UPDATE questionnaire_translation SET language = 'fr'":                      "42501",
		"UPDATE questionnaire_translation SET questionnaire_id = gen_random_uuid()": "42501",
		"TRUNCATE questionnaire_translation":                                        "42501",
	} {
		if _, err := exec(t, app, clinic, stmt); sqlState(err) != want {
			t.Errorf("%s in clinic A: %v, want SQLSTATE %s", stmt, err, want)
		}
	}
}

func TestQuestionnaireTranslation_HoldsOneObjectPerLanguageAndGoesWithItsDraft(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	q := seedQuestionnaires(t, owner)[clinicA]
	ctx := t.Context()
	put := "INSERT INTO questionnaire_translation (questionnaire_id, language, texts) VALUES ($1, $2, $3)"
	if _, err := owner.Exec(ctx, put, q.draft, "en", "{}"); err != nil {
		t.Fatal(err)
	}
	for name, tt := range map[string]struct {
		lang, texts, want string
	}{
		"a language twice":       {lang: "en", texts: "{}", want: "23505"},
		"texts that are a list":  {lang: "de", texts: "[]", want: "23514"},
		"a tag outside the enum": {lang: "xx", texts: "{}", want: "22P02"},
	} {
		if _, err := owner.Exec(ctx, put, q.draft, tt.lang, tt.texts); sqlState(err) != tt.want {
			t.Errorf("%s: %v, want SQLSTATE %s", name, err, tt.want)
		}
	}
	if _, err := owner.Exec(ctx, "DELETE FROM questionnaire WHERE id = $1", q.draft); err != nil {
		t.Fatal(err)
	}
	var left int
	if err := owner.QueryRow(ctx, "SELECT count(*) FROM questionnaire_translation").Scan(&left); err != nil || left != 0 {
		t.Errorf("translations after the draft was deleted: %d, %v; want none", left, err)
	}
}

func TestQuestionnaireResponse_TheLanguageIsUnsetUntilThePatientStarts(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	p, a, r := seedResponses(t, owner)
	var unset int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM questionnaire_response WHERE language IS NOT NULL").Scan(&unset); err != nil || unset != 0 {
		t.Errorf("responses with a language before anyone started: %d, %v; want none", unset, err)
	}
	n, err := exec(t, app, map[string]string{"app.patient_id": p[0].patient},
		"UPDATE questionnaire_response SET language = 'en' WHERE id = $1", r.open[a.aFuture])
	if err != nil || n != 1 {
		t.Errorf("patient A sets the language of their response: %d rows, %v; want 1", n, err)
	}
}
