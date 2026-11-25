-- name: InsertPlan :one
INSERT INTO care_plan (organization_id, patient_id, created_by_account_id, title, quote_id)
VALUES (sqlc.arg(organization_id)::text::uuid, sqlc.arg(patient_id)::text::uuid, sqlc.arg(created_by)::text::uuid,
        sqlc.arg(title), NULLIF(sqlc.arg(quote_id)::text, '')::uuid)
RETURNING id::text AS id, created_at;

-- name: UpdatePlan :execrows
UPDATE care_plan
SET status = sqlc.arg(status), title = sqlc.arg(title), quote_id = NULLIF(sqlc.arg(quote_id)::text, '')::uuid,
    starts_on = sqlc.narg(starts_on)::date, booked_at = sqlc.narg(booked_at)::timestamptz,
    revoked_at = sqlc.narg(revoked_at)::timestamptz,
    revoked_by_account_id = CASE WHEN sqlc.arg(status) = 'revoked' THEN sqlc.arg(changed_by)::text::uuid END,
    updated_at = now(), version_id = version_id + 1
WHERE id = sqlc.arg(id)::text::uuid AND version_id = sqlc.arg(version_id);

-- name: InsertActivity :exec
INSERT INTO care_plan_activity (care_plan_id, position, charge_item_definition_id, day_offset, visit_mode, appointment_id)
VALUES (sqlc.arg(care_plan_id)::text::uuid, sqlc.arg(position), sqlc.arg(service_id)::text::uuid, sqlc.arg(day_offset),
        sqlc.arg(visit_mode), NULLIF(sqlc.arg(appointment_id)::text, '')::uuid);

-- name: UpdateActivity :execrows
UPDATE care_plan_activity
SET position = sqlc.arg(position), charge_item_definition_id = sqlc.arg(service_id)::text::uuid,
    day_offset = sqlc.arg(day_offset), visit_mode = sqlc.arg(visit_mode),
    appointment_id = NULLIF(sqlc.arg(appointment_id)::text, '')::uuid
WHERE id = sqlc.arg(id)::text::uuid AND care_plan_id = sqlc.arg(care_plan_id)::text::uuid;

-- name: DeleteActivitiesExcept :exec
DELETE FROM care_plan_activity
WHERE care_plan_id = sqlc.arg(care_plan_id)::text::uuid AND NOT (id::text = ANY (sqlc.arg(keep)::text[]));

-- name: PlanByID :one
SELECT id::text AS id, organization_id::text AS organization_id, patient_id::text AS patient_id, status, title,
       COALESCE(quote_id::text, '')::text AS quote_id, starts_on, booked_at, revoked_at, created_at, version_id
FROM care_plan
WHERE id = sqlc.arg(id)::text::uuid;

-- name: PlansForPatient :many
SELECT id::text AS id, organization_id::text AS organization_id, patient_id::text AS patient_id, status, title,
       COALESCE(quote_id::text, '')::text AS quote_id, starts_on, booked_at, revoked_at, created_at, version_id
FROM care_plan
WHERE patient_id = sqlc.arg(patient_id)::text::uuid
ORDER BY created_at DESC, id;

-- name: Activities :many
SELECT id::text AS id, care_plan_id::text AS care_plan_id, position, charge_item_definition_id::text AS service_id,
       day_offset, visit_mode, COALESCE(appointment_id::text, '')::text AS appointment_id
FROM care_plan_activity
WHERE care_plan_id::text = ANY (sqlc.arg(plan_ids)::text[])
ORDER BY care_plan_id, position;
