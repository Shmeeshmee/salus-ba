-- name: InsertSegment :execrows
-- A final segment under the id of its live events; one delivered again inserts nothing (no row).
INSERT INTO transcript_segment (id, encounter_id, speaker, lang, original, translation, translation_failed, ts_start, ts_end)
VALUES (@id, @encounter_id, @speaker, @lang, @original, @translation, @translation_failed, @ts_start, @ts_end)
ON CONFLICT (id) DO NOTHING;

-- name: SegmentsByEncounter :many
-- An encounter's transcript in the order it was said; the scope decides whether it is seen.
SELECT id, encounter_id, speaker, lang, original, translation, translation_failed, ts_start, ts_end
FROM transcript_segment
WHERE encounter_id = @encounter_id
ORDER BY ts_start, id;
