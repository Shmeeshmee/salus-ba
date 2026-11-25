-- +goose Up
-- The patient a data request names, for the clinic the request targets (docs/clinic-cms/walk/wave5 finding 3): its
-- admin answers the request and must know whose it is, whatever the patient's consent or restriction there (an app
-- patient who revoked their consent and never had a visit is outside every other clinic branch of patient). The clinic
-- scope (app.organization_id, neither patient setting set) with app.data_requests = 'on' reads the patients its own
-- requests name, read-only. Set only by patient.Service.RequestSubject, for the clinic's admin, around one audited read,
-- and set off again after it. NULLIF and the comparison with 'on': an unset setting reads as '' on a pooled connection
-- and must match nothing.
--
-- The policy asks a function, not data_request itself: data_request's own policies read patient, and a policy on
-- patient reading data_request would be a cycle. The function answers yes or no only, as its owner (SECURITY DEFINER,
-- its search_path fixed), as patient_record_restricted (00078) does.
-- +goose StatementBegin
CREATE FUNCTION data_request_names_patient(patient uuid, organization uuid) RETURNS boolean
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
    SELECT EXISTS (SELECT 1 FROM data_request r WHERE r.patient_id = patient AND r.organization_id = organization)
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION data_request_names_patient(uuid, uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION data_request_names_patient(uuid, uuid) TO salus_app;

CREATE POLICY patient_data_request ON patient FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND current_setting('app.data_requests', true) = 'on'
           AND data_request_names_patient(patient.id, NULLIF(current_setting('app.organization_id', true), '')::uuid));
