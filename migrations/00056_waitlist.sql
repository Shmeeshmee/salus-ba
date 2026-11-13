-- +goose Up
-- The waitlist (SAL-35, docs/plans/appointments.md, ADR 0015 "The waitlist hold lives in Postgres"). The number is a
-- placeholder, renumbered at merge.
--   - slot.status gains busy-tentative (FHIR SlotStatus): a freed slot held for one waitlisted patient. The public
--     directory shows free slots only, and generation and retention delete free slots only, so an offered slot is
--     neither seen nor removed.
--   - waitlist_entry: a patient's wish for a practitioner role or a specialty of a clinic between two dates (the
--     clinic's zone) and, optionally, a time of day (minutes after the clinic's midnight), offered in the order
--     priority descending, then created_at. Active until booked, withdrawn (by the patient, staff or the revocation of
--     the care consent) or past its last day (ends_at); closed_at says when it closed.
--   - waitlist_offer: a slot offered to an entry, held until expires_at; one open offer per slot and per entry. The
--     slot's period is copied, so the patient's app shows it without reading the slot.
-- Row-level security: an entry is its patient's (app.patient_id) and its clinic's while the patient's care consent there
-- is active (as appointment_clinic). An offer is its patient's, and its clinic's (app.organization_id alone) whatever the
-- consent: the jobs that hand a slot on must see every open offer of it. The clinic offers only an entry it sees.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

ALTER TABLE slot DROP CONSTRAINT slot_status_check,
    ADD CONSTRAINT slot_status_check CHECK (status IN ('free', 'busy', 'busy-tentative'));

CREATE TABLE waitlist_entry (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    ends_at timestamptz NOT NULL, -- the end of the latest day in the clinic's zone: past it, the entry has expired
    closed_at timestamptz,        -- when it was booked or withdrawn
    patient_id uuid NOT NULL REFERENCES patient (id),
    organization_id uuid NOT NULL REFERENCES organization (id),
    practitioner_role_id uuid,
    earliest date NOT NULL,
    latest date NOT NULL,
    times_start_minute smallint CHECK (times_start_minute BETWEEN 0 AND 1439),
    times_end_minute smallint CHECK (times_end_minute BETWEEN 1 AND 1440),
    priority smallint DEFAULT 0 NOT NULL CHECK (priority BETWEEN 0 AND 100),
    status text DEFAULT 'active' NOT NULL CHECK (status IN ('active', 'booked', 'withdrawn')),
    specialty text CHECK (specialty ~ '^[0-9]{6,18}$'), -- a SNOMED CT code of practitionerrole.Specialties
    CONSTRAINT waitlist_entry_practitioner_role_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT waitlist_entry_role_or_specialty CHECK ((practitioner_role_id IS NULL) <> (specialty IS NULL)),
    CONSTRAINT waitlist_entry_dates CHECK (earliest <= latest),
    CONSTRAINT waitlist_entry_times CHECK ((times_start_minute IS NULL) = (times_end_minute IS NULL)
                                           AND (times_start_minute IS NULL OR times_start_minute < times_end_minute)),
    CONSTRAINT waitlist_entry_closed CHECK ((status = 'active') = (closed_at IS NULL)),
    -- For the copies of the entry's clinic and patient in waitlist_offer.
    CONSTRAINT waitlist_entry_id_organization_patient_key UNIQUE (id, organization_id, patient_id)
);

-- The offer order of a clinic's active entries; a patient's entries with a clinic (the limit of five, the revocation).
CREATE INDEX waitlist_entry_offer_order ON waitlist_entry (organization_id, priority DESC, created_at) WHERE status = 'active';
CREATE INDEX waitlist_entry_patient_organization ON waitlist_entry (patient_id, organization_id);
-- Retention: closed entries, and active ones past their last day.
CREATE INDEX waitlist_entry_closed_at ON waitlist_entry (closed_at) WHERE closed_at IS NOT NULL;
CREATE INDEX waitlist_entry_ends_at ON waitlist_entry (ends_at) WHERE status = 'active';

ALTER TABLE waitlist_entry ENABLE ROW LEVEL SECURITY;
CREATE POLICY waitlist_entry_own ON waitlist_entry
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY waitlist_entry_clinic ON waitlist_entry FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM consent c
                       WHERE c.patient_id = waitlist_entry.patient_id
                         AND c.organization_id = waitlist_entry.organization_id
                         AND c.status = 'active'));
CREATE POLICY waitlist_entry_clinic_update ON waitlist_entry FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM consent c
                       WHERE c.patient_id = waitlist_entry.patient_id
                         AND c.organization_id = waitlist_entry.organization_id
                         AND c.status = 'active'));

-- Who, which clinic, what and when never change; the patient withdraws, staff order and remove, an acceptance books.
GRANT SELECT, INSERT ON waitlist_entry TO salus_app;
GRANT UPDATE (status, priority, closed_at, updated_at) ON waitlist_entry TO salus_app;
-- Retention: deleted 30 days after it closed or expired, with its offers.
GRANT SELECT, DELETE ON waitlist_entry TO salus_retention;

CREATE TABLE waitlist_offer (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    expires_at timestamptz NOT NULL,
    closed_at timestamptz,
    entry_id uuid NOT NULL,
    slot_id uuid NOT NULL, -- no foreign key: a slot is the slot domain's, and a free one may be deleted after the offer closed
    organization_id uuid NOT NULL,
    patient_id uuid NOT NULL,
    practitioner_role_id uuid NOT NULL, -- the slot's, which the acceptance books with
    appointment_id uuid, -- the appointment an acceptance booked
    state text DEFAULT 'open' NOT NULL CHECK (state IN ('open', 'accepted', 'declined', 'expired', 'withdrawn')),
    period tstzrange NOT NULL CHECK (NOT isempty(period) AND lower_inc(period) AND NOT upper_inc(period)
                                     AND NOT lower_inf(period) AND NOT upper_inf(period)),
    CONSTRAINT waitlist_offer_entry_fkey FOREIGN KEY (entry_id, organization_id, patient_id)
        REFERENCES waitlist_entry (id, organization_id, patient_id) ON DELETE CASCADE,
    CONSTRAINT waitlist_offer_practitioner_role_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT waitlist_offer_appointment_fkey FOREIGN KEY (appointment_id, organization_id, patient_id)
        REFERENCES appointment (id, organization_id, patient_id),
    CONSTRAINT waitlist_offer_closed CHECK ((state = 'open') = (closed_at IS NULL)),
    CONSTRAINT waitlist_offer_accepted CHECK ((state = 'accepted') = (appointment_id IS NOT NULL)),
    CONSTRAINT waitlist_offer_hold CHECK (expires_at > created_at)
);

-- One open offer per slot and per entry.
CREATE UNIQUE INDEX waitlist_offer_open_slot ON waitlist_offer (slot_id) WHERE state = 'open';
CREATE UNIQUE INDEX waitlist_offer_open_entry ON waitlist_offer (entry_id) WHERE state = 'open';
-- An entry is offered a slot once: the slots it was offered.
CREATE INDEX waitlist_offer_entry_slot ON waitlist_offer (entry_id, slot_id);

ALTER TABLE waitlist_offer ENABLE ROW LEVEL SECURITY;
CREATE POLICY waitlist_offer_own ON waitlist_offer FOR SELECT
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY waitlist_offer_own_update ON waitlist_offer FOR UPDATE
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY waitlist_offer_clinic ON waitlist_offer FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
CREATE POLICY waitlist_offer_clinic_update ON waitlist_offer FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- The clinic offers a slot to an active entry it sees (the entry's policy: with the patient's care consent).
CREATE POLICY waitlist_offer_clinic_insert ON waitlist_offer FOR INSERT
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM waitlist_entry e WHERE e.id = waitlist_offer.entry_id AND e.status = 'active'));

GRANT SELECT, INSERT ON waitlist_offer TO salus_app;
GRANT UPDATE (state, closed_at, appointment_id) ON waitlist_offer TO salus_app;
-- Retention: deleted with its entry (the cascade, or the purger of the entry's rule).
GRANT SELECT, DELETE ON waitlist_offer TO salus_retention;
