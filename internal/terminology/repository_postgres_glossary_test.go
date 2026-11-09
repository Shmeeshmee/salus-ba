package terminology_test

import (
	"reflect"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// The glossary finds a term by a label in the line's language that equals a candidate, case and accents aside,
// gives its preferred label in the target language (else the first), the most common term first, each term once
// by its longest label found, and leaves out a term without a label in the target language.
func TestGlossary_FindsTheTermsOfTheCandidatesInTheLinesLanguage(t *testing.T) {
	t.Parallel()
	repo := seedTerms(t)
	got, err := repo.Glossary(t.Context(), terminology.GlossaryQuery{
		From: "en", To: "bs", Limit: 20,
		Candidates: []string{"Stomach Ulcer", "ulcer", "GASTRIC ULCER", "Liver", "hepatitis", "fatty liver", "placeholder"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []terminology.GlossaryEntry{
		{Source: "hepatitis", Target: "upala jetre"},       // rank 200
		{Source: "liver", Target: "jetra"},                 // rank 100
		{Source: "stomach ulcer", Target: "čir na želucu"}, // rank 50, found by two labels of one length: the preferred
	}
	// fatty liver (rank 1000) has no Bosnian label.
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Glossary() = %+v,\nwant %+v", got, want)
	}
}

// Only labels in the line's language count, accents aside, at most Limit terms.
func TestGlossary_MatchesOnlyTheLinesLanguageAndKeepsTheLimit(t *testing.T) {
	t.Parallel()
	repo := seedTerms(t)
	got, err := repo.Glossary(t.Context(), terminology.GlossaryQuery{
		From: "bs", To: "en", Limit: 20, Candidates: []string{"hepatitis", "cir na zelucu", "JETRA"},
	})
	want := []terminology.GlossaryEntry{{Source: "jetra", Target: "liver"}, {Source: "čir na želucu", Target: "stomach ulcer"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Glossary() = %+v, %v;\nwant %+v", got, err, want)
	}
	got, err = repo.Glossary(t.Context(), terminology.GlossaryQuery{From: "bs", To: "en", Limit: 1, Candidates: []string{"cir na zelucu", "jetra"}})
	if err != nil || len(got) != 1 || got[0].Target != "liver" {
		t.Errorf("Glossary() with limit 1 = %+v, %v; want the liver only", got, err)
	}
}
