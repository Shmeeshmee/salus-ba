-- name: InsertRequest :one
INSERT INTO data_request (account_id, patient_id, organization_id, filed_by_account_id, kind, received_at, due_at)
VALUES (@account_id, @patient_id, @organization_id, @filed_by_account_id, @kind, @received_at, @due_at)
RETURNING *;

-- name: InsertChange :exec
-- At @changed_at, or at the transaction's time for none (a withdrawal).
INSERT INTO data_request_change (data_request_id, changed_by_account_id, status, changed_at)
VALUES (@data_request_id, @changed_by_account_id, @status, COALESCE(sqlc.narg(changed_at)::timestamptz, now()));

-- name: RequestByID :one
SELECT * FROM data_request WHERE id = @id;

-- name: RequestsForAccount :many
SELECT * FROM data_request WHERE account_id = @account_id ORDER BY received_at DESC, id;

-- name: RequestsForClinic :many
-- The open ones by their deadline first, then the answered and withdrawn ones, newest first.
SELECT * FROM data_request WHERE organization_id = @organization_id
ORDER BY status IN ('received', 'in_progress') DESC,
         CASE WHEN status IN ('received', 'in_progress') THEN COALESCE(extended_until, due_at) END,
         received_at DESC, id;

-- name: RequestsForOperator :many
-- The requests to the account (salusctl, the owner): the open ones by their deadline first, then the others, newest first.
SELECT * FROM data_request WHERE organization_id IS NULL
ORDER BY status IN ('received', 'in_progress') DESC,
         CASE WHEN status IN ('received', 'in_progress') THEN COALESCE(extended_until, due_at) END,
         received_at DESC, id;

-- name: UpdateRequest :execrows
UPDATE data_request
SET status = @status, extended_until = @extended_until, extension_reason = @extension_reason,
    answered_at = @answered_at, answer_code = @answer_code, version = version + 1
WHERE id = @id AND version = @version;

-- name: RequestHistory :many
-- Oldest first; changes at the same instant in the order a request moves (received, in progress, then final).
SELECT * FROM data_request_change WHERE data_request_id = @data_request_id
ORDER BY changed_at, CASE status WHEN 'received' THEN 0 WHEN 'in_progress' THEN 1 ELSE 2 END, id;

-- name: OpenDeadlines :many
-- The deadline job (app.data_request_deadlines): the open requests of every target whose deadline is before @before.
SELECT * FROM data_request
WHERE status IN ('received', 'in_progress') AND COALESCE(extended_until, due_at) < @before
ORDER BY COALESCE(extended_until, due_at), id;

-- name: MarkWarned :execrows
-- Only if no other run marked it since @seen was read (null for none): two overlapping runs queue a warning once.
UPDATE data_request SET deadline_warned_at = @warned_at
WHERE id = @id AND status IN ('received', 'in_progress') AND deadline_warned_at IS NOT DISTINCT FROM sqlc.narg(seen)::timestamptz;
