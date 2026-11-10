-- +goose Up
-- The review of the term list in the app (docs/plans/v1-finish.md "Contract", SAL-13 and SAL-14; owner,
-- 2026-11-10):
--   - review_status, one status for terms, their translations and their explanations: unreviewed, approved or
--     needs_changes. It replaces term.review_status's draft and reviewed (draft is unreviewed, reviewed is
--     approved), so the cheatsheet's "Not reviewed yet" mark and the review panel read one status. Editing an
--     approved entry sets it back to unreviewed (the service's rule).
--   - reviewed_by: the account that set the status last, an opaque id (owner, 2026-11-10: the terminology tables
--     now hold a reviewer's account id, so they are personal data of staff; retention keeps them with the term).
--     Empty for a status set by salusctl's import (the curator's spreadsheet names no one) and before the review.
--     reviewed_at: when; set with every status but unreviewed, kept when an entry goes back to unreviewed.
--   - term_translation: the review of a term's labels in one language. A term and language with labels and no row
--     is unreviewed. Filled from each term's status for the labels it has.
--   - term_explanation: a short plain-language explanation of a term in an app language (not Latin), at most 500
--     characters, informational only (never a diagnosis), drafted and then reviewed.
-- Both new tables carry version_id for If-Match on the review panel's edits. The app reads them; the reviewer's
-- writes come with SAL-14.

CREATE DOMAIN review_status AS text CHECK (VALUE IN ('unreviewed', 'approved', 'needs_changes'));

ALTER TABLE term DROP CONSTRAINT term_reviewed_has_date, DROP CONSTRAINT term_review_status_check;
ALTER TABLE term ALTER COLUMN review_status DROP DEFAULT;
UPDATE term SET review_status = CASE review_status WHEN 'reviewed' THEN 'approved' ELSE 'unreviewed' END;
ALTER TABLE term
    ALTER COLUMN review_status TYPE review_status,
    ALTER COLUMN review_status SET DEFAULT 'unreviewed',
    ADD COLUMN reviewed_by uuid REFERENCES account (id),
    ADD CONSTRAINT term_reviewed_has_date CHECK (review_status = 'unreviewed' OR reviewed_at IS NOT NULL);

CREATE TABLE term_translation (
    term_id uuid NOT NULL REFERENCES term (id) ON DELETE CASCADE,
    updated_at timestamptz DEFAULT now() NOT NULL,
    reviewed_at timestamptz,
    reviewed_by uuid REFERENCES account (id),
    version_id integer DEFAULT 1 NOT NULL CHECK (version_id > 0),
    language term_language NOT NULL,
    review_status review_status DEFAULT 'unreviewed' NOT NULL,
    note text CHECK (note <> ''),
    PRIMARY KEY (term_id, language),
    CONSTRAINT term_translation_reviewed_has_date CHECK (review_status = 'unreviewed' OR reviewed_at IS NOT NULL)
);

INSERT INTO term_translation (term_id, language, review_status, reviewed_at)
SELECT DISTINCT term_label.term_id, term_label.language, term.review_status, term.reviewed_at
FROM term_label JOIN term ON term.id = term_label.term_id;

CREATE TABLE term_explanation (
    term_id uuid NOT NULL REFERENCES term (id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    reviewed_at timestamptz,
    reviewed_by uuid REFERENCES account (id),
    version_id integer DEFAULT 1 NOT NULL CHECK (version_id > 0),
    language term_language NOT NULL CHECK (language <> 'la'),
    review_status review_status DEFAULT 'unreviewed' NOT NULL,
    text text NOT NULL CHECK (text <> '' AND text = btrim(text) AND length(text) <= 500),
    note text CHECK (note <> ''),
    PRIMARY KEY (term_id, language),
    CONSTRAINT term_explanation_reviewed_has_date CHECK (review_status = 'unreviewed' OR reviewed_at IS NOT NULL)
);

GRANT SELECT ON term_translation, term_explanation TO salus_app;
