-- +goose Up
-- The consultation's transcript, suggestions and drafts (docs/plans/consult-intelligence.md, P0 / SAL-2): the
-- tables the live pipeline stores into. Each row of transcript_segment, suggestion and document_draft belongs to
-- one FHIR Encounter; document_template is a clinic's.
--   - transcript_segment (owner transcript): a final segment, in the language it was spoken in, and its
--     translation. Its id is the segment_id of the live events. Partial segments are never stored. Never changes.
--   - suggestion (owner suggestion): what was suggested to the practitioner, from which segments, and what they
--     did with it (status).
--   - document_template (owner composition): a clinic's template, its fields as [{key, label}]. No personal data.
--   - document_draft (owner composition): a FHIR Composition drafted from a template; fields as
--     [{key, value, source_segment_ids, missing}], the event's shape (internal/composition).
-- Row-level security, the clinic's scope only (app.organization_id; no patient, account or video room setting):
-- a row follows its encounter, which the clinic sees with an active consent or, after a revocation, for an
-- appointment that started before it (encounter_clinic). A video room's scope sees nothing here: it shows an
-- encounter after a revocation (encounter_room), and a transcript must not be written then. A draft also needs
-- its template visible, so it is always from its encounter's clinic. The patient's copy comes with the export
-- feature. NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.
-- Retention: keep, with the clinical record, like encounter (internal/retention).

CREATE TABLE transcript_segment (
    id uuid NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    ts_start timestamptz NOT NULL,
    ts_end timestamptz NOT NULL,
    encounter_id uuid NOT NULL REFERENCES encounter (id),
    speaker text NOT NULL CHECK (speaker IN ('doctor', 'patient')),
    lang text NOT NULL CHECK (lang <> ''),
    original text NOT NULL CHECK (btrim(original) <> ''),
    translation text DEFAULT '' NOT NULL,
    CONSTRAINT transcript_segment_period CHECK (ts_end >= ts_start)
);

-- An encounter's transcript, in the order it was said.
CREATE INDEX transcript_segment_encounter ON transcript_segment (encounter_id, ts_start);

ALTER TABLE transcript_segment ENABLE ROW LEVEL SECURITY;
CREATE POLICY transcript_segment_clinic ON transcript_segment
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM encounter e WHERE e.id = transcript_segment.encounter_id));

GRANT SELECT, INSERT ON transcript_segment TO salus_app;

CREATE TABLE suggestion (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    encounter_id uuid NOT NULL REFERENCES encounter (id),
    version_id integer DEFAULT 1 NOT NULL,
    confidence real NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    type text NOT NULL CHECK (type IN ('diagnosis', 'question', 'red_flag', 'term')),
    status text DEFAULT 'shown' NOT NULL CHECK (status IN ('shown', 'dismissed', 'pinned')),
    text text NOT NULL CHECK (btrim(text) <> ''),
    source_segment_ids uuid[] NOT NULL CHECK (cardinality(source_segment_ids) >= 1)
);

-- An encounter's suggestions, newest last.
CREATE INDEX suggestion_encounter ON suggestion (encounter_id, created_at);

ALTER TABLE suggestion ENABLE ROW LEVEL SECURITY;
CREATE POLICY suggestion_clinic ON suggestion
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM encounter e WHERE e.id = suggestion.encounter_id));

-- What was suggested and from what never changes; what the practitioner did with it does.
GRANT SELECT, INSERT ON suggestion TO salus_app;
GRANT UPDATE (status, updated_at, version_id) ON suggestion TO salus_app;

CREATE TABLE document_template (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    organization_id uuid NOT NULL REFERENCES organization (id),
    version_id integer DEFAULT 1 NOT NULL,
    title text NOT NULL CHECK (btrim(title) <> ''),
    fields jsonb NOT NULL CHECK (jsonb_typeof(fields) = 'array' AND jsonb_array_length(fields) >= 1)
);

-- A clinic's templates.
CREATE INDEX document_template_organization ON document_template (organization_id);

ALTER TABLE document_template ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_template_clinic ON document_template
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- Whose a template is never changes; nothing is deleted while a draft may reference it.
GRANT SELECT, INSERT ON document_template TO salus_app;
GRANT UPDATE (title, fields, updated_at, version_id) ON document_template TO salus_app;

CREATE TABLE document_draft (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    encounter_id uuid NOT NULL REFERENCES encounter (id),
    template_id uuid NOT NULL REFERENCES document_template (id),
    version_id integer DEFAULT 1 NOT NULL,
    status text DEFAULT 'preliminary' NOT NULL CHECK (status IN ('preliminary', 'final', 'entered-in-error')),
    fields jsonb NOT NULL CHECK (jsonb_typeof(fields) = 'array' AND jsonb_array_length(fields) >= 1)
);

-- An encounter's drafts.
CREATE INDEX document_draft_encounter ON document_draft (encounter_id);
CREATE INDEX document_draft_template ON document_draft (template_id);

ALTER TABLE document_draft ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_draft_clinic ON document_draft
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND NULLIF(current_setting('app.video_room', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM encounter e WHERE e.id = document_draft.encounter_id)
           AND EXISTS (SELECT 1 FROM document_template t WHERE t.id = document_draft.template_id));

-- Which encounter and template a draft is from never change; its fields and status do. No DELETE: a discarded
-- draft is entered-in-error, kept with the record.
GRANT SELECT, INSERT ON document_draft TO salus_app;
GRANT UPDATE (status, fields, updated_at, version_id) ON document_draft TO salus_app;
