-- A second active consent with the clinic inserts nothing (no row), so the transaction stays usable for the
-- audit.
-- name: GrantConsent :one
INSERT INTO consent (patient_id, organization_id, status, date_time, updated_at)
VALUES (@patient_id, @organization_id, 'active', @date_time, @date_time)
ON CONFLICT (patient_id, organization_id) WHERE status = 'active' DO NOTHING
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status;

-- name: ConsentByID :one
SELECT id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status
FROM consent
WHERE id = @id;

-- Only an active consent is revoked; a concurrent second revocation finds no row.
-- name: RevokeConsent :one
UPDATE consent
SET status = 'inactive', revoked_at = @revoked_at, updated_at = @revoked_at, version_id = version_id + 1
WHERE id = @id AND status = 'active'
RETURNING id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status;

-- name: ConsentsByPatient :many
SELECT id, updated_at, date_time, revoked_at, patient_id, organization_id, version_id, status
FROM consent
WHERE patient_id = @patient_id
ORDER BY date_time DESC, id;

-- name: ConsentState :one
SELECT coalesce(bool_or(status = 'active'), false)::boolean AS active, max(revoked_at)::timestamptz AS revoked_at
FROM consent
WHERE patient_id = @patient_id AND organization_id = @organization_id;
