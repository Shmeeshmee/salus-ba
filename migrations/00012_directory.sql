-- +goose Up
-- The clinic directory (docs/accounts, package 4): FHIR Organization, Practitioner and PractitionerRole with
-- the fields of SPEC.md section 2 "Directory model" and section 4. Every table is clinic-scoped and row-level
-- secured: the service sets app.organization_id on its transaction and a policy shows only that clinic's
-- rows. NULLIF: after a transaction that set the clinic ends, a pooled connection reads the setting as '',
-- which must match nothing rather than fail (docs/accounts/PROGRESS.md, package 1). The application role owns
-- none of these tables, so RLS applies to it without FORCE; the owner (migrate, salusctl) is not bound by it.
-- updated_at and version_id are meta.lastUpdated and meta.versionId.

CREATE TABLE organization (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    version_id integer DEFAULT 1 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    name text NOT NULL CHECK (btrim(name) <> '')
);

ALTER TABLE organization ENABLE ROW LEVEL SECURITY;
-- A clinic is its own scope: the policy compares the id.
CREATE POLICY organization_clinic ON organization
    USING (id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- Clinics are created by the operator command as the owner; the application renames its own (clinic settings).
GRANT SELECT ON organization TO salus_app;
GRANT UPDATE (name, updated_at, version_id) ON organization TO salus_app;

CREATE TABLE practitioner (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    version_id integer DEFAULT 1 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    -- One HumanName: given and family.
    given_name text NOT NULL,
    family_name text NOT NULL,
    -- For practitioner_role's reference, so a role is always in its practitioner's clinic.
    CONSTRAINT practitioner_id_organization_key UNIQUE (id, organization_id)
);

ALTER TABLE practitioner ENABLE ROW LEVEL SECURITY;
CREATE POLICY practitioner_clinic ON practitioner
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- Created when an invite is accepted (package 5). No UPDATE yet (deactivation is out of scope), no DELETE:
-- practitioners are restricted, not deleted, while records reference them.
GRANT SELECT, INSERT ON practitioner TO salus_app;

CREATE TABLE practitioner_role (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    practitioner_id uuid NOT NULL,
    version_id integer DEFAULT 1 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    CONSTRAINT practitioner_role_practitioner_fkey FOREIGN KEY (practitioner_id, organization_id)
        REFERENCES practitioner (id, organization_id)
);

-- A practitioner has one role, in their clinic.
CREATE UNIQUE INDEX practitioner_role_practitioner_id_key ON practitioner_role (practitioner_id);
-- PractitionerRole?organization=…
CREATE INDEX practitioner_role_organization_id_idx ON practitioner_role (organization_id);

ALTER TABLE practitioner_role ENABLE ROW LEVEL SECURITY;
CREATE POLICY practitioner_role_clinic ON practitioner_role
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

GRANT SELECT, INSERT ON practitioner_role TO salus_app;
