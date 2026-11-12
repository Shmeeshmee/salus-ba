# 12. Chat: monthly-partitioned messages, cursor pages, never deleted, the WebSocket conventions reused

Status: accepted, 2026-11-12 (docs/plans/chat.md, SAL-27)

## Context

Patients and their clinic's doctors message each other before and after visits (epic SAL-26). The messages are part
of the medical record: kept with it, exported with it, never removed by a user. A busy clinic writes thousands a
day, and the table will hold millions within a year; opening a conversation must still load its latest page under
200 ms (p95), and a message must reach the other side under 300 ms (p95).

The transcript already streams live events to the browser (docs/plans/consult-intelligence.md, "Gateway"):
`GET /encounters/{id}/events`, a gorilla/websocket socket carrying `pkg/wire` envelopes, a cursor the client
resumes from with `?after=`, the same-origin check, pings every 25 s, close codes 4403 and 4404. Each socket holds
one `XREAD BLOCK` on the encounter's Redis Stream, so one Redis connection per open socket. That suits the few live
consultations at a time; chat sockets are open whenever a patient or a doctor has the app in front of them.

## Decision

**Storage.** One table `communication` (FHIR `Communication`), range-partitioned by month on `sent_at`, primary key
`(id, sent_at)`. Every message carries its conversation's `organization_id`, `patient_id` and `staff_only`, so the
row-level security policies are equalities and need no join per row. A conversation's messages are numbered by `seq`
(1, 2, 3 … without gaps), assigned under the conversation's row lock in the send's transaction, and `sent_at` never
goes back within a conversation, so `(sent_at, seq)` and `seq` give the same order.

**Pagination by cursor.** A cursor is `(sent_at, seq)` of a message, opaque to the client. The latest page is
`WHERE conversation_id = $1 ORDER BY sent_at DESC, seq DESC LIMIT n` on the index `(conversation_id, sent_at, seq)`;
the next page adds `AND (sent_at, seq) < ($2, $3)`. Postgres reads the partitions in order, newest first, and stops
once the page is full, so the cost depends on the page, not on the size of the table or the depth of the history.
No `OFFSET`.

**Never deleted.** No role has `DELETE` on `communication` or `communication_revision`, and triggers reject
`DELETE` and `TRUNCATE` for every role, the owner included, as for `audit_event`. An edit changes only the body,
`version_id` and `edited_at` (column grants); a trigger copies the earlier body into `communication_revision` in the
same statement. Retention keeps them with the clinical record.

**Partitions.** `communication_add_partitions(from, months)`, owned by the schema owner and `SECURITY DEFINER`,
creates the missing months; the migration runs it for the first six months, the scheduler's daily job keeps three
months ahead. There is no default partition: a missing month fails a send loudly, where a default partition would
fill and then block the creation of that month's partition.

**Delivery.** The transcript's socket conventions move to `pkg/livesocket` (the upgrade's origin check, one writer at
a time, pings, the read loop with its limit, the close codes), used by the transcript and by the chat's
`GET /conversations/{id}/events`. The fan-out is not one Redis Stream read per socket but Redis pub/sub: one
subscription connection per api process (`pkg/redis.Hub`) hands each `chat:<conversation_id>` message to that
process's open sockets of the conversation. Postgres is the record and Redis only transport (ADR 0005): a socket
replays from Postgres every message after the client's cursor, then forwards live ones; a gap in `seq` (pub/sub drops
messages when Redis or a slow socket falls behind) or the socket's check every 5 s fills from Postgres. The client
deduplicates by `seq`.

**Idempotent send.** The client names each message with a UUID, `client_message_id`. Under the conversation's lock
the send looks it up first, among the conversation's messages of the last 24 hours before its latest one
(`IdempotencyWindow`); a repeat returns the first message and publishes nothing. The window keeps the lookup to one or
two partitions: unbounded, it probed every month the table has (72 buffers in 24 months at 3 million messages, 11 ms
cold, measured), and grew with every month. A client retries within seconds; an id repeated more than a day later is
a new message.

## Consequences

- A conversation's sends are serialised by its row lock. A chat is a few people typing: the lock is held for the
  length of one insert and two updates.
- Unique constraints on a partitioned table must include `sent_at`, so the idempotency of `client_message_id` and
  the uniqueness of `seq` rest on the lock, not on a constraint. Tests cover concurrent sends and repeats.
- A repeated `client_message_id` is found within a day of the last message only (above): the index has no `sent_at`,
  so the bound on `sent_at` is what limits the partitions read.
- The api processes each hold one more Redis connection for the subscriptions, independent of the number of
  sockets.
- An edit made while a client was offline reaches it on its next page load; the replay carries new messages only.
- The partitions are created by a function, not by the app role directly, so `salus_app` never owns or alters a
  table.

## Alternatives

- **One unpartitioned table.** Simpler, and the index alone would keep the latest page fast. Partitioning keeps each
  month's index small and in memory, and lets a future archive or tiered storage move old months as a whole. The
  owner chose partitioning (brief of 2026-11-12).
- **Redis Streams per socket, as the transcript.** One connection per socket: does not scale to a chat's number of
  idle sockets. A single reader of many streams would need its own cursor bookkeeping; pub/sub with a Postgres
  replay is simpler for the same guarantee.
- **Postgres `LISTEN/NOTIFY`.** No extra service, but one more long-lived database connection per api process and a
  payload limit of 8000 bytes; Redis is already the live channel (ADR 0005).
- **An idempotency table with a real unique constraint.** One more write per message for a guarantee the lock already
  gives.
