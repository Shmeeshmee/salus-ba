-- +goose Up
-- The shared-visit policies of encounter, document_draft, questionnaire_response and binary (00043), with
-- history_shared's own scope conditions written in front of the call. 00043 called history_shared for every row a
-- read passes over in every scope: a SQL function with SET is never inlined, so even where it is false at once (a
-- patient's or an account's scope, no clinic set) each row paid a function call, and document_draft_shared and
-- binary_shared an EXISTS over encounter first. The patient's timeline reads every one of these tables in the
-- patient's scope (SAL-39, docs/plans/patient-history.PROGRESS.md).
--
-- The conditions are history_shared's first two (no patient, no account) and the clinic it compares with (set):
-- with app.organization_id unset, history_shared compares h.organization_id with NULL and is false. So each policy
-- admits exactly the rows 00043's admitted (test TestHistorySharing_TheGuardedPoliciesAdmitWhat00043Admitted, every
-- scope and sharing state); only where the conditions hold is history_shared called, as before.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

DROP POLICY encounter_shared ON encounter;
CREATE POLICY encounter_shared ON encounter FOR SELECT TO salus_app
    USING (status = 'finished'
           AND NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.organization_id', true), '') IS NOT NULL
           AND history_shared(id, patient_id));

DROP POLICY document_draft_shared ON document_draft;
CREATE POLICY document_draft_shared ON document_draft FOR SELECT TO salus_app
    USING (status = 'final'
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.organization_id', true), '') IS NOT NULL
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.id = document_draft.encounter_id AND history_shared(e.id, e.patient_id)));

DROP POLICY questionnaire_response_shared ON questionnaire_response;
CREATE POLICY questionnaire_response_shared ON questionnaire_response FOR SELECT TO salus_app
    USING (status = 'completed'
           AND NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.organization_id', true), '') IS NOT NULL
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.appointment_id = questionnaire_response.appointment_id
                         AND e.patient_id = questionnaire_response.patient_id
                         AND history_shared(e.id, e.patient_id)));

DROP POLICY binary_shared ON "binary";
CREATE POLICY binary_shared ON "binary" FOR SELECT TO salus_app
    USING (status = 'clean'
           AND NULLIF(current_setting('app.questionnaire_response_id', true), '') IS NULL
           AND NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.organization_id', true), '') IS NOT NULL
           AND EXISTS (SELECT 1 FROM questionnaire_response r
                       WHERE r.id = "binary".questionnaire_response_id AND r.status = 'completed')
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.appointment_id = "binary".appointment_id
                         AND e.patient_id = "binary".patient_id
                         AND history_shared(e.id, e.patient_id)));
