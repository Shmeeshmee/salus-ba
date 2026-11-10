package migrations_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The terminology tables (00030, docs/terminology/SPEC.md §3 "Database") are reference data: the app reads them
// and never writes; salusctl writes them as the owner.
func TestTerminology_TheAppRoleOnlyReads(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	if _, err := owner.Exec(t.Context(), `INSERT INTO term (id, category, rank) VALUES ('00000000-0000-4000-8000-000000000001', 'anatomy', 10);
		INSERT INTO term_label (term_id, language, label, preferred) VALUES ('00000000-0000-4000-8000-000000000001', 'la', 'hepar', true)`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := app.QueryRow(t.Context(), `SELECT count(*) FROM term JOIN term_label ON term_label.term_id = term.id`).Scan(&n); err != nil || n != 1 {
		t.Errorf("%s reads %d labelled terms, %v; want 1", pgtest.AppRole, n, err)
	}
	for _, stmt := range []string{
		`INSERT INTO term (category, rank) VALUES ('drug', 1)`,
		`UPDATE term SET note = 'x'`,
		`DELETE FROM term`,
		`TRUNCATE term_label`,
		`INSERT INTO term_label (term_id, language, label) VALUES ('00000000-0000-4000-8000-000000000001', 'en', 'liver')`,
		`UPDATE term_label SET preferred = false`,
		`DELETE FROM term_label`,
	} {
		if _, err := app.Exec(t.Context(), stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.AppRole, err)
		}
	}
}

// One preferred label per term and language; labels are trimmed and not blank; a reviewed term has its date.
func TestTerminology_Constraints(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)
	if _, err := db.Exec(t.Context(), `INSERT INTO term (id, category, rank) VALUES ('00000000-0000-4000-8000-000000000001', 'anatomy', 10);
		INSERT INTO term_label (term_id, language, label, preferred) VALUES ('00000000-0000-4000-8000-000000000001', 'la', 'hepar', true)`); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ stmt, state string }{
		{`INSERT INTO term_label (term_id, language, label, preferred) VALUES ('00000000-0000-4000-8000-000000000001', 'la', 'iecur', true)`, "23505"},
		{`INSERT INTO term_label (term_id, language, label) VALUES ('00000000-0000-4000-8000-000000000001', 'la', ' iecur')`, "23514"},
		{`INSERT INTO term_label (term_id, language, label) VALUES ('00000000-0000-4000-8000-000000000001', 'la', '')`, "23514"},
		{`INSERT INTO term_label (term_id, language, label) VALUES ('00000000-0000-4000-8000-000000000001', 'xx', 'iecur')`, "22P02"},
		{`UPDATE term SET review_status = 'approved'`, "23514"},
		{`UPDATE term SET review_status = 'reviewed', reviewed_at = now()`, "23514"},
		{`INSERT INTO term (category, rank, wikidata_id) VALUES ('drug', 1, 'P267')`, "23514"},
	} {
		if _, err := db.Exec(t.Context(), c.stmt); sqlState(err) != c.state {
			t.Errorf("%s: %v, want %s", c.stmt, err, c.state)
		}
	}
	if _, err := db.Exec(t.Context(), `INSERT INTO term_label (term_id, language, label) VALUES ('00000000-0000-4000-8000-000000000001', 'la', 'iecur');
		UPDATE term SET review_status = 'approved', reviewed_at = now()`); err != nil {
		t.Errorf("a synonym and a review: %v", err)
	}
	if _, err := db.Exec(t.Context(), `DELETE FROM term`); err != nil {
		t.Errorf("deleting a term with labels: %v", err)
	}
}

// The search folds case and accents through an immutable function, and the labels are indexed on it.
func TestTerminology_FoldIsImmutableAndIndexed(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)
	var folded string
	if err := db.QueryRow(t.Context(), `SELECT term_fold('Čir na ŽELUCU, Đ')`).Scan(&folded); err != nil || folded != "cir na zelucu, d" {
		t.Errorf("term_fold = %q, %v; want %q", folded, err, "cir na zelucu, d")
	}
	var def string
	if err := db.QueryRow(t.Context(), `SELECT indexdef FROM pg_indexes WHERE indexname = 'term_label_folded_trgm'`).Scan(&def); err != nil {
		t.Fatalf("no index term_label_folded_trgm: %v", err)
	}
	if want := `CREATE INDEX term_label_folded_trgm ON public.term_label USING gin (term_fold(label) gin_trgm_ops)`; def != want {
		t.Errorf("index %s, want %s", def, want)
	}
}

// One review status for terms, translations and explanations (00037, docs/plans/v1-finish.md "Contract"):
// unreviewed, approved or needs_changes, the same check on all three; a reviewed entry has its date; a reviewer
// is an account; an explanation is trimmed, at most 500 characters and in an app language, never Latin.
func TestTerminology_ReviewOfTranslationsAndExplanations(t *testing.T) {
	t.Parallel()
	db := pgtest.NewDatabase(t)
	if _, err := db.Exec(t.Context(), `INSERT INTO term (id, category, rank) VALUES ('00000000-0000-4000-8000-000000000001', 'diagnosis', 10);
		INSERT INTO term_label (term_id, language, label, preferred) VALUES ('00000000-0000-4000-8000-000000000001', 'en', 'tonsillitis', true)`); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := db.QueryRow(t.Context(), `SELECT review_status FROM term`).Scan(&status); err != nil || status != "unreviewed" {
		t.Errorf("a new term's review_status = %q, %v; want unreviewed", status, err)
	}
	if _, err := db.Exec(t.Context(), `INSERT INTO term_translation (term_id, language) VALUES ('00000000-0000-4000-8000-000000000001', 'en');
		INSERT INTO term_explanation (term_id, language, text) VALUES ('00000000-0000-4000-8000-000000000001', 'en', 'Inflammation of the tonsils.')`); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"term_translation", "term_explanation"} {
		var status string
		var version int32
		if err := db.QueryRow(t.Context(), `SELECT review_status, version_id FROM `+table).Scan(&status, &version); err != nil || status != "unreviewed" || version != 1 {
			t.Errorf("a new %s row: status %q, version %d, %v; want unreviewed, 1", table, status, version, err)
		}
	}
	for _, c := range []struct{ stmt, state string }{
		{`UPDATE term_translation SET review_status = 'reviewed', reviewed_at = now()`, "23514"},
		{`UPDATE term_translation SET review_status = 'approved'`, "23514"},
		{`UPDATE term_explanation SET review_status = 'needs_changes'`, "23514"},
		{`UPDATE term_explanation SET reviewed_by = '00000000-0000-4000-8000-0000000000ff'`, "23503"},
		{`UPDATE term_explanation SET text = ' Inflammation.'`, "23514"},
		{`UPDATE term_explanation SET text = ''`, "23514"},
		{`UPDATE term_explanation SET text = repeat('a', 501)`, "23514"},
		{`INSERT INTO term_explanation (term_id, language, text) VALUES ('00000000-0000-4000-8000-000000000001', 'la', 'x')`, "23514"},
		{`INSERT INTO term_explanation (term_id, language, text) VALUES ('00000000-0000-4000-8000-000000000001', 'en', 'again')`, "23505"},
		{`INSERT INTO term_translation (term_id, language) VALUES ('00000000-0000-4000-8000-000000000001', 'en')`, "23505"},
		{`UPDATE term SET review_status = 'needs_changes'`, "23514"},
	} {
		if _, err := db.Exec(t.Context(), c.stmt); sqlState(err) != c.state {
			t.Errorf("%s: %v, want %s", c.stmt, err, c.state)
		}
	}
	if _, err := db.Exec(t.Context(), `UPDATE term SET review_status = 'needs_changes', reviewed_at = now(), note = 'Check the Latin.';
		UPDATE term_translation SET review_status = 'approved', reviewed_at = now();
		UPDATE term_explanation SET review_status = 'needs_changes', reviewed_at = now(), note = 'Too technical.'`); err != nil {
		t.Errorf("reviews with their dates: %v", err)
	}
	if _, err := db.Exec(t.Context(), `DELETE FROM term`); err != nil {
		t.Errorf("deleting a term with its translations and explanations: %v", err)
	}
}

// The app reads translations and explanations; it writes none (SAL-14 grants its reviewer writes).
func TestTerminology_TheAppRoleReadsReviews(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	if _, err := owner.Exec(t.Context(), `INSERT INTO term (id, category, rank) VALUES ('00000000-0000-4000-8000-000000000001', 'diagnosis', 10);
		INSERT INTO term_translation (term_id, language) VALUES ('00000000-0000-4000-8000-000000000001', 'en');
		INSERT INTO term_explanation (term_id, language, text) VALUES ('00000000-0000-4000-8000-000000000001', 'en', 'Inflammation of the tonsils.')`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := app.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM term_translation) + (SELECT count(*) FROM term_explanation)`).Scan(&n); err != nil || n != 2 {
		t.Errorf("%s reads %d rows, %v; want 2", pgtest.AppRole, n, err)
	}
	for _, stmt := range []string{`UPDATE term_translation SET note = 'x'`, `DELETE FROM term_explanation`} {
		if _, err := app.Exec(t.Context(), stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.AppRole, err)
		}
	}
}
