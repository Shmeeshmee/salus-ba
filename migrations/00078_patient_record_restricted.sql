-- +goose Up
-- Whether a patient's record is restricted at a clinic (docs/clinic-cms/WAVE5.md S7), the same from any scope: the
-- worker's jobs (chat notices, suggestions, reminders) run with no clinic set, and a practitioner of another clinic the
-- patient invited into the chat reads in their own clinic's scope, where patient_restriction shows none of the
-- treating clinic's rows. The function answers yes or no only, as its owner (SECURITY DEFINER, its search_path fixed);
-- nothing of the restriction itself (who, when, which request) leaves the clinic's scope.
-- +goose StatementBegin
CREATE FUNCTION patient_record_restricted(patient uuid, organization uuid) RETURNS boolean
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
    SELECT EXISTS (SELECT 1 FROM patient_restriction r WHERE r.patient_id = patient AND r.organization_id = organization)
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION patient_record_restricted(uuid, uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION patient_record_restricted(uuid, uuid) TO salus_app;
