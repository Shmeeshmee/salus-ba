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

-- name: SyncTranslations :exec
-- A term has a translation in exactly the languages it has labels in (docs/terminology/SPEC.md §3 "Review"): one
-- for a language without one, with the term's status and date, and none for a language left without labels.
WITH gone AS (
    DELETE FROM term_translation tr
    WHERE tr.term_id = @term_id
      AND NOT EXISTS (SELECT FROM term_label l WHERE l.term_id = tr.term_id AND l.language = tr.language)
)
INSERT INTO term_translation (term_id, language, review_status, reviewed_at)
SELECT DISTINCT l.term_id, l.language, t.review_status, t.reviewed_at
FROM term_label l JOIN term t ON t.id = l.term_id
WHERE l.term_id = @term_id
ON CONFLICT (term_id, language) DO NOTHING;

-- name: SetTranslationsStatus :exec
-- The curator's file sets one status per term: every translation's, with no reviewer (the file names no one).
UPDATE term_translation
SET review_status = @review_status::text, reviewed_by = NULL,
    reviewed_at = CASE WHEN @review_status::text = 'unreviewed' THEN reviewed_at ELSE @at::timestamptz END,
    version_id = version_id + 1, updated_at = @at::timestamptz
WHERE term_id = @term_id;

-- The review panel (SAL-14), on the app's connection: docs/terminology/SPEC.md §3 "Review".

-- name: ReviewQueue :many
-- An entry per term in the language: status_order 0 needs_changes (either needs changes), 1 unreviewed (no
-- translation, or the translation or the explanation unreviewed), 2 approved; an explanation that does not exist
-- does not count. A page by status, rank descending and id, after the keyset (after_order, after_rank, after_id)
-- when after_id is set; each entry with its labels in the language, Latin and English.
WITH entries AS (
    SELECT t.id, t.reviewed_at, t.rank, t.category, t.review_status, t.wikidata_id, t.icd10, t.ta98, t.atc, t.pcs,
           tr.term_id IS NOT NULL AS has_translation, tr.updated_at AS tr_updated_at, tr.reviewed_at AS tr_reviewed_at,
           tr.reviewed_by AS tr_reviewed_by, tr.version_id AS tr_version_id, tr.review_status AS tr_status, tr.note AS tr_note,
           ex.term_id IS NOT NULL AS has_explanation, ex.updated_at AS ex_updated_at, ex.reviewed_at AS ex_reviewed_at,
           ex.reviewed_by AS ex_reviewed_by, ex.version_id AS ex_version_id, ex.review_status AS ex_status, ex.note AS ex_note,
           ex.text AS ex_text,
           CASE WHEN tr.review_status = 'needs_changes' OR ex.review_status = 'needs_changes' THEN 0
                WHEN tr.term_id IS NULL OR tr.review_status = 'unreviewed' OR ex.review_status = 'unreviewed' THEN 1
                ELSE 2 END AS status_order
    FROM term t
    LEFT JOIN term_translation tr ON tr.term_id = t.id AND tr.language = @language::term_language
    LEFT JOIN term_explanation ex ON ex.term_id = t.id AND ex.language = @language::term_language
), page AS (
    SELECT * FROM entries e
    WHERE (sqlc.narg(status_order)::integer IS NULL OR e.status_order = sqlc.narg(status_order)::integer)
      AND (sqlc.narg(after_id)::uuid IS NULL
           OR e.status_order > @after_order::integer
           OR (e.status_order = @after_order::integer
               AND (e.rank < @after_rank::integer OR (e.rank = @after_rank::integer AND e.id > sqlc.narg(after_id)::uuid))))
    ORDER BY e.status_order, e.rank DESC, e.id
    LIMIT @max_entries
)
SELECT p.id, p.reviewed_at, p.rank, p.category, p.review_status, p.wikidata_id, p.icd10, p.ta98, p.atc, p.pcs,
       p.has_translation::boolean AS has_translation, p.tr_updated_at, p.tr_reviewed_at, p.tr_reviewed_by,
       p.tr_version_id, coalesce(p.tr_status, '')::text AS tr_status, p.tr_note,
       p.has_explanation::boolean AS has_explanation, p.ex_updated_at, p.ex_reviewed_at, p.ex_reviewed_by,
       p.ex_version_id, coalesce(p.ex_status, '')::text AS ex_status, p.ex_note, p.ex_text,
       p.status_order::integer AS status_order,
       coalesce(l.languages, '{}')::text[] AS label_languages,
       coalesce(l.labels, '{}')::text[] AS labels,
       coalesce(l.preferred, '{}')::boolean[] AS label_preferred
FROM page p
CROSS JOIN LATERAL (
    SELECT array_agg(language::text ORDER BY language, preferred DESC, label) AS languages,
           array_agg(label ORDER BY language, preferred DESC, label) AS labels,
           array_agg(preferred ORDER BY language, preferred DESC, label) AS preferred
    FROM term_label
    WHERE term_id = p.id AND (language = @language::term_language OR language IN ('la', 'en'))
) l
ORDER BY p.status_order, p.rank DESC, p.id;

-- name: ReviewProgress :many
-- Per language, every term and how many of its entries there are approved (2) or need changes (0), as ReviewQueue.
SELECT l.language::text AS language, count(*)::integer AS terms,
       (count(*) FILTER (WHERE s.status_order = 2))::integer AS approved,
       (count(*) FILTER (WHERE s.status_order = 0))::integer AS needs_changes
FROM term t
CROSS JOIN unnest(enum_range(NULL::term_language)) AS l(language)
LEFT JOIN term_translation tr ON tr.term_id = t.id AND tr.language = l.language
LEFT JOIN term_explanation ex ON ex.term_id = t.id AND ex.language = l.language
CROSS JOIN LATERAL (
    SELECT CASE WHEN tr.review_status = 'needs_changes' OR ex.review_status = 'needs_changes' THEN 0
                WHEN tr.term_id IS NULL OR tr.review_status = 'unreviewed' OR ex.review_status = 'unreviewed' THEN 1
                ELSE 2 END AS status_order
) s
GROUP BY l.language
ORDER BY l.language;

-- name: LockTerm :one
-- A review write holds the term until it commits, so two reviewers' writes of one term run one after the other.
SELECT review_status, reviewed_at, reviewed_by FROM term WHERE id = @id FOR UPDATE;

-- name: TermLabelsIn :many
SELECT label, preferred FROM term_label WHERE term_id = @term_id AND language = @language::term_language
ORDER BY preferred DESC, label;

-- name: TermTranslations :many
SELECT language::text AS language, updated_at, reviewed_at, reviewed_by, version_id, review_status, note
FROM term_translation WHERE term_id = @term_id ORDER BY term_translation.language;

-- name: TermExplanationIn :many
SELECT updated_at, reviewed_at, reviewed_by, version_id, review_status, note, text
FROM term_explanation WHERE term_id = @term_id AND language = @language::term_language;

-- name: DeleteLabelsIn :exec
DELETE FROM term_label WHERE term_id = @term_id AND language = @language::term_language;

-- name: PutTranslation :exec
INSERT INTO term_translation (term_id, language, updated_at, reviewed_at, reviewed_by, version_id, review_status, note)
VALUES (@term_id, @language::term_language, @updated_at, sqlc.narg(reviewed_at), sqlc.narg(reviewed_by), @version_id,
        @review_status::text, sqlc.narg(note))
ON CONFLICT (term_id, language) DO UPDATE
SET updated_at = excluded.updated_at, reviewed_at = excluded.reviewed_at, reviewed_by = excluded.reviewed_by,
    version_id = excluded.version_id, review_status = excluded.review_status, note = excluded.note;

-- name: PutExplanation :exec
INSERT INTO term_explanation (term_id, language, updated_at, reviewed_at, reviewed_by, version_id, review_status, note, text)
VALUES (@term_id, @language::term_language, @updated_at, sqlc.narg(reviewed_at), sqlc.narg(reviewed_by), @version_id,
        @review_status::text, sqlc.narg(note), @text)
ON CONFLICT (term_id, language) DO UPDATE
SET updated_at = excluded.updated_at, reviewed_at = excluded.reviewed_at, reviewed_by = excluded.reviewed_by,
    version_id = excluded.version_id, review_status = excluded.review_status, note = excluded.note, text = excluded.text;

-- name: SetTermReview :exec
UPDATE term
SET review_status = @review_status::text, reviewed_at = sqlc.narg(reviewed_at), reviewed_by = sqlc.narg(reviewed_by),
    updated_at = @updated_at
WHERE id = @id;

-- name: TagCandidates :many
-- The candidates of the texts being tagged (SAL-13) that equal, both folded, a label in their language of a term
-- with an explanation, each with the most common such term. One index lookup (term_label_language_folded) per
-- distinct candidate.
WITH c AS (
    SELECT DISTINCT l.language, p.phrase, term_fold(p.phrase) AS folded
    FROM unnest(@languages::text[]) WITH ORDINALITY AS l (language, n)
    JOIN unnest(@phrases::text[]) WITH ORDINALITY AS p (phrase, n) ON p.n = l.n
)
SELECT DISTINCT ON (c.language, c.phrase) c.language::text AS language, c.phrase::text AS phrase, l.term_id
FROM c
JOIN term_label l ON l.language = c.language::term_language AND term_fold(l.label) = c.folded
JOIN term t ON t.id = l.term_id
WHERE EXISTS (SELECT FROM term_explanation e WHERE e.term_id = l.term_id)
ORDER BY c.language, c.phrase, t.rank DESC, t.id;

-- name: ExplanationsIn :many
-- Each term's explanation in the language, else its English one; one that needs changes is left out.
SELECT DISTINCT ON (e.term_id) e.term_id, e.language::text AS language, e.updated_at, e.reviewed_at, e.reviewed_by,
       e.version_id, e.review_status, e.note, e.text
FROM term_explanation e
WHERE e.language IN (@language::term_language, 'en') AND e.review_status <> 'needs_changes'
ORDER BY e.term_id, e.language = @language::term_language DESC;

-- name: TermIDByWikidata :one
SELECT id FROM term WHERE wikidata_id = @wikidata_id;
