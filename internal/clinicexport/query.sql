-- name: InsertExport :one
-- One queued export per clinic: a second ask while one is queued inserts nothing (the caller then reads the queued one).
INSERT INTO clinic_export (organization_id, requested_by_account_id, requested_at, expires_at)
VALUES (@organization_id, @requested_by_account_id, @requested_at, @expires_at)
ON CONFLICT (organization_id) WHERE status = 'queued' DO NOTHING
RETURNING *;

-- name: QueuedExport :one
SELECT * FROM clinic_export WHERE organization_id = @organization_id AND status = 'queued';

-- name: ExportByID :one
SELECT * FROM clinic_export WHERE id = @id;

-- name: MarkExportReady :execrows
UPDATE clinic_export SET status = 'ready', ready_at = @ready_at, size = @size, kek_id = @kek_id, wrapped_key = @wrapped_key
WHERE id = @id AND status = 'queued';

-- name: MarkExportFailed :execrows
UPDATE clinic_export SET status = 'failed' WHERE id = @id AND status = 'queued';

-- name: ExportsRequestedBefore :many
-- The retention job (salus_retention): the exports asked for before @before, in every clinic.
SELECT id FROM clinic_export WHERE requested_at < @before ORDER BY requested_at LIMIT @max;

-- name: DeleteExport :execrows
DELETE FROM clinic_export WHERE id = @id;
