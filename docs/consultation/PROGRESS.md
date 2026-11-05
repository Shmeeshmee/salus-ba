# Video consultations progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 videosession | done, 2026-11-05 | `consultation(1)` |
| 2 encounter | done, 2026-11-05 | `consultation(2)` |
| 3 Webhooks | done, 2026-11-05 | `consultation(3)` |
| 4 API | done, 2026-11-05 | `consultation(4)` |
| 5 Revocation and cancellation during a call | done, 2026-11-05 | `consultation(5)` |
| 5.1 Owner changes after package 5 | done, 2026-11-05 | `consultation(5.1)` |
| 6 Web | done, 2026-11-05 | `consultation(6)` |
| 7 Mobile | not started | |
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

- 2026-11-05, package 4 (runner): the practitioner's end before they joined (no session, or a session without
  their join) is 409 `consultation_not_started`, a new code: SPEC §2 allows the end "at any time after they
  joined" and names no code for before. The end answers 200 with the consultation's status (the same schema as
  `GET /appointments/{id}/consultation`), also when repeated or when another path ended it first.
- 2026-11-05, package 4 (runner): the end is audited twice, like the other paths: the practitioner's request as
  an `operation` (`E`) on `Appointment/<id>` with their account (`TREAT`), and the writes it causes (the
  session's end, the `Encounter`'s finish, the outcome) with the agent `consultation_end` (`HOPERAT`), as the
  window-close job and the webhook audit theirs.
- 2026-11-05, package 4 (runner): `practitioner_did_not_join` answers a join by either side on a session ended
  `practitioner_absent` (SPEC: "a join request on an ended session"), also when the clinic cancelled the
  appointment afterwards. The consent is still checked first, so a revoked consent answers `consent_required`.
  Without a session (nobody asked for a token in the window) it stays `consultation_ended`.
- 2026-11-05, package 4 (runner): the status's `state` is `not_started` while there is no session (nobody asked
  for a token), `open` from the first token request until the end (whether or not anyone connected), `ended`
  after. The times and `endReason` are left out until they are set (optional in `openapi.yaml`, not `null`). It
  reads the session in the caller's own scope, so after a revocation the clinic sees the consultations of the
  appointments that had started before it, and `not_started` for the others (which the revocation cancels).
- 2026-11-05, package 4 (runner): the latency test sits in `test/e2e/consultation_latency_test.go` behind the
  build tags `e2e && loadtest`, so `make e2e` does not run it and it reuses the suite's helpers; `make lint` now
  vets and lints `test/` with both tags. It books through the API and then moves the appointment's start 5
  minutes ahead as the owner, since the generated slots start at least an hour ahead.

- 2026-11-05, package 5 (runner): the session of an appointment that the revocation cancels (it had not
  started) is ended by `EndPatientAccess`, not by `EndAccess`. After the revocation the clinic's scope, which
  `consultation_access_end` reads in (it carries the appointment and clinic ids only, SPEC §2), no longer shows
  that session; this is the case package 2 left open. The revocation job has the patient's id, so it reads the
  patient's sessions with the clinic in the patient's scope (`ClosingAfter`: those whose window closes after
  `now` − 90 minutes, so whose room may still exist), removes the patient from each, and ends `cancelled` the
  sessions whose appointment is cancelled. The `consultation_access_end` jobs that the same cancellation queued
  then find nothing and do nothing.
- 2026-11-05, package 5 (runner): `end` now writes the session's end in its room's scope (`app.video_room`) as
  well as its clinic's, so a system path can end a session it has already read when the clinic's scope hides it.
  The room's scope shows that one session only, the same row the webhook may update. No migration and no new
  policy. The `Encounter` and the `Appointment` are still written in the clinic's scope.
- 2026-11-05, package 5 (runner): `EndAccess(ctx, appointmentID, org)` takes the clinic id from the job, like
  `CloseWindow`. It removes the patient and, when the appointment is `cancelled`, ends the session; for any
  other status it only removes the patient. Repeating either job writes and audits nothing; each repeat only asks
  LiveKit again to remove and delete what is already gone, so a retry after a failed delete heals. Removing the
  patient is not audited on its own (package 3's rule); ending the session is audited, with the job's kind as
  agent (`consultation_access_end` or `consent_revoked`, `HOPERAT`).
- 2026-11-05, package 5 (runner): a removal that fails in one room does not keep the patient in the others.
  `EndPatientAccess` tries every room and returns the joined error for River to retry. The revocation worker
  calls `EndPatientAccess` only after `CancelForRevokedConsent` succeeds (SPEC order), because it needs the
  statuses that call writes.

- 2026-11-05, owner (after package 5, recorded in `SPEC.md` §2 "Changed after package 5"):
  - The `Encounter` left `in-progress` by a revocation before the start: option (a), the room-based access rule
    on `encounter` in a new migration, finishing it with `period.end` set; package 5.1, test first. Then the dev
    database is wiped (`make down` with volumes, `make up`).
  - Every patient removal is audited (question 6).
  - Approved as described: `video_session.closes_at`; a join after the window closes is `consultation_ended`;
    the window-close job's occupied-room check is not audited; a practitioner's `Encounter` search returns only
    their own role's; a leave after the close without the practitioner ends `practitioner_absent`;
    `consultation_not_started`; `practitioner_did_not_join` to the practitioner too; ending audited twice, with
    agent `consultation_end`; a revocation-cancelled session ended by `consent_revoked` in the room's scope.
  - The text: approved with the owner's changes, recorded in `COPY.md`, the source for packages 6 and 7.
    ADR 0006 notes that "The consultation is not recorded" must change if recording is added. Packages 6 and 7
    are no longer held.

- 2026-11-05, package 5.1 (agent's reading, within the owner's decision): a removal is audited when LiveKit
  lists the patient in the room, before LiveKit is asked to remove them, in its own short transaction after the
  path's decision. No column records a removal, and the owner's change adds no column, so the room itself tells a
  first removal from a repeat: the repeat finds the patient gone and audits nothing, and LiveKit is still asked
  again (package 5's healing retry). Audited first, so a failed audit removes nobody and River or LiveKit retries;
  a removal that then fails is audited again on the retry. A patient who connects in the milliseconds between the
  list and the removal is removed without an event of their own; the session's audit trail still has the path's
  other events. The webhook's re-check uses the same rule.

- 2026-11-05, package 6 (runner): the join button is the practitioner's only, on the calendar card and the
  appointment detail (SPEC §2: admins cannot join, and patients join from the mobile app). The call screen is
  `/appointments/<id>/call`, a path of the app; `/appointments/<id>/consultation` is the JSON status the dev
  server proxies. The detail shows W2 before the window opens; the calendar shows nothing then. Both read the
  clock again every 15 seconds, so the button appears and goes while the screen is open.
- 2026-11-05, package 6 (runner): W18a needs to know why the patient left the room, and LiveKit tells the other
  side nothing about it. When the patient leaves, the web app asks the join endpoint once more: `consent_required`
  shows W18a, anything else W18. No API change; that request is audited like any join, and a 200 issues a token
  the app does not use. W18b, W29 and W28 come from the status's `endReason` and `patientJoinedAt` once the room
  is deleted.
- 2026-11-05, package 6 (runner): a revocation before the appointment's start, with both already in the room,
  cancels the appointment, ends the session `cancelled` and deletes the room. The clinic's scope then no longer
  shows that consultation (SPEC §2 "Tables"), so the practitioner's status read is 404 and the screen shows W27
  without a second line (seen in the walk). The owner may want W18a or W18b there; see the questions below.
- 2026-11-05, package 6 (runner): COPY.md has no text for some cases, so none was written. A practitioner who
  joins from a second tab replaces the first connection (LiveKit's rule for a repeated identity), and the first
  tab shows W23 with "Join again" instead of rejoining on its own, so two tabs do not take turns. When `peerName`
  is empty, the lines that need the name (W14, W16, W17, W18) are left out. A device the browser lists without
  a label is not offered in the picker. Any device error other than a denial (no device, device in use) shows
  W12.
- 2026-11-05, package 6 (runner): W21 or W21b follows whether the patient joined: seen in this connection, or
  `patientJoinedAt` from the status read right after connecting (a patient who came before a reload counts). W6
  shows after 5 seconds below 0.02 RMS. A level that cannot be measured (the browser's audio context before any
  click on the page) counts as neither sound nor silence. Join waits until both devices have opened or failed
  (the walk found a join with both off when it was clicked first). A lost connection (any LiveKit disconnect
  other than leaving, a closed room or a second tab) asks for a new token and rejoins once; if that fails it
  shows W25 with "Try again". A room closed while the status is still `open` is rejoined the same way.

## Waiting on people

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

### 2026-11-05 — package 4: API

**Built.** `videosession`: `Status` and `End` (`service_status.go`; SPEC §2 "Consultation status", "Ending"),
the join's `practitioner_did_not_join` (`ErrPractitionerDidNotJoin`) and the end's `consultation_not_started`
(`ErrNotStarted`), the `Status` model and its `State`s, `Repository.ByAppointment` (the existing
`SessionByAppointment` query); `end` split so its transaction (`ending`) also serves the practitioner's end, which
returns the session as it ended (read again when another path ended it first). `handler_http.go`: `POST
/appointments/{id}/consultation/join`, `POST /appointments/{id}/consultation/end` and `GET
/appointments/{id}/consultation`, `no-store`, errors `{"error": "<code>"}`. `encounter/handler_fhir.go`:
`GET /fhir/R4/Encounter/{id}` (ETag) and `GET /fhir/R4/Encounter?appointment=&patient=&practitioner=` (each once,
a reference or a bare id; anything else 400 `not-supported`), 404 and 403 as `OperationOutcome`s; golden files for
an in-progress and a finished `Encounter`, a search `Bundle`, and the 404, 403, 401 and 400 outcomes.
`api/fhir/CapabilityStatement.json`: `Encounter` with `read`, `search-type` and the three parameters (R4's own
`Encounter-appointment`, `clinical-patient`, `Encounter-practitioner`); the router's golden copy updated with it.
`internal/api`: `newConsultation` builds the `Encounter` service and the sessions; both handlers mounted on the
`appointment` caller. `api/openapi.yaml`: the three paths, `ConsultationJoin`, `ConsultationStatus`, an
`AppointmentID` parameter and the four new error codes; `make gen-clients`. `make bench-join-http` and
`test/e2e/consultation_latency_test.go`. DPIA notes.

**Tests seen failing first.**
- videosession service: `undefined: videosession.Status`, `videosession.StateNotStarted`, `f.svc.Status undefined`
  (build failed; the same build named `End`, `ErrPractitionerDidNotJoin` and `ErrNotStarted`).
- videosession repository: `*postgresRepository does not implement Repository (missing method ByAppointment)`
  (build failed); the `ByAppointment` test then seen failing against a stub that always answered not found:
  `ByAppointment() in the patient's scope = {…}, videosession: not found; want {… Room:r_byappt …}`.
- videosession handler: `too many arguments in call to videosession.NewHandler` (build failed).
- encounter handler: `undefined: encounter.NewHandler` (build failed).
- api/fhir: `Encounter = "", want "read search-type param:appointment:reference param:patient:reference
  param:practitioner:reference "`; then the router's `testdata/metadata.json differs from the response`, rewritten
  with `-update` on purpose (identical to the CapabilityStatement again).
- internal/api: `not enough arguments in call to videosession.NewHandler` (app.go, build failed); with the caller
  wired and the Encounter routes not yet mounted, `GET /fhir/R4/Encounter?appointment=… = 404 … "This resource or
  interaction is not supported."`. Two test bugs found on the way: the synthetic join time lay after the real
  end, so finishing the `Encounter` broke its `encounter_period` check (500 on the end; the join time now lies
  inside the window, before the end), and the count of reads of the appointment left out the patient's FHIR read.

**What the tests cover.** Service (fakes): the status for the patient, the practitioner and the admin, each in
their own scope and audited as a read of the appointment, for no session, open (nobody or both joined) and ended
`practitioner_absent`, at the name level after a revocation; another patient, another practitioner, another
clinic and anonymous are 404, audited without the entity; failures audit nothing. The end by the practitioner,
with and without the patient: ended `practitioner`, the encounter finished in the clinic's scope with whether the
patient joined, the room deleted after the commit, the request and the agent's end audited; repeated: the same
answer, nothing finished again, the room deleted again; after a concurrent end: answered as it ended; after a
revocation it still ends; the patient, the admin, another practitioner, another clinic, another patient and
anonymous get 404, audited, nothing ends; before the practitioner joined `ErrNotStarted`, audited; a failure in
the appointment, the session, the end, the encounter or the audit changes nothing; a failed delete leaves the end
committed. The join after the window: `practitioner_did_not_join` for both sides on a `practitioner_absent`
session (also after a cancellation), `consultation_ended` for `window_closed`, `time_limit` and no session, no
session opened, audited. Repository (real database, `salus_app`): `ByAppointment` in the patient's and the
clinic's scope; another clinic, another patient, no session and a non-UUID are not found. Handlers (`httptest`):
the join's exact JSON and `no-store`; the status and the end's JSON with the times and reason only when set; each
of the six error codes on all three routes, 401 signed out without calling the service, 500 when the caller
fails; the log holds no token or name and the error only on a 500. FHIR handler: the two `Encounter`s and the
search against their golden files, the filter passed, an empty search; twelve unsupported queries (unknown,
repeated, wrong type, several values, unparsable) 400 without calling the service; 404, 403, 401 and the fixed
500. Through `api.New` on the real database and the compose LiveKit: `not_started` for the three, 404 for another
patient's status and join, 404 for the admin's join, `consultation_not_started` before the practitioner joined,
both joins with the URL, a token, the other's name, the role and `no-store`; signed `participant_joined` webhooks;
`open` with the join times; the `Encounter` found and read `in-progress` by the patient, the practitioner and the
admin (six audit events), 404 and an empty search for another patient, 400 for `status`; the end 404 for the
patient and the admin, 200 `ended`/`practitioner` for the practitioner and the same again; the `Encounter`
`finished`, the appointment `fulfilled`, a later join `consultation_ended`; the audit counts; the log holds no
token or name. Refusals through `api.New`: `join_window_not_open` two hours ahead, `practitioner_did_not_join`
for both on a session ended without the practitioner (the admin's status shows it with the patient's join), and
`consent_required` after the revocation.

**HTTP latency (`make bench-join-http`; the running stack: the api container on the compose Postgres and
LiveKit, i7-14700HX, 28 threads).** 1,000 join requests, 10 at a time, alternating the patient and the
practitioner, after one join each opened the session and the room (steady state: session check, access facts,
audit, the other's name, LiveKit's `CreateRoom` and the token). Three runs; nothing else used the shared LiveKit
meanwhile (no `make test`, whose rooms would send webhooks to the dev api).

| Run | Throughput | All p50 / p95 / p99 | Patient p50 / p95 / p99 | Practitioner p50 / p95 / p99 |
| --- | --- | --- | --- | --- |
| 1 | 1,490/s | 5.98 / 12.38 / 18.14 ms | 6.80 / 13.10 / 18.38 ms | 5.27 / 11.25 / 16.87 ms |
| 2 | 1,140/s | 8.01 / 13.89 / 18.86 ms | 8.75 / 14.17 / 18.73 ms | 7.18 / 13.45 / 19.50 ms |
| 3 | 1,369/s | 6.24 / 13.46 / 27.96 ms | 7.15 / 13.46 / 28.02 ms | 5.59 / 13.41 / 25.21 ms |

Next to package 1's service benchmark (about 4.1 to 4.4 ms per join, one at a time, of which about 1 ms is
LiveKit's), the HTTP path at 10 concurrent requests adds the session check, the access log and the queueing for
the database pool. The patient's join is about 1.5 ms slower at the median: its public-name read of the
practitioner is a second transaction after the join's. No change is made for speed here.

**Checks.** `make lint`: 0 issues (both modules), migrations unchanged. `make archcheck`: passes. `make test`:
passes. `make cover`: every package at 80% or more (`internal/videosession` 96.2%, `internal/encounter` 97.2%,
`internal/api` 91.1%, `internal/api/router` 100%, `api/fhir` 100%, total 94.9%). `make fhir-validate`: every file 0
errors (the new `Encounter`s, the search `Bundle`, the outcomes and the CapabilityStatement; only the dom-6
narrative warning every golden file has). `make gen-clients`: run, and a second run leaves no diff.
`make bench-join-http`: three runs above; the api's log after them holds the route pattern, status, duration and
request id for each join and no token, room, identity or name.

**Deferred.** Removal on cancellation and revocation (package 5). The web app's appointment detail showing who
joined, and the clients' use of the three endpoints (packages 6 and 7).

**Dev stack left.** No migration. The api rebuilt by `make up` (through `make bench-join-http`) and running with
this package's routes. The three latency runs left their synthetic e2e data in the dev database, as `make e2e`
runs do: three clinics with an admin and a practitioner, three patients with a consent and one appointment each,
their sessions and the rooms in the compose LiveKit (nobody connected, so each window-close job ends its session
`practitioner_absent` and deletes its room when the window closes, about 65 minutes after the run), their slots,
jobs and audit events.


### 2026-11-05 — package 5: Revocation and cancellation during a call

**What was built.**
- `appointment`: `AccessEndArgs` (`consultation_access_end`, default queue, the appointment's and the clinic's
  ids) and `Jobs.EnqueueAccessEnd`. `cancelled`, which every cancellation goes through (the patient's, staff's and
  `CancelForRevokedConsent`), enqueues it in the cancel's transaction next to the e-mail.
- `videosession` (`service_access.go`): `EndAccess(ctx, appointmentID, org)` and
  `EndPatientAccess(ctx, patientID, org)` (SPEC §2 "Removal"; Decisions above). `Repository.ClosingAfter`
  (`SessionsClosingAfter`) is the patient's sessions with a clinic whose room may still be open. `ending` also sets
  the room's scope.
- `internal/worker`: the `consultation_access_end` worker (`accessEndWorker`, logging the appointment's id only).
  The `consent_revoked` worker calls `EndPatientAccess` after `CancelForRevokedConsent`. The worker's
  videosession service gets the `Appointments` port, for `Lookup`.
- `make bench-removal` (`test/e2e/consultation_removal_test.go`, tags `e2e && loadtest`). DPIA notes.

**Tests seen failing first.**
- appointment River jobs: `jobs.EnqueueAccessEnd undefined (type appointment.Jobs has no field or method
  EnqueueAccessEnd)` (build failed).
- appointment service: `TestCancel_ThePatientCancelsUpTo24HoursBeforeTheStart` failed with `enqueued [cancelled e1… a0…],
  want [cancelled e1… a0… access_end e1… a0…]`. `TestCancel_StaffCancelAtAnyTime` failed with `enqueued [cancelled …];
  want … the e-mail and the access's end`. `TestCancelForRevokedConsent_…` failed with `enqueued [cancelled ×3], want one
  cancellation e-mail and one end of the consultation's access each`.
- videosession service: `f.svc.EndAccess undefined`, `f.svc.EndPatientAccess undefined` (build failed). To check
  that `ending` needs the room's scope, I took it out: `TestEndPatientAccess_EndsTheSessionOfAnAppointmentTheRevocationCancelled`
  failed with `EndPatientAccess() = videosession: end patient access: videosession: end: videosession: not found`. Restored.
- videosession repository: `*postgresRepository does not implement Repository (missing method ClosingAfter)`
  (build failed). Against a stub that returned none: `ClosingAfter() = [], want [first, later]` and
  `ClosingAfter(the first's close) = [], want only the later one`. One test bug on the way: the other clinic's
  seed appointment overlapped its role's appointment (`appointment_no_double_booking`), so it was moved an hour.
- worker: `TestConsultationAccessEnd_EndsTheCancelledConsultation` failed with `Insert() error = job kind is not
  registered in the client's Workers bundle: consultation_access_end`. The scheduler's kinds test failed with
  `Insert(consultation_access_end) … want the kind known`. `TestConsentRevoked_RemovesThePatientFromTheConsultations`
  failed with `run 1: the job ended job_failed`. That event belonged to the `consultation_access_end` job the
  revocation queued ("Unhandled job kind"), not to the revocation: the shared `runWorkerJobWith` returned the
  first job event of any job. The helper now waits for the event of the job it inserted (a condition, no sleep),
  so a job that queues others is measured on its own end. Then, with the workers registered but the
  sessions service built without the `Appointments` port: a nil-pointer panic in both jobs, fixed in
  `newSessions`.
- `internal/api` through `api.New`, written against refusals the join already made. I saw them fail by taking
  code out and then restored it. Without `EnqueueAccessEnd` in the cancel: `queued consultation_access_end [],
  want one with {…}`. Without the join's `booked` check: `POST …/consultation/join = 200 …; want 409
  consultation_ended`. Without the consent check: the patient's join after the revocation `= 200 …; want 409
  consent_required`.
- videosession on the real LiveKit, written after the service. I saw it fail with the removal and the end taken
  out of `removePatient`, then restored them: `the patient is still connected`, `the practitioner is still
  connected`, `Participants() = [p_pat_… p_pra_…], want none`.

**What the tests cover.**
- Service (fakes): on a cancelled appointment, `EndAccess` removes the patient, ends the session `cancelled` in
  the clinic's scope, finishes the Encounter with whether the patient joined, audits the end by
  `consultation_access_end`, and deletes the room after the commit. Repeated, it changes and audits nothing and
  only deletes the room again. With no session, or a session out of the clinic's scope, it does nothing. On a
  booked appointment it only removes the patient. A failed read, removal or end ends nothing; a failed delete
  leaves the end committed.
- Service (fakes): `EndPatientAccess` reads in the patient's scope. On a consultation that started before the
  revocation it removes the patient after the commit and leaves the practitioner and the session. On the session
  of an appointment the revocation cancelled, which the clinic's scope hides, it ends the session `cancelled` in
  the room's scope, audited by `consent_revoked`, and deletes the room. Another clinic's session, and one past its
  hard cap, are left alone. Repeated, it changes and audits nothing. With no session it does nothing. A failure
  fails the job, and a failed removal in one room does not stop the removal from the others.
- Repository (real database, `salus_app`): `ClosingAfter` shows only the patient's own sessions with the clinic,
  closing after the given time; not another clinic's and not another patient's. A clinic's scope after a
  revocation cannot end the session of an appointment that had not started; the room's scope can.
- River (real database): `consultation_access_end` is queued on the default queue in the transaction, with only
  the two ids. A rollback leaves none, and outside a transaction it is `ErrNoTx`.
- Worker (real database, compose LiveKit through a recording provider): the cancelled consultation ends
  `cancelled`, its Encounter `finished`, its appointment stays `cancelled`, two audit events by the job, the
  patient removed and the room deleted, and a repeat changes nothing in the database. The revocation removes the
  patient from the started consultation, whose session stays open, and from the cancelled one, whose session
  ends `cancelled` and whose room is deleted, with one audit event of that end. A repeat changes nothing. Both
  logs hold no name, e-mail, room or identity; the access-end log carries the appointment's id. The scheduler
  knows the new kind.
- Through `api.New`: both sides joined, then right after the practitioner cancels, both get 409
  `consultation_ended` while `consultation_access_end` is still queued. Right after the patient revokes, the
  patient gets 409 `consent_required` and the practitioner 404 (the appointment had not started), while
  `consent_revoked` is still queued.
- Real LiveKit: two Go SDK participants with the provider's tokens. After `EndAccess` on the cancelled
  appointment both are disconnected and the room is empty. After `EndPatientAccess` the patient is disconnected
  `removed by server` and only the practitioner is in the room.

**Removal latency in the stack (`make bench-removal`; api, worker and LiveKit in compose, i7-14700HX).**
Each run uses a fresh synthetic clinic, practitioner and patient. The patient books through the API; the
appointment is moved, as the owner, to have started a minute ago or to start in 5 minutes. Both sides take their
tokens from the join endpoint and connect as Go SDK participants on the compose network. The patient revokes
through the API, and the test timestamps the patient's disconnection. The commit lies between the consent's
`revoked_at`, set in the revocation's transaction, and the revoke's response, so the two columns bracket the
time from the commit to the removal. Three runs each.

| Case | From the revoke's response | From `revoked_at` |
| --- | --- | --- |
| Started before the revocation (patient removed, practitioner stays) | 11, 10, 5 ms | 14, 13, 8 ms |
| Not started (appointment cancelled, then patient removed and room deleted) | 21, 19, 27 ms | 24, 22, 30 ms |

River's notify brings the job to the worker well under a second, as SPEC §2 expected. The not-started case is
about 10 ms slower: the job first cancels the appointment (slot, e-mail, audit) and only then removes the patient.
No change is made for speed here. After the runs, the worker's and the api's logs from that period held no name,
e-mail, token, room or identity.

**Checks.** `make lint`: 0 issues (both modules, `test/` with its tags), migrations unchanged. `make archcheck`:
passes. `make test`: passes. `make cover`: every package at 80% or more (`internal/videosession` 95.7%,
`internal/appointment` 92.9%, `internal/worker` 94.6%, `internal/api` 91.1%, total 94.9%). `make sqlc`:
regenerated (one query). No `openapi.yaml` or FHIR change, so `make gen-clients` and `make fhir-validate` are
not affected. Stack: `make up` (through `make bench-removal`) rebuilt the api and the worker, then the measurement
above ran.

**Deferred.** The `Encounter` of an appointment the revocation cancels after the practitioner joined stays
`in-progress` ("Waiting on people"). Also, `TestScheduler_KnowsEveryKindTheWorkerAndTheNotifierWork` does not
list package 2's `consultation_window_close`. The scheduler does register it, so this is only a gap in the test,
left for a later package.

**Dev stack left.** No migration. The api and the worker were rebuilt by `make up` and run with this package's
code. The six measurement runs left their synthetic e2e data in the dev database, as `make e2e` and package 4's
runs do: six clinics, each with an admin and a practitioner, and six patients, each with a revoked consent and
one appointment with its session, slots, jobs and audit events (285 new `audit_event` rows, which are
append-only). The three started consultations stay `booked` with their sessions open and their `Encounter`s
`in-progress`; their rooms are empty in the compose LiveKit. Each window-close job ends its session
`window_closed` about an hour after its run, finishing the `Encounter` and marking the appointment
`fulfilled`. The three not-started ones are `cancelled` with their sessions ended `cancelled` and their rooms
deleted, and their `Encounter`s stay `in-progress` (the open point above).

### 2026-11-05 — package 5.1: Owner changes after package 5

**What was built.**
- `migrations/00023_encounter_room_scope.sql`: the `encounter_room` (SELECT) and `encounter_room_update` (UPDATE)
  policies. With `app.video_room` set and no patient or account set, a transaction sees and updates the encounter
  of the appointment whose session has that room (through `video_session`'s own policies). There is no INSERT
  policy, no grant change and no new table. The end of a session already runs in its room's scope (`ending`), so
  the `Encounter` of an appointment a revocation cancelled after the practitioner joined now finishes, with
  `period.end` set. The appointment stays `cancelled` (`RecordConsultationOutcome` changes only a booked one).
  No Go change was needed for that.
- `videosession`: `removeFromRoom` is the one removal of a patient, used by the webhook's re-check, `EndAccess`
  and `EndPatientAccess`. It lists the room, audits the removal (operation, action `E`, `Appointment/<id>`, the
  removing agent, `HOPERAT`) when the patient is in it, then asks LiveKit to remove them. The fake provider now
  takes a removed identity out of the room.
- `TestScheduler_KnowsEveryKindTheWorkerAndTheNotifierWork` lists `consultation_window_close` (it already listed
  `consultation_access_end`).
- DPIA notes (package 5.1; package 5's open risk closed).

**Tests seen failing first.**
- migrations (`salus_app`): `TestEncounter_TheRoomScopeSeesAndFinishesOnlyItsRoomsEncounter` failed with `the room
  of a cancelled appointment: sees [], want [3cc2…]`, `another room: sees [], want [c514…]`, `the room with clinic B
  set: sees [6666…], want [3cc2… 6666…]` and `finished [6666…], want the room's and clinic B's own [3cc2… 6666…]`.
  It passed after the migration.
- worker (the case of package 5, real database and compose LiveKit):
  `TestConsentRevoked_FinishesTheEncounterOfAConsultationTheRevocationCancelled` failed with `session cancelled,
  encounter in-progress (period.end set false), appointment cancelled; want cancelled, finished with an end,
  cancelled` and `0 audit events of the encounter's finish by the job, want 1`. It passed after the migration,
  with no Go change.
- scheduler: the test already passed with the kinds registered, so I took both registrations out of
  `addDefaultWorkers`. It then failed with `Insert(consultation_access_end) error = job kind is not registered…` and
  `Insert(consultation_window_close) error = job kind is not registered…`. Restored.
- videosession service (fakes): `TestRemoval_EachPathAuditsTheRemovalOnceAndARepeatNothing` failed on all five
  cases, for example `the webhook's re-check audited []` and `EndAccess, cancelled audited [the end only]`.
  `TestRemoval_IsAuditedBeforeThePatientIsRemoved` failed with `with the audit failing: EndPatientAccess() =
  <nil>, want the error` and `removed [r_room p_patient], events []; want nothing`, and the same for a failed list.
- Written after the code, so I saw them fail by disabling the audit in `removeFromRoom`, then restored it. On the
  real LiveKit, `TestEndPatientAccess_OnLiveKitDisconnectsOnlyThePatient` failed with `EndPatientAccess audited
  []`. On the real database, `TestConsentRevoked_AuditsTheRemovalOfThePatientOnce` failed with `0 audit events of
  the removal, want 1`.

**What the tests cover.**
- RLS (`salus_app`): a room shows exactly its session's encounter, also one the clinic's scope hides after the
  revocation. An unknown room shows nothing, a room with a patient set shows only that patient's, and a room with
  an account set shows nothing. Clinic B without the room still sees only its past appointment's. The room's scope
  with its clinic finishes only those two. A room's scope cannot insert an encounter, with or without a session
  in it.
- Removal audit (fakes), for the webhook's re-check, `EndAccess` on a booked and on a cancelled appointment, and
  `EndPatientAccess` on a started and on a cancelled one: one removal event by the path's agent (before the end's
  event where there is one). The patient is out of the room and the practitioner still in it, and LiveKit is
  asked outside any transaction. A repeat audits nothing and asks LiveKit to remove them again. A failed audit or
  list removes nobody and fails for a retry. The package 5 tests stay as they were: a patient who is not in the
  room is not audited as removed.
- Worker (real database): the package 5 case ends with the session `cancelled`, the Encounter `finished` with
  `period_end` set, and the appointment `cancelled`, with one Encounter update audited by `consent_revoked`. A
  patient in the room of a started consultation is removed with one audit event (`operation`, `E`, `HOPERAT`,
  the clinic, `Appointment/<id>`), and a retry adds none. The logs hold no name, e-mail, room or identity.
- Real LiveKit: the revocation's removal of a connected patient writes the one removal event.

**Removal latency in the stack (`make bench-removal`, as package 5, three runs per case).** The removal now
lists the room first. Started before the revocation: 13, 11, 11 ms from the response (15, 13, 13 ms from
`revoked_at`). Not started: 18, 17, 15 ms (20, 19, 18 ms). These are in package 5's range (5–11 / 19–27 ms), so
no change is made for speed. One earlier run of the bench in this package, made while the stack was rebuilding, was
not recorded. In the dev database the not-started consultations of these runs left no `Encounter` in progress
(5 `finished` with `period_end`; the sixth never recorded the practitioner's join, so it has no Encounter). The
worker's and the api's logs from the period held no e-mail, room or identity.

**Checks.** `make lint`: 0 issues in both modules, migrations from `main` unchanged. `make archcheck`: passes.
`make test`: passes. `make cover`: every package at 80% or more (`internal/videosession` 95.7%, `internal/worker`
94.6%, `internal/encounter` 97.2%, total 94.9%). No query, `openapi.yaml` or FHIR change, so `make sqlc`,
`make gen-clients` and `make fhir-validate` are not affected. Stack: `make up` rebuilt the api, worker and the
others and applied migration 23 (goose version 23), then `make bench-removal` ran.

**Dev database.** After the commit it is wiped as the owner asked (`docker compose … down -v`, then `make up`).
That removes the synthetic data of packages 4, 5 and this package's runs, including package 5's three
in-progress Encounters.

### 2026-11-05 — package 6: Web

**What was built.**
- `web/package.json`: `livekit-client` 2.23.0, exact (the current release, published 2026-10-16), with its
  dependencies in the lockfile.
- `web/src/call.ts` is the one module that uses livekit-client and the browser's devices. It holds the `Media` and
  `Room` contract the screens use, and `liveKit`: device lists, previews with a microphone level (Web Audio, RMS),
  and the room. The room publishes camera and microphone only and sets no name, metadata or attributes. It maps
  LiveKit's events to peer, reconnecting, reconnected and disconnected, with an `Ending`: left, ended, replaced or
  lost. `src/test/fakeCall.ts` is its fake.
- `web/src/DeviceCheck.tsx`: the camera preview, the microphone meter and the W6 silence line, pickers (W5), the
  denied and missing states (W10–W12), "join with camera/microphone off" (W7), and Join and Joining… (W8, W9).
- `web/src/CallScreen.tsx`: joining, waiting (W13, W14), the call (the remote video large, the own camera small;
  W15–W17), the patient leaving or removed (W18, W18a), reconnecting (W24–W26), leave and join again (W23), "End
  consultation" with its confirmation (W19–W22, W32b), the end (W27–W29, W18b), and the join errors (W30–W34,
  W39).
- `web/src/consultation.ts`: the join window (10 minutes before the start to 30 after the end, `booked` only),
  the error code to text mapping, and `useNow`.
- `Calendar.tsx`: W1 under the card of a joinable appointment. `AppointmentDetail.tsx`: W1 or W2, and the
  "Consultation" section (W35–W38). `App.tsx`: the `/appointments/<id>/call` route. `messages.ts`: every string
  from `COPY.md`, word for word, with its W number. `icons.tsx`: microphone, camera and leave. `app.css`: the
  device check and the call, as in SPEC §2 "Web client". `vite.config.ts`: the dev server proxies the three
  consultation endpoints.
- `docs/privacy/dpia.md`: package 6.

**Tests seen failing first** (Vitest, `docker compose … run node npx vitest run <file>`).
- `consultation.test.ts`: `Failed to resolve import "./consultation"` (no test ran).
- `Calendar.test.tsx`: 4 of the 5 new tests failed, because no "Join consultation" button was rendered (10 minutes
  before, until 30 minutes after, opens the call path, appears when the window opens). The other two (hidden
  just before and from 30 minutes after) passed, as they assert absence.
- `AppointmentDetail.test.tsx`: 7 failed (`Unable to find role="button" name "Join consultation"`, the W2 hint
  and the "Consultation" region); the absence cases passed.
- `CallScreen.test.tsx`: `Failed to resolve import "./CallScreen"`. With the screen written, 3 failed on a test
  bug: `Found multiple elements with the text of: Camera` (the preview and the picker shared the label). The
  preview lost its label and the tests query the pickers by role.
- `call.test.ts`: all 17 failed, `endingOf`, `problemOf` and `levelOf` not exported.
- `App.test.tsx`: the call path showed "This page does not exist." (the heading was not found).
- Found in the browser walk, each test first: `does not say it waits for the patient while it reconnects` failed
  with `expected <h2></h2> to be null`, and `offers Join only once the camera and the microphone have opened or
  failed` failed with `expected false to be true`.
- Not seen failing: `does not take a level it cannot measure for silence` passed before its code, because
  `undefined < 0.02` is false in JavaScript. The type (`number | undefined`) and the hook now handle it
  explicitly.

**What the tests cover.** The window's bounds to the second, and that it is closed for every status but
`booked`. The button on the card and on the detail at those bounds, hidden for the admin and on an appointment
that is not booked, and appearing when the clock passes the opening. The status section for `not_started`,
`open` and `ended`, the times in the clinic's zone, the admin's view, and a failed read. The device check:
preview, pickers, a picked camera opened and the previous one stopped, silence after 5 seconds, an unmeasurable
level, a denied camera joining with audio only (camera turn-on disabled in the call), a denied microphone joining
with the camera, missing devices, joining with the camera off, and Join waiting for the devices. Joining: the
token request with the CSRF header, "Joining…", the URL and token passed to the room, and each refusal's text
(`join_window_not_open`, `consultation_ended`, `practitioner_did_not_join`, `consent_required`, 404, 500) with no
connection. A connection failure shows W34. In the call: waiting, then the patient; camera off and muted; mute and
camera toggles; the patient leaving (W18) or removed after withdrawing consent (W18a); reconnecting and
reconnected; a lost connection rejoined with a new token and the same settings; a failed rejoin with "Try again";
events of a left room ignored; leave and join again; a second tab. The end: on a deleted room, the text for
`practitioner` with and without the patient, `time_limit` and `cancelled`; the confirmation and its "Cancel"; the
end with the CSRF header and W28; a patient who joined before this connection; `consultation_not_started`. The
adapter: LiveKit's disconnect reasons, getUserMedia's errors, and the RMS level.

**Browser walk (`make up`, the Vite dev server, two host Chrome 157 profiles, headless, with
`--use-fake-device-for-media-stream` and `--use-fake-ui-for-media-stream`).** Each profile was driven over CDP by
Playwright (Python 1.55.0, in a throwaway container on the host network). The practitioner's profile used the app
with a synthetic session cookie. The patient's profile, on the same origin, took its token from the join endpoint
with the patient's bearer and connected with livekit-client's UMD build and fake devices. The patients are
mobile-only in the product.
- Desktop (1280 px): the calendar card's button, the detail (button, "Not started"), the device check (preview
  playing, pickers, meter), waiting, the patient's video arriving (`videoWidth` > 0) and the own preview. The
  patient muted with the camera off showed W16 and W17, both gone when they came back. The practitioner's toggles
  were seen by the patient. The patient leaving showed W18, and they rejoined.
- LiveKit restarted under the call (`docker restart`): W24, then livekit-client's own resume failed ("requested
  room does not exist"), and the app asked for a new token, rejoined (the join recreated the room) and showed W26.
  The patient rejoined and the video played again.
- The end: the confirmation with W21, then W27 and W28. The patient was disconnected with LiveKit's reason 5 (room
  deleted). The detail then showed "Fulfilled", "Ended", "Patient joined at …" and "Practitioner joined at …".
- Revocation after the start (second clinic): the patient revoked through the API, LiveKit removed them (reason 4),
  W18a showed, and the practitioner ended the consultation.
- Revocation before the start: the appointment was cancelled and the room deleted, and the practitioner saw W27
  alone (Decisions above).
- 390 px: calendar, detail, device check, call, confirmation, "You left…" and the end, with no horizontal scroll.
- Fixed from the walk: the device check's panel class collided with the existing `.check` rule; Join before the
  devices opened joined with both off; "Waiting for the patient" showed while reconnecting; the confirmation sat
  beside the controls; the muted badge had no border.
- The host-browser path to LiveKit works with the dev config (`deployments/livekit.yaml`, package 1's ICE
  settings): signalling on `ws://localhost:7880` and media through the published 127.0.0.1 ports between two host
  browser profiles. No change to the LiveKit config.
- The api's, worker's and LiveKit's logs from the walk held no name, e-mail or token. The api and the worker held
  no room or identity either; LiveKit logs its own opaque ones.

**Checks.** `make web-lint`: 0 warnings, 0 errors. `make web-typecheck`: passes. `make web-test`: 16 files, 202
tests pass. `make web-build`: passes, with Vite's chunk-size warning (835 kB minified, 233 kB gzip, most of it
livekit-client; see Deferred). `make gen-clients`: no diff. Every consultation string in `messages.ts` checked
against `COPY.md` by a script: 57 strings, none missing. `make lint`: 0 issues in both modules, migrations from
`main` unchanged. `make archcheck`: passes. `make test`: passes. `make cover`: every package at 80% or more, total
94.9%. No Go, `openapi.yaml` or FHIR change, so `make sqlc` and `make fhir-validate` are not affected.

**Deferred.**
- Splitting livekit-client into its own chunk (a lazy call screen) to shrink the first load. It is not measured
  here as a problem, so it is not done.
- livekit-client prints its own connection errors to the browser console, the signalling URL with the token
  among them. That is the user's own browser, not a log of ours, but its log level could be lowered.
- The patient's side is package 7. The Playwright e2e journey is package 8.

**Dev stack left.** No migration. The dev database holds the walk's synthetic data, made through the API by a
temporary e2e helper (deleted before the commit): 4 clinics, each with an admin and a practitioner; 4 patients,
each with a consent (2 revoked) and one appointment with its slots, jobs and audit events (303 `audit_event` rows
in all, append-only); and e-mails in Mailpit. The sessions: 3 ended `practitioner` (appointments `fulfilled`,
`Encounter`s `finished`), 1 ended `cancelled` (appointment `cancelled`, `Encounter` `finished` with `period.end`).
No room is left in LiveKit, which was restarted once during the walk and is healthy. The Vite dev container was
stopped and removed. The Playwright image `mcr.microsoft.com/playwright/python:v1.55.0-noble` stays in the host's
Docker image cache. Port 5173 is held by an unrelated container on this machine, so the walk used
`SALUS_WEB_PORT=5175`; nothing of ours was changed for it.

