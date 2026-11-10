package terminology_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// Tagging (SAL-13) looks up a text's phrases as the glossary does, in the text's own language, and gives each term
// found its place in the text: UTF-16 offsets, as the web and the mobile app index strings. A phrase found by a
// shortened last word marks the whole word.
func TestTag_MarksTheTermsOfEachTextInItsLanguage(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{tagged: []terminology.CandidateTerm{
		{Language: "en", Phrase: "tonsillitis", TermID: "t-tonsillitis"},
		{Language: "de", Phrase: "Kopfschmerz", TermID: "t-headache"},
	}}
	got, err := newService(repo, &fakeLanguages{}).Tag(t.Context(), []terminology.Text{
		{Language: "en", Body: "😀 tonsillitis, again."},
		{Language: "de", Body: "Seit gestern Kopfschmerzen."},
		{Language: "bs", Body: "Ništa."},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]terminology.Span{
		{{TermID: "t-tonsillitis", Start: 3, End: 14}}, // the emoji is two UTF-16 units
		{{TermID: "t-headache", Start: 13, End: 26}},
		nil,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Tag() = %+v, want %+v", got, want)
	}
	for _, c := range []terminology.Candidate{{Language: "en", Phrase: "tonsillitis"}, {Language: "de", Phrase: "Kopfschmerz"}, {Language: "bs", Phrase: "Ništa"}} {
		if !slices.Contains(repo.tagQuery, c) {
			t.Errorf("candidates %+v lack %+v", repo.tagQuery, c)
		}
	}
}

// Where found terms overlap, the one that starts first wins, and of those the longest: "stomach ulcer" over
// "ulcer". A term said twice is marked twice.
func TestTag_KeepsTheLongestOfOverlappingTerms(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{tagged: []terminology.CandidateTerm{
		{Language: "en", Phrase: "ulcer", TermID: "t-ulcer"},
		{Language: "en", Phrase: "stomach ulcer", TermID: "t-stomach-ulcer"},
		{Language: "en", Phrase: "stomach", TermID: "t-stomach"},
	}}
	got, err := newService(repo, &fakeLanguages{}).Tag(t.Context(), []terminology.Text{{Language: "en", Body: "A stomach ulcer; an ulcer."}})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]terminology.Span{{{TermID: "t-stomach-ulcer", Start: 2, End: 15}, {TermID: "t-ulcer", Start: 20, End: 25}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Tag() = %+v, want %+v", got, want)
	}
}

// A text in Latin or a language that is none of the app's is refused; texts without words are no lookup; the
// repository's failure comes back without the text.
func TestTag_RefusesBadLanguagesAndReturnsTheRepositorysError(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	svc := newService(repo, &fakeLanguages{})
	for _, lang := range []string{"la", "xx", ""} {
		if _, err := svc.Tag(t.Context(), []terminology.Text{{Language: lang, Body: "hepar"}}); !errors.Is(err, terminology.ErrInvalidSearch) {
			t.Errorf("Tag(%q) = %v, want ErrInvalidSearch", lang, err)
		}
	}
	got, err := svc.Tag(t.Context(), []terminology.Text{{Language: "en", Body: " ... "}})
	if err != nil || !reflect.DeepEqual(got, [][]terminology.Span{nil}) || repo.tagQuery != nil {
		t.Errorf("a text without words = %+v, %v, looked up %v; want no spans and no lookup", got, err, repo.tagQuery != nil)
	}
	repo.err = errors.New("the database is down")
	if _, err := svc.Tag(t.Context(), []terminology.Text{{Language: "en", Body: "placeholder-marker"}}); err == nil || strings.Contains(err.Error(), "marker") {
		t.Errorf("Tag() = %v, want the repository's error without the text", err)
	}
}

// The explanations a viewer reads are in their language, else in English; an explanation that needs changes is
// not shown. Any signed-in caller may read them: they are reference data.
func TestExplanations_AsksForTheViewersLanguageWithEnglishAsTheFallback(t *testing.T) {
	t.Parallel()
	want := []terminology.Explanation{{TermID: "t-1", Language: "en", Text: "Placeholder explanation", Review: terminology.Review{Status: terminology.ReviewUnreviewed}}}
	repo := &fakeRepo{explanations: want}
	svc := newService(repo, &fakeLanguages{})
	got, err := svc.Explanations(t.Context(), "bs")
	if err != nil || !reflect.DeepEqual(got, want) || repo.explanationsLanguage != "bs" {
		t.Errorf("Explanations(bs) = %+v, %v, asked %q; want the repository's in bs", got, err, repo.explanationsLanguage)
	}
	for _, lang := range []string{"la", "xx"} {
		if _, err := svc.Explanations(t.Context(), lang); !errors.Is(err, terminology.ErrInvalidSearch) {
			t.Errorf("Explanations(%q) = %v, want ErrInvalidSearch", lang, err)
		}
	}
}
