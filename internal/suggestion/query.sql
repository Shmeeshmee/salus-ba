-- name: InsertSuggestion :execrows
-- A suggestion when first shown; one stored already inserts nothing (no row).
INSERT INTO suggestion (id, encounter_id, type, text, confidence, source_segment_ids, status)
VALUES (@id, @encounter_id, @type, @text, @confidence, @source_segment_ids, @status)
ON CONFLICT (id) DO NOTHING;

-- name: UpdateSuggestionStatus :one
-- What the practitioner did with a stored suggestion of the encounter, and whether it changed: the same status
-- again changes nothing (changed is false). No row for no such suggestion of the encounter.
WITH current AS (
    SELECT s.id FROM suggestion s WHERE s.id = @id AND s.encounter_id = @encounter_id FOR UPDATE
), updated AS (
    UPDATE suggestion u
    SET status = @status, updated_at = now(), version_id = u.version_id + 1
    FROM current
    WHERE u.id = current.id AND u.status <> @status
    RETURNING u.id
)
SELECT s.id, s.encounter_id, s.type, s.text, s.confidence, s.source_segment_ids, (SELECT count(*) FROM updated) > 0 AS changed
FROM suggestion s JOIN current ON current.id = s.id;

-- name: SuggestionsByEncounter :many
-- The encounter's suggestions, oldest first.
SELECT id, encounter_id, type, text, confidence, source_segment_ids, status
FROM suggestion
WHERE encounter_id = @encounter_id
ORDER BY created_at, id;
