-- +goose Up
-- The clinic's own statistics (SAL-20, docs/plans/analytics.md §4, ADR 0013): counts per clinic, local day and
-- hour, practitioner role and specialty, pre-computed by the clinicstats_refresh and clinicstats_nightly jobs.
--   - clinic_appointment_stat: appointments by the local day and hour of their start and their status now.
--   - clinic_diagnosis_stat: confirmed Conditions by the local day and hour of their signing, diagnosis group,
--     subgroup (NULL for none) and ICD-10 code.
-- No patient identifier: counts only. specialty_code is the role's SNOMED CT code at refresh time, '' for none.
-- Written only by salus_stats (a refresh deletes a clinic's days and inserts them again, in one transaction);
-- read by salus_app in the clinic's scope (row-level security, as condition in 00042).
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE clinic_appointment_stat (
    organization_id uuid NOT NULL REFERENCES organization (id),
    practitioner_role_id uuid NOT NULL,
    day date NOT NULL,
    count integer NOT NULL CHECK (count > 0),
    hour smallint NOT NULL CHECK (hour BETWEEN 0 AND 23),
    specialty_code text NOT NULL,
    status text NOT NULL CHECK (status IN ('booked', 'cancelled', 'fulfilled', 'noshow')),
    PRIMARY KEY (organization_id, day, hour, practitioner_role_id, status),
    CONSTRAINT clinic_appointment_stat_role_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id)
);

CREATE TABLE clinic_diagnosis_stat (
    organization_id uuid NOT NULL REFERENCES organization (id),
    practitioner_role_id uuid NOT NULL,
    day date NOT NULL,
    count integer NOT NULL CHECK (count > 0),
    hour smallint NOT NULL CHECK (hour BETWEEN 0 AND 23),
    specialty_code text NOT NULL,
    group_code text NOT NULL REFERENCES diagnosis_group (code),
    subgroup_code text REFERENCES diagnosis_group (code),
    code text NOT NULL CHECK (code ~ '^[A-Z][0-9]{2}(\.[0-9A-Z]{1,4})?$'),
    PRIMARY KEY (organization_id, day, hour, practitioner_role_id, code),
    CONSTRAINT clinic_diagnosis_stat_role_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id)
);

ALTER TABLE clinic_appointment_stat ENABLE ROW LEVEL SECURITY;
CREATE POLICY clinic_appointment_stat_clinic ON clinic_appointment_stat FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
ALTER TABLE clinic_diagnosis_stat ENABLE ROW LEVEL SECURITY;
CREATE POLICY clinic_diagnosis_stat_clinic ON clinic_diagnosis_stat FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

GRANT SELECT ON clinic_appointment_stat, clinic_diagnosis_stat TO salus_app;
-- A refresh's DELETE names its rows by clinic and day, which needs SELECT.
GRANT SELECT, INSERT, DELETE ON clinic_appointment_stat, clinic_diagnosis_stat TO salus_stats;
