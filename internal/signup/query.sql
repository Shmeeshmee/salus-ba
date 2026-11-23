-- name: InsertSignup :one
INSERT INTO clinic_signup (email, clinic_name, language, token_hash)
VALUES (@email, @clinic_name, @language, @token_hash)
RETURNING id, created_at;

-- name: SignupByTokenHash :one
-- FOR UPDATE: two verifications of one token wait for each other; the second sees it verified.
SELECT id, created_at, verified_at, approved_at, organization_id, email, clinic_name, language, token_hash
FROM clinic_signup
WHERE token_hash = @token_hash
FOR UPDATE;

-- name: SignupByID :one
SELECT id, created_at, verified_at, approved_at, organization_id, email, clinic_name, language, token_hash
FROM clinic_signup
WHERE id = @id;

-- name: SetSignupToken :execrows
-- A new verification link replaces the last one, only while the signup is not verified.
UPDATE clinic_signup SET token_hash = @token_hash
WHERE id = @id AND verified_at IS NULL;

-- name: VerifySignup :execrows
UPDATE clinic_signup SET verified_at = @verified_at, organization_id = @organization_id
WHERE id = @id AND verified_at IS NULL;

-- name: ListSignups :many
-- salusctl signup list (as the owner): the verified signups, the ones waiting for approval first.
SELECT id, created_at, verified_at, approved_at, organization_id, email, clinic_name, language, token_hash
FROM clinic_signup
WHERE verified_at IS NOT NULL
ORDER BY approved_at IS NOT NULL, verified_at, id;

-- name: ApproveSignup :one
-- salusctl signup approve (as the owner): once, and only a verified signup.
UPDATE clinic_signup SET approved_at = @approved_at
WHERE organization_id = @organization_id AND verified_at IS NOT NULL AND approved_at IS NULL
RETURNING id;

-- name: SetupSteps :many
-- The wizard's progress of the clinic in scope; a step without a row is open.
SELECT step, state FROM clinic_setup_step WHERE organization_id = @organization_id;

-- name: PutSetupStep :exec
INSERT INTO clinic_setup_step (organization_id, step, state, updated_at)
VALUES (@organization_id, @step, @state, @updated_at)
ON CONFLICT (organization_id, step) DO UPDATE SET state = EXCLUDED.state, updated_at = EXCLUDED.updated_at;

-- name: DeleteSetupStep :exec
DELETE FROM clinic_setup_step WHERE organization_id = @organization_id AND step = @step;

-- name: DeleteUnverifiedSignupsBefore :execrows
-- The retention job (salus_retention): unverified signups made before @before. A verified one stays with its clinic.
DELETE FROM clinic_signup WHERE verified_at IS NULL AND created_at < @before;
