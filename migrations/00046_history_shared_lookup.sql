-- +goose Up
-- The shared visit's clinic and practitioner role (00043's organization_shared and practitioner_role_shared), looked
-- up once per query instead of once per row. 00043's EXISTS over history_consent_encounter and encounter was planned
-- as a subplan for every row of organization and practitioner_role a query reads, and with them every slot and
-- schedule of a directory read (slot_directory reads schedule, schedule_directory reads practitioner_role, whose
-- policy reads organization): each subplan pulled in encounter's, consent's and appointment's policies, about 1 360
-- subplans in the free-slot search's plan. On a directory of production size (22 000 slots) the search was planned
-- at cost 560 000 to 3 000 000, above jit_optimize_above_cost, and took 82 s of JIT compilation for a 7 ms read
-- (docs/plans/patient-history.PROGRESS.md, test TestHistorySharing_TheFreeSlotSearchStaysUnderJITsOptimizeCost).
--
-- Neither set depends on the row: it is the clinics and roles of the visits shared with the scope's clinic. Each is
-- now a function returning that set, called in an uncorrelated sub-select (an InitPlan: run at most once per query,
-- and only when the policy's other conditions hold) and opaque to the planner (a function with SET is never
-- inlined), so a query's plan holds one cheap parameter instead of the policy tree.
--
-- The access is exactly 00043's. The functions keep the invoker's rights (no SECURITY DEFINER): they read
-- history_consent_encounter and encounter as 00043's policies did, through those tables' policies in the same scope,
-- and keep history_shared's check per visit, so a row is visible exactly when 00043's EXISTS held for it. The scope
-- conditions in front are history_shared's own (no patient, no account, a clinic set): they only skip the lookup
-- where history_shared is false for every visit, which every directory and patient read is.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

-- The clinics of the visits shared with the scope's clinic.
CREATE FUNCTION history_shared_organizations() RETURNS uuid[]
    LANGUAGE sql
    STABLE
    SET search_path = pg_catalog, public
AS $$
    SELECT coalesce(array_agg(DISTINCT e.organization_id), '{}')
    FROM public.history_consent_encounter s
    JOIN public.encounter e ON e.id = s.encounter_id
    WHERE s.removed_at IS NULL
      AND public.history_shared(e.id, e.patient_id)
$$;
REVOKE EXECUTE ON FUNCTION history_shared_organizations() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION history_shared_organizations() TO salus_app;

-- The practitioner roles of the visits shared with the scope's clinic.
CREATE FUNCTION history_shared_practitioner_roles() RETURNS uuid[]
    LANGUAGE sql
    STABLE
    SET search_path = pg_catalog, public
AS $$
    SELECT coalesce(array_agg(DISTINCT e.practitioner_role_id), '{}')
    FROM public.history_consent_encounter s
    JOIN public.encounter e ON e.id = s.encounter_id
    WHERE s.removed_at IS NULL
      AND public.history_shared(e.id, e.patient_id)
$$;
REVOKE EXECUTE ON FUNCTION history_shared_practitioner_roles() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION history_shared_practitioner_roles() TO salus_app;

DROP POLICY organization_shared ON organization;
CREATE POLICY organization_shared ON organization FOR SELECT TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.organization_id', true), '') IS NOT NULL
           AND id = ANY ((SELECT history_shared_organizations())::uuid[]));

DROP POLICY practitioner_role_shared ON practitioner_role;
CREATE POLICY practitioner_role_shared ON practitioner_role FOR SELECT TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.organization_id', true), '') IS NOT NULL
           AND id = ANY ((SELECT history_shared_practitioner_roles())::uuid[]));
