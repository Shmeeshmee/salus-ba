-- +goose Up
-- A booking may name a service, package E of docs/plans/clinic-cms.md.
--   - appointment.service_id: the price-list service the patient picked when booking, or NULL when they skipped it.
--     Shown to the clinic, and the receipt for the visit starts from its line. Nothing is charged in the app.
--   - Held to the appointment's own clinic by a composite foreign key: a service of another clinic cannot be named.
--     That it is active and offered in the appointment's visit mode is the service's check at booking
--     (appointment.ErrInvalidService): a service made inactive later stays on the appointments that named it.
--   - No UPDATE grant: the service is set at booking and kept, a reschedule included (the same appointment).
--   - Personal data as the appointment is (which service a patient booked at a clinic); its retention is the
--     appointment's row's.

ALTER TABLE charge_item_definition ADD CONSTRAINT charge_item_definition_id_organization_key UNIQUE (id, organization_id);

ALTER TABLE appointment ADD COLUMN service_id uuid;
ALTER TABLE appointment ADD CONSTRAINT appointment_service_fkey FOREIGN KEY (service_id, organization_id)
    REFERENCES charge_item_definition (id, organization_id);
