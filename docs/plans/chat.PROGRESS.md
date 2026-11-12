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

## SAL-28: group consults

Migration 00044; the pure rules `access.GuestChat`, `ManagesChat`, `OpensChatThread`, `GrantsChatGuest`; in
`internal/communication` the new `service_participants.go`, `service_threads.go`, `handler_participants.go`
(`GroupHandler`, mounted beside `HTTPHandler` in `internal/api/chat.go`); in `internal/consent` the chat consent
(`KindChat`, `GrantChat`, `RevokeChat`, `ChatGranted`, table `chat_consent`).

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| Invitation | A participant row with `status` `pending_consent` (no table of its own): the add and the invitation are one path, the patient's invitations are a query on it, its id is the invitation's id in `/me/chat-invitations/{id}` |
| The patient's consent | Table `chat_consent` owned by `consent` (FHIR Consent, `KindChat`): one conversation and one practitioner role, at most one active; the patient's own, read by the conversation's clinic and by the invited doctor. Not in `GET /consents` nor the FHIR Consent reads (they list care and transcription consents): the patient sees and revokes it through the conversation's participants and `/me/chat-invitations/{id}/revoke`; the subject-rights export will carry it. Lawful basis noted as Art. 9(2)(a) |
| A decline | Ends the invitation (the participant is removed), audited, with the system message `consent_declined`; no Consent row is written for it. It needs no active care consent; a grant does (`consent_required`) |
| A removal by the treating practitioner | Ends the doctor's access and, for a granted doctor of another clinic, the patient's chat consent for them, in the same transaction: the clinic ends it (`consent.EndChat`, audited; policy `chat_consent_clinic_end` lets the clinic set an active one inactive and nothing else). Changed in the review (it first left the consent active, which the patient could then neither see as granted nor revoke). A new invitation of the same role asks the patient again |
| The guest's scope | `app.organization_id` their own clinic plus `app.practitioner_role_id` their role, set by the service for every practitioner. Policy `conversation_guest` (to `salus_app`) asks `chat_guest_conversations(role, clinic)`, a `SECURITY DEFINER` function owned by the NOLOGIN role `salus_chat_guest`, which reads only the columns it needs: the conversations where the role of that clinic is an active participant of another clinic with the chat consent and the care consent active. Of their side threads, only those the role takes part in (`chat_guest_in_thread`, per row: an active participant, or no participant yet inside the opening transaction, which inserts the opener first); a restrictive policy hides the conversation's pending invitations of other doctors from the guest (both from the review). The guest also reads their own chat consents and, while one is active, the patient's care consents with that conversation's clinic (`consent_chat_guest`), so the service checks `access.GuestChat` on the same facts. Migration tests over every state (pending, granted, care revoked, consent revoked, removed) and scope (no role, no clinic, another clinic or role, a patient or account set) |
| No policy reads another through row-level security in a cycle | `chat_consent` stores the invited role's clinic (`practitioner_organization_id`, copied at the grant, held by a foreign key to `practitioner_role (id, organization_id)`), so its guest policy compares columns: reading `practitioner_role` there recursed, after main's 00043, through `practitioner_role_shared`, `encounter` and `appointment` back to `consent` (test: appointments, roles, encounters, consents, chat consents and conversations read in a clinic's, a guest's and a patient's scope, red on the old policy in all three) |
| The guest's care consents | `consent_chat_guest` asks `chat_guest_care_consents(role, clinic)`, a `SECURITY DEFINER` function owned by `salus_chat_guest` (like `chat_guest_conversations`), planned once per query. An `EXISTS` over `chat_consent` in the policy was repeated at every place appointment's and encounter's policies read a consent, and after main's 00043 it lifted a directory read over `jit_optimize_above_cost` (500 000). Measured: the listed schedules' read (the hourly slot generation, `internal/worker` test) took 81.7 s with JIT and 24 ms without; plan cost of `SELECT FROM schedule` in the directory scope is 337 000 on main, 538 000 with the `EXISTS` and 346 000 with the function. The migration test `ThePoliciesKeepADirectoryReadUnderJITsOptimizeCost` holds the limit. Main alone is at two thirds of it: for the owner, whether `jit = off` for `salus_app` (or a slimmer `practitioner_role_shared`) should follow |
| Refusals to the guest | `not_found` on every path while not granted, also for a side thread of a granted conversation they are not in: nothing of the chat is disclosed to another clinic |
| Who adds and removes | The treating practitioner (`conversation.practitioner_role_id`) while they may write; never into a side thread, never the patient or themselves. A doctor of the clinic must be an active role of it (`practitionerrole.Lookup`), one of another clinic in its public directory (`PublicByOrganization`: listed clinic, active role and practitioner), read before the conversation's transaction. `already_participant` (409) for a role there or invited already; a role that may not be added is `invalid_request` |
| History | `full` is `history_from_seq` 0; `from` takes `from_seq` 0 to `last_seq`; the added doctor's read position starts there, so their unread count is what they may read |
| Side threads | A side thread keeps the doctors it was opened with: the conversation's active practitioners at that moment (pending ones not). A doctor added later is not in it; one removed from the conversation leaves its side threads in the same statement (a `SECURITY DEFINER` trigger, as the patient's scope that revokes does not see the threads). Opened with no body (`POST /conversations/{id}/threads`), listed as a `ConversationList` without `next`; its practitioner role is the conversation's treating one (the foreign key wants a role of the clinic), its encounter the conversation's. The patient gets 403 on the list and the creation (they know the conversation), 404 on the thread itself |
| System messages | Written through `appendMessage`, which `Send` now uses too (the same lock, numbering and time), published after the commit like a text. A revocation by the patient writes `participant_removed`, the contract's code: the UI cannot tell it from a removal by the doctor (see "Reported") |
| Audit | Entity `conversation_participant` for every add, invitation, removal, decision and revocation, referenced as `ConversationParticipant/<id>` (the participant changed, an opaque id), and for the invitations' list; a revocation of no active chat consent commits the consent domain's refusal audit; the consent service audits its Consent create and update in the same transaction; a side thread's creation and list as `conversation` |
| Decision and revocation answers | 204 without a body |
| Additions to the contract | `ConversationParticipant.reason`; `AddParticipantRequest`, `ChatInvitation`, `ChatInvitationList`, `ChatInvitationDecision` schemas; error `already_participant` (409); from the review, `GET /me/chat-invitations?status=pending_consent\|active` and `ChatInvitation.status` |
| Concurrent changes of one invitation | A decision or revocation re-reads the invitation under the conversation's lock, and activating or removing a participant whose row did not change is `not_found`: a grant racing a removal, two grants, a revocation racing a removal (tests on Postgres for each) |

Review (fresh subagent): fixed 1 (a removal ends the chat consent; the guest's care consents need an active
participant), 2 (races of grant, removal and revocation), 3 (granted invitations listed for revocation), 5 (a guest's
side threads), 6 (other doctors' pending invitations hidden from a guest), 7 (the participant in the audit), 8 (the
refused revocation's audit kept), and 4 in the plan's contract text (`from_seq` is the last message not seen, the
backend kept); declined none.

### Measured (2026-11-12)

The guest's policy runs for every practitioner's query on `conversation` (the service sets their role). A first
version of `chat_guest_conversations` started from the role's participant rows: `make bench-chat`'s doctor takes part
in 50 000 conversations, and every send of theirs read them all (delivery at 200 a second: p95 18.9 s). It now starts
from the role's active chat consents (none for a clinic's own doctors). Measured on the shared host (load average 14
to 28 from other projects):

| Measure | p50 | p95 |
| --- | --- | --- |
| A practitioner's transaction (lock the conversation, read its messages) with 50 000 participant rows of their role, without `app.practitioner_role_id` (1000 runs, interleaved) | 6.67 ms | 14.7 ms |
| The same with the role set (the guest's policy evaluated) | 6.76 ms | 14.1 ms |
| `make bench-chat`, this branch, three runs: latest page | 9 to 15 ms | 19 to 65 ms |
| `make bench-chat`, this branch: a send with nothing else running | 8 to 12 ms | 18 to 96 ms |
| `make bench-chat`, this branch: delivery at 200 a second | 16 to 19 ms | 0.48 to 5.1 s |
| `make bench-chat`, `main` (bfc2901), one run, same host: delivery at 200 a second | 12 ms | 33 ms |

The isolated measure shows no cost of the policy. The load test's 200-a-second phase failed its target in all three
runs of this branch and passed in the one run of `main`, while its 400-a-second phase passed on this branch once (p95
57 ms) and failed on `main` (1.5 s): the host's load decides those tails (the backends wait on `LWLock:WALWrite` and
`IO:WalSync`, as SAL-27 found). Not settled here: the delivery target should be measured again on a quiet host before
the merge.

Reported, not changed:

- The plan's contract says each wave-1 package adds its methods to the web `ChatApi`, the mobile `Chats` and their
  fakes; this package is backend only (the brief), so they are SAL-31's. The generated clients are regenerated, and
  the two fakes give their participants the new required `status`.
- A revocation by the patient has no system code of its own (`participant_removed` is used); a `consent_revoked` code
  would let the UI say who ended the access.

Copy waiting for the owner: none. The API returns codes only; the UI's strings for the five system codes and the
invitation are SAL-31's.

Shared files changed: `internal/communication/{model,errors,repository,repository_postgres,service,service_messages,
service_open,query}.go|sql`, `fakes_test.go`, `communicationdb`, every `*db/models.go` (sqlc), `internal/api/chat.go`,
`internal/api/chat_load_test.go` (seed), `api/openapi.yaml`, `api/chat_contract_test.go`, the generated clients,
`web/src/chat/fakeChat.ts`, `mobile/test/fake_chat.dart`, `migrations/chat_test.go` (one case gives its practitioner a
clinic, so it still fails for the reason it names), `ownership.yaml`, `retention/rules.go`, `docs/privacy/dpia.md`.

## SAL-30: attachments, read receipts, typing, search and terms

Built on SAL-27 and SAL-28 (migration 00045; `internal/binary`, `internal/communication`, `internal/worker`, `internal/api`).
Routes: `POST /conversations/{id}/attachments`, `GET /conversations/{id}/attachments/{attachment_id}/content` and
`…/thumbnail`, `GET /conversations/{id}/search?q=`, `GET /conversations/search?q=`; `attachment_ids` on a send;
`ChatMessage.attachments` and `terms`; the socket's `read`, `typing` and `receipt` frames.

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| A file's second parent | A `binary` row has a questionnaire answer or a conversation (`conversation_id`, the uploading `conversation_participant_id`, then `communication_id` with its `sent_at` once sent), exactly one (a check). `communication` calls `binary`'s chat methods (`UploadChat`, `AttachChat`, `ChatFiles`, `ChatFile`, `OpenChat`); the chat decides who uploads, attaches and reads, and audits, in the transactions `binary` joins; `binary` keeps the file's rules (type from the bytes, size, encryption, scan, thumbnail). Not a second `Holder` as docs/attachments/SPEC.md section 7 foresaw: a chat holder would make `binary` and `communication` need each other at construction. The questionnaire files' policies, checks and tests are unchanged (`binary_own` and `binary_own_delete` now name their kind of parent) |
| What a chat takes | Photos (JPEG, PNG) and PDFs, 20 MB each, at most 10 per message; no video. An upload is the participant's alone until sent; one never sent is deleted after 24 h. A participant has at most 10 uploads waiting (neither sent nor rejected, `binary.MaxPendingChat`): the next is refused `too_many_files` (409), audited; their uploads count and insert under an advisory lock (review, finding 3). No rate limit: none exists in the repo to reuse for uploads |
| Who reads a file | Whoever reads its message (enter; after their `history_from_seq`, the message then read to check it), once clean (409 `scanning` before, 404 when rejected); before it is sent, its uploader only. Row-level security mirrors the messages' (the patient never in a doctor-only thread, a clinic only with the consent active and the conversation set, the scan job its conversation only), tested over every scope |
| Infected or unscannable | clamd `FOUND`: rejected, key cleared, object deleted (as questionnaire files). A chat file past clamd's stream limit is rejected at once; any other scan failure retries, and the chat file's job gives up after 8 attempts (about 80 minutes of River's backoff), when the worker rejects it (`Unscannable`): never `scanning` for ever. A questionnaire file's job keeps its 25 attempts and stays quarantined, as before |
| A rejected file in its message | Listed with `status: rejected` (the contract's enum), never opened, no thumbnail; its row goes 24 h after the scan, then the message lists it no more |
| Retention of `binary` | Rule stays `delete`, 24 h, with the purger; the basis now says: a chat file attached to a message is part of the clinical record and kept with it (a trigger refuses deleting an attached file that is not rejected, for every role, the owner included), a chat upload never attached goes 24 h after its upload, and a thumbnail's object goes with its file's row |
| Thumbnails | Images only, made by the scan job from the clean bytes (decoded in the worker, at most 40 megapixels and at most 192 MiB of decode as `decodeCost` reckons it from the header by format, else none; one decode at a time per worker; review, finding 1), a JPEG of at most 320 px a side (quality 80), sealed under its own id and data key like a file; none for a PDF. The image's orientation tag is not applied (the apps send photos upright, docs/attachments/SPEC.md section 2). A thumbnail that fails (the store) leaves the file clean without one, logged as `thumbnail_error` (review, finding 5) |
| Cheaper storage for older attachments | Deferred, not built: the production object store is not chosen (ADR 0010, docs/attachments/SPEC.md section 8) and Garage has no storage classes; the partitions by month and the files' rows by message make moving old months' objects a later job once the store is chosen |
| Receipts | The reader's own position stays stored (`read_seq`). A position that moves (`PUT …/read` or the `read` frame) goes to the conversation's channel as `receipt {participant_id, seq}` and into Redis `chat:<id>:receipt:<participant>` (never lower, 7 days after that participant's last read, each key its own TTL), so a socket opening later sends the kept receipts of the conversation's current participants once it has replayed (a removed participant's never; review, finding 4). Never in Postgres |
| Typing | The `typing` frame, at most once a second per participant (`SET NX` of `chat:<id>:typing:<participant>`, 1 s TTL; the socket drops faster frames before Redis), to the channel as `typing {participant_id}`; the apps show it 5 s after the last frame. The socket's own participant, no database call: its access is checked every 5 s as before. `read` counts against the send limit; `typing` frames are no longer `unknown_command` (the SAL-27 socket test that sent one now sends an unknown type) |
| Search configuration | `chat_search`: Postgres's `simple` parser and dictionary after `unaccent`, for every app language at once (they mix in a conversation): no stemming (Postgres has none for Bosnian, Croatian or Serbian, and one language's stemmer mangles another's words), case and diacritics aside so `mucnina` finds `mučnina`. Every word of the query must start a word of the body (prefix, all words). Latin and Cyrillic Serbian do not find each other (no transliteration) |
| Search index | A GIN index on `to_tsvector('chat_search', body)` on `communication`, so on every partition, those created later included (migration test) |
| Search through the index | Row-level security cannot use the text index: the match operator `@@` is not leakproof, so the policies run first over every message the scope sees (one search across a doctor's 50 010 conversations took over 24 s in `make bench-chat`, a rare word in a 100 000-message conversation 450 ms at p50). So `communication_search(q, conversation, lim)`, owned by the schema owner and `SECURITY DEFINER` like `communication_add_partitions`, finds the keys through the index and decides itself what the scope may read, as the policies do (migration test over every scope); the app then reads the rows by those keys through the policies, and the service checks each result's conversation too: three layers that agree. It plans each search for its own values (`EXECUTE`) |
| Search across a doctor's conversations | A read-only scope `app.chat_practitioner_role_id`, set by the search only, beside the clinic's and SAL-28's `app.practitioner_role_id` (the same role; the policies and `communication_search` require both): its policies show the messages, and their files, of the conversations the role takes part in as an active participant of the scope's clinic (never a `pending_consent` invitation, never removed), after its `history_from_seq`, in the conversations the scope sees at all (the participant's policy reads the conversation's: the clinic's own while the care consent is active; a granted guest's conversation and only the side threads they take part in, SAL-28). `communication_search` decides the same itself: the clinic's own conversations under the care consent, or, for a doctor of another clinic, a conversation (or its side thread) whose chat consent for the role is active, with the role still an active participant of it, under the care consent with the treating clinic (finding 7, after SAL-28's merge). Not unified with `app.practitioner_role_id`: every practitioner transaction sets that one, so its policies would show a practitioner's messages across all their conversations in every transaction, where 00041 and SAL-28 show them one conversation at a time (`TestChatGroup_AGuestSeesTheConversationOnlyWhileGranted` asserts no message without a conversation set). The service also checks each result's conversation as `enter` does (participant, `access.ReadsChat`, history): the database and the service decide alike (the end-to-end test, on the owner's connection, caught a revoked conversation found by the role scope alone before this check). A patient's search uses their own scope; at most 50 results, newest first; audited once per patient whose messages it returns (one event naming nobody when it returns none; review, finding 2); the query is never logged. The service's check can only drop a result the search function already excluded (a revocation between the two), so a page may be short: search has no paging, so nothing is skipped |
| Terms | Tagged on read with `terminology`'s matching (SAL-13, as transcript lines): the body in the reader's UI language (their account's chosen language, else the one its client last matched, `account.detected_language`, as `uiLanguage` in internal/api reads it), then English for spans that overlap none; never stored, the body unchanged. On the JSON API's pages, searches and a send's or edit's answer; not in socket frames, which are the same bytes for every reader whatever their language. The reader's language is read through `auth.Service.Me`, which audits an account read per page |
| Message without text | `body` may be empty with files; the row's `body` is then null and `attachment_count` above 0 (a check); the JSON has no `body`. Such a message has no text to edit (`forbidden`) |
| Socket frames | Message frames (live and replay) carry `attachments` as at send time. Nothing tells a socket when a scan ends: the apps read the page again (see "Needs a contract change") |

Needs a contract change (reported, not made):

- An `attachment_scanned` frame (or a `message_edited` with the new statuses) when a file's scan ends, so the apps need not
  read the page again to see a thumbnail or a rejection. Today the scan job (`binary`, the worker) does not publish on
  the chat's channel.
- A `DELETE /conversations/{id}/attachments/{attachment_id}` for an upload removed before it is sent. The contract of
  wave 1 has none; today such an upload stays the uploader's alone, is never shown to anyone, and is deleted 24 h after
  its upload by `binary`'s retention.

Copy waiting for the owner: none. The backend adds no user-facing string; the new error codes (`scanning`,
`unsupported_type`, `too_large`, `length_required`, `too_many_files` on the chat's routes) are binary's, rendered by the
apps (SAL-31).

Review (fresh subagent, 2026-11-12): fixed 1 (decode budget by format, one decode at a time, `GOMEMLIMIT`), 2 (search
audit per patient), 3 (10 uploads waiting per participant), 4 (receipt per participant with its own TTL, current
participants only), 5 (a failed thumbnail leaves the file clean), 6 (role scope's file policy after
`history_from_seq`, unsent files to their uploader only); 8 not a bug (search has no paging; "else detected" is
`account.detected_language`, wording made explicit); 7 done at the merge of main after SAL-28 (migration renumbered 00045; the search and the search scope's policies follow SAL-28's participant status and guest scope: migration, service and Postgres tests over the guest pending, declined, granted, chat consent revoked, care consent revoked and removed, and a side thread without them); left 9 (the owner's decision).

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

What the standard library's decoders allocate for crafted 40-megapixel images (`BenchmarkDecodeCrafted`, review,
finding 1; `runtime.MemStats.TotalAlloc` over one decode, whose coefficients and pixels all live until it returns, so
the decode's peak):

| Crafted file (40 MP) | File | Decode | Bytes a pixel |
| --- | --- | --- | --- |
| Progressive JPEG, gray | 78 KB | 190.8 MiB | 5.0 |
| Progressive JPEG, YCbCr 4:4:4 | 235 KB | 572.2 MiB | 15.0 |
| Progressive JPEG, CMYK | 313 KB | 915.6 MiB | 24.0 |
| PNG, RGB 16-bit | 237 KB | 305.3 MiB | 8.0 |
| PNG, RGB 16-bit, interlaced | 237 KB | 610.7 MiB | 16.0 |

The old bound (40 MP, "4 bytes a pixel") let each through, in a worker of 768 MB running two scans. Now `decodeCost`
reckons a decode from the header (JPEG: a byte a component, 4 more for a CMYK or RGB conversion, 4 a component when
progressive; PNG: its image's bytes a pixel, twice when interlaced; plus 1 MiB), checked above what the decoder
allocates for each kind (`TestDecodeCost_BoundsWhatTheDecoderAllocates`), and refuses past 192 MiB: every file above
is refused from its header with under 1 MiB allocated (`TestDecodeImage_RefusesACraftedImagePastTheMemoryBudgetFromItsHeader`),
while a 12-megapixel progressive photo (about 172 MiB) still gets its thumbnail. Decodes run one at a time per
worker, and the worker's `GOMEMLIMIT` is 640 MiB (`SALUS_WORKER_GOMEMLIMIT`, under its 768m limit).

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

#### Before the merge (after main's SAL-28, migration 00045, 2026-11-12)

One `make bench-chat` run on a quiet host (load average 2.0 at the start, 2.7 to 6.1 during, 3.2 at the end), same
seed (3 000 021 messages, 50 010 conversations, 24 partitions), with finding 7's search (participant status, guest
scope). Every target met:

| Measure | p50 | p95 | p99 | Target |
| --- | --- | --- | --- | --- |
| Delivery at 200 messages a second | 9.6 ms | 13.2 ms | 16.1 ms | p95 < 300 ms: met |
| Send's commit at 200 a second | 9.5 ms | 13.1 ms | 15.1 ms | |
| Latest page (service, audited, 50 messages) | 6.5 ms | 11.1 ms | 13.8 ms | p95 < 200 ms: met |
| Page 22 of a long conversation | 5.9 ms | 10.3 ms | 11.9 ms | p95 < 200 ms: met |
| A send with nothing else running | 5.6 ms | 9.2 ms | 14.4 ms | p95 < 300 ms: met |
| Search a 100 000-message conversation, a rare word | 19 ms | 31 ms | 40 ms | none (capacity) |
| Search a 100 000-message conversation, a word in every message | 15 ms | 20 ms | 25 ms | none |
| Search a small conversation, a rare word | 16 ms | 19 ms | 24 ms | none |
| Search across 50 010 conversations, a rare word | 258 ms | 288 ms | 314 ms | none |
| Search across 50 010 conversations, a word in every message | 286 ms | 306 ms | 319 ms | none |

Capacity: delivery at 400 a second 13.0 ms p95, at 1000 a second 207 ms p95. The earlier misses of delivery at 200 a
second (p95 17 to 21 s) were the host's load, as the A/B run suggested.
