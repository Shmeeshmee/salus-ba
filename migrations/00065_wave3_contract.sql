-- +goose Up
-- The clinic system's wave 3 contract (docs/clinic-cms/IMPLEMENTATION_PLAN.md "3.0: the contract"): every table,
-- column and value the reception, desk, signup, quote and clinic profile packages build on. No behaviour: the Go
-- services refuse with errors.ErrUnsupported until their package (F1, F2, O1, O2, X1, X2) fills them.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

-- F1: the reception role (owner, 2026-11-23). A receptionist is staff of one clinic and, like an admin, has no
-- Practitioner and no name on the invite.
ALTER TABLE account DROP CONSTRAINT account_role_check;
ALTER TABLE account ADD CONSTRAINT account_role_check CHECK (role IN ('admin', 'practitioner', 'reception', 'patient'));
ALTER TABLE invite DROP CONSTRAINT invite_role_check;
ALTER TABLE invite ADD CONSTRAINT invite_role_check CHECK (role IN ('admin', 'practitioner', 'reception'));
ALTER TABLE invite DROP CONSTRAINT invite_name_practitioner_only;
ALTER TABLE invite ADD CONSTRAINT invite_name_practitioner_only CHECK (
    (role = 'practitioner' AND btrim(given_name) <> '' AND btrim(family_name) <> '')
    OR (role IN ('admin', 'reception') AND given_name = '' AND family_name = ''));

-- F2: a patient the clinic registers (W1). A patient's own profile is unchanged: it has an account and no
-- registering clinic. A clinic-registered profile has no account and is the registering clinic's only: no other
-- clinic sees it, whatever consent or appointment it might have (the restrictive policy). Linking it to an account
-- the patient opens later is a later package, which relaxes patient_owner.
ALTER TABLE patient ALTER COLUMN account_id DROP NOT NULL;
ALTER TABLE patient ADD COLUMN registered_by_organization_id uuid REFERENCES organization (id);
ALTER TABLE patient ADD CONSTRAINT patient_owner CHECK ((account_id IS NULL) <> (registered_by_organization_id IS NULL));
CREATE INDEX patient_registered_by ON patient (registered_by_organization_id, family_name, given_name)
    WHERE registered_by_organization_id IS NOT NULL;

-- The clinic scope (no patient and no account set) reads, registers and corrects the profiles it registered.
CREATE POLICY patient_registered ON patient
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND registered_by_organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND account_id IS NULL
                AND registered_by_organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
CREATE POLICY patient_registered_only_there ON patient AS RESTRICTIVE
    USING (registered_by_organization_id IS NULL
           OR registered_by_organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- F2: the care consent of a clinic-registered patient, given in person and recorded by staff. A consent given in
-- the app has neither column; one recorded by staff has both.
ALTER TABLE consent ADD COLUMN recorded_by_account_id uuid REFERENCES account (id);
ALTER TABLE consent ADD COLUMN given_in_person boolean DEFAULT false NOT NULL;
ALTER TABLE consent ADD CONSTRAINT consent_in_person CHECK (given_in_person = (recorded_by_account_id IS NOT NULL));
-- The clinic records it only to itself and only as given in person. Revocation by the clinic is F2's (a policy for
-- UPDATE with its rule).
CREATE POLICY consent_clinic_in_person ON consent FOR INSERT
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND given_in_person);
-- And only for a patient that clinic registered, recorded by an account of that clinic, never in a patient's scope,
-- for every role, so no clinic grants itself a patient who has an account and nobody names another clinic's staff. A trigger, not the policy: a policy on consent reading patient, whose policies read
-- consent, is refused as recursive.
-- +goose StatementBegin
CREATE FUNCTION consent_guard_in_person() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog, public
AS $$
BEGIN
    IF NOT NEW.given_in_person THEN
        RETURN NEW;
    END IF;
    IF NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL OR NOT EXISTS (
        SELECT 1 FROM patient p WHERE p.id = NEW.patient_id AND p.registered_by_organization_id = NEW.organization_id) THEN
        RAISE EXCEPTION 'consent: given in person only for a patient the clinic registered' USING ERRCODE = 'check_violation';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM account a WHERE a.id = NEW.recorded_by_account_id AND a.organization_id = NEW.organization_id) THEN
        RAISE EXCEPTION 'consent: recorded by the clinic''s own staff only' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER consent_guard_in_person BEFORE INSERT ON consent
    FOR EACH ROW EXECUTE FUNCTION consent_guard_in_person();

-- F2, F3: arrival at the desk (W6) and bookings by staff. arrived is between booked and fulfilled, for an in-person
-- visit only; arrived_at stays when the visit is then fulfilled or marked a no-show.
ALTER TABLE appointment DROP CONSTRAINT appointment_status_check;
ALTER TABLE appointment ADD CONSTRAINT appointment_status_check
    CHECK (status IN ('booked', 'arrived', 'cancelled', 'fulfilled', 'noshow'));
ALTER TABLE appointment ADD COLUMN arrived_at timestamptz;
ALTER TABLE appointment ADD COLUMN booked_by_account_id uuid REFERENCES account (id); -- staff; null for the patient's own
ALTER TABLE appointment ADD CONSTRAINT appointment_arrived CHECK ((status <> 'arrived' OR arrived_at IS NOT NULL)
                                                                  AND (arrived_at IS NULL OR visit_mode = 'in-person')
                                                                  AND (arrived_at IS NULL OR status <> 'booked'));
GRANT UPDATE (arrived_at) ON appointment TO salus_app;
-- Staff book in their clinic's scope, for a patient with the clinic's active care consent, and say who booked.
CREATE POLICY appointment_clinic_insert ON appointment FOR INSERT
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND booked_by_account_id IS NOT NULL
                AND EXISTS (SELECT 1 FROM consent c
                            WHERE c.patient_id = appointment.patient_id
                              AND c.organization_id = appointment.organization_id
                              AND c.status = 'active'));

-- Who booked, and arrival, are the clinic's: a staff booking names an account of its own clinic and is made in the
-- clinic's scope; a patient's own booking names nobody. Arrival is set once, in the clinic's scope, from booked, and
-- never changes after. For every role but the table's owner (as appointment_guard_change).
-- +goose StatementBegin
CREATE FUNCTION appointment_guard_staff() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog, public
AS $$
DECLARE
    clinic boolean := NULLIF(current_setting('app.patient_id', true), '') IS NULL
                      AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                      AND NULLIF(current_setting('app.organization_id', true), '')::uuid = NEW.organization_id;
BEGIN
    IF session_user = (SELECT r.rolname FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner WHERE c.oid = TG_RELID) THEN
        RETURN NEW;
    END IF;
    IF TG_OP = 'INSERT' THEN
        IF NEW.booked_by_account_id IS NOT NULL AND NOT (clinic AND EXISTS (
            SELECT 1 FROM account a WHERE a.id = NEW.booked_by_account_id AND a.organization_id = NEW.organization_id)) THEN
            RAISE EXCEPTION 'appointment: booked by the clinic''s own staff, in its scope' USING ERRCODE = 'insufficient_privilege';
        END IF;
        IF NEW.arrived_at IS NOT NULL OR NEW.status = 'arrived' THEN
            RAISE EXCEPTION 'appointment: a booking has not arrived' USING ERRCODE = 'check_violation';
        END IF;
        RETURN NEW;
    END IF;
    IF NEW.booked_by_account_id IS DISTINCT FROM OLD.booked_by_account_id THEN
        RAISE EXCEPTION 'appointment: who booked never changes' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.arrived_at IS DISTINCT FROM OLD.arrived_at OR (NEW.status = 'arrived' AND OLD.status <> 'arrived') THEN
        IF NOT clinic THEN
            RAISE EXCEPTION 'appointment: only the clinic marks an arrival' USING ERRCODE = 'insufficient_privilege';
        END IF;
        IF OLD.arrived_at IS NOT NULL OR OLD.status <> 'booked' OR NEW.status <> 'arrived' THEN
            RAISE EXCEPTION 'appointment: an arrival is set once, from booked' USING ERRCODE = 'check_violation';
        END IF;
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER appointment_guard_staff BEFORE INSERT OR UPDATE ON appointment
    FOR EACH ROW EXECUTE FUNCTION appointment_guard_staff();

-- Staff move a booked visit (F2) in their clinic's scope, as its patient does in theirs: 00054's guard said the patient
-- only. The rest of the guard is unchanged.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION appointment_guard_change() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
DECLARE
    own boolean := NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL
                   AND NULLIF(current_setting('app.patient_id', true), '')::uuid = OLD.patient_id;
    clinic boolean := NULLIF(current_setting('app.patient_id', true), '') IS NULL
                      AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                      AND NULLIF(current_setting('app.organization_id', true), '')::uuid = OLD.organization_id;
    moved boolean := NEW.period IS DISTINCT FROM OLD.period OR NEW.slot_id IS DISTINCT FROM OLD.slot_id;
BEGIN
    -- session_user, not current_user: a SECURITY DEFINER function owned by the owner runs as it, yet stays guarded.
    IF session_user = (SELECT r.rolname FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner WHERE c.oid = TG_RELID) THEN
        RETURN NEW;
    END IF;
    IF moved AND NOT ((own OR clinic) AND OLD.status = 'booked' AND NEW.status = 'booked') THEN
        RAISE EXCEPTION 'appointment: only its patient or its clinic moves a booked appointment' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF (NEW.period IS DISTINCT FROM OLD.period) <> (NEW.slot_id IS DISTINCT FROM OLD.slot_id) THEN
        RAISE EXCEPTION 'appointment: a reschedule changes the period and the slot together' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.period IS DISTINCT FROM OLD.period AND NEW.reschedule_count IS DISTINCT FROM OLD.reschedule_count + 1 THEN
        RAISE EXCEPTION 'appointment: a reschedule counts once' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.period IS NOT DISTINCT FROM OLD.period AND NEW.reschedule_count IS DISTINCT FROM OLD.reschedule_count THEN
        RAISE EXCEPTION 'appointment: the reschedule count changes only with the period' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.confirmed_at IS DISTINCT FROM OLD.confirmed_at OR NEW.confirmed_via IS DISTINCT FROM OLD.confirmed_via THEN
        IF NOT own THEN
            RAISE EXCEPTION 'appointment: only its patient confirms' USING ERRCODE = 'insufficient_privilege';
        END IF;
        IF OLD.confirmed_at IS NOT NULL THEN
            RAISE EXCEPTION 'appointment: a confirmation never changes' USING ERRCODE = 'check_violation';
        END IF;
        IF OLD.status <> 'booked' THEN
            RAISE EXCEPTION 'appointment: only a booked appointment is confirmed' USING ERRCODE = 'check_violation';
        END IF;
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

-- The change log knows arrived: set from booked; from arrived the visit is fulfilled, a no-show or cancelled.
ALTER TABLE appointment_change DROP CONSTRAINT appointment_change_from_status_check;
ALTER TABLE appointment_change ADD CONSTRAINT appointment_change_from_status_check
    CHECK (from_status IN ('booked', 'arrived', 'cancelled', 'fulfilled', 'noshow'));
ALTER TABLE appointment_change DROP CONSTRAINT appointment_change_to_status_check;
ALTER TABLE appointment_change ADD CONSTRAINT appointment_change_to_status_check
    CHECK (to_status IN ('booked', 'arrived', 'cancelled', 'fulfilled', 'noshow'));
ALTER TABLE appointment_change DROP CONSTRAINT appointment_change_status_set;
ALTER TABLE appointment_change ADD CONSTRAINT appointment_change_status_set CHECK (kind <> 'status_set'
    OR (from_status = 'booked' AND to_status IN ('arrived', 'fulfilled', 'noshow', 'cancelled'))
    OR (from_status = 'arrived' AND to_status IN ('fulfilled', 'noshow', 'cancelled')));

-- O1: a clinic's self-signup (W3). Before verification there is no clinic, so the table is not clinic-scoped (as
-- account): the signup service finds a row by its token's hash and checks the rest. The token is never stored, only
-- its SHA-256. approved_at is set by the operator (salusctl, as the owner): the app has no grant on it.
CREATE TABLE clinic_signup (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    verified_at timestamptz,
    approved_at timestamptz,
    organization_id uuid REFERENCES organization (id), -- created on verification, unlisted
    -- The admin's e-mail: trimmed and lower-cased like account.email.
    email text NOT NULL CHECK (email = lower(btrim(email))),
    clinic_name text NOT NULL CHECK (btrim(clinic_name) <> ''),
    language language NOT NULL, -- of the verification e-mail and the clinic's default
    token_hash bytea NOT NULL CHECK (octet_length(token_hash) = 32),
    CONSTRAINT clinic_signup_verified CHECK ((verified_at IS NULL) = (organization_id IS NULL)),
    CONSTRAINT clinic_signup_approved CHECK (approved_at IS NULL OR verified_at IS NOT NULL)
);
CREATE UNIQUE INDEX clinic_signup_token_hash_key ON clinic_signup (token_hash);
CREATE UNIQUE INDEX clinic_signup_organization_key ON clinic_signup (organization_id);
CREATE INDEX clinic_signup_email ON clinic_signup (email);
-- No DELETE: the retention job removes unverified signups as the owner.
GRANT SELECT, INSERT ON clinic_signup TO salus_app;
GRANT UPDATE (verified_at, organization_id, token_hash) ON clinic_signup TO salus_app;

-- X1: treatment quotes (W4). Not a receipt: never numbered, never reported, never charged, so it has its own tables
-- rather than a kind of invoice. Sent once to a patient of the clinic; a sent quote never changes (no UPDATE grant).
CREATE TABLE quote (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    patient_id uuid NOT NULL REFERENCES patient (id),
    appointment_id uuid, -- the visit it was written at; null for none
    created_by_account_id uuid NOT NULL REFERENCES account (id),
    issued_at timestamptz NOT NULL,
    valid_until timestamptz NOT NULL, -- 30 days after issue by default (X1)
    total_minor bigint NOT NULL CHECK (total_minor >= 0),
    currency text DEFAULT 'BAM' NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    note text DEFAULT '' NOT NULL,
    CONSTRAINT quote_valid CHECK (valid_until > issued_at),
    CONSTRAINT quote_appointment_fkey FOREIGN KEY (appointment_id, organization_id, patient_id)
        REFERENCES appointment (id, organization_id, patient_id)
);
CREATE INDEX quote_organization_issued ON quote (organization_id, issued_at);
CREATE INDEX quote_patient ON quote (patient_id, issued_at);

CREATE TABLE quote_line (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    quote_id uuid NOT NULL REFERENCES quote (id),
    position integer NOT NULL CHECK (position > 0),
    charge_item_definition_id uuid REFERENCES charge_item_definition (id),
    name text NOT NULL CHECK (name <> ''), -- copied from the price list when written
    quantity integer NOT NULL CHECK (quantity > 0),
    unit_price_minor bigint NOT NULL CHECK (unit_price_minor >= 0),
    discount_minor bigint DEFAULT 0 NOT NULL CHECK (discount_minor >= 0),
    vat_rate_bp integer CHECK (vat_rate_bp BETWEEN 0 AND 10000),
    total_minor bigint NOT NULL CHECK (total_minor >= 0),
    CONSTRAINT quote_line_position_key UNIQUE (quote_id, position)
);

ALTER TABLE quote ENABLE ROW LEVEL SECURITY;
CREATE POLICY quote_own ON quote FOR SELECT
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
-- The clinic's scope: its quotes, written only for a patient with its active care consent, by its own staff.
CREATE POLICY quote_clinic ON quote
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM consent c
                            WHERE c.patient_id = quote.patient_id
                              AND c.organization_id = quote.organization_id
                              AND c.status = 'active')
                AND EXISTS (SELECT 1 FROM account a
                            WHERE a.id = quote.created_by_account_id AND a.organization_id = quote.organization_id));
ALTER TABLE quote_line ENABLE ROW LEVEL SECURITY;
CREATE POLICY quote_line_seen ON quote_line FOR SELECT
    USING (EXISTS (SELECT 1 FROM quote q WHERE q.id = quote_line.quote_id));
-- Lines are written in the clinic's scope of their quote.
CREATE POLICY quote_line_clinic ON quote_line FOR INSERT
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND EXISTS (SELECT 1 FROM quote q
                            WHERE q.id = quote_line.quote_id
                              AND q.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid));
-- And only in the quote's own transaction, from its clinic's price list, for every role (as invoice_line): a sent
-- quote never gains a line, and a foreign key alone would take another clinic's service.
-- +goose StatementBegin
CREATE FUNCTION quote_line_guard_insert() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog, public
AS $$
DECLARE
    org uuid;
    created timestamptz;
BEGIN
    SELECT q.organization_id, q.created_at INTO org, created FROM quote q WHERE q.id = NEW.quote_id;
    IF created IS DISTINCT FROM now() THEN
        RAISE EXCEPTION 'quote_line: lines are added only with their quote' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.charge_item_definition_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM charge_item_definition d WHERE d.id = NEW.charge_item_definition_id AND d.organization_id = org) THEN
        RAISE EXCEPTION 'quote_line: a service of another clinic' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER quote_line_guard_insert BEFORE INSERT ON quote_line
    FOR EACH ROW EXECUTE FUNCTION quote_line_guard_insert();
GRANT SELECT, INSERT ON quote, quote_line TO salus_app;

-- X2: the clinic's profile for patients abroad. Not personal data. languages are the language enum's tags as text,
-- as every enumerated array of the schema (a language[] would need its array type registered in pgx and sqlc).
ALTER TABLE organization ADD COLUMN languages text[] DEFAULT '{}' NOT NULL
    CHECK (languages <@ enum_range(NULL::language)::text[]);
ALTER TABLE organization ADD COLUMN profile_description text DEFAULT '' NOT NULL; -- in the clinic's default language
ALTER TABLE organization ADD COLUMN welcomes_international boolean DEFAULT false NOT NULL;
GRANT UPDATE (languages, profile_description, welcomes_international) ON organization TO salus_app;

CREATE TABLE organization_profile_translation (
    organization_id uuid NOT NULL REFERENCES organization (id),
    language language NOT NULL,
    description text NOT NULL CHECK (description <> ''),
    PRIMARY KEY (organization_id, language)
);
ALTER TABLE organization_profile_translation ENABLE ROW LEVEL SECURITY;
-- Seen where its clinic is seen (organization's own policies: the clinic, the directory); written by the clinic.
CREATE POLICY organization_profile_translation_read ON organization_profile_translation FOR SELECT
    USING (EXISTS (SELECT 1 FROM organization o WHERE o.id = organization_profile_translation.organization_id));
CREATE POLICY organization_profile_translation_clinic ON organization_profile_translation
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
GRANT SELECT, INSERT, DELETE ON organization_profile_translation TO salus_app;
GRANT UPDATE (description) ON organization_profile_translation TO salus_app;
