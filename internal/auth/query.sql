-- name: CreateAccount :one
-- DO NOTHING on a taken e-mail returns no row and leaves the transaction usable, so the failure can be
-- audited in it.
INSERT INTO account (email, password_hash, role, organization_id, language, detected_language)
VALUES (@email, @password_hash, @role, sqlc.narg('organization_id'), sqlc.narg('language'), sqlc.narg('detected_language'))
ON CONFLICT (email) DO NOTHING
RETURNING id;

-- name: AccountByEmail :one
SELECT id, mfa_enrolled_at, organization_id, practitioner_id, role, email, password_hash, language, detected_language,
       EXISTS (SELECT FROM term_reviewer WHERE account_id = account.id) AS term_reviewer
FROM account
WHERE email = @email;

-- name: AccountByID :one
SELECT id, mfa_enrolled_at, organization_id, practitioner_id, role, email, password_hash, language, detected_language,
       EXISTS (SELECT FROM term_reviewer WHERE account_id = account.id) AS term_reviewer
FROM account
WHERE id = @id;

-- name: SetPasswordHash :execrows
UPDATE account SET password_hash = @password_hash WHERE id = @id;

-- name: SetAccountLanguage :execrows
-- The user's choice; null clears it (the client's match applies again).
UPDATE account SET language = sqlc.narg('language') WHERE id = @id;

-- name: SetDetectedLanguage :execrows
-- What the client matched at a registration or sign-in.
UPDATE account SET detected_language = @detected_language WHERE id = @id;

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

-- name: RevokeSessions :exec
UPDATE session SET revoked_at = @revoked_at WHERE account_id = @account_id AND revoked_at IS NULL;

-- name: ResetMFA :execrows
UPDATE account SET totp_secret = NULL, totp_last_step = NULL, mfa_enrolled_at = NULL WHERE id = @id;

-- name: SetPractitioner :execrows
UPDATE account SET practitioner_id = @practitioner_id WHERE id = @id;

-- name: PutInvite :one
-- One open invite per e-mail and clinic: a new one replaces it, token included.
INSERT INTO invite (created_at, expires_at, organization_id, invited_by, role, email, given_name, family_name)
VALUES (@created_at, @expires_at, @organization_id, sqlc.narg('invited_by'), @role, @email, @given_name, @family_name)
ON CONFLICT (organization_id, email) WHERE accepted_at IS NULL
DO UPDATE SET created_at = excluded.created_at, expires_at = excluded.expires_at, invited_by = excluded.invited_by,
    role = excluded.role, given_name = excluded.given_name, family_name = excluded.family_name, token_hash = NULL
RETURNING id;

-- name: OpenInvites :many
SELECT id, created_at, expires_at, accepted_at, organization_id, invited_by, role, email, given_name, family_name
FROM invite
WHERE organization_id = @organization_id AND accepted_at IS NULL AND expires_at > @now
ORDER BY created_at, id;

-- name: InviteByID :one
SELECT id, created_at, expires_at, accepted_at, organization_id, invited_by, role, email, given_name, family_name
FROM invite
WHERE id = @id;

-- name: InviteByTokenHash :one
SELECT id, created_at, expires_at, accepted_at, organization_id, invited_by, role, email, given_name, family_name
FROM invite
WHERE token_hash = @token_hash;

-- name: SetInviteToken :execrows
UPDATE invite SET token_hash = @token_hash WHERE id = @id;

-- name: InviteOrganization :one
-- SECURITY DEFINER: the only way to find an invite's clinic before the clinic is set.
SELECT invite_organization(@token_hash::bytea)::uuid AS organization_id;

-- name: AcceptInvite :execrows
UPDATE invite SET accepted_at = @accepted_at
WHERE id = @id AND accepted_at IS NULL AND expires_at > @accepted_at;

-- name: DeleteSessionsEndedBefore :execrows
-- Retention (salus_retention): a session ends at revoked_at or expires_at, whichever comes first (least()
-- skips a null revoked_at).
DELETE FROM session WHERE least(revoked_at, expires_at) < @before;

-- name: DeleteInvitesEndedBefore :execrows
-- Retention (salus_retention, past row-level security): an invite ends at accepted_at or expires_at,
-- whichever comes first.
DELETE FROM invite WHERE least(accepted_at, expires_at) < @before;

-- name: GrantTermReviewer :exec
-- The operator's (salus_app may only read term_reviewer).
INSERT INTO term_reviewer (account_id) VALUES (@account_id) ON CONFLICT (account_id) DO NOTHING;

-- name: RevokeTermReviewer :exec
DELETE FROM term_reviewer WHERE account_id = @account_id;
