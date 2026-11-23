-- +goose Up
-- Package O1 of docs/clinic-cms: a clinic signs itself up (W3). On a verified signup the API creates the clinic, as
-- salusctl create-clinic does, through the same service path. salus_app may insert a clinic only in that clinic's own
-- scope (organization_clinic's check: the id must be app.organization_id), so it can create one clinic at a time and
-- never write into another. It is created unlisted (listed defaults to false); the operator lists it.
GRANT INSERT (id, name) ON organization TO salus_app;

-- The setup wizard's progress: one row per step that is done or skipped; a step without a row is open. Nothing in it is
-- personal data. Kept with the clinic.
CREATE TABLE clinic_setup_step (
    organization_id uuid NOT NULL REFERENCES organization (id),
    step text NOT NULL CHECK (step IN ('details', 'billing', 'hours', 'practitioner', 'services')),
    state text NOT NULL CHECK (state IN ('done', 'skipped')),
    updated_at timestamptz DEFAULT now() NOT NULL,
    PRIMARY KEY (organization_id, step)
);
ALTER TABLE clinic_setup_step ENABLE ROW LEVEL SECURITY;
CREATE POLICY clinic_setup_step_clinic ON clinic_setup_step
    USING (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
GRANT SELECT, INSERT, DELETE ON clinic_setup_step TO salus_app;
GRANT UPDATE (state, updated_at) ON clinic_setup_step TO salus_app;

-- The retention job deletes unverified signups (internal/retention, "clinic_signup"); a verified one stays with its
-- clinic. clinic_signup has no row-level security: a signup belongs to no clinic yet.
GRANT SELECT, DELETE ON clinic_signup TO salus_retention;

-- A clinic that signed itself up is listed only once the operator approves it (W3): its admin may not list it through
-- the clinic settings (salus_app has UPDATE on listed, 00017) while its signup waits. salusctl signup approve sets
-- approved_at first, then lists it. A clinic made by salusctl create-clinic has no signup and is unaffected.
-- +goose StatementBegin
CREATE FUNCTION organization_guard_signup_listed() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.listed AND NOT OLD.listed
       AND EXISTS (SELECT 1 FROM clinic_signup WHERE organization_id = NEW.id AND approved_at IS NULL) THEN
        RAISE EXCEPTION 'a signed-up clinic is listed by the operator' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER organization_guard_signup_listed BEFORE UPDATE OF listed ON organization
    FOR EACH ROW EXECUTE FUNCTION organization_guard_signup_listed();
