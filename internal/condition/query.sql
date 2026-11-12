-- name: InsertCondition :exec
INSERT INTO condition (id, recorded_at, organization_id, patient_id, encounter_id, document_draft_id,
                       practitioner_role_id, field_key, code, verification_status)
VALUES (@id, @recorded_at, @organization_id, @patient_id, @encounter_id, @document_draft_id,
        @practitioner_role_id, @field_key, @code, @verification_status);

-- name: ListDiagnoses :many
-- A clinic's confirmed Conditions recorded in [from, to), oldest first: the columns salus_stats may read.
SELECT recorded_at, patient_id, practitioner_role_id, code
FROM condition
WHERE organization_id = @organization_id AND recorded_at >= @from_at AND recorded_at < @to_at
  AND verification_status = 'confirmed'
ORDER BY recorded_at, code;

-- name: ListUnconfirmed :many
SELECT id FROM condition WHERE verification_status = 'unconfirmed' ORDER BY recorded_at, id;
