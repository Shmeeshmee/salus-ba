-- +goose Up
-- Patient history (SAL-38, docs/plans/patient-history.md, ADR 0014): the patient's allergies and current
-- medications, and their consent to share chosen visits from other clinics with one clinic.
--
-- Allergies and medications (FHIR AllergyIntolerance, MedicationStatement). An entry is either the patient's own
-- (organization_id, encounter_id and recorder_role_id null, always in the profile) or recorded by a clinic in one of
-- its visits (all three set). The profile is what every clinic with the patient's active care consent sees; a
-- clinic's entry belongs to its visit and is seen wherever the visit is seen (the clinic's own scope, another
-- clinic's only while the visit is shared with it), until the patient adds it to their profile (in_profile). A
-- confirmation names the confirming role and its clinic for that clinic's record; the service never shows them to
-- another clinic (ADR 0014, "Doctor entries").
--
-- History consent (FHIR Consent, the third kind beside 00016's care consent and 00040's transcription consent): per
-- patient and clinic, off unless the patient grants it, listing the visits it shares. A table of its own, as 00040
-- says why: the policies that read consent as the care consent must never be opened by it. history_consent_encounter
-- holds one row per period a visit is shared: removing a visit sets removed_at, adding it again adds a row, so the
-- record of what was shared when is kept as proof (Art. 7(1)). Only a finished visit of the patient's at another
-- clinic can be listed, checked here as well as in the service.
--
-- Sharing opens, read-only, to the receiving clinic's scope (app.organization_id, neither patient setting set) while
-- the history consent is active, lists the visit, and the care consent with that clinic is active:
--   - the visit's encounter, its signed documents (final, never a draft), its completed questionnaire responses and
--     their clean files (never one in quarantine or rejected), and the entries recorded in it;
--   - the visit's clinic and practitioner role, for the clinic name and specialty the shared visit is marked with.
-- Never the transcript, the suggestions, the appointment or anything not listed: 00031's transcript_segment_clinic
-- and suggestion_clinic, which followed encounter's visibility, are recreated for the clinic's own encounters. Which
-- practitioner of the clinic sees it is the service's rule (access.HistoryVisit). A removal or revocation ends it,
-- and what was shared, when its transaction commits; both are final (adding again adds a row, granting again too).
-- What each scope may change of an entry is checked by profile_entry_update_check, below.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE allergy_intolerance (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    recorded_date timestamptz NOT NULL,
    confirmed_at timestamptz,
    patient_id uuid NOT NULL REFERENCES patient (id),
    organization_id uuid REFERENCES organization (id), -- the recording clinic; null for the patient's own
    encounter_id uuid REFERENCES encounter (id), -- the visit it was recorded in
    recorder_role_id uuid REFERENCES practitioner_role (id),
    confirmed_by_role_id uuid REFERENCES practitioner_role (id),
    confirmed_organization_id uuid REFERENCES organization (id),
    term_id uuid REFERENCES term (id), -- the substance from the terminology, when picked from it
    version_id integer DEFAULT 1 NOT NULL,
    in_profile boolean DEFAULT true NOT NULL,
    code_text text NOT NULL CHECK (code_text <> '' AND code_text = btrim(code_text) AND length(code_text) <= 200),
    category text CHECK (category IN ('food', 'medication', 'environment', 'biologic')),
    reaction_text text CHECK (reaction_text <> '' AND reaction_text = btrim(reaction_text) AND length(reaction_text) <= 200),
    severity text CHECK (severity IN ('mild', 'moderate', 'severe')),
    clinical_status text DEFAULT 'active' NOT NULL CHECK (clinical_status IN ('active', 'inactive', 'resolved')),
    verification_status text NOT NULL CHECK (verification_status IN ('unconfirmed', 'confirmed', 'entered-in-error')),
    CONSTRAINT allergy_intolerance_recorder_fkey FOREIGN KEY (recorder_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT allergy_intolerance_confirmer_fkey FOREIGN KEY (confirmed_by_role_id, confirmed_organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT allergy_intolerance_source CHECK (num_nulls(organization_id, encounter_id, recorder_role_id) IN (0, 3)),
    CONSTRAINT allergy_intolerance_own_in_profile CHECK (organization_id IS NOT NULL OR in_profile),
    CONSTRAINT allergy_intolerance_confirmation CHECK (num_nulls(confirmed_at, confirmed_by_role_id, confirmed_organization_id) IN (0, 3)),
    -- Confirmed exactly when it carries a confirmation; a removed entry keeps the one it had.
    CONSTRAINT allergy_intolerance_confirmed CHECK (verification_status = 'entered-in-error'
                                                    OR (verification_status = 'confirmed') = (confirmed_at IS NOT NULL)),
    -- FHIR: a reaction has at least one manifestation; a severity is a reaction's.
    CONSTRAINT allergy_intolerance_severity CHECK (severity IS NULL OR reaction_text IS NOT NULL)
);

CREATE TABLE medication_statement (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    date_asserted timestamptz NOT NULL,
    confirmed_at timestamptz,
    patient_id uuid NOT NULL REFERENCES patient (id),
    organization_id uuid REFERENCES organization (id),
    encounter_id uuid REFERENCES encounter (id),
    recorder_role_id uuid REFERENCES practitioner_role (id),
    confirmed_by_role_id uuid REFERENCES practitioner_role (id),
    confirmed_organization_id uuid REFERENCES organization (id),
    term_id uuid REFERENCES term (id),
    dose_value numeric CHECK (dose_value > 0 AND dose_value < 100000),
    period numeric CHECK (period > 0 AND period <= 1000),
    version_id integer DEFAULT 1 NOT NULL,
    frequency integer CHECK (frequency BETWEEN 1 AND 100),
    in_profile boolean DEFAULT true NOT NULL,
    as_needed boolean DEFAULT false NOT NULL,
    medication_text text NOT NULL CHECK (medication_text <> '' AND medication_text = btrim(medication_text) AND length(medication_text) <= 200),
    -- UCUM codes (http://unitsofmeasure.org): the units and the annotated counts a patient doses in.
    dose_unit text CHECK (dose_unit IN ('mg', 'g', 'ug', 'mL', '[iU]', '{tablet}', '{capsule}', '{drop}', '{puff}', '{patch}')),
    period_unit text CHECK (period_unit IN ('h', 'd', 'wk', 'mo')), -- UCUM, as FHIR Timing.repeat.periodUnit
    status text DEFAULT 'active' NOT NULL CHECK (status IN ('active', 'stopped', 'entered-in-error')),
    CONSTRAINT medication_statement_recorder_fkey FOREIGN KEY (recorder_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT medication_statement_confirmer_fkey FOREIGN KEY (confirmed_by_role_id, confirmed_organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT medication_statement_source CHECK (num_nulls(organization_id, encounter_id, recorder_role_id) IN (0, 3)),
    CONSTRAINT medication_statement_own_in_profile CHECK (organization_id IS NOT NULL OR in_profile),
    CONSTRAINT medication_statement_confirmation CHECK (num_nulls(confirmed_at, confirmed_by_role_id, confirmed_organization_id) IN (0, 3)),
    CONSTRAINT medication_statement_dose CHECK (num_nulls(dose_value, dose_unit) IN (0, 2)),
    CONSTRAINT medication_statement_timing CHECK (num_nulls(frequency, period, period_unit) IN (0, 3))
);

-- A patient's entries, and a visit's.
CREATE INDEX allergy_intolerance_patient ON allergy_intolerance (patient_id);
CREATE INDEX allergy_intolerance_encounter ON allergy_intolerance (encounter_id) WHERE encounter_id IS NOT NULL;
CREATE INDEX medication_statement_patient ON medication_statement (patient_id);
CREATE INDEX medication_statement_encounter ON medication_statement (encounter_id) WHERE encounter_id IS NOT NULL;

CREATE TABLE history_consent (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    date_time timestamptz NOT NULL, -- granted
    revoked_at timestamptz,
    patient_id uuid NOT NULL REFERENCES patient (id),
    organization_id uuid NOT NULL REFERENCES organization (id), -- the receiving clinic
    version_id integer DEFAULT 1 NOT NULL, -- bumped by every change of the selection
    status text NOT NULL CHECK (status IN ('active', 'inactive')),
    CONSTRAINT history_consent_revoked_when_inactive CHECK ((status = 'inactive') = (revoked_at IS NOT NULL))
);

CREATE UNIQUE INDEX history_consent_one_active ON history_consent (patient_id, organization_id) WHERE status = 'active';
CREATE INDEX history_consent_patient_organization ON history_consent (patient_id, organization_id);

CREATE TABLE history_consent_encounter (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    added_at timestamptz NOT NULL,
    removed_at timestamptz,
    consent_id uuid NOT NULL REFERENCES history_consent (id),
    encounter_id uuid NOT NULL REFERENCES encounter (id),
    CONSTRAINT history_consent_encounter_period CHECK (removed_at IS NULL OR removed_at >= added_at)
);

-- The open selection of a visit, at most one per consent; the lookup of a visit's open selections.
CREATE UNIQUE INDEX history_consent_encounter_open ON history_consent_encounter (consent_id, encounter_id) WHERE removed_at IS NULL;
CREATE INDEX history_consent_encounter_encounter ON history_consent_encounter (encounter_id) WHERE removed_at IS NULL;

-- The patient's own scope sees all of theirs, grants an active one, and changes an active one (its version) or
-- revokes it; a revocation is final, granting again adds a row. The receiving clinic reads only the active one.
ALTER TABLE history_consent ENABLE ROW LEVEL SECURITY;
CREATE POLICY history_consent_own ON history_consent FOR SELECT TO salus_app
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY history_consent_own_insert ON history_consent FOR INSERT TO salus_app
    WITH CHECK (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid AND status = 'active');
CREATE POLICY history_consent_own_update ON history_consent FOR UPDATE TO salus_app
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid AND status = 'active')
    WITH CHECK (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY history_consent_clinic ON history_consent FOR SELECT TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND status = 'active'
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- The patient sees every selection of theirs, removals included; the clinic only the open ones of its active consent,
-- so after a removal or a revocation it does not even learn which visits were shared.
ALTER TABLE history_consent_encounter ENABLE ROW LEVEL SECURITY;
CREATE POLICY history_consent_encounter_own ON history_consent_encounter FOR SELECT TO salus_app
    USING (EXISTS (SELECT 1 FROM history_consent h
                   WHERE h.id = history_consent_encounter.consent_id
                     AND h.patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid));
CREATE POLICY history_consent_encounter_clinic ON history_consent_encounter FOR SELECT TO salus_app
    USING (removed_at IS NULL
           AND EXISTS (SELECT 1 FROM history_consent h
                       WHERE h.id = history_consent_encounter.consent_id
                         AND h.status = 'active'
                         AND h.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
           AND NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL);
-- The patient lists a finished visit of their own at another clinic, into their own active consent.
CREATE POLICY history_consent_encounter_own_insert ON history_consent_encounter FOR INSERT TO salus_app
    WITH CHECK (EXISTS (SELECT 1 FROM history_consent h JOIN encounter e ON e.id = history_consent_encounter.encounter_id
                        WHERE h.id = history_consent_encounter.consent_id
                          AND h.status = 'active'
                          AND h.patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
                          AND e.patient_id = h.patient_id
                          AND e.organization_id <> h.organization_id
                          AND e.status = 'finished'));
-- A removal is final: only an open selection is updated, and only to removed.
CREATE POLICY history_consent_encounter_own_update ON history_consent_encounter FOR UPDATE TO salus_app
    USING (removed_at IS NULL
           AND EXISTS (SELECT 1 FROM history_consent h
                       WHERE h.id = history_consent_encounter.consent_id
                         AND h.patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid))
    WITH CHECK (removed_at IS NOT NULL);

-- Whether the visit encounter_id of the patient is shared with the clinic of the transaction's scope: the clinic's
-- scope only, an active history consent listing it, and an active care consent. Invoker's rights, so the policies of
-- the tables it reads apply too.
CREATE FUNCTION history_shared(encounter_id uuid, patient_id uuid) RETURNS boolean
    LANGUAGE sql
    STABLE
    SET search_path = pg_catalog, public
AS $$
    SELECT NULLIF(pg_catalog.current_setting('app.patient_id', true), '') IS NULL
       AND NULLIF(pg_catalog.current_setting('app.account_id', true), '') IS NULL
       AND EXISTS (SELECT 1 FROM public.history_consent_encounter s
                   JOIN public.history_consent h ON h.id = s.consent_id
                   WHERE s.encounter_id = history_shared.encounter_id
                     AND s.removed_at IS NULL
                     AND h.status = 'active'
                     AND h.patient_id = history_shared.patient_id
                     AND h.organization_id = NULLIF(pg_catalog.current_setting('app.organization_id', true), '')::uuid
                     AND EXISTS (SELECT 1 FROM public.consent c
                                 WHERE c.patient_id = h.patient_id
                                   AND c.organization_id = h.organization_id
                                   AND c.status = 'active'))
$$;
REVOKE EXECUTE ON FUNCTION history_shared(uuid, uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION history_shared(uuid, uuid) TO salus_app;

CREATE POLICY encounter_shared ON encounter FOR SELECT TO salus_app
    USING (status = 'finished' AND history_shared(id, patient_id));
CREATE POLICY document_draft_shared ON document_draft FOR SELECT TO salus_app
    USING (status = 'final'
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.id = document_draft.encounter_id AND history_shared(e.id, e.patient_id)));
CREATE POLICY questionnaire_response_shared ON questionnaire_response FOR SELECT TO salus_app
    USING (status = 'completed'
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.appointment_id = questionnaire_response.appointment_id
                         AND e.patient_id = questionnaire_response.patient_id
                         AND history_shared(e.id, e.patient_id)));
CREATE POLICY binary_shared ON "binary" FOR SELECT TO salus_app
    USING (status = 'clean'
           AND NULLIF(current_setting('app.questionnaire_response_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM questionnaire_response r
                       WHERE r.id = "binary".questionnaire_response_id AND r.status = 'completed')
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.appointment_id = "binary".appointment_id
                         AND e.patient_id = "binary".patient_id
                         AND history_shared(e.id, e.patient_id)));
CREATE POLICY organization_shared ON organization FOR SELECT TO salus_app
    USING (EXISTS (SELECT 1 FROM history_consent_encounter s JOIN encounter e ON e.id = s.encounter_id
                   WHERE s.removed_at IS NULL AND e.organization_id = organization.id
                     AND history_shared(e.id, e.patient_id)));
CREATE POLICY practitioner_role_shared ON practitioner_role FOR SELECT TO salus_app
    USING (EXISTS (SELECT 1 FROM history_consent_encounter s JOIN encounter e ON e.id = s.encounter_id
                   WHERE s.removed_at IS NULL AND e.practitioner_role_id = practitioner_role.id
                     AND history_shared(e.id, e.patient_id)));
-- The drafts and signed documents a clinic writes are its own encounters' (00031's document_draft_clinic followed
-- encounter's visibility and relied on the template's to stay closed): recreated for the clinic's own encounters,
-- otherwise as 00031. A shared visit's signed documents are read through document_draft_shared only.
DROP POLICY document_draft_clinic ON document_draft;
CREATE POLICY document_draft_clinic ON document_draft
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.id = document_draft.encounter_id
                         AND e.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
           AND EXISTS (SELECT 1 FROM document_template t WHERE t.id = document_draft.template_id));

-- The transcript and the suggestions are the clinic's own: 00031 opened them wherever the clinic's scope sees the
-- encounter, which encounter_shared would now stretch to another clinic's shared visit. Recreated for the clinic's own
-- encounters only, otherwise as 00031.
DROP POLICY transcript_segment_clinic ON transcript_segment;
CREATE POLICY transcript_segment_clinic ON transcript_segment
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.id = transcript_segment.encounter_id
                         AND e.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid));
DROP POLICY suggestion_clinic ON suggestion;
CREATE POLICY suggestion_clinic ON suggestion
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM encounter e
                       WHERE e.id = suggestion.encounter_id
                         AND e.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid));

-- The profile tables: the patient's own scope sees all of theirs, writes its own entries and deletes only its own
-- unconfirmed ones; a clinic sees the profile under an active care consent and the entries of the visits it sees,
-- records entries in its own visits and confirms (updates) what it sees.
ALTER TABLE allergy_intolerance ENABLE ROW LEVEL SECURITY;
CREATE POLICY allergy_intolerance_own ON allergy_intolerance FOR SELECT TO salus_app
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY allergy_intolerance_own_insert ON allergy_intolerance FOR INSERT TO salus_app
    WITH CHECK (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
                AND organization_id IS NULL AND confirmed_at IS NULL);
CREATE POLICY allergy_intolerance_own_update ON allergy_intolerance FOR UPDATE TO salus_app
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY allergy_intolerance_own_delete ON allergy_intolerance FOR DELETE TO salus_app
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
           AND organization_id IS NULL AND confirmed_at IS NULL);
CREATE POLICY allergy_intolerance_clinic ON allergy_intolerance FOR SELECT TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND ((in_profile AND EXISTS (SELECT 1 FROM consent c
                                        WHERE c.patient_id = allergy_intolerance.patient_id
                                          AND c.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                                          AND c.status = 'active'))
                OR EXISTS (SELECT 1 FROM encounter e WHERE e.id = allergy_intolerance.encounter_id)));
CREATE POLICY allergy_intolerance_clinic_insert ON allergy_intolerance FOR INSERT TO salus_app
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND NOT in_profile
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND (confirmed_organization_id IS NULL OR confirmed_organization_id = organization_id)
                AND EXISTS (SELECT 1 FROM encounter e
                            WHERE e.id = allergy_intolerance.encounter_id
                              AND e.organization_id = allergy_intolerance.organization_id
                              AND e.patient_id = allergy_intolerance.patient_id));
-- A clinic updates the profile under its care consent, and its own visits' entries only: a shared visit is read-only.
CREATE POLICY allergy_intolerance_clinic_update ON allergy_intolerance FOR UPDATE TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND ((in_profile AND EXISTS (SELECT 1 FROM consent c
                                        WHERE c.patient_id = allergy_intolerance.patient_id
                                          AND c.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                                          AND c.status = 'active'))
                OR (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                    AND EXISTS (SELECT 1 FROM encounter e WHERE e.id = allergy_intolerance.encounter_id))));

ALTER TABLE medication_statement ENABLE ROW LEVEL SECURITY;
CREATE POLICY medication_statement_own ON medication_statement FOR SELECT TO salus_app
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY medication_statement_own_insert ON medication_statement FOR INSERT TO salus_app
    WITH CHECK (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
                AND organization_id IS NULL AND confirmed_at IS NULL);
CREATE POLICY medication_statement_own_update ON medication_statement FOR UPDATE TO salus_app
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY medication_statement_own_delete ON medication_statement FOR DELETE TO salus_app
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
           AND organization_id IS NULL AND confirmed_at IS NULL);
CREATE POLICY medication_statement_clinic ON medication_statement FOR SELECT TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND ((in_profile AND EXISTS (SELECT 1 FROM consent c
                                        WHERE c.patient_id = medication_statement.patient_id
                                          AND c.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                                          AND c.status = 'active'))
                OR EXISTS (SELECT 1 FROM encounter e WHERE e.id = medication_statement.encounter_id)));
CREATE POLICY medication_statement_clinic_insert ON medication_statement FOR INSERT TO salus_app
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND NOT in_profile
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND (confirmed_organization_id IS NULL OR confirmed_organization_id = organization_id)
                AND EXISTS (SELECT 1 FROM encounter e
                            WHERE e.id = medication_statement.encounter_id
                              AND e.organization_id = medication_statement.organization_id
                              AND e.patient_id = medication_statement.patient_id));
-- A clinic updates the profile under its care consent, and its own visits' entries only: a shared visit is read-only.
CREATE POLICY medication_statement_clinic_update ON medication_statement FOR UPDATE TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND ((in_profile AND EXISTS (SELECT 1 FROM consent c
                                        WHERE c.patient_id = medication_statement.patient_id
                                          AND c.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                                          AND c.status = 'active'))
                OR (organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                    AND EXISTS (SELECT 1 FROM encounter e WHERE e.id = medication_statement.encounter_id))));

-- What a scope may change of an entry, which row-level security cannot say (it does not see the old row): the patient
-- never confirms, edits only their own entries and, of a clinic's, only adds it to the profile or removes it
-- (entered-in-error); a clinic never moves an entry into or out of the profile, confirms only as itself with one of
-- its roles, and edits only its own entries. The owner (migrations, salusctl) is not checked.
-- +goose StatementBegin
CREATE FUNCTION profile_entry_update_check() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog, public
AS $$
DECLARE
    clinic uuid := NULLIF(current_setting('app.organization_id', true), '')::uuid;
    patient_scope boolean := NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL;
    -- The columns compared apart: all others are the entry's content.
    apart text[] := ARRAY['updated_at', 'version_id', 'in_profile', 'verification_status', 'status',
                          'confirmed_at', 'confirmed_by_role_id', 'confirmed_organization_id'];
    old_row jsonb := to_jsonb(OLD);
    new_row jsonb := to_jsonb(NEW);
    old_state text := coalesce(old_row->>'verification_status', old_row->>'status');
    new_state text := coalesce(new_row->>'verification_status', new_row->>'status');
    confirmation_changed boolean := (NEW.confirmed_at, NEW.confirmed_by_role_id, NEW.confirmed_organization_id)
        IS DISTINCT FROM (OLD.confirmed_at, OLD.confirmed_by_role_id, OLD.confirmed_organization_id);
    -- An edit: the content, or the status to anything but a removal or (for an allergy) a confirmation.
    edited boolean := (new_row - apart) IS DISTINCT FROM (old_row - apart)
        OR (new_state IS DISTINCT FROM old_state AND new_state NOT IN ('entered-in-error', 'confirmed'));
    own boolean := OLD.organization_id IS NOT DISTINCT FROM clinic AND NOT patient_scope;
BEGIN
    -- The table's owner (migrations, salusctl) only; every other role is checked.
    IF current_user = (SELECT r.rolname FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner WHERE c.oid = TG_RELID) THEN
        RETURN NEW;
    END IF;
    -- A removal is final: the entry is kept with the record and shown to no one.
    IF old_state = 'entered-in-error' THEN
        RAISE EXCEPTION 'a removed entry does not change' USING ERRCODE = 'insufficient_privilege';
    END IF;
    -- A confirmation vouches for what it confirmed: an edit keeps it only when the confirming clinic makes it.
    IF edited AND NEW.confirmed_at IS NOT NULL AND NOT confirmation_changed
       AND (patient_scope OR NEW.confirmed_organization_id IS DISTINCT FROM clinic) THEN
        RAISE EXCEPTION 'an edit of a confirmed entry clears or renews the confirmation' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF patient_scope THEN
        -- The patient never confirms; of a clinic's entry they only add it to the profile or remove it.
        IF confirmation_changed AND (NEW.confirmed_at IS NOT NULL OR OLD.organization_id IS NOT NULL) THEN
            RAISE EXCEPTION 'a patient does not confirm an entry' USING ERRCODE = 'insufficient_privilege';
        END IF;
        IF OLD.organization_id IS NOT NULL AND (edited OR new_state = 'confirmed' AND old_state <> 'confirmed') THEN
            RAISE EXCEPTION 'a patient does not edit a clinic''s entry' USING ERRCODE = 'insufficient_privilege';
        END IF;
        RETURN NEW;
    END IF;
    -- A clinic never moves an entry into or out of the profile.
    IF NEW.in_profile IS DISTINCT FROM OLD.in_profile THEN
        RAISE EXCEPTION 'a clinic does not change the profile' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF confirmation_changed AND NEW.confirmed_at IS NOT NULL THEN
        -- It confirms only as itself, with one of its roles, and never over another clinic's confirmation.
        IF OLD.confirmed_at IS NOT NULL AND OLD.confirmed_organization_id IS DISTINCT FROM clinic THEN
            RAISE EXCEPTION 'a clinic does not overwrite another clinic''s confirmation' USING ERRCODE = 'insufficient_privilege';
        END IF;
        IF NEW.confirmed_organization_id IS DISTINCT FROM clinic
           OR NOT EXISTS (SELECT 1 FROM practitioner_role r WHERE r.id = NEW.confirmed_by_role_id AND r.organization_id = clinic) THEN
            RAISE EXCEPTION 'a clinic confirms only as itself' USING ERRCODE = 'insufficient_privilege';
        END IF;
    ELSIF confirmation_changed AND NOT (own OR OLD.confirmed_organization_id IS NOT DISTINCT FROM clinic) THEN
        -- It clears a confirmation only of its own entry (to correct it) or its own confirmation.
        RAISE EXCEPTION 'a clinic clears only its own entry''s or its own confirmation' USING ERRCODE = 'insufficient_privilege';
    END IF;
    -- It edits and removes only its own entries; of another's it may only confirm.
    IF NOT own AND (edited OR (new_state IS DISTINCT FROM old_state AND NOT (new_state = 'confirmed' AND confirmation_changed))) THEN
        RAISE EXCEPTION 'a clinic edits only its own entries' USING ERRCODE = 'insufficient_privilege';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER allergy_intolerance_update_check BEFORE UPDATE ON allergy_intolerance
    FOR EACH ROW EXECUTE FUNCTION profile_entry_update_check();
CREATE TRIGGER medication_statement_update_check BEFORE UPDATE ON medication_statement
    FOR EACH ROW EXECUTE FUNCTION profile_entry_update_check();

-- Who, which clinic, which visit and who recorded it never change. The patient's own unconfirmed entries can be
-- deleted; everything else is kept with the clinical record (entered-in-error instead).
GRANT SELECT, INSERT, DELETE ON allergy_intolerance TO salus_app;
GRANT UPDATE (updated_at, version_id, in_profile, term_id, code_text, category, reaction_text, severity, clinical_status,
    verification_status, confirmed_at, confirmed_by_role_id, confirmed_organization_id) ON allergy_intolerance TO salus_app;
GRANT SELECT, INSERT, DELETE ON medication_statement TO salus_app;
GRANT UPDATE (updated_at, version_id, in_profile, term_id, medication_text, dose_value, dose_unit, frequency, period,
    period_unit, as_needed, status, confirmed_at, confirmed_by_role_id, confirmed_organization_id) ON medication_statement TO salus_app;
-- Consents are kept as proof: no DELETE. A selection's removal is its removed_at.
GRANT SELECT, INSERT ON history_consent TO salus_app;
GRANT UPDATE (status, revoked_at, updated_at, version_id) ON history_consent TO salus_app;
GRANT SELECT, INSERT ON history_consent_encounter TO salus_app;
GRANT UPDATE (removed_at) ON history_consent_encounter TO salus_app;
