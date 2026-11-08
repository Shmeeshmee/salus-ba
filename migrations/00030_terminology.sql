-- +goose Up
-- Medical terminology (docs/terminology, package 0.1; SPEC.md §3 "Database"):
--   - term: one concept of a category, with the codes that identify it (each nullable, as Wikidata has them;
--     several codes of one kind are joined with ", "), its rank (the number of Wikipedia language editions with
--     an article about the Wikidata item, its sitelinks: higher is more common) and the curator's review.
--     No personal data: the review keeps only its date, never who reviewed;
--   - term_label: the labels of a term per language (term_language: Latin plus the ten app languages, its own
--     enum beside the languages plan's language), one preferred per term and language, the rest synonyms;
--   - term_fold: lower case without accents, immutable so the labels can be indexed on it (unaccent itself is
--     only stable). The trigram index serves the search's prefix (LIKE) and word-similarity (<%) matches;
--   - reference data: salus_app may only read it. salusctl writes it as the owner.

CREATE EXTENSION IF NOT EXISTS pg_trgm SCHEMA public;
CREATE EXTENSION IF NOT EXISTS unaccent SCHEMA public;

-- +goose StatementBegin
CREATE FUNCTION term_fold(text) RETURNS text
    LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
BEGIN ATOMIC
    SELECT lower(public.unaccent('public.unaccent'::regdictionary, $1));
END;
-- +goose StatementEnd

CREATE TYPE term_category AS ENUM ('anatomy', 'diagnosis', 'symptom', 'procedure', 'drug');
CREATE TYPE term_language AS ENUM ('la', 'bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en', 'de', 'ar', 'tr', 'it', 'fr');

CREATE TABLE term (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    reviewed_at timestamptz,
    rank integer DEFAULT 0 NOT NULL CHECK (rank >= 0),
    category term_category NOT NULL,
    review_status text DEFAULT 'draft' NOT NULL CHECK (review_status IN ('draft', 'reviewed')),
    wikidata_id text UNIQUE CHECK (wikidata_id ~ '^Q[1-9][0-9]*$'),
    icd10 text CHECK (icd10 <> ''),
    ta98 text CHECK (ta98 <> ''),
    atc text CHECK (atc <> ''),
    pcs text CHECK (pcs <> ''),
    note text CHECK (note <> ''),
    CONSTRAINT term_reviewed_has_date CHECK (review_status = 'draft' OR reviewed_at IS NOT NULL)
);

CREATE INDEX term_category_rank ON term (category, rank DESC);

CREATE TABLE term_label (
    created_at timestamptz DEFAULT now() NOT NULL,
    term_id uuid NOT NULL REFERENCES term (id) ON DELETE CASCADE,
    language term_language NOT NULL,
    preferred boolean DEFAULT false NOT NULL,
    label text NOT NULL CHECK (label <> '' AND label = btrim(label) AND length(label) <= 300),
    PRIMARY KEY (term_id, language, label)
);

CREATE UNIQUE INDEX term_label_one_preferred ON term_label (term_id, language) WHERE preferred;
CREATE INDEX term_label_folded_trgm ON term_label USING gin (term_fold(label) gin_trgm_ops);

GRANT SELECT ON term, term_label TO salus_app;
