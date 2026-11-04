-- +goose Up
-- Invites (docs/accounts, package 5): an admin invites a practitioner or another admin by e-mail, with the
-- fields of SPEC.md section 4. Clinic-scoped and row-level secured like the directory (package 1's NULLIF
-- form). The invite_email job issues the token: only its SHA-256 is stored, replacing any earlier one, so no
-- raw token is at rest. An open invite (not accepted) for the same e-mail in the same clinic is replaced, not
-- duplicated. Accepting an invite has to find its clinic before the clinic is known: one SECURITY DEFINER
-- function returns only the clinic id of a token hash, owned by salus_invite_lookup, a role that can read the
-- two columns it needs and nothing else (SPEC.md §2 "Before the clinic is known").

-- An agent that is no account: the operator command (salusctl). FHIR AuditEvent.agent.name.
ALTER TABLE audit_event ADD COLUMN agent_name text;

CREATE TABLE invite (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    -- 72 hours after the invite was made (or last replaced).
    expires_at timestamptz NOT NULL,
    accepted_at timestamptz,
    organization_id uuid NOT NULL REFERENCES organization (id),
    -- The admin who invited; null for the first admin, invited by the operator command.
    invited_by uuid REFERENCES account (id),
    role text NOT NULL CHECK (role IN ('admin', 'practitioner')),
    -- Where the invite goes, and the login the new account gets: trimmed and lower-cased like account.email.
    email text NOT NULL CHECK (email = lower(btrim(email))),
    -- The Practitioner's one HumanName, created on acceptance; empty for an admin.
    given_name text DEFAULT '' NOT NULL,
    family_name text DEFAULT '' NOT NULL,
    -- SHA-256 of the token in the e-mailed link; null until the job issues one.
    token_hash bytea,
    CONSTRAINT invite_name_practitioner_only CHECK (
        (role = 'practitioner' AND btrim(given_name) <> '' AND btrim(family_name) <> '')
        OR (role = 'admin' AND given_name = '' AND family_name = ''))
);

-- One open invite per e-mail and clinic: a new one replaces it (INSERT … ON CONFLICT … DO UPDATE). Also the
-- index of the clinic's open invites (GET /invites).
CREATE UNIQUE INDEX invite_open_email_key ON invite (organization_id, email) WHERE accepted_at IS NULL;
CREATE UNIQUE INDEX invite_token_hash_key ON invite (token_hash);

ALTER TABLE invite ENABLE ROW LEVEL SECURITY;
CREATE POLICY invite_clinic ON invite
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- No DELETE or TRUNCATE: the retention job removes old invites as the owner. The e-mail and the clinic of an
-- invite never change; a replacement updates the rest.
GRANT SELECT, INSERT ON invite TO salus_app;
GRANT UPDATE (created_at, expires_at, accepted_at, invited_by, role, given_name, family_name, token_hash)
    ON invite TO salus_app;

-- The owner of the lookup function: NOLOGIN, not a superuser, no BYPASSRLS. Cluster-wide like salus_app, so
-- the same DO block (test packages migrate in parallel).
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE salus_invite_lookup NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEROLE NOCREATEDB;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'salus_invite_lookup'
               AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolcanlogin)) THEN
        RAISE EXCEPTION 'role salus_invite_lookup exists with SUPERUSER, BYPASSRLS, CREATEROLE, CREATEDB or LOGIN';
    END IF;
END
$$;
-- +goose StatementEnd

GRANT SELECT (organization_id, token_hash) ON invite TO salus_invite_lookup;
CREATE POLICY invite_lookup ON invite FOR SELECT TO salus_invite_lookup USING (true);

-- The clinic of the invite with the token hash, or null. Nothing else about the invite leaves the function;
-- the service then sets the clinic and reads the invite through row-level security.
CREATE FUNCTION invite_organization(token_hash bytea) RETURNS uuid
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
    SELECT i.organization_id FROM public.invite AS i WHERE i.token_hash = invite_organization.token_hash
$$;

ALTER FUNCTION invite_organization(bytea) OWNER TO salus_invite_lookup;
REVOKE EXECUTE ON FUNCTION invite_organization(bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION invite_organization(bytea) TO salus_app;
