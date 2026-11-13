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

## SAL-29: push notifications, the e-mail fallback, chat hours and webhooks

Built: `pkg/push` (the `Sender` port, `ByPlatform`), its fake `pkg/push/pushtest` (the default), and the adapters
`pkg/push/fcm` (FCM HTTP v1 on the service account's OAuth assertion) and `pkg/push/apns` (APNs over HTTP/2 with an
ES256 provider token), both tested against `httptest` servers and off until the owner adds credentials
(`config.LoadPush`, docs/deploy/push.md). Migration 00047: `push_device`, `organization_chat_settings`,
`webhook_endpoint`, `webhook_dead_letter`, and an index on `account.practitioner_id`. The routes of the plan's
"Contract of wave 1", SAL-29 part, as settled: `PUT` and `DELETE /me/devices`, `GET /organizations/{id}/chat-hours`,
`GET` and `PUT /clinic/chat-settings`, `GET` and `POST /clinic/webhooks`, `DELETE /clinic/webhooks/{id}`.

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| The send's job | `Send` enqueues one `chat_message_sent` job in its transaction (InsertTx), args `conversation_id`, `organization_id`, `message_id`, `sender_participant_id`, `seq`: ids and the number only. A repeat of a `client_message_id` enqueues nothing. In `internal/communication` only: the `Jobs` port and `MessageSentArgs` (new `service_notify.go`), its River implementation (new `repository_river.go`), the `Jobs` field of `Ports` (`service.go`, one line), the enqueue at the end of `Send`'s transaction (`service_messages.go`), and two reads for the jobs in `service_notify.go`: `NoticeParticipants` (the active participants with their `read_seq`) and `NoticeMessage` (the message, for a webhook with clinical content), both in the clinic's scope with the consent checked as well, audited with the job as agent. The test fixture (`fakes_test.go`) got a fake `Jobs` that rolls back with the fake transaction |
| Which binary works which job | The notifier works `chat_message_sent` (the fan-out) and `chat_push` on a new `push` queue beside `email`, and `chat_unread_email` on `email` (CLAUDE.md names it for e-mail and push). The worker works `webhook_delivery` on a new `webhook` queue: a clinic's slow or failing endpoint never holds up a person's notification, and the webhook key reaches only the api (which seals) and the worker (which signs), never the notifier. The scheduler registers every kind, as River's leader |
| The fan-out | One transaction: an unread check per other active participant, scheduled the clinic's `email_after_minutes` later; for the patient, a `chat_push` per device of their account (staff register no devices: the app is the patients'); a `webhook_delivery` per clinic webhook that takes `message.created`. A conversation the clinic no longer sees (revoked consent) tells nobody |
| E-mail fallback | `chat_unread_email` is unique by the participant and their `read_seq` when the message came (River unique args, default states): a burst to someone who has not read on gives one check, River keeps a run check a day, so at most one e-mail a day while they read nothing, and a message after they read on gets its own check (so none is missed). When it runs it e-mails only if the participant's `read_seq` is still below the message and they are still a participant. Patients get `chat_unread` with `PATIENT_APP_URL/chat`, practitioners `chat_unread_staff` with `WEB_BASE_URL/chat` (SAL-31: the screens' paths), each in the account's language. A practitioner's address comes from the new `auth.PractitionerAccount` (by practitioner, in the clinic, audited with the job as agent); a patient's account from the new `patient.NoticeAccount` (in the clinic's scope, audited) |
| Push | The fixed text "New message from your clinic" (title "Salus"), from the notification templates in the account's language; no data payload at all (no conversation id): the app opens its chat list. A token is one row (`UNIQUE (token)`): registering it moves it to the account signed in last. Deleted on `DELETE /me/devices`, on FCM `UNREGISTERED` or APNs 410/`BadDeviceToken`, and with the account (foreign key). Staff calling `PUT /me/devices` get 403. 5 attempts |
| Chat hours | Owned by `organization` (`organization.Chat`, a second service of the domain like terminology's `Reviews`): `hours` is a list of `{day: mon…sun (FHIR days-of-week), start, end}` in the clinic's zone, up to 21 stretches, no overlap, `24:00` for midnight; `response_time_hours` 1–168 (default 24); `email_after_minutes` 5–10080 (default 60). Read by the clinic's staff and by a patient with an active care consent (404 otherwise); added field `time_zone` to the chat-hours answer. A clinic that states no hours is `open_now` (no out-of-hours notice). The admin's change is audited, also when refused |
| Webhooks | Owned by `notification` (no new domain). HMAC-SHA256 over `"<t>.<body>"` with the endpoint's secret, header `Salus-Signature: t=<unix>,v1=<hex>`, the stable `Salus-Event-Id` (a UUIDv8 of the message's id and the event: the same on every attempt and every fan-out retry). Body: `id`, `type`, `organization_id`, `data.conversation_id`, `data.message_id`; `data.message` (`seq`, `sent_at`, `sender_role`, `patient_id`, `body`) only with `include_clinical`, read in the clinic's scope (a revoked consent cancels the delivery). River retries 8 times with its backoff (about 1.5 h); the last failure writes `webhook_dead_letter` (ids, attempts, last status), kept 30 days (a purger on the retention role). At most 10 endpoints per clinic; `include_clinical` cannot be changed, only a new endpoint (no update in the contract). The secret `whsec_` + 43 characters, shown once, sealed with `WEBHOOK_KEY` (AES-256-GCM bound to the endpoint id, through `auth.AESSecretBox` as CLAUDE.md asks of a sibling with the right shape) |
| SSRF | At creation: https, no credentials or fragment, a host name with a dot (not `localhost`) or a public address. At every connection: the dialer's `Control` refuses loopback, private, link-local, unique-local, multicast, unspecified, carrier-grade NAT, `0.0.0.0/8`, `192.0.0.0/24` and `198.18.0.0/15` on the resolved address (DNS rebinding), IPv4-mapped addresses judged as IPv4; no proxy from the environment; no redirect followed; 10 s per attempt; errors never carry the URL |
| `WEBHOOK_KEY` | A key of its own, loaded like `TOTP_KEY` and `BINARY_KEK` (`WEBHOOK_KEY` or `WEBHOOK_KEY_FILE`), required by the api and the worker. Production: a third root-only key file `keys/webhook_key` on the encrypted volume (`salus-volume` writes and checks it, `deploy.sh` and the deploy workflow refuse an env file that sets it and a volume without it, the prod compose mounts it into the api and the worker only), so one more key of the same kind and nothing else of ADR 0011 changed. **Before the first deploy of this release the owner generates it, adds it to Bitwarden and runs `salus-volume keys`** (deployments/production/README.md "Keys") |
| Push credentials | `FCM_CREDENTIALS_FILE` (the service account's JSON; the project id comes from it), `APNS_KEY_FILE`, `APNS_KEY_ID`, `APNS_TEAM_ID`, `APNS_BUNDLE_ID`, `APNS_SANDBOX`; all of a platform's or none (a half configuration is refused, naming the variable). Each platform independent; the other stays on the fake. Their compose secrets are not in the prod compose yet: they land with the credentials and the app's token registration (docs/deploy/push.md) |
| Audit | Device registration and unregistration (on the account), each job's read of devices, participants, accounts, the patient and the message (job as agent), the admin's settings change and every webhook list, creation and deletion (refusals too). Never the token, URL or secret |
| After SAL-28 and SAL-30 (merge of main) | `NoticeConversation` gives active participants only (never one `pending_consent`), a doctor of another clinic only while the patient's chat consent for them is active (the care consent was already checked), and never the patient in a `staff_only` side thread; a granted guest's unread e-mail looks up their role and account in their own clinic. A message with files enqueues the same ids-only job: nothing of its files reaches a push, an e-mail or a webhook (the clinical webhook carries the text only) |
| Clients | `web/src/chat/chatApi.ts` and `mobile/lib/chat.dart` gained `chatHours(organizationId)`, with their fakes and tests; the admin's settings and webhooks and the device registration are in the generated clients only |

No message text, sender or patient name in any push, e-mail, webhook without clinical content, job argument, job
error or log line: `internal/worker/chat_notify_test.go` sends a recognisable synthetic text through the chat as the
api sends it, runs the real notifier and worker, and finds it nowhere outside the record; `internal/api` checks the
job's arguments and the log through `api.New`; the notification service's tests check each message.

### Measured (make bench-chat, 2026-11-12)

The same seed as SAL-27's (3 000 021 messages, 24 partitions); the host shared with three other chat packages'
stacks and other projects, load average 3 to 7 for the runs before the change and 22 to 32 for the runs after, so
runs on different code are not comparable. The load test therefore measures the job's cost in one run (new
subtests): the same send alternating with and without its job, and delivery at 200 a second without the job just
before the same with it, on other conversations.

| Measure | p50 | p95 | p99 |
| --- | --- | --- | --- |
| Before (main, load average about 5): a send alone | 8.1 ms | 13.1 ms | 14.8 ms |
| Before (main): delivery at 200 a second | 11.5 ms | 363 ms | 488 ms |
| Before (main, a second run, heavier load): delivery at 200 a second | 1.52 s | 4.90 s | 6.62 s |
| After, one run (load average about 30): a send with its job, alternating | 9.0 ms | 26.8 ms | 99.6 ms |
| After, same run: a send without it, alternating | 8.4 ms | 28.0 ms | 234 ms |
| After, same run: delivery at 200 a second without the job | 164 ms | 5.38 s | 6.27 s |
| After, same run, next: delivery at 200 a second with the job | 16.0 ms | 448 ms | 732 ms |
| After, same run: delivery at 400 and 1000 a second (capacity) | 13.9 ms, 834 ms | 1.00 s, 4.02 s | 1.23 s, 4.59 s |

- The job adds one River insert to the send's transaction: under 1 ms at the median, inside the noise at p95
  (alternating sends). The delivery numbers move by a factor of ten between two adjacent windows on the same code,
  so the host's load decides them; no regression from the job can be read out of them, and the 300 ms target at
  200 a second was not met on this host either before (363 ms) or after, with or without the job.
- With the job the backends also wait on `Lock:object` (36 samples of about 10 000 at 200 a second, 339 at 1000
  a second, none without the job): River's insert notification (`pg_notify`, which takes the notify queue's lock at
  commit). Small at the clinics' rates; at the capacity rates it is one more serialised step of a commit. Not tuned:
  River debounces its notifications per queue, and the alternative (polling only) delays every notification.
- To rerun on a quiet host: `make bench-chat`; the two new subtests print the comparison.

### Copy waiting for the owner's approval (all ten languages, `internal/notification/templates/<language>/`)

| Key | English |
| --- | --- |
| `chat_push.txt` | Title "Salus", body "New message from your clinic" |
| `chat_unread.txt` | Subject "New message from your clinic"; "You have a new message from your clinic." / "To read it, open the Salus app:" and the link |
| `chat_unread_staff.txt` | Subject "New chat message in Salus"; "You have an unread chat message in Salus." / "To read it, sign in to Salus:" and the link |

### Blocked

- The patient app's token registration (Firebase SDK, APNs entitlement, the call to `PUT /me/devices`): with the
  credentials (owner's instruction), docs/deploy/push.md.
- FCM and APNs in production: their DPAs and transfer basis (docs/privacy/processors.md), then the credentials.

- **Review (fresh subagent, 2026-11-12):** fixed: the webhook SSRF denylist (NAT64, 6to4, Teredo, IPv4-compatible, documentation, discard-only, 240/4, broadcast), a dead letter for any three-digit status (00047's CHECK 1–999), a re-check for a later message still unread after a partial read, a queued push only while the device is still the account's, the ten-endpoint cap under an advisory lock, a consent check on every webhook attempt and an unopenable secret dead-lettered at once; declined (owner's decision, a retention change): an age-based deletion of stale `push_device` rows.
- **Independent review (2026-11-12):** fixed: the notifier's and the worker's chat service (`noticeChats`) lacked `ChatConsents`, so a conversation with a granted doctor of another clinic panicked every fan-out, push, unread e-mail and webhook job; tests `TestChatNotifications_ReachAConversationWithAGrantedGuest` (red on the nil pointer in `noticed`) and `TestChatNotifications_TellNobodyOnceTheCareConsentIsRevoked` (a guard, green before and after). Measured on the way, not changed here: in a stack database with 22 372 slots the directory's `SearchFreeSlots` costs 765 022 as `salus_app` (over `jit_optimize_above_cost`, 85.6 s under `EXPLAIN ANALYZE`), about 580 per slot row for the schedule and practitioner-role policies; dropping 00047's objects (then numbered 00046) leaves it unchanged, so it predates this migration, and the empty-database cost test does not see it.

## SAL-31: the chat on web and mobile, and the AI summary

Built on `web/src/chat/fakeChat.ts` and `mobile/test/fake_chat.dart`, then the real clients. After SAL-28 and SAL-30
were merged (2026-11-12) every route of theirs goes through the generated types (web `components['schemas']`,
mobile `salus_api` models and `ChatApi`), and both fakes keep main's rules. After SAL-29 was merged the chat hours are
the generated `ChatHours` too (`time_zone`, `hours` as `{day: mon..sun, start, end}`, `response_time_hours`,
`open_now`), read by both real clients through `GET /organizations/{id}/chat-hours`; the web shows none (doctors), the
patient app names each day in the UI language from its code. The screens register no push device: SAL-29's push
registration and notification settings are not part of SAL-31's screens.

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| Summary engine | `pkg/llm.Summarizer` with the fake (`llmtest.Summarizer`) only; `config.LoadChatSummary` refuses any `CHAT_SUMMARY_ENGINE` but `fake`, whatever `LLM_ENGINE` says. **Waiting for the owner**: the basis for a real model reading the chat (SAL-12's transcription consent covers the consultation only), then the processor's DPA |
| Long conversation | At least 50 messages the doctor reads (`last_seq - history_from_seq`, one page); below it the endpoint answers `400 invalid_request` and the web app shows no button. A summary covers the latest 500 messages (bounds the read and the model's input) |
| Summary endpoint | `POST /conversations/{id}/summary`, optional `{lang}` (a staff language, default `bs`); practitioners only (`403` for the patient); the read audited as one read of the rows summarised, in its transaction; the model called after the commit, 30 s timeout; not stored. `ai_generated` is a plain boolean in the schema (a `const` broke the Dart generator) |
| Web: virtual list | Written by hand (`web/src/chat/VirtualList.tsx`, 137 lines: overscan 8, a 64 px estimate replaced by each row's measured height, `ResizeObserver` for late changes); no dependency |
| Web: add a doctor | Only the clinic's own doctors (the web app has no search of another clinic's); only the treating practitioner sees add and remove; "from a message" picks among the latest 20 loaded |
| Web: names | The patient's name read once through FHIR `Patient/{id}` (an audited read), doctors from the clinic's staff list; system messages from their code only |
| Web: chat hours | Not shown to doctors: they are the patient app's notice |
| Both: socket | As `web/src/live.ts`: backoff, resume after the last shown cursor, drop a seen `seq`, stop for good on 4401, 4403, 4404 (`ChatEnd` gains `unauthenticated`) |
| Both: attachment status | The contract has no status route or frame for a scan's outcome: web takes it from a `message_edited` frame, mobile re-reads the newest page while a file is scanning. **Waiting for the owner**: a scan-outcome frame needs a contract change |
| Both: refused handshake | A handshake the API refuses (401, 403, 404 before the upgrade) gives the client no status, so a connection that never opened reads `GET /conversations/{id}`: `unauthenticated`, `consent_required`, `forbidden` or `not_found` end the view as the 44xx close codes do; anything else (the conversation answers, or the API is unreachable) tries again with the backoff. Each check is an audited read, so it is made on the first refused handshake in a row and then on every third (web: about twice a minute at the 10 s wait; mobile: every 90 s at the 30 s wait). No server change |
| Shapes taken from main | `from_seq` is the last message the added doctor does not read (web sends the chosen message's seq minus one); a side thread opens without a body; an attachment's `thumbnail` is a required boolean and `content_type` one of JPEG, PNG, PDF; mobile always sends `attachment_ids` (empty for none, as the generated `SendMessageRequest` does); an invitation has no history field: what the invited doctor would read is their participant's `history_from_seq` (the invitation id is the participant's), read from `GET /conversations/{id}` |
| Web: ended view | On 4403/4404/4401 (or a refused handshake) the open view unmounts: no message, summary, header or composer stays, only why and the way back; the hook drops the messages |
| Web: composer | After a refusal a retry cannot change (anything but `rate_limited`, `internal error`, `timeout` and an unreachable API) the text unlocks and the next send is a new message |
| Web: Messages | Shown to and routed for practitioners only; an admin reads no conversation |
| Mobile: revoke | The chat view lists the granted invitations (`GET /me/chat-invitations?status=active`) on their own read (a failure is read again when the app comes back to the foreground), each as a visible line above the messages, "Access granted to {doctor}, {clinic}." with "Revoke access" beside it (Art. 7(3): as easy as granting), also after a restart. Pending and granted are read again on `participant_invited`, `participant_removed`, `consent_granted` and `consent_declined`. The legal strings ("revoke ... in this conversation at any time") stay as written, now true |
| Mobile: ended view | On 4403/4404 (or a refused handshake) the view drops the messages, the invitations, the granted doctors, their names, the hours and the composer's text and files, and shows only the notice; on 4401 the app goes back to sign-in through the home screen's sign-out path |
| Mobile: chat hours' zone | The hours are shown as the clinic states them, with its zone named ("Times are in the clinic's time zone, Europe/Sarajevo."), not converted: a day's hours can cross midnight in another zone |
| Mobile: consent prompt | Grant is disabled until the invited doctor and their clinic are shown; until then the card says so (`chatInviteUnresolved`) and offers to load them again; decline stays possible |
| Mobile: read position | Marked read only while the newest message is in view (the list within 48 px of its bottom) and the app is resumed |
| Mobile: removed file | A file taken off before sending stays uploaded and counts toward the 10 unsent (`too_many_files`, said in both apps with the 24-hour purge) until it is purged. **Waiting for the owner**: the delete route needs a contract change |
| Mobile: wiring | `SalusApp` and `HomeScreen` take an optional `Chats` (none hides Messages); `main.dart` passes the real client. `chat_screen.dart` is 558 lines, over the ~400 guideline |
| Routing | `/conversations` added to the Vite dev proxy (with `ws`) and to the production Caddyfile |
| e2e | `TestChat_ThePatientAndTheDoctorChat` and `TestChat_TheDoctorAddsAColleagueAndThePatientSeesIt` (invite, grant, the patient sees both system messages, the colleague reads the history, the granted list, the revocation ends access) pass on main's backend |

### Measured

| Measure | Result |
| --- | --- |
| Web, `fakeChat({ history: 5000 })`, every page loaded by scrolling up, 600 px viewport, rows of 48, 84 and 120 px (Vitest, jsdom, `ChatView.scroll.test.tsx`) | at most 152 DOM nodes under the list, 24 rows rendered; a render after a scroll step 5.0 to 5.4 ms median, 8.5 to 11.2 ms max over three runs of 50 steps (no long task, which starts at 50 ms) |
| Mobile, `FakeChat(history: 5000)`, every page loaded, 390x844 | 10 message widgets built (`ListView.builder`) |

### Copy waiting for the owner

Web, area `chat` (five staff languages) and `shell.messages`: the list, the view, the composer, the socket's ends, the
system messages (each with an anonymous variant), the reasons, the add-doctor dialog, the threads, the attachments,
the summary (`summary.label` "AI generated", `summary.hint` "An AI generated summary of what was written. It is not a
diagnosis.") and the errors; tagged `legal`: `staffOnlyNote`, `pending`, `ended.consent_required`,
`participant_invited`, `consent_granted`, `consent_declined`, `threads.note`, `error.consent_required`.
Mobile (ten languages): `chats*`, `chat*` keys of the list, composer, files, chat hours and out-of-hours notice,
system messages, and the consent prompt and revocation (`chatInvite*`, `chatGrant`, `chatDecline`, `chatGranted`,
`chatDeclined`, `chatRevoke*`, `chatRevoked`, all tagged `legal`). Open question: whether the out-of-hours notice
names an emergency number.

Added after the reviews: web `chat.error.too_many_files` and mobile `chatTooManyFiles` ("You already have {n} files
waiting to be sent. Files not sent are removed after 24 hours; try again later.", the limit as an ICU plural);
mobile `chatInviteUnresolved` (`legal`), `chatGrantedTo` (`legal`, "Access granted to {doctor}, {clinic}."),
`chatGrantedToUnknown` (`legal`, "Access granted to a doctor from another clinic."), `chatHoursZone` ("Times are in
the clinic's time zone, {zone}."). `chat.summary.hint` is now shown in the summary panel, not only as a tooltip.

Review (fresh subagent): fixed 1 (granted invitations listed for revocation), 2 (nothing stays after an end), 3
(refused handshake ends the view), 4 (grant waits for the doctor and clinic), 5 (read only at the bottom, in the
foreground), 6 (composer unlocks), 7 (hint in the panel), 8 (e2e passes), 9 (`from_seq` is seq minus one), 10
(Messages for practitioners only), 11 (`too_many_files` said); declined none (the delete route of 11 waits for a
contract change). Second review (fresh subagent): fixed all seven (mobile end clears the view and signs in again;
granted read on its own with a retry on resume; visible revoke line; reload on the consent codes; the hours' zone
named; `too_many_files` reworded with the limit; access checks spaced out); declined none.
