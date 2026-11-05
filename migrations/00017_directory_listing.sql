-- +goose Up
-- Directory additions (docs/booking, package 4): a clinic chooses to be listed and has an IANA time zone
-- (validated by the service with time.LoadLocation), and a practitioner role has a specialty, a SNOMED CT code
-- from the R4 c80-practice-codes value set (the list and its displays are held in practitionerrole; the
-- system is always http://snomed.info/sct).
-- The directory scope (app.directory = 'public') is for anonymous and patient reads of listed clinics:
-- SELECT-only policies show listed clinics and the active practitioners and roles of listed clinics. The
-- services read only the public columns in that scope (name, time zone, specialty). NULLIF: an unset setting
-- reads as '' on a pooled connection and must match nothing.

ALTER TABLE organization
    ADD COLUMN listed boolean DEFAULT false NOT NULL,
    ADD COLUMN time_zone text DEFAULT 'Europe/Sarajevo' NOT NULL CHECK (btrim(time_zone) <> '');

ALTER TABLE practitioner_role ADD COLUMN specialty_code text;

-- Organization?listed=true
CREATE INDEX organization_listed_idx ON organization (id) WHERE listed;

-- The clinic's admin sets the listing and the zone (Organization update) and a role's specialty.
GRANT UPDATE (listed, time_zone) ON organization TO salus_app;
GRANT UPDATE (specialty_code, updated_at, version_id) ON practitioner_role TO salus_app;

CREATE POLICY organization_directory ON organization FOR SELECT
    USING (listed AND NULLIF(current_setting('app.directory', true), '') = 'public');

CREATE POLICY practitioner_directory ON practitioner FOR SELECT
    USING (active
           AND NULLIF(current_setting('app.directory', true), '') = 'public'
           AND EXISTS (SELECT 1 FROM organization o WHERE o.id = practitioner.organization_id AND o.listed));

CREATE POLICY practitioner_role_directory ON practitioner_role FOR SELECT
    USING (active
           AND NULLIF(current_setting('app.directory', true), '') = 'public'
           AND EXISTS (SELECT 1 FROM organization o WHERE o.id = practitioner_role.organization_id AND o.listed));
