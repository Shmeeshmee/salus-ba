-- name: InsertQuestionnaireResponse :one
-- A request, in the clinic's scope: in progress, no answers. DO NOTHING when the appointment has a live request
-- of the series returns no row and leaves the transaction usable for the refusal's audit.
INSERT INTO questionnaire_response (organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id, series_id,
                                    source, status, due_at, created_at, updated_at)
VALUES (@organization_id, @appointment_id, @patient_id, @practitioner_role_id, @questionnaire_id, @series_id,
        @source, 'in-progress', @due_at, @at, @at)
ON CONFLICT (appointment_id, series_id) WHERE status <> 'stopped' DO NOTHING
RETURNING id, created_at, updated_at, due_at, authored_at, organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id,
          series_id, version_id, source, status, answers, attachments;

-- name: QuestionnaireResponseByID :one
SELECT id, created_at, updated_at, due_at, authored_at, organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id,
       series_id, version_id, source, status, answers, attachments
FROM questionnaire_response
WHERE id = @id;

-- name: LockQuestionnaireResponse :one
-- Changes to one response run one after the other: a withdrawal waits for a submit, and sees it.
SELECT id, created_at, updated_at, due_at, authored_at, organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id,
       series_id, version_id, source, status, answers, attachments
FROM questionnaire_response
WHERE id = @id
FOR NO KEY UPDATE;

-- name: StopQuestionnaireResponse :one
-- A withdrawal: stopped, the answers cleared at once.
UPDATE questionnaire_response
SET status = 'stopped', answers = '[]', updated_at = @at, version_id = version_id + 1
WHERE id = @id
RETURNING id, created_at, updated_at, due_at, authored_at, organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id,
          series_id, version_id, source, status, answers, attachments;

-- name: SaveQuestionnaireResponseAnswers :one
-- The patient's save: the answers replaced, while in progress.
UPDATE questionnaire_response
SET answers = @answers, updated_at = @at, version_id = version_id + 1
WHERE id = @id AND status = 'in-progress'
RETURNING id, created_at, updated_at, due_at, authored_at, organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id,
          series_id, version_id, source, status, answers, attachments;

-- name: CompleteQuestionnaireResponse :one
-- The patient's submit: completed with the answers kept, authored now; final.
UPDATE questionnaire_response
SET status = 'completed', answers = @answers, authored_at = @at, updated_at = @at, version_id = version_id + 1
WHERE id = @id AND status = 'in-progress'
RETURNING id, created_at, updated_at, due_at, authored_at, organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id,
          series_id, version_id, source, status, answers, attachments;

-- name: SearchQuestionnaireResponses :many
-- QuestionnaireResponse?appointment=…&patient=…&status=…; an absent parameter matches everything, the scope
-- decides the rest.
SELECT id, created_at, updated_at, due_at, authored_at, organization_id, appointment_id, patient_id, practitioner_role_id, questionnaire_id,
       series_id, version_id, source, status, answers, attachments
FROM questionnaire_response
WHERE (sqlc.narg(appointment_id)::uuid IS NULL OR appointment_id = sqlc.narg(appointment_id)::uuid)
  AND (sqlc.narg(patient_id)::uuid IS NULL OR patient_id = sqlc.narg(patient_id)::uuid)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
ORDER BY due_at, created_at, id;

-- name: DeleteUnsubmittedQuestionnaireResponses :execrows
-- The retention purger, as salus_retention in every clinic: the responses never submitted once due.
DELETE FROM questionnaire_response
WHERE status <> 'completed' AND due_at < @before;

-- name: DeleteQuestionnaireResponsesOfAppointment :many
-- The discard job, in the clinic's scope with the appointment's notice: every response of the appointment; the
-- discard policy lets it delete them only once the appointment is cancelled.
DELETE FROM questionnaire_response
WHERE appointment_id = @appointment_id
RETURNING id;

-- name: AddQuestionnaireResponseAttachment :execrows
-- binary's upload, in its transaction and the patient's scope: the file appended to the answer's files. Not
-- version_id, which guards the typed answers only (docs/attachments/SPEC.md §2 "Where an attachment answer
-- lives").
UPDATE questionnaire_response
SET attachments = attachments || jsonb_build_array(@attachment::jsonb), updated_at = @at
WHERE id = @id;

-- name: RemoveQuestionnaireResponseAttachment :execrows
-- binary's deletion and rejection: the file taken out of the answer's files, the others kept in order.
UPDATE questionnaire_response
SET attachments = coalesce((SELECT jsonb_agg(a.e ORDER BY a.n) FROM jsonb_array_elements(attachments) WITH ORDINALITY AS a(e, n)
                            WHERE a.e->>'binaryId' <> @binary_id::text), '[]'),
    updated_at = @at
WHERE id = @id AND attachments @> jsonb_build_array(jsonb_build_object('binaryId', @binary_id::text));

-- name: ClearQuestionnaireResponseAttachment :execrows
-- binary's scan of a clean file: the file no longer pending.
UPDATE questionnaire_response
SET attachments = (SELECT jsonb_agg(CASE WHEN a.e->>'binaryId' = @binary_id::text THEN a.e || '{"pending": false}' ELSE a.e END ORDER BY a.n)
                   FROM jsonb_array_elements(attachments) WITH ORDINALITY AS a(e, n)),
    updated_at = @at
WHERE id = @id AND attachments @> jsonb_build_array(jsonb_build_object('binaryId', @binary_id::text, 'pending', true));

-- name: QuestionnaireResponseAttachments :one
SELECT attachments FROM questionnaire_response WHERE id = @id;
