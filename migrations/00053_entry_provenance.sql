-- +goose Up
-- Who did what to an allergy or a medication (SAL-41, docs/plans/patient-history.md, ADR 0014): one FHIR R4
-- Provenance per clinical write, append-only, written in the same transaction as the write. A clinical write is a
-- practitioner's: recording an entry in a visit (CREATE), changing one (UPDATE), confirming one (UPDATE, the agent the
-- verifier) and removing one (NULLIFY: kept as entered-in-error). The patient's own writes are patient-reported: the
-- entry names the patient as its asserter and the AuditEvent records the write, so they write none.
--
-- A row is one Provenance: its target is the entry's version the write produced (target_version), recorded when it
-- was written, the agent the practitioner role on behalf of its clinic. activity is a code of
-- http://terminology.hl7.org/CodeSystem/v3-DataOperation, agent_type one of
-- http://terminology.hl7.org/CodeSystem/provenance-participant-type.
--
-- A Provenance names the clinic that wrote, so it is seen only where that clinic may be (access.Attributed, ADR 0014
-- "Doctor entries"): by the patient, and by the writing clinic's scope while it sees the entry. Another clinic that
-- sees the entry (in the profile, or through a shared visit) sees "confirmed by a clinician" and never the clinic's
-- Provenance. A clinic writes only its own, on an entry it sees, at the entry's current version. No UPDATE and no
-- DELETE for the application: a Provenance goes only with its entry, when the patient deletes their own unconfirmed
-- entry (a confirmation the patient's change cleared since): ON DELETE CASCADE, run as the table's owner.
-- The policies read the entry table, whose policies never read these tables: no recursion.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE allergy_intolerance_provenance (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    recorded timestamptz NOT NULL,
    allergy_intolerance_id uuid NOT NULL REFERENCES allergy_intolerance (id) ON DELETE CASCADE,
    practitioner_role_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    target_version integer NOT NULL CHECK (target_version >= 1),
    activity text NOT NULL CHECK (activity IN ('CREATE', 'UPDATE', 'NULLIFY')),
    agent_type text NOT NULL CHECK (agent_type IN ('author', 'verifier')),
    CONSTRAINT allergy_intolerance_provenance_agent_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT allergy_intolerance_provenance_created CHECK ((activity = 'CREATE') = (target_version = 1))
);

CREATE TABLE medication_statement_provenance (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    recorded timestamptz NOT NULL,
    medication_statement_id uuid NOT NULL REFERENCES medication_statement (id) ON DELETE CASCADE,
    practitioner_role_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    target_version integer NOT NULL CHECK (target_version >= 1),
    activity text NOT NULL CHECK (activity IN ('CREATE', 'UPDATE', 'NULLIFY')),
    agent_type text NOT NULL CHECK (agent_type IN ('author', 'verifier')),
    CONSTRAINT medication_statement_provenance_agent_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    CONSTRAINT medication_statement_provenance_created CHECK ((activity = 'CREATE') = (target_version = 1))
);

-- An entry's Provenances (the cascade, and the policies' lookup of the entry runs the other way).
CREATE INDEX allergy_intolerance_provenance_entry ON allergy_intolerance_provenance (allergy_intolerance_id);
CREATE INDEX medication_statement_provenance_entry ON medication_statement_provenance (medication_statement_id);

ALTER TABLE allergy_intolerance_provenance ENABLE ROW LEVEL SECURITY;
CREATE POLICY allergy_intolerance_provenance_read ON allergy_intolerance_provenance FOR SELECT TO salus_app
    USING ((NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL
            OR organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
           AND EXISTS (SELECT 1 FROM allergy_intolerance a WHERE a.id = allergy_intolerance_provenance.allergy_intolerance_id));
CREATE POLICY allergy_intolerance_provenance_insert ON allergy_intolerance_provenance FOR INSERT TO salus_app
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM allergy_intolerance a
                            WHERE a.id = allergy_intolerance_provenance.allergy_intolerance_id
                              AND a.version_id = allergy_intolerance_provenance.target_version));

ALTER TABLE medication_statement_provenance ENABLE ROW LEVEL SECURITY;
CREATE POLICY medication_statement_provenance_read ON medication_statement_provenance FOR SELECT TO salus_app
    USING ((NULLIF(current_setting('app.patient_id', true), '') IS NOT NULL
            OR organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid)
           AND EXISTS (SELECT 1 FROM medication_statement m WHERE m.id = medication_statement_provenance.medication_statement_id));
CREATE POLICY medication_statement_provenance_insert ON medication_statement_provenance FOR INSERT TO salus_app
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
                AND EXISTS (SELECT 1 FROM medication_statement m
                            WHERE m.id = medication_statement_provenance.medication_statement_id
                              AND m.version_id = medication_statement_provenance.target_version));

-- Append-only.
GRANT SELECT, INSERT ON allergy_intolerance_provenance, medication_statement_provenance TO salus_app;
