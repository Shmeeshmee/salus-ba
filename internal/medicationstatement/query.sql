-- The patient's medications as the transaction's scope (app.patient_id or app.organization_id, 00043) shows them,
-- newest first.
-- name: MedicationsByPatient :many
SELECT * FROM medication_statement WHERE patient_id = @patient_id ORDER BY date_asserted DESC, id;

-- name: MedicationByID :one
SELECT * FROM medication_statement WHERE id = @id;

-- name: InsertMedication :exec
INSERT INTO medication_statement (id, created_at, updated_at, date_asserted, confirmed_at, patient_id, organization_id,
                                  encounter_id, recorder_role_id, confirmed_by_role_id, confirmed_organization_id, term_id,
                                  dose_value, period, frequency, in_profile, as_needed, medication_text, dose_unit,
                                  period_unit, status)
VALUES (@id, @date_asserted, @date_asserted, @date_asserted, @confirmed_at, @patient_id, @organization_id,
        @encounter_id, @recorder_role_id, @confirmed_by_role_id, @confirmed_organization_id, @term_id,
        @dose_value, @period, @frequency, @in_profile, @as_needed, @medication_text, @dose_unit,
        @period_unit, @status);

-- The next version of an entry read at version_id; no row when it changed since.
-- name: UpdateMedication :one
UPDATE medication_statement
SET updated_at = @updated_at, version_id = version_id + 1, in_profile = @in_profile, term_id = @term_id,
    medication_text = @medication_text, dose_value = @dose_value, dose_unit = @dose_unit, frequency = @frequency,
    period = @period, period_unit = @period_unit, as_needed = @as_needed, status = @status,
    confirmed_at = @confirmed_at, confirmed_by_role_id = @confirmed_by_role_id,
    confirmed_organization_id = @confirmed_organization_id
WHERE id = @id AND version_id = @version_id
RETURNING version_id;

-- name: DeleteMedication :execrows
DELETE FROM medication_statement WHERE id = @id AND version_id = @version_id;

-- name: InsertMedicationProvenance :exec
INSERT INTO medication_statement_provenance (id, recorded, medication_statement_id, practitioner_role_id, organization_id,
                                             patient_id, target_version, activity, agent_type)
VALUES (@id, @recorded, @medication_statement_id, @practitioner_role_id, @organization_id, @patient_id, @target_version,
        @activity, @agent_type);

-- A Provenance as the transaction's scope (00053) shows it.
-- name: MedicationProvenanceByID :one
SELECT * FROM medication_statement_provenance WHERE id = @id;

-- Whether the entry has a Provenance the scope sees (a patient's entry: any; 00053).
-- name: MedicationHasProvenance :one
SELECT EXISTS (SELECT 1 FROM medication_statement_provenance WHERE medication_statement_id = @id);
