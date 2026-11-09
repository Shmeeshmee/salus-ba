-- +goose Up
-- The clinic of an encounter, for the consultation's stream consumer (SAL-6, docs/plans/consult-intelligence.md):
-- it reads the encounter's Redis Stream, whose entries name the encounter only, and stores what is said into
-- transcript_segment and suggestion in the clinic's scope, where row-level security (the consent and the
-- after-revocation rule of encounter_clinic) still decides whether the encounter is seen. Like
-- invite_organization (00013): one SECURITY DEFINER function returns only the clinic id of an encounter id,
-- owned by salus_encounter_lookup, a role that can read the two columns it needs and nothing else. No row,
-- no patient and no other column leaves the function.

-- The owner of the lookup function: NOLOGIN, not a superuser, no BYPASSRLS. Cluster-wide like salus_app, so
-- the same DO block (test packages migrate in parallel).
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE salus_encounter_lookup NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEROLE NOCREATEDB;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'salus_encounter_lookup'
               AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolcanlogin)) THEN
        RAISE EXCEPTION 'role salus_encounter_lookup exists with SUPERUSER, BYPASSRLS, CREATEROLE, CREATEDB or LOGIN';
    END IF;
END
$$;
-- +goose StatementEnd

GRANT SELECT (id, organization_id) ON encounter TO salus_encounter_lookup;
CREATE POLICY encounter_lookup ON encounter FOR SELECT TO salus_encounter_lookup USING (true);

-- The clinic of the encounter with the id, or null. Nothing else about the encounter leaves the function; the
-- service then sets the clinic and reads the encounter through row-level security.
CREATE FUNCTION encounter_organization(encounter_id uuid) RETURNS uuid
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
    SELECT e.organization_id FROM public.encounter AS e WHERE e.id = encounter_organization.encounter_id
$$;

ALTER FUNCTION encounter_organization(uuid) OWNER TO salus_encounter_lookup;
REVOKE EXECUTE ON FUNCTION encounter_organization(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION encounter_organization(uuid) TO salus_app;
