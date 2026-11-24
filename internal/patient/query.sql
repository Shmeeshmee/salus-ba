-- name: PatientByAccount :one
SELECT id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone, registered_by_organization_id
FROM patient
WHERE account_id = @account_id;

-- name: PatientByID :one
SELECT id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone, registered_by_organization_id
FROM patient
WHERE id = @id;

-- A second profile for the account inserts nothing (no row), so the transaction stays usable for the audit.
-- name: CreatePatient :one
INSERT INTO patient (account_id, given_name, family_name, birth_date, phone, updated_at)
VALUES (@account_id, @given_name, @family_name, @birth_date, sqlc.narg('phone'), @updated_at)
ON CONFLICT (account_id) DO NOTHING
RETURNING id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone, registered_by_organization_id;

-- Optimistic locking: only the current version changes.
-- name: UpdatePatient :one
UPDATE patient
SET given_name = @given_name, family_name = @family_name, birth_date = @birth_date, phone = sqlc.narg('phone'),
    version_id = version_id + 1, updated_at = @updated_at
WHERE account_id = @account_id AND version_id = @version_id
RETURNING id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone, registered_by_organization_id;

-- A patient without an account, registered by the clinic of the transaction's scope (docs/clinic-cms F2, W1).
-- name: RegisterPatient :one
INSERT INTO patient (registered_by_organization_id, given_name, family_name, birth_date, phone, updated_at)
VALUES (@organization_id, @given_name, @family_name, @birth_date, sqlc.narg('phone'), @updated_at)
RETURNING id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone, registered_by_organization_id;

-- The clinic's patients: those it registered and those with its active care consent, whose given or family name,
-- or both in either order, contain the pattern (already escaped for LIKE), on the birth date when given, and, for a
-- practitioner's role, with an appointment with that role. Row-level security limits the rows to the scope too.
-- name: SearchClinicPatients :many
SELECT id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone, registered_by_organization_id
FROM patient p
WHERE (p.registered_by_organization_id = @organization_id
       OR EXISTS (SELECT 1 FROM consent c
                  WHERE c.patient_id = p.id AND c.organization_id = @organization_id AND c.status = 'active'))
  AND (p.given_name || ' ' || p.family_name ILIKE @pattern OR p.family_name || ' ' || p.given_name ILIKE @pattern)
  AND (sqlc.narg('birth_date')::date IS NULL OR p.birth_date = sqlc.narg('birth_date')::date)
  AND (sqlc.narg('practitioner_role_id')::uuid IS NULL
       OR EXISTS (SELECT 1 FROM appointment a
                  WHERE a.patient_id = p.id AND a.organization_id = @organization_id
                    AND a.practitioner_role_id = sqlc.narg('practitioner_role_id')::uuid))
ORDER BY p.family_name, p.given_name, p.id
LIMIT @max_rows;

-- The clinic corrects a profile it registered (docs/clinic-cms R2): only the current version, only its own.
-- name: CorrectRegisteredPatient :one
UPDATE patient
SET given_name = @given_name, family_name = @family_name, birth_date = @birth_date, phone = sqlc.narg('phone'),
    version_id = version_id + 1, updated_at = @updated_at
WHERE id = @id AND registered_by_organization_id = @organization_id AND version_id = @version_id
RETURNING id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone, registered_by_organization_id;

-- The clinic's patients (those it registered and those with its active care consent) whose lower-case given name,
-- family name and birth date, joined by the unit separator (chr 31), are one of the keys (the people of an import,
-- docs/clinic-cms M3). Row-level security limits the rows to the scope too.
-- name: MatchingClinicPatients :many
SELECT id, updated_at, account_id, version_id, birth_date, given_name, family_name, phone, registered_by_organization_id
FROM patient p
WHERE (p.registered_by_organization_id = @organization_id
       OR EXISTS (SELECT 1 FROM consent c
                  WHERE c.patient_id = p.id AND c.organization_id = @organization_id AND c.status = 'active'))
  AND lower(p.given_name) || chr(31) || lower(p.family_name) || chr(31) || to_char(p.birth_date, 'YYYY-MM-DD') = ANY(@keys::text[])
ORDER BY p.id;
