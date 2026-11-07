-- name: SearchTerms :many
-- The terms with a label in any language that matches the folded text: 3 exact, 2 prefix (LIKE on the
-- escaped text, which the trigram index serves), 1 word similarity (<%, also from the index). A term counts
-- by its best label.
WITH hits AS (
    SELECT l.term_id,
           max(CASE WHEN term_fold(l.label) = term_fold(@text::text) THEN 3
                    WHEN term_fold(l.label) LIKE term_fold(@prefix::text) || '%' THEN 2
                    ELSE 1 END)::integer AS tier,
           max(word_similarity(term_fold(@text::text), term_fold(l.label)))::real AS similarity
    FROM term_label l
    WHERE term_fold(l.label) LIKE term_fold(@prefix::text) || '%'
       OR term_fold(@text::text) <% term_fold(l.label)
    GROUP BY l.term_id
)
SELECT t.id, t.reviewed_at, t.rank, t.category, t.review_status, t.wikidata_id, t.icd10, t.ta98, t.atc, t.pcs
FROM hits h
JOIN term t ON t.id = h.term_id
WHERE sqlc.narg(category)::term_category IS NULL OR t.category = sqlc.narg(category)::term_category
ORDER BY h.tier DESC, h.similarity DESC, t.rank DESC, t.id
LIMIT @max_terms;

-- name: TermLabels :many
-- The labels of the terms in the languages, by language in the enum's order, preferred first.
SELECT term_id, language, preferred, label
FROM term_label
WHERE term_id = ANY(@term_ids::uuid[]) AND language::text = ANY(@languages::text[])
ORDER BY term_id, language, preferred DESC, label;
