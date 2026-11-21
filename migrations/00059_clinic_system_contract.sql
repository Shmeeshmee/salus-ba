-- +goose Up
-- The clinic system's shared contract (docs/plans/clinic-cms.md "Contract", step 0; ADR 0016). The number is a
-- placeholder: a branch merging first takes it, and this file then takes the next free number.
--   - The visit mode, in-person or video: a key of each schedule rule (schedule.availability), a column of slot and of
--     appointment. Everything that exists is video. An appointment's mode is copied from its slot at booking and never
--     changes.
--   - organization: the clinic's address (in-person visits, receipts) and billing details (Salus's invoices to the
--     clinic). Not personal data.
--   - charge_item_definition (and its translations): the clinic's price list, package B. Not personal data.
--   - invoice, invoice_line, invoice_sequence: the receipts and invoices a clinic issues, package C. A receipt to a
--     patient is health data (Art. 9). Append-only once issued except the fiscal columns.
--   - subscription_plan, clinic_subscription, subscription_invoice: what a clinic owes Salus, package D. Not personal
--     data.
-- Money is an integer in the currency's minor unit with its ISO 4217 code; a VAT rate is in basis points (1700 is
-- 17 %), NULL for exempt (Q9).
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

-- Visit modes.
UPDATE schedule
SET availability = (SELECT coalesce(jsonb_agg(r || '{"mode": "video"}' ORDER BY n), '[]')
                    FROM jsonb_array_elements(availability) WITH ORDINALITY AS e (r, n));

ALTER TABLE slot ADD COLUMN visit_mode text DEFAULT 'video' NOT NULL CHECK (visit_mode IN ('in-person', 'video'));
ALTER TABLE appointment ADD COLUMN visit_mode text DEFAULT 'video' NOT NULL CHECK (visit_mode IN ('in-person', 'video'));
-- The app inserts the slot's mode with the appointment; no UPDATE grant: it never changes.
GRANT SELECT (visit_mode) ON appointment TO salus_stats;

-- The clinic's address and billing details, set by its admin.
ALTER TABLE organization
    ADD COLUMN address_line text CHECK (address_line <> ''),
    ADD COLUMN postal_code text CHECK (postal_code <> ''),
    ADD COLUMN legal_name text CHECK (legal_name <> ''),
    ADD COLUMN tax_id text CHECK (tax_id ~ '^[0-9]{12,13}$'), -- the JIB (13 digits) or the IDB/PDV number (12)
    ADD COLUMN vat_registered boolean DEFAULT false NOT NULL,
    ADD COLUMN billing_email text CHECK (billing_email <> '');
GRANT UPDATE (address_line, postal_code, legal_name, tax_id, vat_registered, billing_email) ON organization TO salus_app;

-- The price list.
CREATE TABLE charge_item_definition (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    name text NOT NULL CHECK (name <> ''),
    code text CHECK (code <> ''),
    price_minor bigint NOT NULL CHECK (price_minor >= 0),
    currency text DEFAULT 'BAM' NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    duration_minutes integer CHECK (duration_minutes BETWEEN 5 AND 480), -- NULL: the slot's length
    vat_rate_bp integer CHECK (vat_rate_bp BETWEEN 0 AND 10000),
    modes text[] DEFAULT '{in-person,video}' NOT NULL
        CHECK (cardinality(modes) > 0 AND modes <@ ARRAY['in-person', 'video']::text[]),
    active boolean DEFAULT true NOT NULL,
    version_id integer DEFAULT 1 NOT NULL
);

ALTER TABLE charge_item_definition ENABLE ROW LEVEL SECURITY;
CREATE POLICY charge_item_definition_clinic ON charge_item_definition
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- Prices are shown at booking: the directory sees the active services of listed clinics (organization's policy).
CREATE POLICY charge_item_definition_directory ON charge_item_definition FOR SELECT
    USING (active
           AND NULLIF(current_setting('app.directory', true), '') = 'public'
           AND EXISTS (SELECT 1 FROM organization o WHERE o.id = charge_item_definition.organization_id));
-- Never deleted: a service no longer offered is made inactive, and receipts copy what they need.
GRANT SELECT, INSERT ON charge_item_definition TO salus_app;
GRANT UPDATE (name, code, price_minor, currency, duration_minutes, vat_rate_bp, modes, active, version_id, updated_at)
    ON charge_item_definition TO salus_app;

-- A service's name for patients in another language, as questionnaire_translation.
CREATE TABLE charge_item_definition_translation (
    charge_item_definition_id uuid NOT NULL REFERENCES charge_item_definition (id) ON DELETE CASCADE,
    language language NOT NULL,
    name text NOT NULL CHECK (name <> ''),
    PRIMARY KEY (charge_item_definition_id, language)
);

ALTER TABLE charge_item_definition_translation ENABLE ROW LEVEL SECURITY;
-- Seen where its service is seen; written only in the service's clinic.
CREATE POLICY charge_item_definition_translation_read ON charge_item_definition_translation FOR SELECT
    USING (EXISTS (SELECT 1 FROM charge_item_definition d WHERE d.id = charge_item_definition_translation.charge_item_definition_id));
CREATE POLICY charge_item_definition_translation_clinic ON charge_item_definition_translation
    USING (EXISTS (SELECT 1 FROM charge_item_definition d
                   WHERE d.id = charge_item_definition_translation.charge_item_definition_id
                     AND d.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid));
GRANT SELECT, INSERT, DELETE ON charge_item_definition_translation TO salus_app;
GRANT UPDATE (name) ON charge_item_definition_translation TO salus_app;

-- Receipts and invoices.
-- For invoice's reference to its visit, which holds even when the receipt names no patient.
ALTER TABLE appointment ADD CONSTRAINT appointment_id_organization_key UNIQUE (id, organization_id);

CREATE TABLE invoice (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    patient_id uuid REFERENCES patient (id),
    company_name text CHECK (company_name <> ''),
    company_tax_id text CHECK (company_tax_id ~ '^[0-9]{12,13}$'),
    appointment_id uuid,
    kind text NOT NULL CHECK (kind IN ('receipt', 'invoice')),
    reverses_id uuid,
    number bigint NOT NULL CHECK (number > 0),
    issued_at timestamptz NOT NULL,
    issued_by_account_id uuid NOT NULL REFERENCES account (id),
    payment_method text NOT NULL CHECK (payment_method IN ('cash', 'card', 'transfer')),
    total_minor bigint NOT NULL,
    currency text DEFAULT 'BAM' NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    fiscal_status text DEFAULT 'pending' NOT NULL CHECK (fiscal_status IN ('pending', 'fiscalized', 'failed')),
    fiscal_id text,
    fiscal_code text,
    fiscalized_at timestamptz,
    -- A patient, a company or neither (an anonymous cash sale); a company's tax id only with its name.
    CONSTRAINT invoice_buyer CHECK (patient_id IS NULL OR company_name IS NULL),
    CONSTRAINT invoice_company CHECK (company_tax_id IS NULL OR company_name IS NOT NULL),
    CONSTRAINT invoice_fiscalized CHECK ((fiscal_status = 'fiscalized') = (fiscal_id IS NOT NULL AND fiscal_code IS NOT NULL
                                                                          AND fiscalized_at IS NOT NULL)),
    CONSTRAINT invoice_number_key UNIQUE (organization_id, number),
    -- A receipt is reversed once, by a receipt of its own clinic.
    CONSTRAINT invoice_reverses_key UNIQUE (reverses_id),
    CONSTRAINT invoice_id_organization_key UNIQUE (id, organization_id),
    CONSTRAINT invoice_reverses_fkey FOREIGN KEY (reverses_id, organization_id) REFERENCES invoice (id, organization_id),
    -- A visit's receipt is of the visit's clinic and, when it names a patient, of the visit's patient (MATCH SIMPLE
    -- checks the second key only with a patient, so the first holds the clinic alone).
    CONSTRAINT invoice_appointment_fkey FOREIGN KEY (appointment_id, organization_id)
        REFERENCES appointment (id, organization_id),
    CONSTRAINT invoice_appointment_patient_fkey FOREIGN KEY (appointment_id, organization_id, patient_id)
        REFERENCES appointment (id, organization_id, patient_id)
);

CREATE INDEX invoice_patient_idx ON invoice (patient_id, issued_at) WHERE patient_id IS NOT NULL;
CREATE INDEX invoice_appointment_idx ON invoice (appointment_id) WHERE appointment_id IS NOT NULL;
CREATE INDEX invoice_issued_idx ON invoice (organization_id, issued_at);

ALTER TABLE invoice ENABLE ROW LEVEL SECURITY;
CREATE POLICY invoice_clinic ON invoice
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
CREATE POLICY invoice_own ON invoice FOR SELECT
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
-- Issued once; only the fiscal columns change (the trigger says how). No DELETE: retention decides (placeholder).
GRANT SELECT, INSERT ON invoice TO salus_app;
GRANT UPDATE (fiscal_status, fiscal_id, fiscal_code, fiscalized_at) ON invoice TO salus_app;

-- An issued receipt never changes, for every role, the owner too: only the fiscal columns move, from pending to
-- fiscalized or failed, and from failed to fiscalized (the retried report). A fiscalized receipt never changes. Removal
-- is retention's (no DELETE grant to the app); the rule that removes receipts comes with the legal period.
-- +goose StatementBegin
CREATE FUNCTION invoice_guard_change() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
DECLARE
    fiscal text[] := ARRAY['fiscal_status', 'fiscal_id', 'fiscal_code', 'fiscalized_at'];
BEGIN
    IF (to_jsonb(NEW) - fiscal) IS DISTINCT FROM (to_jsonb(OLD) - fiscal) THEN
        RAISE EXCEPTION 'invoice: an issued receipt never changes' USING ERRCODE = 'check_violation';
    END IF;
    IF to_jsonb(NEW) IS NOT DISTINCT FROM to_jsonb(OLD) THEN
        RETURN NEW;
    END IF;
    IF NOT ((OLD.fiscal_status = 'pending' AND NEW.fiscal_status IN ('fiscalized', 'failed'))
            OR (OLD.fiscal_status = 'failed' AND NEW.fiscal_status = 'fiscalized')) THEN
        RAISE EXCEPTION 'invoice: the fiscal status moves from pending once, or from failed to fiscalized'
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER invoice_guard_change BEFORE UPDATE ON invoice
    FOR EACH ROW EXECUTE FUNCTION invoice_guard_change();

-- +goose StatementBegin
CREATE FUNCTION invoice_reject_change() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
BEGIN
    RAISE EXCEPTION '% is part of an issued receipt: % is not allowed', TG_TABLE_NAME, TG_OP USING ERRCODE = 'check_violation';
END
$$;
-- +goose StatementEnd

CREATE TABLE invoice_line (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    invoice_id uuid NOT NULL REFERENCES invoice (id),
    charge_item_definition_id uuid REFERENCES charge_item_definition (id), -- NULL for an item off the price list
    name text NOT NULL CHECK (name <> ''), -- copied at issue
    quantity integer NOT NULL CHECK (quantity <> 0),
    unit_price_minor bigint NOT NULL CHECK (unit_price_minor >= 0),
    discount_minor bigint DEFAULT 0 NOT NULL CHECK (discount_minor >= 0),
    vat_rate_bp integer CHECK (vat_rate_bp BETWEEN 0 AND 10000), -- copied at issue; NULL for exempt
    total_minor bigint NOT NULL
);

CREATE INDEX invoice_line_invoice_idx ON invoice_line (invoice_id);

ALTER TABLE invoice_line ENABLE ROW LEVEL SECURITY;
-- Seen where its receipt is seen; written only in the receipt's clinic.
CREATE POLICY invoice_line_read ON invoice_line FOR SELECT
    USING (EXISTS (SELECT 1 FROM invoice i WHERE i.id = invoice_line.invoice_id));
CREATE POLICY invoice_line_clinic ON invoice_line FOR INSERT
    WITH CHECK (EXISTS (SELECT 1 FROM invoice i
                        WHERE i.id = invoice_line.invoice_id
                          AND i.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid));
GRANT SELECT, INSERT ON invoice_line TO salus_app;

CREATE TRIGGER invoice_line_no_change BEFORE UPDATE ON invoice_line
    FOR EACH ROW EXECUTE FUNCTION invoice_reject_change();

-- A line is added only in the transaction that issues its receipt (now() is the transaction's start, as the receipt's
-- created_at), and only from the receipt's clinic's price list (a foreign key alone would take another clinic's: it
-- bypasses row-level security). For every role.
-- +goose StatementBegin
CREATE FUNCTION invoice_line_guard_insert() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog, public
AS $$
DECLARE
    org uuid;
    created timestamptz;
BEGIN
    SELECT i.organization_id, i.created_at INTO org, created FROM invoice i WHERE i.id = NEW.invoice_id;
    IF created IS DISTINCT FROM now() THEN
        RAISE EXCEPTION 'invoice_line: lines are added only with their receipt' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.charge_item_definition_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM charge_item_definition d WHERE d.id = NEW.charge_item_definition_id AND d.organization_id = org) THEN
        RAISE EXCEPTION 'invoice_line: a service of another clinic' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER invoice_line_guard_insert BEFORE INSERT ON invoice_line
    FOR EACH ROW EXECUTE FUNCTION invoice_line_guard_insert();

-- The clinic's receipt numbers: one row per clinic, locked by the UPDATE … RETURNING of the receipt's transaction, so
-- numbers have no gap and no repeat (package C).
CREATE TABLE invoice_sequence (
    organization_id uuid NOT NULL PRIMARY KEY REFERENCES organization (id),
    last_number bigint DEFAULT 0 NOT NULL CHECK (last_number >= 0)
);

ALTER TABLE invoice_sequence ENABLE ROW LEVEL SECURITY;
CREATE POLICY invoice_sequence_clinic ON invoice_sequence
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
GRANT SELECT, INSERT ON invoice_sequence TO salus_app;
GRANT UPDATE (last_number) ON invoice_sequence TO salus_app;

-- What a clinic owes Salus. Plans are edited by the owner in salusctl (as the table's owner); the app reads them.
CREATE TABLE subscription_plan (
    code text NOT NULL PRIMARY KEY CHECK (code ~ '^[a-z][a-z0-9_]*$'),
    name text NOT NULL CHECK (name <> ''),
    price_per_seat_minor bigint NOT NULL CHECK (price_per_seat_minor >= 0),
    currency text DEFAULT 'BAM' NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    features text[] DEFAULT '{}' NOT NULL,
    active boolean DEFAULT true NOT NULL
);

GRANT SELECT ON subscription_plan TO salus_app;

CREATE TABLE clinic_subscription (
    organization_id uuid NOT NULL PRIMARY KEY REFERENCES organization (id),
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    plan_code text NOT NULL REFERENCES subscription_plan (code),
    status text NOT NULL CHECK (status IN ('trial', 'active', 'past_due', 'suspended', 'cancelled')),
    trial_ends_at timestamptz,
    current_period_start timestamptz,
    current_period_end timestamptz,
    CONSTRAINT clinic_subscription_period CHECK ((current_period_start IS NULL) = (current_period_end IS NULL)
                                                 AND (current_period_start IS NULL OR current_period_start < current_period_end))
);

ALTER TABLE clinic_subscription ENABLE ROW LEVEL SECURITY;
CREATE POLICY clinic_subscription_clinic ON clinic_subscription
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
GRANT SELECT, INSERT ON clinic_subscription TO salus_app;
GRANT UPDATE (plan_code, status, trial_ends_at, current_period_start, current_period_end, updated_at)
    ON clinic_subscription TO salus_app;

CREATE TABLE subscription_invoice (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organization (id),
    number bigint NOT NULL UNIQUE CHECK (number > 0), -- Salus's own sequence, package D
    period_start date NOT NULL,
    period_end date NOT NULL,
    seats integer NOT NULL CHECK (seats >= 0),
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    currency text DEFAULT 'BAM' NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    issued_at timestamptz NOT NULL,
    due_at timestamptz NOT NULL,
    paid_at timestamptz,
    status text DEFAULT 'open' NOT NULL CHECK (status IN ('open', 'paid', 'void')),
    CONSTRAINT subscription_invoice_period CHECK (period_start < period_end),
    CONSTRAINT subscription_invoice_paid CHECK ((status = 'paid') = (paid_at IS NOT NULL)),
    -- The monthly job issues one invoice per clinic and month (package D).
    CONSTRAINT subscription_invoice_month_key UNIQUE (organization_id, period_start)
);

ALTER TABLE subscription_invoice ENABLE ROW LEVEL SECURITY;
CREATE POLICY subscription_invoice_clinic ON subscription_invoice
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- The monthly job inserts; marking paid or void is the owner's, in salusctl (as the table's owner). No UPDATE grant.
GRANT SELECT, INSERT ON subscription_invoice TO salus_app;
