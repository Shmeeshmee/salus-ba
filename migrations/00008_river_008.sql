-- +goose Up
-- River main migration 008, verbatim from `river migrate-get --version 8 --up` (River v0.49.0).
-- River's own ledger (version 001) is left out: goose tracks these files. Do not edit; a River upgrade
-- adds the new versions as new files.
-- +goose StatementBegin
-- No-op. PostgreSQL sequences already prevent automatically generated job IDs
-- from being reused.
SELECT 1;
-- +goose StatementEnd
