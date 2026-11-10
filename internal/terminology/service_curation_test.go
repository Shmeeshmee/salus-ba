package terminology_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// fakeCuration keeps the terms in memory by id and counts the transactions and writes.
type fakeCuration struct {
	err          error // returned by every method when set
	terms        map[string]terminology.Term
	gotStat      string
	translations []string // the SetTranslationsStatus calls, "<id> <status> <at>"
	updates      int
	txs          int
	// explanations are the stored explanations by "<term id> <language>" (ImportExplanations)
	explanations map[string]terminology.Explanation
}

func (f *fakeCuration) TermIDByWikidata(_ context.Context, wikidataID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	for id, t := range f.terms {
		if t.WikidataID == wikidataID {
			return id, nil
		}
	}
	return "", terminology.ErrTermNotFound
}

func (f *fakeCuration) LockEntry(_ context.Context, termID, language string) (terminology.EntryState, error) {
	if e, ok := f.explanations[termID+" "+language]; ok {
		return terminology.EntryState{Explanation: &e}, f.err
	}
	return terminology.EntryState{}, f.err
}

func (f *fakeCuration) PutExplanation(_ context.Context, e terminology.Explanation) error {
	if f.explanations == nil {
		f.explanations = map[string]terminology.Explanation{}
	}
	f.explanations[e.TermID+" "+e.Language] = e
	return f.err
}

func newFakeCuration(terms ...terminology.Term) *fakeCuration {
	f := &fakeCuration{terms: map[string]terminology.Term{}}
	for _, t := range terms {
		f.terms[t.ID] = t
	}
	return f
}

func (f *fakeCuration) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.txs++
	return fn(ctx)
}

func (f *fakeCuration) CreateFromSeed(_ context.Context, t terminology.Term) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for _, have := range f.terms {
		if have.WikidataID == t.WikidataID {
			return false, nil
		}
	}
	t.ID = "new-" + t.WikidataID
	f.terms[t.ID] = t
	return true, nil
}

func (f *fakeCuration) Term(_ context.Context, id string) (terminology.Term, error) {
	if f.err != nil {
		return terminology.Term{}, f.err
	}
	t, ok := f.terms[id]
	if !ok {
		return terminology.Term{}, terminology.ErrTermNotFound
	}
	return t, nil
}

func (f *fakeCuration) UpdateTerm(_ context.Context, t terminology.Term, at time.Time) error {
	for _, have := range f.terms {
		if have.ID != t.ID && t.WikidataID != "" && have.WikidataID == t.WikidataID {
			return terminology.ErrDuplicateWikidata
		}
	}
	if at.IsZero() {
		return errors.New("no updated_at")
	}
	f.updates++
	f.terms[t.ID] = t
	return nil
}

func (f *fakeCuration) SetTranslationsStatus(_ context.Context, id, status string, at time.Time) error {
	f.translations = append(f.translations, id+" "+status+" "+at.Format(time.RFC3339))
	return f.err
}

func (f *fakeCuration) Terms(_ context.Context, status string) ([]terminology.Term, error) {
	f.gotStat = status
	return []terminology.Term{{ID: "t1"}}, f.err
}

func (f *fakeCuration) CountByCategory(context.Context) ([]terminology.CategoryCount, error) {
	return []terminology.CategoryCount{{Category: "anatomy", Terms: 1}}, f.err
}

var now = time.Date(2026, 11, 8, 9, 30, 0, 0, time.UTC)

func newCurator(f *fakeCuration) *terminology.Curator {
	return terminology.NewCurator(f, f, clock.Fixed{T: now})
}

func labels(pairs ...string) []terminology.Label {
	var out []terminology.Label
	seen := map[string]bool{}
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, terminology.Label{Language: pairs[i], Text: pairs[i+1], Preferred: !seen[pairs[i]]})
		seen[pairs[i]] = true
	}
	return out
}

func row(line int, t terminology.Term) terminology.ImportRow {
	return terminology.ImportRow{Term: t, Line: line}
}

func reasons(r terminology.ImportReport) map[int]string {
	out := map[int]string{}
	for _, e := range r.Refused {
		out[e.Line] = e.Reason
	}
	return out
}

func TestImportSeed_CreatesNewTermsAsDraftsAndKeepsTheOnesThere(t *testing.T) {
	t.Parallel()
	f := newFakeCuration(terminology.Term{ID: "liver", WikidataID: "Q9368", Category: "anatomy", ReviewStatus: "approved"})
	fever := terminology.Term{
		Category: "symptom", WikidataID: "Q38933", ICD10: " R50.9 ", Rank: 120,
		Labels: labels("la", "  febris ", "en", "fever", "en", "high   temperature", "en", "fever"),
	}

	got, err := newCurator(f).ImportSeed(t.Context(), []terminology.ImportRow{
		row(1, fever),
		row(2, terminology.Term{Category: "anatomy", WikidataID: "Q9368", Labels: labels("en", "liver, changed")}),
	})

	if err != nil || got.Created != 1 || got.Unchanged != 1 || got.Updated != 0 || len(got.Refused) != 0 {
		t.Fatalf("ImportSeed = %+v, %v; want one created, one unchanged", got, err)
	}
	want := terminology.Term{
		ID: "new-Q38933", Category: "symptom", WikidataID: "Q38933", ICD10: "R50.9", Rank: 120, ReviewStatus: "unreviewed",
		Labels: labels("la", "febris", "en", "fever", "en", "high temperature"),
	}
	if !reflect.DeepEqual(f.terms["new-Q38933"], want) {
		t.Errorf("created %+v,\nwant %+v", f.terms["new-Q38933"], want)
	}
	if f.terms["liver"].ReviewStatus != "approved" || f.txs != 2 {
		t.Errorf("the existing term was changed or the rows did not run in their own transactions (%d)", f.txs)
	}
}

func TestImportSeed_RefusesAnInvalidRowAndAppliesTheRest(t *testing.T) {
	t.Parallel()
	f := newFakeCuration()
	ok := terminology.Term{Category: "drug", WikidataID: "Q18216", Labels: labels("en", "aspirin")}
	long := strings.Repeat("a", 301)
	rows := []terminology.ImportRow{
		row(1, terminology.Term{Category: "drug", Labels: labels("en", "no id")}),
		row(2, terminology.Term{Category: "drug", WikidataID: "P31", Labels: labels("en", "x")}),
		row(3, terminology.Term{Category: "organ", WikidataID: "Q1", Labels: labels("en", "x")}),
		row(4, terminology.Term{Category: "drug", WikidataID: "Q2", Labels: labels("xx", "x")}),
		row(5, terminology.Term{Category: "drug", WikidataID: "Q3", Labels: labels("en", long)}),
		row(6, terminology.Term{Category: "drug", WikidataID: "Q4", Rank: -1, Labels: labels("en", "x")}),
		row(7, terminology.Term{Category: "drug", WikidataID: "Q5", Labels: []terminology.Label{{Language: "en", Text: "only a synonym"}}}),
		row(8, ok),
		row(9, terminology.Term{Category: "drug", WikidataID: "Q6", Labels: labels("en", "a | b")}),
	}

	got, err := newCurator(f).ImportSeed(t.Context(), rows)

	want := map[int]string{
		1: "wikidata: missing", 2: "wikidata: not a Wikidata item id", 3: "category: unknown",
		4: "labels: unknown language", 5: "en: a label longer than 300 characters", 6: "rank: negative",
		7: "en: synonyms without a preferred label", 9: "en: a label with |",
	}
	if err != nil || got.Created != 1 || !reflect.DeepEqual(reasons(got), want) {
		t.Errorf("ImportSeed = %+v, %v;\nwant one created and refused %v", got, err, want)
	}
}

func TestImportSeed_StopsOnADatabaseError(t *testing.T) {
	t.Parallel()
	f := newFakeCuration()
	f.err = errors.New("connection lost")

	_, err := newCurator(f).ImportSeed(t.Context(), []terminology.ImportRow{row(1, terminology.Term{Category: "drug", WikidataID: "Q1", Labels: labels("en", "x")})})

	if err == nil {
		t.Error("ImportSeed with the database gone: error = nil")
	}
}

func TestImportCurated(t *testing.T) {
	t.Parallel()
	reviewedOn := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	stored := []terminology.Term{
		{ID: "liver", Category: "anatomy", WikidataID: "Q9368", TA98: "A05.8.01.001", Rank: 100, ReviewStatus: "unreviewed", Labels: labels("la", "hepar", "en", "liver")},
		{ID: "ulcer", ReviewedAt: reviewedOn, Category: "diagnosis", WikidataID: "Q1065994", ICD10: "K25", Rank: 50, ReviewStatus: "approved", Labels: labels("en", "stomach ulcer", "en", "gastric ulcer")},
		{ID: "fever", ReviewedAt: reviewedOn, Category: "symptom", WikidataID: "Q38933", Rank: 120, ReviewStatus: "approved", Labels: labels("en", "fever")},
		{ID: "cough", ReviewedAt: reviewedOn, Category: "symptom", WikidataID: "Q35805", Rank: 110, ReviewStatus: "approved", Labels: labels("en", "cough")},
		{ID: "rash", Category: "symptom", WikidataID: "Q1139839", Rank: 90, ReviewStatus: "unreviewed", Labels: labels("en", "rash")},
	}
	f := newFakeCuration(stored...)
	edited := func(t terminology.Term, edit func(*terminology.Term)) terminology.Term {
		edit(&t)
		return t
	}
	rows := []terminology.ImportRow{
		// draft to reviewed, Latin filled: the date is now
		row(2, edited(stored[0], func(t *terminology.Term) {
			t.ReviewStatus, t.Labels = "approved", labels("la", "hepar", "la", "iecur", "en", "liver")
		})),
		// the same content, synonyms in another order: unchanged, its date kept
		row(3, edited(stored[1], func(t *terminology.Term) {
			t.ReviewedAt, t.Labels = time.Time{}, labels("en", "stomach ulcer", "en", "gastric ulcer")
		})),
		// reviewed back to draft: its last date kept
		row(4, edited(stored[2], func(t *terminology.Term) {
			t.ReviewedAt, t.ReviewStatus, t.Note = time.Time{}, "unreviewed", "Check the Latin."
		})),
		// a label fixed on a reviewed term: still reviewed, its date kept
		row(5, edited(stored[3], func(t *terminology.Term) {
			t.ReviewedAt, t.Labels = time.Time{}, labels("en", "cough", "hr", "kašalj")
		})),
		row(6, terminology.Term{Category: "symptom", ReviewStatus: "unreviewed", Labels: labels("en", "new")}),
		row(7, edited(stored[0], func(t *terminology.Term) { t.ID = "missing" })),
		row(8, edited(stored[1], func(t *terminology.Term) { t.ReviewStatus = "done" })),
		row(9, edited(stored[1], func(t *terminology.Term) {})),
		row(10, edited(stored[4], func(t *terminology.Term) { t.WikidataID = "Q9368" })),
	}

	got, err := newCurator(f).ImportCurated(t.Context(), rows)

	wantRefused := map[int]string{
		6: "id: missing; new terms come from the seed", 7: "id: no such term", 8: "review_status: not unreviewed, approved or needs_changes",
		9: "id: repeated in the file", 10: "wikidata: another term has it",
	}
	if err != nil || got.Updated != 3 || got.Unchanged != 1 || got.Created != 0 || !reflect.DeepEqual(reasons(got), wantRefused) {
		t.Fatalf("ImportCurated = %+v, %v;\nwant 3 updated, 1 unchanged, refused %v", got, err, wantRefused)
	}
	for _, c := range []struct {
		at         time.Time
		id, status string
	}{{now, "liver", "approved"}, {reviewedOn, "ulcer", "approved"}, {reviewedOn, "fever", "unreviewed"}, {reviewedOn, "cough", "approved"}} {
		if term := f.terms[c.id]; term.ReviewStatus != c.status || !term.ReviewedAt.Equal(c.at) {
			t.Errorf("%s: %s reviewed at %v, want %s at %v", c.id, term.ReviewStatus, term.ReviewedAt, c.status, c.at)
		}
	}
	if got := f.terms["liver"].Labels; !reflect.DeepEqual(got, labels("la", "hepar", "la", "iecur", "en", "liver")) {
		t.Errorf("liver's labels = %v, want them replaced", got)
	}
	if f.terms["fever"].Note != "Check the Latin." || f.updates != 3 {
		t.Errorf("note %q, %d updates; want the note and 3 updates", f.terms["fever"].Note, f.updates)
	}
	// The file has one status per term: a status it changes is every translation's too, so the review panel's
	// queue and the term's status agree (docs/terminology/SPEC.md §3 "Review").
	if want := []string{"liver approved " + now.Format(time.RFC3339), "fever unreviewed " + now.Format(time.RFC3339)}; !reflect.DeepEqual(f.translations, want) {
		t.Errorf("translations' statuses set %v, want %v (only where the status changed)", f.translations, want)
	}
}

// needs_changes is a review like approved: the date is now. A status the curator's file changes names no reviewer,
// so an in-app reviewer's account is cleared; a row that keeps the status keeps it (00037).
func TestImportCurated_NeedsChangesAndTheReviewer(t *testing.T) {
	t.Parallel()
	reviewedOn := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	stored := []terminology.Term{
		{ID: "liver", ReviewedAt: reviewedOn, ReviewedBy: "acc-1", Category: "anatomy", WikidataID: "Q9368", Rank: 100, ReviewStatus: "approved", Labels: labels("en", "liver")},
		{ID: "fever", ReviewedAt: reviewedOn, ReviewedBy: "acc-1", Category: "symptom", WikidataID: "Q38933", Rank: 120, ReviewStatus: "approved", Labels: labels("en", "fever")},
	}
	f := newFakeCuration(stored...)
	flagged, kept := stored[0], stored[1]
	flagged.ReviewedAt, flagged.ReviewedBy, flagged.ReviewStatus, flagged.Note = time.Time{}, "", "needs_changes", "Check the Latin."
	kept.ReviewedAt, kept.ReviewedBy, kept.Labels = time.Time{}, "", labels("en", "fever", "hr", "vrućica")

	got, err := newCurator(f).ImportCurated(t.Context(), []terminology.ImportRow{row(2, flagged), row(3, kept)})

	if err != nil || got.Updated != 2 || len(got.Refused) != 0 {
		t.Fatalf("ImportCurated = %+v, %v; want 2 updated", got, err)
	}
	if term := f.terms["liver"]; term.ReviewStatus != "needs_changes" || !term.ReviewedAt.Equal(now) || term.ReviewedBy != "" {
		t.Errorf("liver: %s at %v by %q, want needs_changes at %v by no one", term.ReviewStatus, term.ReviewedAt, term.ReviewedBy, now)
	}
	if term := f.terms["fever"]; term.ReviewStatus != "approved" || !term.ReviewedAt.Equal(reviewedOn) || term.ReviewedBy != "acc-1" {
		t.Errorf("fever: %s at %v by %q, want approved at %v by acc-1", term.ReviewStatus, term.ReviewedAt, term.ReviewedBy, reviewedOn)
	}
}

func TestExport(t *testing.T) {
	t.Parallel()
	f := newFakeCuration()
	c := newCurator(f)

	for _, status := range []string{"", "unreviewed", "approved", "needs_changes"} {
		if terms, err := c.Export(t.Context(), status); err != nil || len(terms) != 1 || f.gotStat != status {
			t.Errorf("Export(%q) = %v, %v; asked %q", status, terms, err, f.gotStat)
		}
	}
	if _, err := c.Export(t.Context(), "done"); !errors.Is(err, terminology.ErrInvalidStatus) {
		t.Errorf("Export(done) = %v, want ErrInvalidStatus", err)
	}
	if counts, err := c.Counts(t.Context()); err != nil || len(counts) != 1 {
		t.Errorf("Counts() = %v, %v", counts, err)
	}
}
