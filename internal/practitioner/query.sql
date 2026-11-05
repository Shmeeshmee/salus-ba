-- name: PractitionerByID :one
SELECT id, updated_at, organization_id, version_id, active, given_name, family_name
FROM practitioner
WHERE id = @id;

-- name: CreatePractitioner :one
INSERT INTO practitioner (organization_id, given_name, family_name)
VALUES (@organization_id, @given_name, @family_name)
RETURNING id, updated_at, organization_id, version_id, active, given_name, family_name;

-- name: PublicPractitionersByIDs :many
-- The public directory: the name of each active practitioner, which row-level security shows only for a
-- listed clinic. Not the clinic or anything else.
SELECT id, updated_at, version_id, active, given_name, family_name
FROM practitioner
WHERE id = ANY(@ids::uuid[]) AND active
ORDER BY id;
