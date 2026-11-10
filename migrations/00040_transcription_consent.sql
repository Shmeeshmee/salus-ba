-- +goose Up
-- The patient's consent to transcription, translation and AI assistance in their consultations with a clinic
-- (SAL-12, docs/plans/v1-finish.md "Decisions": a separate, granular, revocable FHIR Consent per patient and
-- clinic, beside the care consent of 00016). The transcription agent joins a consultation only while both are
-- active; the suggestions and the drafting run only while both are. Same shape and rules as consent: at most one
-- active per patient and clinic, a grant after a withdrawal adds a row (the history is proof, Art. 7(1)), the
-- same two row-level security scopes. A table of its own rather than a kind column on consent: the policies of
-- patient, appointment, encounter, video_session and questionnaire_response read consent as the care consent, and
-- a transcription consent must never open them.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE transcription_consent (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    date_time timestamptz NOT NULL, -- granted
    revoked_at timestamptz,
    patient_id uuid NOT NULL REFERENCES patient (id),
    organization_id uuid NOT NULL REFERENCES organization (id),
    version_id integer DEFAULT 1 NOT NULL,
    status text NOT NULL CHECK (status IN ('active', 'inactive')),
    CONSTRAINT transcription_consent_revoked_when_inactive CHECK ((status = 'inactive') = (revoked_at IS NOT NULL))
);

CREATE UNIQUE INDEX transcription_consent_one_active ON transcription_consent (patient_id, organization_id) WHERE status = 'active';
CREATE INDEX transcription_consent_patient_organization ON transcription_consent (patient_id, organization_id);

ALTER TABLE transcription_consent ENABLE ROW LEVEL SECURITY;
CREATE POLICY transcription_consent_own ON transcription_consent
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY transcription_consent_clinic ON transcription_consent FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- No DELETE or TRUNCATE: kept as proof (retention keep). Who and with whom never change; only the withdrawal does.
GRANT SELECT, INSERT ON transcription_consent TO salus_app;
GRANT UPDATE (status, revoked_at, updated_at, version_id) ON transcription_consent TO salus_app;
