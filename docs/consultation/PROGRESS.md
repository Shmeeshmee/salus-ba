# Video consultations progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 videosession | done, 2026-11-05 | `consultation(1)` |
| 2 encounter | not started | |
| 3 Webhooks | not started | |
| 4 API | not started | |
| 5 Revocation and cancellation during a call | not started | |
| 6 Web | held: copy review | |
| 7 Mobile | held: copy review | |
| 8 End-to-end | not started | |

## Decisions

- 2026-11-05, owner (on approval, recorded in `SPEC.md` §2 "Changed on approval"):
  - A patient "the practitioner did not join" ended state; `video_session.patient_joined_at` kept and shown to
    the clinic through `GET /appointments/{id}/consultation`.
  - A hard cap 2 hours after the scheduled end (`time_limit`), tested.
  - A separate e2e LiveKit config is acceptable if one config cannot serve both; stop only if that fails too.
  - Decisions 2 (consent for the practitioner too) and 3 (removal through River jobs) and everything else
    approved as written.

- 2026-11-05, package 1 (runner): `video_session.closes_at` added beside SPEC §2's columns (SPEC §5 has its
  line). The never-ended retention rule needs the appointment's end, and `salus_retention` may read only the
  tables whose rule is `delete` (00014, tested), so the purger cannot join `appointment`. The owner may want to
  confirm it.
- 2026-11-05, package 1 (runner): a join after the window has closed answers `consultation_ended`, not
  `join_window_not_open` (SPEC §2 "No new token is issued once the window has closed"); before it opens,
  `join_window_not_open`. The consent is checked before the status, so the appointment a revocation cancelled
  answers `consent_required` (package 8's second journey).

## Waiting on people

- The owner, before package 6: review of every new web and mobile string, sent on its own. Packages 6 and 7
  are marked `hold` in `RUN.tsv` until the answer is recorded here.
- Before production: `SPEC.md` §8 (LiveKit domain, TLS, TURN, keys, node IP; retention and lawful basis;
  localisation).

## Log

### 2026-11-05 — plan

`SPEC.md`, this plan and `RUN.tsv` written from the owner's brief, ADR 0006 and the booking feature. 

### 2026-11-05 — plan approved

Owner approved with the changes under Decisions; `SPEC.md` and the plan updated. Committed as
`consultation(0): spec and plan`.

### 2026-11-05 — package 1: videosession

**Built.** Migration `00021_video_session.sql`: the table (SPEC §2 "Tables", plus `closes_at`, see Decisions),
row-level security (patient own: select and insert; clinic: with an active consent or for an appointment
that started before the latest revocation; webhook `app.video_room`: select and update of its one room, no
insert), grants (`salus_app` select, insert and the join/end columns only; `salus_retention` select, delete).
Ownership `video_session` → `videosession` (personal data), retention rule `delete` 30 days after `ended_at`
(or after `closes_at` when never ended), `videosession.Purger` wired into `retention_apply`; context
`consultation: [videosession]`; `sqlc.yaml` entry. `internal/videosession`: model, errors, `Repository` and
`Provider` ports (`CreateRoom`, `Token`, `RemoveParticipant`, `DeleteRoom`, `Participants`),
`repository_postgres.go`, `repository_livekit.go`, `service.go` (`Join`), `service_retention.go`.
`appointment.Service.Attendance` (the appointment with the caller's access facts, unaudited, in the caller's
scope) as the port `videosession` consumes. `auditevent.CodeOperation`. `config.LoadLiveKit` (four
`LIVEKIT_*`, each required), called by `cmd/api` and `cmd/worker` only (tested), which refuse to start
without them; the routes and jobs that use them come in packages 2 to 4. Compose: `deployments/livekit.yaml`
replaces `--dev`, healthcheck, `LIVEKIT_*` on api and worker, `TEST_LIVEKIT_*` on `tools`, which now depends on
LiveKit (so CI's `make cover` starts it). depguard rule `livekit`: `github.com/livekit` and
`github.com/twitchtv/twirp` only in `repository_livekit.go`, its test and `test/e2e` (a stray import in
`service_*.go` was tried and rejected). `make bench-join`. DPIA notes and the processors note updated;
`configs/.env.example`.

**SDK versions (pinned exact in `go.mod`).** `github.com/livekit/server-sdk-go/v2` v2.19.0,
`github.com/livekit/protocol` v1.53.2 (the current releases on 2026-11-05), `github.com/twitchtv/twirp`
v8.1.3+incompatible (LiveKit's transport, for its error codes), and through them `github.com/pion/webrtc/v4`
v4.2.23. LiveKit server stays `livekit/livekit-server:v1.13.11`.

**ICE settings (`deployments/livekit.yaml`).** `rtc.tcp_port` 7881, `rtc.udp_port` 7882 (one muxed UDP port),
`use_external_ip: false`, no `node_ip` (LiveKit offers its container address, which clients on the compose
network reach directly), `enable_loopback_candidate: true` (127.0.0.1, which a browser on the host reaches
through the published 127.0.0.1:7882/udp and 7881/tcp). One config for both, so far: the Go SDK participant in
the `tools` container reached `connected` (package 1's tests); the host browser is proved in package 6 and the
e2e network in package 8. Also: LiveKit hands clients its default public STUN servers (the Go SDK tried
`stun.l.google.com`); harmless locally, but production sets its own STUN/TURN (SPEC §8).

**Tests seen failing first.**
- config: `undefined: config.LoadLiveKit`, `config.LiveKit` (build failed).
- cmd/api, cmd/worker: `TestRun_FailsWithoutTheLiveKitSettings`: `run() error = postgres: ping: failed to
  connect …, want one naming LIVEKIT_API_SECRET`.
- migrations: the `TestVideoSession_*` tests: `relation "video_session" does not exist (SQLSTATE 42P01)`; then
  `TestRetention_RoleIsGrantedSelectAndDeleteOnExactlyTheTablesWhoseRuleIsDelete`: `salus_retention holds …
  video_session:[DELETE SELECT], want …` without it (until the retention rule).
- appointment: `undefined: appointment.Attendance`, `f.svc.Attendance` (build failed).
- videosession service: `undefined: videosession.Service`, `NewService`, `Ports` (build failed).
- videosession repository: `undefined: videosession.NewPostgresRepository`; purger: `undefined:
  videosession.NewPurger` (build failed).
- worker: `New() error = retention: a rule that removes rows has no applier, or an applier has no such rule:
  video_session`; then the retention tests that list every table (`internal/retention`, `internal/worker`)
  extended with `video_session`.
- videosession provider: `undefined: videosession.NewLiveKitProvider` (build failed).

**What the tests cover.** The window boundaries (10 min before the start and 30 min after the end, ± 1 s);
patient, practitioner of the role, another practitioner, the admin, another clinic's admin, anonymous; consent
active, revoked (booked and cancelled), never; `booked`, `cancelled`, `fulfilled`, `noshow`; an ended session;
each outcome's error and its one `AuditEvent` (`operation`, `E`, purpose `PATRQT`/`TREAT`, the entity only when
the caller may see the appointment); the practitioner's `peerName` writes a `Patient` read, the patient's does
not; one session per appointment, the room created on every request outside the transaction with
`max_participants` 2 and the empty timeout to the hard cap; random `r_`/`p_` names (26 base32 characters, never
one of our ids); failures leave no session or event and no name, token, room or identity in the error text.
Real database: two first requests at once give one row (repository) and one session and one room through the
real services (`internal/api/consultation_test.go`, with the audit events); RLS per scope; the purger. Real
LiveKit: the decoded token (one room, camera and microphone, subscribe, no data, no name, metadata or
attributes, 5 minutes), the room read back (2, no metadata, empty and departure timeouts), no room from a token
alone (`auto_create` off), a join with the service's token, a second connection with the same identity replacing
the first (`duplicate identity`, waited for as an event), remove, delete and list idempotent on what is gone,
errors without the room or identity. Package 1 has no log line of its own (no handler, no job): the "no
token, room, identity or name" check is on every error; the log tests come with the handler and the jobs.

**Benchmark (`make bench-join`, steady state: the session exists; real Postgres, i7-14700HX).**

| Provider | ns/op (3 runs) |
| --- | --- |
| compose LiveKit | 4,354,933 · 4,173,330 · 4,078,160 |
| fake | 3,328,715 · 3,224,800 · 3,091,316 |

About 3.2 ms is ours (the transaction: access facts, session, audit; and the directory read), about 1 ms is
LiveKit's `CreateRoom` and the token. The HTTP latency comes in package 4.

**Checks.** `make lint`: 0 issues, migrations unchanged. `make archcheck`: passes. `make test`: passes.
`make cover`: every package at 80% or more (`internal/videosession` 96.5%, `internal/appointment` 92.9%,
total 94.8%). `make sqlc`: no diff. Stack: `make up` with migration 21 applied, LiveKit on its config (healthy),
api and worker up with the `LIVEKIT_*` settings, `/healthz` 200.

**Deferred.** The window-close job (package 2), webhooks and `app.video_room` in the service (3), the HTTP
endpoints and the HTTP latency (4), removal (5). LiveKit's default public STUN servers for clients: production
STUN/TURN with SPEC §8.

**Dev stack left.** Migration 00021 applied (empty `video_session`); LiveKit recreated on
`deployments/livekit.yaml`; the stack rebuilt and up as it was.

