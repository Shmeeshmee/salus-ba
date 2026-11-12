-- +goose Up
-- The questions of a shared questionnaire response (SAL-40, docs/plans/patient-history.md, ADR 0014): 00043 opens
-- another clinic's completed response to the receiving clinic's scope, but not the questionnaire version it
-- answers, which the practitioner needs to read the answers. A policy on questionnaire chained on
-- questionnaire_response made the planner spin, so, like encounter_organization (00033), one SECURITY DEFINER
-- function returns a version only when a completed response to it, of a finished visit shared with the scope's
-- clinic (history_shared), exists: nothing else of the other clinic's questionnaires, and no response, leaves it. It
-- is owned by salus_history_lookup, a role that reads only the questionnaire and the columns the check needs. The
-- version's translations stay closed: the receiving clinic reads the questions in their original language.

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
GRANT EXECUTE ON FUNCTION history_shared(uuid, uuid) TO salus_history_lookup;
CREATE POLICY questionnaire_history_lookup ON questionnaire FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY questionnaire_response_history_lookup ON questionnaire_response FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY encounter_history_lookup ON encounter FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY history_consent_encounter_history_lookup ON history_consent_encounter FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY history_consent_history_lookup ON history_consent FOR SELECT TO salus_history_lookup USING (true);
CREATE POLICY consent_history_lookup ON consent FOR SELECT TO salus_history_lookup USING (true);

-- The version questionnaire_id when a completed response to it of a finished visit is shared with the clinic of the
-- transaction's scope; no row otherwise.
CREATE FUNCTION history_questionnaire(questionnaire_id uuid) RETURNS SETOF questionnaire
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
    SELECT q.* FROM public.questionnaire AS q
    WHERE q.id = history_questionnaire.questionnaire_id
      AND EXISTS (SELECT 1 FROM public.questionnaire_response AS r
                  JOIN public.encounter AS e ON e.appointment_id = r.appointment_id AND e.patient_id = r.patient_id
                  WHERE r.questionnaire_id = q.id AND r.status = 'completed' AND e.status = 'finished'
                    AND public.history_shared(e.id, e.patient_id))
$$;

ALTER FUNCTION history_questionnaire(uuid) OWNER TO salus_history_lookup;
REVOKE EXECUTE ON FUNCTION history_questionnaire(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION history_questionnaire(uuid) TO salus_app;
