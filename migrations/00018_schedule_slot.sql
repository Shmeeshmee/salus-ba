-- +goose Up
-- Schedules and slots (docs/booking, package 5). A schedule is a practitioner role's weekly availability
-- (FHIR Schedule; SPEC.md "Schedule shape"): rules {day, start, end} in the clinic's time zone, validated by
-- the service, and a slot length. Slots (FHIR Slot) are generated from it 28 days ahead by the scheduler's
-- slots_generate job (SPEC.md "Slot generation"); a slot is free or busy, and busy only in the same
-- transaction as the appointment that holds it (package 6).
-- Row-level secured in two scopes: the clinic (app.organization_id) reads and writes its own rows; the
-- directory (app.directory = 'public') only reads the schedules of active roles of listed clinics and their
-- free slots. NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

-- A schedule names its role with the role's clinic, so both are always the same clinic.
ALTER TABLE practitioner_role ADD CONSTRAINT practitioner_role_id_organization_key UNIQUE (id, organization_id);

CREATE TABLE schedule (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL,
    practitioner_role_id uuid NOT NULL,
    version_id integer DEFAULT 1 NOT NULL,
    slot_minutes integer NOT NULL CHECK (slot_minutes BETWEEN 5 AND 240 AND slot_minutes % 5 = 0),
    availability jsonb DEFAULT '[]' NOT NULL CHECK (jsonb_typeof(availability) = 'array'),
    CONSTRAINT schedule_practitioner_role_id_key UNIQUE (practitioner_role_id),
    -- For slot's reference, so a slot is always in its schedule's clinic.
    CONSTRAINT schedule_id_organization_key UNIQUE (id, organization_id),
    CONSTRAINT schedule_practitioner_role_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id)
);

ALTER TABLE schedule ENABLE ROW LEVEL SECURITY;
CREATE POLICY schedule_clinic ON schedule
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- The role's policy shows only active roles of listed clinics in this scope.
CREATE POLICY schedule_directory ON schedule FOR SELECT
    USING (NULLIF(current_setting('app.directory', true), '') = 'public'
           AND EXISTS (SELECT 1 FROM practitioner_role r WHERE r.id = schedule.practitioner_role_id));

-- Created with the role's first availability edit; never deleted (retention restrict), the role never changes.
GRANT SELECT, INSERT ON schedule TO salus_app;
GRANT UPDATE (slot_minutes, availability, updated_at, version_id) ON schedule TO salus_app;

CREATE TABLE slot (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    schedule_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    version_id integer DEFAULT 1 NOT NULL,
    status text DEFAULT 'free' NOT NULL CHECK (status IN ('free', 'busy')),
    period tstzrange NOT NULL CHECK (NOT isempty(period) AND lower_inc(period) AND NOT upper_inc(period)
                                     AND NOT lower_inf(period) AND NOT upper_inf(period)),
    CONSTRAINT slot_schedule_fkey FOREIGN KEY (schedule_id, organization_id) REFERENCES schedule (id, organization_id),
    -- Generated slots never overlap, so generation inserts with ON CONFLICT DO NOTHING. The index also serves
    -- Slot?schedule=…&start=… and generation's lookups by schedule.
    CONSTRAINT slot_no_overlap EXCLUDE USING gist (schedule_id WITH =, period WITH &&)
);

-- The retention job: free slots whose end is past their period.
CREATE INDEX slot_free_end_idx ON slot (upper(period)) WHERE status = 'free';

ALTER TABLE slot ENABLE ROW LEVEL SECURITY;
CREATE POLICY slot_clinic ON slot
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- The schedule's policy shows only schedules of active roles of listed clinics in this scope; a busy slot is
-- not shown (it is nobody's business outside the clinic).
CREATE POLICY slot_directory ON slot FOR SELECT
    USING (status = 'free'
           AND NULLIF(current_setting('app.directory', true), '') = 'public'
           AND EXISTS (SELECT 1 FROM schedule s WHERE s.id = slot.schedule_id));
-- Whatever a query asks, the application deletes only free slots: a busy one stays with its appointment.
CREATE POLICY slot_delete_free_only ON slot AS RESTRICTIVE FOR DELETE
    USING (status = 'free');

GRANT SELECT, INSERT, DELETE ON slot TO salus_app;
GRANT UPDATE (status, updated_at, version_id) ON slot TO salus_app;
-- The retention rule of slot is delete (free slots 30 days after their end).
GRANT SELECT, DELETE ON slot TO salus_retention;
