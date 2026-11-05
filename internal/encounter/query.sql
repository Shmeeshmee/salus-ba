-- name: StartEncounter :one
-- The practitioner's first join, in the clinic's scope; a concurrent start for the appointment waits for this
-- one's commit and then inserts nothing (no row).
INSERT INTO encounter (appointment_id, organization_id, patient_id, practitioner_role_id, status, period_start, created_at, updated_at)
VALUES (@appointment_id, @organization_id, @patient_id, @practitioner_role_id, 'in-progress', @period_start, @period_start, @period_start)
ON CONFLICT (appointment_id) DO NOTHING
RETURNING id, created_at, updated_at, period_start, period_end, appointment_id, organization_id, patient_id, practitioner_role_id,
          version_id, status;

-- name: EncounterByAppointment :one
SELECT id, created_at, updated_at, period_start, period_end, appointment_id, organization_id, patient_id, practitioner_role_id,
       version_id, status
FROM encounter
WHERE appointment_id = @appointment_id;

-- name: EncounterByID :one
SELECT id, created_at, updated_at, period_start, period_end, appointment_id, organization_id, patient_id, practitioner_role_id,
       version_id, status
FROM encounter
WHERE id = @id;

-- name: FinishEncounter :one
-- Only an encounter in progress is finished: a finished one never changes again, and a repeat leaves no row.
UPDATE encounter
SET status = 'finished', period_end = @at, updated_at = @at, version_id = version_id + 1
WHERE appointment_id = @appointment_id AND status = 'in-progress'
RETURNING id, created_at, updated_at, period_start, period_end, appointment_id, organization_id, patient_id, practitioner_role_id,
          version_id, status;

-- name: SearchEncounters :many
-- Encounter?appointment=…&patient=…; an absent parameter matches everything, the scope decides the rest.
SELECT id, created_at, updated_at, period_start, period_end, appointment_id, organization_id, patient_id, practitioner_role_id,
       version_id, status
FROM encounter
WHERE (sqlc.narg(appointment_id)::uuid IS NULL OR appointment_id = sqlc.narg(appointment_id)::uuid)
  AND (sqlc.narg(patient_id)::uuid IS NULL OR patient_id = sqlc.narg(patient_id)::uuid)
ORDER BY period_start, id;
