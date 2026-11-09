-- +goose Up
-- Document templates and drafting (docs/plans/consult-intelligence.md, E / SAL-8).
--   - document_template: its original language (the enum language), its status (draft while the clinic has not
--     reviewed its wording, as the seeded examples are; active once it has) and the LOINC code of the document
--     type it writes (Composition.type). Its fields become [{key, type, label, instruction, required}].
--   - document_template_translation (owner composition): a template's texts in one other language, like
--     questionnaire_translation (docs/i18n/SPEC.md §3): {title, fields: {<key>: {label, instruction}}}. No
--     personal data. Seen and written only with its template, in its clinic's scope.
--   - document_draft: the language, title and type it was drafted with (a draft keeps the wording it was made
--     from, whatever later happens to its template; its fields hold each field's label, type and required flag
--     too), and once signed the signature: when, by which PractitionerRole, the ids of the DocumentReference and
--     Provenance it produced, and its PDF's size, SHA-1, key-encryption key and wrapped data key (the PDF lives
--     encrypted in the object store, like an attachment, ADR 0010). One open (preliminary) draft per encounter
--     and template: a regeneration replaces its fields.
-- Retention: document_draft is keep (00031); the translation holds no personal data.

ALTER TABLE document_template
    ADD COLUMN language language DEFAULT 'bs' NOT NULL,
    ADD COLUMN status text DEFAULT 'active' NOT NULL CHECK (status IN ('draft', 'active')),
    ADD COLUMN type_code text DEFAULT '11488-4' NOT NULL CHECK (type_code ~ '^[0-9]{1,7}-[0-9]$');

GRANT UPDATE (language, status, type_code) ON document_template TO salus_app;

CREATE TABLE document_template_translation (
    template_id uuid NOT NULL REFERENCES document_template (id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT now() NOT NULL,
    language language NOT NULL,
    texts jsonb NOT NULL CHECK (jsonb_typeof(texts) = 'object'),
    PRIMARY KEY (template_id, language)
);

ALTER TABLE document_template_translation ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_template_translation_clinic ON document_template_translation
    USING (EXISTS (SELECT 1 FROM document_template t WHERE t.id = document_template_translation.template_id));

-- A template's translations are replaced as a whole on an edit.
GRANT SELECT, INSERT, UPDATE (texts), DELETE ON document_template_translation TO salus_app;

ALTER TABLE document_draft
    ADD COLUMN signed_at timestamptz,
    ADD COLUMN pdf_size bigint CHECK (pdf_size > 0),
    ADD COLUMN signer_role_id uuid REFERENCES practitioner_role (id),
    ADD COLUMN document_reference_id uuid UNIQUE,
    ADD COLUMN provenance_id uuid UNIQUE,
    ADD COLUMN language language DEFAULT 'bs' NOT NULL,
    ADD COLUMN title text DEFAULT '' NOT NULL,
    ADD COLUMN type_code text DEFAULT '11488-4' NOT NULL CHECK (type_code ~ '^[0-9]{1,7}-[0-9]$'),
    ADD COLUMN pdf_kek_id text,
    ADD COLUMN pdf_sha1 bytea CHECK (octet_length(pdf_sha1) = 20),
    ADD COLUMN pdf_wrapped_key bytea,
    -- A signature is whole or absent.
    ADD CONSTRAINT document_draft_signature CHECK (num_nulls(signed_at, pdf_size, signer_role_id, document_reference_id,
        provenance_id, pdf_kek_id, pdf_sha1, pdf_wrapped_key) IN (0, 8));

-- The open draft of an encounter's template, which a regeneration replaces.
CREATE UNIQUE INDEX document_draft_open ON document_draft (encounter_id, template_id) WHERE status = 'preliminary';

GRANT UPDATE (signed_at, pdf_size, signer_role_id, document_reference_id, provenance_id, language, title, type_code,
    pdf_kek_id, pdf_sha1, pdf_wrapped_key) ON document_draft TO salus_app;
