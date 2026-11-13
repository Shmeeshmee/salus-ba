-- +goose Up
-- Appointment reminders (SAL-33, docs/plans/appointments.md "Contract of wave 1", ADR 0015 "Reminders"). The number is a
-- placeholder: it takes the next free number when it is merged.
--   - appointment_reminder (appointment): one row per appointment, offset and start, due at the start minus the offset,
--     scheduled until its dispatch job runs (or cancelled with the appointment, or skipped when it no longer applies).
--   - notification_delivery (notification): one row per reminder (or waitlist offer, SAL-35), channel and device; ids
--     only: no address, number, token or text. Named by its id, like push_device (00047): not row-level secured.
--   - notification_preference (notification): the patient's channels, quiet hours and device time zone, by account.
--   - sms_usage (notification): messages and parts a clinic sent per month (UTC), counted in the SMS send's
--     transaction. Not personal data.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE appointment_reminder (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    start_at timestamptz NOT NULL, -- the appointment's start it was scheduled for: a reschedule leaves it behind
    due_at timestamptz NOT NULL,
    dispatched_at timestamptz,
    appointment_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    patient_id uuid NOT NULL, -- copied for row-level security
    offset_minutes integer NOT NULL CHECK (offset_minutes BETWEEN 15 AND 10080),
    state text DEFAULT 'scheduled' NOT NULL CHECK (state IN ('scheduled', 'dispatched', 'cancelled', 'skipped')),
    CONSTRAINT appointment_reminder_appointment_fkey FOREIGN KEY (appointment_id, organization_id, patient_id)
        REFERENCES appointment (id, organization_id, patient_id),
    CONSTRAINT appointment_reminder_once UNIQUE (appointment_id, offset_minutes, start_at),
    CONSTRAINT appointment_reminder_due CHECK (due_at = start_at - make_interval(mins => offset_minutes)),
    CONSTRAINT appointment_reminder_dispatched CHECK ((state = 'dispatched') = (dispatched_at IS NOT NULL))
);

-- The sweep's scan (appointment_reminder_scheduled) and the retention's.
CREATE INDEX appointment_reminder_scheduled_due ON appointment_reminder (due_at, id) WHERE state = 'scheduled';
CREATE INDEX appointment_reminder_due_at ON appointment_reminder (due_at);

ALTER TABLE appointment_reminder ENABLE ROW LEVEL SECURITY;
-- The patient's own: their booking, reschedule and cancellation schedule and cancel them.
CREATE POLICY appointment_reminder_own ON appointment_reminder
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
-- The clinic's: the dispatch job (a notifier job in the clinic's scope) and the staff's cancellation. A reminder holds
-- ids and times only; the appointment behind it is read through appointment's own policies.
CREATE POLICY appointment_reminder_clinic ON appointment_reminder
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

GRANT SELECT, INSERT ON appointment_reminder TO salus_app;
GRANT UPDATE (state, dispatched_at) ON appointment_reminder TO salus_app;
-- Retention: deleted 90 days after its due time.
GRANT SELECT, DELETE ON appointment_reminder TO salus_retention;

-- Sent once in the database: a reminder leaves scheduled once and never changes after, for every role.
-- +goose StatementBegin
CREATE FUNCTION appointment_reminder_once() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
BEGIN
    IF OLD.state <> 'scheduled' THEN
        RAISE EXCEPTION 'appointment_reminder: a reminder leaves scheduled once' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER appointment_reminder_once BEFORE UPDATE ON appointment_reminder
    FOR EACH ROW EXECUTE FUNCTION appointment_reminder_once();

-- appointment_reminder_scheduled gives the sweep (appointment_reminders_sweep, hourly) the reminders of every clinic
-- still scheduled, whatever the scope: as a view it reads appointment_reminder with its owner's rights, which row-level
-- security does not hold. Ids and times only; the sweep takes those due whose start is still ahead (their dispatch job
-- was lost) and enqueues their dispatch again.
CREATE VIEW appointment_reminder_scheduled WITH (security_barrier) AS
    SELECT id, organization_id, due_at, start_at FROM appointment_reminder WHERE state = 'scheduled';

GRANT SELECT ON appointment_reminder_scheduled TO salus_app;

-- The one-time link names the reminder it was sent in (00054 left the key to this file).
ALTER TABLE appointment_action_token
    ADD CONSTRAINT appointment_action_token_reminder_fkey FOREIGN KEY (reminder_id) REFERENCES appointment_reminder (id);

CREATE TABLE notification_delivery (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    sent_at timestamptz,
    source_id uuid NOT NULL, -- the reminder (appointment_reminder) or the offer (waitlist_offer, SAL-35)
    organization_id uuid NOT NULL REFERENCES organization (id),
    -- Deleted with the account (erasure, the subject-rights feature).
    account_id uuid NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    device_id uuid, -- the push_device, for a push; no key: a device goes when the app unregisters
    segments integer CHECK (segments BETWEEN 1 AND 10), -- an SMS's parts, once sent
    kind text NOT NULL CHECK (kind IN ('reminder', 'waitlist_offer')),
    channel text NOT NULL CHECK (channel IN ('push', 'email', 'sms')),
    state text DEFAULT 'pending' NOT NULL CHECK (state IN ('pending', 'sent', 'cancelled', 'failed')),
    CONSTRAINT notification_delivery_once UNIQUE NULLS NOT DISTINCT (kind, source_id, channel, device_id),
    CONSTRAINT notification_delivery_device CHECK ((channel = 'push') = (device_id IS NOT NULL)),
    CONSTRAINT notification_delivery_sent CHECK ((state = 'sent') = (sent_at IS NOT NULL)),
    CONSTRAINT notification_delivery_segments CHECK (segments IS NULL OR (channel = 'sms' AND state = 'sent'))
);

CREATE INDEX notification_delivery_account ON notification_delivery (account_id);
-- Retention: 90 days after it was sent, or created when it never was.
CREATE INDEX notification_delivery_retention ON notification_delivery ((coalesce(sent_at, created_at)));

GRANT SELECT, INSERT ON notification_delivery TO salus_app;
GRANT UPDATE (state, sent_at, segments) ON notification_delivery TO salus_app;
GRANT SELECT, DELETE ON notification_delivery TO salus_retention;

-- Sent once in the database: a delivery leaves pending once, for every role.
-- +goose StatementBegin
CREATE FUNCTION notification_delivery_once() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
BEGIN
    IF OLD.state <> 'pending' THEN
        RAISE EXCEPTION 'notification_delivery: a delivery leaves pending once' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER notification_delivery_once BEFORE UPDATE ON notification_delivery
    FOR EACH ROW EXECUTE FUNCTION notification_delivery_once();

-- The patient's channels and quiet hours (in their zone; none when both are null) and their device's IANA zone (null:
-- the clinic's is used). No row: push and e-mail on, SMS off, quiet hours 22:00 to 08:00.
CREATE TABLE notification_preference (
    account_id uuid NOT NULL PRIMARY KEY REFERENCES account (id) ON DELETE CASCADE,
    updated_at timestamptz DEFAULT now() NOT NULL,
    quiet_start time,
    quiet_end time,
    push boolean NOT NULL,
    email boolean NOT NULL,
    sms boolean NOT NULL,
    time_zone text CHECK (char_length(time_zone) BETWEEN 1 AND 64),
    CONSTRAINT notification_preference_quiet CHECK ((quiet_start IS NULL) = (quiet_end IS NULL))
);

GRANT SELECT, INSERT ON notification_preference TO salus_app;
GRANT UPDATE (push, email, sms, quiet_start, quiet_end, time_zone, updated_at) ON notification_preference TO salus_app;

CREATE TABLE sms_usage (
    organization_id uuid NOT NULL REFERENCES organization (id),
    month date NOT NULL CHECK (extract(day FROM month) = 1),
    messages integer DEFAULT 0 NOT NULL CHECK (messages >= 0),
    segments integer DEFAULT 0 NOT NULL CHECK (segments >= 0),
    PRIMARY KEY (organization_id, month)
);

ALTER TABLE sms_usage ENABLE ROW LEVEL SECURITY;
CREATE POLICY sms_usage_clinic ON sms_usage
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

GRANT SELECT, INSERT ON sms_usage TO salus_app;
GRANT UPDATE (messages, segments) ON sms_usage TO salus_app;
