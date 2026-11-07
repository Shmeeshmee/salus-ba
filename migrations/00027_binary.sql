-- +goose Up
-- Questionnaire attachments (docs/attachments, package 0.1; SPEC.md §2 "Tables", §5): FHIR Binary, a file the
-- patient attached to an answer of a questionnaire response. Health data. The bytes live encrypted in the object
-- store under the row's id; the row holds the wrapped data key, so deleting the row makes the stored ciphertext
-- unreadable (crypto-shredding). "binary" is a reserved word in Postgres and is quoted everywhere.
-- organization_id, patient_id and appointment_id are copied once from the response (the holder's Target), for
-- row-level security and the audit; the patient's insert policy checks they are the response's.
-- Row-level security:
--   - the patient (app.patient_id) sees and deletes their own files and inserts files for themselves, on their
--     own response;
--   - a clinic (app.organization_id, no patient, account or response setting) sees the files of its clinic whose
--     questionnaire_response it can see: the EXISTS applies that table's own policies, so consent and D9 follow
--     without being written twice;
--   - the job scope (app.questionnaire_response_id with app.organization_id, set by binary's scan and discard
--     jobs) sees, updates and deletes the files of that one response, whatever the consent.
-- A response's deletion, by any path, deletes its files' rows through the foreign key (the referential action
-- runs as the table's owner, so salus_retention needs no privilege here while the rule is keep).
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE "binary" (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    scanned_at timestamptz,
    organization_id uuid NOT NULL,
    patient_id uuid NOT NULL,
    appointment_id uuid NOT NULL,
    questionnaire_response_id uuid NOT NULL REFERENCES questionnaire_response (id) ON DELETE CASCADE,
    size bigint NOT NULL CHECK (size > 0), -- plaintext bytes
    segment_size integer NOT NULL CHECK (segment_size > 0),
    link_id text NOT NULL,
    content_type text NOT NULL CHECK (content_type IN ('application/pdf', 'image/jpeg', 'image/png', 'video/mp4')),
    title text NOT NULL CHECK (char_length(title) <= 200),
    status text DEFAULT 'quarantined' NOT NULL CHECK (status IN ('quarantined', 'clean', 'rejected')),
    kek_id text NOT NULL,
    sha1 bytea NOT NULL CHECK (octet_length(sha1) = 20),
    wrapped_key bytea, -- cleared when the scan rejects the file
    CONSTRAINT binary_scanned CHECK ((status = 'quarantined') = (scanned_at IS NULL)),
    CONSTRAINT binary_key CHECK ((status = 'rejected') = (wrapped_key IS NULL))
);

-- An item's files (the count per item, the identical-retry check) and, by its prefix, a response's.
CREATE INDEX binary_response_item ON "binary" (questionnaire_response_id, link_id);

ALTER TABLE "binary" ENABLE ROW LEVEL SECURITY;
CREATE POLICY binary_own ON "binary" FOR SELECT
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY binary_own_insert ON "binary" FOR INSERT
    WITH CHECK (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM questionnaire_response r
                            WHERE r.id = "binary".questionnaire_response_id
                              AND r.patient_id = "binary".patient_id
                              AND r.organization_id = "binary".organization_id
                              AND r.appointment_id = "binary".appointment_id));
CREATE POLICY binary_own_delete ON "binary" FOR DELETE
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY binary_clinic ON "binary" FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.questionnaire_response_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM questionnaire_response r WHERE r.id = "binary".questionnaire_response_id));
CREATE POLICY binary_job ON "binary" FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND questionnaire_response_id = NULLIF(current_setting('app.questionnaire_response_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
CREATE POLICY binary_job_update ON "binary" FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND questionnaire_response_id = NULLIF(current_setting('app.questionnaire_response_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
CREATE POLICY binary_job_delete ON "binary" FOR DELETE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND questionnaire_response_id = NULLIF(current_setting('app.questionnaire_response_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- What a file is, whose and where never changes; the scan sets its status, its scan time and, on a rejection,
-- clears the key. salus_retention gets SELECT and DELETE with the delete rule and its purger (package 3.1).
GRANT SELECT, INSERT, DELETE ON "binary" TO salus_app;
GRANT UPDATE (status, scanned_at, wrapped_key, updated_at) ON "binary" TO salus_app;

-- The answer to an attachment item: {linkId, binaryId, contentType, size, title, hash, pending} per file,
-- written only through binary's holder calls, outside answers and version_id (SPEC.md §2 "Where an attachment
-- answer lives"). Empty once withdrawn, like the answers.
ALTER TABLE questionnaire_response
    ADD COLUMN attachments jsonb DEFAULT '[]' NOT NULL CHECK (jsonb_typeof(attachments) = 'array'),
    ADD CONSTRAINT questionnaire_response_withdrawn_attachments CHECK (status <> 'stopped' OR attachments = '[]');
GRANT UPDATE (attachments) ON questionnaire_response TO salus_app;

-- The scan and discard jobs' holder calls: read and update that one response of that clinic, whatever the
-- consent (a file is scanned or discarded after a revocation too).
CREATE POLICY questionnaire_response_job ON questionnaire_response FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND id = NULLIF(current_setting('app.questionnaire_response_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
CREATE POLICY questionnaire_response_job_update ON questionnaire_response FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND id = NULLIF(current_setting('app.questionnaire_response_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
