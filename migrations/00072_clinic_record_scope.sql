-- +goose Up
-- The clinic record's scope (docs/clinic-cms/WAVE5.md S11, package FX): the record a clinic hands over to a patient it
-- registered, and its data export, hold every appointment of the clinic, the future visits a withdrawal of consent
-- cancelled included, since the clinic holds the booking and the data is the patient's. The clinic scope
-- (app.organization_id, neither patient setting set) with app.clinic_record = 'on' reads its own appointments and their
-- patients whatever the consent. Read-only; set only by appointment.Service.ClinicRecord, whose every read is audited.
-- NULLIF and the comparison with 'on': an unset setting reads as '' on a pooled connection and must match nothing.

CREATE POLICY appointment_clinic_record ON appointment FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND current_setting('app.clinic_record', true) = 'on'
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

CREATE POLICY patient_clinic_record ON patient FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND current_setting('app.clinic_record', true) = 'on'
           AND EXISTS (SELECT 1 FROM appointment a
                       WHERE a.patient_id = patient.id
                         AND a.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid));
