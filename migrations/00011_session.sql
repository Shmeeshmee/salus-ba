-- +goose Up
-- Sessions (docs/accounts, package 3): one row per sign-in, with the fields of SPEC.md section 4. The token
-- itself is never stored, only its SHA-256. Every request reads its session here by token hash
-- (revoked_at IS NULL AND expires_at > now); revoked_at is the only revocation (SPEC.md §2, changed on
-- approval). Not row-level secured, like account: a patient's session has no clinic, and the session is
-- found by its token before the clinic is known; the auth service checks organization_id.
CREATE TABLE session (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    -- Absolute lifetime, no idle timeout: 12 h for staff, 30 days for patients.
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    account_id uuid NOT NULL REFERENCES account (id),
    -- The clinic of a staff session; null for a patient's.
    organization_id uuid,
    -- Whether the sign-in passed a TOTP code. Staff cannot sign in without one.
    mfa_verified boolean NOT NULL,
    token_hash bytea NOT NULL,
    CONSTRAINT session_staff_mfa_verified CHECK (organization_id IS NULL OR mfa_verified)
);

CREATE UNIQUE INDEX session_token_hash_key ON session (token_hash);
CREATE INDEX session_account_id_idx ON session (account_id);

-- No DELETE or TRUNCATE: logout sets revoked_at; the retention job removes old rows as the owner.
GRANT SELECT, INSERT, UPDATE ON session TO salus_app;
