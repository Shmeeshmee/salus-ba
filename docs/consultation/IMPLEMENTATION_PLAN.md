# Video consultations implementation plan

Read `SPEC.md` first. Packages run in order; each is one commit `consultation(<n>): <summary>` on
`feat/consultation`, made with `/run-package docs/consultation <n>`.

## Rules for every package

- Test first: write the failing test, run it and record the failure in `PROGRESS.md`, then the least code to
  pass, then refactor. Service tests with hand-written fakes and `clock.Fixed`; repository tests on
  `pgtest.NewDatabase` (and `NewAppDatabase` for every RLS check); the LiveKit repository against the compose
  LiveKit; handlers with `httptest`, FHIR handlers against golden files.
- Every new domain is added to `docs/architecture/archcheck/config.yaml` (context `consultation`), every new
  table to `ownership.yaml` with `personal_data` and a retention rule, every domain with queries to
  `sqlc.yaml`, in the commit that adds it. The archcheck allowlist never grows.
- Every migration is a new file, with its grants, `ENABLE ROW LEVEL SECURITY` and policies. Policies read
  settings as `NULLIF(current_setting('…', true), '')`.
- Every read or write of `Encounter`, every token issued or refused, and every lifecycle change audits in the
  same transaction, refusals included; a test per service method asserts the event.
- LiveKit types appear only in `internal/videosession/repository_livekit.go` (and `test/e2e`); `depguard`
  forbids the SDK everywhere else.
- No personal data, token, room name or participant identity in a log line, error, panic, golden file,
  fixture or LiveKit request. A test asserts the log output of each new handler and job holds none of them,
  and a test of the provider asserts the room and token carry no name or metadata.
- A change to `api/openapi.yaml` is followed by `make gen-clients` in the same commit.
- Timing tests wait for a condition (as in `4c62e68`), never on a longer sleep; a flaky one is fixed, not
  slowed or skipped.
- Pin every version. Update `PROGRESS.md` in the same commit.

## Check sets

Every check runs through `make`, in Docker. Never `go`, `npm` or `flutter` on the host.

- **Go**: `make lint`, `make test`, `make archcheck`, `make cover` (every package at 80% or more).
- **FHIR**: `make fhir-validate`.
- **Clients**: `make gen-clients` leaves no diff.
- **Web**: `make web-lint`, `make web-typecheck`, `make web-test`, `make web-build`.
- **Mobile**: `make mobile-analyze`, `make mobile-test`.
- **Stack**: `make up`, then the package's manual check, then `make down`.

## Packages

### 1. videosession

Migration: `video_session` (SPEC §2 "Tables", §5), RLS (patient scope; clinic scope as `appointment`'s; the
`app.video_room` webhook scope), grants to `salus_app`, `SELECT, DELETE` to `salus_retention`. Ownership entry,
retention rule `delete` (SPEC §5) and its purger. `config.yaml`: context `consultation: [videosession]`.
`go.mod`: `github.com/livekit/server-sdk-go/v2` and `github.com/livekit/protocol`, exact versions (the
current releases, recorded in `PROGRESS.md`); `depguard` rule. `internal/config`: the four `LIVEKIT_*`
settings (SPEC §2 "Configuration"), loaded by api and worker only. Compose: `deployments/livekit.yaml` (keys,
`auto_create: false`, JSON logs; the webhook URL comes in package 3), `TEST_LIVEKIT_*` on `tools`; CI starts
LiveKit for the Go tests.
`internal/videosession`: `model.go` (`Session`, `Grant`, `Room`, `Join` result), `errors.go`, `repository.go`
(the `Repository` and `Provider` ports: `CreateRoom`, `Token`, `RemoveParticipant`, `DeleteRoom`,
`Participants`; `ParseWebhook` arrives in package 3), `repository_postgres.go`, `repository_livekit.go`,
`service.go`: `Join` (SPEC §2 "Who may join", "Join response", "Rooms", "Ids", "Token grant"), the ports it
consumes from `appointment` (the appointment and its facts), `patient` and `practitioner` (display names), and
`auditevent`. The window-close job is not enqueued yet: it comes with its worker in package 2, when there is
an `Encounter` to finish, so no job kind exists without a worker.
`BenchmarkJoin` (service on the real database with the real LiveKit, and with a fake provider, to separate
LiveKit's share), numbers in `PROGRESS.md`; `make bench-join`.
Tests: the window boundaries (10 min before start ± 1 s, 30 min after end ± 1 s); patient, practitioner of the
role, another practitioner, admin, another clinic, anonymous; consent active, revoked, never; appointment
`booked`, `cancelled`, `fulfilled`, `noshow`; ended session; each outcome's code and audit event. Two
concurrent first requests create one session and one room (real database). The token decoded in the test
grants exactly one room, camera and microphone, no data, no name, no metadata, 5 minutes; the room has
`max_participants` 2 and no metadata (read back from LiveKit). Ids are random and never equal an id of ours.
Practitioner's `peerName` writes a `Patient` audit event; the patient's does not. As `salus_app`: a patient
scope sees only their own session, a clinic scope after a revocation sees only sessions of appointments
started before it, `app.video_room` shows exactly one row. The log holds no token, room, identity or name.
Checks: Go; Stack (a repository test joins the compose LiveKit with a token from the service through the Go
SDK, and a second connection with the same identity replaces the first).

### 2. encounter

Migration: `encounter` (SPEC §2 "Tables", "Encounter"), RLS like `video_session` (no webhook scope: the
webhook sets the clinic after reading its session), grants. Ownership, retention rule `keep`. `config.yaml`:
`encounter` in `consultation`.
`internal/encounter`: model, errors, `Service.Start(ctx, agent, appointmentID, at)` (idempotent),
`Service.Finish(ctx, agent, appointmentID, at, patientJoined)` (idempotent; a never-started encounter is not
created), `Read`, `Search` (by appointment, patient, practitioner; SPEC §2 "Encounter"), audit. Port
`Appointments` (`appointment.Service`: the appointment with its access facts, and the outcome).
`internal/appointment`: `RecordConsultationOutcome` (SPEC §2 "Outcome") with its audit event.
`internal/videosession`: `end` (SPEC §2 "Ending"), the `encounter` port, and the window-close job enqueued
in the session's insert transaction (`repository_river.go`); `internal/worker`: the
`consultation_window_close` worker (SPEC §2 "The window closes"), snoozing while the room is occupied.
Tests: `Start` twice creates one encounter with the first time; `Finish` twice changes nothing the second
time; `Finish` with and without the patient gives `fulfilled` and `noshow`; an appointment already marked by
hand or cancelled keeps its status; an encounter after `finished` never changes; the window-close job with an
empty room ends the session, deletes the room and finishes the encounter; with a participant it snoozes; when
the practitioner never joined the session ends `practitioner_absent`, the appointment stays `booked`, no
encounter exists and `patient_joined_at` is kept; at 2 hours after the scheduled end (± 1 s) the job ends an
occupied room (`time_limit`), deletes it, finishes the encounter and records the outcome, and no snooze is
scheduled past the cap; the room is deleted
after the commit and a delete failure leaves the database state committed and the job retried; each access
rule case of `Read` and `Search`; audit per call. Real database: `Start` from two goroutines creates one row.
Checks: Go.

### 3. Webhooks

`Provider.ParseWebhook(body, authorization)` in `repository_livekit.go`, returning the domain `Event` (kind,
room, identity, joined time). `videosession.Service.HandleEvent` (SPEC §2 "Idempotent, order-free handling",
"Webhook scope"); `handler_http.go`: `POST /webhooks/livekit` (body limit, 401 on a bad signature, 200 on any
handled event, 500 through `httpx.InternalError`). `openapi.yaml`, `make gen-clients`. Compose:
`webhook.urls` in `deployments/livekit.yaml` pointing at the api, with the dev key.
Tests: a valid signed body is accepted; a wrong key, a wrong secret, a changed body, a missing header and an
expired token are 401 and handle nothing; each event kind's effect; every event delivered twice, and every
permutation of practitioner joined, patient joined, practitioner left, patient left and room finished,
reaches the same final state (table of orders); an event for an unknown room is 200 with no effect; a patient
join after their consent was revoked removes them; the log holds the request id and the event kind only.
Checks: Go, Clients; Stack (a participant joins with a token from the service and the api's log shows the
webhook handled; the `Encounter` row appears when the practitioner identity joins).

### 4. API

`videosession/handler_http.go`: `POST /appointments/{id}/consultation/join` and
`/appointments/{id}/consultation/end` (SPEC §2 "Join response", "Ending"), and `GET
/appointments/{id}/consultation` (SPEC §2 "Consultation status"); a join on a session ended
`practitioner_absent` is 409 `practitioner_did_not_join`; `encounter/handler_fhir.go`:
`Encounter` read and search by `appointment`, `patient`, `practitioner`. `openapi.yaml`, `make gen-clients`;
`api/fhir/CapabilityStatement.json` gains `Encounter` with exactly `read`, `search-type` and the three
parameters. Golden files for an in-progress and a finished `Encounter`, a search `Bundle`, and the
`OperationOutcome`s (404, 400). Latency: a Go load test against the running stack (`make bench-join-http`:
1,000 join requests, 10 concurrent, real Postgres and LiveKit) recording p50, p95 and p99 in `PROGRESS.md`,
next to package 1's benchmark.
Tests: the error codes per case through HTTP (404, 409 `consent_required`, `join_window_not_open`,
`consultation_ended`, `practitioner_did_not_join`); the status endpoint for the patient, the practitioner
and the admin, with and without access, and its audit event; end by the patient or an admin is 404; end twice is 200; the access matrix for
`Encounter` read and search (patient, practitioner with and without access, admin, other clinic, after a
revocation); an unknown search parameter is 400; audit per read and per returned resource; `make
fhir-validate` passes with the new golden files; the handler log holds no token or name.
Checks: Go, FHIR, Clients.

### 5. Revocation and cancellation during a call

`appointment`: the cancel transaction enqueues `consultation_access_end` (SPEC §2 "Removal") through its
`Jobs` port. `videosession.Service.EndAccess` and `EndPatientAccess`; `internal/worker`: the
`consultation_access_end` worker and the `consent_revoked` worker's call to `EndPatientAccess`.
Tests: cancelling enqueues the job in the cancel's transaction (a rollback leaves none) with only the two ids;
`EndAccess` on a cancelled appointment removes the patient, ends the session and deletes the room; with no
session it does nothing; the revocation job removes the patient from the open room of an appointment that
started before the revocation and leaves the practitioner; a token request right after either commit is
refused (before the job runs); repeating either job changes nothing. Real LiveKit: a joined participant is
gone from `Participants` after `EndAccess`. Measured: the time from the revoke's commit to the participant's
removal in the stack, recorded in `PROGRESS.md`.
Checks: Go; Stack.

### 5.1. Owner changes after package 5

Migration (new file): `encounter`'s `app.video_room` policy, which shows and lets `salus_app` update the
encounter of the appointment whose session has that room (SPEC §2 "Tables", "Changed after package 5").
`videosession`: the end of a session cancelled by a revocation finishes its `Encounter` in the room's scope,
with `period.end` set. Every removal of a patient from a room (the webhook's re-check, `EndAccess`,
`EndPatientAccess`) writes an `AuditEvent` (operation, action `E`, `Appointment/<id>`, the removing agent,
`HOPERAT`), in the transaction that decides it. `TestScheduler_KnowsEveryKindTheWorkerAndTheNotifierWork`
lists `consultation_window_close` and `consultation_access_end`.
Tests: as `salus_app`, `app.video_room` shows exactly the one encounter of that room and nothing else, and
without it, after the revocation, the clinic scope still sees nothing; the case of package 5 (practitioner joins
before the start, the patient revokes before the start) ends with the `Encounter` `finished` and `period.end`
set; each removal path writes one audit event and a repeat writes none; the scheduler test fails without the
two kinds. After the commit, the dev database is wiped (`make down` with volumes, then `make up`), which also
removes the synthetic data of packages 4 and 5.
Checks: Go; Stack.

### 6. Web

**Held until the owner has answered the copy review** (SPEC §2 "Copy review"). Before this package starts,
every new web and mobile string (permission prompts, device check, waiting, reconnecting, ended, "the
practitioner did not join", the time limit, removed, whether the patient and practitioner joined,
errors, the end-consultation confirmation, the iOS and Android permission descriptions) goes to the owner in
one list, on its own; the answer is recorded in `PROGRESS.md` and the strings go into `web/src/messages.ts` and
`mobile/lib/messages.dart` as approved. **Answered 2026-11-05: `COPY.md` holds the approved text, word for
word; W18a and W18b replace W18 when the patient was removed, and W39 is the web's `practitioner_did_not_join`.**
`web/package.json`: `livekit-client`, exact version. `web/src`: the "Join" button on the appointment detail and
the calendar card (SPEC §2 "Join button"); on the appointment detail, whether the patient and the
practitioner joined (SPEC §2 "Consultation status"); the device check (camera preview, microphone level, device pickers,
denied-permission state); the call screen (remote video large, own preview small, microphone, camera, leave,
"End consultation" with a confirmation); waiting, reconnecting, ended and removed states. Visual direction as
in SPEC §2 "Web client": light, flat, mint accents, minimal inline SVG icons (`icons.tsx`), no LiveKit UI
components. LiveKit is reached through one small module (`web/src/call.ts`) so the screens are tested with a
fake room.
Tests (Vitest): each state of the call screen from fake room events; the button's window (shown 10 min before,
hidden 30 min after the end, hidden when not `booked`); a denied camera still joins with audio; reconnecting
and reconnected; "the consultation has ended" on a deleted room; end asks to confirm and calls the endpoint
with the CSRF header; a 409 `join_window_not_open` and `consultation_ended` show their messages.
Checks: Web, Clients; the screens walked in a browser against the stack, at desktop width, joined from a
second browser profile with fake devices.

### 7. Mobile

**Held with package 6** (the same copy review). `mobile/pubspec.yaml`: `livekit_client`, exact version, and
whatever permission handling it needs (its own, or `permission_handler` pinned, recorded). iOS `Info.plist`
`NSCameraUsageDescription` and `NSMicrophoneUsageDescription`; Android `CAMERA`, `RECORD_AUDIO`,
`MODIFY_AUDIO_SETTINGS`, `BLUETOOTH_CONNECT` as the package requires; the wording as approved. `mobile/lib`: the
"Join" button in my appointments, the device check, the permission-denied state with a way to the settings, the
waiting screen ("waiting for the practitioner"), the call screen, reconnecting, ended, "the practitioner did
not join" and removed states, all
behind one small module (`call.dart`) so widgets are tested with a fake room.
Tests (widget, `MockClient` and a fake room): each state; the button's window; a denied permission; reconnect;
ended; "the practitioner did not join" on a 409 `practitioner_did_not_join` and when the room is deleted
while waiting; the waiting screen until the practitioner's participant appears; a 409 shows its message.
Checks: Mobile, Clients. Running on a device is the manual step in `README.md`, which gains the LiveKit URL for
an emulator.

### 8. End-to-end

`deployments/docker-compose.yml`: the `playwright` service (pinned image, profile `e2e`) and the e2e network
settings LiveKit needs so in-network clients reach its media ports (package 1 records the ICE settings; this
package proves them; a separate e2e LiveKit config is acceptable if one cannot serve both, SPEC §2). `test/e2e/consultation_test.go` (build tag `e2e`) and a Playwright spec under
`test/e2e/web/`: on an appointment booked through the API with its start a few minutes ahead (synthetic data,
fresh per run), the practitioner signs in to the web app in Chromium with fake media devices and joins; the
patient joins as a Go SDK participant publishing a synthetic audio track; the test waits (as a condition) for
the `Encounter` to be `in-progress`, the practitioner ends the consultation, the `Encounter` is `finished` and
the appointment `fulfilled`. Second journey: both join, the patient revokes consent through the API, the patient
participant is disconnected (waited for as a condition), and their next join request is 409 `consent_required`.
After both: LiveKit's log holds none of the run's synthetic names, e-mails or appointment ids.
Checks: all of the above, `make e2e` twice in a row.

## Definition of done

Every package committed, `PROGRESS.md` complete, all check sets pass locally and in CI on the PR into `main`.
