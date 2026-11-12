-- name: CreateAppointment :one
-- Booking, in the patient's scope; a time the role is booked at already violates
-- appointment_no_double_booking (23P01), which aborts the transaction.
INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, created_at, updated_at)
VALUES (@patient_id, @practitioner_role_id, @organization_id, @slot_id, tstzrange(@start_at::timestamptz, @end_at::timestamptz),
        @created_at, @created_at)
RETURNING id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
          cancelled_by_account_id, version_id, status, cancellation_reason,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: AppointmentByID :one
SELECT id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
       cancelled_by_account_id, version_id, status, cancellation_reason,
       lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at
FROM appointment
WHERE id = @id;

-- name: AppointmentByIDForShare :one
-- AppointmentByID, locking the row against a change (a cancellation's UPDATE waits) until the transaction
-- ends: a write that rests on the appointment being booked. Row-level security asks the UPDATE policies too.
SELECT id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
       cancelled_by_account_id, version_id, status, cancellation_reason,
       lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at
FROM appointment
WHERE id = @id
FOR SHARE;

-- name: SearchAppointments :many
-- Appointment?patient=…&date=ge…&date=lt…&status=…, a clinic's calendar, a patient's visits in a clinic. An
-- absent parameter matches everything; the scope decides the rest.
SELECT id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
       cancelled_by_account_id, version_id, status, cancellation_reason,
       lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at
FROM appointment
WHERE (sqlc.narg(patient_id)::uuid IS NULL OR patient_id = sqlc.narg(patient_id)::uuid)
  AND (sqlc.narg(organization_id)::uuid IS NULL OR organization_id = sqlc.narg(organization_id)::uuid)
  AND (sqlc.narg(practitioner_role_id)::uuid IS NULL OR practitioner_role_id = sqlc.narg(practitioner_role_id)::uuid)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(from_at)::timestamptz IS NULL OR lower(period) >= sqlc.narg(from_at)::timestamptz)
  AND (sqlc.narg(to_at)::timestamptz IS NULL OR lower(period) < sqlc.narg(to_at)::timestamptz)
ORDER BY lower(period), id;

-- name: CancelAppointment :one
-- Only a booked appointment is cancelled; a concurrent change that committed first leaves no row.
UPDATE appointment
SET status = 'cancelled', cancelled_at = @at, cancellation_reason = @reason, cancelled_by_account_id = sqlc.narg(cancelled_by),
    updated_at = @at, version_id = version_id + 1
WHERE id = @id AND status = 'booked'
RETURNING id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
          cancelled_by_account_id, version_id, status, cancellation_reason,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: SetAppointmentStatus :one
UPDATE appointment
SET status = @status, updated_at = @at, version_id = version_id + 1
WHERE id = @id AND status = 'booked'
RETURNING id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
          cancelled_by_account_id, version_id, status, cancellation_reason,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: ListBookedForStats :many
-- A clinic's appointments starting in [from, to), oldest first: the columns salus_stats may read.
SELECT lower(period)::timestamptz AS start_at, patient_id, practitioner_role_id, status
FROM appointment
WHERE organization_id = @organization_id AND lower(period) >= @from_at::timestamptz AND lower(period) < @to_at::timestamptz
ORDER BY lower(period), status;
