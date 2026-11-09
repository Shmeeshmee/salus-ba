-- +goose Up
-- The speaker who is not the patient is the 'practitioner', as FHIR and the rest of the codebase name them, not the
-- 'doctor' of 00031 (internal/transcript, api/openapi.yaml TranscriptEvent). No code writes transcript_segment yet
-- (the consumers come with SAL-6); the UPDATE carries over any row written by hand so the new check can be added.
-- Idempotent: it rewrites only 'doctor' rows, and the check is dropped and added by name.

ALTER TABLE transcript_segment DROP CONSTRAINT IF EXISTS transcript_segment_speaker_check;

UPDATE transcript_segment SET speaker = 'practitioner' WHERE speaker = 'doctor';

ALTER TABLE transcript_segment
    ADD CONSTRAINT transcript_segment_speaker_check CHECK (speaker IN ('practitioner', 'patient'));
