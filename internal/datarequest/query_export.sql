-- The patient's export (package D2, WAVE5.md S4), always in the patient's scope (app.account_id).

-- name: InsertExport :one
-- One queued export per patient: a second while one is queued inserts nothing (the caller reports it in progress).
INSERT INTO data_request_export (data_request_id, account_id, requested_at)
VALUES (@data_request_id, @account_id, @requested_at)
ON CONFLICT (account_id) WHERE status = 'queued' DO NOTHING
RETURNING *;

-- name: ExportOf :one
-- The export of the request, with the request's kind; the request is read through its own row-level security.
SELECT x.*, r.kind, r.organization_id FROM data_request_export x JOIN data_request r ON r.id = x.data_request_id
WHERE x.data_request_id = @data_request_id;

-- name: ClaimExport :one
-- A build attempt claims the queued export's object key (D7) and gets the key it replaces, whose object it deletes.
UPDATE data_request_export n SET object_key = @object_key
FROM (SELECT q.data_request_id, q.object_key FROM data_request_export q
      WHERE q.data_request_id = @data_request_id AND q.status = 'queued' FOR UPDATE) o
WHERE n.data_request_id = o.data_request_id
RETURNING o.object_key AS previous;

-- name: MarkExportReady :execrows
-- Only the attempt whose claim is still the row's: of two overlapping attempts, the later claim wins.
UPDATE data_request_export SET status = 'ready', ready_at = @ready_at, expires_at = @expires_at, size = @size,
    kek_id = @kek_id, wrapped_key = @wrapped_key
WHERE data_request_id = @data_request_id AND status = 'queued' AND object_key = @object_key
  AND EXISTS (SELECT 1 FROM data_request r WHERE r.id = @data_request_id AND r.status IN ('received', 'in_progress'));

-- name: MarkExportFailed :execrows
UPDATE data_request_export SET status = 'failed' WHERE data_request_id = @data_request_id AND status = 'queued';

-- name: MarkClaimFailed :execrows
-- A build attempt whose request closed meanwhile fails the export only while its claim is still the row's (D7).
UPDATE data_request_export SET status = 'failed'
WHERE data_request_id = @data_request_id AND status = 'queued' AND object_key = @object_key;

-- name: FulfilExportRequest :one
SELECT data_request_export_fulfil(@data_request_id::uuid)::boolean AS done;

-- name: ExportsDoneBefore :many
-- The retention job (salus_retention): the exports ready before @before, and those asked for before it that never
-- became ready (failed, or a build that never finished).
SELECT data_request_id, object_key FROM data_request_export
WHERE ready_at < @before OR status <> 'ready' AND requested_at < @before
ORDER BY requested_at LIMIT @max;

-- name: DeleteExport :execrows
DELETE FROM data_request_export WHERE data_request_id = @data_request_id;
