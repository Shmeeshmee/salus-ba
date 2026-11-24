-- +goose Up
-- Closing and erasing a patient's account (docs/clinic-cms/WAVE5.md S5, S6, package D3).
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing; the comparison with 'on' alike.

-- erased_at marks the account whose login, password, MFA and languages the erasure job blanked, so a second run
-- erases nothing more. The row stays: data requests, audit events and the clinics' records reference it. An account is
-- erased only once its erasure date has come, with nothing of the login left but a placeholder derived from its id.
ALTER TABLE account ADD COLUMN erased_at timestamptz;
ALTER TABLE account ADD CONSTRAINT account_erased CHECK (erased_at IS NULL OR erase_after IS NOT NULL AND erased_at >= erase_after);
ALTER TABLE account ADD CONSTRAINT account_erased_blank CHECK (erased_at IS NULL
                                                               OR email = id::text || '@erased.invalid' AND password_hash = ''
                                                                  AND totp_secret IS NULL AND totp_last_step IS NULL
                                                                  AND mfa_enrolled_at IS NULL AND language IS NULL
                                                                  AND detected_language IS NULL);

-- The erasure scope (app.erasure = 'on', no patient, account or clinic set): the closure files, and the erasure job
-- moves and answers, the account's own erasure request (S5, S6). Only a request for the account target, of kind
-- erasure, of a closed account, filed by that account. The guard (00071) still holds: forward only, final once
-- answered. Set only by datarequest's closure and erasure, whose every step is audited.
CREATE POLICY data_request_erasure ON data_request
    USING (current_setting('app.erasure', true) = 'on'
           AND NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.organization_id', true), '') IS NULL
           AND organization_id IS NULL AND kind = 'erasure'
           AND EXISTS (SELECT 1 FROM account a WHERE a.id = data_request.account_id AND a.closed_at IS NOT NULL))
    WITH CHECK (current_setting('app.erasure', true) = 'on'
                AND NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND NULLIF(current_setting('app.organization_id', true), '') IS NULL
                AND organization_id IS NULL AND kind = 'erasure' AND filed_by_account_id = account_id
                AND EXISTS (SELECT 1 FROM account a WHERE a.id = data_request.account_id AND a.closed_at IS NOT NULL));
-- Its history: filed by the patient (the closure), moved and answered by the job (null).
CREATE POLICY data_request_change_erasure ON data_request_change FOR INSERT
    WITH CHECK (current_setting('app.erasure', true) = 'on'
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND EXISTS (SELECT 1 FROM data_request r
                            WHERE r.id = data_request_change.data_request_id AND r.status = data_request_change.status
                              AND (data_request_change.changed_by_account_id IS NULL
                                   OR data_request_change.changed_by_account_id = r.account_id)));

-- Whether the profile belongs to an account whose erasure date has come: SECURITY DEFINER, as the clinic's scope does
-- not see a profile whose consents the closure revoked. Opaque ids in, a boolean out.
-- +goose StatementBegin
CREATE FUNCTION patient_erasure_due(patient uuid) RETURNS boolean
    LANGUAGE sql STABLE SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
    SELECT EXISTS (SELECT 1 FROM patient p JOIN account a ON a.id = p.account_id
                   WHERE p.id = patient_erasure_due.patient AND a.erase_after <= now())
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION patient_erasure_due(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION patient_erasure_due(uuid) TO salus_app;

-- The erasure's restriction of the record a clinic keeps (S6): in that clinic's scope with app.erasure on, by the job
-- (no staff, no request of the clinic), for the profile of an account whose erasure date has come. The clinic never
-- lifts it (00071's restrictive policy).
CREATE POLICY patient_restriction_erasure ON patient_restriction FOR INSERT
    WITH CHECK (current_setting('app.erasure', true) = 'on'
                AND NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND restricted_by_account_id IS NULL AND data_request_id IS NULL
                AND patient_erasure_due(patient_restriction.patient_id));
