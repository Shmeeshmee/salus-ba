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

-- The column grant applies under every UPDATE policy of appointment (the patient's own, the clinic's with the
-- consent, a revoked clinic's past ones), so this trigger says who may change what: the period and the slot only in
-- the patient's own scope, of a booked appointment that stays booked, the period and the slot together, the reschedule count up by
-- exactly one with every change of the period and never otherwise; the confirmation only in the patient's scope (their app, or the
-- one-time link, which acts as the patient), and never cleared once set. It guards every role but the table's
-- owner (migrations, salusctl, test fixtures), which row-level security does not hold either (as 00043's
-- profile_entry_update_check).
-- +goose StatementBegin
CREATE FUNCTION appointment_guard_change() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
DECLARE
    own boolean := NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL
                   AND NULLIF(current_setting('app.patient_id', true), '')::uuid = OLD.patient_id;
    moved boolean := NEW.period IS DISTINCT FROM OLD.period OR NEW.slot_id IS DISTINCT FROM OLD.slot_id;
BEGIN
    IF current_user = (SELECT r.rolname FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner WHERE c.oid = TG_RELID) THEN
        RETURN NEW;
    END IF;
    IF moved AND NOT (own AND OLD.status = 'booked' AND NEW.status = 'booked') THEN
        RAISE EXCEPTION 'appointment: only its patient moves a booked appointment' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF (NEW.period IS DISTINCT FROM OLD.period) <> (NEW.slot_id IS DISTINCT FROM OLD.slot_id) THEN
        RAISE EXCEPTION 'appointment: a reschedule changes the period and the slot together' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.period IS DISTINCT FROM OLD.period AND NEW.reschedule_count IS DISTINCT FROM OLD.reschedule_count + 1 THEN
        RAISE EXCEPTION 'appointment: a reschedule counts once' USING ERRCODE = 'check_violation';
    END IF;
    IF NEW.period IS NOT DISTINCT FROM OLD.period AND NEW.reschedule_count IS DISTINCT FROM OLD.reschedule_count THEN
        RAISE EXCEPTION 'appointment: the reschedule count changes only with the period' USING ERRCODE = 'check_violation';
    END IF;
    IF (NEW.confirmed_at IS DISTINCT FROM OLD.confirmed_at OR NEW.confirmed_via IS DISTINCT FROM OLD.confirmed_via) AND NOT own THEN
        RAISE EXCEPTION 'appointment: only its patient confirms' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF OLD.confirmed_at IS NOT NULL AND NEW.confirmed_at IS NULL THEN
        RAISE EXCEPTION 'appointment: a confirmation is never cleared' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER appointment_guard_change BEFORE UPDATE OF period, slot_id, reschedule_count, confirmed_at, confirmed_via ON appointment
    FOR EACH ROW EXECUTE FUNCTION appointment_guard_change();
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
    -- status_set: booked to fulfilled, noshow or cancelled (every cancellation, SAL-32 review); status_corrected:
    -- noshow and fulfilled into each other (SAL-36).
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
    CONSTRAINT appointment_change_status CHECK ((kind IN ('status_set', 'status_corrected')) = (from_status IS NOT NULL AND to_status IS NOT NULL)),
    CONSTRAINT appointment_change_status_set CHECK (kind <> 'status_set' OR (from_status = 'booked' AND to_status IN ('fulfilled', 'noshow', 'cancelled'))),
    CONSTRAINT appointment_change_status_corrected CHECK (kind <> 'status_corrected'
                                                          OR (from_status IN ('fulfilled', 'noshow') AND to_status IN ('fulfilled', 'noshow') AND from_status <> to_status))
);

CREATE INDEX appointment_change_appointment ON appointment_change (appointment_id, at);

ALTER TABLE appointment_change ENABLE ROW LEVEL SECURITY;
-- Exactly where the appointment is seen: appointment's own policies decide (the patient's own, the clinic's with the
-- consent or after a revocation the past ones, a job's one appointment).
CREATE POLICY appointment_change_seen ON appointment_change FOR SELECT
    USING (EXISTS (SELECT 1 FROM appointment a WHERE a.id = appointment_change.appointment_id));
-- A row is written only for an appointment the scope sees, and says who wrote it as the scope is: in the patient's own
-- scope their app or link (by the patient's own account) or a job (the revocation's, by no account); in the clinic's
-- scope its staff or a job. A patient's scope never claims staff, a clinic's never the patient's app or link.
CREATE POLICY appointment_change_write ON appointment_change FOR INSERT
    WITH CHECK (EXISTS (SELECT 1 FROM appointment a WHERE a.id = appointment_change.appointment_id)
                AND CASE WHEN NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL
                         THEN patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
                              AND (via = 'system'
                                   OR (via IN ('app', 'link')
                                       AND by_account_id = (SELECT p.account_id FROM patient p WHERE p.id = appointment_change.patient_id)))
                         ELSE via IN ('staff', 'system')
                    END);

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
    -- The start the link was issued for: SAL-34 refuses a link whose start_at is not the appointment's start now (a
    -- reschedule leaves the old reminders' links useless), and the link expires at it.
    start_at timestamptz NOT NULL,
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    channel text NOT NULL CHECK (channel IN ('email', 'sms')),
    language language NOT NULL,
    used_action text CHECK (used_action IN ('confirm', 'cancel')),
    CONSTRAINT appointment_action_token_appointment_fkey FOREIGN KEY (appointment_id, organization_id, patient_id)
        REFERENCES appointment (id, organization_id, patient_id),
    CONSTRAINT appointment_action_token_used CHECK ((used_at IS NULL) = (used_action IS NULL)),
    CONSTRAINT appointment_action_token_expiry CHECK (expires_at > created_at AND expires_at = start_at)
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

-- Single use in the database: a used link never changes again, for every role (the use is a conditional UPDATE,
-- WHERE used_at IS NULL, in the transaction that confirms or cancels).
-- +goose StatementBegin
CREATE FUNCTION appointment_action_token_used_once() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
BEGIN
    IF OLD.used_at IS NOT NULL THEN
        RAISE EXCEPTION 'appointment_action_token: a used link never changes' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER appointment_action_token_used_once BEFORE UPDATE ON appointment_action_token
    FOR EACH ROW EXECUTE FUNCTION appointment_action_token_used_once();
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
