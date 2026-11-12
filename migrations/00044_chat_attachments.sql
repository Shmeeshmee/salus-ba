-- +goose Up
-- Chat attachments and search (docs/plans/chat.md, SAL-30).
--
-- Attachments are "binary" rows (ADR 0010, docs/attachments) with a second kind of parent: a conversation, uploaded by
-- one of its participants before the message is sent, then attached to that message once (communication_id with its
-- sent_at, the message's key). A questionnaire file keeps its parent, its policies and its checks; a chat file has
-- neither response, appointment nor item. An attached file is part of the medical record, like its message: no role
-- deletes it, the owner included, but a rejected one (its bytes were never readable). Images get a thumbnail made by
-- the scan job from the clean file, stored and encrypted like the file under its own id (thumbnail_id, the object's
-- key) and data key.
--
-- Row-level security of the chat files, as the messages' (00041):
--   - the patient (app.patient_id) sees the files of their conversations, never of a doctor-only one;
--   - a scope with app.conversation_id set sees, inserts and attaches that conversation's files when it sees the
--     conversation (the clinic's needs the active care consent); an insert names one of its active participants and
--     the conversation's clinic and patient;
--   - the practitioner role's scope (app.chat_practitioner_role_id with app.organization_id, for the search across a
--     doctor's conversations) reads the files and messages of the conversations the role takes part in while the
--     consent is active, the messages after its history_from_seq only;
--   - the scan job's scope (app.binary_conversation_id with app.organization_id) reads and updates the files of that
--     one conversation, whatever the consent.
-- The questionnaire files' policies binary_own and binary_own_delete now name their kind of parent, so a patient sees
-- and deletes their questionnaire files exactly as before and a chat file only through the policies above.
--
-- Search: one text search configuration, chat_search, for every language of the app (they mix in one conversation):
-- the simple parser and dictionary, without stemming (Postgres has none for Bosnian, Croatian or Serbian, and one
-- language's stemmer would mangle the others' words), after unaccent, so a word typed without its diacritics finds
-- it. A GIN index on the body's vector, created on every partition, those made later included.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

ALTER TABLE "binary"
    ALTER COLUMN appointment_id DROP NOT NULL,
    ALTER COLUMN questionnaire_response_id DROP NOT NULL,
    ALTER COLUMN link_id DROP NOT NULL,
    ADD COLUMN conversation_id uuid REFERENCES conversation (id),
    -- The participant who uploaded it: only they attach it to a message.
    ADD COLUMN conversation_participant_id uuid REFERENCES conversation_participant (id),
    ADD COLUMN communication_id uuid,
    ADD COLUMN communication_sent_at timestamptz,
    ADD COLUMN thumbnail_id uuid UNIQUE,
    ADD COLUMN thumbnail_size bigint CHECK (thumbnail_size > 0),
    ADD COLUMN thumbnail_wrapped_key bytea,
    ADD CONSTRAINT binary_parent CHECK (
        (questionnaire_response_id IS NOT NULL AND appointment_id IS NOT NULL AND link_id IS NOT NULL
            AND conversation_id IS NULL AND conversation_participant_id IS NULL AND communication_id IS NULL)
        OR (questionnaire_response_id IS NULL AND appointment_id IS NULL AND link_id IS NULL
            AND conversation_id IS NOT NULL AND conversation_participant_id IS NOT NULL)),
    ADD CONSTRAINT binary_chat_type CHECK (conversation_id IS NULL OR content_type IN ('application/pdf', 'image/jpeg', 'image/png')),
    ADD CONSTRAINT binary_message CHECK ((communication_id IS NULL) = (communication_sent_at IS NULL)),
    ADD CONSTRAINT binary_message_fkey FOREIGN KEY (communication_id, communication_sent_at) REFERENCES communication (id, sent_at),
    ADD CONSTRAINT binary_thumbnail CHECK (
        (thumbnail_id IS NULL AND thumbnail_size IS NULL AND thumbnail_wrapped_key IS NULL)
        OR (thumbnail_id IS NOT NULL AND thumbnail_size IS NOT NULL AND thumbnail_wrapped_key IS NOT NULL AND status = 'clean'));

-- A page's files, by its messages; the purger's uploads never attached.
CREATE INDEX binary_conversation_message ON "binary" (conversation_id, communication_id) WHERE conversation_id IS NOT NULL;
CREATE INDEX binary_unattached ON "binary" (created_at) WHERE conversation_id IS NOT NULL AND communication_id IS NULL;

ALTER POLICY binary_own ON "binary"
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid AND questionnaire_response_id IS NOT NULL);
ALTER POLICY binary_own_delete ON "binary"
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid AND questionnaire_response_id IS NOT NULL);

CREATE POLICY binary_chat_own ON "binary" FOR SELECT
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid
           AND conversation_id IS NOT NULL
           AND EXISTS (SELECT 1 FROM conversation c WHERE c.id = "binary".conversation_id));
CREATE POLICY binary_chat ON "binary" FOR SELECT
    USING (conversation_id = NULLIF(current_setting('app.conversation_id', true), '')::uuid
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM conversation c WHERE c.id = NULLIF(current_setting('app.conversation_id', true), '')::uuid));
CREATE POLICY binary_chat_insert ON "binary" FOR INSERT
    WITH CHECK (conversation_id = NULLIF(current_setting('app.conversation_id', true), '')::uuid
                AND NULLIF(current_setting('app.account_id', true), '') IS NULL
                AND communication_id IS NULL
                AND EXISTS (SELECT 1 FROM conversation c
                            WHERE c.id = "binary".conversation_id
                              AND c.organization_id = "binary".organization_id
                              AND c.patient_id = "binary".patient_id)
                AND EXISTS (SELECT 1 FROM conversation_participant p
                            WHERE p.id = "binary".conversation_participant_id
                              AND p.conversation_id = "binary".conversation_id
                              AND p.removed_at IS NULL));
CREATE POLICY binary_chat_update ON "binary" FOR UPDATE
    USING (conversation_id = NULLIF(current_setting('app.conversation_id', true), '')::uuid
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM conversation c WHERE c.id = NULLIF(current_setting('app.conversation_id', true), '')::uuid));
CREATE POLICY binary_chat_practitioner ON "binary" FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND conversation_id IS NOT NULL
           AND EXISTS (SELECT 1 FROM conversation_participant p
                       WHERE p.conversation_id = "binary".conversation_id
                         AND p.practitioner_role_id = NULLIF(current_setting('app.chat_practitioner_role_id', true), '')::uuid
                         AND p.removed_at IS NULL));
CREATE POLICY binary_chat_job ON "binary" FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND conversation_id = NULLIF(current_setting('app.binary_conversation_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
CREATE POLICY binary_chat_job_update ON "binary" FOR UPDATE
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND conversation_id = NULLIF(current_setting('app.binary_conversation_id', true), '')::uuid
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);

-- The send attaches a file to its message; the scan job stores the thumbnail.
GRANT UPDATE (communication_id, communication_sent_at, thumbnail_id, thumbnail_size, thumbnail_wrapped_key) ON "binary" TO salus_app;

-- An attached file is part of the record (as its message, 00041): attached once, never deleted unless rejected.
-- +goose StatementBegin
CREATE FUNCTION binary_keep_attached() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.communication_id IS NOT NULL AND OLD.status <> 'rejected' THEN
            RAISE EXCEPTION 'binary: a file attached to a message is part of the medical record: DELETE is not allowed';
        END IF;
        RETURN OLD;
    END IF;
    IF OLD.communication_id IS NOT NULL AND (NEW.communication_id IS DISTINCT FROM OLD.communication_id
                                             OR NEW.communication_sent_at IS DISTINCT FROM OLD.communication_sent_at) THEN
        RAISE EXCEPTION 'binary: a file is attached to one message, once';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER binary_keep_attached BEFORE UPDATE OR DELETE ON "binary"
    FOR EACH ROW EXECUTE FUNCTION binary_keep_attached();

-- A text may go without a body when it carries files; a system message carries none.
ALTER TABLE communication
    ADD COLUMN attachment_count smallint DEFAULT 0 NOT NULL CHECK (attachment_count BETWEEN 0 AND 10),
    ADD CONSTRAINT communication_system_attachments CHECK (kind = 'text' OR attachment_count = 0),
    DROP CONSTRAINT communication_text,
    ADD CONSTRAINT communication_text CHECK (kind <> 'text' OR (sender_account_id IS NOT NULL AND sender_participant_id IS NOT NULL
        AND sender_role <> 'system' AND system_code IS NULL
        AND ((body IS NOT NULL AND char_length(body) BETWEEN 1 AND 4000) OR (body IS NULL AND attachment_count > 0))));

-- The practitioner role's scope, for the search across a doctor's conversations: read only.
CREATE POLICY communication_practitioner ON communication FOR SELECT
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM conversation_participant p
                       WHERE p.conversation_id = communication.conversation_id
                         AND p.practitioner_role_id = NULLIF(current_setting('app.chat_practitioner_role_id', true), '')::uuid
                         AND p.removed_at IS NULL
                         AND communication.seq > p.history_from_seq));

CREATE TEXT SEARCH CONFIGURATION public.chat_search (COPY = pg_catalog.simple);
ALTER TEXT SEARCH CONFIGURATION public.chat_search
    ALTER MAPPING FOR asciiword, word, numword, asciihword, hword, numhword, hword_asciipart, hword_part, hword_numpart
    WITH public.unaccent, pg_catalog.simple;

CREATE INDEX communication_search ON communication USING gin (to_tsvector('public.chat_search'::regconfig, body));
