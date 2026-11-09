package questionnaire_test

import (
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// docs/i18n/SPEC.md §3 "Questionnaires": the original language, the translations, complete or nothing.

// english is a complete English translation of content() (placeholder wording).
func english() questionnaire.Translation {
	return questionnaire.Translation{
		Language: "en", Title: "Placeholder title in English", Description: "Placeholder description in English",
		Items: map[string]questionnaire.ItemTranslation{
			"q1": {Text: "Placeholder question in English"},
			"q2": {Text: "Placeholder choice in English", Options: []string{"Placeholder first", "Placeholder second"}},
		},
	}
}

// german is a complete German translation of content() (placeholder wording).
func german() questionnaire.Translation {
	t := english()
	t.Language, t.Title = "de", "Placeholder title in German"
	return t
}

func withTranslations(ts ...questionnaire.Translation) questionnaire.Content {
	in := content()
	in.Translations = ts
	return in
}

func languagesOf(ts []questionnaire.Translation) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.Language)
	}
	return out
}

func TestCreate_TakesTheClinicsDefaultLanguageUnlessOneIsGiven(t *testing.T) {
	t.Parallel()
	f := newFixture()

	byDefault, err := f.svc.Create(t.Context(), practitionerA1, content())
	if err != nil || byDefault.Language != "hr" || f.row(byDefault.ID).Language != "hr" {
		t.Errorf("Create() without a language = %q (stored %q), %v; want clinic A's default hr", byDefault.Language, f.row(byDefault.ID).Language, err)
	}
	in := content()
	in.Language = "de"
	german, err := f.svc.Create(t.Context(), practitionerA1, in)
	if err != nil || german.Language != "de" || f.row(german.ID).Language != "de" {
		t.Errorf("Create() in German = %q, %v; want de", german.Language, err)
	}
	translated := withTranslations(english())
	translated.Translations[0].Language = "hr"
	if _, err := f.svc.Create(t.Context(), practitionerA1, translated); !errors.Is(err, questionnaire.ErrInvalid) {
		t.Errorf("Create() with a translation in the clinic's default language = %v, want invalid", err)
	}
	if !slices.Equal(f.clinics.asked, []string{clinicA, clinicA}) {
		t.Errorf("default language asked for %v, want clinic A for each create without a language", f.clinics.asked)
	}
}

func TestCreate_RollsBackWhenTheDefaultLanguageCannotBeRead(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.clinics.err = errors.New("fake: organization unavailable")

	if _, err := f.svc.Create(t.Context(), practitionerA1, content()); !errors.Is(err, f.clinics.err) {
		t.Errorf("Create() = %v, want the organization's error wrapped", err)
	}
	if len(f.repo.rows) != 3 || len(f.audit.events) != 0 {
		t.Errorf("rows %d, events %d after a failed read; want nothing stored", len(f.repo.rows), len(f.audit.events))
	}
}

func TestCreate_StoresTheTranslationsInTheLanguageListsOrder(t *testing.T) {
	t.Parallel()
	f := newFixture()

	got, err := f.svc.Create(t.Context(), practitionerA1, withTranslations(german(), english()))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"en", "de"}; !slices.Equal(languagesOf(got.Translations), want) || !slices.Equal(languagesOf(f.repo.translated[got.ID]), want) {
		t.Errorf("translations returned %v, stored %v; want %v", languagesOf(got.Translations), languagesOf(f.repo.translated[got.ID]), want)
	}
	if !reflect.DeepEqual(got.Translations[0], english()) {
		t.Errorf("English translation = %+v, want %+v", got.Translations[0], english())
	}
}

func TestSave_RefusesAnInvalidLanguageOrTranslation(t *testing.T) {
	t.Parallel()
	long := func(n int) string { return strings.Repeat("x", n) }
	edit := func(fn func(*questionnaire.Translation)) questionnaire.Translation {
		t := english()
		t.Items = maps.Clone(t.Items)
		fn(&t)
		return t
	}
	tests := []struct {
		name    string
		in      questionnaire.Content
		linkIDs []string
	}{
		{name: "unknown original language", in: func() questionnaire.Content { c := content(); c.Language = "xx"; return c }()},
		{name: "unknown translation language", in: withTranslations(edit(func(t *questionnaire.Translation) { t.Language = "es" }))},
		{name: "translation in the original language", in: func() questionnaire.Content {
			c := withTranslations(english())
			c.Language = "en"
			return c
		}()},
		{name: "two translations in one language", in: withTranslations(english(), english())},
		{name: "title over 200", in: withTranslations(edit(func(t *questionnaire.Translation) { t.Title = long(201) }))},
		{name: "description over 1000", in: withTranslations(edit(func(t *questionnaire.Translation) { t.Description = long(1001) }))},
		{name: "text over 500", in: withTranslations(edit(func(t *questionnaire.Translation) { t.Items["q1"] = questionnaire.ItemTranslation{Text: long(501)} })), linkIDs: []string{"q1"}},
		{name: "help over 500", in: func() questionnaire.Content {
			c := withTranslations(edit(func(t *questionnaire.Translation) {
				t.Items["q1"] = questionnaire.ItemTranslation{Text: "Placeholder", Help: long(501)}
			}))
			c.Items[0].Help = "Placeholder help"
			return c
		}(), linkIDs: []string{"q1"}},
		{name: "help the original lacks", in: withTranslations(edit(func(t *questionnaire.Translation) {
			t.Items["q1"] = questionnaire.ItemTranslation{Text: "Placeholder", Help: "Placeholder help"}
		})), linkIDs: []string{"q1"}},
		{name: "options on a question without", in: withTranslations(edit(func(t *questionnaire.Translation) {
			t.Items["q1"] = questionnaire.ItemTranslation{Text: "Placeholder", Options: []string{"A", "B"}}
		})), linkIDs: []string{"q1"}},
		{name: "fewer options", in: withTranslations(edit(func(t *questionnaire.Translation) {
			t.Items["q2"] = questionnaire.ItemTranslation{Text: "Placeholder", Options: []string{"Placeholder first"}}
		})), linkIDs: []string{"q2"}},
		{name: "option over 200", in: withTranslations(edit(func(t *questionnaire.Translation) {
			t.Items["q2"] = questionnaire.ItemTranslation{Options: []string{"Placeholder first", long(201)}}
		})), linkIDs: []string{"q2"}},
		{name: "the same option twice", in: withTranslations(edit(func(t *questionnaire.Translation) {
			t.Items["q2"] = questionnaire.ItemTranslation{Options: []string{"Placeholder same", "Placeholder same"}}
		})), linkIDs: []string{"q2"}},
		{name: "both items, in document order", in: withTranslations(edit(func(t *questionnaire.Translation) {
			t.Items["q2"] = questionnaire.ItemTranslation{Options: []string{"Placeholder first"}}
			t.Items["q1"] = questionnaire.ItemTranslation{Text: long(501)}
		})), linkIDs: []string{"q1", "q2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for name, save := range map[string]func(f fixture) error{
				"Create": func(f fixture) error { _, err := f.svc.Create(t.Context(), practitionerA1, tt.in); return err },
				"UpdateDraft": func(f fixture) error {
					_, err := f.svc.UpdateDraft(t.Context(), practitionerA1, draftA, tt.in, 1)
					return err
				},
			} {
				f := newFixture()
				err := save(f)
				invalid, ok := errors.AsType[*questionnaire.InvalidError](err)
				if !ok || !slices.Equal(invalid.LinkIDs, tt.linkIDs) {
					t.Errorf("%s() = %v, want an InvalidError naming %v", name, err, tt.linkIDs)
					continue
				}
				if len(f.repo.translated) != 0 || len(f.repo.rows) != 3 || f.row(draftA).VersionID != 1 {
					t.Errorf("%s() stored something: %+v", name, f.repo.translated)
				}
				assertNoContent(t, err)
			}
		})
	}
}

// A translation may be incomplete while it is a draft.
func TestSave_AcceptsAnIncompleteTranslation(t *testing.T) {
	t.Parallel()
	f := newFixture()
	partial := questionnaire.Translation{Language: "en", Items: map[string]questionnaire.ItemTranslation{
		"q2": {Options: []string{"", "Placeholder second"}},
	}}

	got, err := f.svc.UpdateDraft(t.Context(), practitionerA1, draftA, withTranslations(partial), 1)
	if err != nil || len(got.Translations) != 1 || !reflect.DeepEqual(got.Translations[0], partial) {
		t.Errorf("UpdateDraft() = %+v, %v; want the partial translation stored", got.Translations, err)
	}
}

func TestUpdateDraft_SetsTheLanguageAndTheTranslations(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.repo.translated = map[string][]questionnaire.Translation{draftA: {english()}}

	kept, err := f.svc.UpdateDraft(t.Context(), practitionerA1, draftA, content(), 1)
	if err != nil || kept.Language != "bs" || !reflect.DeepEqual(kept.Translations, []questionnaire.Translation{english()}) {
		t.Fatalf("UpdateDraft() without language or translations = %q %+v, %v; want the draft's kept", kept.Language, kept.Translations, err)
	}
	in := withTranslations(german())
	in.Language = "en"
	set, err := f.svc.UpdateDraft(t.Context(), practitionerA1, draftA, in, 2)
	if err != nil || set.Language != "en" || f.row(draftA).Language != "en" || !slices.Equal(languagesOf(f.repo.translated[draftA]), []string{"de"}) {
		t.Fatalf("UpdateDraft() in English with German = %q %v, %v; want en and only de", set.Language, languagesOf(f.repo.translated[draftA]), err)
	}
	none := content()
	none.Translations = []questionnaire.Translation{}
	if _, err := f.svc.UpdateDraft(t.Context(), practitionerA1, draftA, none, 3); err != nil || len(f.repo.translated[draftA]) != 0 {
		t.Errorf("UpdateDraft() with no translations = %v, stored %v; want them removed", err, f.repo.translated[draftA])
	}
}

// An item whose wording changes loses that wording's translation unless the same save translates it again; a
// removed item's translation goes with it (SPEC §3: texts of items that changed are missing until translated).
func TestUpdateDraft_DropsTheTranslationOfAChangedText(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.repo.translated = map[string][]questionnaire.Translation{draftA: {english(), german()}}
	in := content()
	in.Items[0].Text = "Placeholder question, reworded"
	in.Items[1].Options = []questionnaire.Option{{Value: "One"}, {Value: "Two"}, {Value: "Three"}}
	in.Items = append(in.Items, question("q3", questionnaire.ItemString))
	resent := german()
	resent.Items = maps.Clone(resent.Items)
	resent.Items["q1"] = questionnaire.ItemTranslation{Text: "Placeholder question in German, reworded"}
	in.Translations = []questionnaire.Translation{english(), resent}

	if _, err := f.svc.UpdateDraft(t.Context(), practitionerA1, draftA, in, 1); err != nil {
		t.Fatal(err)
	}
	want := []questionnaire.Translation{english(), german()}
	want[0].Items = map[string]questionnaire.ItemTranslation{"q2": {Text: "Placeholder choice in English"}}
	want[1].Items = map[string]questionnaire.ItemTranslation{
		"q1": {Text: "Placeholder question in German, reworded"},
		"q2": {Text: "Placeholder choice in English"},
	}
	if got := f.repo.translated[draftA]; !reflect.DeepEqual(got, want) {
		t.Errorf("translations after rewording q1 and adding an option to q2:\n got %+v\nwant %+v", got, want)
	}

	removed := content()
	removed.Items = removed.Items[1:]
	if _, err := f.svc.UpdateDraft(t.Context(), practitionerA1, draftA, removed, 2); err != nil {
		t.Fatal(err)
	}
	for _, tr := range f.repo.translated[draftA] {
		if _, ok := tr.Items["q1"]; ok {
			t.Errorf("%s translation keeps the removed item q1: %+v", tr.Language, tr.Items)
		}
	}
}

func TestPublish_RefusesAnIncompleteTranslation(t *testing.T) {
	t.Parallel()
	missingQ2 := german()
	missingQ2.Items = map[string]questionnaire.ItemTranslation{"q1": {Text: "Placeholder"}}
	blankOption := english()
	blankOption.Items = maps.Clone(blankOption.Items)
	blankOption.Items["q2"] = questionnaire.ItemTranslation{Text: "Placeholder", Options: []string{"Placeholder first", " "}}
	noTitle := english()
	noTitle.Title = ""
	noDescription := english()
	noDescription.Description = ""
	noHelp := english()
	noHelp.Items = maps.Clone(noHelp.Items)
	noHelp.Items["q1"] = questionnaire.ItemTranslation{Text: "Placeholder"}
	tests := []struct {
		name     string
		ts       []questionnaire.Translation
		language string
		linkIDs  []string
		help     bool // q1 has help text
	}{
		{name: "an item missing", ts: []questionnaire.Translation{english(), missingQ2}, language: "de", linkIDs: []string{"q2"}},
		{name: "a blank option, the first in the list", ts: []questionnaire.Translation{blankOption, missingQ2}, language: "en", linkIDs: []string{"q2"}},
		{name: "the title only", ts: []questionnaire.Translation{noTitle}, language: "en"},
		{name: "the description", ts: []questionnaire.Translation{noDescription}, language: "en"},
		{name: "a help text", help: true, ts: []questionnaire.Translation{noHelp}, language: "en", linkIDs: []string{"q1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			f.repo.rows[1].Description = "Placeholder description"
			if tt.help {
				f.repo.rows[1].Items = items()
				f.repo.rows[1].Items[0].Help = "Placeholder help"
			}
			f.repo.translated = map[string][]questionnaire.Translation{draftA: tt.ts}

			_, err := f.svc.Publish(t.Context(), practitionerA1, draftA, 1)
			incomplete, ok := errors.AsType[*questionnaire.TranslationIncompleteError](err)
			if !ok || !errors.Is(err, questionnaire.ErrTranslationIncomplete) || incomplete.Language != tt.language || !slices.Equal(incomplete.LinkIDs, tt.linkIDs) {
				t.Fatalf("Publish() = %+v, want translation_incomplete for %s naming %v", err, tt.language, tt.linkIDs)
			}
			if f.row(draftA).Status != questionnaire.StatusDraft || f.row(activeA).Status != questionnaire.StatusActive {
				t.Error("Publish() changed a version although refused")
			}
			assertEvents(t, f.audit, staffEvent(practitionerA1, auditevent.CodeUpdate, auditevent.ActionUpdate, "Questionnaire/"+draftA, auditevent.OutcomeMinorFailure))
			assertNoContent(t, err)
		})
	}
}

func TestPublish_PublishesCompleteTranslationsWithTheVersion(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.repo.translated = map[string][]questionnaire.Translation{draftA: {english(), german()}}

	got, err := f.svc.Publish(t.Context(), practitionerA1, draftA, 1)
	if err != nil || got.Status != questionnaire.StatusActive || !slices.Equal(languagesOf(got.Translations), []string{"en", "de"}) {
		t.Errorf("Publish() = %s %v, %v; want active with en and de", got.Status, languagesOf(got.Translations), err)
	}
}

func TestRevise_CopiesTheTranslations(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.repo.rows = f.repo.rows[:1] // series A without its draft
	f.repo.translated = map[string][]questionnaire.Translation{activeA: {english()}}

	got, err := f.svc.Revise(t.Context(), practitionerA1, activeA)
	if err != nil || !reflect.DeepEqual(got.Translations, []questionnaire.Translation{english()}) || got.Language != "bs" {
		t.Errorf("Revise() = %q %+v, %v; want bs with the English translation", got.Language, got.Translations, err)
	}
}

// Staff read every translation, incomplete ones of a draft included; a patient only complete ones.
func TestRead_GivesTheTranslations(t *testing.T) {
	t.Parallel()
	f := newFixture()
	partial := german()
	partial.Title = ""
	f.repo.translated = map[string][]questionnaire.Translation{activeA: {english(), partial}, draftA: {partial}}
	me := questionnaire.Caller{AccountID: patient.AccountID, PatientID: patientA}

	staff, err := f.svc.Read(t.Context(), practitionerA1, activeA)
	if err != nil || !slices.Equal(languagesOf(staff.Translations), []string{"en", "de"}) {
		t.Errorf("Read() as staff = %v, %v; want en and de", languagesOf(staff.Translations), err)
	}
	found, err := f.svc.Search(t.Context(), practitionerA1, questionnaire.Filter{SeriesID: seriesA})
	if err != nil || len(found) != 2 || len(found[0].Translations) != 2 || len(found[1].Translations) != 1 {
		t.Errorf("Search() as staff = %+v, %v; want each version with its translations", found, err)
	}
	if len(f.repo.outside) != 0 {
		t.Errorf("repository calls outside a transaction: %v", f.repo.outside)
	}

	f = newFixture() // the fake keeps the clinic scope a staff call set
	f.repo.translated = map[string][]questionnaire.Translation{activeA: {english(), partial}, draftA: {partial}}
	f.repo.referenced = map[string][]string{patientA: {activeA}}
	mine, err := f.svc.Read(t.Context(), me, activeA)
	if err != nil || !slices.Equal(languagesOf(mine.Translations), []string{"en"}) {
		t.Errorf("Read() as the patient = %v, %v; want only the complete en", languagesOf(mine.Translations), err)
	}
	listed, err := f.svc.Search(t.Context(), me, questionnaire.Filter{})
	if err != nil || len(listed) != 1 || !slices.Equal(languagesOf(listed[0].Translations), []string{"en"}) {
		t.Errorf("Search() as the patient = %+v, %v; want only the complete en", listed, err)
	}
	if len(f.repo.outside) != 0 {
		t.Errorf("repository calls outside a transaction: %v", f.repo.outside)
	}
}

func TestService_ReturnsTheTranslationsErrorWrapped(t *testing.T) {
	t.Parallel()
	f := newFixture()
	boom := errors.New("fake: translations unavailable")
	f.repo.translationsErr = boom

	if _, err := f.svc.Read(t.Context(), practitionerA1, activeA); !errors.Is(err, boom) {
		t.Errorf("Read() = %v, want the error wrapped", err)
	}
	if _, err := f.svc.Publish(t.Context(), practitionerA1, draftA, 1); !errors.Is(err, boom) {
		t.Errorf("Publish() = %v, want the error wrapped", err)
	}
	if _, err := f.svc.UpdateDraft(t.Context(), practitionerA1, draftA, content(), 1); !errors.Is(err, boom) {
		t.Errorf("UpdateDraft() = %v, want the error wrapped", err)
	}
}
