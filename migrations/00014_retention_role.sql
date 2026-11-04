-- +goose Up
-- The retention job's role (docs/booking, package 1; SPEC.md §2 "Retention privileges"). salus_app cannot
-- delete sessions or invites (00011, 00013), and invite is row-level secured, so a job without a clinic sees
-- none. salus_retention logs in for the retention_apply job only (its own credentials,
-- RETENTION_DATABASE_URL, which only cmd/worker loads): BYPASSRLS, not a superuser, no CREATEROLE or
-- CREATEDB, owner of nothing. No password here: cmd/migrate up sets it from RETENTION_DATABASE_PASSWORD.
-- Cluster-wide like salus_app, so the same DO block: test packages migrate in parallel, and an existing role
-- with other attributes fails the migration rather than being used.
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE salus_retention LOGIN NOSUPERUSER BYPASSRLS NOCREATEROLE NOCREATEDB;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'salus_retention'
               AND (rolsuper OR NOT rolbypassrls OR rolcreaterole OR rolcreatedb OR NOT rolcanlogin)) THEN
        RAISE EXCEPTION 'role salus_retention exists with SUPERUSER, CREATEROLE, CREATEDB, NOLOGIN or without BYPASSRLS';
    END IF;
END
$$;
-- +goose StatementEnd

-- Exactly the tables whose retention rule is delete (internal/retention), and nothing else: SELECT because
-- a DELETE … WHERE reads the columns it filters on. A later migration that adds a delete rule grants its
-- table here in the same file. No INSERT, UPDATE or TRUNCATE, and nothing on audit_event.
GRANT SELECT, DELETE ON session, invite TO salus_retention;
