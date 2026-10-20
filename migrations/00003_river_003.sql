-- +goose Up
-- River main migration 003, verbatim from `river migrate-get --version 3 --up` (River v0.49.0).
-- River's own ledger (version 001) is left out: goose tracks these files. Do not edit; a River upgrade
-- adds the new versions as new files.
-- +goose StatementBegin
ALTER TABLE river_job ALTER COLUMN tags SET DEFAULT '{}';
UPDATE river_job SET tags = '{}' WHERE tags IS NULL;
ALTER TABLE river_job ALTER COLUMN tags SET NOT NULL;
-- +goose StatementEnd
