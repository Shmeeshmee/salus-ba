-- +goose Up
-- Versioned appointment settings (post-appointments gaps, SAL-32): the clinic's settings row is replaced whole, so
-- each save names the version it was read at and is refused (412) when another save came first. The number is a
-- placeholder: it takes the next free number when it is merged.
--   - organization_appointment_settings.version: 1 when the row is first stored, one up with every save. A clinic
--     with no row reads as version 0. No personal data.

ALTER TABLE organization_appointment_settings
    ADD COLUMN version integer DEFAULT 1 NOT NULL CHECK (version >= 1);

GRANT UPDATE (version) ON organization_appointment_settings TO salus_app;
