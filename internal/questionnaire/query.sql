-- name: QuestionnaireByID :one
SELECT id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items
FROM questionnaire
WHERE id = @id;

-- name: LockQuestionnaire :one
-- Lifecycle steps of one version run one after the other: the second publish of a draft sees it active.
SELECT id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items
FROM questionnaire
WHERE id = @id
FOR NO KEY UPDATE;

-- name: SearchQuestionnaires :many
SELECT id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items
FROM questionnaire
WHERE (sqlc.narg(series_id)::uuid IS NULL OR series_id = sqlc.narg(series_id)::uuid)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(version)::integer IS NULL OR version = sqlc.narg(version)::integer)
ORDER BY created_at, id;

-- name: CreateQuestionnaire :one
-- Version 1 of a new series.
INSERT INTO questionnaire (series_id, organization_id, version, status, title, description, items, updated_at)
VALUES (gen_random_uuid(), @organization_id, 1, 'draft', @title, @description, @items, @updated_at)
RETURNING id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items;

-- name: CreateQuestionnaireVersion :one
-- The series' next version, a draft copy of the version from. DO NOTHING when the series has a draft (or
-- another revise took the number first) returns no row and leaves the transaction usable for the refusal's
-- audit.
INSERT INTO questionnaire (series_id, organization_id, version, status, title, description, items, updated_at)
SELECT f.series_id, f.organization_id, (SELECT max(v.version) + 1 FROM questionnaire v WHERE v.series_id = f.series_id),
       'draft', f.title, f.description, f.items, @updated_at
FROM questionnaire f
WHERE f.id = @from_id
ON CONFLICT DO NOTHING
RETURNING id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items;

-- name: UpdateQuestionnaireDraft :one
-- Optimistic locking: only the current version of a draft changes.
UPDATE questionnaire
SET title = @title, description = @description, items = @items, version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND status = 'draft' AND version_id = @version_id
RETURNING id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items;

-- name: DeleteQuestionnaireDraft :execrows
DELETE FROM questionnaire
WHERE id = @id AND status = 'draft';

-- name: PublishQuestionnaire :one
UPDATE questionnaire
SET status = 'active', published_at = @at, version_id = version_id + 1, updated_at = @at
WHERE id = @id AND status = 'draft'
RETURNING id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items;

-- name: RetireQuestionnaire :one
UPDATE questionnaire
SET status = 'retired', retired_at = @at, version_id = version_id + 1, updated_at = @at
WHERE id = @id AND status = 'active'
RETURNING id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items;

-- name: RetireActiveQuestionnaire :exec
-- Publishing a version retires the one that was active, in the same transaction.
UPDATE questionnaire
SET status = 'retired', retired_at = @at, version_id = version_id + 1, updated_at = @at
WHERE series_id = @series_id AND status = 'active';

-- name: QuestionnaireDefault :one
SELECT practitioner_role_id, updated_at, organization_id, series_id, version_id
FROM questionnaire_default
WHERE practitioner_role_id = @practitioner_role_id;

-- name: SetQuestionnaireDefault :one
INSERT INTO questionnaire_default (practitioner_role_id, organization_id, series_id, updated_at)
VALUES (@practitioner_role_id, @organization_id, @series_id, @updated_at)
ON CONFLICT (practitioner_role_id) DO UPDATE
SET series_id = excluded.series_id, version_id = questionnaire_default.version_id + 1, updated_at = excluded.updated_at
RETURNING practitioner_role_id, updated_at, organization_id, series_id, version_id;

-- name: ClearQuestionnaireDefault :exec
DELETE FROM questionnaire_default
WHERE practitioner_role_id = @practitioner_role_id;
