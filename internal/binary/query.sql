-- "binary" is a reserved word in Postgres and is quoted in every statement (SPEC.md §3 O15).

-- name: InsertBinary :one
-- An upload, in the patient's scope: quarantined; the insert policy checks the copied clinic, patient and
-- appointment are the response's.
INSERT INTO "binary" (id, organization_id, patient_id, appointment_id, questionnaire_response_id, link_id, content_type, size,
                      title, sha1, kek_id, wrapped_key, segment_size, status, created_at, updated_at)
VALUES (@id, @organization_id, @patient_id, @appointment_id, @questionnaire_response_id, @link_id, @content_type, @size,
        @title, @sha1, @kek_id, @wrapped_key, @segment_size, 'quarantined', @at, @at)
RETURNING *;

-- name: BinaryByID :one
SELECT * FROM "binary" WHERE id = @id;

-- name: BinariesOfItem :many
-- An item's files that count: every one but the rejected, oldest first.
SELECT * FROM "binary"
WHERE questionnaire_response_id = @questionnaire_response_id AND link_id = @link_id AND status <> 'rejected'
ORDER BY created_at, id;

-- name: ResponseBinarySize :one
-- The plaintext bytes of a response's files that count.
SELECT coalesce(sum(size), 0)::bigint FROM "binary"
WHERE questionnaire_response_id = @questionnaire_response_id AND status <> 'rejected';

-- name: LockResponseBinaries :exec
-- Uploads to one response run their checks and inserts one after the other, until the transaction ends.
SELECT pg_advisory_xact_lock(hashtextextended('binary:' || @questionnaire_response_id::text, 0));

-- name: DeleteBinary :execrows
DELETE FROM "binary" WHERE id = @id;
