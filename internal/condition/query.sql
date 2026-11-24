-- name: InsertCondition :exec
INSERT INTO condition (id, recorded_at, organization_id, patient_id, encounter_id, document_draft_id,
                       practitioner_role_id, field_key, code, verification_status)
VALUES (@id, @recorded_at, @organization_id, @patient_id, @encounter_id, @document_draft_id,
        @practitioner_role_id, @field_key, @code, @verification_status);

-- name: ListDiagnoses :many
-- A clinic's confirmed Conditions recorded in [from, to), oldest first: the columns salus_stats may read. A patient
-- who objected to the analytics, or whose record is restricted at the clinic, is left out (docs/clinic-cms/WAVE5.md S7, S9).
SELECT recorded_at, patient_id, practitioner_role_id, code
FROM condition
WHERE condition.organization_id = @organization_id AND recorded_at >= @from_at AND recorded_at < @to_at
  AND verification_status = 'confirmed'
  AND NOT analytics_left_out(condition.patient_id, condition.organization_id)
ORDER BY recorded_at, code;

-- name: ListUnconfirmed :many
SELECT id FROM condition WHERE verification_status = 'unconfirmed' ORDER BY recorded_at, id;

-- name: ConditionByID :one
SELECT id, updated_at, recorded_at, organization_id, patient_id, encounter_id, document_draft_id, practitioner_role_id,
       version_id, field_key, code, verification_status
FROM condition WHERE id = @id;

-- name: ConfirmedFields :many
-- The coded fields of the signed documents ids whose Condition is confirmed: the clinic's own (condition_clinic),
-- and a visit's shared with it, for a practitioner with an appointment with its patient there (migration 00051).
SELECT document_draft_id, field_key, code FROM condition
WHERE document_draft_id = ANY(@ids::uuid[]) AND verification_status = 'confirmed' AND organization_id = @organization_id
UNION
SELECT h.document_draft_id, h.field_key, h.code FROM history_confirmed_conditions(@ids::uuid[], @practitioner_id::uuid) AS h;
