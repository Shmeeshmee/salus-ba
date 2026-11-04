-- name: PractitionerByID :one
SELECT id, updated_at, organization_id, version_id, active, given_name, family_name
FROM practitioner
WHERE id = @id;

-- name: CreatePractitioner :one
INSERT INTO practitioner (organization_id, given_name, family_name)
VALUES (@organization_id, @given_name, @family_name)
RETURNING id, updated_at, organization_id, version_id, active, given_name, family_name;
