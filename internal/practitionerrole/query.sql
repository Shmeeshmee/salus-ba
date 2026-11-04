-- name: PractitionerRoleByID :one
SELECT id, updated_at, organization_id, practitioner_id, version_id, active, specialty_code
FROM practitioner_role
WHERE id = @id;

-- name: PractitionerRolesByOrganization :many
-- PractitionerRole?organization=…, oldest first so a search answers in a stable order.
SELECT id, updated_at, organization_id, practitioner_id, version_id, active, specialty_code
FROM practitioner_role
WHERE organization_id = @organization_id
ORDER BY created_at, id;

-- name: PublicPractitionerRolesByOrganization :many
-- The public directory: the active roles of a clinic, which row-level security shows only when it is listed.
SELECT id, updated_at, organization_id, practitioner_id, version_id, active, specialty_code
FROM practitioner_role
WHERE organization_id = @organization_id AND active
ORDER BY created_at, id;

-- name: CreatePractitionerRole :one
INSERT INTO practitioner_role (organization_id, practitioner_id)
VALUES (@organization_id, @practitioner_id)
RETURNING id, updated_at, organization_id, practitioner_id, version_id, active, specialty_code;

-- name: SetPractitionerRoleSpecialty :one
-- Optimistic locking: only the current version changes.
UPDATE practitioner_role
SET specialty_code = @specialty_code, version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND version_id = @version_id
RETURNING id, updated_at, organization_id, practitioner_id, version_id, active, specialty_code;
