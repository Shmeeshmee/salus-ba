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
  AND NOT EXISTS (SELECT 1 FROM patient_restriction r WHERE r.patient_id = p.id AND r.organization_id = @organization_id)
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

-- The ids of the patients the clinic registered (its data export, docs/clinic-cms M2 joined in wave 4). Row-level
-- security limits the rows to the scope too.
-- name: RegisteredByClinic :many
SELECT id FROM patient WHERE registered_by_organization_id = @organization_id ORDER BY id;

-- The erasure's restriction of the record a clinic keeps (docs/clinic-cms/WAVE5.md S6, D3): by the job, in the clinic's
-- scope with the erasure scope (00075), after any staff restriction there is lifted (UnrestrictPatient), so the job's
-- replaces it.
-- name: RestrictForErasure :exec
INSERT INTO patient_restriction (restricted_at, patient_id, organization_id)
VALUES (@restricted_at, @patient_id, @organization_id)
ON CONFLICT (patient_id, organization_id) DO NOTHING;
-- A restriction of the patient's record at the clinic (docs/clinic-cms/WAVE5.md S7), by its staff; nothing changes
-- for a record restricted already. Row-level security admits only the clinic's scope and its own staff and requests.
-- A request it names must be the clinic's and the patient's (as row-level security checks; tested first so a refusal
-- inserts nothing instead of failing the transaction).
-- name: RestrictPatient :execrows
INSERT INTO patient_restriction (restricted_at, patient_id, organization_id, restricted_by_account_id, data_request_id)
SELECT @restricted_at, @patient_id, @organization_id, @restricted_by_account_id, sqlc.narg('data_request_id')::uuid
WHERE sqlc.narg('data_request_id')::uuid IS NULL
   OR EXISTS (SELECT 1 FROM data_request r WHERE r.id = sqlc.narg('data_request_id')::uuid
              AND r.organization_id = @organization_id AND r.patient_id = @patient_id)
ON CONFLICT (patient_id, organization_id) DO NOTHING;

-- Lifts a staff restriction; the erasure's (by nobody) stays, by row-level security and here.
-- name: UnrestrictPatient :exec
DELETE FROM patient_restriction
WHERE patient_id = @patient_id AND organization_id = @organization_id AND restricted_by_account_id IS NOT NULL;

-- The patients whose record the clinic restricts, for its data export's manifest (D4); in the clinic's scope.
-- name: RestrictedAtClinic :many
SELECT patient_id FROM patient_restriction WHERE organization_id = @organization_id ORDER BY patient_id;

-- Read from any scope (00078): the worker's jobs and another clinic's guest practitioner see the treating clinic's.
-- name: PatientRestricted :one
SELECT patient_record_restricted(@patient_id::uuid, @organization_id::uuid)::boolean AS restricted;
