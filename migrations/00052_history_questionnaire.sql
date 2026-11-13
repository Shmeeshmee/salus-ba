-- +goose Up
-- Two lookups for the practitioner's view of the patient's history (SAL-40, docs/plans/patient-history.md, ADR 0014),
-- each a SECURITY DEFINER function like encounter_organization (00033), not a chained policy (a policy on
-- questionnaire chained on questionnaire_response made the planner spin):
--   - history_questionnaire: the questions of a shared questionnaire response. 00043 opens another clinic's completed
--     response to the receiving clinic's scope, but not the version it answers. It returns a version only when a
--     completed response to it, of a finished visit shared with the scope's clinic (history_shared), exists: nothing
--     else of the other clinic's questionnaires, and no response, leaves it. Its translations stay closed: the
--     receiving clinic reads the questions in their original language.
--   - history_confirmed_conditions: which coded fields of a shared visit's signed documents have a confirmed
--     Condition (00042's condition_clinic shows a clinic only its own), so a code backfilled unconfirmed by 00042
--     never reads as a diagnosis. The document, the field key and the code, nothing else of the Condition.
-- Both answer only the practitioner given, and only when they have an appointment with the patient at the scope's
-- clinic (access.Profile's rule), so a colleague without one learns nothing. They are owned by salus_history_lookup,
-- a role that reads only the questionnaire and the columns the checks need.

-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE salus_history_lookup NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEROLE NOCREATEDB;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'salus_history_lookup'
               AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolcanlogin)) THEN
        RAISE EXCEPTION 'role salus_history_lookup exists with SUPERUSER, BYPASSRLS, CREATEROLE, CREATEDB or LOGIN';
    END IF;
END
$$;
-- +goose StatementEnd

GRANT SELECT ON questionnaire TO salus_history_lookup; -- the clinic's content, no personal data: the whole row is returned
GRANT SELECT (questionnaire_id, appointment_id, patient_id, status) ON questionnaire_response TO salus_history_lookup;
GRANT SELECT (id, appointment_id, patient_id, status) ON encounter TO salus_history_lookup;
GRANT SELECT (consent_id, encounter_id, removed_at) ON history_consent_encounter TO salus_history_lookup;
GRANT SELECT (id, patient_id, organization_id, status) ON history_consent TO salus_history_lookup;
GRANT SELECT (patient_id, organization_id, status) ON consent TO salus_history_lookup;
GRANT SELECT (patient_id, organization_id, practitioner_role_id) ON appointment TO salus_history_lookup;
GRANT SELECT (id, practitioner_id, organization_id) ON practitioner_role TO salus_history_lookup;
GRANT SELECT (encounter_id, patient_id, document_draft_id, field_key, code, verification_status) ON condition TO salus_history_lookup;
GRANT EXECUTE ON FUNCTION history_shared(uuid, uuid) TO salus_history_lookup;
CREATE POLICY questionnaire_history_lookup ON questionnaire FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY questionnaire_response_history_lookup ON questionnaire_response FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY encounter_history_lookup ON encounter FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY history_consent_encounter_history_lookup ON history_consent_encounter FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY history_consent_history_lookup ON history_consent FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY consent_history_lookup ON consent FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY appointment_history_lookup ON appointment FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY practitioner_role_history_lookup ON practitioner_role FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY condition_history_lookup ON condition FOR SELECT TO salus_history_lookup USING (true);

-- The versions of questionnaire_ids that a completed response of a finished visit shared with the clinic of the
-- transaction's scope answers, when practitioner_id has an appointment with that visit's patient at the clinic; no
-- row otherwise.
CREATE FUNCTION history_questionnaire(questionnaire_ids uuid[], practitioner_id uuid) RETURNS SETOF questionnaire
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
    SELECT q.* FROM public.questionnaire AS q
    WHERE q.id = ANY (history_questionnaire.questionnaire_ids)
      AND EXISTS (SELECT 1 FROM public.questionnaire_response AS r
                  JOIN public.encounter AS e ON e.appointment_id = r.appointment_id AND e.patient_id = r.patient_id
                  WHERE r.questionnaire_id = q.id AND r.status = 'completed' AND e.status = 'finished'
                    AND public.history_shared(e.id, e.patient_id)
                    AND EXISTS (SELECT 1 FROM public.appointment AS a
                                JOIN public.practitioner_role AS pr ON pr.id = a.practitioner_role_id
                                WHERE a.patient_id = e.patient_id AND pr.practitioner_id = history_questionnaire.practitioner_id
                                  AND a.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid))
$$;

-- The coded fields of the signed documents document_ids, of finished visits shared with the clinic of the
-- transaction's scope, whose Condition is confirmed, when practitioner_id has an appointment with the visit's
-- patient at the clinic; no row otherwise.
CREATE FUNCTION history_confirmed_conditions(document_ids uuid[], practitioner_id uuid)
    RETURNS TABLE (document_draft_id uuid, field_key text, code text)
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
    SELECT c.document_draft_id, c.field_key, c.code FROM public.condition AS c
    JOIN public.encounter AS e ON e.id = c.encounter_id AND e.patient_id = c.patient_id
    WHERE c.document_draft_id = ANY (history_confirmed_conditions.document_ids) AND c.verification_status = 'confirmed'
      AND e.status = 'finished' AND public.history_shared(e.id, e.patient_id)
      AND EXISTS (SELECT 1 FROM public.appointment AS a
                  JOIN public.practitioner_role AS pr ON pr.id = a.practitioner_role_id
                  WHERE a.patient_id = e.patient_id AND pr.practitioner_id = history_confirmed_conditions.practitioner_id
                    AND a.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
$$;

ALTER FUNCTION history_questionnaire(uuid[], uuid) OWNER TO salus_history_lookup;
REVOKE EXECUTE ON FUNCTION history_questionnaire(uuid[], uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION history_questionnaire(uuid[], uuid) TO salus_app;
ALTER FUNCTION history_confirmed_conditions(uuid[], uuid) OWNER TO salus_history_lookup;
REVOKE EXECUTE ON FUNCTION history_confirmed_conditions(uuid[], uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION history_confirmed_conditions(uuid[], uuid) TO salus_app;
