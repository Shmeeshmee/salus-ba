package terminology_test

import (
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// Synthetic terms. The labels are ordinary medical words; the ranks are made up so each ranking rule is visible.
const (
	liver     = "00000000-0000-4000-8000-000000000001" // anatomy, rank 100
	hepatitis = "00000000-0000-4000-8000-000000000002" // diagnosis, rank 200
	ulcer     = "00000000-0000-4000-8000-000000000003" // diagnosis, rank 50
	fattyLvr  = "00000000-0000-4000-8000-000000000004" // diagnosis, rank 1000
)

// seedTerms writes the synthetic terms as the owner and returns the repository on the app role.
func seedTerms(t *testing.T) terminology.Repository {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	insertTerms(t, owner)
	return terminology.NewPostgresRepository(app)
}

// insertTerms writes the synthetic terms on db.
func insertTerms(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	_, err := db.Exec(t.Context(), `
		INSERT INTO term (id, category, rank, wikidata_id, ta98, icd10, review_status, reviewed_at) VALUES
			('`+liver+`', 'anatomy', 100, 'Q9368', 'A05.8.01.001', NULL, 'approved', '2026-11-05T10:00:00Z'),
			('`+hepatitis+`', 'diagnosis', 200, 'Q131742', NULL, 'K75.9', 'unreviewed', NULL),
			('`+ulcer+`', 'diagnosis', 50, 'Q1065994', NULL, 'K25', 'unreviewed', NULL),
			('`+fattyLvr+`', 'diagnosis', 1000, NULL, NULL, 'K76.0', 'unreviewed', NULL);
		INSERT INTO term_label (term_id, language, label, preferred) VALUES
			('`+liver+`', 'la', 'hepar', true), ('`+liver+`', 'la', 'iecur', false),
			('`+liver+`', 'bs', 'jetra', true), ('`+liver+`', 'hr', 'jetra', true),
			('`+liver+`', 'sr-Latn', 'jetra', true), ('`+liver+`', 'sr-Cyrl', 'јетра', true),
			('`+liver+`', 'en', 'liver', true),
			('`+hepatitis+`', 'la', 'hepatitis', true), ('`+hepatitis+`', 'bs', 'upala jetre', true),
			('`+hepatitis+`', 'en', 'hepatitis', true),
			('`+ulcer+`', 'la', 'ulcus ventriculi', true), ('`+ulcer+`', 'bs', 'čir na želucu', true),
			('`+ulcer+`', 'en', 'stomach ulcer', true), ('`+ulcer+`', 'en', 'gastric ulcer', false),
			('`+fattyLvr+`', 'la', 'hepar adiposum', true), ('`+fattyLvr+`', 'en', 'fatty liver', true)`)
	if err != nil {
		t.Fatal(err)
	}
}

func search(t *testing.T, repo terminology.Repository, s terminology.Search) []string {
	t.Helper()
	if s.Limit == 0 {
		s.Limit = 25
	}
	if s.Languages == nil {
		s.Languages = []string{"la", "bs", "en"}
	}
	if s.UILanguage == "" {
		s.UILanguage = "bs"
	}
	terms, err := repo.Search(t.Context(), s)
	if err != nil {
		t.Fatalf("Search(%q): %v", s.Text, err)
	}
	ids := []string{}
	for _, term := range terms {
		ids = append(ids, term.ID)
	}
	return ids
}

func TestSearch_Ranking(t *testing.T) {
	t.Parallel()
	repo := seedTerms(t)
	for _, c := range []struct {
		name, text, category string
		want                 []string
	}{
		{"an exact label, then a prefix of a more common term, then a trigram match", "hepar", "", []string{liver, fattyLvr, hepatitis}},
		{"a prefix before a trigram match of a more common term", "jetr", "", []string{liver, hepatitis}},
		{"equal matches by rank", "hep", "", []string{fattyLvr, hepatitis, liver}},
		{"a misspelling within trigram distance", "hepatits", "", []string{hepatitis}},
		{"accents and case do not matter", "CIR", "", []string{ulcer}},
		{"a word inside a label", "ventricul", "", []string{ulcer}},
		{"another category's match filtered out", "liver", "anatomy", []string{liver}},
		{"the category filter", "hep", "diagnosis", []string{fattyLvr, hepatitis}},
		{"one letter: the labels that start with it, by rank", "u", "", []string{hepatitis, ulcer}},
		{"two letters: a prefix, found by word similarity", "ul", "", []string{ulcer}},
		{"LIKE wildcards are plain characters", "%", "", []string{}},
		{"no match", "zzzz", "", []string{}},
	} {
		got := search(t, repo, terminology.Search{Text: c.text, Category: c.category})
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Search(%q, %q) = %v, want %v", c.name, c.text, c.category, got, c.want)
		}
	}
}

// Synthetic terms for the language order: one label each, in one language, all prefixes of "zeta" (equal word
// similarity), with ranks against the language order; and an exact German label of the lowest rank.
const (
	zetaDe    = "00000000-0000-4000-8000-000000000011" // de prefix, rank 900
	zetaEn    = "00000000-0000-4000-8000-000000000012" // en prefix, rank 500
	zetaLa    = "00000000-0000-4000-8000-000000000013" // la prefix, rank 100
	zetaBs    = "00000000-0000-4000-8000-000000000014" // bs prefix, rank 50
	zetaExact = "00000000-0000-4000-8000-000000000015" // de exact, rank 1
)

// Within each match type, a label in the caller's UI language or in Latin ranks first, then English, then the
// other languages; the match type still comes first (owner's answer, 2026-11-08; SPEC §3 "Search").
func TestSearch_RanksByLanguageWithinAMatchType(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, err := owner.Exec(t.Context(), `
		INSERT INTO term (id, category, rank) VALUES
			('`+zetaDe+`', 'anatomy', 900), ('`+zetaEn+`', 'anatomy', 500), ('`+zetaLa+`', 'anatomy', 100),
			('`+zetaBs+`', 'anatomy', 50), ('`+zetaExact+`', 'anatomy', 1);
		INSERT INTO term_label (term_id, language, label, preferred) VALUES
			('`+zetaDe+`', 'de', 'zetamin', true), ('`+zetaEn+`', 'en', 'zetavor', true),
			('`+zetaLa+`', 'la', 'zetarius', true), ('`+zetaBs+`', 'bs', 'zetanik', true),
			('`+zetaExact+`', 'de', 'zeta', true)`)
	if err != nil {
		t.Fatal(err)
	}
	repo := terminology.NewPostgresRepository(app)
	for _, c := range []struct {
		ui   string
		want []string
	}{
		{"bs", []string{zetaExact, zetaLa, zetaBs, zetaEn, zetaDe}},
		{"de", []string{zetaExact, zetaDe, zetaLa, zetaEn, zetaBs}},
		{"en", []string{zetaExact, zetaEn, zetaLa, zetaDe, zetaBs}},
	} {
		got := search(t, repo, terminology.Search{Text: "zeta", UILanguage: c.ui})
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Search(zeta) in UI language %s = %v, want %v", c.ui, got, c.want)
		}
	}
}

func TestSearch_Limit(t *testing.T) {
	t.Parallel()
	repo := seedTerms(t)
	if got := search(t, repo, terminology.Search{Text: "hep", Limit: 2}); !reflect.DeepEqual(got, []string{fattyLvr, hepatitis}) {
		t.Errorf("Search(hep, limit 2) = %v, want the two best", got)
	}
}

// A match carries its codes, review and the labels in the requested languages only: preferred first, then the
// synonyms, by language in the enum's order.
func TestSearch_ReturnsTheTermWithItsLabelsInTheRequestedLanguages(t *testing.T) {
	t.Parallel()
	repo := seedTerms(t)
	terms, err := repo.Search(t.Context(), terminology.Search{Text: "jetra", Languages: []string{"sr-Cyrl", "la"}, Limit: 1})
	if err != nil || len(terms) != 1 {
		t.Fatalf("Search(jetra) = %v, %v; want one term", terms, err)
	}
	got := terms[0]
	if got.ReviewedAt = got.ReviewedAt.UTC(); got.ReviewedAt.Format("2006-01-02T15:04") != "2026-11-05T10:00" {
		t.Errorf("ReviewedAt = %v", got.ReviewedAt)
	}
	want := terminology.Term{
		ReviewedAt: got.ReviewedAt, ID: liver, Category: "anatomy", WikidataID: "Q9368", TA98: "A05.8.01.001",
		ReviewStatus: "approved", Rank: 100,
		Labels: []terminology.Label{
			{Language: "la", Text: "hepar", Preferred: true}, {Language: "la", Text: "iecur"},
			{Language: "sr-Cyrl", Text: "јетра", Preferred: true},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Search(jetra) = %+v,\nwant %+v", got, want)
	}

	terms, err = repo.Search(t.Context(), terminology.Search{Text: "ulcus", Languages: []string{"en"}, Limit: 1})
	if err != nil || len(terms) != 1 {
		t.Fatalf("Search(ulcus) = %v, %v; want one term", terms, err)
	}
	if got := terms[0]; got.ICD10 != "K25" || got.ReviewStatus != "unreviewed" || !got.ReviewedAt.IsZero() ||
		!reflect.DeepEqual(got.Labels, []terminology.Label{{Language: "en", Text: "stomach ulcer", Preferred: true}, {Language: "en", Text: "gastric ulcer"}}) {
		t.Errorf("Search(ulcus) = %+v", got)
	}
}
