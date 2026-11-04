-- name: OrganizationByID :one
SELECT id, updated_at, version_id, active, name
FROM organization
WHERE id = @id;

-- name: RenameOrganization :one
-- Optimistic locking: only the current version changes.
UPDATE organization
SET name = @name, version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND version_id = @version_id
RETURNING id, updated_at, version_id, active, name;

-- name: CreateOrganization :one
-- The operator command only (salusctl, as the owner): salus_app has no INSERT on organization.
INSERT INTO organization (name) VALUES (@name)
RETURNING id, updated_at, version_id, active, name;
