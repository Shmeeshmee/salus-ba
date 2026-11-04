-- name: PractitionerRoleByID :one
SELECT id, updated_at, organization_id, practitioner_id, version_id, active
FROM practitioner_role
WHERE id = @id;

-- name: PractitionerRolesByOrganization :many
-- PractitionerRole?organization=…, oldest first so a search answers in a stable order.
SELECT id, updated_at, organization_id, practitioner_id, version_id, active
FROM practitioner_role
WHERE organization_id = @organization_id
ORDER BY created_at, id;

-- name: CreatePractitionerRole :one
INSERT INTO practitioner_role (organization_id, practitioner_id)
VALUES (@organization_id, @practitioner_id)
RETURNING id, updated_at, organization_id, practitioner_id, version_id, active;
