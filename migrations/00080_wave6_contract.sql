-- +goose Up
-- The wave 6 contract (docs/clinic-cms/WAVE6.md, package 6.0, U1 to U6): the tables and columns the second half's
-- packages build on. No behaviour beyond what the database itself must hold. The number is main's next free one; the
-- integration renumbers.
--   - location (U1): a clinic's sites, each with a name, address, phone and time zone (FHIR Location). Not personal
--     data. Today's single address becomes each clinic's first location. A video visit has no location.
--   - room, device (U3, U4): what a location lists, optional (a FHIR Location that is partOf its site, a FHIR Device).
--     Not personal data. Two in-person slots in one room, and two visits that need one device, never overlap.
--   - The location (and room) on a slot and a visit (U2); the device a price-list service requires and a visit holds
--     (U4). The location of an in-person rule is a key of the rule in schedule.availability (`location_id`, `room_id`),
--     as its mode is, and is set by LC1 and RM1.
--   - diagnostic_report, diagnostic_report_result (U5): a lab result attached to a visit (FHIR DiagnosticReport and its
--     Observation results). Health data (Art. 9). The patient sees a report only once a practitioner released it; a
--     released report never changes.
--   - care_plan, care_plan_activity (U6): a treatment plan for a patient, its ordered steps (FHIR CarePlan and its
--     activity). Health data (Art. 9). Written for a patient with the clinic's active care consent, by its own staff.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

-- U1: locations.
CREATE TABLE location (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    name text NOT NULL CHECK (btrim(name) <> ''),
    address_line text CHECK (address_line <> ''),
    postal_code text CHECK (postal_code <> ''),
    city_id text REFERENCES place (id),
    phone text CHECK (phone ~ '^\+[0-9]{8,15}$'),
    time_zone text DEFAULT 'Europe/Sarajevo' NOT NULL CHECK (btrim(time_zone) <> ''),
    active boolean DEFAULT true NOT NULL,
    version_id integer DEFAULT 1 NOT NULL,
    -- For the references below, so a slot, visit, room or device is always in its location's clinic.
    CONSTRAINT location_id_organization_key UNIQUE (id, organization_id)
);
CREATE INDEX location_organization ON location (organization_id);

ALTER TABLE location ENABLE ROW LEVEL SECURITY;
CREATE POLICY location_clinic ON location
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- The patient picks a location when booking in person: the directory sees the active locations of listed clinics.
CREATE POLICY location_directory ON location FOR SELECT
    USING (active
           AND NULLIF(current_setting('app.directory', true), '') = 'public'
           AND EXISTS (SELECT 1 FROM organization o WHERE o.id = location.organization_id AND o.listed));
-- Never deleted: a location no longer used is made inactive; past visits keep pointing at it.
GRANT SELECT, INSERT ON location TO salus_app;
GRANT UPDATE (name, address_line, postal_code, city_id, phone, time_zone, active, updated_at, version_id) ON location TO salus_app;

-- The first location of every clinic that exists: its name and today's address.
INSERT INTO location (organization_id, name, address_line, postal_code, city_id, time_zone)
SELECT id, name, address_line, postal_code, city_id, time_zone FROM organization;

-- U3, U4: rooms and devices of a location.
CREATE TABLE room (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL,
    location_id uuid NOT NULL,
    name text NOT NULL CHECK (btrim(name) <> ''),
    active boolean DEFAULT true NOT NULL,
    version_id integer DEFAULT 1 NOT NULL,
    CONSTRAINT room_location_fkey FOREIGN KEY (location_id, organization_id) REFERENCES location (id, organization_id),
    CONSTRAINT room_id_location_key UNIQUE (id, location_id)
);
CREATE TABLE device (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL,
    location_id uuid NOT NULL,
    name text NOT NULL CHECK (btrim(name) <> ''),
    active boolean DEFAULT true NOT NULL,
    version_id integer DEFAULT 1 NOT NULL,
    CONSTRAINT device_location_fkey FOREIGN KEY (location_id, organization_id) REFERENCES location (id, organization_id),
    CONSTRAINT device_id_organization_key UNIQUE (id, organization_id)
);
CREATE INDEX room_location ON room (location_id);
CREATE INDEX device_location ON device (location_id);

ALTER TABLE room ENABLE ROW LEVEL SECURITY;
CREATE POLICY room_clinic ON room
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
ALTER TABLE device ENABLE ROW LEVEL SECURITY;
CREATE POLICY device_clinic ON device
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- Never deleted (made inactive), never moved to another location.
GRANT SELECT, INSERT ON room, device TO salus_app;
GRANT UPDATE (name, active, updated_at, version_id) ON room, device TO salus_app;

-- U4: the device a service requires; none for most.
ALTER TABLE charge_item_definition ADD COLUMN device_id uuid,
    ADD CONSTRAINT charge_item_definition_device_fkey FOREIGN KEY (device_id, organization_id) REFERENCES device (id, organization_id);
GRANT UPDATE (device_id) ON charge_item_definition TO salus_app;

-- U2, U3: the location and room of an in-person slot, copied from its rule; a video slot has neither. Two in-person
-- slots in one room never overlap, whoever's they are.
ALTER TABLE slot ADD COLUMN location_id uuid, ADD COLUMN room_id uuid,
    ADD CONSTRAINT slot_location_fkey FOREIGN KEY (location_id, organization_id) REFERENCES location (id, organization_id),
    ADD CONSTRAINT slot_room_fkey FOREIGN KEY (room_id, location_id) REFERENCES room (id, location_id),
    ADD CONSTRAINT slot_location_in_person CHECK (visit_mode = 'in-person' OR location_id IS NULL),
    ADD CONSTRAINT slot_room_location CHECK (room_id IS NULL OR location_id IS NOT NULL),
    ADD CONSTRAINT slot_room_no_overlap EXCLUDE USING gist (room_id WITH =, period WITH &&) WHERE (room_id IS NOT NULL);

-- U2 to U4: a visit's location and room, copied from its slot, and the device its service requires. Two visits that
-- need one device never overlap.
ALTER TABLE appointment ADD COLUMN location_id uuid, ADD COLUMN room_id uuid, ADD COLUMN device_id uuid,
    ADD CONSTRAINT appointment_location_fkey FOREIGN KEY (location_id, organization_id) REFERENCES location (id, organization_id),
    ADD CONSTRAINT appointment_room_fkey FOREIGN KEY (room_id, location_id) REFERENCES room (id, location_id),
    ADD CONSTRAINT appointment_device_fkey FOREIGN KEY (device_id, organization_id) REFERENCES device (id, organization_id),
    ADD CONSTRAINT appointment_location_in_person CHECK (visit_mode = 'in-person' OR (location_id IS NULL AND device_id IS NULL)),
    ADD CONSTRAINT appointment_room_location CHECK (room_id IS NULL OR location_id IS NOT NULL),
    ADD CONSTRAINT appointment_device_no_overlap EXCLUDE USING gist (device_id WITH =, period WITH &&)
        WHERE (device_id IS NOT NULL AND status <> 'cancelled');
-- A visit moved to another slot takes that slot's location and room (M1's move, LC1).
GRANT UPDATE (location_id, room_id, device_id) ON appointment TO salus_app;
CREATE INDEX appointment_organization_location ON appointment (organization_id, location_id) WHERE location_id IS NOT NULL;
-- A patient sees the address of their own visit, whatever became of the location since (appointment's policy holds the scope).
CREATE POLICY location_own ON location FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL
           AND EXISTS (SELECT 1 FROM appointment a WHERE a.location_id = location.id));

-- Every in-person rule, slot and visit that exists is at its clinic's first location.
UPDATE schedule s
SET availability = (SELECT coalesce(jsonb_agg(CASE WHEN r->>'mode' = 'in-person'
                                                   THEN r || jsonb_build_object('location_id', l.id) ELSE r END ORDER BY n), '[]')
                    FROM jsonb_array_elements(s.availability) WITH ORDINALITY AS e (r, n))
FROM location l
WHERE l.organization_id = s.organization_id
  AND s.availability @> '[{"mode": "in-person"}]';
UPDATE slot SET location_id = l.id FROM location l WHERE l.organization_id = slot.organization_id AND slot.visit_mode = 'in-person';
UPDATE appointment SET location_id = l.id
FROM location l WHERE l.organization_id = appointment.organization_id AND appointment.visit_mode = 'in-person';

-- U5: lab results on a visit. Staff of the visit's clinic attach one (a PDF, values, or both) for a patient with its
-- active care consent; a practitioner releases it (`final`), and only then does the patient see it. The PDF is a
-- Binary whose storage LB1 decides.
CREATE TABLE diagnostic_report (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    patient_id uuid NOT NULL REFERENCES patient (id),
    appointment_id uuid NOT NULL,
    created_by_account_id uuid NOT NULL REFERENCES account (id),
    -- http://hl7.org/fhir/diagnostic-report-status: attached and not yet released, or released.
    status text DEFAULT 'preliminary' NOT NULL CHECK (status IN ('preliminary', 'final')),
    title text NOT NULL CHECK (btrim(title) <> '' AND char_length(title) <= 200), -- DiagnosticReport.code.text, as the lab names it
    loinc_code text CHECK (loinc_code ~ '^[0-9]{1,7}-[0-9]$'), -- DiagnosticReport.code.coding, http://loinc.org; absent for none
    effective_date date, -- when the sample was taken, as the lab says; absent if it does not
    released_at timestamptz,
    released_by_account_id uuid REFERENCES account (id),
    version_id integer DEFAULT 1 NOT NULL,
    CONSTRAINT diagnostic_report_released CHECK ((status = 'final') = (released_at IS NOT NULL)
                                                 AND (status = 'final') = (released_by_account_id IS NOT NULL)),
    CONSTRAINT diagnostic_report_appointment_fkey FOREIGN KEY (appointment_id, organization_id, patient_id)
        REFERENCES appointment (id, organization_id, patient_id),
    CONSTRAINT diagnostic_report_id_organization_key UNIQUE (id, organization_id)
);
CREATE INDEX diagnostic_report_appointment ON diagnostic_report (appointment_id);
CREATE INDEX diagnostic_report_patient ON diagnostic_report (patient_id, organization_id);

-- One typed-in value of a report (an Observation in DiagnosticReport.result): the test, its value and UCUM unit or a
-- text value, the reference range as printed, and the interpretation flag.
CREATE TABLE diagnostic_report_result (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    diagnostic_report_id uuid NOT NULL REFERENCES diagnostic_report (id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position > 0),
    test text NOT NULL CHECK (btrim(test) <> '' AND char_length(test) <= 200), -- Observation.code.text
    loinc_code text CHECK (loinc_code ~ '^[0-9]{1,7}-[0-9]$'), -- Observation.code.coding, http://loinc.org
    value_decimal numeric, -- Observation.valueQuantity.value
    unit text CHECK (unit <> '' AND char_length(unit) <= 40), -- Observation.valueQuantity.code, UCUM (http://unitsofmeasure.org)
    value_text text CHECK (value_text <> '' AND char_length(value_text) <= 200), -- Observation.valueString ("negative")
    reference_range text CHECK (reference_range <> '' AND char_length(reference_range) <= 100), -- referenceRange.text
    -- http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation: normal, low, high, critically low or
    -- high, abnormal; absent for none.
    interpretation text CHECK (interpretation IN ('N', 'L', 'H', 'LL', 'HH', 'A')),
    CONSTRAINT diagnostic_report_result_value CHECK ((value_decimal IS NULL) <> (value_text IS NULL)
                                                     AND (unit IS NULL OR value_decimal IS NOT NULL)),
    CONSTRAINT diagnostic_report_result_position_key UNIQUE (diagnostic_report_id, position)
);

ALTER TABLE diagnostic_report ENABLE ROW LEVEL SECURITY;
-- The patient sees their released reports only.
CREATE POLICY diagnostic_report_own ON diagnostic_report FOR SELECT
    USING (status = 'final' AND patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
-- The clinic's scope: its reports, written only for a visit's patient with its active care consent, by its own staff.
CREATE POLICY diagnostic_report_clinic ON diagnostic_report
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM account a
                            WHERE a.id = diagnostic_report.created_by_account_id AND a.organization_id = diagnostic_report.organization_id
                              AND a.role IN ('admin', 'practitioner'))
                AND (diagnostic_report.released_by_account_id IS NULL
                     OR EXISTS (SELECT 1 FROM account a
                                WHERE a.id = diagnostic_report.released_by_account_id
                                  AND a.organization_id = diagnostic_report.organization_id
                                  AND a.role = 'practitioner')));
-- Written only for a patient with the clinic's active care consent; once written, its release goes on after a revocation.
CREATE POLICY diagnostic_report_consent ON diagnostic_report AS RESTRICTIVE FOR INSERT
    WITH CHECK (EXISTS (SELECT 1 FROM consent c
                        WHERE c.patient_id = diagnostic_report.patient_id
                          AND c.organization_id = diagnostic_report.organization_id
                          AND c.status = 'active'));
-- Never deleted (a mistaken report is LB1's: entered-in-error is not in the contract); released once.
GRANT SELECT, INSERT ON diagnostic_report TO salus_app;
GRANT UPDATE (status, title, loinc_code, effective_date, released_at, released_by_account_id, updated_at, version_id)
    ON diagnostic_report TO salus_app;

ALTER TABLE diagnostic_report_result ENABLE ROW LEVEL SECURITY;
CREATE POLICY diagnostic_report_result_seen ON diagnostic_report_result FOR SELECT
    USING (EXISTS (SELECT 1 FROM diagnostic_report r WHERE r.id = diagnostic_report_result.diagnostic_report_id));
-- Written in the clinic's scope of their report, while it is not released.
CREATE POLICY diagnostic_report_result_write ON diagnostic_report_result FOR ALL
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM diagnostic_report r
                       WHERE r.id = diagnostic_report_result.diagnostic_report_id
                         AND r.status = 'preliminary'
                         AND r.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid))
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND EXISTS (SELECT 1 FROM diagnostic_report r
                            WHERE r.id = diagnostic_report_result.diagnostic_report_id
                              AND r.status = 'preliminary'
                              AND r.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid));
GRANT SELECT, INSERT, DELETE ON diagnostic_report_result TO salus_app;
GRANT UPDATE (position, test, loinc_code, value_decimal, unit, value_text, reference_range, interpretation)
    ON diagnostic_report_result TO salus_app;

-- A report is inserted preliminary, released once, and never changes after, for every role, the owner too.
-- +goose StatementBegin
CREATE FUNCTION diagnostic_report_guard() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog, public
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'preliminary' THEN
            RAISE EXCEPTION 'a report is attached before it is released' USING ERRCODE = 'check_violation';
        END IF;
    ELSIF OLD.status = 'final' THEN
        RAISE EXCEPTION 'a released report never changes' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER diagnostic_report_guard BEFORE INSERT OR UPDATE ON diagnostic_report
    FOR EACH ROW EXECUTE FUNCTION diagnostic_report_guard();

-- U6: treatment plans. A plan is drafted, then booked (`active`: every step has its visit, booked in one
-- transaction), and ends `completed` or `revoked` (cancelled: its future steps' visits are cancelled). It carries
-- its quote (X1).
CREATE TABLE care_plan (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    patient_id uuid NOT NULL REFERENCES patient (id),
    created_by_account_id uuid NOT NULL REFERENCES account (id),
    -- http://hl7.org/fhir/request-status, the four a plan here takes.
    status text DEFAULT 'draft' NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'revoked')),
    title text NOT NULL CHECK (btrim(title) <> '' AND char_length(title) <= 200),
    quote_id uuid REFERENCES quote (id),
    starts_on date, -- day 0 of the steps' offsets; set when booked
    booked_at timestamptz,
    revoked_at timestamptz,
    revoked_by_account_id uuid REFERENCES account (id),
    version_id integer DEFAULT 1 NOT NULL,
    -- A draft is not booked, a booked or completed plan is; a revoked one may have been either.
    CONSTRAINT care_plan_booked CHECK ((booked_at IS NULL) = (starts_on IS NULL)
                                       AND (status <> 'draft' OR booked_at IS NULL)
                                       AND (status NOT IN ('active', 'completed') OR booked_at IS NOT NULL)),
    CONSTRAINT care_plan_revoked CHECK ((status = 'revoked') = (revoked_at IS NOT NULL)
                                       AND (revoked_by_account_id IS NULL OR revoked_at IS NOT NULL))
);
CREATE INDEX care_plan_patient ON care_plan (patient_id, organization_id);
CREATE INDEX care_plan_organization ON care_plan (organization_id, created_at);

-- A step: a service of the clinic's price list, a day offset from the plan's start and a visit mode; its visit once
-- booked. Positions are unique at the end of a transaction, so moving a step can renumber the others.
CREATE TABLE care_plan_activity (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    care_plan_id uuid NOT NULL REFERENCES care_plan (id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position > 0),
    charge_item_definition_id uuid NOT NULL REFERENCES charge_item_definition (id),
    day_offset integer NOT NULL CHECK (day_offset BETWEEN 0 AND 730),
    visit_mode text NOT NULL CHECK (visit_mode IN ('in-person', 'video')),
    appointment_id uuid REFERENCES appointment (id),
    CONSTRAINT care_plan_activity_position_key UNIQUE (care_plan_id, position) DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT care_plan_activity_appointment_key UNIQUE (appointment_id)
);

ALTER TABLE care_plan ENABLE ROW LEVEL SECURITY;
-- The patient sees their plans once booked; a draft, and a draft revoked before it was booked, are the clinic's.
CREATE POLICY care_plan_own ON care_plan FOR SELECT
    USING (booked_at IS NOT NULL AND patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY care_plan_clinic ON care_plan
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM account a
                            WHERE a.id = care_plan.created_by_account_id AND a.organization_id = care_plan.organization_id
                              AND a.role IN ('admin', 'practitioner'))
                AND (care_plan.revoked_by_account_id IS NULL
                     OR EXISTS (SELECT 1 FROM account a
                                WHERE a.id = care_plan.revoked_by_account_id AND a.organization_id = care_plan.organization_id))
                AND (care_plan.quote_id IS NULL
                     OR EXISTS (SELECT 1 FROM quote q
                                WHERE q.id = care_plan.quote_id
                                  AND q.patient_id = care_plan.patient_id
                                  AND q.organization_id = care_plan.organization_id)));
-- Written only for a patient with the clinic's active care consent; once written, its cancellation goes on after a revocation.
CREATE POLICY care_plan_consent ON care_plan AS RESTRICTIVE FOR INSERT
    WITH CHECK (EXISTS (SELECT 1 FROM consent c
                        WHERE c.patient_id = care_plan.patient_id
                          AND c.organization_id = care_plan.organization_id
                          AND c.status = 'active'));
-- Never deleted: a plan that is not wanted is revoked.
GRANT SELECT, INSERT ON care_plan TO salus_app;
GRANT UPDATE (status, title, quote_id, starts_on, booked_at, revoked_at, revoked_by_account_id, updated_at, version_id)
    ON care_plan TO salus_app;

ALTER TABLE care_plan_activity ENABLE ROW LEVEL SECURITY;
CREATE POLICY care_plan_activity_seen ON care_plan_activity FOR SELECT
    USING (EXISTS (SELECT 1 FROM care_plan p WHERE p.id = care_plan_activity.care_plan_id));
-- Written in the clinic's scope of its plan, from the plan's clinic's price list, to a visit of the plan's patient
-- at that clinic.
CREATE POLICY care_plan_activity_write ON care_plan_activity FOR ALL
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM care_plan p
                       WHERE p.id = care_plan_activity.care_plan_id
                         AND p.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid))
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND EXISTS (SELECT 1 FROM care_plan p
                            JOIN charge_item_definition d ON d.id = care_plan_activity.charge_item_definition_id
                                                         AND d.organization_id = p.organization_id
                            WHERE p.id = care_plan_activity.care_plan_id
                              AND p.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
                AND (care_plan_activity.appointment_id IS NULL
                     OR EXISTS (SELECT 1 FROM appointment a JOIN care_plan p ON p.id = care_plan_activity.care_plan_id
                                WHERE a.id = care_plan_activity.appointment_id
                                  AND a.patient_id = p.patient_id
                                  AND a.organization_id = p.organization_id)));
-- A draft's steps are deleted and rewritten as it is laid out; a booked plan's are kept (TP1).
GRANT SELECT, INSERT, DELETE ON care_plan_activity TO salus_app;
GRANT UPDATE (position, charge_item_definition_id, day_offset, visit_mode, appointment_id) ON care_plan_activity TO salus_app;
