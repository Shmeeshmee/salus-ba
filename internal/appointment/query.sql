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
SELECT lower(period)::timestamptz AS start_at, patient_id, practitioner_role_id, status, booked_via
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
INSERT INTO appointment_action_token (created_at, expires_at, start_at, appointment_id, organization_id, patient_id, account_id,
                                      reminder_id, token_hash, channel, language)
VALUES (@created_at, @start_at, @start_at, @appointment_id, @organization_id, @patient_id, @account_id, sqlc.narg(reminder_id),
        @token_hash, @channel, @language);

-- name: DeleteActionTokensExpiredBefore :execrows
-- Retention, as salus_retention: links whose period after their expiry is over.
DELETE FROM appointment_action_token
WHERE expires_at < @before::timestamptz;

-- name: ListFulfilledForStats :many
-- A clinic's fulfilled appointments starting in [from, to), oldest first: the platform heatmap reads no other.
SELECT lower(period)::timestamptz AS start_at, patient_id, practitioner_role_id, status
FROM appointment
WHERE organization_id = @organization_id AND status = 'fulfilled'
  AND lower(period) >= @from_at::timestamptz AND lower(period) < @to_at::timestamptz
ORDER BY lower(period);

-- SAL-34 (docs/plans/appointments.md): the patient's confirmation and reschedule, and the use of a one-time link.

-- name: AppointmentByIDForUpdate :one
-- AppointmentByID, locking the row until the transaction ends: a reschedule decides on what it reads, and a concurrent
-- reschedule or cancellation waits for it. Row-level security asks the UPDATE policies too.
SELECT id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
       cancelled_by_account_id, version_id, status, cancellation_reason, confirmed_at, confirmed_via, reschedule_count, booked_via,
       lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at
FROM appointment
WHERE id = @id
FOR UPDATE;

-- name: ConfirmAppointment :one
-- Only a booked, unconfirmed appointment, in the patient's scope (appointment_guard_change).
UPDATE appointment
SET confirmed_at = @at, confirmed_via = @via, updated_at = @at, version_id = version_id + 1
WHERE id = @id AND status = 'booked' AND confirmed_at IS NULL
RETURNING id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
          cancelled_by_account_id, version_id, status, cancellation_reason, confirmed_at, confirmed_via, reschedule_count, booked_via,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: RescheduleAppointment :one
-- The period and the slot together, the count up by one, and the confirmation in the app when there was none (choosing
-- the time confirms it; a confirmation never changes). Only a booked appointment still at the count the reschedule
-- was decided on; an overlap with the role's other appointments violates appointment_no_double_booking (23P01).
UPDATE appointment
SET period = tstzrange(@start_at::timestamptz, @end_at::timestamptz), slot_id = @slot_id, reschedule_count = reschedule_count + 1,
    confirmed_at = COALESCE(confirmed_at, @at), confirmed_via = COALESCE(confirmed_via, 'app'),
    updated_at = @at, version_id = version_id + 1
WHERE id = @id AND status = 'booked' AND reschedule_count = @from_count
RETURNING id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
          cancelled_by_account_id, version_id, status, cancellation_reason, confirmed_at, confirmed_via, reschedule_count, booked_via,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: FindActionToken :one
-- The ids of the link with the hash, whatever the scope (appointment_action_token_find, SECURITY DEFINER).
SELECT f.id::uuid AS id, f.appointment_id::uuid AS appointment_id, f.organization_id::uuid AS organization_id,
       f.patient_id::uuid AS patient_id, f.account_id::uuid AS account_id
FROM appointment_action_token_find(@token_hash::bytea) AS f;

-- name: ActionTokenByID :one
-- In the link's patient's scope.
SELECT id, created_at, start_at, used_at, appointment_id, organization_id, patient_id, account_id, reminder_id, token_hash,
       channel, language, used_action
FROM appointment_action_token
WHERE id = @id;

-- name: UseActionToken :execrows
-- Once: a used link has used_at set and is never changed again (the trigger refuses it for every role).
UPDATE appointment_action_token
SET used_at = @at, used_action = @action
WHERE id = @id AND used_at IS NULL;

-- SAL-33 (docs/plans/appointments.md): the reminders.

-- name: InsertReminder :one
-- A scheduled reminder; none when the appointment has one of that offset for that start already.
INSERT INTO appointment_reminder (appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at)
VALUES (@appointment_id, @organization_id, @patient_id, @offset_minutes, @start_at, @due_at)
ON CONFLICT (appointment_id, offset_minutes, start_at) WHERE state IN ('scheduled', 'dispatched') DO NOTHING
RETURNING id, created_at;

-- name: CancelReminders :execrows
-- The scheduled ones, and the dispatched ones whose sends may still be held: all of them are dead with the change.
UPDATE appointment_reminder SET state = 'cancelled'
WHERE appointment_id = @appointment_id AND state IN ('scheduled', 'dispatched');

-- name: LockReminder :one
SELECT id, appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at, state, dispatched_at
FROM appointment_reminder
WHERE id = @id
FOR UPDATE;

-- name: ReminderByID :one
SELECT id, appointment_id, organization_id, patient_id, offset_minutes, start_at, due_at, state, dispatched_at
FROM appointment_reminder
WHERE id = @id;

-- name: SetReminderState :execrows
UPDATE appointment_reminder SET state = @state, dispatched_at = sqlc.narg(dispatched_at)
WHERE id = @id AND state = 'scheduled';

-- name: OverdueReminders :many
-- The sweep's page, of every clinic (the view reads with its owner's rights): due by at, the start still ahead.
SELECT id, organization_id, due_at
FROM appointment_reminder_scheduled
WHERE due_at <= @at::timestamptz AND start_at > @at::timestamptz
  AND (due_at, id) > (@after_due::timestamptz, @after_id::uuid)
ORDER BY due_at, id
LIMIT @max::integer;

-- name: DeleteRemindersDueBefore :execrows
-- Retention, as salus_retention.
DELETE FROM appointment_reminder
WHERE due_at < @before::timestamptz;

-- SAL-35: the waitlist (waitlist_entry, waitlist_offer).

-- name: InsertWaitlistEntry :one
-- The patient joins, in their own scope.
INSERT INTO waitlist_entry (created_at, updated_at, ends_at, patient_id, organization_id, practitioner_role_id, specialty,
                            earliest, latest, times_start_minute, times_end_minute)
VALUES (@created_at, @created_at, @ends_at, @patient_id, @organization_id, sqlc.narg(practitioner_role_id), sqlc.narg(specialty),
        @earliest, @latest, sqlc.narg(times_start_minute), sqlc.narg(times_end_minute))
RETURNING id, created_at, ends_at, closed_at, patient_id, organization_id, practitioner_role_id, earliest, latest,
          times_start_minute, times_end_minute, priority, status, specialty;

-- name: CountActiveWaitlistEntries :one
SELECT count(*)::integer FROM waitlist_entry
WHERE patient_id = @patient_id AND organization_id = @organization_id AND status = 'active' AND ends_at > @now::timestamptz;

-- name: ListPatientWaitlistEntries :many
-- The patient's entries, newest first, each with its open offer.
SELECT e.id, e.created_at, e.ends_at, e.closed_at, e.patient_id, e.organization_id, e.practitioner_role_id, e.earliest, e.latest,
       e.times_start_minute, e.times_end_minute, e.priority, e.status, e.specialty,
       o.id AS offer_id, o.slot_id AS offer_slot_id, o.expires_at AS offer_expires_at,
       lower(o.period)::timestamptz AS offer_start, upper(o.period)::timestamptz AS offer_end
FROM waitlist_entry e
LEFT JOIN waitlist_offer o ON o.entry_id = e.id AND o.state = 'open'
WHERE e.patient_id = @patient_id
ORDER BY e.created_at DESC, e.id;

-- name: ListClinicWaitlistEntries :many
-- A page of the clinic's entries in offer order, each with its open offer; with a role, those it could be offered: its
-- own and its specialty's; with a status, those in it (active and expired split by ends_at at now).
SELECT e.id, e.created_at, e.ends_at, e.closed_at, e.patient_id, e.organization_id, e.practitioner_role_id, e.earliest, e.latest,
       e.times_start_minute, e.times_end_minute, e.priority, e.status, e.specialty,
       o.id AS offer_id, o.slot_id AS offer_slot_id, o.expires_at AS offer_expires_at,
       lower(o.period)::timestamptz AS offer_start, upper(o.period)::timestamptz AS offer_end
FROM waitlist_entry e
LEFT JOIN waitlist_offer o ON o.entry_id = e.id AND o.state = 'open'
WHERE e.organization_id = @organization_id
  AND (sqlc.narg(practitioner_role_id)::uuid IS NULL OR e.practitioner_role_id = sqlc.narg(practitioner_role_id)::uuid
       OR (sqlc.narg(specialty)::text IS NOT NULL AND e.specialty = sqlc.narg(specialty)::text))
  AND (sqlc.narg(status)::text IS NULL
       OR (sqlc.narg(status)::text = 'active' AND e.status = 'active' AND e.ends_at > @now::timestamptz)
       OR (sqlc.narg(status)::text = 'expired' AND e.status = 'active' AND e.ends_at <= @now::timestamptz)
       OR (sqlc.narg(status)::text IN ('booked', 'withdrawn') AND e.status = sqlc.narg(status)::text))
ORDER BY e.priority DESC, e.created_at, e.id
LIMIT @max_rows OFFSET @skip_rows;

-- name: LockWaitlistEntries :exec
-- Joins of one patient with one clinic run one after the other, so the limit of active entries holds (an advisory
-- lock until the transaction ends).
SELECT pg_advisory_xact_lock(hashtextextended('waitlist:' || @patient_id::text || ':' || @organization_id::text, 0));

-- name: WaitlistEntryForUpdate :one
SELECT id, created_at, ends_at, closed_at, patient_id, organization_id, practitioner_role_id, earliest, latest,
       times_start_minute, times_end_minute, priority, status, specialty
FROM waitlist_entry
WHERE id = @id
FOR UPDATE;

-- name: ActiveWaitlistEntries :many
SELECT id, created_at, ends_at, closed_at, patient_id, organization_id, practitioner_role_id, earliest, latest,
       times_start_minute, times_end_minute, priority, status, specialty
FROM waitlist_entry
WHERE patient_id = @patient_id AND organization_id = @organization_id AND status = 'active'
ORDER BY created_at, id;

-- name: CloseWaitlistEntry :one
UPDATE waitlist_entry
SET status = @status, closed_at = @at, updated_at = @at
WHERE id = @id AND status = 'active'
RETURNING id, created_at, ends_at, closed_at, patient_id, organization_id, practitioner_role_id, earliest, latest,
          times_start_minute, times_end_minute, priority, status, specialty;

-- name: SetWaitlistPriority :one
UPDATE waitlist_entry
SET priority = @priority, updated_at = @at
WHERE id = @id AND status = 'active'
RETURNING id, created_at, ends_at, closed_at, patient_id, organization_id, practitioner_role_id, earliest, latest,
          times_start_minute, times_end_minute, priority, status, specialty;

-- name: WaitlistCandidates :many
-- The freed slot's next entries in offer order: of its clinic, its role or the role's specialty, its day and times
-- (the clinic's zone, computed by the service), not past their last day; never offered this slot and without an open
-- offer; of a patient who never held the slot (an appointment of it in any status, or a reschedule away from it) and
-- has no booked appointment overlapping it. The service checks the care consent of each again.
SELECT e.id, e.created_at, e.ends_at, e.closed_at, e.patient_id, e.organization_id, e.practitioner_role_id, e.earliest, e.latest,
       e.times_start_minute, e.times_end_minute, e.priority, e.status, e.specialty
FROM waitlist_entry e
WHERE e.organization_id = @organization_id AND e.status = 'active' AND e.ends_at > @now::timestamptz
  AND (e.practitioner_role_id = @practitioner_role_id
       OR (sqlc.narg(specialty)::text IS NOT NULL AND e.specialty = sqlc.narg(specialty)::text))
  AND e.earliest <= @day::date AND e.latest >= @day::date
  AND (e.times_start_minute IS NULL
       OR (e.times_start_minute <= @start_minute::smallint AND e.times_end_minute >= @end_minute::smallint))
  AND NOT EXISTS (SELECT 1 FROM waitlist_offer o WHERE o.entry_id = e.id AND (o.state = 'open' OR o.slot_id = @slot_id))
  AND NOT EXISTS (SELECT 1 FROM appointment a
                  WHERE a.patient_id = e.patient_id
                    AND (a.slot_id = @slot_id
                         OR (a.status = 'booked' AND a.period && tstzrange(@start_at::timestamptz, @end_at::timestamptz))))
  AND NOT EXISTS (SELECT 1 FROM appointment_change c WHERE c.patient_id = e.patient_id AND c.from_slot_id = @slot_id)
  AND NOT (e.id = ANY (@skip::uuid[]))
ORDER BY e.priority DESC, e.created_at, e.id
LIMIT @max_rows
-- The entries a concurrent job or withdrawal holds are passed over; the one chosen stays locked until the offer is stored.
FOR UPDATE OF e SKIP LOCKED;

-- name: InsertWaitlistOffer :one
-- A job of the clinic offers the slot to an active entry it sees.
INSERT INTO waitlist_offer (created_at, expires_at, entry_id, slot_id, organization_id, patient_id, practitioner_role_id, period)
VALUES (@created_at, @expires_at, @entry_id, @slot_id, @organization_id, @patient_id, @practitioner_role_id,
        tstzrange(@start_at::timestamptz, @end_at::timestamptz))
RETURNING id, created_at, expires_at, entry_id, slot_id, organization_id, patient_id, practitioner_role_id, appointment_id, state,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: WaitlistOfferByID :one
SELECT id, created_at, expires_at, entry_id, slot_id, organization_id, patient_id, practitioner_role_id, appointment_id, state,
       lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at
FROM waitlist_offer
WHERE id = @id;

-- name: WaitlistOfferForUpdate :one
-- Acceptance and expiry serialise on the offer's row.
SELECT id, created_at, expires_at, entry_id, slot_id, organization_id, patient_id, practitioner_role_id, appointment_id, state,
       lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at
FROM waitlist_offer
WHERE id = @id
FOR UPDATE;

-- name: OpenWaitlistOffer :many
-- The open offer of a slot, or of an entry.
SELECT id, created_at, expires_at, entry_id, slot_id, organization_id, patient_id, practitioner_role_id, appointment_id, state,
       lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at
FROM waitlist_offer
WHERE state = 'open'
  AND (sqlc.narg(slot_id)::uuid IS NULL OR slot_id = sqlc.narg(slot_id)::uuid)
  AND (sqlc.narg(entry_id)::uuid IS NULL OR entry_id = sqlc.narg(entry_id)::uuid);

-- name: CloseWaitlistOffer :one
UPDATE waitlist_offer
SET state = @state, closed_at = @at, appointment_id = sqlc.narg(appointment_id)
WHERE id = @id AND state = 'open'
RETURNING id, created_at, expires_at, entry_id, slot_id, organization_id, patient_id, practitioner_role_id, appointment_id, state,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: DeleteWaitlistEntriesClosedBefore :execrows
-- Retention, as salus_retention: entries 30 days after they closed, or after their last day while active; their
-- offers go with them.
DELETE FROM waitlist_entry
WHERE closed_at < @before::timestamptz OR (status = 'active' AND ends_at < @before::timestamptz);

-- name: DeleteWaitlistOffersClosedBefore :execrows
-- Retention, as salus_retention: the offers of the entries DeleteWaitlistEntriesClosedBefore deletes.
DELETE FROM waitlist_offer o
USING waitlist_entry e
WHERE o.entry_id = e.id AND (e.closed_at < @before::timestamptz OR (e.status = 'active' AND e.ends_at < @before::timestamptz));

-- SAL-36 (docs/plans/appointments.md): no-shows.

-- name: CorrectAppointmentStatus :one
-- A no-show corrected into an attendance or back: only from the status the correction names, and only between those
-- two.
UPDATE appointment
SET status = @to_status::text, updated_at = @at, version_id = version_id + 1
WHERE id = @id AND status = @from_status::text AND status IN ('fulfilled', 'noshow') AND @to_status::text IN ('fulfilled', 'noshow')
RETURNING id, created_at, updated_at, cancelled_at, patient_id, practitioner_role_id, organization_id, slot_id,
          cancelled_by_account_id, version_id, status, cancellation_reason, confirmed_at, confirmed_via, reschedule_count, booked_via,
          lower(period)::timestamptz AS start_at, upper(period)::timestamptz AS end_at;

-- name: CountNoShowsSince :many
-- Each patient's noshow appointments at the clinic that started in [since, until], in one query for a range's
-- patients (the repeat no-show flag); a patient without any has no row.
SELECT patient_id, count(*)::integer AS noshows
FROM appointment
WHERE organization_id = @organization_id AND patient_id = ANY (@patient_ids::uuid[]) AND status = 'noshow'
  AND lower(period) >= @since::timestamptz AND lower(period) <= @until::timestamptz
GROUP BY patient_id;
