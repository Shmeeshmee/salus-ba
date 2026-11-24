-- +goose Up
-- Who the anonymised analytics leave out (docs/clinic-cms/WAVE5.md D4): a patient who objected to them (S9,
-- patient.analytics_objected_at) at every clinic, and a patient whose record is restricted at a clinic (S7) at that
-- clinic. The aggregate builds (salus_stats) still read no patient table: this function answers the one question, as
-- its owner (SECURITY DEFINER, its search_path fixed), for a patient id the builds already read on the source row.
-- Existing aggregates are not recomputed beyond the builds' own window.
-- +goose StatementBegin
CREATE FUNCTION analytics_left_out(patient uuid, organization uuid) RETURNS boolean
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
    SELECT EXISTS (SELECT 1 FROM patient p WHERE p.id = patient AND p.analytics_objected_at IS NOT NULL)
        OR EXISTS (SELECT 1 FROM patient_restriction r WHERE r.patient_id = patient AND r.organization_id = organization)
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION analytics_left_out(uuid, uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION analytics_left_out(uuid, uuid) TO salus_stats;
