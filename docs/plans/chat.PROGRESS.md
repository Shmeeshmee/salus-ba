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

## SAL-29: push notifications, the e-mail fallback, chat hours and webhooks

Built: `pkg/push` (the `Sender` port, `ByPlatform`), its fake `pkg/push/pushtest` (the default), and the adapters
`pkg/push/fcm` (FCM HTTP v1 on the service account's OAuth assertion) and `pkg/push/apns` (APNs over HTTP/2 with an
ES256 provider token), both tested against `httptest` servers and off until the owner adds credentials
(`config.LoadPush`, docs/deploy/push.md). Migration 00043: `push_device`, `organization_chat_settings`,
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
