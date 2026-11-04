-- +goose Up
-- Consents (docs/booking, package 3): FHIR Consent, one per patient per clinic, "share my profile and
-- appointments with this clinic for care" (SPEC.md "Consent shape"). At most one active per patient and
-- clinic; granting again after a revocation adds a row, so the history is kept as proof (Art. 7(1)).
-- Row-level secured in two scopes: a patient's own transaction (app.patient_id) sees and writes only their
-- consents; a clinic's (app.organization_id) only reads the consents given to it. The clinic branch, here
-- and on patient, holds only while neither app.patient_id nor app.account_id is set, so a patient's
-- transaction with a clinic set never sees another patient.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE consent (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    date_time timestamptz NOT NULL, -- granted
    revoked_at timestamptz,
    patient_id uuid NOT NULL REFERENCES patient (id),
    organization_id uuid NOT NULL REFERENCES organization (id),
    version_id integer DEFAULT 1 NOT NULL,
    status text NOT NULL CHECK (status IN ('active', 'inactive')),
    CONSTRAINT consent_revoked_when_inactive CHECK ((status = 'inactive') = (revoked_at IS NOT NULL))
);

-- The partial unique index also serves the patient policy's lookup of an active consent; the other index
-- the consent state of a patient with a clinic, revocations included.
CREATE UNIQUE INDEX consent_one_active ON consent (patient_id, organization_id) WHERE status = 'active';
CREATE INDEX consent_patient_organization ON consent (patient_id, organization_id);

ALTER TABLE consent ENABLE ROW LEVEL SECURITY;
CREATE POLICY consent_own ON consent
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY consent_clinic ON consent FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- The clinic branch of patient: read-only, with an active consent for the clinic. Access ends when the
-- revocation commits. The past-appointment branch comes with the appointment table (package 6).
CREATE POLICY patient_consent ON patient FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM consent c
                       WHERE c.patient_id = patient.id
                         AND c.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                         AND c.status = 'active'));

-- No DELETE or TRUNCATE: consents are kept as proof (retention keep). Who and with whom never change; only
-- the revocation does.
GRANT SELECT, INSERT ON consent TO salus_app;
GRANT UPDATE (status, revoked_at, updated_at, version_id) ON consent TO salus_app;
