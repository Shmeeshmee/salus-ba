-- name: CreateSession :exec
-- The first token request; a concurrent first request for the appointment waits for this one's commit and
-- then does nothing, so both read the same session.
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
