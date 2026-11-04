-- name: CreateAccount :one
-- DO NOTHING on a taken e-mail returns no row and leaves the transaction usable, so the failure can be
-- audited in it.
INSERT INTO account (email, password_hash, role)
VALUES (@email, @password_hash, @role)
ON CONFLICT (email) DO NOTHING
RETURNING id;

-- name: AccountByEmail :one
SELECT id, mfa_enrolled_at, organization_id, role, email, password_hash
FROM account
WHERE email = @email;

-- name: AccountByID :one
SELECT id, mfa_enrolled_at, organization_id, role, email, password_hash
FROM account
WHERE id = @id;

-- name: SetPasswordHash :execrows
UPDATE account SET password_hash = @password_hash WHERE id = @id;

-- name: TOTPSecret :one
SELECT totp_secret FROM account WHERE id = @id;

-- name: SetTOTPSecret :execrows
UPDATE account SET totp_secret = @totp_secret WHERE id = @id;

-- name: ConfirmMFA :execrows
-- Only once: an enrolled account keeps its secret until reset.
UPDATE account SET mfa_enrolled_at = @enrolled_at, totp_last_step = @step::bigint
WHERE id = @id AND mfa_enrolled_at IS NULL;

-- name: UseTOTPStep :execrows
-- A step is used at most once: a code is rejected unless its step is newer than the last used one.
UPDATE account SET totp_last_step = @step::bigint
WHERE id = @id AND (totp_last_step IS NULL OR totp_last_step < @step::bigint);

-- name: CreateSession :one
INSERT INTO session (created_at, expires_at, account_id, organization_id, mfa_verified, token_hash)
VALUES (@created_at, @expires_at, @account_id, @organization_id, @mfa_verified, @token_hash)
RETURNING id;

-- name: SessionByTokenHash :one
-- The per-request check: one read on the unique token_hash index.
SELECT id, expires_at, account_id, organization_id, mfa_verified
FROM session
WHERE token_hash = @token_hash AND revoked_at IS NULL AND expires_at > @now;

-- name: RevokeSession :execrows
UPDATE session SET revoked_at = @revoked_at WHERE id = @id AND revoked_at IS NULL;
