-- name: InsertSuggestion :execrows
-- A suggestion when first shown; one stored already inserts nothing (no row).
INSERT INTO suggestion (id, encounter_id, type, text, confidence, source_segment_ids, status)
VALUES (@id, @encounter_id, @type, @text, @confidence, @source_segment_ids, @status)
ON CONFLICT (id) DO NOTHING;

-- name: UpdateSuggestionStatus :execrows
-- What the practitioner did with a stored suggestion of the encounter; the same status again changes nothing.
UPDATE suggestion
SET status = @status, updated_at = now(), version_id = version_id + 1
WHERE id = @id AND encounter_id = @encounter_id AND status <> @status;
