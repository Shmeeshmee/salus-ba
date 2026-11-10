-- +goose Up
-- The patient's signed documents and transcript copy (SAL-17, docs/consultation/SPEC.md "Patient documents").
--   - organization.patient_transcript_copy: whether the clinic gives its patients a copy of their consultations'
--     stored transcripts in the app. Off unless the clinic's admin turns it on. No personal data.
--   - document_draft_own: the patient's own scope (app.patient_id) sees the final (signed) documents of their own
--     encounters, never a preliminary or discarded draft. Whatever the consent: a signed document is the
--     patient's own record (GDPR Art. 15), which a revocation does not take from them; the service applies the
--     same rule (access.SignedDocument). Read-only.
--   - transcript_segment_own: the patient's own scope sees the stored segments of their own encounters. Whether the
--     clinic shares them (patient_transcript_copy) is the service's check (access.TranscriptCopy): the patient's
--     scope does not see the clinic's row. Read-only.
-- A video room's scope sees neither (as in 00031). No new table; the retention of both tables is unchanged (keep).

ALTER TABLE organization ADD COLUMN patient_transcript_copy boolean DEFAULT false NOT NULL;

GRANT UPDATE (patient_transcript_copy) ON organization TO salus_app;

CREATE POLICY document_draft_own ON document_draft FOR SELECT
    USING (status = 'final'
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.id = document_draft.encounter_id
                         AND e.patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid));

CREATE POLICY transcript_segment_own ON transcript_segment FOR SELECT
    USING (NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.id = transcript_segment.encounter_id
                         AND e.patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid));
