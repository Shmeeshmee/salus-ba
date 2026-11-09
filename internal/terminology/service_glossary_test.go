package terminology_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// The glossary of a line (SAL-5): its candidate phrases go to the repository with the two languages, and the
// entries come back as the repository found them.
func TestGlossary_LooksUpTheLinesPhrasesInItsLanguage(t *testing.T) {
	t.Parallel()
	want := []terminology.GlossaryEntry{{Source: "Kopfschmerz", Target: "glavobolja"}}
	repo := &fakeRepo{glossary: want}
	got, err := newService(repo, &fakeLanguages{}).Glossary(t.Context(), "Ich habe seit gestern Kopfschmerzen.", "de", "bs")
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Glossary() = %+v, %v; want %+v", got, err, want)
	}
	q := repo.glossaryQuery
	if q == nil || q.From != "de" || q.To != "bs" || q.Limit != 20 {
		t.Fatalf("query %+v; want de to bs, at most 20", q)
	}
	for _, c := range []string{"Kopfschmerzen", "Kopfschmerz", "gestern Kopfschmerzen", "seit gestern Kopfschmerzen", "Ich habe seit"} {
		if !slices.Contains(q.Candidates, c) {
			t.Errorf("candidates %q lack %q", q.Candidates, c)
		}
	}
	for _, c := range []string{"Kopfschmerzen.", "Ich habe seit gestern", "Kopfschme"} {
		if slices.Contains(q.Candidates, c) {
			t.Errorf("candidates %q hold %q", q.Candidates, c)
		}
	}
}

// A phrase's last word may carry up to three more letters than the term (a plural, a case ending, a Turkish
// suffix), as long as four letters of it are left: "ağrısı" finds "ağrı"; a word of two letters or fewer is no
// phrase of its own; punctuation around a word is not part of it, inside it is ("COVID-19").
func TestGlossary_CandidatesAllowAShortSuffixAndDropPunctuation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		text    string
		want    []string
		notWant []string
	}{
		{text: "Baş ağrısı var ve.", want: []string{"Baş", "Baş ağrısı", "ağrısı", "ağrı", "Baş ağrı", "Baş ağrısı var"}, notWant: []string{"ve", "ağr"}},
		{text: "«COVID-19», doktore?", want: []string{"COVID-19", "doktore", "doktor", "COVID-19 doktore"}},
		{text: "عندي صداع؟", want: []string{"صداع", "عندي صداع", "عندي"}, notWant: []string{"صداع؟"}},
	} {
		repo := &fakeRepo{}
		if _, err := newService(repo, &fakeLanguages{}).Glossary(t.Context(), tc.text, "tr", "bs"); err != nil {
			t.Fatal(err)
		}
		for _, c := range tc.want {
			if !slices.Contains(repo.glossaryQuery.Candidates, c) {
				t.Errorf("%q: candidates %q lack %q", tc.text, repo.glossaryQuery.Candidates, c)
			}
		}
		for _, c := range tc.notWant {
			if slices.Contains(repo.glossaryQuery.Candidates, c) {
				t.Errorf("%q: candidates %q hold %q", tc.text, repo.glossaryQuery.Candidates, c)
			}
		}
	}
}

// Only the first 200 words of a line are looked up, each candidate once.
func TestGlossary_BoundsTheCandidates(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	text := strings.Repeat("placeholder ", 300) + "Kopfschmerzen"
	if _, err := newService(repo, &fakeLanguages{}).Glossary(t.Context(), text, "de", "bs"); err != nil {
		t.Fatal(err)
	}
	c := repo.glossaryQuery.Candidates
	if slices.Contains(c, "Kopfschmerzen") {
		t.Errorf("a word after the 200th was looked up")
	}
	sorted := slices.Clone(c)
	slices.Sort(sorted)
	if len(slices.Compact(sorted)) != len(c) {
		t.Errorf("candidates repeat: %d of them, %d distinct", len(c), len(slices.Compact(sorted)))
	}
}

// A line with no word is no lookup; two languages that are not the terms' (or the same one twice) are refused;
// the repository's failure is returned without the line's text.
func TestGlossary_RefusesBadLanguagesAndReturnsTheRepositorysError(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	svc := newService(repo, &fakeLanguages{})
	if got, err := svc.Glossary(t.Context(), " ... ", "de", "bs"); err != nil || got != nil || repo.glossaryQuery != nil {
		t.Errorf("a line without words = %v, %v, looked up %v; want nothing", got, err, repo.glossaryQuery != nil)
	}
	for _, langs := range [][2]string{{"xx", "bs"}, {"de", "de-DE"}, {"bs", "bs"}} {
		if _, err := svc.Glossary(t.Context(), "Kopfschmerzen", langs[0], langs[1]); !errors.Is(err, terminology.ErrInvalidSearch) {
			t.Errorf("Glossary(%s to %s) = %v, want ErrInvalidSearch", langs[0], langs[1], err)
		}
	}
	repo.err = errors.New("the database is down")
	_, err := svc.Glossary(t.Context(), "Kopfschmerzen placeholder-marker", "de", "bs")
	if err == nil || strings.Contains(err.Error(), "marker") {
		t.Errorf("Glossary() = %v, want the repository's error without the text", err)
	}
}
