package terminology_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// explainedTerms are the synthetic terms with explanations: the liver in Bosnian (approved) and English
// (unreviewed), the stomach ulcer in English only, the hepatitis in Bosnian only and needing changes; the fatty liver
// has none.
func explainedTerms(t *testing.T) (*pgxpool.Pool, terminology.Repository) {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	insertTerms(t, owner)
	_, err := owner.Exec(t.Context(), `
		INSERT INTO term_explanation (term_id, language, text, review_status, reviewed_at) VALUES
			('`+liver+`', 'bs', 'Placeholder: organ u trbuhu.', 'approved', '2026-11-10T10:00:00Z'),
			('`+liver+`', 'en', 'Placeholder: an organ in the belly.', 'unreviewed', NULL),
			('`+ulcer+`', 'en', 'Placeholder: a sore in the stomach.', 'unreviewed', NULL),
			('`+hepatitis+`', 'bs', 'Placeholder: upala.', 'needs_changes', '2026-11-10T10:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	return owner, terminology.NewPostgresRepository(app)
}

// Tag finds a candidate by a label in its own language, case and accents aside, only for a term with an explanation
// in any language (the fatty liver has none), and gives each found candidate once.
func TestTag_FindsTheCandidatesThatNameAnExplainedTerm(t *testing.T) {
	t.Parallel()
	_, repo := explainedTerms(t)
	got, err := repo.Tag(t.Context(), []terminology.Candidate{
		{Language: "bs", Phrase: "JETRA"}, {Language: "bs", Phrase: "cir na zelucu"}, {Language: "en", Phrase: "jetra"},
		{Language: "en", Phrase: "Fatty liver"}, {Language: "en", Phrase: "Gastric Ulcer"}, {Language: "en", Phrase: "placeholder"},
		{Language: "bs", Phrase: "upala jetre"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[terminology.Candidate]string{
		{Language: "bs", Phrase: "JETRA"}:         liver,
		{Language: "bs", Phrase: "cir na zelucu"}: ulcer,
		{Language: "en", Phrase: "Gastric Ulcer"}: ulcer,
		{Language: "bs", Phrase: "upala jetre"}:   hepatitis, // its explanation needs changes, but it has one
	}
	found := map[terminology.Candidate]string{}
	for _, c := range got {
		found[terminology.Candidate{Language: c.Language, Phrase: c.Phrase}] = c.TermID
	}
	if !reflect.DeepEqual(found, want) || len(got) != len(want) {
		t.Errorf("Tag() = %+v,\nwant %+v", got, want)
	}
}

// The explanations in a language are each term's there, else its English one, without one that needs changes, by
// term id.
func TestExplanations_InTheLanguageElseEnglishWithoutOneThatNeedsChanges(t *testing.T) {
	t.Parallel()
	_, repo := explainedTerms(t)
	got, err := repo.Explanations(t.Context(), "bs")
	if err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	for _, e := range got {
		texts[e.TermID] = e.Language + " " + e.Review.Status + " " + e.Text
	}
	want := map[string]string{
		liver: "bs approved Placeholder: organ u trbuhu.",
		ulcer: "en unreviewed Placeholder: a sore in the stomach.",
	}
	if !reflect.DeepEqual(texts, want) || len(got) != 2 {
		t.Errorf("Explanations(bs) = %+v,\nwant %+v", texts, want)
	}
	if got, err := repo.Explanations(t.Context(), "de"); err != nil || len(got) != 2 || got[0].Language != "en" {
		t.Errorf("Explanations(de) = %+v, %v; want the two English ones", got, err)
	}
}

// The explanations' import finds a term by its Wikidata id, and ErrTermNotFound for one no term has.
func TestTermIDByWikidata_FindsTheTerm(t *testing.T) {
	t.Parallel()
	owner, _ := pgtest.NewAppDatabase(t)
	insertTerms(t, owner)
	repo := terminology.NewPostgresCurationRepository(owner)
	if id, err := repo.TermIDByWikidata(t.Context(), "Q9368"); err != nil || id != liver {
		t.Errorf("TermIDByWikidata(Q9368) = %q, %v; want the liver", id, err)
	}
	if _, err := repo.TermIDByWikidata(t.Context(), "Q1"); !errors.Is(err, terminology.ErrTermNotFound) {
		t.Errorf("TermIDByWikidata(Q1) = %v, want ErrTermNotFound", err)
	}
}
