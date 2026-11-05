-- +goose Up
-- Patients (docs/booking, package 2): one FHIR Patient per patient account, not clinic-scoped, with the
-- fields of SPEC.md section 4 and nothing else. Created by the patient's first save with every required
-- field, so there is no incomplete row. Row-level secured: a patient's own transaction sees only their row,
-- by app.patient_id or, while the profile is looked up by account, app.account_id. The clinic branch (an
-- active consent, a past appointment) is added by the migrations that create consent and appointment.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE patient (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    account_id uuid NOT NULL REFERENCES account (id),
    version_id integer DEFAULT 1 NOT NULL,
    birth_date date NOT NULL CHECK (birth_date >= DATE '1900-01-01'),
    -- One HumanName (use official): given and family, trimmed by the service.
    given_name text NOT NULL CHECK (btrim(given_name) <> ''),
    family_name text NOT NULL CHECK (btrim(family_name) <> ''),
    -- Optional, E.164: + and 8 to 15 digits.
    phone text CHECK (phone ~ '^\+[0-9]{8,15}$'),
    CONSTRAINT patient_account_id_key UNIQUE (account_id)
);

ALTER TABLE patient ENABLE ROW LEVEL SECURITY;
CREATE POLICY patient_own ON patient
    USING (id = NULLIF(current_setting('app.patient_id', true), '')::uuid
           OR account_id = NULLIF(current_setting('app.account_id', true), '')::uuid);

-- No DELETE or TRUNCATE: erasure comes with the subject-rights feature. The account a profile belongs to
-- never changes.
GRANT SELECT, INSERT ON patient TO salus_app;
GRANT UPDATE (given_name, family_name, birth_date, phone, updated_at, version_id) ON patient TO salus_app;
