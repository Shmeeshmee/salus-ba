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

-- name: MarkBinaryScanned :execrows
-- The scan job's result, in the job scope, once: clean, or rejected with the wrapped key cleared (the bytes can
-- no longer be read by anyone).
UPDATE "binary"
SET status = @status::text, scanned_at = @at, updated_at = @at,
    wrapped_key = CASE WHEN @status::text = 'rejected' THEN NULL ELSE wrapped_key END
WHERE id = @id AND status = 'quarantined';

-- name: BinariesOfResponse :many
-- Every file of a response, for the discard job.
SELECT * FROM "binary" WHERE questionnaire_response_id = @questionnaire_response_id ORDER BY created_at, id;

-- name: DeleteRejectedBinariesBefore :execrows
-- The purger, as salus_retention: rejected files 24 h after their scan (binary_rejected_scanned).
DELETE FROM "binary" WHERE status = 'rejected' AND scanned_at < @before;

-- name: ExistingBinaries :many
-- The purger, as salus_retention: which of the stored objects' keys still have a row.
SELECT id FROM "binary" WHERE id = ANY(@ids::uuid[]);
