-- name: SearchTerms :many
-- The terms with a label in any language that matches the folded text: 3 exact, 2 prefix, 1 word similarity
-- (<%, served by the trigram index). Within a match type, a label in Latin or the caller's UI language comes
-- first, then English, then the other languages: a label scores tier * 3 + its language's place (2, 1, 0), and a
-- term counts by its best label. A label that starts with a text of two
-- characters or more is also within word similarity (n of its n+1 trigrams, at least 0.67 against pg_trgm's 0.6),
-- so only a one-character text needs the LIKE in the filter: one index scan, not two. Each candidate label is
-- folded once (folded is materialized). Each term comes with its labels in the languages, as three parallel
-- arrays by language in the enum's order, preferred first, then alphabetically: one round trip per search.
-- Measured in docs/terminology/PROGRESS.md, 1.1.
WITH folded AS MATERIALIZED (
    SELECT l.term_id, l.language, term_fold(l.label) AS label
    FROM term_label l
    WHERE term_fold(@text::text) <% term_fold(l.label)
       OR (char_length(@text::text) < 2 AND term_fold(l.label) LIKE term_fold(@prefix::text) || '%')
), hits AS (
    SELECT f.term_id,
           max(3 * CASE WHEN f.label = term_fold(@text::text) THEN 3
                        WHEN f.label LIKE term_fold(@prefix::text) || '%' THEN 2
                        ELSE 1 END
               + CASE WHEN f.language = 'la' OR f.language::text = @ui_language::text THEN 2
                      WHEN f.language = 'en' THEN 1
                      ELSE 0 END)::integer AS score,
           max(word_similarity(term_fold(@text::text), f.label))::real AS similarity
    FROM folded f
    GROUP BY f.term_id
), best AS (
    SELECT t.id, t.reviewed_at, t.rank, t.category, t.review_status, t.wikidata_id, t.icd10, t.ta98, t.atc, t.pcs,
           h.score, h.similarity
    FROM hits h
    JOIN term t ON t.id = h.term_id
    WHERE sqlc.narg(category)::term_category IS NULL OR t.category = sqlc.narg(category)::term_category
    ORDER BY h.score DESC, h.similarity DESC, t.rank DESC, t.id
    LIMIT @max_terms
)
SELECT b.id, b.reviewed_at, b.rank, b.category, b.review_status, b.wikidata_id, b.icd10, b.ta98, b.atc, b.pcs,
       coalesce(l.languages, '{}')::text[] AS label_languages,
       coalesce(l.labels, '{}')::text[] AS labels,
       coalesce(l.preferred, '{}')::boolean[] AS label_preferred
FROM best b
CROSS JOIN LATERAL (
    SELECT array_agg(language::text ORDER BY language, preferred DESC, label) AS languages,
           array_agg(label ORDER BY language, preferred DESC, label) AS labels,
           array_agg(preferred ORDER BY language, preferred DESC, label) AS preferred
    FROM term_label
    WHERE term_id = b.id AND language::text = ANY(@languages::text[])
) l
ORDER BY b.score DESC, b.similarity DESC, b.rank DESC, b.id;

-- The curation (salusctl, as the owner; salus_app may only read): docs/terminology/SPEC.md §3 "Curation".

-- name: CreateSeedTerm :one
-- A new term from the seed, a draft; nothing when a term already has the Wikidata id (no row back).
INSERT INTO term (category, rank, wikidata_id, icd10, ta98, atc, pcs)
VALUES (@category, @rank, @wikidata_id, sqlc.narg(icd10), sqlc.narg(ta98), sqlc.narg(atc), sqlc.narg(pcs))
ON CONFLICT (wikidata_id) DO NOTHING
RETURNING id;

-- name: InsertLabels :exec
-- A term's labels, as three parallel arrays of one length (unnest in the select list zips them).
INSERT INTO term_label (term_id, language, label, preferred)
SELECT @term_id::uuid, unnest(@languages::text[])::term_language, unnest(@labels::text[]), unnest(@preferred::boolean[]);

-- name: DeleteLabels :exec
DELETE FROM term_label WHERE term_id = @term_id;

-- name: UpdateTerm :execrows
UPDATE term
SET category = @category, rank = @rank, wikidata_id = sqlc.narg(wikidata_id), icd10 = sqlc.narg(icd10),
    ta98 = sqlc.narg(ta98), atc = sqlc.narg(atc), pcs = sqlc.narg(pcs), review_status = @review_status,
    reviewed_at = sqlc.narg(reviewed_at), reviewed_by = sqlc.narg(reviewed_by), note = sqlc.narg(note),
    updated_at = @updated_at
WHERE id = @id;

-- name: ListTerms :many
-- One term (id) or every term of a review status (any when null), each with its labels in every language as three
-- parallel arrays, by language in the enum's order, preferred first, then alphabetically.
SELECT t.id, t.reviewed_at, t.reviewed_by, t.rank, t.category, t.review_status, t.wikidata_id, t.icd10, t.ta98, t.atc,
       t.pcs, t.note,
       coalesce(l.languages, '{}')::text[] AS label_languages,
       coalesce(l.labels, '{}')::text[] AS labels,
       coalesce(l.preferred, '{}')::boolean[] AS label_preferred
FROM term t
CROSS JOIN LATERAL (
    SELECT array_agg(language::text ORDER BY language, preferred DESC, label) AS languages,
           array_agg(label ORDER BY language, preferred DESC, label) AS labels,
           array_agg(preferred ORDER BY language, preferred DESC, label) AS preferred
    FROM term_label
    WHERE term_id = t.id
) l
WHERE (sqlc.narg(id)::uuid IS NULL OR t.id = sqlc.narg(id)::uuid)
  AND (sqlc.narg(review_status)::text IS NULL OR t.review_status = sqlc.narg(review_status)::text)
ORDER BY t.category, t.rank DESC, t.id;

-- name: CountTermsByCategory :many
SELECT category, count(*)::integer AS terms FROM term GROUP BY category ORDER BY category;

-- name: GlossaryTerms :many
-- The terms with a label in the line's language equal to one of the line's candidate phrases, both folded, each
-- by its longest label found (the preferred one of a length), with its label in the target language (the
-- preferred one, else the first alphabetically); none without one. The most common first. One index lookup
-- (term_label_language_folded) per candidate.
WITH found AS (
    SELECT DISTINCT ON (l.term_id) l.term_id, l.label
    FROM (SELECT DISTINCT term_fold(c) AS folded FROM unnest(@candidates::text[]) AS c) AS c
    JOIN term_label l ON l.language = @source_language::term_language AND term_fold(l.label) = c.folded
    ORDER BY l.term_id, char_length(l.label) DESC, l.preferred DESC, l.label
), target AS (
    SELECT DISTINCT ON (l.term_id) l.term_id, l.label
    FROM term_label l
    WHERE l.language = @target_language::term_language AND l.term_id IN (SELECT term_id FROM found)
    ORDER BY l.term_id, l.preferred DESC, l.label
)
SELECT f.label::text AS source, tl.label::text AS target
FROM found f
JOIN target tl ON tl.term_id = f.term_id
JOIN term t ON t.id = f.term_id
ORDER BY t.rank DESC, t.id
LIMIT @max_terms;
