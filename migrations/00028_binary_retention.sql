-- +goose Up
-- Questionnaire attachments, package 3.1 (docs/attachments/SPEC.md §5 "Retention", §3 O16): the retention rule of
-- "binary" becomes delete, 24 h, with binary's purger, so salus_retention gets SELECT and DELETE on it, as on every
-- table whose rule is delete (00014). The purger deletes the rejected rows 24 h after their scan, and reads the
-- rows of every clinic (BYPASSRLS) to tell which stored objects no longer have one. No INSERT, UPDATE or TRUNCATE.
GRANT SELECT, DELETE ON "binary" TO salus_retention;

-- The purger's rejected rows past their period, without reading every file.
CREATE INDEX binary_rejected_scanned ON "binary" (scanned_at) WHERE status = 'rejected';
