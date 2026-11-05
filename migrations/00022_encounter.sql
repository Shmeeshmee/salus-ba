-- +goose Up
-- Encounters (docs/consultation, package 2; SPEC.md §2 "Tables", "Encounter", §5): the FHIR Encounter of an
-- appointment's video consultation, started when the practitioner first joins and finished when the
-- consultation ends. patient_id and practitioner_role_id are the Encounter's own subject and participant,
-- copied once from the appointment when it starts and never changed; row-level security needs them.
-- Row-level security, like video_session's but with no webhook scope (the webhook sets the clinic after
-- reading its session):
--   - the patient (app.patient_id) sees the encounters of their own appointments; they never write one;
--   - a clinic (app.organization_id, neither patient setting set) sees, starts and finishes the encounters of
--     its appointments with a patient who has an active consent with it, and, after a revocation, of those
--     that started before the latest one.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE encounter (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    period_start timestamptz NOT NULL,
    period_end timestamptz,
    appointment_id uuid NOT NULL UNIQUE REFERENCES appointment (id),
    organization_id uuid NOT NULL,
    patient_id uuid NOT NULL REFERENCES patient (id),
    practitioner_role_id uuid NOT NULL,
    version_id integer DEFAULT 1 NOT NULL,
    status text NOT NULL CHECK (status IN ('in-progress', 'finished')),
    CONSTRAINT encounter_practitioner_role_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT encounter_finished CHECK ((status = 'finished') = (period_end IS NOT NULL)),
    CONSTRAINT encounter_period CHECK (period_end IS NULL OR period_end >= period_start)
);

-- Encounter?patient=… and a clinic's encounters (the appointment has its unique index).
CREATE INDEX encounter_patient ON encounter (patient_id);
CREATE INDEX encounter_organization_role ON encounter (organization_id, practitioner_role_id);

ALTER TABLE encounter ENABLE ROW LEVEL SECURITY;
CREATE POLICY encounter_own ON encounter FOR SELECT
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY encounter_clinic ON encounter
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM appointment a
                       WHERE a.id = encounter.appointment_id
                         AND a.organization_id = encounter.organization_id
                         AND a.patient_id = encounter.patient_id
                         AND (EXISTS (SELECT 1 FROM consent c
                                      WHERE c.patient_id = a.patient_id
                                        AND c.organization_id = a.organization_id
                                        AND c.status = 'active')
                              OR lower(a.period) < (SELECT max(c.revoked_at) FROM consent c
                                                    WHERE c.patient_id = a.patient_id
                                                      AND c.organization_id = a.organization_id))));

-- Whose, which appointment's and when it started never change; only the finish does. No DELETE: kept with
-- the clinical record (retention keep).
GRANT SELECT, INSERT ON encounter TO salus_app;
GRANT UPDATE (status, period_end, updated_at, version_id) ON encounter TO salus_app;
