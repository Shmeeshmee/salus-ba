-- +goose Up
-- The clinic system's wave 5 contract (docs/clinic-cms/WAVE5.md "5.0: the contract"): the data subject's requests
-- (GDPR Art. 12 to 21), account closure and erasure, the restricted record and the analytics objection. No behaviour:
-- the Go services refuse with errors.ErrUnsupported until their package (D1 to D6) fills them.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

-- A data subject's request (S1, S2, S3). Its target is the patient's Salus account (organization_id null: Salus
-- answers, through the operator, who reads and writes these rows as the table's owner from salusctl) or the patient's
-- record at one clinic (organization_id: the clinic's admin answers). account_id is the patient's own account when they
-- filed it in the app; it is null for a request the clinic filed at the desk for a patient it registered (S10), whose
-- profile is patient_id. A request is never deleted, and its answer is a code, never free text about health.
CREATE TABLE data_request (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    received_at timestamptz NOT NULL,
    due_at timestamptz NOT NULL, -- one month from receipt, in the answerer's time zone (S3)
    extended_until timestamptz,  -- set once, at most two months after due_at, with its reason
    answered_at timestamptz,
    account_id uuid REFERENCES account (id),
    patient_id uuid REFERENCES patient (id),
    organization_id uuid REFERENCES organization (id),
    filed_by_account_id uuid NOT NULL REFERENCES account (id),
    version integer DEFAULT 1 NOT NULL CHECK (version > 0),
    kind text NOT NULL CHECK (kind IN ('access', 'portability', 'rectification', 'restriction', 'objection', 'erasure')),
    status text DEFAULT 'received' NOT NULL
        CHECK (status IN ('received', 'in_progress', 'fulfilled', 'refused', 'withdrawn')),
    -- Told to the patient (Art. 12(3)): the request is complex, or one of many.
    extension_reason text CHECK (extension_reason IN ('complex', 'many_requests')),
    -- done: as asked; done_partly_kept: done, with what the law keeps kept (an erasure under the medical-record
    -- retention); the refusals: identity not verified, manifestly unfounded or excessive (Art. 12(5)), retention law
    -- (Art. 17(3)), no data of the patient.
    answer_code text CHECK (answer_code IN ('done', 'done_partly_kept', 'identity_unverified', 'manifestly_unfounded',
                                            'excessive', 'retention_law', 'no_data')),
    CONSTRAINT data_request_subject CHECK (organization_id IS NULL AND account_id IS NOT NULL
                                           OR organization_id IS NOT NULL AND patient_id IS NOT NULL),
    CONSTRAINT data_request_own_filing CHECK (account_id IS NULL OR filed_by_account_id = account_id),
    CONSTRAINT data_request_due CHECK (due_at > received_at),
    CONSTRAINT data_request_extension CHECK ((extended_until IS NULL) = (extension_reason IS NULL)
                                             AND (extended_until IS NULL OR extended_until > due_at
                                                  AND extended_until <= due_at + interval '2 months')),
    CONSTRAINT data_request_answer CHECK ((status IN ('fulfilled', 'refused')) = (answered_at IS NOT NULL AND answer_code IS NOT NULL)
                                          AND (answer_code IS NULL
                                               OR (status = 'fulfilled') = (answer_code IN ('done', 'done_partly_kept'))))
);
CREATE INDEX data_request_account ON data_request (account_id, received_at) WHERE account_id IS NOT NULL;
CREATE INDEX data_request_clinic ON data_request (organization_id, status, due_at) WHERE organization_id IS NOT NULL;
CREATE INDEX data_request_patient ON data_request (patient_id) WHERE patient_id IS NOT NULL;
CREATE INDEX data_request_open_due ON data_request (due_at) WHERE status IN ('received', 'in_progress');

ALTER TABLE data_request ENABLE ROW LEVEL SECURITY;
-- The patient's scope (their account): their own requests, filed by themselves, naming their own profile. That a
-- clinic target is one where the patient has a record is the service's rule (D1).
CREATE POLICY data_request_own ON data_request
    USING (account_id = NULLIF(current_setting('app.account_id', true), '')::uuid)
    WITH CHECK (account_id = NULLIF(current_setting('app.account_id', true), '')::uuid
                AND (patient_id IS NULL OR EXISTS (SELECT 1 FROM patient p
                                                   WHERE p.id = data_request.patient_id
                                                     AND p.account_id = data_request.account_id)));
-- The clinic's scope: the requests targeting it. It files only for a patient it registered (S10), by its own staff.
CREATE POLICY data_request_clinic ON data_request FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
CREATE POLICY data_request_clinic_update ON data_request FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- Never in a patient's name: a clinic's filing has no account and names its own staff.
CREATE POLICY data_request_clinic_insert ON data_request FOR INSERT
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND account_id IS NULL
                AND EXISTS (SELECT 1 FROM patient p
                            WHERE p.id = data_request.patient_id
                              AND p.registered_by_organization_id = data_request.organization_id)
                AND EXISTS (SELECT 1 FROM account a
                            WHERE a.id = data_request.filed_by_account_id
                              AND a.organization_id = data_request.organization_id));
GRANT SELECT, INSERT ON data_request TO salus_app;
GRANT UPDATE (status, extended_until, extension_reason, answered_at, answer_code, version) ON data_request TO salus_app;

-- A request moves forward only: received, then in progress, then fulfilled, refused or withdrawn, and an answered or
-- withdrawn request never changes again. It is extended once. Who it is for, its target, kind, filing and due date never
-- change. The patient's own scope may only withdraw it. For every role, the owner (the operator) included; no role
-- deletes a request.
-- +goose StatementBegin
CREATE FUNCTION data_request_guard() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog, public
AS $$
DECLARE
    owner boolean := session_user = (SELECT r.rolname FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner WHERE c.oid = TG_RELID);
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'received' OR NEW.answered_at IS NOT NULL OR NEW.answer_code IS NOT NULL
           OR NEW.extended_until IS NOT NULL OR NEW.version <> 1 THEN
            RAISE EXCEPTION 'data_request: a request is filed received, unanswered and not extended' USING ERRCODE = 'check_violation';
        END IF;
        IF NOT owner AND NEW.received_at NOT BETWEEN now() - interval '5 minutes' AND now() + interval '5 minutes' THEN
            RAISE EXCEPTION 'data_request: a request is received when it is filed' USING ERRCODE = 'check_violation';
        END IF;
        RETURN NEW;
    END IF;
    IF TG_OP <> 'UPDATE' THEN
        RAISE EXCEPTION 'data_request: a request is never deleted' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF NEW.account_id IS DISTINCT FROM OLD.account_id OR NEW.patient_id IS DISTINCT FROM OLD.patient_id
       OR NEW.organization_id IS DISTINCT FROM OLD.organization_id OR NEW.kind <> OLD.kind
       OR NEW.filed_by_account_id <> OLD.filed_by_account_id OR NEW.received_at <> OLD.received_at
       OR NEW.due_at <> OLD.due_at OR NEW.id <> OLD.id THEN
        RAISE EXCEPTION 'data_request: only its status, extension and answer change' USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.status IN ('fulfilled', 'refused', 'withdrawn') THEN
        RAISE EXCEPTION 'data_request: an answered or withdrawn request never changes' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.status <> OLD.status AND NOT (OLD.status = 'received' AND NEW.status <> 'received'
                                         OR OLD.status = 'in_progress' AND NEW.status <> 'received') THEN
        RAISE EXCEPTION 'data_request: a request moves forward only' USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.extended_until IS NOT NULL AND (NEW.extended_until IS DISTINCT FROM OLD.extended_until
                                           OR NEW.extension_reason IS DISTINCT FROM OLD.extension_reason) THEN
        RAISE EXCEPTION 'data_request: a request is extended once' USING ERRCODE = 'check_violation';
    END IF;
    IF NULLIF(current_setting('app.account_id', true), '') IS NOT NULL
       AND (NEW.status <> 'withdrawn' OR NEW.extended_until IS DISTINCT FROM OLD.extended_until
            OR NEW.answered_at IS DISTINCT FROM OLD.answered_at OR NEW.answer_code IS DISTINCT FROM OLD.answer_code) THEN
        RAISE EXCEPTION 'data_request: the patient only withdraws a request' USING ERRCODE = 'insufficient_privilege';
    END IF;
    -- A request the patient filed is withdrawn by the patient only; one the clinic filed for them (S10) by the clinic.
    IF NEW.status = 'withdrawn' AND NOT owner AND OLD.account_id IS NOT NULL
       AND OLD.account_id IS DISTINCT FROM NULLIF(current_setting('app.account_id', true), '')::uuid THEN
        RAISE EXCEPTION 'data_request: only the patient withdraws their request' USING ERRCODE = 'insufficient_privilege';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER data_request_guard BEFORE INSERT OR UPDATE OR DELETE ON data_request
    FOR EACH ROW EXECUTE FUNCTION data_request_guard();
CREATE TRIGGER data_request_no_truncate BEFORE TRUNCATE ON data_request
    FOR EACH STATEMENT EXECUTE FUNCTION data_request_guard();

-- The append-only history of a request's status changes: who changed it (an account, or null for a job or the
-- operator) and when. Seen and written where its request is seen, with the request's current status; in the patient's
-- scope by the patient, in the clinic's by its own staff or a job (null).
CREATE TABLE data_request_change (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    changed_at timestamptz NOT NULL,
    data_request_id uuid NOT NULL REFERENCES data_request (id),
    changed_by_account_id uuid REFERENCES account (id),
    status text NOT NULL CHECK (status IN ('received', 'in_progress', 'fulfilled', 'refused', 'withdrawn'))
);
CREATE INDEX data_request_change_request ON data_request_change (data_request_id, changed_at);

ALTER TABLE data_request_change ENABLE ROW LEVEL SECURITY;
CREATE POLICY data_request_change_visible ON data_request_change
    USING (EXISTS (SELECT 1 FROM data_request r WHERE r.id = data_request_change.data_request_id))
    WITH CHECK (EXISTS (SELECT 1 FROM data_request r
                        WHERE r.id = data_request_change.data_request_id AND r.status = data_request_change.status)
                AND (changed_by_account_id = NULLIF(current_setting('app.account_id', true), '')::uuid
                     OR NULLIF(current_setting('app.account_id', true), '') IS NULL
                        AND (changed_by_account_id IS NULL
                             OR EXISTS (SELECT 1 FROM account a
                                        WHERE a.id = data_request_change.changed_by_account_id
                                          AND a.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid))));
GRANT SELECT, INSERT ON data_request_change TO salus_app;

-- +goose StatementBegin
CREATE FUNCTION data_request_change_reject() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
BEGIN
    RAISE EXCEPTION 'data_request_change is append-only: % is not allowed', TG_OP;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER data_request_change_no_update_delete BEFORE UPDATE OR DELETE ON data_request_change
    FOR EACH ROW EXECUTE FUNCTION data_request_change_reject();
CREATE TRIGGER data_request_change_no_truncate BEFORE TRUNCATE ON data_request_change
    FOR EACH STATEMENT EXECUTE FUNCTION data_request_change_reject();

-- Closing the account (S5) and its erasure 30 days later (S6): both set at closure, both cleared when the operator
-- reopens it within the 30 days. salus_app's UPDATE on account covers them.
ALTER TABLE account ADD COLUMN closed_at timestamptz;
ALTER TABLE account ADD COLUMN erase_after timestamptz;
ALTER TABLE account ADD CONSTRAINT account_closed CHECK ((closed_at IS NULL) = (erase_after IS NULL)
                                                         AND (erase_after IS NULL OR erase_after > closed_at));
CREATE INDEX account_erase_after ON account (erase_after) WHERE erase_after IS NOT NULL;

-- The patient's objection to the use of their data in the anonymised analytics (S9): from then on their events are
-- left out of new aggregates. Null for none.
ALTER TABLE patient ADD COLUMN analytics_objected_at timestamptz;
GRANT UPDATE (analytics_objected_at) ON patient TO salus_app;

-- A patient's record restricted at one clinic (S6, S7): hidden from the clinic's everyday screens and searches, opened
-- by its admin only with a reason. Set by the clinic's staff (restricted_by_account_id) or by the erasure job (null),
-- optionally for a request targeting that clinic; lifting a staff restriction deletes the row, audited. The erasure's
-- restriction is never lifted by the clinic (S6); how the job writes it (as salus_retention, which has no INSERT
-- today, or in a scope) is D3's, in its own migration.
CREATE TABLE patient_restriction (
    restricted_at timestamptz NOT NULL,
    patient_id uuid NOT NULL REFERENCES patient (id),
    organization_id uuid NOT NULL REFERENCES organization (id),
    restricted_by_account_id uuid REFERENCES account (id),
    data_request_id uuid REFERENCES data_request (id),
    PRIMARY KEY (patient_id, organization_id)
);
CREATE INDEX patient_restriction_clinic ON patient_restriction (organization_id);

ALTER TABLE patient_restriction ENABLE ROW LEVEL SECURITY;
-- The clinic's scope only, by its own staff.
CREATE POLICY patient_restriction_clinic ON patient_restriction
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM account a
                            WHERE a.id = patient_restriction.restricted_by_account_id
                              AND a.organization_id = patient_restriction.organization_id)
                AND (data_request_id IS NULL
                     OR EXISTS (SELECT 1 FROM data_request r
                                WHERE r.id = patient_restriction.data_request_id
                                  AND r.organization_id = patient_restriction.organization_id
                                  AND r.patient_id = patient_restriction.patient_id)));
-- Only a staff restriction is lifted by the clinic.
CREATE POLICY patient_restriction_lift ON patient_restriction AS RESTRICTIVE FOR DELETE
    USING (restricted_by_account_id IS NOT NULL);
GRANT SELECT, INSERT, DELETE ON patient_restriction TO salus_app;
