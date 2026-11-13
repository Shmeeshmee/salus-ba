-- name: CreateAppointment :one
-- Booking, in the patient's scope; a time the role is booked at already violates
-- appointment_no_double_booking (23P01), which aborts the transaction.
INSERT INTO appointment (patient_id, practitioner_role_id, organization_id, slot_id, period, created_at, updated_at, booked_via)
VALUES (@patient_id, @practitioner_role_id, @organization_id, @slot_id, tstzrange(@start_at::timestamptz, @end_at::timestamptz),
        @created_at, @created_at, @booked_via)
RETURNING id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
          cancelled_by_account_id, version_id, status, cancellation_reason, confirmed_at, confirmed_via, reschedule_count, booked_via,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: AppointmentByID :one
SELECT id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
       cancelled_by_account_id, version_id, status, cancellation_reason, confirmed_at, confirmed_via, reschedule_count, booked_via,
       lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at
FROM appointment
WHERE id = @id;

-- name: AppointmentByIDForShare :one
-- AppointmentByID, locking the row against a change (a cancellation's UPDATE waits) until the transaction
-- ends: a write that rests on the appointment being booked. Row-level security asks the UPDATE policies too.
SELECT id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
       cancelled_by_account_id, version_id, status, cancellation_reason, confirmed_at, confirmed_via, reschedule_count, booked_via,
       lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at
FROM appointment
WHERE id = @id
FOR SHARE;

-- name: SearchAppointments :many
-- Appointment?patient=…&date=ge…&date=lt…&status=…, a clinic's calendar, a patient's visits in a clinic. An
-- absent parameter matches everything; the scope decides the rest.
SELECT id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
       cancelled_by_account_id, version_id, status, cancellation_reason, confirmed_at, confirmed_via, reschedule_count, booked_via,
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
          cancelled_by_account_id, version_id, status, cancellation_reason, confirmed_at, confirmed_via, reschedule_count, booked_via,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: SetAppointmentStatus :one
UPDATE appointment
SET status = @status, updated_at = @at, version_id = version_id + 1
WHERE id = @id AND status = 'booked'
RETURNING id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
          cancelled_by_account_id, version_id, status, cancellation_reason, confirmed_at, confirmed_via, reschedule_count, booked_via,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: ListBookedForStats :many
-- A clinic's appointments starting in [from, to), oldest first: the columns salus_stats may read.
SELECT lower(period)::timestamptz AS start_at, patient_id, practitioner_role_id, status
FROM appointment
WHERE organization_id = @organization_id AND lower(period) >= @from_at::timestamptz AND lower(period) < @to_at::timestamptz
ORDER BY lower(period), status;

-- SAL-32 step 0 (docs/plans/appointments.md): the history and the one-time links.

-- name: InsertAppointmentChange :one
-- One row of an appointment's history, with the appointment's clinic and patient (the foreign key checks them);
-- row-level security lets the scope write it only for an appointment it sees.
INSERT INTO appointment_change (at, appointment_id, organization_id, patient_id, by_account_id, from_slot_id, to_slot_id,
                                from_period, to_period, kind, via, agent, from_status, to_status)
SELECT @at, a.id, a.organization_id, a.patient_id, sqlc.narg(by_account_id), sqlc.narg(from_slot_id), sqlc.narg(to_slot_id),
       CASE WHEN sqlc.narg(from_start)::timestamptz IS NULL THEN NULL ELSE tstzrange(sqlc.narg(from_start)::timestamptz, sqlc.narg(from_end)::timestamptz) END,
       CASE WHEN sqlc.narg(to_start)::timestamptz IS NULL THEN NULL ELSE tstzrange(sqlc.narg(to_start)::timestamptz, sqlc.narg(to_end)::timestamptz) END,
       @kind, @via, sqlc.narg(agent), sqlc.narg(from_status), sqlc.narg(to_status)
FROM appointment a
WHERE a.id = @appointment_id
RETURNING id;

-- name: ListAppointmentChanges :many
SELECT id, at, appointment_id, by_account_id, from_slot_id, to_slot_id, kind, via, agent, from_status, to_status,
       lower(from_period)::timestamptz AS from_start, upper(from_period)::timestamptz AS from_end,
       lower(to_period)::timestamptz AS to_start, upper(to_period)::timestamptz AS to_end
FROM appointment_change
WHERE appointment_id = @appointment_id
ORDER BY at, id;

-- name: InsertActionToken :exec
-- A notifier job's one-time link, in the clinic scope with the job's appointment set.
INSERT INTO appointment_action_token (created_at, expires_at, appointment_id, organization_id, patient_id, account_id, reminder_id,
                                      token_hash, channel, language)
VALUES (@created_at, @expires_at, @appointment_id, @organization_id, @patient_id, @account_id, sqlc.narg(reminder_id),
        @token_hash, @channel, @language);

-- name: DeleteActionTokensExpiredBefore :execrows
-- Retention, as salus_retention: links whose period after their expiry is over.
DELETE FROM appointment_action_token
WHERE expires_at < @before::timestamptz;
