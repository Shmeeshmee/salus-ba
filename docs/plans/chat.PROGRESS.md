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
| A removal by the treating practitioner | Ends the doctor's access (row-level security needs the active participant row) but leaves the patient's chat consent as it was; a new invitation of the same role asks the patient again, and that grant ends the earlier consent first. Open in the DPIA |
| The guest's scope | `app.organization_id` their own clinic plus `app.practitioner_role_id` their role, set by the service for every practitioner. Policy `conversation_guest` (to `salus_app`) asks `chat_guest_conversations(role, clinic)`, a `SECURITY DEFINER` function owned by the NOLOGIN role `salus_chat_guest`, which reads only the columns it needs: the conversations where the role of that clinic is an active participant of another clinic with the chat consent and the care consent active. A side thread follows its conversation. The guest also reads their own chat consents and, while one is active, the patient's care consents with that conversation's clinic (`consent_chat_guest`), so the service checks `access.GuestChat` on the same facts. Migration tests over every state (pending, granted, care revoked, consent revoked, removed) and scope (no role, no clinic, another clinic or role, a patient or account set) |
| No policy reads another through row-level security in a cycle | `chat_consent` stores the invited role's clinic (`practitioner_organization_id`, copied at the grant, held by a foreign key to `practitioner_role (id, organization_id)`), so its guest policy compares columns: reading `practitioner_role` there recursed, after main's 00043, through `practitioner_role_shared`, `encounter` and `appointment` back to `consent` (test: appointments, roles, encounters, consents, chat consents and conversations read in a clinic's, a guest's and a patient's scope, red on the old policy in all three) |
| Refusals to the guest | `not_found` on every path while not granted, also for a side thread of a granted conversation they are not in: nothing of the chat is disclosed to another clinic |
| Who adds and removes | The treating practitioner (`conversation.practitioner_role_id`) while they may write; never into a side thread, never the patient or themselves. A doctor of the clinic must be an active role of it (`practitionerrole.Lookup`), one of another clinic in its public directory (`PublicByOrganization`: listed clinic, active role and practitioner), read before the conversation's transaction. `already_participant` (409) for a role there or invited already; a role that may not be added is `invalid_request` |
| History | `full` is `history_from_seq` 0; `from` takes `from_seq` 0 to `last_seq`; the added doctor's read position starts there, so their unread count is what they may read |
| Side threads | A side thread keeps the doctors it was opened with: the conversation's active practitioners at that moment (pending ones not). A doctor added later is not in it; one removed from the conversation leaves its side threads in the same statement (a `SECURITY DEFINER` trigger, as the patient's scope that revokes does not see the threads). Opened with no body (`POST /conversations/{id}/threads`), listed as a `ConversationList` without `next`; its practitioner role is the conversation's treating one (the foreign key wants a role of the clinic), its encounter the conversation's. The patient gets 403 on the list and the creation (they know the conversation), 404 on the thread itself |
| System messages | Written through `appendMessage`, which `Send` now uses too (the same lock, numbering and time), published after the commit like a text. A revocation by the patient writes `participant_removed`, the contract's code: the UI cannot tell it from a removal by the doctor (see "Reported") |
| Audit | Entity `conversation_participant` for every add, invitation, removal, decision and revocation and for the invitations' list; the consent service audits its Consent create and update in the same transaction; a side thread's creation and list as `conversation` |
| Decision and revocation answers | 204 without a body |
| Additions to the contract | `ConversationParticipant.reason`; `AddParticipantRequest`, `ChatInvitation`, `ChatInvitationList`, `ChatInvitationDecision` schemas; error `already_participant` (409) |

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
