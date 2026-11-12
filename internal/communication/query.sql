-- Queries of the chat (migration 00041). The scope of the transaction decides what each one sees; none of them
-- checks access itself.

-- name: CreateConversation :one
-- A new conversation, unless the patient, role and encounter have their one already (no row then).
INSERT INTO conversation (organization_id, patient_id, practitioner_role_id, encounter_id, created_by, created_at)
VALUES (@organization_id, @patient_id, @practitioner_role_id, sqlc.narg(encounter_id), @created_by, @created_at)
ON CONFLICT (patient_id, practitioner_role_id, encounter_id) WHERE NOT staff_only DO NOTHING
RETURNING id, created_at, last_sent_at, organization_id, patient_id, practitioner_role_id, encounter_id, parent_id, created_by, last_seq, staff_only;

-- name: CareConversation :one
-- The one conversation with the patient for the role and encounter.
SELECT id, created_at, last_sent_at, organization_id, patient_id, practitioner_role_id, encounter_id, parent_id, created_by, last_seq, staff_only
FROM conversation
WHERE patient_id = @patient_id AND practitioner_role_id = @practitioner_role_id
  AND encounter_id IS NOT DISTINCT FROM sqlc.narg(encounter_id) AND NOT staff_only;

-- name: InsertParticipant :one
INSERT INTO conversation_participant (conversation_id, patient_id, practitioner_role_id, role, history_from_seq, added_by, added_at)
VALUES (@conversation_id, sqlc.narg(patient_id), sqlc.narg(practitioner_role_id), @role, @history_from_seq, @added_by, @added_at)
RETURNING id, added_at, removed_at, conversation_id, patient_id, practitioner_role_id, added_by, history_from_seq, read_seq, role;

-- name: Conversation :one
SELECT id, created_at, last_sent_at, organization_id, patient_id, practitioner_role_id, encounter_id, parent_id, created_by, last_seq, staff_only
FROM conversation WHERE id = @id;

-- name: ConversationForUpdate :one
-- The conversation, locked until the transaction ends: a send's number and time are decided under it.
SELECT id, created_at, last_sent_at, organization_id, patient_id, practitioner_role_id, encounter_id, parent_id, created_by, last_seq, staff_only
FROM conversation WHERE id = @id
FOR UPDATE;

-- name: Participants :many
-- The conversation's active participants, the patient first, then in the order they were added.
SELECT id, added_at, removed_at, conversation_id, patient_id, practitioner_role_id, added_by, history_from_seq, read_seq, role
FROM conversation_participant
WHERE conversation_id = @conversation_id AND removed_at IS NULL
ORDER BY role = 'patient' DESC, added_at, id;

-- name: ConversationsOfParticipant :many
-- The conversations the patient or the practitioner role takes part in (one of the two is null), newest activity
-- first, before the cursor when there is one, with that participant's read position.
SELECT c.id, c.created_at, c.last_sent_at, c.organization_id, c.patient_id, c.practitioner_role_id, c.encounter_id, c.parent_id,
       c.created_by, c.last_seq, c.staff_only, p.read_seq
FROM conversation c
JOIN conversation_participant p ON p.conversation_id = c.id AND p.removed_at IS NULL
WHERE (p.patient_id = sqlc.narg(patient_id) OR p.practitioner_role_id = sqlc.narg(practitioner_role_id))
  AND (sqlc.narg(before_at)::timestamptz IS NULL
       OR (coalesce(c.last_sent_at, c.created_at), c.id) < (sqlc.narg(before_at)::timestamptz, sqlc.narg(before_id)::uuid))
ORDER BY coalesce(c.last_sent_at, c.created_at) DESC, c.id DESC
LIMIT @lim;

-- name: LatestMessages :many
-- The newest page: the partitions are read newest first and the scan stops once the page is full (ADR 0012).
SELECT id, sent_at, edited_at, conversation_id, organization_id, patient_id, sender_participant_id, sender_account_id,
       client_message_id, seq, version_id, staff_only, sender_role, kind, body, system_code, system_args, attachment_count
FROM communication
WHERE conversation_id = @conversation_id AND seq > @from_seq
ORDER BY sent_at DESC, seq DESC
LIMIT @lim;

-- name: MessagesBefore :many
SELECT id, sent_at, edited_at, conversation_id, organization_id, patient_id, sender_participant_id, sender_account_id,
       client_message_id, seq, version_id, staff_only, sender_role, kind, body, system_code, system_args, attachment_count
FROM communication
WHERE conversation_id = @conversation_id AND seq > @from_seq AND (sent_at, seq) < (@sent_at::timestamptz, @seq::bigint)
ORDER BY sent_at DESC, seq DESC
LIMIT @lim;

-- name: MessagesAfter :many
SELECT id, sent_at, edited_at, conversation_id, organization_id, patient_id, sender_participant_id, sender_account_id,
       client_message_id, seq, version_id, staff_only, sender_role, kind, body, system_code, system_args, attachment_count
FROM communication
WHERE conversation_id = @conversation_id AND seq > @from_seq AND (sent_at, seq) > (@sent_at::timestamptz, @seq::bigint)
ORDER BY sent_at, seq
LIMIT @lim;

-- name: Message :one
SELECT id, sent_at, edited_at, conversation_id, organization_id, patient_id, sender_participant_id, sender_account_id,
       client_message_id, seq, version_id, staff_only, sender_role, kind, body, system_code, system_args, attachment_count
FROM communication
WHERE conversation_id = @conversation_id AND id = @id;

-- name: MessageByClientID :one
-- A repeat of the client's id within the idempotency window: run-time pruning reads that day's partitions only.
SELECT id, sent_at, edited_at, conversation_id, organization_id, patient_id, sender_participant_id, sender_account_id,
       client_message_id, seq, version_id, staff_only, sender_role, kind, body, system_code, system_args, attachment_count
FROM communication
WHERE conversation_id = @conversation_id AND client_message_id = @client_message_id AND sender_participant_id = @sender_participant_id
  AND sent_at >= @since::timestamptz;

-- name: InsertMessage :one
INSERT INTO communication (sent_at, conversation_id, organization_id, patient_id, sender_participant_id, sender_account_id,
    client_message_id, seq, staff_only, sender_role, kind, body, system_code, system_args, attachment_count)
VALUES (@sent_at, @conversation_id, @organization_id, @patient_id, sqlc.narg(sender_participant_id), sqlc.narg(sender_account_id),
    @client_message_id, @seq, @staff_only, @sender_role, @kind, sqlc.narg(body), sqlc.narg(system_code), sqlc.narg(system_args), @attachment_count)
RETURNING id;

-- name: AdvanceConversation :exec
UPDATE conversation SET last_seq = @last_seq, last_sent_at = @last_sent_at WHERE id = @id;

-- name: MarkRead :exec
UPDATE conversation_participant SET read_seq = greatest(read_seq, @read_seq) WHERE id = @id;

-- name: UpdateBody :execrows
-- The next version of the body; no row when the stored version is not the one before it.
UPDATE communication SET body = @body, version_id = @version_id, edited_at = @edited_at
WHERE conversation_id = @conversation_id AND id = @id AND version_id = @previous_version;

-- name: Revisions :many
SELECT edited_at, body, version_id
FROM communication_revision
WHERE conversation_id = @conversation_id AND communication_id = @communication_id
ORDER BY version_id;

-- name: AddPartitions :exec
SELECT communication_add_partitions(@from_day::date, @months::integer);

-- name: SearchConversation :many
-- A conversation's messages whose body matches the query (chat_search, migration 00044), newest first, of numbers
-- above from_seq: found by communication_search (the text index, which row-level security cannot use), read by
-- their keys through the policies.
SELECT id, sent_at, edited_at, conversation_id, organization_id, patient_id, sender_participant_id, sender_account_id,
       client_message_id, seq, version_id, staff_only, sender_role, kind, body, system_code, system_args, attachment_count
FROM communication
WHERE (id, sent_at) IN (SELECT s.id, s.sent_at FROM communication_search(to_tsquery('public.chat_search'::regconfig, @query), @conversation_id::uuid, @lim::integer) s)
  AND conversation_id = @conversation_id AND seq > @from_seq
ORDER BY sent_at DESC, seq DESC;

-- name: SearchMessages :many
-- The messages the scope may read whose body matches the query, newest first: the patient's own conversations, or
-- those of the practitioner role's scope after its history_from_seq; found by communication_search, read through the
-- policies.
SELECT id, sent_at, edited_at, conversation_id, organization_id, patient_id, sender_participant_id, sender_account_id,
       client_message_id, seq, version_id, staff_only, sender_role, kind, body, system_code, system_args, attachment_count
FROM communication
WHERE (id, sent_at) IN (SELECT s.id, s.sent_at FROM communication_search(to_tsquery('public.chat_search'::regconfig, @query), NULL, @lim::integer) s)
ORDER BY sent_at DESC, seq DESC;
