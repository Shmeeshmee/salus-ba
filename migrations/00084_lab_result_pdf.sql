-- +goose Up
-- Lab result PDFs (docs/clinic-cms/WAVE6.md U5, the follow-up of LB1): the lab's PDF of a DiagnosticReport is a
-- "binary" row (ADR 0010, docs/attachments) with a third kind of parent, the report (diagnostic_report_id): encrypted
-- under its own data key, its type read from its bytes (a PDF only), scanned before anyone opens it. Health data
-- (Art. 9). A report holds at most one file that is not rejected: a new upload before the release replaces the one
-- there (its row deleted, its object after the commit). Once the report is released its file is frozen: no file is
-- added to it or deleted from it, for every role, the owner too (but a rejected one, whose bytes were never readable).
-- The file has no response, appointment, item or conversation; its clinic and patient are the report's (the foreign
-- key and the insert policy hold them).
-- Row-level security of the report files:
--   - the clinic's scope (app.organization_id, no patient, account, response or report job setting), as
--     diagnosticreport sets it,
--     sees the files of the reports it sees (the EXISTS applies diagnostic_report's own policies), and inserts and
--     deletes the file of its own report while it is preliminary;
--   - the patient (app.patient_id) sees the file of their report once it is released, never before;
--   - the scan job's scope (app.binary_diagnostic_report_id with app.organization_id) reads and updates the file of
--     that one report, whatever the release or the consent.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

ALTER TABLE "binary"
    ADD COLUMN diagnostic_report_id uuid,
    ADD CONSTRAINT binary_diagnostic_report_fkey FOREIGN KEY (diagnostic_report_id, organization_id)
        REFERENCES diagnostic_report (id, organization_id),
    DROP CONSTRAINT binary_parent,
    ADD CONSTRAINT binary_parent CHECK (
        (questionnaire_response_id IS NOT NULL AND appointment_id IS NOT NULL AND link_id IS NOT NULL
            AND conversation_id IS NULL AND conversation_participant_id IS NULL AND communication_id IS NULL
            AND diagnostic_report_id IS NULL)
        OR (questionnaire_response_id IS NULL AND appointment_id IS NULL AND link_id IS NULL
            AND conversation_id IS NOT NULL AND conversation_participant_id IS NOT NULL AND diagnostic_report_id IS NULL)
        OR (questionnaire_response_id IS NULL AND appointment_id IS NULL AND link_id IS NULL
            AND conversation_id IS NULL AND conversation_participant_id IS NULL AND communication_id IS NULL
            AND diagnostic_report_id IS NOT NULL)),
    ADD CONSTRAINT binary_report_type CHECK (diagnostic_report_id IS NULL OR content_type = 'application/pdf');

-- A report's file, and a list's files.
CREATE INDEX binary_diagnostic_report ON "binary" (diagnostic_report_id) WHERE diagnostic_report_id IS NOT NULL;

CREATE POLICY binary_report ON "binary" FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.questionnaire_response_id', true), '') IS NULL
           AND NULLIF(current_setting('app.binary_diagnostic_report_id', true), '') IS NULL
           AND diagnostic_report_id IS NOT NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM diagnostic_report r WHERE r.id = "binary".diagnostic_report_id));
CREATE POLICY binary_report_insert ON "binary" FOR INSERT
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND NULLIF(current_setting('app.questionnaire_response_id', true), '') IS NULL
                AND NULLIF(current_setting('app.binary_diagnostic_report_id', true), '') IS NULL
                AND diagnostic_report_id IS NOT NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM diagnostic_report r
                            WHERE r.id = "binary".diagnostic_report_id
                              AND r.organization_id = "binary".organization_id
                              AND r.patient_id = "binary".patient_id
                              AND r.status = 'preliminary'));
CREATE POLICY binary_report_delete ON "binary" FOR DELETE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.questionnaire_response_id', true), '') IS NULL
           AND NULLIF(current_setting('app.binary_diagnostic_report_id', true), '') IS NULL
           AND diagnostic_report_id IS NOT NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM diagnostic_report r
                       WHERE r.id = "binary".diagnostic_report_id AND r.status = 'preliminary'));
CREATE POLICY binary_report_own ON "binary" FOR SELECT
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
           AND diagnostic_report_id IS NOT NULL
           AND EXISTS (SELECT 1 FROM diagnostic_report r WHERE r.id = "binary".diagnostic_report_id AND r.status = 'final'));
CREATE POLICY binary_report_job ON "binary" FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND diagnostic_report_id = NULLIF(current_setting('app.binary_diagnostic_report_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
CREATE POLICY binary_report_job_update ON "binary" FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND diagnostic_report_id = NULLIF(current_setting('app.binary_diagnostic_report_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- A released report's file is frozen, for every role (the owner too): none is added, none deleted but a rejected one.
-- SECURITY DEFINER: the retention role, which deletes rejected rows, reads no report.
-- +goose StatementBegin
CREATE FUNCTION binary_report_frozen() RETURNS trigger
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
DECLARE
    row_ record;
BEGIN
    IF TG_OP = 'DELETE' THEN
        row_ := OLD;
        IF OLD.status = 'rejected' THEN
            RETURN OLD;
        END IF;
    ELSE
        row_ := NEW;
    END IF;
    IF row_.diagnostic_report_id IS NOT NULL
       AND EXISTS (SELECT 1 FROM public.diagnostic_report r WHERE r.id = row_.diagnostic_report_id AND r.status = 'final') THEN
        RAISE EXCEPTION 'binary: the file of a released report never changes' USING ERRCODE = 'check_violation';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER binary_report_frozen BEFORE INSERT OR DELETE ON "binary"
    FOR EACH ROW EXECUTE FUNCTION binary_report_frozen();
