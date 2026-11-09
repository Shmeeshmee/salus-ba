-- +goose Up
-- Languages (docs/i18n, package 0.1; SPEC.md §3 "Languages", "The account's language fields", "The clinic's
-- default language", "Questionnaires", §5):
--   - the enum language: BCP 47 tags, the five staff languages first. Adding one later is
--     ALTER TYPE language ADD VALUE in a new migration, plus its catalogs;
--   - account.language (the user's own choice) and account.detected_language (what the client matched at the
--     last registration or sign-in), both null until known. The service refuses a staff account's choice
--     outside the staff languages;
--   - organization.default_language: a staff language, bs unless an admin changes it;
--   - questionnaire.language: the original language, the clinic's default for the existing ones; bs when an
--     insert gives none (the service passes the clinic's default);
--   - questionnaire_translation: one translation of one version per language, the texts keyed by linkId. No
--     personal data. Row-level secured through questionnaire: a scope sees the translations of the versions
--     it sees there (the clinic its own, a patient those their responses reference), and only the clinic
--     writes them, only while the version is a draft. That the language is not the version's own is the
--     service's rule. Deleting a draft deletes its translations;
--   - questionnaire_response.language: the language the patient answers in, null until they start.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TYPE language AS ENUM ('bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en', 'de', 'ar', 'tr', 'it', 'fr');

ALTER TABLE account
    ADD COLUMN language language,
    ADD COLUMN detected_language language;

ALTER TABLE organization
    ADD COLUMN default_language language DEFAULT 'bs' NOT NULL
        CONSTRAINT organization_default_language_staff CHECK (default_language IN ('bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en'));
GRANT UPDATE (default_language) ON organization TO salus_app;

ALTER TABLE questionnaire ADD COLUMN language language DEFAULT 'bs' NOT NULL;
UPDATE questionnaire q SET language = o.default_language FROM organization o WHERE o.id = q.organization_id;
GRANT UPDATE (language) ON questionnaire TO salus_app;

CREATE TABLE questionnaire_translation (
    questionnaire_id uuid NOT NULL REFERENCES questionnaire (id) ON DELETE CASCADE,
    language language NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    -- {title, description, items: {linkId: {text, help, options}}}, as the service validates it.
    texts jsonb DEFAULT '{}' NOT NULL CHECK (jsonb_typeof(texts) = 'object'),
    PRIMARY KEY (questionnaire_id, language)
);

ALTER TABLE questionnaire_translation ENABLE ROW LEVEL SECURITY;
-- Read: the EXISTS applies questionnaire's own policies (the clinic's, and the patient's).
CREATE POLICY questionnaire_translation_read ON questionnaire_translation FOR SELECT
    USING (EXISTS (SELECT 1 FROM questionnaire q WHERE q.id = questionnaire_translation.questionnaire_id));
-- Write: the clinic scope only, on its own drafts.
CREATE POLICY questionnaire_translation_insert ON questionnaire_translation FOR INSERT
    WITH CHECK (NULLIF(current_setting('app.patient_id', true), '') IS NULL
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND EXISTS (SELECT 1 FROM questionnaire q
                            WHERE q.id = questionnaire_translation.questionnaire_id AND q.status = 'draft'));
CREATE POLICY questionnaire_translation_update ON questionnaire_translation FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM questionnaire q
                       WHERE q.id = questionnaire_translation.questionnaire_id AND q.status = 'draft'));
CREATE POLICY questionnaire_translation_delete ON questionnaire_translation FOR DELETE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM questionnaire q
                       WHERE q.id = questionnaire_translation.questionnaire_id AND q.status = 'draft'));

-- Which version and language a translation is never change; its texts do.
GRANT SELECT, INSERT, DELETE ON questionnaire_translation TO salus_app;
GRANT UPDATE (texts, updated_at) ON questionnaire_translation TO salus_app;

ALTER TABLE questionnaire_response ADD COLUMN language language;
GRANT UPDATE (language) ON questionnaire_response TO salus_app;
