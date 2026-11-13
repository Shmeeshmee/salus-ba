-- +goose Up
-- Who did what to an allergy or a medication (SAL-41, docs/plans/patient-history.md, ADR 0014): one FHIR R4
-- Provenance per write that is not the patient's own on their own entry, append-only, written in the same transaction
-- as the write. A practitioner's write: recording an entry in a visit (CREATE, which confirms it too), changing one
-- (UPDATE), confirming one (UPDATE, the agent the verifier) and removing one (NULLIFY: kept as entered-in-error). The
-- patient's write on a clinic's entry: adding it to the profile (UPDATE) and removing it (NULLIFY), the patient the
-- agent. The patient's writes on their own entries are patient-reported: the entry names the patient as its source
-- and the AuditEvent records the write, so they write none.
--
-- A row is one Provenance: its target is the entry's version the write produced (target_version), recorded when it
-- was written, its agent a practitioner role on behalf of its clinic or the patient. activity is a code of
-- http://terminology.hl7.org/CodeSystem/v3-DataOperation, agent_type one of
-- http://terminology.hl7.org/CodeSystem/provenance-participant-type. Each insert is tied to the write it attests: the
-- entry at that version and in the state the write left, written by that agent (the insert policies), and one per
-- version (the write that produced it).
--
-- A clinic's Provenance names the clinic, so it is seen only where that clinic may be (access.Attributed, ADR 0014
-- "Doctor entries"): by the patient, and by the writing clinic's scope while it sees the entry. Another clinic that
-- sees the entry (in the profile, or through a shared visit) sees "confirmed by a clinician" and never a Provenance;
-- the patient's Provenances are the patient's scope's only.
--
-- No UPDATE and no DELETE for the application, and never deleted with the entry: the foreign key has no action, so
-- deleting an entry with a Provenance fails. The service deletes only the patient's own entry that no clinic ever
-- confirmed or wrote (none has a Provenance), and keeps any other as entered-in-error. The entry's delete policy does
-- not read these tables (their policies read the entry's: it would recurse).
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE allergy_intolerance_provenance (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    recorded timestamptz NOT NULL,
    allergy_intolerance_id uuid NOT NULL REFERENCES allergy_intolerance (id), -- NO ACTION: an entry with a Provenance is never deleted
    practitioner_role_id uuid, -- a clinic's agent, with its organization_id
    organization_id uuid,
    patient_id uuid REFERENCES patient (id), -- the patient as the agent, on a clinic's entry
    target_version integer NOT NULL CHECK (target_version >= 1),
    activity text NOT NULL CHECK (activity IN ('CREATE', 'UPDATE', 'NULLIFY')),
    agent_type text NOT NULL CHECK (agent_type IN ('author', 'verifier')),
    CONSTRAINT allergy_intolerance_provenance_agent_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT allergy_intolerance_provenance_created CHECK ((activity = 'CREATE') = (target_version = 1)),
    -- One agent: a clinic's practitioner role, or the patient, who never creates or verifies.
    CONSTRAINT allergy_intolerance_provenance_one_agent CHECK (num_nulls(practitioner_role_id, organization_id) IN (0, 2)
                                                   AND (organization_id IS NULL) <> (patient_id IS NULL)),
    CONSTRAINT allergy_intolerance_provenance_patient_agent CHECK (patient_id IS NULL OR (activity <> 'CREATE' AND agent_type = 'author')),
    -- A creation is stored as its author; the recorder's confirmation is its second agent, rendered by the service.
    CONSTRAINT allergy_intolerance_provenance_created_by_author CHECK (activity <> 'CREATE' OR agent_type = 'author'),
    -- One write, one version, one Provenance.
    CONSTRAINT allergy_intolerance_provenance_one_per_version UNIQUE (allergy_intolerance_id, target_version)
);

CREATE TABLE medication_statement_provenance (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    recorded timestamptz NOT NULL,
    medication_statement_id uuid NOT NULL REFERENCES medication_statement (id), -- NO ACTION: an entry with a Provenance is never deleted
    practitioner_role_id uuid, -- a clinic's agent, with its organization_id
    organization_id uuid,
    patient_id uuid REFERENCES patient (id), -- the patient as the agent, on a clinic's entry
    target_version integer NOT NULL CHECK (target_version >= 1),
    activity text NOT NULL CHECK (activity IN ('CREATE', 'UPDATE', 'NULLIFY')),
    agent_type text NOT NULL CHECK (agent_type IN ('author', 'verifier')),
    CONSTRAINT medication_statement_provenance_agent_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT medication_statement_provenance_created CHECK ((activity = 'CREATE') = (target_version = 1)),
    -- One agent: a clinic's practitioner role, or the patient, who never creates or verifies.
    CONSTRAINT medication_statement_provenance_one_agent CHECK (num_nulls(practitioner_role_id, organization_id) IN (0, 2)
                                                   AND (organization_id IS NULL) <> (patient_id IS NULL)),
    CONSTRAINT medication_statement_provenance_patient_agent CHECK (patient_id IS NULL OR (activity <> 'CREATE' AND agent_type = 'author')),
    -- A creation is stored as its author; the recorder's confirmation is its second agent, rendered by the service.
    CONSTRAINT medication_statement_provenance_created_by_author CHECK (activity <> 'CREATE' OR agent_type = 'author'),
    -- One write, one version, one Provenance.
    CONSTRAINT medication_statement_provenance_one_per_version UNIQUE (medication_statement_id, target_version)
);

-- An entry's Provenances (the foreign key, the service's check before a delete).
CREATE INDEX allergy_intolerance_provenance_entry ON allergy_intolerance_provenance (allergy_intolerance_id);
CREATE INDEX medication_statement_provenance_entry ON medication_statement_provenance (medication_statement_id);

ALTER TABLE allergy_intolerance_provenance ENABLE ROW LEVEL SECURITY;
CREATE POLICY allergy_intolerance_provenance_read ON allergy_intolerance_provenance FOR SELECT TO salus_app
    USING ((NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL
            OR organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
           AND EXISTS (SELECT 1 FROM allergy_intolerance a WHERE a.id = allergy_intolerance_provenance.allergy_intolerance_id));
-- A clinic's Provenance names the write it attests, in the state that write leaves: recording by the entry's recorder
-- (who confirms it with the same role), a confirmation by the entry's confirmer of a confirmed entry, a removal of its
-- own clinic's entry now entered-in-error, a change of its own clinic's entry not removed. Which role of the clinic
-- acted, for a change or a removal, is the service's to bind (the caller's role).
CREATE POLICY allergy_intolerance_provenance_clinic_insert ON allergy_intolerance_provenance FOR INSERT TO salus_app
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND patient_id IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM allergy_intolerance a
                            WHERE a.id = allergy_intolerance_provenance.allergy_intolerance_id
                              AND a.version_id = allergy_intolerance_provenance.target_version
                              AND CASE
                                  WHEN allergy_intolerance_provenance.activity = 'CREATE' THEN
                                      a.organization_id = allergy_intolerance_provenance.organization_id
                                      AND a.recorder_role_id = allergy_intolerance_provenance.practitioner_role_id
                                      AND a.confirmed_organization_id = allergy_intolerance_provenance.organization_id
                                      AND a.confirmed_by_role_id = allergy_intolerance_provenance.practitioner_role_id
                                  WHEN allergy_intolerance_provenance.activity = 'NULLIFY' THEN
                                      a.verification_status = 'entered-in-error'
                                      AND a.organization_id = allergy_intolerance_provenance.organization_id
                                  WHEN allergy_intolerance_provenance.agent_type = 'verifier' THEN
                                      a.verification_status = 'confirmed'
                                      AND a.confirmed_organization_id = allergy_intolerance_provenance.organization_id
                                      AND a.confirmed_by_role_id = allergy_intolerance_provenance.practitioner_role_id
                                  ELSE NOT (a.verification_status = 'entered-in-error')
                                      AND a.organization_id = allergy_intolerance_provenance.organization_id
                              END));
-- The patient's Provenance: their addition of a clinic's entry of theirs to the profile, or its removal.
CREATE POLICY allergy_intolerance_provenance_patient_insert ON allergy_intolerance_provenance FOR INSERT TO salus_app
    WITH CHECK (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM allergy_intolerance a
                            WHERE a.id = allergy_intolerance_provenance.allergy_intolerance_id
                              AND a.version_id = allergy_intolerance_provenance.target_version
                              AND a.patient_id = allergy_intolerance_provenance.patient_id
                              AND a.organization_id IS NOT NULL
                              AND CASE WHEN allergy_intolerance_provenance.activity = 'NULLIFY' THEN a.verification_status = 'entered-in-error'
                                       ELSE a.in_profile AND NOT (a.verification_status = 'entered-in-error') END));

ALTER TABLE medication_statement_provenance ENABLE ROW LEVEL SECURITY;
CREATE POLICY medication_statement_provenance_read ON medication_statement_provenance FOR SELECT TO salus_app
    USING ((NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL
            OR organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
           AND EXISTS (SELECT 1 FROM medication_statement m WHERE m.id = medication_statement_provenance.medication_statement_id));
-- A clinic's Provenance names the write it attests, in the state that write leaves: recording by the entry's recorder
-- (who confirms it with the same role), a confirmation by the entry's confirmer of a confirmed entry, a removal of its
-- own clinic's entry now entered-in-error, a change of its own clinic's entry not removed. Which role of the clinic
-- acted, for a change or a removal, is the service's to bind (the caller's role).
CREATE POLICY medication_statement_provenance_clinic_insert ON medication_statement_provenance FOR INSERT TO salus_app
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND patient_id IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM medication_statement m
                            WHERE m.id = medication_statement_provenance.medication_statement_id
                              AND m.version_id = medication_statement_provenance.target_version
                              AND CASE
                                  WHEN medication_statement_provenance.activity = 'CREATE' THEN
                                      m.organization_id = medication_statement_provenance.organization_id
                                      AND m.recorder_role_id = medication_statement_provenance.practitioner_role_id
                                      AND m.confirmed_organization_id = medication_statement_provenance.organization_id
                                      AND m.confirmed_by_role_id = medication_statement_provenance.practitioner_role_id
                                  WHEN medication_statement_provenance.activity = 'NULLIFY' THEN
                                      m.status = 'entered-in-error'
                                      AND m.organization_id = medication_statement_provenance.organization_id
                                  WHEN medication_statement_provenance.agent_type = 'verifier' THEN
                                      m.confirmed_at IS NOT NULL AND m.status <> 'entered-in-error'
                                      AND m.confirmed_organization_id = medication_statement_provenance.organization_id
                                      AND m.confirmed_by_role_id = medication_statement_provenance.practitioner_role_id
                                  ELSE NOT (m.status = 'entered-in-error')
                                      AND m.organization_id = medication_statement_provenance.organization_id
                              END));
-- The patient's Provenance: their addition of a clinic's entry of theirs to the profile, or its removal.
CREATE POLICY medication_statement_provenance_patient_insert ON medication_statement_provenance FOR INSERT TO salus_app
    WITH CHECK (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM medication_statement m
                            WHERE m.id = medication_statement_provenance.medication_statement_id
                              AND m.version_id = medication_statement_provenance.target_version
                              AND m.patient_id = medication_statement_provenance.patient_id
                              AND m.organization_id IS NOT NULL
                              AND CASE WHEN medication_statement_provenance.activity = 'NULLIFY' THEN m.status = 'entered-in-error'
                                       ELSE m.in_profile AND NOT (m.status = 'entered-in-error') END));

-- Append-only.
GRANT SELECT, INSERT ON allergy_intolerance_provenance, medication_statement_provenance TO salus_app;
