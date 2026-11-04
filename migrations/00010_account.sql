-- +goose Up
-- Accounts (docs/accounts, package 2): one row per person who can sign in, with the fields of SPEC.md
-- section 4. Not clinic-scoped and not row-level secured: a patient has no clinic and login finds the account
-- by e-mail first; the auth service checks organization_id. organization_id and practitioner_id reference
-- tables of the directory, which arrive in package 4.
CREATE TABLE account (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    mfa_enrolled_at timestamptz,
    totp_last_step bigint,
    organization_id uuid,
    practitioner_id uuid,
    role text NOT NULL CHECK (role IN ('admin', 'practitioner', 'patient')),
    -- The login: trimmed and lower-cased by the service; the check makes sure nothing else is stored.
    email text NOT NULL CHECK (email = lower(btrim(email))),
    -- argon2id in PHC form; the password itself is never stored.
    password_hash text NOT NULL,
    -- AES-256-GCM sealed TOTP secret (package 3).
    totp_secret bytea,
    -- Staff belong to exactly one clinic; a patient to none.
    CONSTRAINT account_organization_staff_only CHECK ((organization_id IS NULL) = (role = 'patient'))
);

CREATE UNIQUE INDEX account_email_key ON account (email);

-- No DELETE or TRUNCATE: closing and erasing an account come with the subject-rights feature.
GRANT SELECT, INSERT, UPDATE ON account TO salus_app;
