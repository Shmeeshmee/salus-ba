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
