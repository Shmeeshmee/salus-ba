-- +goose Up
-- The privacy foundation (docs/accounts, package 1): the application role salus_app, the grants it needs on
-- River's tables, and the append-only audit_event table (FHIR R4 AuditEvent, as columns).

-- salus_app is the role api, worker, notifier and scheduler log in as: LOGIN, not a superuser, no
-- BYPASSRLS, CREATEROLE or CREATEDB, owner of nothing, so grants and row-level security apply to it. No
-- password here: cmd/migrate up sets it from APP_DATABASE_PASSWORD. Roles are cluster-wide and test
-- packages migrate in parallel, so the role may exist already, or be created by another session at the same
-- moment (which raises unique_violation rather than duplicate_object). An existing role with any of the
-- refused attributes fails the migration rather than being used.
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE salus_app LOGIN NOSUPERUSER NOBYPASSRLS NOCREATEROLE NOCREATEDB;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'salus_app'
               AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR NOT rolcanlogin)) THEN
        RAISE EXCEPTION 'role salus_app exists with SUPERUSER, BYPASSRLS, CREATEROLE, CREATEDB or NOLOGIN';
    END IF;
END
$$;
-- +goose StatementEnd

-- River's client (worker, notifier, and api when it enqueues) reads and writes its job tables. Its pgx
-- driver only reads and prunes river_notification (an outbox for drivers without LISTEN/NOTIFY).
GRANT SELECT, INSERT, UPDATE, DELETE ON river_job, river_leader, river_queue TO salus_app;
GRANT SELECT, DELETE ON river_notification TO salus_app;
GRANT USAGE ON SEQUENCE river_job_id_seq TO salus_app;

-- One row per access to an account or directory record (SPEC.md "AuditEvent shape"). Codings are stored as
-- system + code. The agent is Account/<agent_account_id>, absent for a failed login with an unknown e-mail;
-- the entity is a <ResourceType>/<id> reference. No e-mail, name, IP address or user agent.
CREATE TABLE audit_event (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    recorded timestamptz NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    agent_account_id uuid,
    organization_id uuid,
    outcome smallint NOT NULL CHECK (outcome IN (0, 4, 8, 12)),
    action text NOT NULL CHECK (action IN ('C', 'R', 'U', 'D', 'E')),
    type_system text NOT NULL,
    type_code text NOT NULL,
    subtype_system text NOT NULL,
    subtype_code text NOT NULL,
    purpose_system text NOT NULL,
    purpose_code text NOT NULL,
    entity_reference text,
    request_id text
);

CREATE INDEX audit_event_entity_recorded_idx ON audit_event (entity_reference, recorded);
CREATE INDEX audit_event_agent_recorded_idx ON audit_event (agent_account_id, recorded);

-- Append-only for every role, the owner included. Retention removes old rows later by dropping whole time
-- ranges as the owner (SPEC.md section 6); the application can never remove one.
-- +goose StatementBegin
CREATE FUNCTION audit_event_reject_change() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
BEGIN
    RAISE EXCEPTION 'audit_event is append-only: % is not allowed', TG_OP;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER audit_event_no_update_delete
    BEFORE UPDATE OR DELETE ON audit_event
    FOR EACH ROW EXECUTE FUNCTION audit_event_reject_change();

CREATE TRIGGER audit_event_no_truncate
    BEFORE TRUNCATE ON audit_event
    FOR EACH STATEMENT EXECUTE FUNCTION audit_event_reject_change();

GRANT INSERT, SELECT ON audit_event TO salus_app;
