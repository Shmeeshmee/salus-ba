package terminology_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

const reviewerID = "00000000-0000-4000-8000-0000000000a1"

var reviewedAt = time.Date(2026, 11, 9, 14, 0, 0, 0, time.UTC)

// reviewDatabase is a database with the synthetic terms and, in Bosnian: the liver's labels and explanation
// approved, hepatitis's labels needing changes, the ulcer's labels unreviewed, and no Bosnian label for fatty
// liver. It returns the owner's pool and the review repository on the app role.
func reviewDatabase(t *testing.T) (*pgxpool.Pool, terminology.ReviewRepository) {
	owner, _, repo := reviewDatabases(t)
	return owner, repo
}

// reviewDatabases is reviewDatabase with the app role's pool too.
func reviewDatabases(t *testing.T) (owner, app *pgxpool.Pool, repo terminology.ReviewRepository) {
	t.Helper()
	owner, app = pgtest.NewAppDatabase(t)
	insertTerms(t, owner)
	_, err := owner.Exec(t.Context(), `
		INSERT INTO account (id, email, password_hash, role) VALUES ('`+reviewerID+`', 'reviewer@example.test', 'x', 'patient');
		INSERT INTO term_translation (term_id, language, review_status, reviewed_at, reviewed_by, version_id, note) VALUES
			('`+liver+`', 'bs', 'approved', '2026-11-09T14:00:00Z', '`+reviewerID+`', 3, NULL),
			('`+liver+`', 'la', 'approved', '2026-11-09T14:00:00Z', NULL, 1, NULL),
			('`+hepatitis+`', 'bs', 'needs_changes', '2026-11-09T14:00:00Z', '`+reviewerID+`', 2, 'Use the standard term.'),
			('`+ulcer+`', 'bs', 'unreviewed', NULL, NULL, 1, NULL);
		INSERT INTO term_explanation (term_id, language, text, review_status, reviewed_at, version_id) VALUES
			('`+liver+`', 'bs', 'Jetra je organ u trbuhu.', 'approved', '2026-11-09T14:00:00Z', 2)`)
	if err != nil {
		t.Fatal(err)
	}
	return owner, app, terminology.NewPostgresReviewRepository(app)
}

func queueIDs(t *testing.T, repo terminology.ReviewRepository, f terminology.QueueFilter) ([]string, []terminology.QueueRow) {
	t.Helper()
	rows, err := repo.Queue(t.Context(), f)
	if err != nil {
		t.Fatalf("Queue(%+v): %v", f, err)
	}
	ids := []string{}
	for _, r := range rows {
		ids = append(ids, r.Term.ID+" "+r.Status)
	}
	return ids, rows
}

func TestQueue_UnreviewedFirstThenNeedsChangesThenApprovedEachByRank(t *testing.T) {
	t.Parallel()
	_, repo := reviewDatabase(t)

	all, rows := queueIDs(t, repo, terminology.QueueFilter{Language: "bs", Limit: 10})
	want := []string{fattyLvr + " unreviewed", ulcer + " unreviewed", hepatitis + " needs_changes", liver + " approved"}
	if !reflect.DeepEqual(all, want) {
		t.Fatalf("Queue(bs) = %v,\nwant %v", all, want)
	}
	first, _ := queueIDs(t, repo, terminology.QueueFilter{Language: "bs", Limit: 2})
	next, _ := queueIDs(t, repo, terminology.QueueFilter{Language: "bs", Limit: 2, AfterStatus: "unreviewed", AfterRank: 50, AfterID: ulcer})
	if !reflect.DeepEqual(append(first, next...), want) {
		t.Errorf("two pages = %v + %v, want %v", first, next, want)
	}
	for status, ids := range map[string][]string{
		"approved": {liver + " approved"}, "needs_changes": {hepatitis + " needs_changes"},
		"unreviewed": {fattyLvr + " unreviewed", ulcer + " unreviewed"},
	} {
		if got, _ := queueIDs(t, repo, terminology.QueueFilter{Language: "bs", Status: status, Limit: 10}); !reflect.DeepEqual(got, ids) {
			t.Errorf("Queue(bs, %s) = %v, want %v", status, got, ids)
		}
	}

	wantLiver := terminology.QueueRow{
		Status: "approved",
		Term: terminology.Term{
			ID: liver, ReviewedAt: time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC), Category: "anatomy", WikidataID: "Q9368",
			TA98: "A05.8.01.001", ReviewStatus: "approved", Rank: 100,
			Labels: []terminology.Label{
				{Language: "la", Text: "hepar", Preferred: true}, {Language: "la", Text: "iecur"},
				{Language: "bs", Text: "jetra", Preferred: true}, {Language: "en", Text: "liver", Preferred: true},
			},
		},
		Translation: &terminology.Translation{TermID: liver, Language: "bs", VersionID: 3, Review: terminology.Review{Status: "approved", ReviewedAt: reviewedAt, ReviewedBy: reviewerID}},
		Explanation: &terminology.Explanation{TermID: liver, Language: "bs", Text: "Jetra je organ u trbuhu.", VersionID: 2, Review: terminology.Review{Status: "approved", ReviewedAt: reviewedAt}},
	}
	got := rows[3]
	if got.Translation != nil {
		got.Translation.UpdatedAt = time.Time{}
	}
	if got.Explanation != nil {
		got.Explanation.UpdatedAt = time.Time{}
	}
	if !reflect.DeepEqual(got, wantLiver) {
		t.Errorf("the liver's entry = %+v\n%+v %+v,\nwant %+v\n%+v %+v", got, got.Translation, got.Explanation, wantLiver, wantLiver.Translation, wantLiver.Explanation)
	}
	if rows[0].Translation != nil || rows[0].Explanation != nil || rows[2].Translation.Review.Note != "Use the standard term." {
		t.Errorf("fatty liver has %+v %+v, hepatitis note %q; want no translation or explanation, and the note", rows[0].Translation, rows[0].Explanation, rows[2].Translation.Review.Note)
	}
}

// An unreviewed explanation puts an approved translation's entry back in the queue; one that needs changes, among
// those that need changes.
func TestQueue_AnExplanationCountsWhenThereIsOne(t *testing.T) {
	t.Parallel()
	owner, repo := reviewDatabase(t)
	if _, err := owner.Exec(t.Context(), `UPDATE term_explanation SET review_status = 'unreviewed'`); err != nil {
		t.Fatal(err)
	}
	want := []string{fattyLvr + " unreviewed", liver + " unreviewed", ulcer + " unreviewed"}
	if got, _ := queueIDs(t, repo, terminology.QueueFilter{Language: "bs", Status: "unreviewed", Limit: 10}); !reflect.DeepEqual(got, want) {
		t.Errorf("Queue(bs, unreviewed) = %v, want %v (by rank)", got, want)
	}
	if _, err := owner.Exec(t.Context(), `UPDATE term_explanation SET review_status = 'needs_changes', reviewed_at = now()`); err != nil {
		t.Fatal(err)
	}
	if got, _ := queueIDs(t, repo, terminology.QueueFilter{Language: "bs", Status: "needs_changes", Limit: 10}); !reflect.DeepEqual(got, []string{hepatitis + " needs_changes", liver + " needs_changes"}) {
		t.Errorf("Queue(bs, needs_changes) = %v", got)
	}
}

func TestProgress_CountsEveryTermPerLanguage(t *testing.T) {
	t.Parallel()
	_, repo := reviewDatabase(t)
	got, err := repo.Progress(t.Context())
	if err != nil || len(got) != 11 {
		t.Fatalf("Progress() = %+v, %v; want 11 languages", got, err)
	}
	if want := (terminology.LanguageProgress{Language: "la", Terms: 4, Approved: 1}); got[0] != want {
		t.Errorf("la = %+v, want %+v", got[0], want)
	}
	if want := (terminology.LanguageProgress{Language: "bs", Terms: 4, Approved: 1, NeedsChanges: 1}); got[1] != want {
		t.Errorf("bs = %+v, want %+v", got[1], want)
	}
	if want := (terminology.LanguageProgress{Language: "fr", Terms: 4}); got[10] != want {
		t.Errorf("fr = %+v, want %+v", got[10], want)
	}
}

// inTx runs fn in a transaction on the app role's pool, as the service does.
func inTx(t *testing.T, app *pgxpool.Pool, fn func(ctx context.Context) error) {
	t.Helper()
	if err := postgres.NewTransactor(app).WithTx(t.Context(), fn); err != nil {
		t.Fatal(err)
	}
}

func TestLockEntry_ReadsTheTermsReviewItsLabelsTranslationsAndExplanation(t *testing.T) {
	t.Parallel()
	_, app, repo := reviewDatabases(t)
	var got terminology.EntryState
	inTx(t, app, func(ctx context.Context) error {
		var err error
		got, err = repo.LockEntry(ctx, liver, "bs")
		return err
	})
	for i := range got.Translations {
		got.Translations[i].UpdatedAt = time.Time{}
	}
	got.Explanation.UpdatedAt = time.Time{}
	want := terminology.EntryState{
		TermReview: terminology.Review{Status: "approved", ReviewedAt: time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)},
		Labels:     []terminology.Label{{Language: "bs", Text: "jetra", Preferred: true}},
		Translations: []terminology.Translation{
			{TermID: liver, Language: "la", VersionID: 1, Review: terminology.Review{Status: "approved", ReviewedAt: reviewedAt}},
			{TermID: liver, Language: "bs", VersionID: 3, Review: terminology.Review{Status: "approved", ReviewedAt: reviewedAt, ReviewedBy: reviewerID}},
		},
		Explanation: &terminology.Explanation{TermID: liver, Language: "bs", Text: "Jetra je organ u trbuhu.", VersionID: 2, Review: terminology.Review{Status: "approved", ReviewedAt: reviewedAt}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LockEntry(liver, bs) = %+v %+v,\nwant %+v %+v", got, got.Explanation, want, want.Explanation)
	}
	inTx(t, app, func(ctx context.Context) error {
		s, err := repo.LockEntry(ctx, fattyLvr, "bs")
		if err != nil || s.Explanation != nil || len(s.Labels) != 0 || len(s.Translations) != 0 {
			t.Errorf("LockEntry(fatty liver, bs) = %+v, %v; want nothing in Bosnian", s, err)
		}
		for _, id := range []string{"00000000-0000-4000-8000-0000000000ff", "not-a-uuid"} {
			if _, err := repo.LockEntry(ctx, id, "bs"); !errors.Is(err, terminology.ErrTermNotFound) {
				t.Errorf("LockEntry(%s) = %v, want ErrTermNotFound", id, err)
			}
		}
		return nil
	})
}

// The app role writes what a reviewer does: a language's labels, a translation and an explanation (new or
// changed), and the term's review.
func TestReviewWrites_StoreTheLabelsTranslationExplanationAndTheTermsReview(t *testing.T) {
	t.Parallel()
	owner, app, repo := reviewDatabases(t)
	at := time.Date(2026, 11, 10, 9, 30, 0, 0, time.UTC)
	labels := []terminology.Label{{Language: "bs", Text: "masna jetra", Preferred: true}, {Language: "bs", Text: "steatoza"}}
	tr := terminology.Translation{TermID: fattyLvr, Language: "bs", VersionID: 1, UpdatedAt: at, Review: terminology.Review{Status: "unreviewed"}}
	ex := terminology.Explanation{TermID: fattyLvr, Language: "bs", Text: "Masnoća u jetri.", VersionID: 1, UpdatedAt: at, Review: terminology.Review{Status: "unreviewed"}}
	inTx(t, app, func(ctx context.Context) error {
		if _, err := repo.LockEntry(ctx, fattyLvr, "bs"); err != nil {
			return err
		}
		if err := repo.ReplaceLabels(ctx, fattyLvr, "bs", labels); err != nil {
			return err
		}
		if err := repo.PutTranslation(ctx, tr); err != nil {
			return err
		}
		return repo.PutExplanation(ctx, ex)
	})
	tr.VersionID, tr.Review = 2, terminology.Review{Status: "needs_changes", ReviewedAt: at, ReviewedBy: reviewerID, Note: "Steatoza is Latin-ish."}
	ex.VersionID, ex.Text, ex.Review = 2, "Masnoća u jetri, često bez tegoba.", terminology.Review{Status: "approved", ReviewedAt: at, ReviewedBy: reviewerID}
	termReview := terminology.Review{Status: "needs_changes", ReviewedAt: at, ReviewedBy: reviewerID, Note: "ignored"}
	inTx(t, app, func(ctx context.Context) error {
		if err := repo.PutTranslation(ctx, tr); err != nil {
			return err
		}
		if err := repo.PutExplanation(ctx, ex); err != nil {
			return err
		}
		return repo.SetTermReview(ctx, fattyLvr, termReview, at)
	})

	var got terminology.EntryState
	inTx(t, app, func(ctx context.Context) error {
		var err error
		got, err = repo.LockEntry(ctx, fattyLvr, "bs")
		return err
	})
	if !reflect.DeepEqual(got.Labels, labels) {
		t.Errorf("labels = %+v, want %+v", got.Labels, labels)
	}
	if len(got.Translations) != 1 || !reflect.DeepEqual(got.Translations[0], tr) {
		t.Errorf("translations = %+v, want [%+v]", got.Translations, tr)
	}
	if got.Explanation == nil || !reflect.DeepEqual(*got.Explanation, ex) {
		t.Errorf("explanation = %+v, want %+v", got.Explanation, ex)
	}
	if want := (terminology.Review{Status: "needs_changes", ReviewedAt: at, ReviewedBy: reviewerID}); got.TermReview != want {
		t.Errorf("the term's review = %+v, want %+v (its note is the curator's)", got.TermReview, want)
	}
	var updated time.Time
	var latin int
	if err := owner.QueryRow(t.Context(), `SELECT updated_at, (SELECT count(*) FROM term_label WHERE term_id = term.id AND language = 'la') FROM term WHERE id = $1`, fattyLvr).Scan(&updated, &latin); err != nil || !updated.Equal(at) || latin != 1 {
		t.Errorf("the term's updated_at %v and Latin labels %d, %v; want %v and its one Latin label untouched", updated, latin, err, at)
	}
}

// The curation keeps a translation in exactly the languages a term has labels in: a seed term's are unreviewed, a
// new language's takes the term's status, a language left without labels loses its own; a status set for the
// whole term is every translation's, with no reviewer.
func TestCuration_KeepsATranslationPerLanguageWithLabels(t *testing.T) {
	t.Parallel()
	owner, app, repo := reviewDatabases(t)
	curation := terminology.NewPostgresCurationRepository(owner)
	at := time.Date(2026, 11, 10, 9, 30, 0, 0, time.UTC)

	if created, err := curation.CreateFromSeed(t.Context(), terminology.Term{
		Category: "symptom", WikidataID: "Q38933", Rank: 120, ReviewStatus: "unreviewed",
		Labels: []terminology.Label{{Language: "la", Text: "febris", Preferred: true}, {Language: "en", Text: "fever", Preferred: true}},
	}); err != nil || !created {
		t.Fatalf("CreateFromSeed() = %v, %v", created, err)
	}
	var fever string
	if err := owner.QueryRow(t.Context(), `SELECT id FROM term WHERE wikidata_id = 'Q38933'`).Scan(&fever); err != nil {
		t.Fatal(err)
	}
	statuses := func(id string) map[string]string {
		t.Helper()
		out := map[string]string{}
		inTx(t, app, func(ctx context.Context) error {
			s, err := repo.LockEntry(ctx, id, "la")
			for _, tr := range s.Translations {
				out[tr.Language] = tr.Review.Status + " " + tr.Review.ReviewedBy
			}
			return err
		})
		return out
	}
	if got := statuses(fever); !reflect.DeepEqual(got, map[string]string{"la": "unreviewed ", "en": "unreviewed "}) {
		t.Errorf("a seed term's translations = %v, want la and en unreviewed", got)
	}

	liverTerm := termOf(t, curation, liver) // approved; translations la, bs approved
	liverTerm.Labels = []terminology.Label{
		{Language: "la", Text: "hepar", Preferred: true}, {Language: "bs", Text: "jetra", Preferred: true}, {Language: "de", Text: "Leber", Preferred: true},
	}
	if err := curation.UpdateTerm(t.Context(), liverTerm, at); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"la": "approved ", "bs": "approved " + reviewerID, "de": "approved "}
	if got := statuses(liver); !reflect.DeepEqual(got, want) {
		t.Errorf("after the labels changed: %v, want %v (de new with the term's status; hr, sr and en gone)", got, want)
	}
	if err := curation.SetTranslationsStatus(t.Context(), liver, "needs_changes", at); err != nil {
		t.Fatal(err)
	}
	if got := statuses(liver); !reflect.DeepEqual(got, map[string]string{"la": "needs_changes ", "bs": "needs_changes ", "de": "needs_changes "}) {
		t.Errorf("after SetTranslationsStatus(needs_changes): %v, want every one needs_changes with no reviewer", got)
	}
	if err := curation.SetTranslationsStatus(t.Context(), liver, "unreviewed", at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM term_translation WHERE term_id = $1 AND review_status = 'unreviewed' AND reviewed_at = $2 AND version_id > 1`, liver, at).Scan(&n); err != nil || n != 3 {
		t.Errorf("unreviewed translations keeping their date, at a new version: %d, %v; want 3", n, err)
	}
}
