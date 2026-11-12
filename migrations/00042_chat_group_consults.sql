-- +goose Up
-- Group consults (docs/plans/chat.md, SAL-28): the treating practitioner adds a doctor of their clinic, or invites a
-- doctor of another clinic, who takes part only once the patient consents (chat_consent, a FHIR Consent of its own:
-- that conversation and that practitioner role, revocable); doctor-only side threads (00041's staff_only and
-- parent_id) keep the participants they were opened with.
--
-- Row-level security of the guest, a doctor of another clinic (app.organization_id their own clinic,
-- app.practitioner_role_id their role, neither patient setting set): a conversation, and its side threads, only while
-- they are an active participant of it, the patient's chat consent for them is active and the patient's care consent
-- with the treating clinic is active. Like encounter_organization (00033), one SECURITY DEFINER function, owned by a
-- role that reads only the columns it needs, returns the ids of those conversations: the conversation's policy cannot
-- read its participants (their policy reads the conversation), and the guest's scope sees neither the treating
-- clinic's consents nor its conversations. The participants, messages and revisions follow the conversation through
-- their own policies (00041). The guest also reads their own chat consents and, while one is active, the patient's care
-- consents with that conversation's clinic, so the service decides as the database does.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

-- A practitioner participant names their clinic (the guest's is not the conversation's); an invitation across clinics
-- waits for the patient's consent; why a doctor was added is one of the contract's reasons.
ALTER TABLE conversation_participant
    ADD COLUMN organization_id uuid REFERENCES organization (id),
    ADD COLUMN status text DEFAULT 'active' NOT NULL CHECK (status IN ('active', 'pending_consent')),
    ADD COLUMN reason text CHECK (reason IN ('second_opinion', 'specialist_review', 'handover'));

UPDATE conversation_participant AS p SET organization_id = r.organization_id
FROM practitioner_role AS r
WHERE r.id = p.practitioner_role_id AND p.organization_id IS NULL;

ALTER TABLE conversation_participant
    ADD CONSTRAINT conversation_participant_organization CHECK ((role = 'practitioner') = (organization_id IS NOT NULL)),
    ADD CONSTRAINT conversation_participant_role_organization_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    ADD CONSTRAINT conversation_participant_pending CHECK (status = 'active' OR role = 'practitioner');

-- The patient's pending invitations.
CREATE INDEX conversation_participant_pending ON conversation_participant (conversation_id)
    WHERE status = 'pending_consent' AND removed_at IS NULL;

GRANT UPDATE (status) ON conversation_participant TO salus_app;

-- A practitioner removed from a conversation leaves its side threads in the same statement, whoever removes them: the
-- patient's scope, which revokes or declines, does not see the threads. SECURITY DEFINER, as the owner: it changes only
-- removed_at of the role's rows in the conversation's own threads.
-- +goose StatementBegin
CREATE FUNCTION conversation_participant_leave_threads() RETURNS trigger
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
BEGIN
    UPDATE public.conversation_participant AS t SET removed_at = NEW.removed_at
    WHERE t.removed_at IS NULL
      AND t.practitioner_role_id = NEW.practitioner_role_id
      AND t.conversation_id IN (SELECT c.id FROM public.conversation AS c WHERE c.parent_id = NEW.conversation_id);
    RETURN NULL;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER conversation_participant_removed_from_threads AFTER UPDATE OF removed_at ON conversation_participant
    FOR EACH ROW WHEN (OLD.removed_at IS NULL AND NEW.removed_at IS NOT NULL AND NEW.role = 'practitioner')
    EXECUTE FUNCTION conversation_participant_leave_threads();

-- The target of chat_consent's foreign key: a chat consent is the conversation's patient's, with its clinic.
ALTER TABLE conversation ADD CONSTRAINT conversation_patient UNIQUE (id, organization_id, patient_id);

-- The patient's consent that a practitioner role of another clinic, invited into their conversation with a clinic
-- (organization_id: the conversation's clinic, the custodian), reads and writes in it (FHIR Consent, provision.actor
-- the role, provision.data the conversation). At most one active per conversation and role; a grant after a
-- revocation or a removal adds a row, the history is proof (Art. 7(1)). Owned by consent.
CREATE TABLE chat_consent (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    date_time timestamptz NOT NULL, -- granted
    revoked_at timestamptz,
    patient_id uuid NOT NULL REFERENCES patient (id),
    organization_id uuid NOT NULL REFERENCES organization (id),
    conversation_id uuid NOT NULL,
    practitioner_role_id uuid NOT NULL REFERENCES practitioner_role (id),
    version_id integer DEFAULT 1 NOT NULL,
    status text NOT NULL CHECK (status IN ('active', 'inactive')),
    CONSTRAINT chat_consent_revoked_when_inactive CHECK ((status = 'inactive') = (revoked_at IS NOT NULL)),
    CONSTRAINT chat_consent_conversation_fkey FOREIGN KEY (conversation_id, organization_id, patient_id)
        REFERENCES conversation (id, organization_id, patient_id)
);

CREATE UNIQUE INDEX chat_consent_one_active ON chat_consent (conversation_id, practitioner_role_id) WHERE status = 'active';
CREATE INDEX chat_consent_patient ON chat_consent (patient_id, organization_id);
CREATE INDEX chat_consent_role ON chat_consent (practitioner_role_id) WHERE status = 'active';

ALTER TABLE chat_consent ENABLE ROW LEVEL SECURITY;
CREATE POLICY chat_consent_own ON chat_consent TO salus_app
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid);
CREATE POLICY chat_consent_clinic ON chat_consent FOR SELECT TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid);
-- The guest's own, while their role is of the clinic set (practitioner_role's policy shows only that clinic's roles).
CREATE POLICY chat_consent_guest ON chat_consent FOR SELECT TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND practitioner_role_id = NULLIF(current_setting('app.practitioner_role_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM practitioner_role r
                       WHERE r.id = chat_consent.practitioner_role_id
                         AND r.organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid));

-- No DELETE or TRUNCATE: kept as proof (retention keep). Who, with whom and about what never change; only the
-- revocation does.
GRANT SELECT, INSERT ON chat_consent TO salus_app;
GRANT UPDATE (status, revoked_at, updated_at, version_id) ON chat_consent TO salus_app;

-- The guest reads the patient's care consents with a clinic while a chat consent of theirs there is active (the
-- care consent rules every doctor's access to the chat).
CREATE POLICY consent_chat_guest ON consent FOR SELECT TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND EXISTS (SELECT 1 FROM chat_consent g
                       WHERE g.patient_id = consent.patient_id
                         AND g.organization_id = consent.organization_id
                         AND g.practitioner_role_id = NULLIF(current_setting('app.practitioner_role_id', true), '')::uuid
                         AND g.status = 'active'));

-- The owner of the guest's lookup: NOLOGIN, not a superuser, no BYPASSRLS. Cluster-wide like salus_app, so the same
-- DO block (test packages migrate in parallel).
-- +goose StatementBegin
DO $$
BEGIN
    CREATE ROLE salus_chat_guest NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEROLE NOCREATEDB;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'salus_chat_guest'
               AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolcanlogin)) THEN
        RAISE EXCEPTION 'role salus_chat_guest exists with SUPERUSER, BYPASSRLS, CREATEROLE, CREATEDB or LOGIN';
    END IF;
END
$$;
-- +goose StatementEnd

GRANT SELECT (id, organization_id, patient_id) ON conversation TO salus_chat_guest;
GRANT SELECT (conversation_id, practitioner_role_id, organization_id, status, removed_at) ON conversation_participant TO salus_chat_guest;
GRANT SELECT (conversation_id, practitioner_role_id, patient_id, organization_id, status) ON chat_consent TO salus_chat_guest;
GRANT SELECT (patient_id, organization_id, status) ON consent TO salus_chat_guest;
CREATE POLICY conversation_guest_lookup ON conversation FOR SELECT TO salus_chat_guest USING (true);
CREATE POLICY conversation_participant_guest_lookup ON conversation_participant FOR SELECT TO salus_chat_guest USING (true);
CREATE POLICY chat_consent_guest_lookup ON chat_consent FOR SELECT TO salus_chat_guest USING (true);
CREATE POLICY consent_guest_lookup ON consent FOR SELECT TO salus_chat_guest USING (true);

-- The conversations (never a side thread: those follow their conversation) the practitioner role of the clinic takes
-- part in as a guest whose access is granted. Nothing but their ids leaves the function.
CREATE FUNCTION chat_guest_conversations(role_id uuid, organization_id uuid) RETURNS uuid[]
    LANGUAGE sql
    STABLE
    SECURITY DEFINER
    SET search_path = pg_catalog, public
AS $$
    SELECT coalesce(array_agg(c.id), '{}')
    FROM public.conversation_participant AS p
    JOIN public.conversation AS c ON c.id = p.conversation_id
    WHERE p.practitioner_role_id = chat_guest_conversations.role_id
      AND p.organization_id = chat_guest_conversations.organization_id
      AND p.organization_id <> c.organization_id
      AND p.status = 'active'
      AND p.removed_at IS NULL
      AND EXISTS (SELECT 1 FROM public.chat_consent AS g
                  WHERE g.conversation_id = c.id
                    AND g.practitioner_role_id = p.practitioner_role_id
                    AND g.patient_id = c.patient_id
                    AND g.organization_id = c.organization_id
                    AND g.status = 'active')
      AND EXISTS (SELECT 1 FROM public.consent AS k
                  WHERE k.patient_id = c.patient_id
                    AND k.organization_id = c.organization_id
                    AND k.status = 'active')
$$;

ALTER FUNCTION chat_guest_conversations(uuid, uuid) OWNER TO salus_chat_guest;
REVOKE EXECUTE ON FUNCTION chat_guest_conversations(uuid, uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION chat_guest_conversations(uuid, uuid) TO salus_app;

-- The guest's conversations and their side threads: read, locked and counted by a send, and a side thread opened.
-- Planned once per query: the function's argument is the scope only.
CREATE POLICY conversation_guest ON conversation TO salus_app
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND coalesce(parent_id, id) = ANY ((SELECT chat_guest_conversations(
                   NULLIF(current_setting('app.practitioner_role_id', true), '')::uuid,
                   NULLIF(current_setting('app.organization_id', true), '')::uuid))::uuid[]));
