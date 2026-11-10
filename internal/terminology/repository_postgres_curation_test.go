package terminology_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// curation returns the curation repository on the owner's connection to a database with the synthetic terms.
func curation(t *testing.T) terminology.CurationRepository {
	t.Helper()
	db := pgtest.NewDatabase(t)
	insertTerms(t, db)
	return terminology.NewPostgresCurationRepository(db)
}

func termOf(t *testing.T, repo terminology.CurationRepository, id string) terminology.Term {
	t.Helper()
	term, err := repo.Term(t.Context(), id)
	if err != nil {
		t.Fatalf("Term(%s): %v", id, err)
	}
	return term
}

func TestTerm_ReturnsTheTermWithEveryLabel(t *testing.T) {
	t.Parallel()
	repo := curation(t)

	got := termOf(t, repo, ulcer)

	want := terminology.Term{
		ID: ulcer, Category: "diagnosis", WikidataID: "Q1065994", ICD10: "K25", ReviewStatus: "unreviewed", Rank: 50,
		Labels: []terminology.Label{
			{Language: "la", Text: "ulcus ventriculi", Preferred: true}, {Language: "bs", Text: "čir na želucu", Preferred: true},
			{Language: "en", Text: "stomach ulcer", Preferred: true}, {Language: "en", Text: "gastric ulcer"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Term(ulcer) = %+v,\nwant %+v", got, want)
	}
	for _, id := range []string{"00000000-0000-4000-8000-0000000000ff", "not-a-uuid"} {
		if _, err := repo.Term(t.Context(), id); !errors.Is(err, terminology.ErrTermNotFound) {
			t.Errorf("Term(%s) = %v, want ErrTermNotFound", id, err)
		}
	}
}

func TestCreateFromSeed_InsertsANewTermOnceByItsWikidataID(t *testing.T) {
	t.Parallel()
	repo := curation(t)
	seed := terminology.Term{
		Category: "symptom", WikidataID: "Q38933", ICD10: "R50.9", Rank: 120, ReviewStatus: "unreviewed",
		Labels: []terminology.Label{{Language: "la", Text: "febris", Preferred: true}, {Language: "en", Text: "fever", Preferred: true}, {Language: "en", Text: "pyrexia"}},
	}

	created, err := repo.CreateFromSeed(t.Context(), seed)
	if err != nil || !created {
		t.Fatalf("CreateFromSeed(fever) = %v, %v; want created", created, err)
	}
	again := seed
	again.Labels = []terminology.Label{{Language: "en", Text: "something else", Preferred: true}}
	if created, err := repo.CreateFromSeed(t.Context(), again); err != nil || created {
		t.Errorf("CreateFromSeed(fever) again = %v, %v; want not created", created, err)
	}

	terms, err := repo.Terms(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	var got *terminology.Term
	for i := range terms {
		if terms[i].WikidataID == "Q38933" {
			got = &terms[i]
		}
	}
	if got == nil {
		t.Fatal("the seed term is not in Terms")
	}
	seed.ID = got.ID
	seed.Labels = []terminology.Label{{Language: "la", Text: "febris", Preferred: true}, {Language: "en", Text: "fever", Preferred: true}, {Language: "en", Text: "pyrexia"}}
	if !reflect.DeepEqual(*got, seed) {
		t.Errorf("the seed term = %+v,\nwant %+v", *got, seed)
	}
}

func TestUpdateTerm_WritesTheColumnsAndReplacesTheLabels(t *testing.T) {
	t.Parallel()
	repo := curation(t)
	at := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)
	edit := terminology.Term{
		ReviewedAt: at, ID: ulcer, Category: "diagnosis", WikidataID: "Q1065994", ICD10: "K25.9", PCS: "44.4",
		ReviewStatus: "approved", Note: "Checked against the textbook.", Rank: 51,
		Labels: []terminology.Label{{Language: "la", Text: "ulcus ventriculi", Preferred: true}, {Language: "hr", Text: "vrijed želuca", Preferred: true}, {Language: "hr", Text: "čir na želucu"}},
	}

	if err := repo.UpdateTerm(t.Context(), edit, at); err != nil {
		t.Fatalf("UpdateTerm: %v", err)
	}

	got := termOf(t, repo, ulcer)
	got.ReviewedAt = got.ReviewedAt.UTC()
	edit.Labels = []terminology.Label{{Language: "la", Text: "ulcus ventriculi", Preferred: true}, {Language: "hr", Text: "vrijed želuca", Preferred: true}, {Language: "hr", Text: "čir na želucu"}}
	if !reflect.DeepEqual(got, edit) {
		t.Errorf("after UpdateTerm = %+v,\nwant %+v", got, edit)
	}
}

// reviewed_by is the reviewer's account, written and read back, and cleared (00037).
func TestUpdateTerm_WritesAndClearsTheReviewer(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)
	insertTerms(t, db)
	repo := terminology.NewPostgresCurationRepository(db)
	var reviewer string
	if err := db.QueryRow(t.Context(), `INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id::text`).Scan(new(string)); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(t.Context(), `INSERT INTO account (email, password_hash, role, organization_id)
		SELECT 'reviewer@example.test', 'h', 'practitioner', id FROM organization LIMIT 1 RETURNING id::text`).Scan(&reviewer); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	edit := termOf(t, repo, ulcer)
	edit.ReviewStatus, edit.ReviewedAt, edit.ReviewedBy = "approved", at, reviewer

	if err := repo.UpdateTerm(t.Context(), edit, at); err != nil {
		t.Fatalf("UpdateTerm: %v", err)
	}
	if got := termOf(t, repo, ulcer); got.ReviewedBy != reviewer {
		t.Errorf("reviewed_by = %q, want %q", got.ReviewedBy, reviewer)
	}
	edit.ReviewedBy = ""
	if err := repo.UpdateTerm(t.Context(), edit, at); err != nil {
		t.Fatalf("UpdateTerm: %v", err)
	}
	if got := termOf(t, repo, ulcer); got.ReviewedBy != "" {
		t.Errorf("reviewed_by = %q after clearing, want none", got.ReviewedBy)
	}
}

func TestUpdateTerm_RefusesAnUnknownIDOrAnotherTermsWikidataID(t *testing.T) {
	t.Parallel()
	repo := curation(t)
	at := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)

	missing := terminology.Term{ID: "00000000-0000-4000-8000-0000000000ff", Category: "diagnosis", ReviewStatus: "unreviewed"}
	if err := repo.UpdateTerm(t.Context(), missing, at); !errors.Is(err, terminology.ErrTermNotFound) {
		t.Errorf("UpdateTerm(unknown id) = %v, want ErrTermNotFound", err)
	}
	taken := termOf(t, repo, ulcer)
	taken.WikidataID = "Q9368" // the liver's
	if err := repo.UpdateTerm(t.Context(), taken, at); !errors.Is(err, terminology.ErrDuplicateWikidata) {
		t.Errorf("UpdateTerm(the liver's Wikidata id) = %v, want ErrDuplicateWikidata", err)
	}
}

func TestTerms_ByCategoryAndRankFilteredByStatus(t *testing.T) {
	t.Parallel()
	repo := curation(t)
	ids := func(status string) []string {
		t.Helper()
		terms, err := repo.Terms(t.Context(), status)
		if err != nil {
			t.Fatalf("Terms(%q): %v", status, err)
		}
		out := []string{}
		for _, term := range terms {
			out = append(out, term.ID)
		}
		return out
	}

	if got, want := ids(""), []string{liver, fattyLvr, hepatitis, ulcer}; !reflect.DeepEqual(got, want) {
		t.Errorf("Terms() = %v, want %v", got, want)
	}
	if got, want := ids("unreviewed"), []string{fattyLvr, hepatitis, ulcer}; !reflect.DeepEqual(got, want) {
		t.Errorf("Terms(draft) = %v, want %v", got, want)
	}
	if got, want := ids("approved"), []string{liver}; !reflect.DeepEqual(got, want) {
		t.Errorf("Terms(reviewed) = %v, want %v", got, want)
	}
}

func TestCountByCategory(t *testing.T) {
	t.Parallel()
	repo := curation(t)

	got, err := repo.CountByCategory(t.Context())

	want := []terminology.CategoryCount{{Category: "anatomy", Terms: 1}, {Category: "diagnosis", Terms: 3}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("CountByCategory() = %v, %v; want %v", got, err, want)
	}
}
