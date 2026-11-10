-- +goose Up
-- The review panel of the term list (SAL-14; docs/plans/v1-finish.md "Decisions", owner 2026-11-10):
--   - term_reviewer: the staff accounts the operator made reviewers (salusctl term-reviewer grant|revoke). A
--     platform permission, not a clinic's: the term list is shared by every clinic. The app only reads it, so no
--     clinic admin and no bug in the app can grant it. Gone with the account.
--   - the reviewer's writes: salus_app replaces a term's labels in one language (term_label), writes the
--     translations' and explanations' reviews and texts, and a term's review columns, whose status follows its
--     translations (docs/terminology/SPEC.md §3 "Review"). Never a term's codes, rank or curator's note, a new term
--     or a deletion of one: those stay salusctl's, as the owner.

CREATE TABLE term_reviewer (
    account_id uuid NOT NULL PRIMARY KEY REFERENCES account (id) ON DELETE CASCADE,
    granted_at timestamptz DEFAULT now() NOT NULL
);

GRANT SELECT ON term_reviewer TO salus_app;
GRANT INSERT, DELETE ON term_label TO salus_app;
GRANT INSERT ON term_translation, term_explanation TO salus_app;
-- Updates only of a review and a text, never of a row's key or creation time.
GRANT UPDATE (updated_at, reviewed_at, reviewed_by, version_id, review_status, note) ON term_translation TO salus_app;
GRANT UPDATE (updated_at, reviewed_at, reviewed_by, version_id, review_status, note, text) ON term_explanation TO salus_app;
GRANT UPDATE (review_status, reviewed_at, reviewed_by, updated_at) ON term TO salus_app;

-- A term has a translation in every language it has labels in (docs/terminology/SPEC.md §3 "Review"): 00037 filled
-- them, and the terms imported since get theirs here, with the term's status, as 00037 did.
INSERT INTO term_translation (term_id, language, review_status, reviewed_at)
SELECT DISTINCT term_label.term_id, term_label.language, term.review_status, term.reviewed_at
FROM term_label JOIN term ON term.id = term_label.term_id
ON CONFLICT (term_id, language) DO NOTHING;
