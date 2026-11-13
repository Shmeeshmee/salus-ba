-- +goose Up
-- No-show reports (SAL-36, docs/plans/appointments.md): the clinic statistics count the appointments by how they were
-- booked too, for the slots refilled from the waitlist. The number is a placeholder: it takes the next free number when
-- it is merged.
--   - clinic_appointment_stat.booked_via: 'direct' or 'waitlist' (appointment.booked_via, 00054), in the key. Rows
--     counted before it are direct bookings: no waitlist existed then. Counts only, no patient identifier.

ALTER TABLE clinic_appointment_stat
    ADD COLUMN booked_via text DEFAULT 'direct' NOT NULL CHECK (booked_via IN ('direct', 'waitlist')),
    DROP CONSTRAINT clinic_appointment_stat_pkey,
    ADD PRIMARY KEY (organization_id, day, hour, practitioner_role_id, status, booked_via);
