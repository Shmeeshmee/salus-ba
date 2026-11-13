-- "binary" is a reserved word in Postgres and is quoted in every statement (SPEC.md §3 O15).

-- name: InsertBinary :one
-- An upload, quarantined: a questionnaire file in the patient's scope, whose insert policy checks the copied clinic,
-- patient and appointment are the response's; or a chat file in its conversation's scope, whose policy checks the
-- clinic, the patient and the uploading participant are the conversation's.
INSERT INTO "binary" (id, organization_id, patient_id, appointment_id, questionnaire_response_id, link_id, conversation_id,
                      conversation_participant_id, content_type, size, title, sha1, kek_id, wrapped_key, segment_size, status,
                      created_at, updated_at)
VALUES (@id, @organization_id, @patient_id, sqlc.narg(appointment_id), sqlc.narg(questionnaire_response_id), sqlc.narg(link_id),
        sqlc.narg(conversation_id), sqlc.narg(conversation_participant_id), @content_type, @size, @title, @sha1, @kek_id,
        @wrapped_key, @segment_size, 'quarantined', @at, @at)
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
-- The purger, as salus_retention: which of the stored objects' keys still have a row, as a file or its thumbnail.
SELECT id FROM "binary" WHERE id = ANY(@ids::uuid[])
UNION ALL
SELECT thumbnail_id FROM "binary" WHERE thumbnail_id = ANY(@ids::uuid[]);

-- name: CleanBinaries :many
-- The clean files among ids that the scope shows: the patient's timeline (SAL-39).
SELECT * FROM "binary" WHERE id = ANY(@ids::uuid[]) AND status = 'clean' ORDER BY id;

-- name: AttachChatBinaries :many
-- A send attaches the participant's files of the conversation that are neither attached nor rejected, in the
-- conversation's scope, to its message.
UPDATE "binary"
SET communication_id = @communication_id, communication_sent_at = @communication_sent_at, updated_at = @at
WHERE id = ANY(@ids::uuid[]) AND conversation_id = @conversation_id AND conversation_participant_id = @conversation_participant_id
  AND communication_id IS NULL AND status <> 'rejected'
RETURNING *;

-- name: LockChatUploads :exec
-- A participant's uploads count their files and insert one after the other, until the transaction ends.
SELECT pg_advisory_xact_lock(hashtextextended('binary-chat:' || @conversation_participant_id::text, 0));

-- name: PendingChatBinaries :one
-- The participant's files of the conversation waiting for a message: neither attached nor rejected.
SELECT count(*) FROM "binary"
WHERE conversation_id = @conversation_id AND conversation_participant_id = @conversation_participant_id
  AND communication_id IS NULL AND status <> 'rejected';

-- name: ChatBinariesOfMessages :many
-- The files of the messages, as the scope shows them, oldest first.
SELECT * FROM "binary"
WHERE conversation_id = ANY(@conversation_ids::uuid[]) AND communication_id = ANY(@communication_ids::uuid[])
ORDER BY created_at, id;

-- name: SetBinaryThumbnail :execrows
-- The scan job's thumbnail of a clean image, once.
UPDATE "binary"
SET thumbnail_id = @thumbnail_id, thumbnail_size = @thumbnail_size, thumbnail_wrapped_key = @thumbnail_wrapped_key, updated_at = @at
WHERE id = @id AND status = 'clean' AND thumbnail_id IS NULL;

-- name: DeleteUnattachedChatBinariesBefore :execrows
-- The purger, as salus_retention: chat files never attached to a message, 24 h after their upload.
DELETE FROM "binary" WHERE conversation_id IS NOT NULL AND communication_id IS NULL AND created_at < @before;
