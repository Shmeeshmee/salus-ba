# Video consultations progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 videosession | done, 2026-11-05 | `consultation(1)` |
| 2 encounter | done, 2026-11-05 | `consultation(2)` |
| 3 Webhooks | done, 2026-11-05 | `consultation(3)` |
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
- 2026-11-05, package 2 (runner): `Encounter.Start` is a system call with no caller, so it reads the
  appointment through a new `appointment.Service.Lookup` (in the caller's transaction and scope, unaudited
  like `patient.Lookup`); the encounter audits its own create. `Start` does not check the appointment's
  status (no SPEC rule); package 3 ignores joins after `ended_at`.
- 2026-11-05, package 2 (runner): `encounter.Finish` records the outcome only when it finishes an encounter
  in progress, and `videosession.end` calls it only when the practitioner joined; so a session ended
  `practitioner_absent` leaves the appointment `booked`, and a repeat records nothing. The session's end is
  audited as an `operation` (`E`) on `Appointment/<id>` by the agent (`HOPERAT`); the job's check of an
  occupied room, once a minute, reads only its own row and writes nothing, and is not audited.
  `RecordConsultationOutcome` audits only when it changes the status.
- 2026-11-05, package 2 (runner): `Encounter` search scoping. A practitioner always gets only their own
  role's encounters, without having to name themselves (the `Appointment` search requires the name); naming
  another practitioner is `ErrForbidden`, audited. A patient's own `patient` replaces whatever they name, as in
  the `Appointment` search.
- 2026-11-05, package 2 (runner): the hard cap is `closes_at` + 90 minutes (the scheduled end + 2 hours), so
  the job reads only `video_session`. A session already ended when the job runs only has its room deleted
  again (a retry after a failed delete heals); one the clinic's scope no longer shows (after a revocation, an
  appointment that had not started, which the revocation cancels) is left to package 5. The job's agent name
  is its kind, `consultation_window_close`, like `consent_revoked`. `worker.New` takes the video provider;
  `cmd/worker` builds it from `LIVEKIT_*`.

- 2026-11-05, package 3 (runner): the patient's access is checked again on every patient join, not only the
  first (a reconnect after a revocation is removed too), in the clinic's scope: the appointment through
  `appointment.Lookup` (the scope no longer showing it, or a status other than `booked`, removes the patient)
  and the consent through a new `Consents` port on `consent.Service.State` (SPEC lists `consent` among
  `videosession`'s dependencies). The removal is a provider call after the commit and is not audited on its
  own; the join it follows is. Every join the webhook records is audited as an `operation` (`E`) on
  `Appointment/<id>` by `livekit_webhook` (`HOPERAT`), like the session's end.
- 2026-11-05, package 3 (runner): `participant_left` and `room_finished` after the window has closed apply the
  window-close job's rules (`closing`): with the practitioner never joined the session ends
  `practitioner_absent`, not `window_closed`, so package 4's `practitioner_did_not_join` holds whichever path
  ends it; past the hard cap `time_limit`; with someone still in the room nothing. Before the window closes
  they ask nothing of LiveKit and change nothing.
- 2026-11-05, package 3 (runner): "every order reaches the same state" is tested inside the window. After the
  window has closed the result rightly depends on the order: a leave that arrives before the practitioner's
  join ends the session `practitioner_absent`, and SPEC §2 ignores joins after the end. A join keeps the first
  join delivered (SPEC's "set if null"), not the earliest time.
- 2026-11-05, package 3 (runner): the practitioner's join on an appointment the clinic's scope no longer shows
  is recorded and starts no `Encounter` (`encounter.ErrNotFound` is no effect, so LiveKit does not retry
  forever). An unknown identity in a known room (join or leave) is `unknown room`, like an unknown room. A body
  over 64 KiB is 400 `invalid_request`, as the other handlers' oversized bodies; a signed body that is not an
  event is a 500 (LiveKit retries). The domain's event kinds are the three the consultation acts on and
  `other`, so the log line carries only fixed values.

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

### 2026-11-05 — package 2: encounter

**Built.** Migration `00022_encounter.sql`: the table (SPEC §2 "Tables", "Encounter"; one per appointment,
the role's foreign key with its clinic, `finished` exactly when `period_end` is set and not before the start),
row-level security (patient own: select only; clinic: with an active consent or for an appointment that
started before the latest revocation; no webhook scope), grants (`salus_app` select, insert and the finish
columns only). Ownership `encounter` → `encounter` (personal data), retention rule `keep`; context
`consultation: [encounter, videosession]`; `sqlc.yaml` entry. `internal/encounter`: model, errors,
`Repository`, `repository_postgres.go`, `service.go` (`Start`, `Finish`, audited with the agent),
`service_read.go` (`Read`, `Search` by appointment, patient and practitioner). `internal/appointment`:
`Lookup` and `RecordConsultationOutcome` (SPEC §2 "Outcome"). `internal/videosession`: `Open` says whether it
created the session, `ByID`, `End`; the `Jobs` port (`repository_river.go`, `consultation_window_close` on the
default queue, enqueued in the first join's transaction at `closes_at`) and the `Encounters` port;
`service_end.go`: `CloseWindow` (SPEC §2 "The window closes", "The hard cap") and `end` (SPEC §2 "Ending",
used by the job now and by packages 4 and 5). `internal/worker`: the `consultation_window_close` worker
(snoozes with `river.JobSnooze`, which River 0.51 does not count as an attempt), registered in the worker and,
without services, in the scheduler. DPIA notes.

**Tests seen failing first.**
- migrations: the `TestEncounter_*` tests: `relation "encounter" does not exist (SQLSTATE 42P01)`.
- archcheck: `retention-rule-missing: table encounter holds personal data and has no rule`.
- appointment: `f.svc.Lookup undefined`, `f.svc.RecordConsultationOutcome undefined` (build failed).
- encounter: `no non-test Go files in /src/internal/encounter` (build failed); then
  `TestRead_FailsWithTheBackend`: `appointments: Read() = encounter: not found, want its error` (the first
  cut turned a backend error into a not-found; fixed); then `undefined: encounter.NewPostgresRepository`.
- videosession: `*fakeRepository does not implement videosession.Repository (wrong type for method Open)`
  (build failed: `Open`'s flag, `ByID`, `End`, the `Jobs` and `Encounters` ports, `CloseWindow`,
  `NewRiverJobs`).
- worker: `too many arguments in call to worker.New` (build failed). The snooze test was written after the
  worker; it was seen failing by removing the `JobSnooze` return: `the job ended job_completed, want snoozed
  a minute`.
- internal/api: the one-job assertion on two concurrent first joins was added after the enqueue; seen failing
  without it: `0 window-close jobs …; want one at the window's close`.

**What the tests cover.** `Start` twice gives one encounter with the first time and one create event; real
database: two `Start`s at once create one row, created once. `Finish` with and without the patient records
`fulfilled` and `noshow`; twice changes, records and audits nothing the second time; a never-started
encounter is not created; a finished encounter never changes (a later `Start` gives it back as it is); a
failure rolls back. `RecordConsultationOutcome`: `booked` → `fulfilled`/`noshow` audited by the agent; a
`fulfilled`, `noshow` or `cancelled` appointment kept, unaudited. `Read`: the patient, the practitioner of the
role, the admin, the name level after a revocation; another patient, no profile, another clinic, an unknown
id and a practitioner without access are 404, audited without the entity. `Search`: the patient (also naming
another), by appointment and practitioner, the practitioner (also by patient), the second practitioner, the
admin (by practitioner, by appointment), another clinic; naming another practitioner is forbidden, audited;
each returned encounter audited, none gives one event. RLS per scope, writes by scope, the app role's
grants and the constraints (migrations). The window-close job: the first join enqueues it once (also for two
concurrent first joins, real database), in the join's transaction, at the close; an empty room ends the
session `window_closed`, finishes the encounter with whether the patient joined and deletes the room after
the commit; an occupied room snoozes a minute; the practitioner never joined: `practitioner_absent`, no
encounter, the appointment `booked`, `patient_joined_at` kept (also through the real worker); the hard cap at
−61 s, −20 s, −1 s (snoozes of 60 s, 20 s, 1 s: never past it), at the cap and +1 s (`time_limit`, the
occupied room deleted, the encounter finished); a delete failure leaves the end committed and fails the job
for River's retry, which deletes the room again and changes nothing else; an ended session only has its room
deleted; out of scope, nothing; failures before the commit change nothing and carry no room or identity.
Real worker on the compose LiveKit: window closed → session `window_closed`, encounter `finished`,
appointment `fulfilled`, three audit events by the job, a retry completes and changes nothing; snoozed with
an occupied room; the log holds the session id and never a name, e-mail, room or identity.

**Checks.** `make lint`: 0 issues, migrations unchanged. `make archcheck`: passes. `make test`: passes.
`make cover`: every package at 80% or more (`internal/encounter` 96.1%, `internal/videosession` 95.4%,
`internal/appointment` 92.9%, `internal/worker` 96.2%, total 94.8%). `make sqlc`: no diff after the
regeneration (every `*db/models.go` gains the `Encounter` row type, as in package 1).

**Deferred.** The practitioner's end (`POST …/consultation/end`, package 4) and the cancellation's end
(package 5) call `end` with their own agent. `Encounter.Start` from the practitioner's `participant_joined`
(package 3). FHIR read and search over HTTP, the `CapabilityStatement` and the golden files (package 4).

**Dev stack left.** Migration 00022 applied (`encounter` empty). The stack rebuilt with `make up` and running
as before; api `/readyz` 200, worker and scheduler started with the new job kind.

### 2026-11-05 — package 3: Webhooks

**Built.** `videosession`: the domain `Event` (kind, room, identity, time) and `EventKind`
(`participant_joined`, `participant_left`, `room_finished`, `other`); `ErrInvalidSignature`, `ErrUnknownRoom`.
`Provider.ParseWebhook` in `repository_livekit.go`: LiveKit's own receiver check (its `webhook.Receive` takes an
`*http.Request`, so it is done with `auth.ParseAPIToken` and `Verify`): issued by our API key, signed with our
secret, unexpired (LiveKit's minute of leeway), the `sha256` claim equal to the body's, compared in constant
time, and only then the body read with LiveKit's protojson; the participant's join time in milliseconds when
LiveKit gives them. `Repository`: `SetRoom` (`app.video_room`), `ByRoom`, `Joined` (two sqlc queries, "set if
null and not ended"). `service_webhook.go`: `HandleWebhook` (SPEC §2 "Idempotent, order-free handling",
"Webhook scope"; Decisions above). The `Encounters` port gains `Start`, the `Appointments` port `Lookup`, and a
`Consents` port. `handler_http.go`: `POST /webhooks/livekit` (64 KiB, 401 `unauthenticated` logged with the
request id only, 200 `{}` for every handled event and for an unknown room, 500 through `httpx.InternalError`;
one log line with the request id and the kind). `internal/api`: `New` takes `config.LiveKit` (from
`cmd/api`'s `LoadLiveKit`) and builds the encounter and videosession services; the route is mounted. The
session middleware leaves the webhook alone: LiveKit's `Authorization` has no `Bearer` prefix and there is no
cookie, so there is no CSRF check either (tested through `api.New`). `api/openapi.yaml`: the path, the
`livekitWebhook` scheme, `LiveKitWebhookEvent`, `WebhookReceived`; `make gen-clients`.
`deployments/livekit.yaml`: `webhook` to `http://api:8080/webhooks/livekit` with `devkey`. DPIA notes.

**Tests seen failing first.**
- videosession provider: `undefined: videosession.Event`, `videosession.EventParticipantJoined` (build failed).
  Then `ParseWebhook() with an expired one = {…}, <nil>; want ErrInvalidSignature`: a test bug, the SDK signs a
  negative validity as its default 6 hours; the expired token is now written by hand as LiveKit's HS256 JWT, and
  the same token unexpired was seen accepted, so the expiry is what refuses it.
- videosession service: `unknown field Consents in struct literal of type videosession.Ports`,
  `f.svc.HandleWebhook undefined` (build failed).
- videosession repository: `*postgresRepository does not implement Repository (missing method ByRoom)` (build
  failed).
- videosession handler: `undefined: videosession.NewHandler` (build failed).
- internal/api: `too many arguments in call to api.New` (build failed).
- internal/api, the real-services webhook tests, written after the service: seen failing by taking the code out
  and restored. Without the removal: `removed [], want the patient`; without the window check on a leave:
  `order 26 (…) reached … practitioner_absent …`; without the `IS NULL` guards of the join queries: `every
  order reached … | 5, want … | 3` and, in the repository test, `Joined() changed [true true true true]`.

**What the tests cover.** The provider: a signed event of each kind read as the domain's (join time in ms, in
seconds, the event's time otherwise; other kinds `other`); a wrong key, a wrong secret, a changed body, no
header, an expired token, a token without the body hash, a header that is not a token and a garbled one are
`ErrInvalidSignature` with no event; a signed body that is not an event fails without its content. The service
(fakes): the practitioner's join recorded at LiveKit's time, the `Encounter` started then in the clinic's
scope, audited, a repeat changes nothing; the patient's join recorded and audited once, access checked on each;
a patient removed after the commit when the consent is revoked, the appointment cancelled, marked by hand or out
of scope, again on a repeat; a failed removal fails the webhook and the redelivery removes again; joins after the
end ignored; no `Encounter` out of scope; a failure in the join, the encounter, the audit, the session read or
the consent check rolls back and fails; leaving before the window closes asks and changes nothing; after it,
`window_closed` with the room empty (encounter finished, room deleted after the commit), nothing when occupied,
`practitioner_absent`, `time_limit` at the cap, for a leave and for `room_finished`; a late leave after the end
changes and deletes nothing; a failed participants check fails; other kinds read nothing; unknown rooms and
identities are `ErrUnknownRoom` with no effect; an unsigned webhook handles nothing. The repository (real
database, `salus_app`): the webhook's scope shows the one session of its room, of any clinic; `Joined` records
each role's first join only, sets `updated_at`, never after the end. The handler (`httptest`): 200 `{}` with the
body and header passed as sent, `unknown room` 200, 401 for a bad and a missing signature, the fixed 500, the
64 KiB limit; each log is exactly one line with the request id (and the kind), never the room, identity or
token. Real services (`internal/api`): every one of the 120 orders of the practitioner's and the patient's
joins and leaves and `room_finished`, each delivered twice, reaches the same state (both joins, `Encounter`
in progress from the practitioner's join, appointment `booked`, three webhook events); after the window, the
last one leaving ends the consultation (`window_closed`, `Encounter` finished, `fulfilled`, six webhook
events), and a repeat changes nothing; a patient joining after a revocation is removed, their join recorded;
an unknown room changes nothing. Through `api.New`: no signature and a session-style `Bearer` are 401, a
signed event for an unknown room is 200, the log holds nothing from the body.

**Checks.** `make lint`: 0 issues, migrations unchanged. `make archcheck`: passes. `make test`: passes.
`make cover`: every package at 80% or more (`internal/videosession` 95.9%, `internal/encounter` 96.1%,
`internal/api` 91.1%, total 94.8%). `make sqlc`: regenerated (the three queries). `make gen-clients`: run, and a
second run leaves no diff. Stack: `make up` (api rebuilt), LiveKit recreated on the new config; a temporary Go
program in the `tools` container (deleted afterwards) seeded a synthetic appointment starting in 5 minutes,
took tokens from the service as `New` builds it, and joined the compose LiveKit as the practitioner and then the
patient with the Go SDK. LiveKit sent `room_started`, two `participant_joined`, two `participant_left` and
`room_finished` to the api, each `200 OK`; the api's log shows `livekit webhook` with the kind and the request id
only for each; the `encounter` row appeared `in-progress` on the practitioner's join and `patient_joined_at` on
the patient's; the audit trail of the synthetic clinic: the practitioner's and the patient's joins (`operation`),
the role and `Patient` reads, then `livekit_webhook` `operation` (practitioner joined), `create Encounter`,
`operation` (patient joined). LiveKit's own log names the random room and identities only.

**Deferred.** The join, end and status endpoints and the FHIR `Encounter` (package 4); removal on cancellation
and revocation (5).

**Dev stack left.** No migration. The stack rebuilt with `make up`; LiveKit recreated on
`deployments/livekit.yaml` with the webhook. The stack check's synthetic rows (clinic, practitioner, role,
accounts, patient, consent, appointment, session, encounter, window-close job) were deleted; its seven
`audit_event` rows stay, as the table is append-only. The room was deleted.

