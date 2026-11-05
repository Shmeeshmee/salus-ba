-- +goose Up
-- Video sessions (docs/consultation, package 1; SPEC.md §2 "Tables", §5): the LiveKit room of one
-- appointment's consultation, created on the first token request. The room and both participant identities
-- are random (r_…, p_…), never derived from an id of ours, so LiveKit learns nothing about who is in the
-- call. closes_at is the join window's close (the appointment's end + 30 minutes), copied once: the
-- retention period of a session that never ended runs from it, as salus_retention may read only this table.
-- Row-level security, like appointment's:
--   - the patient (app.patient_id) sees and creates the sessions of their own appointments, a new one with its
--     appointment's clinic;
--   - a clinic (app.organization_id, neither patient setting set) sees, creates and ends the sessions of its
--     appointments with a patient who has an active consent with it, and, after a revocation, of those that
--     started before the latest one;
--   - the LiveKit webhook (app.video_room, neither patient setting set) sees and updates the one session of
--     its room, whatever the clinic; it never creates one.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE video_session (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    closes_at timestamptz NOT NULL,
    patient_joined_at timestamptz,
    practitioner_joined_at timestamptz,
    ended_at timestamptz,
    appointment_id uuid NOT NULL UNIQUE REFERENCES appointment (id),
    organization_id uuid NOT NULL,
    room text NOT NULL UNIQUE,
    patient_identity text NOT NULL UNIQUE,
    practitioner_identity text NOT NULL UNIQUE,
    end_reason text CHECK (end_reason IN ('practitioner', 'window_closed', 'practitioner_absent', 'time_limit', 'cancelled')),
    CONSTRAINT video_session_ended CHECK ((ended_at IS NULL) = (end_reason IS NULL)),
    CONSTRAINT video_session_two_identities CHECK (patient_identity <> practitioner_identity)
);

ALTER TABLE video_session ENABLE ROW LEVEL SECURITY;
CREATE POLICY video_session_own ON video_session FOR SELECT
    USING (EXISTS (SELECT 1 FROM appointment a
                   WHERE a.id = video_session.appointment_id
                     AND a.patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid));
CREATE POLICY video_session_own_create ON video_session FOR INSERT
    WITH CHECK (EXISTS (SELECT 1 FROM appointment a
                        WHERE a.id = video_session.appointment_id
                          AND a.organization_id = video_session.organization_id
                          AND a.patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid));
CREATE POLICY video_session_clinic ON video_session
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM appointment a
                       WHERE a.id = video_session.appointment_id
                         AND a.organization_id = video_session.organization_id
                         AND (EXISTS (SELECT 1 FROM consent c
                                      WHERE c.patient_id = a.patient_id
                                        AND c.organization_id = a.organization_id
                                        AND c.status = 'active')
                              OR lower(a.period) < (SELECT max(c.revoked_at) FROM consent c
                                                    WHERE c.patient_id = a.patient_id
                                                      AND c.organization_id = a.organization_id))));
CREATE POLICY video_session_webhook ON video_session FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND room = NULLIF(current_setting('app.video_room', true), ''));
CREATE POLICY video_session_webhook_update ON video_session FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND room = NULLIF(current_setting('app.video_room', true), ''));

-- What a session is for and its random names never change; only who joined and the end do. No DELETE for
-- the application: retention deletes (internal/retention, 30 days after the end).
GRANT SELECT, INSERT ON video_session TO salus_app;
GRANT UPDATE (patient_joined_at, practitioner_joined_at, ended_at, end_reason, updated_at) ON video_session TO salus_app;
GRANT SELECT, DELETE ON video_session TO salus_retention;
