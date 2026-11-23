-- name: SignupByTokenHash :one
SELECT id, created_at, verified_at, approved_at, organization_id, email, clinic_name, language, token_hash
FROM clinic_signup
WHERE token_hash = @token_hash;
