-- name: PractitionerByID :one
SELECT id, updated_at, organization_id, version_id, active, given_name, family_name
FROM practitioner
WHERE id = @id;
