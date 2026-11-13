-- The patient's allergies as the transaction's scope (app.patient_id or app.organization_id, 00043) shows them,
-- newest first.
-- name: AllergiesByPatient :many
SELECT * FROM allergy_intolerance WHERE patient_id = @patient_id ORDER BY recorded_date DESC, id;

-- name: AllergyByID :one
SELECT * FROM allergy_intolerance WHERE id = @id;

-- name: InsertAllergy :exec
INSERT INTO allergy_intolerance (id, created_at, updated_at, recorded_date, confirmed_at, patient_id, organization_id,
                                 encounter_id, recorder_role_id, confirmed_by_role_id, confirmed_organization_id, term_id,
                                 in_profile, code_text, category, reaction_text, severity, clinical_status, verification_status)
VALUES (@id, @recorded_date, @recorded_date, @recorded_date, @confirmed_at, @patient_id, @organization_id,
        @encounter_id, @recorder_role_id, @confirmed_by_role_id, @confirmed_organization_id, @term_id,
        @in_profile, @code_text, @category, @reaction_text, @severity, @clinical_status, @verification_status);

-- The next version of an entry read at version_id; no row when it changed since.
-- name: UpdateAllergy :one
UPDATE allergy_intolerance
SET updated_at = @updated_at, version_id = version_id + 1, in_profile = @in_profile, term_id = @term_id,
    code_text = @code_text, category = @category, reaction_text = @reaction_text, severity = @severity,
    clinical_status = @clinical_status, verification_status = @verification_status, confirmed_at = @confirmed_at,
    confirmed_by_role_id = @confirmed_by_role_id, confirmed_organization_id = @confirmed_organization_id
WHERE id = @id AND version_id = @version_id
RETURNING version_id;

-- name: DeleteAllergy :execrows
DELETE FROM allergy_intolerance WHERE id = @id AND version_id = @version_id;

-- name: InsertAllergyProvenance :exec
INSERT INTO allergy_intolerance_provenance (id, recorded, allergy_intolerance_id, practitioner_role_id, organization_id,
                                            patient_id, target_version, activity, agent_type)
VALUES (@id, @recorded, @allergy_intolerance_id, @practitioner_role_id, @organization_id, @patient_id, @target_version,
        @activity, @agent_type);

-- A Provenance as the transaction's scope (00053) shows it.
-- name: AllergyProvenanceByID :one
SELECT * FROM allergy_intolerance_provenance WHERE id = @id;

-- Whether the entry has a Provenance the scope sees (a patient's entry: any; 00053).
-- name: AllergyHasProvenance :one
SELECT EXISTS (SELECT 1 FROM allergy_intolerance_provenance WHERE allergy_intolerance_id = @id);
