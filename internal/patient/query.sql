-- name: PatientByAccount :one
SELECT id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone
FROM patient
WHERE account_id = @account_id;

-- name: PatientByID :one
SELECT id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone
FROM patient
WHERE id = @id;

-- A second profile for the account inserts nothing (no row), so the transaction stays usable for the audit.
-- name: CreatePatient :one
INSERT INTO patient (account_id, given_name, family_name, birth_date, phone, updated_at)
VALUES (@account_id, @given_name, @family_name, @birth_date, sqlc.narg('phone'), @updated_at)
ON CONFLICT (account_id) DO NOTHING
RETURNING id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone;

-- Optimistic locking: only the current version changes.
-- name: UpdatePatient :one
UPDATE patient
SET given_name = @given_name, family_name = @family_name, birth_date = @birth_date, phone = sqlc.narg('phone'),
    version_id = version_id + 1, updated_at = @updated_at
WHERE account_id = @account_id AND version_id = @version_id
RETURNING id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone;
