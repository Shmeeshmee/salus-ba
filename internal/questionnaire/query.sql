-- name: QuestionnaireByID :one
SELECT id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items, language
FROM questionnaire
WHERE id = @id;

-- name: LockQuestionnaire :one
-- Lifecycle steps of one version run one after the other: the second publish of a draft sees it active.
SELECT id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items, language
FROM questionnaire
WHERE id = @id
FOR NO KEY UPDATE;

-- name: SearchQuestionnaires :many
SELECT id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items, language
FROM questionnaire
WHERE (sqlc.narg(series_id)::uuid IS NULL OR series_id = sqlc.narg(series_id)::uuid)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(version)::integer IS NULL OR version = sqlc.narg(version)::integer)
ORDER BY created_at, id;

-- name: CreateQuestionnaire :one
-- Version 1 of a new series.
-- No language given (before the service passes the clinic's default) is the column's default, bs.
INSERT INTO questionnaire (series_id, organization_id, version, status, title, description, items, language, updated_at)
VALUES (gen_random_uuid(), @organization_id, 1, 'draft', @title, @description, @items,
        COALESCE(sqlc.narg('language')::language, 'bs'), @updated_at)
RETURNING id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items, language;

-- name: CreateQuestionnaireVersion :one
-- The series' next version, a draft copy of the version from. DO NOTHING when the series has a draft (or
-- another revise took the number first) returns no row and leaves the transaction usable for the refusal's
-- audit.
INSERT INTO questionnaire (series_id, organization_id, version, status, title, description, items, language, updated_at)
SELECT f.series_id, f.organization_id, (SELECT max(v.version) + 1 FROM questionnaire v WHERE v.series_id = f.series_id),
       'draft', f.title, f.description, f.items, f.language, @updated_at
FROM questionnaire f
WHERE f.id = @from_id
ON CONFLICT DO NOTHING
RETURNING id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items, language;

-- name: UpdateQuestionnaireDraft :one
-- Optimistic locking: only the current version of a draft changes. No language given keeps the draft's.
UPDATE questionnaire
SET title = @title, description = @description, items = @items,
    language = COALESCE(sqlc.narg('language')::language, language), version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND status = 'draft' AND version_id = @version_id
RETURNING id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items, language;

-- name: DeleteQuestionnaireDraft :execrows
DELETE FROM questionnaire
WHERE id = @id AND status = 'draft';

-- name: PublishQuestionnaire :one
UPDATE questionnaire
SET status = 'active', published_at = @at, version_id = version_id + 1, updated_at = @at
WHERE id = @id AND status = 'draft'
RETURNING id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items, language;

-- name: RetireQuestionnaire :one
UPDATE questionnaire
SET status = 'retired', retired_at = @at, version_id = version_id + 1, updated_at = @at
WHERE id = @id AND status = 'active'
RETURNING id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items, language;

-- name: RetireActiveQuestionnaire :exec
-- Publishing a version retires the one that was active, in the same transaction.
UPDATE questionnaire
SET status = 'retired', retired_at = @at, version_id = version_id + 1, updated_at = @at
WHERE series_id = @series_id AND status = 'active';

-- name: QuestionnaireTranslations :many
SELECT language, texts
FROM questionnaire_translation
WHERE questionnaire_id = @questionnaire_id
ORDER BY language;

-- name: DeleteQuestionnaireTranslations :exec
-- Saving a draft's translations replaces them all; row-level security lets only a draft's go.
DELETE FROM questionnaire_translation
WHERE questionnaire_id = @questionnaire_id;

-- name: InsertQuestionnaireTranslation :exec
INSERT INTO questionnaire_translation (questionnaire_id, language, texts, created_at, updated_at)
VALUES (@questionnaire_id, @language, @texts, @at, @at);

-- name: CopyQuestionnaireTranslations :exec
-- A new version starts with the translations of the version it copies.
INSERT INTO questionnaire_translation (questionnaire_id, language, texts, created_at, updated_at)
SELECT @to_id::uuid, t.language, t.texts, @at, @at
FROM questionnaire_translation t
WHERE t.questionnaire_id = @from_id::uuid;

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

-- name: SharedQuestionnaire :one
-- A version another clinic's completed response answers, of a visit shared with the scope's clinic (migration 00050).
SELECT id, updated_at, published_at, retired_at, series_id, organization_id, version, version_id, status, title, description, items, language
FROM history_questionnaire(@id::uuid);
