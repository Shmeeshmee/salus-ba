# Chat: progress

The running record of docs/plans/chat.md, one section per merged ticket, written at merge time.

## SAL-27: core messaging (step 0)

As built (agent decisions, for the owner to review; the plan's "Decisions" and ADR 0012 hold the rest):

| Topic | Decision |
| --- | --- |
| Domain | `internal/communication` (context `consultation`), four tables of migration 00041; the pure rules `access.OpensChat`, `ReadsChat`, `WritesChat` |
| Participants | The patient by their profile, a doctor by their practitioner role, as the rest of the schema names people; a message names its participant and the account signed in. No lookup from a role to an account was added |
| Staff after a revocation | `not_found` (404, socket close 4404), as the clinic's row-level security answers, so the service and the database agree; the patient keeps reading and gets `consent_required` for a write |
| Idempotency window | A repeat of a `client_message_id` is looked for within 24 hours of the conversation's last message: unbounded, the lookup probed every monthly partition (72 buffers, 11.5 ms cold at 24 months), bounded it reads one or two (2.7 ms). An id repeated a day later is a new message |
| Socket | The transcript's socket conventions moved to `pkg/livesocket` (the transcript uses it, its tests unchanged); the fan-out is `pkg/redis.Hub`, one Redis connection per api process |
| Edits while offline | The replay carries new messages; an edit made while a client was away shows on its next page load |
| Read position | Not audited: it is the reader's own, not the patient's data |
| Review (fresh subagent, Copilot out of quota) | Fixed: the socket re-checks the session every 5 s (a logout closes it, 4401); no participant's `read_seq` in the API; `Conversation.cursor` so a client opening an empty or changed conversation resumes without a gap; the hub makes no Redis call under its fan-out lock; revisions written only by the `SECURITY DEFINER` trigger (no INSERT for the app role); `pg_catalog, public` search paths; an idempotent repeat audited as a read; an edit frame never before its message; at most five socket sends at once. Declined: confirming SUBSCRIBE before the replay (a message published in that instant is late by up to the 5 s check, never lost: ADR 0012); the id lookups of an edit or its revisions read every partition's primary key (rare operations, a few ms), and a doctor's conversation list sorts their conversations on each page (hundreds per doctor, not measured at 50 000): SAL-30's search work measures both. The transcript's socket has the same once-only session check (pre-existing, not changed here) |
| Parallel stacks | `deployments/docker-compose.yml` takes the app image's tag from `SALUS_IMAGE_TAG` (default `dev`), so each worktree's stack builds its own image (configs/.env.example) |

### Measured (make bench-chat, 2026-11-12)

3 000 021 synthetic messages in 50 010 conversations over 24 monthly partitions (50 000 of 20 to 60 messages, 10 of
about 100 000), after `VACUUM (ANALYZE)` and a checkpoint; the compose Postgres 18.6 and Redis on a shared developer
host (28 cores, load average 10 to 14 from other projects during the runs); the test pool capped at 8 connections
(the api's has one per CPU). Every service is the api's own: the role lookup, the consent check and the audit write
are in the numbers.

| Measure | p50 | p95 | p99 | Target |
| --- | --- | --- | --- | --- |
| Latest page (50 messages), 1000 random conversations, one in ten of 100 000 messages | 9.6 ms | 39.9 ms | 99.0 ms | p95 < 200 ms: met |
| Page 22 of a 100 000-message conversation | 8.2 ms | 13.9 ms | 20.8 ms | met |
| A send, nothing else running | 8.2 ms | 12.3 ms | 15.8 ms | |
| Delivery, 200 conversations, 200 messages a second | 11.1 ms | 86.5 ms | 180 ms | p95 < 300 ms: met |
| Delivery, 400 messages a second (capacity) | 391 ms | 723 ms | 1.03 s | not met |
| Delivery, 1000 messages a second (capacity) | 405 ms | 775 ms | 977 ms | not met |

- The latest page's plan reads the newest partition and stops: the older 23 are never executed (0.8 to 3.5 ms in the
  database for a 100 000-message conversation).
- Delivery is the send's commit plus about 1 ms: the hub's fan-out costs nothing measurable. Above about 300 messages a
  second the backends wait on the CPU and on the WAL (`LWLock:WALWrite`, `IO:WalSync`: the flush of every commit of a
  durable write), on a shared host and a pool of 8. Not tuned further: 200 a second sustained is 720 000 messages an
  hour, far above what the clinics will send, and `synchronous_commit` stays on for the medical record. Runs on the
  same code varied by a factor of two with the host's load; earlier runs before the database was vacuumed after the
  seed had multi-second tails from its autovacuum and write-back.
- The load test asserts the targets at 200 messages a second and reports 400 and 1000 as capacity.

## SAL-30: attachments, read receipts, typing, search and terms

Built on SAL-27 (migration 00044; `internal/binary`, `internal/communication`, `internal/worker`, `internal/api`).
Routes: `POST /conversations/{id}/attachments`, `GET /conversations/{id}/attachments/{attachment_id}/content` and
`…/thumbnail`, `GET /conversations/{id}/search?q=`, `GET /conversations/search?q=`; `attachment_ids` on a send;
`ChatMessage.attachments` and `terms`; the socket's `read`, `typing` and `receipt` frames.

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| A file's second parent | A `binary` row has a questionnaire answer or a conversation (`conversation_id`, the uploading `conversation_participant_id`, then `communication_id` with its `sent_at` once sent), exactly one (a check). `communication` calls `binary`'s chat methods (`UploadChat`, `AttachChat`, `ChatFiles`, `ChatFile`, `OpenChat`); the chat decides who uploads, attaches and reads, and audits, in the transactions `binary` joins; `binary` keeps the file's rules (type from the bytes, size, encryption, scan, thumbnail). Not a second `Holder` as docs/attachments/SPEC.md section 7 foresaw: a chat holder would make `binary` and `communication` need each other at construction. The questionnaire files' policies, checks and tests are unchanged (`binary_own` and `binary_own_delete` now name their kind of parent) |
| What a chat takes | Photos (JPEG, PNG) and PDFs, 20 MB each, at most 10 per message; no video. An upload is the participant's alone until sent; one never sent is deleted after 24 h |
| Who reads a file | Whoever reads its message (enter; after their `history_from_seq`, the message then read to check it), once clean (409 `scanning` before, 404 when rejected); before it is sent, its uploader only. Row-level security mirrors the messages' (the patient never in a doctor-only thread, a clinic only with the consent active and the conversation set, the scan job its conversation only), tested over every scope |
| Infected or unscannable | clamd `FOUND`: rejected, key cleared, object deleted (as questionnaire files). A chat file past clamd's stream limit is rejected at once; any other scan failure retries, and the chat file's job gives up after 8 attempts (about 80 minutes of River's backoff), when the worker rejects it (`Unscannable`): never `scanning` for ever. A questionnaire file's job keeps its 25 attempts and stays quarantined, as before |
| A rejected file in its message | Listed with `status: rejected` (the contract's enum), never opened, no thumbnail; its row goes 24 h after the scan, then the message lists it no more |
| Retention of `binary` | Rule stays `delete`, 24 h, with the purger; the basis now says: a chat file attached to a message is part of the clinical record and kept with it (a trigger refuses deleting an attached file that is not rejected, for every role, the owner included), a chat upload never attached goes 24 h after its upload, and a thumbnail's object goes with its file's row |
| Thumbnails | Images only, made by the scan job from the clean bytes (decoded in the worker, at most 40 megapixels, else none), a JPEG of at most 320 px a side (quality 80), sealed under its own id and data key like a file; none for a PDF. The image's orientation tag is not applied (the apps send photos upright, docs/attachments/SPEC.md section 2) |
| Cheaper storage for older attachments | Deferred, not built: the production object store is not chosen (ADR 0010, docs/attachments/SPEC.md section 8) and Garage has no storage classes; the partitions by month and the files' rows by message make moving old months' objects a later job once the store is chosen |
| Receipts | The reader's own position stays stored (`read_seq`). A position that moves (`PUT …/read` or the `read` frame) goes to the conversation's channel as `receipt {participant_id, seq}` and into a Redis hash `chat:<id>:receipts` (never lower, 7-day TTL), so a socket opening later sends the kept receipts once it has replayed. Never in Postgres |
| Typing | The `typing` frame, at most once a second per participant (`SET NX` of `chat:<id>:typing:<participant>`, 1 s TTL; the socket drops faster frames before Redis), to the channel as `typing {participant_id}`; the apps show it 5 s after the last frame. The socket's own participant, no database call: its access is checked every 5 s as before. `read` counts against the send limit; `typing` frames are no longer `unknown_command` (the SAL-27 socket test that sent one now sends an unknown type) |
| Search configuration | `chat_search`: Postgres's `simple` parser and dictionary after `unaccent`, for every app language at once (they mix in a conversation): no stemming (Postgres has none for Bosnian, Croatian or Serbian, and one language's stemmer mangles another's words), case and diacritics aside so `mucnina` finds `mučnina`. Every word of the query must start a word of the body (prefix, all words). Latin and Cyrillic Serbian do not find each other (no transliteration) |
| Search index | A GIN index on `to_tsvector('chat_search', body)` on `communication`, so on every partition, those created later included (migration test) |
| Search through the index | Row-level security cannot use the text index: the match operator `@@` is not leakproof, so the policies run first over every message the scope sees (one search across a doctor's 50 010 conversations took over 24 s in `make bench-chat`, a rare word in a 100 000-message conversation 450 ms at p50). So `communication_search(q, conversation, lim)`, owned by the schema owner and `SECURITY DEFINER` like `communication_add_partitions`, finds the keys through the index and decides itself what the scope may read, as the policies do (migration test over every scope); the app then reads the rows by those keys through the policies, and the service checks each result's conversation too: three layers that agree. It plans each search for its own values (`EXECUTE`) |
| Search across a doctor's conversations | A read-only scope `app.chat_practitioner_role_id` (with the clinic's): its policy shows the messages of the conversations the role takes part in (not removed), after its `history_from_seq`, while the consent is active (the participant's policy reads the conversation's). The service also checks each result's conversation as `enter` does (participant, `access.ReadsChat`, history): the database and the service decide alike (the end-to-end test, on the owner's connection, caught a revoked conversation found by the role scope alone before this check). A patient's search uses their own scope; at most 50 results, newest first, audited as one search; the query is never logged |
| Terms | Tagged on read with `terminology`'s matching (SAL-13, as transcript lines): the body in the reader's UI language (their account's, else detected), then English for spans that overlap none; never stored, the body unchanged. On the JSON API's pages, searches and a send's or edit's answer; not in socket frames, which are the same bytes for every reader whatever their language. The reader's language is read through `auth.Service.Me`, which audits an account read per page |
| Message without text | `body` may be empty with files; the row's `body` is then null and `attachment_count` above 0 (a check); the JSON has no `body`. Such a message has no text to edit (`forbidden`) |
| Socket frames | Message frames (live and replay) carry `attachments` as at send time. Nothing tells a socket when a scan ends: the apps read the page again (see "Needs a contract change") |

Needs a contract change (reported, not made):

- An `attachment_scanned` frame (or a `message_edited` with the new statuses) when a file's scan ends, so the apps need not
  read the page again to see a thumbnail or a rejection. Today the scan job (`binary`, the worker) does not publish on
  the chat's channel.

Copy waiting for the owner: none. The backend adds no user-facing string; the new error codes (`scanning`,
`unsupported_type`, `too_large`, `length_required` on the chat's routes) are binary's, rendered by the apps (SAL-31).

### Measured (SAL-30, 2026-11-12)

`make bench-chat` on SAL-27's table (3 000 021 messages, 50 010 conversations, 24 monthly partitions), its bodies now
carrying one word of a synthetic vocabulary of 5 000 (`w0` to `w4999`: a word in about one message in 5 000, a prefix
such as `w12` in more) besides `placeholder`, in every message. Through the service: the role's scope, the service's
own check of every result, the audit. One practitioner takes part in all 50 010 conversations: the worst case (a
doctor has hundreds). The shared host ran at a load average of 30 to 95 from other sessions throughout.

| Measure | p50 | p95 | p99 |
| --- | --- | --- | --- |
| Search a 100 000-message conversation, a rare word | 28 ms | 59 ms | 110 ms |
| Search a 100 000-message conversation, a word in every message | 23 ms | 178 ms | 453 ms |
| Search a small conversation, a rare word | 35 ms | 83 ms | 433 ms |
| Search across 50 010 conversations, a rare word | 121 ms | 295 ms | 414 ms |
| Search across 50 010 conversations, a word in every message | 124 ms | 462 ms | 1.07 s |
| A 12-megapixel JPEG's thumbnail (decode, scale, encode; `BenchmarkThumbnail`) | 353 ms, 49 MB allocated | | |

- Before `communication_search` the same searches ran under row-level security alone: a rare word in a long
  conversation 451 ms at p50 (1.27 s p95), one search across the 50 010 conversations over 24 s (the run was stopped).
  The match operator is not leakproof, so the policies ran first over every message the scope sees. Before its
  windows (7 days, 90, all) a word in every message across them took 4.3 s at p50 (14.8 s p95).
- The SAL-27 measures in the same runs: latest page 9.3 to 11.8 ms p50, 25 ms p95 (target met); page 22 7 to 12 ms
  p50 (met); a send alone 11 to 19 ms p50. Delivery at 200 messages a second missed its target in both runs (p95 17 to
  21 s, the commits waiting on `LWLock:WALWrite` and `IO:WalSync`), and as much without the new GIN index (an A/B run
  with `communication_search` dropped after the seed: commit p95 24 s, delivery 4.8 s, load average 48): the host's
  load, not SAL-30's index. To be measured again on a quiet host before the merge; SAL-27's run had 86 ms p95 at a load
  average of 10 to 14.
