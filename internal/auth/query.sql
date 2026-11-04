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
