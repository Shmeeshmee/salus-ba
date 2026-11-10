package terminology_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// fakeRepo records the search it was asked and returns its terms or its error.
type fakeRepo struct {
	err           error
	got           *terminology.Search
	glossaryQuery *terminology.GlossaryQuery
	terms         []terminology.Term
	glossary      []terminology.GlossaryEntry
}

func (f *fakeRepo) Search(_ context.Context, s terminology.Search) ([]terminology.Term, error) {
	f.got = &s
	return f.terms, f.err
}

func (f *fakeRepo) Glossary(_ context.Context, q terminology.GlossaryQuery) ([]terminology.GlossaryEntry, error) {
	f.glossaryQuery = &q
	return f.glossary, f.err
}

// fakeLanguages is the callers' UI languages by account id, counting the reads.
type fakeLanguages struct {
	err   error
	tags  map[string]string
	reads int
}

func (f *fakeLanguages) language(_ context.Context, accountID string) (string, error) {
	f.reads++
	return f.tags[accountID], f.err
}

var staff = terminology.Caller{AccountID: "a-staff", Staff: true}

func newService(repo *fakeRepo, langs *fakeLanguages) *terminology.Service {
	return terminology.NewService(repo, langs.language)
}

func TestSearch_RefusesACallerWhoIsNotStaff(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	svc := newService(repo, &fakeLanguages{})

	_, err := svc.Search(t.Context(), terminology.Caller{AccountID: "a-patient"}, terminology.Search{Text: "hepar", Limit: 10})

	if !errors.Is(err, terminology.ErrForbidden) || repo.got != nil {
		t.Errorf("Search as a patient = %v, searched %v; want ErrForbidden and no search", err, repo.got != nil)
	}
}

func TestSearch_RefusesAnInvalidSearch(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		s    terminology.Search
	}{
		{"empty text", terminology.Search{Text: "", Limit: 10}},
		{"blank text", terminology.Search{Text: " \t ", Limit: 10}},
		{"101 characters", terminology.Search{Text: strings.Repeat("č", 101), Limit: 10}},
		{"not UTF-8", terminology.Search{Text: "he\xffpar", Limit: 10}},
		{"a NUL", terminology.Search{Text: "he\x00par", Limit: 10}},
		{"limit 0", terminology.Search{Text: "hepar", Limit: 0}},
		{"limit 26", terminology.Search{Text: "hepar", Limit: 26}},
		{"unknown category", terminology.Search{Text: "hepar", Category: "organ", Limit: 10}},
		{"unknown language", terminology.Search{Text: "hepar", Languages: []string{"la", "xx"}, Limit: 10}},
		{"empty language", terminology.Search{Text: "hepar", Languages: []string{""}, Limit: 10}},
		{"repeated language", terminology.Search{Text: "hepar", Languages: []string{"la", "en", "la"}, Limit: 10}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			repo := &fakeRepo{}
			_, err := newService(repo, &fakeLanguages{}).Search(t.Context(), staff, c.s)
			if !errors.Is(err, terminology.ErrInvalidSearch) || repo.got != nil {
				t.Errorf("Search(%s) = %v, searched %v; want ErrInvalidSearch and no search", c.name, err, repo.got != nil)
			}
		})
	}
}

func TestSearch_AcceptsTheLimitsEdges(t *testing.T) {
	t.Parallel()
	for _, s := range []terminology.Search{
		{Text: "a", Limit: 1},
		{Text: strings.Repeat("č", 100), Limit: 25, Category: "drug", Languages: []string{"la", "bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}},
	} {
		if _, err := newService(&fakeRepo{}, &fakeLanguages{}).Search(t.Context(), staff, s); err != nil {
			t.Errorf("Search(%d characters, limit %d) = %v; want no error", len([]rune(s.Text)), s.Limit, err)
		}
	}
}

func TestSearch_TrimsTheTextAndPassesTheSearchOn(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	langs := &fakeLanguages{}

	_, err := newService(repo, langs).Search(t.Context(), staff,
		terminology.Search{Text: "  jetra ", Category: "anatomy", Languages: []string{"hr", "la"}, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}

	want := terminology.Search{Text: "jetra", Category: "anatomy", UILanguage: "hr", Languages: []string{"hr", "la"}, Limit: 5}
	if repo.got == nil || !reflect.DeepEqual(*repo.got, want) || langs.reads != 0 {
		t.Errorf("searched %+v, read the UI language %d times; want %+v and no read", repo.got, langs.reads, want)
	}
}

func TestSearch_DefaultLanguagesAreLatinTheUILanguageAndEnglish(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		ui   string
		want []string
	}{
		{"hr", []string{"la", "hr", "en"}},
		{"sr-Cyrl", []string{"la", "sr-Cyrl", "en"}},
		{"en", []string{"la", "en"}},
		{"", []string{"la", "en"}},
	} {
		repo := &fakeRepo{}
		langs := &fakeLanguages{tags: map[string]string{staff.AccountID: c.ui}}
		if _, err := newService(repo, langs).Search(t.Context(), staff, terminology.Search{Text: "jetra", Limit: 10}); err != nil {
			t.Fatal(err)
		}
		if repo.got == nil || !reflect.DeepEqual(repo.got.Languages, c.want) {
			t.Errorf("UI language %q: searched in %v; want %v", c.ui, repo.got, c.want)
		}
	}
}

// The ranking's UI language (labels in it and in Latin first): the account's, when the request names no
// languages; else the first requested language other than Latin, as the web asks for Latin, the UI language and
// English first.
func TestSearch_TheUILanguageOfTheRanking(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name      string
		ui        string
		want      string
		languages []string
	}{
		{"the account's, without languages", "hr", "hr", nil},
		{"English, for an account without one", "", "en", nil},
		{"the web's order: Latin, the UI language, English", "hr", "bs", []string{"la", "bs", "en", "hr"}},
		{"the web's order in English", "", "en", []string{"la", "en", "bs"}},
		{"none but Latin", "", "", []string{"la"}},
	} {
		repo := &fakeRepo{}
		langs := &fakeLanguages{tags: map[string]string{staff.AccountID: c.ui}}
		if _, err := newService(repo, langs).Search(t.Context(), staff, terminology.Search{Text: "jetra", Languages: c.languages, Limit: 10}); err != nil {
			t.Fatal(err)
		}
		if repo.got == nil || repo.got.UILanguage != c.want {
			t.Errorf("%s: searched %+v; want the UI language %q", c.name, repo.got, c.want)
		}
	}
}

func TestSearch_GroupsTheLabelsPerRequestedLanguageInRequestOrder(t *testing.T) {
	t.Parallel()
	reviewed := terminology.Term{ID: "t1", Category: "anatomy", WikidataID: "Q9368", TA98: "A05.8.01.001", ReviewStatus: "approved", Rank: 100,
		Labels: []terminology.Label{ // the repository's order: by language in the enum, preferred first, synonyms alphabetically
			{Language: "la", Text: "hepar", Preferred: true}, {Language: "la", Text: "iecur"},
			{Language: "hr", Text: "jetra", Preferred: true},
		}}
	unmarked := terminology.Term{ID: "t2", Category: "diagnosis", ICD10: "K25", ReviewStatus: "unreviewed",
		Labels: []terminology.Label{{Language: "en", Text: "gastric ulcer"}, {Language: "en", Text: "stomach ulcer"}}}
	repo := &fakeRepo{terms: []terminology.Term{reviewed, unmarked}}

	got, err := newService(repo, &fakeLanguages{}).Search(t.Context(), staff,
		terminology.Search{Text: "hepar", Languages: []string{"hr", "en", "la"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}

	reviewed.Labels, unmarked.Labels = nil, nil
	want := []terminology.Match{
		{Term: reviewed, Labels: []terminology.Names{
			{Language: "hr", Preferred: "jetra", Synonyms: []string{}},
			{Language: "la", Preferred: "hepar", Synonyms: []string{"iecur"}},
		}},
		{Term: unmarked, Labels: []terminology.Names{{Language: "en", Preferred: "gastric ulcer", Synonyms: []string{"stomach ulcer"}}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Search = %+v\nwant %+v", got, want)
	}
}

func TestSearch_NoMatchIsAnEmptyList(t *testing.T) {
	t.Parallel()
	got, err := newService(&fakeRepo{terms: []terminology.Term{}}, &fakeLanguages{}).Search(t.Context(), staff,
		terminology.Search{Text: "zzz", Limit: 10})
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("Search = %v, %v; want an empty, non-nil list", got, err)
	}
}

func TestSearch_ErrorsNeverCarryTheSearchText(t *testing.T) {
	t.Parallel()
	const text = "Marko Marković" // a doctor could type a patient's name
	failure := errors.New("connection reset")
	for _, c := range []struct {
		want   error
		repo   *fakeRepo
		langs  *fakeLanguages
		name   string
		search terminology.Search
	}{
		{failure, &fakeRepo{err: failure}, &fakeLanguages{}, "the repository fails", terminology.Search{Text: text, Languages: []string{"la"}, Limit: 10}},
		{failure, &fakeRepo{}, &fakeLanguages{err: failure}, "the UI language fails", terminology.Search{Text: text, Limit: 10}},
		{terminology.ErrInvalidSearch, &fakeRepo{}, &fakeLanguages{}, "the search is invalid", terminology.Search{Text: text, Limit: 99}},
	} {
		_, err := newService(c.repo, c.langs).Search(t.Context(), staff, c.search)
		if !errors.Is(err, c.want) || strings.Contains(err.Error(), "Marko") {
			t.Errorf("%s: Search = %v; want %v without the search text", c.name, err, c.want)
		}
	}
}
