-- name: CreateSession :execrows
-- The first token request; a concurrent first request for the appointment waits for this one's commit and
-- then does nothing (no row), so both read the same session and only the first enqueues the window's close.
INSERT INTO video_session (appointment_id, organization_id, room, patient_identity, practitioner_identity, closes_at,
                           created_at, updated_at)
VALUES (@appointment_id, @organization_id, @room, @patient_identity, @practitioner_identity, @closes_at, @created_at, @created_at)
ON CONFLICT (appointment_id) DO NOTHING;

-- name: SessionByAppointment :one
SELECT id, created_at, updated_at, closes_at, patient_joined_at, practitioner_joined_at, ended_at, appointment_id,
       organization_id, room, patient_identity, practitioner_identity, end_reason
FROM video_session
WHERE appointment_id = @appointment_id;

-- name: DeleteSessionsEndedBefore :execrows
-- Retention, as salus_retention: sessions 30 days after their end, or after their window's close when they
-- never ended, in every clinic.
DELETE FROM video_session
WHERE coalesce(ended_at, closes_at) < @before::timestamptz;

-- name: SessionByID :one
SELECT id, created_at, updated_at, closes_at, patient_joined_at, practitioner_joined_at, ended_at, appointment_id,
       organization_id, room, patient_identity, practitioner_identity, end_reason
FROM video_session
WHERE id = @id;

-- name: EndSession :one
-- Only a session not ended yet ends: a repeat, or a later path, leaves no row.
UPDATE video_session
SET ended_at = @at, end_reason = @reason, updated_at = @at
WHERE id = @id AND ended_at IS NULL
RETURNING id, created_at, updated_at, closes_at, patient_joined_at, practitioner_joined_at, ended_at, appointment_id,
          organization_id, room, patient_identity, practitioner_identity, end_reason;

-- name: SessionByRoom :one
SELECT id, created_at, updated_at, closes_at, patient_joined_at, practitioner_joined_at, ended_at, appointment_id,
       organization_id, room, patient_identity, practitioner_identity, end_reason
FROM video_session
WHERE room = @room;

-- name: PatientJoined :one
-- The patient's first join, never after the end: a repeat or a late event leaves no row.
UPDATE video_session
SET patient_joined_at = @at, updated_at = @updated_at
WHERE id = @id AND patient_joined_at IS NULL AND ended_at IS NULL
RETURNING id, created_at, updated_at, closes_at, patient_joined_at, practitioner_joined_at, ended_at, appointment_id,
          organization_id, room, patient_identity, practitioner_identity, end_reason;

-- name: PractitionerJoined :one
-- The practitioner's first join, never after the end: a repeat or a late event leaves no row.
UPDATE video_session
SET practitioner_joined_at = @at, updated_at = @updated_at
WHERE id = @id AND practitioner_joined_at IS NULL AND ended_at IS NULL
RETURNING id, created_at, updated_at, closes_at, patient_joined_at, practitioner_joined_at, ended_at, appointment_id,
          organization_id, room, patient_identity, practitioner_identity, end_reason;
