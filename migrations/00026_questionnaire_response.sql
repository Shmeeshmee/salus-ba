-- +goose Up
-- Questionnaire responses (docs/questionnaire, package 3; SPEC.md §2 "Tables", "Requests", "Patient flow", §5):
-- FHIR QuestionnaireResponse, a request to the patient to fill in one questionnaire version before one
-- appointment, and the patient's answers. Health data. organization_id, patient_id, practitioner_role_id and
-- due_at (the appointment's start) are copied once from the appointment, as encounter's are: row-level
-- security and the retention purger need them, and salus_retention reads only this table.
-- Row-level security:
--   - the patient (app.patient_id) sees their own responses and updates them (the answers); they never
--     insert or delete one;
--   - a clinic (app.organization_id, neither patient setting set) sees, requests and withdraws the responses
--     of its patients with an active consent with it, and, after a revocation, sees only the completed ones of
--     appointments that started before the latest one (D9);
--   - the notifier's notice scope (app.organization_id with app.appointment_id) reads the responses of that
--     one appointment of its clinic, whatever the consent;
--   - the discard job (the same two settings) deletes the responses of that one appointment once it is
--     cancelled (D8); nothing else deletes, apart from salus_retention's purger.
-- And the patient scope's policy on questionnaire: the versions the patient's own responses reference.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE questionnaire_response (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    due_at timestamptz NOT NULL,
    authored_at timestamptz,
    organization_id uuid NOT NULL,
    appointment_id uuid NOT NULL REFERENCES appointment (id),
    patient_id uuid NOT NULL REFERENCES patient (id),
    practitioner_role_id uuid NOT NULL,
    questionnaire_id uuid NOT NULL,
    series_id uuid NOT NULL,
    version_id integer DEFAULT 1 NOT NULL,
    source text NOT NULL CHECK (source IN ('default', 'added')),
    status text DEFAULT 'in-progress' NOT NULL CHECK (status IN ('in-progress', 'completed', 'stopped')),
    -- A flat list of {linkId, values}; the FHIR handler nests it under the version's sections.
    answers jsonb DEFAULT '[]' NOT NULL CHECK (jsonb_typeof(answers) = 'array'),
    CONSTRAINT questionnaire_response_practitioner_role_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT questionnaire_response_questionnaire_fkey FOREIGN KEY (questionnaire_id, organization_id)
        REFERENCES questionnaire (id, organization_id),
    CONSTRAINT questionnaire_response_authored CHECK ((status = 'completed') = (authored_at IS NOT NULL)),
    CONSTRAINT questionnaire_response_withdrawn CHECK (status <> 'stopped' OR answers = '[]')
);

-- One live request per questionnaire per appointment; a withdrawn one may be requested again.
CREATE UNIQUE INDEX questionnaire_response_one_live ON questionnaire_response (appointment_id, series_id)
    WHERE status <> 'stopped';
-- QuestionnaireResponse?patient=…, and the purger's unsubmitted rows by due time.
CREATE INDEX questionnaire_response_patient ON questionnaire_response (patient_id);
CREATE INDEX questionnaire_response_unsubmitted_due ON questionnaire_response (due_at) WHERE status <> 'completed';

ALTER TABLE questionnaire_response ENABLE ROW LEVEL SECURITY;
CREATE POLICY questionnaire_response_own ON questionnaire_response FOR SELECT
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY questionnaire_response_own_update ON questionnaire_response FOR UPDATE
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY questionnaire_response_clinic ON questionnaire_response FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND (EXISTS (SELECT 1 FROM consent c
                        WHERE c.patient_id = questionnaire_response.patient_id
                          AND c.organization_id = questionnaire_response.organization_id
                          AND c.status = 'active')
                OR (status = 'completed'
                    AND due_at < (SELECT max(c.revoked_at) FROM consent c
                                  WHERE c.patient_id = questionnaire_response.patient_id
                                    AND c.organization_id = questionnaire_response.organization_id))));
CREATE POLICY questionnaire_response_clinic_insert ON questionnaire_response FOR INSERT
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM consent c
                            WHERE c.patient_id = questionnaire_response.patient_id
                              AND c.organization_id = questionnaire_response.organization_id
                              AND c.status = 'active'));
CREATE POLICY questionnaire_response_clinic_update ON questionnaire_response FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM consent c
                       WHERE c.patient_id = questionnaire_response.patient_id
                         AND c.organization_id = questionnaire_response.organization_id
                         AND c.status = 'active'));
CREATE POLICY questionnaire_response_notice ON questionnaire_response FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND appointment_id = NULLIF(current_setting('app.appointment_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- The discard job's: the appointment's own notice policy (00019) shows it the cancelled appointment.
CREATE POLICY questionnaire_response_discard ON questionnaire_response FOR DELETE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND appointment_id = NULLIF(current_setting('app.appointment_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM appointment a
                       WHERE a.id = questionnaire_response.appointment_id AND a.status = 'cancelled'));

-- The patient's read of the versions their own responses reference (SPEC.md §2 "FHIR"); read-only.
CREATE POLICY questionnaire_patient ON questionnaire FOR SELECT
    USING (EXISTS (SELECT 1 FROM questionnaire_response r
                   WHERE r.questionnaire_id = questionnaire.id
                     AND r.patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid));

-- Which appointment, patient, role, clinic and version a response is, its source and due time never change;
-- the status, the answers and the submit time do. DELETE for the discard job, bound by its policy above.
GRANT SELECT, INSERT, DELETE ON questionnaire_response TO salus_app;
GRANT UPDATE (status, answers, authored_at, updated_at, version_id) ON questionnaire_response TO salus_app;
-- The retention rule is delete (internal/retention): unsubmitted responses once due.
GRANT SELECT, DELETE ON questionnaire_response TO salus_retention;
