-- +goose Up
-- Chat between a patient and their clinic's doctors (docs/plans/chat.md, SAL-27; ADR 0012): conversations, their
-- participants and their messages (FHIR Communication), partitioned by month, never deleted; an edit keeps the
-- earlier body in communication_revision.
--
-- Row-level security (plan, "Row-level security"):
--   - the patient (app.patient_id) sees, opens and writes in their own conversations, never a doctor-only one;
--   - a clinic (app.organization_id, neither patient setting set) sees and writes its conversations and their
--     participants while the patient's care consent with it is active;
--   - a conversation's messages and revisions are seen by its patient, and in a scope with app.conversation_id
--     set to a conversation that scope sees. The service sets it after reading the conversation and checking the
--     caller. The EXISTS reads only the setting, so it is planned once per query, not once per row.
-- NULLIF: an unset setting reads as '' on a pooled connection and must match nothing.

CREATE TABLE conversation (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    last_sent_at timestamptz,
    organization_id uuid NOT NULL REFERENCES organization (id),
    patient_id uuid NOT NULL REFERENCES patient (id),
    practitioner_role_id uuid NOT NULL,
    encounter_id uuid REFERENCES encounter (id),
    parent_id uuid REFERENCES conversation (id),
    created_by uuid NOT NULL REFERENCES account (id),
    last_seq bigint DEFAULT 0 NOT NULL CHECK (last_seq >= 0),
    staff_only boolean DEFAULT false NOT NULL,
    CONSTRAINT conversation_practitioner_role_fkey FOREIGN KEY (practitioner_role_id, organization_id)
        REFERENCES practitioner_role (id, organization_id),
    -- A doctor-only side thread belongs to a conversation; a conversation with the patient belongs to none.
    CONSTRAINT conversation_side_thread CHECK (staff_only = (parent_id IS NOT NULL)),
    CONSTRAINT conversation_last_sent CHECK ((last_seq = 0) = (last_sent_at IS NULL)),
    -- The target of the messages' foreign key: a message copies these for its policies.
    CONSTRAINT conversation_scope UNIQUE (id, organization_id, patient_id, staff_only)
);

-- One conversation with the patient per patient, practitioner role and encounter (none counts as one).
CREATE UNIQUE INDEX conversation_one_per_care ON conversation (patient_id, practitioner_role_id, encounter_id)
    NULLS NOT DISTINCT WHERE NOT staff_only;
CREATE INDEX conversation_organization ON conversation (organization_id, last_sent_at DESC);
CREATE INDEX conversation_parent ON conversation (parent_id) WHERE parent_id IS NOT NULL;

ALTER TABLE conversation ENABLE ROW LEVEL SECURITY;
CREATE POLICY conversation_own ON conversation
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid AND NOT staff_only);
CREATE POLICY conversation_clinic ON conversation
    USING (NULLIF(current_setting('app.patient_id', true), '') IS NULL
           AND NULLIF(current_setting('app.account_id', true), '') IS NULL
           AND organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM consent c
                       WHERE c.patient_id = conversation.patient_id
                         AND c.organization_id = conversation.organization_id
                         AND c.status = 'active'));

GRANT SELECT, INSERT ON conversation TO salus_app;
GRANT UPDATE (last_seq, last_sent_at) ON conversation TO salus_app;

CREATE TABLE conversation_participant (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    added_at timestamptz DEFAULT now() NOT NULL,
    removed_at timestamptz,
    conversation_id uuid NOT NULL REFERENCES conversation (id),
    -- Who takes part: the conversation's patient, or a practitioner by their role (a doctor of another clinic
    -- too, SAL-28).
    patient_id uuid REFERENCES patient (id),
    practitioner_role_id uuid REFERENCES practitioner_role (id),
    added_by uuid NOT NULL REFERENCES account (id),
    -- The messages after this number are the participant's to read: 0 is the whole history (SAL-28).
    history_from_seq bigint DEFAULT 0 NOT NULL CHECK (history_from_seq >= 0),
    -- The participant's own read position (owner, 2026-11-12): unread counts and the e-mail fallback.
    read_seq bigint DEFAULT 0 NOT NULL CHECK (read_seq >= 0),
    role text NOT NULL CHECK (role IN ('patient', 'practitioner')),
    CONSTRAINT conversation_participant_role CHECK (CASE role WHEN 'patient' THEN patient_id IS NOT NULL AND practitioner_role_id IS NULL
        ELSE practitioner_role_id IS NOT NULL AND patient_id IS NULL END),
    CONSTRAINT conversation_participant_removed CHECK (removed_at IS NULL OR removed_at >= added_at)
);

-- Once each while active; the patient's row has no role, so there is one patient row.
CREATE UNIQUE INDEX conversation_participant_active ON conversation_participant (conversation_id, practitioner_role_id)
    NULLS NOT DISTINCT WHERE removed_at IS NULL;
-- A practitioner's and a patient's conversations.
CREATE INDEX conversation_participant_role ON conversation_participant (practitioner_role_id) WHERE removed_at IS NULL;
CREATE INDEX conversation_participant_patient ON conversation_participant (patient_id) WHERE removed_at IS NULL;

-- A participant is seen and written exactly where their conversation is.
ALTER TABLE conversation_participant ENABLE ROW LEVEL SECURITY;
CREATE POLICY conversation_participant_of_conversation ON conversation_participant
    USING (EXISTS (SELECT 1 FROM conversation c WHERE c.id = conversation_participant.conversation_id));

GRANT SELECT, INSERT ON conversation_participant TO salus_app;
GRANT UPDATE (read_seq, removed_at) ON conversation_participant TO salus_app;

CREATE TABLE communication (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    sent_at timestamptz NOT NULL,
    edited_at timestamptz,
    conversation_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    patient_id uuid NOT NULL,
    -- Who wrote: the participant, and the account signed in for accountability; neither for a system message.
    sender_participant_id uuid REFERENCES conversation_participant (id),
    sender_account_id uuid REFERENCES account (id),
    client_message_id uuid NOT NULL,
    seq bigint NOT NULL CHECK (seq > 0),
    version_id integer DEFAULT 1 NOT NULL CHECK (version_id > 0),
    staff_only boolean NOT NULL,
    sender_role text NOT NULL CHECK (sender_role IN ('patient', 'practitioner', 'system')),
    kind text NOT NULL CHECK (kind IN ('text', 'system')),
    body text,
    system_code text,
    system_args jsonb,
    PRIMARY KEY (id, sent_at),
    CONSTRAINT communication_conversation_fkey FOREIGN KEY (conversation_id, organization_id, patient_id, staff_only)
        REFERENCES conversation (id, organization_id, patient_id, staff_only),
    CONSTRAINT communication_text CHECK (kind <> 'text' OR (sender_account_id IS NOT NULL AND sender_participant_id IS NOT NULL
        AND sender_role <> 'system'
        AND char_length(body) BETWEEN 1 AND 4000 AND system_code IS NULL)),
    CONSTRAINT communication_system CHECK (kind <> 'system' OR (sender_account_id IS NULL AND sender_participant_id IS NULL
        AND sender_role = 'system'
        AND body IS NULL AND system_code IS NOT NULL AND system_code ~ '^[a-z_]{1,64}$')),
    CONSTRAINT communication_edit CHECK ((version_id = 1) = (edited_at IS NULL))
) PARTITION BY RANGE (sent_at);

-- A page of a conversation, newest first, and the next page by (sent_at, seq) (ADR 0012).
CREATE INDEX communication_page ON communication (conversation_id, sent_at, seq);
-- A repeated send (idempotency) is looked up by its client's id, under the conversation's lock.
CREATE INDEX communication_client ON communication (conversation_id, client_message_id);

ALTER TABLE communication ENABLE ROW LEVEL SECURITY;
CREATE POLICY communication_own ON communication
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid AND NOT staff_only);
CREATE POLICY communication_conversation ON communication
    USING (conversation_id = NULLIF(current_setting('app.conversation_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM conversation c
                       WHERE c.id = NULLIF(current_setting('app.conversation_id', true), '')::uuid));

GRANT SELECT, INSERT ON communication TO salus_app;
GRANT UPDATE (body, version_id, edited_at) ON communication TO salus_app;

CREATE TABLE communication_revision (
    communication_id uuid NOT NULL,
    sent_at timestamptz NOT NULL,
    edited_at timestamptz NOT NULL,
    conversation_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    patient_id uuid NOT NULL,
    version_id integer NOT NULL,
    staff_only boolean NOT NULL,
    body text NOT NULL,
    PRIMARY KEY (communication_id, version_id),
    FOREIGN KEY (communication_id, sent_at) REFERENCES communication (id, sent_at)
);

ALTER TABLE communication_revision ENABLE ROW LEVEL SECURITY;
CREATE POLICY communication_revision_own ON communication_revision
    USING (patient_id = NULLIF(current_setting('app.patient_id', true), '')::uuid AND NOT staff_only);
CREATE POLICY communication_revision_conversation ON communication_revision
    USING (conversation_id = NULLIF(current_setting('app.conversation_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM conversation c
                       WHERE c.id = NULLIF(current_setting('app.conversation_id', true), '')::uuid));

GRANT SELECT, INSERT ON communication_revision TO salus_app;

-- Never deleted, for every role, the owner included (as audit_event). TRUNCATE of a partition is refused by the
-- trigger each partition gets from the function below.
-- +goose StatementBegin
CREATE FUNCTION communication_reject_removal() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
BEGIN
    RAISE EXCEPTION '% is part of the medical record: % is not allowed', TG_TABLE_NAME, TG_OP;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER communication_no_delete BEFORE DELETE ON communication
    FOR EACH ROW EXECUTE FUNCTION communication_reject_removal();
CREATE TRIGGER communication_no_truncate BEFORE TRUNCATE ON communication
    FOR EACH STATEMENT EXECUTE FUNCTION communication_reject_removal();
CREATE TRIGGER communication_revision_no_change BEFORE UPDATE OR DELETE ON communication_revision
    FOR EACH ROW EXECUTE FUNCTION communication_reject_removal();
CREATE TRIGGER communication_revision_no_truncate BEFORE TRUNCATE ON communication_revision
    FOR EACH STATEMENT EXECUTE FUNCTION communication_reject_removal();

-- An edit: the next version of a text, with its time; the earlier body goes to communication_revision in the same
-- statement, as the editing role (its policies apply).
-- +goose StatementBegin
CREATE FUNCTION communication_keep_revision() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = public, pg_catalog
AS $$
BEGIN
    IF OLD.kind <> 'text' THEN
        RAISE EXCEPTION 'communication: a % message is never edited', OLD.kind;
    END IF;
    IF NEW.version_id <> OLD.version_id + 1 OR NEW.edited_at IS NULL THEN
        RAISE EXCEPTION 'communication: an edit is the next version, with its time';
    END IF;
    INSERT INTO communication_revision (communication_id, sent_at, edited_at, conversation_id, organization_id, patient_id,
        version_id, staff_only, body)
    VALUES (OLD.id, OLD.sent_at, coalesce(OLD.edited_at, OLD.sent_at), OLD.conversation_id, OLD.organization_id, OLD.patient_id,
        OLD.version_id, OLD.staff_only, OLD.body);
    RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER communication_revision_on_edit BEFORE UPDATE ON communication
    FOR EACH ROW EXECUTE FUNCTION communication_keep_revision();

-- communication_add_partitions creates the monthly partitions communication_yYYYYmMM from the month of from_day for
-- months months, skipping those that exist, each with the TRUNCATE trigger (a statement trigger is not inherited).
-- It runs as the schema owner, so the app role adds months without owning or altering a table (ADR 0012); the
-- partitions get no grants, so they are read only through communication and its policies.
-- +goose StatementBegin
CREATE FUNCTION communication_add_partitions(from_day date, months integer) RETURNS void
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public, pg_catalog
AS $$
DECLARE
    first date := date_trunc('month', from_day)::date;
    lo date;
    name text;
BEGIN
    IF months IS NULL OR months < 1 OR months > 24 THEN
        RAISE EXCEPTION 'communication_add_partitions: months must be 1 to 24';
    END IF;
    PERFORM pg_advisory_xact_lock(hashtext('communication_add_partitions'));
    FOR i IN 0 .. months - 1 LOOP
        lo := (first + make_interval(months => i))::date;
        name := format('communication_y%sm%s', to_char(lo, 'YYYY'), to_char(lo, 'MM'));
        CONTINUE WHEN to_regclass(name) IS NOT NULL;
        EXECUTE format('CREATE TABLE %I PARTITION OF communication FOR VALUES FROM (%L) TO (%L)',
            name, to_char(lo, 'YYYY-MM-DD') || ' 00:00:00+00', to_char((lo + interval '1 month')::date, 'YYYY-MM-DD') || ' 00:00:00+00');
        EXECUTE format('CREATE TRIGGER %I BEFORE TRUNCATE ON %I FOR EACH STATEMENT EXECUTE FUNCTION communication_reject_removal()',
            name || '_no_truncate', name);
    END LOOP;
END
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION communication_add_partitions(date, integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION communication_add_partitions(date, integer) TO salus_app;

SELECT communication_add_partitions('2026-11-01', 6);
