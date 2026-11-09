-- +goose Up
-- The live translation of the transcript (docs/plans/consult-intelligence.md, SAL-5):
--   - transcript_segment.translation_failed: the segment should have been translated (it was said in a language
--     other than the listener's) and was not, after a retry within the latency budget: translation is then
--     empty and the original is all there is. False for a segment that needed no translation. Personal data no
--     more than the segment itself is; retention as the segment's (keep, with the clinical record).
--   - term_label_language_folded: the glossary of each final segment looks up its phrases among one language's
--     labels, folded (case and accents aside), by equality: one index lookup per phrase.

ALTER TABLE transcript_segment ADD COLUMN translation_failed boolean DEFAULT false NOT NULL;

CREATE INDEX term_label_language_folded ON term_label (language, term_fold(label));
