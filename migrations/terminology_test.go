package migrations_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The terminology tables (00030, docs/terminology/SPEC.md §3 "Database") are reference data: the app reads them;
// since 00039 it writes only what the review panel needs (a term's review columns, a language's labels replaced),
// never a term's codes or note, a new term or a deletion of one; salusctl writes the rest as the owner.
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
		`UPDATE term SET rank = 1, wikidata_id = 'Q1'`,
		`TRUNCATE term_label`,
		`UPDATE term_label SET preferred = false`,
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

// The app reads translations and explanations, and writes them for a reviewer (00039): never a deletion.
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
	for _, stmt := range []string{`DELETE FROM term_translation`, `DELETE FROM term_explanation`, `TRUNCATE term_explanation`} {
		if _, err := app.Exec(t.Context(), stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.AppRole, err)
		}
	}
}

// The reviewer's writes (00039, SAL-14): the app replaces a language's labels, writes translations and
// explanations, and a term's review columns, as the review panel does.
func TestTerminology_TheAppRoleWritesTheReview(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	if _, err := owner.Exec(t.Context(), `INSERT INTO term (id, category, rank) VALUES ('00000000-0000-4000-8000-000000000001', 'anatomy', 10);
		INSERT INTO term_label (term_id, language, label, preferred) VALUES ('00000000-0000-4000-8000-000000000001', 'la', 'hepar', true)`); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`DELETE FROM term_label WHERE language = 'la'`,
		`INSERT INTO term_label (term_id, language, label, preferred) VALUES ('00000000-0000-4000-8000-000000000001', 'la', 'hepar', true)`,
		`INSERT INTO term_translation (term_id, language) VALUES ('00000000-0000-4000-8000-000000000001', 'la')`,
		`UPDATE term_translation SET review_status = 'approved', reviewed_at = now(), note = 'x', version_id = 2, updated_at = now()`,
		`INSERT INTO term_explanation (term_id, language, text) VALUES ('00000000-0000-4000-8000-000000000001', 'en', 'The liver.')
		 ON CONFLICT (term_id, language) DO UPDATE SET text = excluded.text`,
		`UPDATE term SET review_status = 'approved', reviewed_at = now(), reviewed_by = NULL, updated_at = now()`,
		`SELECT id FROM term WHERE id = '00000000-0000-4000-8000-000000000001' FOR UPDATE`,
	} {
		if _, err := app.Exec(t.Context(), stmt); err != nil {
			t.Errorf("%s as %s: %v", stmt, pgtest.AppRole, err)
		}
	}
}

// The reviewer permission (00039): one row per staff account the operator made a reviewer, gone with the account;
// the app reads it and can never grant or revoke it.
func TestTerminology_TheReviewerPermissionIsTheOperators(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	const account = "00000000-0000-4000-8000-0000000000a1"
	if _, err := owner.Exec(t.Context(), `INSERT INTO account (id, email, password_hash, role) VALUES ('`+account+`', 'reviewer@example.test', 'x', 'patient'); -- the service refuses a patient; the table does not care
		INSERT INTO term_reviewer (account_id) VALUES ('`+account+`')`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := app.QueryRow(t.Context(), `SELECT count(*) FROM term_reviewer WHERE granted_at IS NOT NULL`).Scan(&n); err != nil || n != 1 {
		t.Errorf("%s reads %d reviewers, %v; want 1", pgtest.AppRole, n, err)
	}
	for _, stmt := range []string{
		`INSERT INTO term_reviewer (account_id) VALUES ('` + account + `')`,
		`UPDATE term_reviewer SET granted_at = now()`,
		`DELETE FROM term_reviewer`,
	} {
		if _, err := app.Exec(t.Context(), stmt); sqlState(err) != "42501" {
			t.Errorf("%s as %s: %v, want insufficient_privilege", stmt, pgtest.AppRole, err)
		}
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO term_reviewer (account_id) VALUES ('`+account+`')`); sqlState(err) != "23505" {
		t.Errorf("a second grant: %v, want unique_violation", err)
	}
	if _, err := owner.Exec(t.Context(), `DELETE FROM account`); err != nil {
		t.Fatalf("deleting the account: %v", err)
	}
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM term_reviewer`).Scan(&n); err != nil || n != 0 {
		t.Errorf("reviewers after the account's deletion: %d, %v; want 0", n, err)
	}
}
