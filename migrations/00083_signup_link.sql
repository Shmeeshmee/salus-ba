-- +goose Up
-- BG4, U8 (docs/clinic-cms/WAVE6.md): the signup link and the verified signup's e-mail. The number is the next free one
-- after the second half's LC1, RM1, LB1 and TP1; the integration renumbers.
--   - token_issued_at: when the latest verification e-mail's token was issued. The link's 72 hours count from it, not
--     from the signup, so a delayed e-mail job never sends a link with less time. Existing rows count from their signup.
--   - email: a verified signup forgets the admin's e-mail once the admin accepted their invite (the account holds it
--     from then on); an unverified one keeps it, its link is still to be sent.
ALTER TABLE clinic_signup ADD COLUMN token_issued_at timestamptz;
UPDATE clinic_signup SET token_issued_at = created_at;
ALTER TABLE clinic_signup ALTER COLUMN token_issued_at SET DEFAULT now();
ALTER TABLE clinic_signup ALTER COLUMN token_issued_at SET NOT NULL;

ALTER TABLE clinic_signup ALTER COLUMN email DROP NOT NULL;
ALTER TABLE clinic_signup ADD CONSTRAINT clinic_signup_email_kept CHECK (email IS NOT NULL OR verified_at IS NOT NULL);
-- The clinics whose admin joined already (an accepted invite may have been removed by retention: the account stays).
UPDATE clinic_signup s SET email = NULL
WHERE s.verified_at IS NOT NULL
  AND EXISTS (SELECT 1 FROM account a WHERE a.organization_id = s.organization_id AND a.role = 'admin');

GRANT UPDATE (token_issued_at, email) ON clinic_signup TO salus_app;
