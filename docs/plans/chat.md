# Chat: doctor–patient messaging with group consults

Epic SAL-26. Status: planned, 2026-11-12. Design in ADR 0012. Step 0 (SAL-27) lands the core and the contract every
later package builds on; SAL-28 to SAL-31 then run in parallel.

## Tickets

| Ticket | Branch | What |
| --- | --- | --- |
| SAL-27 | `sal-27-chat-core` | Conversations, participants and messages, clinic-scoped with RLS, optional link to an encounter; messages partitioned by month, never deleted, edits keep history; cursor pagination; send and receive over a WebSocket with reconnect, replay and idempotent send; the contract (Go types, OpenAPI, generated TS and Dart types, web and mobile fakes); load test with millions of messages |
| SAL-28 | `sal-28-group-consults` | A doctor adds a doctor (same clinic: the treating doctor decides; another clinic: the patient's consent first), history full or from a chosen point, a system message naming who joined and why, remove participant, doctor-only side threads, every add, remove and consent audited |
| SAL-29 | `sal-29-chat-notifications` | Push through `pkg/push` (a fake by default; FCM and APNs adapters off until the owner adds credentials), e-mail fallback for a message unread for a configurable time, the clinic's chat hours and expected response time, signed webhooks with retries and a dead letter; no message text in any notification |
| SAL-30 | `sal-30-chat-attachments` | Attachments through the existing upload and scan pipeline (`binary`), thumbnails; read receipts and typing through Redis; search within a conversation and across a doctor's conversations; medical terms tagged for the hover explanations (SAL-13) |
| SAL-31 | `sal-31-chat-ui` | The chat on web (doctors) and mobile (patients): conversation list with unread counts, virtualized message list, add-doctor dialog, doctor-only thread panel, attachment preview, the patient's consent prompt; an on-demand AI summary of a long conversation through `pkg/llm` (fake by default, originals kept, labelled AI generated) |

## Waves

```
Step 0 (one PR): SAL-27, this plan, ADR 0012
Wave 1, in parallel, each in its own worktree and compose project:  SAL-28   SAL-29   SAL-30   SAL-31
```

- **Merge order** in wave 1: SAL-28, SAL-30, SAL-29, SAL-31, one at a time, main merged into the next branch first
  (no force-push). SAL-28 first: it changes the access rule the others read through. SAL-31 last: its UI wires
  the endpoints of the other three, against the fakes until they are merged.
- **Shared files.** `api/openapi.yaml`, `internal/api/app.go`, `docs/architecture/archcheck/ownership.yaml`,
  `internal/retention/rules.go` and the i18n catalogs are edited by several branches; the later merge resolves.
  Inside `internal/communication`, each package adds its own files (`service_participants.go` for SAL-28,
  `service_attachments.go` and `service_search.go` for SAL-30, `service_summary.go` for SAL-31) and changes the
  core's files only where it must.
- **Compose projects** `-p salus-sal28`, `salus-sal29`, `salus-sal30`, `salus-sal31`, each with an override file
  without published host ports and its own app image tag.
- **Migrations.** SAL-27 takes `00041`. Main took `00042` (SAL-19) and `00043` (SAL-38) meanwhile, so each chat
  branch takes the next free number when it merges main before its own merge: SAL-28 took `00044`.

## Decisions

### Owner, 2026-11-12

| Topic | Decision |
| --- | --- |
| Read state | Each participant's own read position is stored (`conversation_participant.read_seq`, the last message number they have read): it drives unread counts (SAL-31) and the e-mail fallback (SAL-29), which must survive a Redis restart. The API never shows one participant's position to another (that would be a stored receipt): the caller sees only their own `unread`. What others see of it (read receipts) and the typing indicator stay ephemeral in Redis, with a TTL, never stored (SAL-30) |
| Partitions, pagination, deletion, gateway | Messages partitioned by month, cursor pagination, messages never deleted, the existing WebSocket gateway reused (the brief of 2026-11-12; ADR 0012) |

### Agent, for the owner to review (2026-11-12)

| Topic | Decision |
| --- | --- |
| Domain | One domain `communication` (FHIR `Communication`), context `consultation`, as CLAUDE.md already lists it. It owns `conversation`, `conversation_participant`, `communication` (a message) and `communication_revision` (an edit's earlier text) |
| API | Plain JSON (`api/openapi.yaml`), like the transcript: the chat is a live UI channel. A message is FHIR-shaped (`Communication`: `sent`, `sender`, `subject`, `encounter`, `payload.contentString`); the FHIR read and the record export come with the subject-rights feature (`Patient/$everything` does not exist yet), not here. The CapabilityStatement does not change |
| Who opens a conversation | `access.OpensChat`: a practitioner with a patient who has an appointment with them, or the patient with a practitioner they have an appointment with, of any time or status, under the clinic's active care consent. One conversation per patient, practitioner role and encounter (none counts as one): opening again returns the existing one |
| Who reads and writes | The patient of the conversation reads always (their own record, GDPR Art. 15) and writes while the care consent is active. A practitioner while they are an active participant and the consent is active (`access.ReadsChat`, `access.WritesChat`). No admin. After a revocation the clinic sees nothing of the chat (as `access.Profile` after a revocation: no clinical content), immediately: the socket checks again at least every 5 s, and staff are answered `not_found` (404, close 4404), as the clinic's row-level security answers, so the service and the database agree |
| Fan-out | Redis pub/sub on `chat:<conversation_id>`, one subscription connection per api process with an in-process hub (`pkg/redis.Hub`), not one `XREAD BLOCK` per socket as the transcript does: thousands of open chat sockets would hold thousands of Redis connections. Pub/sub can drop a message; Postgres is the record, so a socket that sees a gap in the message numbers, or whose 5 s check finds newer messages, fills it from Postgres |
| Socket | `GET /conversations/{id}/events?after=<cursor>`, mounted in `router.Config.Streams` beside the encounter's stream; `after` is the conversation's `cursor` (its last message's, or the start's when empty) or the last message the client has, so nothing sent between the page and the socket is missed. Every 5 s it also checks that the session is still open (`auth.Handler.Active`; 4401 `unauthenticated` otherwise), and it takes at most five `send` frames at once and one every 200 ms. The conventions of the transcript's socket (wire envelope, same-origin check, ping every 25 s, close 4403/4404) move to `pkg/livesocket`, used by both |
| Ordering | `seq` numbers a conversation's messages 1, 2, 3 … without gaps, assigned under the conversation's row lock in the send's transaction; `sent_at` never goes backwards within a conversation (`greatest(now, last_sent_at)`), so the order by `(sent_at, seq)` is the order by `seq` and a page reads only the partitions it needs |
| Idempotent send | The client's `client_message_id` (a UUID) per sender and conversation: a retry under the same lock finds the first message and returns it (200, not 201), with no second message and no second fan-out; looked for within 24 hours of the conversation's last message, so the lookup reads one or two partitions (measured, ADR 0012) |
| Edits | The sender edits a text message's body (`If-Match` on `version_id`); a trigger copies the earlier body into `communication_revision` on every edit, so history cannot be skipped. System messages are never edited |
| Never deleted | No `DELETE` grant to any role on `communication` and `communication_revision`, and triggers reject `DELETE` and `TRUNCATE` for every role, the owner included (as `audit_event`). Retention: kept with the clinical record (`ThenKeep`), like the encounter and the transcript |
| Partitions | Monthly range partitions on `sent_at`, `communication_yYYYYmMM`, created by `communication_add_partitions(from, months)` (owned by the schema owner, `SECURITY DEFINER`, executable by `salus_app`). The migration creates them from 2026-11 to 2027-04; the scheduler's daily `communication_partitions` job keeps three months ahead. No default partition: a missing month fails the send loudly instead of filling a partition that later blocks the next one |
| Message size | Text 1 to 4000 characters after trimming |
| Audit | Every page read, socket subscription, send and edit writes an `AuditEvent` (entity `Communication` or the conversation) in its transaction, without the text |

## Contract (SAL-27, migration 00041)

### Tables

| Table | Columns |
| --- | --- |
| `conversation` | `id`, `organization_id`, `patient_id`, `practitioner_role_id` (the treating doctor), `encounter_id` (optional), `staff_only` (a doctor-only side thread, SAL-28), `parent_id` (the conversation a side thread belongs to), `last_seq`, `last_sent_at`, `created_at`, `created_by` |
| `conversation_participant` | `id`, `conversation_id`, `role` (`patient` \| `practitioner`), `patient_id` (for the patient) or `practitioner_role_id` (for a practitioner, of any clinic: SAL-28), `history_from_seq` (0: the whole history; SAL-28), `read_seq`, `added_by` (an account), `added_at`, `removed_at`. People are named as the rest of the schema names them, the patient by their profile and a doctor by their role; no account is resolved from a role |
| `communication` | partitioned by `sent_at`; `id`, `sent_at`, `conversation_id`, `organization_id`, `patient_id`, `staff_only` (copied for the row-level security), `seq`, `sender_participant_id` and `sender_account_id` (the participant who wrote and the account signed in, for accountability; both null for a system message), `sender_role` (`patient` \| `practitioner` \| `system`), `client_message_id`, `kind` (`text` \| `system`), `body`, `system_code` and `system_args` (a system message's code and its opaque ids, SAL-28), `version_id`, `edited_at`. Primary key `(id, sent_at)`; index `(conversation_id, sent_at, seq)` and `(conversation_id, client_message_id)` |
| `communication_revision` | `communication_id`, `sent_at`, `version_id`, `body`, `edited_at`, `organization_id`, `patient_id`, `staff_only`; key `(communication_id, version_id)` |

### Row-level security

| Scope | Sees |
| --- | --- |
| `app.patient_id` | Their conversations, participants, messages and revisions that are not `staff_only`; may insert their own conversation and messages and update the read position and the conversation's counters |
| `app.organization_id` (no patient or account scope) | The clinic's conversations while the patient's care consent with the clinic is active |
| any | A participant row exactly where its conversation is seen (the policy reads the conversation) |
| `app.organization_id` + `app.conversation_id` | That conversation's messages and revisions, when it is the clinic's. The service sets `app.conversation_id` after it has read the conversation and checked the caller (`access.Chat`) |

SAL-28 adds the scope of a doctor from another clinic (a participant with the patient's consent); the policies above
are what it extends.

### Go (`internal/communication`)

`Conversation`, `Participant`, `Message`, `Revision`, `Cursor` (opaque: `sent_at` and `seq`), `Page`, the roles
(`RolePatient`, `RolePractitioner`, `RoleSystem`), the kinds (`KindText`, `KindSystem`), and the wire messages:

| Frame | `type`, `version` | Payload |
| --- | --- | --- |
| server: a new message | `message`, 1 | `Message` |
| server: an edit | `message_edited`, 1 | `Message` (its new body and `version_id`) |
| server: a send's answer | `send_ack`, 1 | `client_message_id`, `message` |
| server: a refused command | `command_error`, 1 | as the transcript's (`command`, `id`, `error`) |
| client: send | `send`, 1 | `client_message_id`, `body` |

The Redis channel is `chat:<conversation_id>`, carrying the same envelope bytes as the socket frame.

### HTTP (`api/openapi.yaml`, tag `chat`)

| Route | What |
| --- | --- |
| `POST /conversations` | Open (or return) a conversation: staff name `patient_id`, a patient `practitioner_role_id`; optional `encounter_id` |
| `GET /conversations` | The caller's conversations, newest message first, with their unread count; `before` cursor, 50 per page |
| `GET /conversations/{id}` | One conversation with its participants |
| `GET /conversations/{id}/messages` | A page of messages, newest first: `before` cursor, `limit` 1 to 100 (default 50); `next` is the cursor of the older page, absent at the start |
| `POST /conversations/{id}/messages` | Send: `client_message_id`, `body`. 201 with the message; 200 with the first one for a repeated `client_message_id` |
| `PATCH /conversations/{id}/messages/{message_id}` | Edit the body; `If-Match: W/"<version_id>"` |
| `GET /conversations/{id}/messages/{message_id}/revisions` | Earlier bodies of an edited message |
| `PUT /conversations/{id}/read` | The caller's read position: `seq` (never goes back) |
| `GET /conversations/{id}/events` | The WebSocket: every message after `after` from Postgres, then live; `send` commands |

The generated clients: `web/src/api/schema.d.ts` and `mobile/packages/salus_api` (`make gen-clients`). A Go test
keeps the frames' JSON in step with the schemas.

### Fakes

- Web: `web/src/chat/chatApi.ts`, the `ChatApi` interface the screens call (the JSON API's chat calls and the
  socket as `follow`), and `web/src/chat/fakeChat.ts`, an in-memory `ChatApi` (pages, idempotent send, edit with
  revisions, read position, live frames, `deliver` as the other side, `end` for an end of access, a synthetic history
  of any length) for Vitest and for building SAL-31's screens before the backend of SAL-28 to SAL-30 is merged.
  SAL-31 writes the real `ChatApi` on `api/client.ts` and a WebSocket.
- Mobile: `mobile/lib/chat.dart`, the `Chats` interface on the generated models (the generated `ChatApi` is the raw
  HTTP client), and `mobile/test/fake_chat.dart`, the same fake for the patient app.
- Go: none shared. Each consuming domain declares its own port and fakes it (CLAUDE.md "A port is declared by the
  package that consumes it").

## Targets and load test (SAL-27)

Message delivered (send on one socket to the frame on another participant's socket) under 300 ms p95; the latest page
of a conversation under 200 ms p95 with millions of messages in the table. `make bench-chat` seeds a throwaway
database with synthetic messages (several million, over 24 monthly partitions, many small conversations and a few
with 100 000 messages each) and measures both through the HTTP handler and the socket against the compose Postgres
and Redis. The numbers go in `docs/plans/chat.PROGRESS.md`.

## Contract of wave 1 (settled in step 0)

The shapes the wave's backend packages implement and SAL-31's screens are built against, in parallel. A package may
add a field, never rename or drop one here; a change it needs is reported, not made. Each adds its schemas to
`api/openapi.yaml` and its methods to the web `ChatApi`, the mobile `Chats` and their fakes.

**SAL-28 (group consults).**

| Route | What |
| --- | --- |
| `POST /conversations/{id}/participants` | The treating practitioner adds a practitioner role: `practitioner_role_id`, `organization_id`, `reason` (`second_opinion` \| `specialist_review` \| `handover`), `history` (`full` \| `from`) and `from_seq` with `from`: `from_seq` is the last message the added doctor does **not** see (they read the messages with `seq` > `from_seq`; to show them message 5 onwards, send 4; 0 is the whole history, at most `last_seq`). Same clinic: active at once. Another clinic: `status` `pending_consent` until the patient consents. 201 `ConversationParticipant` |
| `DELETE /conversations/{id}/participants/{participant_id}` | The treating practitioner removes one (not the patient, not themselves); a granted doctor of another clinic's chat consent ends with it; 204 |
| `GET /me/chat-invitations` | The patient's pending invitations of doctors from other clinics: `id`, `conversation_id`, `organization_id`, `practitioner_role_id`, `reason`, `invited_at`, `status`. `?status=active` lists the granted ones instead (the default is `pending_consent`), whose `id` `/revoke` takes (added in SAL-28's review) |
| `POST /me/chat-invitations/{id}/decision` | `decision`: `grant` \| `decline`; a grant is a FHIR `Consent` (granular: that conversation and that role; revocable) |
| `POST /me/chat-invitations/{id}/revoke` | Revokes a granted one; the doctor loses access at once |
| `POST /conversations/{id}/threads`, `GET /conversations/{id}/threads` | A doctor-only side thread (`staff_only`, `parent_id`), and the list; never the patient's |

`ConversationParticipant` gains `status` (`active` \| `pending_consent`) and `organization_id`. System messages
(`system.code`, `system.args` with opaque ids and codes only): `participant_added` (`practitioner_role_id`,
`organization_id`, `reason`), `participant_invited` (same), `participant_removed` (`practitioner_role_id`),
`consent_granted` and `consent_declined` (`practitioner_role_id`).

**SAL-29 (notifications and webhooks).**

| Route | What |
| --- | --- |
| `PUT /me/devices` | Registers the app's push token: `platform` (`android` \| `ios`), `token`; `DELETE /me/devices` with the same body unregisters |
| `GET /organizations/{id}/chat-hours` | For the clinic's patients and staff: `hours` (per weekday, `start` and `end` in the clinic's zone), `response_time_hours`, `open_now` |
| `GET`, `PUT /clinic/chat-settings` | The admin's: `hours`, `response_time_hours`, `email_after_minutes` (the e-mail fallback's delay) |
| `GET`, `POST /clinic/webhooks`, `DELETE /clinic/webhooks/{id}` | The admin's endpoints: `url` (https), `events`, `include_clinical` (off by default); the secret is shown once, on creation |

`Conversation` gains nothing; the out-of-hours notice is the app's, from `chat-hours`.

**SAL-30 (attachments, receipts, typing, search, terms).**

| Route or frame | What |
| --- | --- |
| `POST /conversations/{id}/attachments` | Uploads a file through the scan pipeline (as `binary` uploads stream today); 201 `ChatAttachment` with `status` `scanning` |
| `SendMessageRequest` and the `send` frame | Gain `attachment_ids` (UUIDs of the sender's clean or scanning uploads); `body` may then be empty |
| `ChatMessage` | Gains `attachments`: `ChatAttachment` (`id`, `content_type`, `size`, `filename`, `status` `scanning` \| `clean` \| `rejected`, `thumbnail`), and `terms`: the SAL-13 `TermSpan` list over `body`, computed on read, never stored |
| `GET /conversations/{id}/attachments/{attachment_id}/content`, `…/thumbnail` | The bytes, for whoever may read the message, once clean |
| `read` frame (client), `receipt` frame (server) | `read` `{seq}` moves the read position (as `PUT …/read`) and tells the others `receipt` `{participant_id, seq}` |
| `typing` frame (client and server) | `{participant_id}`, at most one a second per participant, gone after 5 s; Redis only |
| `GET /conversations/{id}/search?q=`, `GET /conversations/search?q=` | `{results: [{conversation_id, message: ChatMessage}]}`, newest first, only what the caller may read |

**SAL-31 (UI and the AI summary).** `POST /conversations/{id}/summary` for a practitioner who may read the
conversation: `{summary, generated_at, up_to_seq, ai_generated: true}`, not stored, through a `pkg/llm`
`Summarizer` with the fake by default.

## Notes for the wave-1 packages

- **SAL-28.** Cross-clinic: a new consent of the patient (FHIR `Consent`, granular, revocable) per conversation and
  invited practitioner role; nothing of the conversation (not even its participants) is visible to that doctor before
  it is active (RLS and service, with tests). History: `history_from_seq`, set when added (full, or from a message
  the adding doctor chooses). Side threads: a `staff_only` conversation with `parent_id`; the patient's policies
  already exclude it, SAL-28 adds the tests through the service and the API. System messages through `system_code`
  (`participant_added`, `participant_removed`) and `system_args` (opaque ids only); the UI renders them from its
  catalog.
- **SAL-29.** A River job in the send's transaction (`InsertTx`), args with opaque ids only; push and e-mail say
  only that there is a new message from the clinic, never the text, the sender or the patient's name. `pkg/push`
  (`Sender` port, `pushtest` fake) with FCM and APNs adapters, off by default (config); Google and Apple get an entry
  in `docs/privacy/processors.md` before they are switched on. The unread check reads `read_seq`. Webhooks: HMAC-SHA256
  over the body with the clinic's secret, a stable event id, River retries with backoff, a dead letter after the last
  attempt, no clinical content unless the clinic enables it.
- **SAL-30.** Attachments as `binary` rows with a second kind of parent (the message) next to the questionnaire
  answer; the scan's outcome gates every download, as today. Receipts and typing: Redis keys with a TTL, through the
  socket. Search: Postgres full-text over the messages the caller may see (the same RLS scopes and
  `history_from_seq`), measured against the load test's data. Terms: `terminology`'s matching, as SAL-13 does for
  transcript lines.
- **SAL-31.** Web against `fakeChat.ts` first, then the real client. The AI summary is generated on demand for a
  practitioner who may read the conversation, through a `pkg/llm` port with the fake by default; it is not stored,
  is labelled AI generated, and never replaces a message. A real engine needs the patient's AI consent (SAL-12's
  transcription consent covers the consultation, not the chat: ask the owner) and the processor's DPA.

## Checks per PR

`make lint`, `make test`, `make archcheck`, `make e2e`, plus `make web-typecheck`, `make web-test`,
`make mobile-analyze` and `make mobile-test` where they changed, and CI green. Copilot is out of review quota until
2026-12-01: a fresh subagent that has not seen the work reviews each PR's diff; valid findings are fixed before the
merge, declined ones listed in the PR.

## Copy waiting for the owner

Every new user-facing string goes to the owner first (CLAUDE.md "Copy"), listed per PR in its description and in
PROGRESS when merged.

## Personal data

| Field | Why |
| --- | --- |
| `communication.body`, `communication_revision.body` | The message itself: care between visits (GDPR Art. 9(2)(h)); part of the clinical record |
| `sender_account_id`, participants' `account_id`, `added_by`, `created_by` | Who wrote, who takes part, who added whom: the record's accountability |
| `read_seq` | Unread counts and the e-mail fallback (owner, 2026-11-12) |
| `client_message_id` | Idempotent send; an opaque UUID the client makes |
| `encounter_id` | The optional link to the consultation the conversation is about |
| `binary.conversation_id`, `conversation_participant_id`, `communication_id`, `communication_sent_at` (SAL-30) | A chat file's conversation, who uploaded it and the message it was sent with: who may read it, and it is kept with its message |
| `binary.title`, `content_type`, `size`, `sha1` of a chat file (SAL-30) | The file's name so the reader can tell files apart, and what a viewer needs; as for questionnaire files (docs/attachments/SPEC.md section 5) |
| `binary.thumbnail_id`, `thumbnail_size`, `thumbnail_wrapped_key` and the thumbnail's object (SAL-30) | A clean image's preview in the message, made by the server, encrypted like the file |
| `communication.attachment_count` (SAL-30) | How many files a message carries: a message of files alone has no text |
| Redis `chat:<conversation>:receipt:<participant>` (number; 7 days after the participant's last read) and `chat:<conversation>:typing:<participant>` (1 s) (SAL-30) | What the others see of a participant's reading and typing, ephemeral (owner, 2026-11-12) |
| `conversation_participant.organization_id` | A practitioner's clinic: who is a doctor of another clinic, whose access needs the patient's consent (SAL-28) |
| `conversation_participant.status`, `reason` | Whether an invited doctor waits for the patient's consent, and why the treating practitioner added them (a code): shown to the patient with the invitation (SAL-28) |
| `chat_consent` (`patient_id`, `organization_id`, `conversation_id`, `practitioner_role_id`, `date_time`, `revoked_at`) | Proof of the patient's consent that a doctor of another clinic reads and writes in that conversation, and of its revocation (GDPR Art. 7(1); SAL-28) |
| The AI summary (SAL-31) | Not stored: generated on demand from the latest messages for the doctor who reads them, labelled AI generated; the fake engine only until the owner decides the basis for a real model |
| `push_device.token`, `platform`, `account_id` (SAL-29) | The patient app's push token: to tell the patient of a new message. Deleted on sign-out, when the platform says it is gone, and with the account |
| `webhook_endpoint.created_by` (SAL-29) | Which admin created the clinic's webhook: accountability. The URL and the sealed secret are the clinic's, not personal data |
| `webhook_dead_letter` ids (SAL-29) | The event, conversation and message of a delivery that failed its last attempt, for the clinic's admin and support: 30 days |
