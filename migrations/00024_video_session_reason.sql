-- +goose Up
-- Why the consultation ended or its patient was removed (docs/consultation, package 7.1; SPEC.md §2
-- "Consultation status", "Changed after package 6"):
--   - removal_reason records, once, why the patient was removed from the room: their consent was withdrawn
--     or the appointment cancelled. The end alone cannot tell: a revocation before the start cancels the
--     appointment and ends the session 'cancelled' too. Set in the room's scope (video_session_webhook_update),
--     with the removal's audit event.
--   - the practitioner who was in the room keeps reading the session: a clinic's scope narrowed to one
--     appointment (app.appointment_id, the setting of appointment's notice scope) sees that appointment's
--     session once its practitioner joined, whatever the consent. The service checks that the caller is the
--     appointment's practitioner. Read only: no update or insert through it.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

ALTER TABLE video_session
    ADD COLUMN removal_reason text CHECK (removal_reason IN ('consent_withdrawn', 'cancelled'));

CREATE POLICY video_session_attended ON video_session FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND appointment_id = NULLIF(current_setting('app.appointment_id', true), '')::uuid
           AND practitioner_joined_at IS NOT NULL);

GRANT UPDATE (removal_reason) ON video_session TO salus_app;
