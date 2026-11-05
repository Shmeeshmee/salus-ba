-- name: ScheduleByID :one
SELECT id, updated_at, organization_id, practitioner_role_id, version_id, slot_minutes, availability
FROM schedule
WHERE id = @id;

-- name: ScheduleByRole :one
SELECT id, updated_at, organization_id, practitioner_role_id, version_id, slot_minutes, availability
FROM schedule
WHERE practitioner_role_id = @practitioner_role_id;

-- name: LockSchedule :one
-- slots_generate: generations of one schedule, and its edits, run one after the other.
SELECT id, updated_at, organization_id, practitioner_role_id, version_id, slot_minutes, availability
FROM schedule
WHERE id = @id
FOR NO KEY UPDATE;

-- name: ListedSchedules :many
-- The hourly slot generation, in the directory scope: row-level security shows the schedules of active roles
-- of listed clinics.
SELECT id, updated_at, organization_id, practitioner_role_id, version_id, slot_minutes, availability
FROM schedule
ORDER BY created_at, id;

-- name: CreateSchedule :one
-- DO NOTHING when the role has one returns no row and leaves the transaction usable for the refusal's audit.
INSERT INTO schedule (organization_id, practitioner_role_id, slot_minutes, availability)
VALUES (@organization_id, @practitioner_role_id, @slot_minutes, @availability)
ON CONFLICT (practitioner_role_id) DO NOTHING
RETURNING id, updated_at, organization_id, practitioner_role_id, version_id, slot_minutes, availability;

-- name: UpdateSchedule :one
-- Optimistic locking: only the current version changes.
UPDATE schedule
SET slot_minutes = @slot_minutes, availability = @availability, version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND version_id = @version_id
RETURNING id, updated_at, organization_id, practitioner_role_id, version_id, slot_minutes, availability;
