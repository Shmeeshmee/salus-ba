-- name: OrganizationByID :one
SELECT id, updated_at, version_id, active, listed, name, time_zone, default_language, patient_transcript_copy
FROM organization
WHERE id = @id;

-- name: ListedOrganizations :many
-- Organization?listed=true: row-level security shows only listed clinics in the directory scope.
SELECT id, updated_at, version_id, active, listed, name, time_zone, default_language, patient_transcript_copy
FROM organization
WHERE listed
ORDER BY name, id;

-- name: UpdateOrganization :one
-- Optimistic locking: only the current version changes.
UPDATE organization
SET name = @name, listed = @listed, time_zone = @time_zone, default_language = @default_language, patient_transcript_copy = @patient_transcript_copy, version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND version_id = @version_id
RETURNING id, updated_at, version_id, active, listed, name, time_zone, default_language, patient_transcript_copy;

-- name: CreateOrganization :one
-- The operator command only (salusctl, as the owner): salus_app has no INSERT on organization.
INSERT INTO organization (name) VALUES (@name)
RETURNING id, updated_at, version_id, active, listed, name, time_zone, default_language, patient_transcript_copy;
