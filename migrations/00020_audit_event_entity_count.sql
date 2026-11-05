-- +goose Up
-- The retention job's audit (docs/booking/SPEC.md §2 "Applying the rules"): one event per run per table it
-- deletes from, about the table rather than a resource. AuditEvent.entity.name is the table and
-- entity.detail "rows" the count deleted, zero included. Both or neither; a count is never negative. No
-- personal data: a table name and a number.
ALTER TABLE audit_event
    ADD COLUMN entity_name text,
    ADD COLUMN entity_count bigint,
    ADD CONSTRAINT audit_event_entity_count CHECK ((entity_name IS NULL) = (entity_count IS NULL) AND entity_count >= 0);
