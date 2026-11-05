-- +goose Up
-- The room's scope on encounter (docs/consultation, package 5.1; SPEC.md §2 "Tables", "Changed after
-- package 5"): like video_session's webhook scope, a transaction with app.video_room set (neither patient
-- setting set) sees and updates the encounter of the appointment whose session has that room, whatever the
-- consent. A revocation that cancels an appointment after the practitioner joined then finishes its
-- Encounter, with period_end set, although the clinic's scope no longer shows it. It never creates one: the
-- insert stays the clinic's. video_session's own policies decide which session the room shows.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE POLICY encounter_room ON encounter FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM video_session v
                       WHERE v.appointment_id = encounter.appointment_id
                         AND v.room = NULLIF(current_setting('app.video_room', true), '')));
CREATE POLICY encounter_room_update ON encounter FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM video_session v
                       WHERE v.appointment_id = encounter.appointment_id
                         AND v.room = NULLIF(current_setting('app.video_room', true), '')));
