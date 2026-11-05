-- +goose Up
-- Appointments (docs/booking, package 6): FHIR Appointment, a patient's booking of a slot with a practitioner
-- role (SPEC.md "Booking rules", section 4). Booked at once, cancelled by the patient (pat) or staff (prov),
-- marked fulfilled or noshow by the practitioner. The database refuses a double booking: no two appointments
-- of one role that are not cancelled overlap (the owner's settled guarantee).
-- Row-level security:
--   - the patient (app.patient_id) sees and writes their own appointments;
--   - a clinic (app.organization_id, neither patient setting set) sees its appointments with a patient who
--     has an active consent with it, and, after a revocation, only those that started before the latest
--     one (SPEC.md "Past and future after revocation"); it changes them with an active consent, after a
--     revocation only marks a past one fulfilled or noshow (the care record), and never books;
--   - the notifier's e-mail jobs (app.organization_id with app.appointment_id) read the one appointment of
--     their clinic they were given, and its patient's row for the account to write to, whatever the consent:
--     a cancellation after a revocation is mailed too.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE appointment (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    cancelled_at timestamptz,
    patient_id uuid NOT NULL REFERENCES patient (id),
    practitioner_role_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    -- The slot booked. No foreign key: a freed slot may be deleted by generation or retention while the
    -- cancelled appointment is kept with the clinical record.
    slot_id uuid NOT NULL,
    cancelled_by_account_id uuid REFERENCES account (id), -- empty when a job cancelled (a revocation)
    version_id integer DEFAULT 1 NOT NULL,
    status text DEFAULT 'booked' NOT NULL CHECK (status IN ('booked', 'cancelled', 'fulfilled', 'noshow')),
    -- http://terminology.hl7.org/CodeSystem/appointment-cancellation-reason: pat (the patient), prov (staff).
    cancellation_reason text CHECK (cancellation_reason IN ('pat', 'prov')),
    period tstzrange NOT NULL CHECK (NOT isempty(period) AND lower_inc(period) AND NOT upper_inc(period)
                                     AND NOT lower_inf(period) AND NOT upper_inf(period)),
    CONSTRAINT appointment_practitioner_role_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT appointment_cancelled CHECK ((status = 'cancelled') = (cancelled_at IS NOT NULL)
                                            AND (status = 'cancelled') = (cancellation_reason IS NOT NULL)
                                            AND (cancelled_by_account_id IS NULL OR status = 'cancelled')),
    CONSTRAINT appointment_no_double_booking EXCLUDE USING gist (practitioner_role_id WITH =, period WITH &&)
        WHERE (status <> 'cancelled')
);

-- A patient's appointments in a clinic (access facts, the revocation job); a clinic's calendar by start.
CREATE INDEX appointment_patient_organization ON appointment (patient_id, organization_id);
CREATE INDEX appointment_organization_start ON appointment (organization_id, lower(period));

ALTER TABLE appointment ENABLE ROW LEVEL SECURITY;
CREATE POLICY appointment_own ON appointment
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY appointment_clinic ON appointment FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND (EXISTS (SELECT 1 FROM consent c
                        WHERE c.patient_id = appointment.patient_id
                          AND c.organization_id = appointment.organization_id
                          AND c.status = 'active')
                OR lower(period) < (SELECT max(c.revoked_at) FROM consent c
                                    WHERE c.patient_id = appointment.patient_id
                                      AND c.organization_id = appointment.organization_id)));
CREATE POLICY appointment_clinic_update ON appointment FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM consent c
                       WHERE c.patient_id = appointment.patient_id
                         AND c.organization_id = appointment.organization_id
                         AND c.status = 'active'));
CREATE POLICY appointment_clinic_record_past ON appointment FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND lower(period) < (SELECT max(c.revoked_at) FROM consent c
                                WHERE c.patient_id = appointment.patient_id
                                  AND c.organization_id = appointment.organization_id))
    WITH CHECK (status IN ('fulfilled', 'noshow'));
CREATE POLICY appointment_notice ON appointment FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND id = NULLIF(current_setting('app.appointment_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- The patient branches that come with appointments: the clinic keeps the patient of a past appointment
-- after a revocation (the service gives it the name only), and the notifier reads the patient of its one
-- appointment. Both read-only; both through appointment's own policies above.
CREATE POLICY patient_past_appointment ON patient FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM appointment a
                       WHERE a.patient_id = patient.id
                         AND a.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                         AND lower(a.period) < (SELECT max(c.revoked_at) FROM consent c
                                                WHERE c.patient_id = patient.id AND c.organization_id = a.organization_id)));
CREATE POLICY patient_notice ON patient FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM appointment a
                       WHERE a.patient_id = patient.id
                         AND a.id = NULLIF(current_setting('app.appointment_id', true), '')::uuid));

-- No DELETE or TRUNCATE: appointments are kept with the clinical record (retention keep). Who, with whom,
-- where and when never change; only the status and its cancellation do.
GRANT SELECT, INSERT ON appointment TO salus_app;
GRANT UPDATE (status, cancelled_at, cancellation_reason, cancelled_by_account_id, updated_at, version_id) ON appointment TO salus_app;
