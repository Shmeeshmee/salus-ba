-- +goose Up
-- docs/clinic-cms R2 (V2): a clinic records that a patient it registered withdrew the care consent they gave it in
-- person. Its scope (no patient and no account set) may revoke only such a consent, of its own clinic, and only into
-- the revoked state; the column grant of 00016 already limits the update to status, revoked_at, updated_at and
-- version_id. A consent given in the app stays the patient's to revoke (consent_own).
CREATE POLICY consent_clinic_withdraw_in_person ON consent FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND given_in_person
           AND status = 'active')
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND given_in_person
                AND status = 'inactive');
