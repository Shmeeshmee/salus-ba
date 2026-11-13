-- +goose Up
-- The appointments epic's shared contract (SAL-32, docs/plans/appointments.md "Contract (step 0)", ADR 0015). The
-- number is a placeholder: 00051 to 00053 are taken by branches merging first, and this file takes the next free number when
-- it is merged.
--   - appointment: whether the patient confirmed it and how, how often it was rescheduled, and how it was booked. A
--     reschedule moves the same appointment to another slot of its role, so period and slot_id may now change (00019's
--     "when never changes" is amended: who, with whom and where still never change).
--   - appointment_change: the history of an appointment's confirmations, reschedules and statuses, append-only, seen
--     exactly where its appointment is seen.
--   - appointment_action_token: a one-time link of a reminder, stored as its SHA-256 only; found by hash through a
--     SECURITY DEFINER function, then read in the token's patient's scope.
--   - organization_appointment_settings: the clinic's reminder, SMS, reschedule, waitlist and no-show rules (no
--     personal data), as organization_chat_settings (00047).
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

ALTER TABLE appointment
    ADD COLUMN confirmed_at timestamptz,
    ADD COLUMN confirmed_via text CHECK (confirmed_via IN ('app', 'link')),
    ADD COLUMN reschedule_count smallint DEFAULT 0 NOT NULL CHECK (reschedule_count >= 0),
    ADD COLUMN booked_via text DEFAULT 'direct' NOT NULL CHECK (booked_via IN ('direct', 'waitlist')),
    ADD CONSTRAINT appointment_confirmed CHECK ((confirmed_at IS NULL) = (confirmed_via IS NULL)),
    -- For the copies of the appointment's clinic and patient in appointment_change and appointment_action_token.
    ADD CONSTRAINT appointment_id_organization_patient_key UNIQUE (id, organization_id, patient_id);

GRANT UPDATE (period, slot_id, confirmed_at, confirmed_via, reschedule_count) ON appointment TO salus_app;
-- The clinic statistics count the appointments booked through the waitlist (SAL-36).
GRANT SELECT (booked_via) ON appointment TO salus_stats;

CREATE TABLE appointment_change (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    at timestamptz NOT NULL,
    appointment_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    patient_id uuid NOT NULL,
    by_account_id uuid REFERENCES account (id), -- null for the system
    from_slot_id uuid,
    to_slot_id uuid,
    from_period tstzrange,
    to_period tstzrange,
    kind text NOT NULL CHECK (kind IN ('confirmed', 'rescheduled', 'status_set', 'status_corrected')),
    via text NOT NULL CHECK (via IN ('app', 'link', 'staff', 'system')),
    agent text CHECK (agent ~ '^[a-z][a-z0-9_]{0,63}$'), -- the job's kind, for the system
    from_status text CHECK (from_status IN ('booked', 'cancelled', 'fulfilled', 'noshow')),
    to_status text CHECK (to_status IN ('booked', 'cancelled', 'fulfilled', 'noshow')),
    CONSTRAINT appointment_change_appointment_fkey FOREIGN KEY (appointment_id, organization_id, patient_id)
        REFERENCES appointment (id, organization_id, patient_id),
    CONSTRAINT appointment_change_who CHECK ((via = 'system') = (agent IS NOT NULL) AND (via = 'system') = (by_account_id IS NULL)),
    CONSTRAINT appointment_change_rescheduled CHECK ((kind = 'rescheduled') = (from_period IS NOT NULL AND to_period IS NOT NULL
                                                     AND from_slot_id IS NOT NULL AND to_slot_id IS NOT NULL)),
    CONSTRAINT appointment_change_status CHECK ((kind IN ('status_set', 'status_corrected')) = (from_status IS NOT NULL AND to_status IS NOT NULL))
);

CREATE INDEX appointment_change_appointment ON appointment_change (appointment_id, at);

ALTER TABLE appointment_change ENABLE ROW LEVEL SECURITY;
-- Exactly where the appointment is seen: appointment's own policies decide (the patient's own, the clinic's with the
-- consent or after a revocation the past ones, a job's one appointment).
CREATE POLICY appointment_change_seen ON appointment_change FOR SELECT
    USING (EXISTS (SELECT 1 FROM appointment a WHERE a.id = appointment_change.appointment_id));
CREATE POLICY appointment_change_write ON appointment_change FOR INSERT
    WITH CHECK (EXISTS (SELECT 1 FROM appointment a WHERE a.id = appointment_change.appointment_id));

-- Append-only, for every role, the owner included (as communication_revision, 00041).
CREATE TRIGGER appointment_change_no_change BEFORE UPDATE OR DELETE ON appointment_change
    FOR EACH ROW EXECUTE FUNCTION communication_reject_removal();
CREATE TRIGGER appointment_change_no_truncate BEFORE TRUNCATE ON appointment_change
    FOR EACH STATEMENT EXECUTE FUNCTION communication_reject_removal();

GRANT SELECT, INSERT ON appointment_change TO salus_app;

CREATE TABLE appointment_action_token (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    appointment_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    patient_id uuid NOT NULL,
    account_id uuid NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    reminder_id uuid, -- SAL-33's appointment_reminder adds the foreign key
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    channel text NOT NULL CHECK (channel IN ('email', 'sms')),
    language language NOT NULL,
    used_action text CHECK (used_action IN ('confirm', 'cancel')),
    CONSTRAINT appointment_action_token_appointment_fkey FOREIGN KEY (appointment_id, organization_id, patient_id)
        REFERENCES appointment (id, organization_id, patient_id),
    CONSTRAINT appointment_action_token_used CHECK ((used_at IS NULL) = (used_action IS NULL)),
    CONSTRAINT appointment_action_token_expiry CHECK (expires_at > created_at)
);

CREATE INDEX appointment_action_token_appointment ON appointment_action_token (appointment_id);
CREATE INDEX appointment_action_token_expires ON appointment_action_token (expires_at);

ALTER TABLE appointment_action_token ENABLE ROW LEVEL SECURITY;
-- The token's patient, as the link's transaction is scoped: reads and uses it.
CREATE POLICY appointment_action_token_own ON appointment_action_token FOR SELECT
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY appointment_action_token_use ON appointment_action_token FOR UPDATE
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
-- A notifier job issues one for the appointment it was given (app.organization_id with app.appointment_id), and reads
-- it back.
CREATE POLICY appointment_action_token_notice ON appointment_action_token FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND appointment_id = NULLIF(current_setting('app.appointment_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
CREATE POLICY appointment_action_token_issue ON appointment_action_token FOR INSERT
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND appointment_id = NULLIF(current_setting('app.appointment_id', true), '')::uuid
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

GRANT SELECT, INSERT ON appointment_action_token TO salus_app;
GRANT UPDATE (used_at, used_action) ON appointment_action_token TO salus_app;
-- Retention: deleted 30 days after it expires.
GRANT SELECT, DELETE ON appointment_action_token TO salus_retention;

-- appointment_action_token_find gives the ids of the token with the hash, whatever the scope: the link's request has
-- no session, and the service then scopes its transaction to the patient it returns. Only a holder of the token knows
-- its hash; nothing else of the row is returned.
-- +goose StatementBegin
CREATE FUNCTION appointment_action_token_find(hash bytea)
    RETURNS TABLE (id uuid, appointment_id uuid, organization_id uuid, patient_id uuid, account_id uuid)
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
    SELECT t.id, t.appointment_id, t.organization_id, t.patient_id, t.account_id
    FROM public.appointment_action_token t
    WHERE t.token_hash = hash AND octet_length(hash) = 32
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION appointment_action_token_find(bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION appointment_action_token_find(bytea) TO salus_app;

CREATE TABLE organization_appointment_settings (
    organization_id uuid NOT NULL PRIMARY KEY REFERENCES organization (id),
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    reminder_offsets_minutes integer[] DEFAULT '{1440,60}' NOT NULL
        CHECK (cardinality(reminder_offsets_minutes) <= 3 AND 15 <= ALL (reminder_offsets_minutes)
               AND 10080 >= ALL (reminder_offsets_minutes) AND array_position(reminder_offsets_minutes, NULL) IS NULL),
    sms_reminders boolean DEFAULT false NOT NULL,
    reschedule_notice_hours integer DEFAULT 24 NOT NULL CHECK (reschedule_notice_hours BETWEEN 0 AND 168),
    reschedule_max integer DEFAULT 2 NOT NULL CHECK (reschedule_max BETWEEN 0 AND 5),
    waitlist_enabled boolean DEFAULT false NOT NULL,
    waitlist_hold_minutes integer DEFAULT 15 NOT NULL CHECK (waitlist_hold_minutes BETWEEN 5 AND 120),
    noshow_flag_threshold integer DEFAULT 0 NOT NULL CHECK (noshow_flag_threshold BETWEEN 0 AND 10),
    noshow_flag_window_days integer DEFAULT 180 NOT NULL CHECK (noshow_flag_window_days BETWEEN 30 AND 730)
);

ALTER TABLE organization_appointment_settings ENABLE ROW LEVEL SECURITY;
CREATE POLICY organization_appointment_settings_clinic ON organization_appointment_settings
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

GRANT SELECT, INSERT ON organization_appointment_settings TO salus_app;
GRANT UPDATE (reminder_offsets_minutes, sms_reminders, reschedule_notice_hours, reschedule_max, waitlist_enabled,
              waitlist_hold_minutes, noshow_flag_threshold, noshow_flag_window_days, updated_at)
    ON organization_appointment_settings TO salus_app;
