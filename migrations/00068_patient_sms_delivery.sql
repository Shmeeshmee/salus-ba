-- +goose Up
-- SMS reminders for patients without an account (docs/clinic-cms/WAVE4.md R3, decision V3): a delivery is addressed to
-- an account (SAL-33) or, for a profile the clinic registered (no account), to the patient, by SMS only. Ids only, as
-- before: the phone is read at the send, never stored here.

ALTER TABLE notification_delivery ALTER COLUMN account_id DROP NOT NULL;

-- Deleted with the patient's profile, as an account's deliveries are with the account.
ALTER TABLE notification_delivery ADD COLUMN patient_id uuid REFERENCES patient (id) ON DELETE CASCADE;

ALTER TABLE notification_delivery
    ADD CONSTRAINT notification_delivery_recipient CHECK ((account_id IS NULL) <> (patient_id IS NULL)),
    ADD CONSTRAINT notification_delivery_patient_sms CHECK (patient_id IS NULL OR channel = 'sms');

CREATE INDEX notification_delivery_patient ON notification_delivery (patient_id) WHERE patient_id IS NOT NULL;
