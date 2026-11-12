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

-- name: EncounterOrganization :one
-- The clinic of an encounter whatever the scope (migration 00033), null for an unknown id.
SELECT encounter_organization(@id::uuid)::uuid AS organization_id;

-- name: PatientHistoryEncounters :many
-- The patient's timeline (SAL-39): their encounters after the page's position, newest first by start, the
-- reference ("Encounter/<id>", compared byte by byte) breaking a tie; a null bound does not narrow.
SELECT id, created_at, updated_at, period_start, period_end, appointment_id, organization_id, patient_id, practitioner_role_id,
       version_id, status
FROM encounter
WHERE patient_id = @patient_id
  AND (sqlc.narg(organization_id)::uuid IS NULL OR organization_id = sqlc.narg(organization_id)::uuid)
  AND (sqlc.narg(from_at)::timestamptz IS NULL OR period_start >= sqlc.narg(from_at)::timestamptz)
  AND (sqlc.narg(to_at)::timestamptz IS NULL OR period_start < sqlc.narg(to_at)::timestamptz)
  AND (sqlc.narg(after_at)::timestamptz IS NULL
       OR period_start < sqlc.narg(after_at)::timestamptz
       OR (period_start = sqlc.narg(after_at)::timestamptz
           AND ('Encounter/' || id::text) COLLATE "C" < sqlc.arg(after_ref)::text COLLATE "C"))
ORDER BY period_start DESC, id DESC
LIMIT @lim;

-- name: EncountersOfPatient :many
-- The patient's encounters the scope shows, oldest first: the visits whose signed documents a patient's timeline
-- lists (SAL-39). One fixed condition, so its prepared plan is kept, where SearchEncounters is planned again for
-- every set of absent parameters.
SELECT id, created_at, updated_at, period_start, period_end, appointment_id, organization_id, patient_id, practitioner_role_id,
       version_id, status
FROM encounter
WHERE patient_id = @patient_id
ORDER BY period_start, id;
