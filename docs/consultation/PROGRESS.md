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
| 7 Mobile | done, 2026-11-05 | `consultation(7)` |
| 7.1 Owner changes after package 6 | done, 2026-11-05 | `consultation(7.1)` |
| 8 End-to-end | done, 2026-11-05 | `consultation(8)` |

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

- 2026-11-05, package 7 (runner): the patient learns why their connection ended from LiveKit's reason and our
  API, as the owner's brief says. Removed from the room (`participantRemoved`): the join endpoint is asked once;
  `consent_required` shows M27, `consultation_ended` M28 (a removal happens only on a revocation or a
  cancellation, so with the consent active it was the cancellation), `practitioner_did_not_join` M25 and M26, and
  a 200 rejoins. The room deleted or closed: the status says how it ended (`practitioner_absent` M25 and M26,
  `time_limit` M22 and M24, `cancelled` M28, otherwise M22 and M23); a status still `open` rejoins, as on the web.
  A lost connection asks for a new token and rejoins; a rejoin refused as `consultation_ended` shows the end from
  the status without rejoining again, so it cannot loop. Each join request is audited as any join.
- 2026-11-05, package 7 (runner): COPY.md has no text for some cases, so none was written. On Android the
  explanation M35 is shown on the device check and the system prompt follows once that frame is on screen; there
  is no "continue" button, as COPY.md has no label for one (see the questions). A patient who joins from a second
  device replaces the first connection, and the first goes back to the device check (W23 is web only). Leave and
  the back button close the call screen and return to my appointments, which are read again. The ended and the
  removed screens have the approved heading where COPY.md gives one (M22, M25) and none for M27 and M28
  (`ui.Page` takes no title). The microphone level has no label (W5 is web only). M2 shows under a booked
  appointment until its window opens; the list reads the clock every 15 seconds, as the web does.
- 2026-11-05, package 7 (runner): `permission_handler` 13.0.2 is needed: livekit_client cannot say whether a
  permission is still to be asked (M35 before the prompt) and cannot open the app's settings (M11), and
  `url_launcher` cannot open them on Android. On iOS the plugin (Swift Package Manager, as the project uses)
  enables the camera and microphone from the Info.plist keys, so no Podfile macro is needed. Android declares
  `CAMERA`, `RECORD_AUDIO`, `MODIFY_AUDIO_SETTINGS`, the network state permissions and the pre-Android 12
  Bluetooth ones (install time, no prompt), and the camera as not required. `BLUETOOTH_CONNECT` is not declared:
  on Android 12 and later it is a run-time prompt with no approved explanation; without it a Bluetooth headset
  may not carry the call's audio (see the questions).
- 2026-11-05, package 7 (runner): the device check stops its previews on Join (on Android a camera already open
  for the preview would keep LiveKit from opening it) and opens them again when the join is refused. Silence (M5)
  shows when no reading has reached 0.02 for 5 seconds; until a first reading arrives the level is not measured,
  as on the web. The level is the loudest band of LiveKit's audio visualizer; the threshold is the web's and has
  not been checked on a device.

- 2026-11-05, owner (after package 6, recorded in `SPEC.md` §2 "Changed after package 6"): the status
  `reason` and the practitioner's room-based read of it, replacing the join probe; W40 for a second tab; the
  LiveKit client log levels on web and mobile, fixed before the PR merges; the bundle size left, in SPEC §7.
  Package 7.1.

- 2026-11-05, package 7.1 (runner): the removal's reason is not derivable from what is stored: a revocation
  before the start cancels the appointment and ends the session `cancelled`, the same end as a plain
  cancellation, and the consent's state today says nothing about the order of a cancellation and a later
  revocation. So `video_session.removal_reason` (`consent_withdrawn` | `cancelled`, migration 00024) records it,
  once (the first reason stays), in the room's scope, in the removal's audited transaction, and only when LiveKit
  lists the patient in the room (package 5.1's rule for the audit). The webhook's re-check now asks the consent
  first, so a patient whose consent was withdrawn is `consent_withdrawn` even when the revocation also cancelled
  the appointment; an appointment marked fulfilled or no-show by hand removes the patient with no reason.
- 2026-11-05, package 7.1 (runner): `reason` on the status is the removal's reason first, then the end's
  (`practitioner` and `window_closed` → `ended`, `practitioner_absent` → `practitioner_did_not_join`,
  `time_limit`, `cancelled`), and, while open, `left` once the patient joined; absent otherwise. `endReason` stays
  as it was.
- 2026-11-05, package 7.1 (runner): the room rule for the practitioner. The practitioner's request carries no room
  (and must not), so `app.video_room` cannot be set for it. Migration 00023's idea is kept: a new read-only policy
  `video_session_attended` shows a clinic's scope narrowed to one appointment (`app.appointment_id`, the setting
  appointment's notice scope already uses) that appointment's session once its practitioner joined, whatever the
  consent. `Status` uses it only when the caller's own access rule finds nothing and the caller is a practitioner
  (not an admin): it reads the session narrowed to the appointment, requires `practitioner_joined_at`, and asks
  `appointment.Service.LookupPractitioner` (new: the practitioner of the role, in the caller's transaction and
  scope, unaudited, like `Lookup`) whether the caller is that practitioner. Audited as the read of the
  appointment; anyone else 404, audited without the entity.
- 2026-11-05, package 7.1 (runner): web. When the patient goes (present before, absent now) the app reads the
  status once and shows W18, W18a (`consent_withdrawn`) or W18b (`cancelled`); W18 while the read is pending or
  fails. The end screen: W18a for an end `cancelled` whose reason is `consent_withdrawn` (the revocation before
  the start), W18b for any other cancellation, otherwise as before (an end by the practitioner after a removal
  still says W28). W40 replaces W23 when LiveKit replaces the connection with a second tab, with W23's "Join
  again" button, which takes the call back to this tab (the other then shows W40); W40 stays on screen while that
  join runs.
- 2026-11-05, package 7.1 (runner): mobile. A removal or a deleted room reads the status: `consent_withdrawn` M27,
  `cancelled` M28, then the end as before; an open status without a removal rejoins. A rejoin refused for consent
  (a lost connection) still shows M27; that is a join the patient needed, not a probe.
- 2026-11-05, package 7.1 (runner): client logs. Web: `setLogLevel(LogLevel.silent)` when `call.ts` loads.
  Mobile: package 7's `lk.disableLogging()` threw (`package:logging` refuses a level on a non-root logger unless
  `hierarchicalLoggingEnabled` is on), so `LiveKitMedia` could not be built; it now turns hierarchical logging on
  first, and `logging` 1.3.0 (already resolved through livekit_client) is a direct, exact dependency for that.
  On native platforms livekit_client sends the token in a header, not the URL. Chrome's own network error lines
  still print a failed signalling URL with its token (see "Waiting on people").

- 2026-11-05, package 8 (runner): one runner. The Go journeys run in the `playwright` service, built from the
  pinned `mcr.microsoft.com/playwright:v1.55.0-noble` with dev.Dockerfile's Go toolchain copied in
  (`build/package/playwright.Dockerfile`), so the consultation journey starts the web journey as a subprocess
  (`test/e2e/web`, `@playwright/test` 1.55.0 exact) and hands it the synthetic data in variables, with no
  rendezvous between containers. The whole `make e2e` suite runs there; the load tests (`bench-*`) stay on `tools`.
- 2026-11-05, package 8 (runner): one LiveKit config serves both; no separate e2e config. The browser runs the
  Vite dev server of `web/` on its container's own localhost (a secure context for the camera and the `Secure`
  session cookie, as on a developer's host). The join answer's URL (`ws://localhost:7880`, for a host browser) is
  forwarded inside the container to `livekit:7880` by the spec's global setup, as Docker's published port does on a
  host; only signalling passes through it. Media goes straight to LiveKit's container address: the browser's
  selected candidate pair was `172.18.0.3:7882/udp host` (LiveKit's address on the compose network), and the
  loopback candidate leads nowhere in that container, so the media packets the journeys count prove the ICE
  settings of package 1 for in-network clients.
- 2026-11-05, package 8 (runner): the patient's tone is G.711 µ-law (PCMU, 8 kHz), generated in the test: Opus
  needs an encoder library (cgo libopus) the repo does not have, and LiveKit 1.13 accepts and forwards PCMU (seen:
  the practitioner's subscriber receives it, and the browser counts its inbound audio packets). Published as the
  microphone source, the only audio source the token allows.
- 2026-11-05, package 8 (runner): the order between the two processes is a condition, not a sleep. The Go test
  waits for the `Encounter` to be `in-progress`, then creates a file the spec waits for (`expect.poll`) before it
  ends the consultation, so the in-progress state is always observed. The Go side fails at once if the browser
  exits before that.
- 2026-11-05, package 8 (runner): the second journey runs on a consultation that has started (start a minute
  ago), so the revocation removes the patient (`PARTICIPANT_REMOVED`) and the practitioner stays, as SPEC §2
  "Removal" says; the practitioner then ends it through the API (`ROOM_DELETED`), the `Encounter` finishes and the
  appointment is fulfilled. The practitioner's side of this journey is a Go participant: the browser journey is
  the first one.
- 2026-11-05, package 8 (runner): the LiveKit log check reads LiveKit's whole container log (`docker compose
  logs`, which only the host can read), piped into `scripts/e2e-livekit-log.sh` in the playwright container, with
  the values the journeys wrote to the `e2e` volume. A value is matched as a whole word, ignoring case: the first
  run found the given name "Ena" inside ordinary words of the log (a false positive) when matched as a substring.

## Waiting on people

- Before production: `SPEC.md` §8 (LiveKit domain, TLS, TURN, keys, node IP; retention and lawful basis;
  localisation).
- Package 7.1, for the owner: Chrome itself prints a failed signalling connection's URL, with the token in its
  query, to the practitioner's own console (`WebSocket connection to 'ws://…/rtc/v1?access_token=…' failed` and
  the `…/rtc/v1/validate?access_token=…` request). No log level reaches it: it is the browser's network log, and
  LiveKit's browser signalling carries the token in the URL because a browser WebSocket cannot send a header. The
  token is valid 5 minutes for that one room. Accept, or ask for another measure.
- Package 7, for the owner: (1) whether M35 should have its own button before the Android system prompt
  (COPY.md has no label; the prompt now follows once M35 is on screen); (2) whether to support Bluetooth headsets
  on Android 12 and later, which needs `BLUETOOTH_CONNECT` and approved text for its prompt; (3) text for a
  patient replaced by their own second device (now: back to the device check, no message).
- Package 8, for the owner: LiveKit logs each client connection's `remoteAddress` (IP and port) and SDK at `info`;
  in production a client's IP is personal data. Decide with the production LiveKit setup (SPEC §8) whether to keep
  it, lower the level or filter it.

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

### 2026-11-05 — package 7: Mobile

**What was built.**
- `mobile/pubspec.yaml`: `livekit_client` 2.13.2 (the current release, published 2026-10-26; it pins
  `flutter_webrtc` 1.6.3) and `permission_handler` 13.0.2 (current, 2026-09-04), both exact, with their
  dependencies in the lockfile.
- `mobile/lib/call.dart` is the one file that uses livekit_client and permission_handler. It holds the `Media`
  and `Room` contract the screens use, and `LiveKitMedia`: the permission state (allowed, not asked, refused),
  the system prompt, the app's settings, previews (the camera's picture, the microphone's level from LiveKit's
  audio visualizer), and the room. The room publishes camera and microphone only and sets no name, metadata or
  attributes; LiveKit's client log is off. It maps LiveKit's events to peer, reconnecting, reconnected and
  disconnected, with an `Ending`: left, removed, ended, replaced or lost. `test/fake_call.dart` is its fake.
- `mobile/lib/device_check.dart`: M3 and M4, M35 before the system prompt on Android, the camera preview, the
  microphone level and M5, the denied and missing states (M9 to M12) with "Open settings", the "Join with …
  off" options (M6), Join and Joining… (M7, M8). A device allowed in the settings opens when the app comes back.
- `mobile/lib/call_screen.dart`: joining, waiting (M13, M14), the call (the practitioner's video large, the own
  camera small; M15 to M17), the practitioner leaving (M18), reconnecting (M19 to M21), the end (M22 to M26),
  removed (M27, M28), and the join errors (M29, M30, M32 to M34).
- `appointments_screen.dart`: M1 inside the join window, M2 before it; the list reads the clock every 15
  seconds and is read again after the call. `booking.dart`: the join and status calls. `messages.dart`: every M
  string from `COPY.md`, word for word, with its number. `ui.Page`: a page may have no title. The `Media` is
  passed from `main.dart` through the screens to my appointments, as the clock is.
- iOS `Info.plist`: `NSCameraUsageDescription` and `NSMicrophoneUsageDescription` as approved. Android
  `AndroidManifest.xml`: the permissions under Decisions.
- `README.md`: the LiveKit URL on an emulator (`ws://localhost:7880` through `adb reverse`). `docs/privacy/dpia.md`:
  package 7.

**Tests seen failing first** (`docker compose … run flutter flutter test`).
- The first run, with `test/fake_call.dart`, `call_test.dart`, `copy_test.dart` and the new
  `appointments_test.dart` cases written: compilation failed, `Error when reading 'lib/call.dart': No such file
  or directory` and `No named parameter with the name 'media'`.
- With `call.dart`'s contract and the `media` parameter in place, before any screen: 47 failed. Among them every
  device check, joining, call and end test (`Found 0 widgets with type "FilledButton"` for Join consultation),
  both window tests (`Expected: exactly 2 matching candidates, Actual: Found 0`), and the iOS and Android
  manifest tests (`Expected: contains '<key>NSCameraUsageDescription</key>…'`).
- With the screens written, 2 failed on test bugs: a record compared with a matcher (`Expected: (true,
  _IsNull)`), and a tap on a Join below the screen's edge; fixed in the tests.
- `copy_test.dart`'s strings check passed on its first run, because `messages.dart` was written before the
  suite compiled. It was then seen failing with M14 changed by one character (`Expected: contains '$name will
  join you here. Keep this screen open.'`), and passing again with the approved text.
- `call_adapter_test.dart` (the disconnect reasons, the permission states, the level) was written after
  `call.dart`; it was seen failing against three broken mappings (`Expected: Ending.removed, Actual:
  Ending.lost`; `Access.unasked`/`Access.refused`; `0.4`/`0.2`) and passing with the real ones.
- Found while reviewing, test first: `says nothing about the practitioner while it asks why it was removed`
  failed (`Found 1 widget with text containing has left the`) before the stage hid its lines without a room.
- Not seen failing: `every consultation screen fits a phone 390 px wide` and `prints nothing through a
  consultation` passed on their first run, as the screens were already written; they guard against regressions.

**What the tests cover.** The window to the second (10 minutes before the start, until 30 minutes after the
end), closed for every status but `booked`, M2 before it, and Join appearing when the clock passes the opening.
The device check: preview and level, no prompt when both were allowed, M35 on screen before the system prompt
on Android and not on iOS, a denied camera (M9, settings opened, joins with audio only, camera turn-on
disabled), a camera allowed in the settings opening on return, a denied microphone (M10), both denied still
joining, missing devices (M12), joining with either off, silence after 5 seconds and again after sound, an
unmeasured level, previews stopped on back and on Join. Joining: the token request without the CSRF header, the
URL and token passed to the room, Joining…, each refusal's text with no connection, `practitioner_did_not_join`
(M25, M26), a failed connection (M34). In the call: waiting until the practitioner's participant appears, the
lines without a name, camera off and muted, the practitioner leaving, the toggles, leave and back, reconnecting
and reconnected, a lost connection rejoined with a new token and the same settings, a failed rejoin with "Try
again", a second device. The end: a deleted room for each `endReason`, a room deleted while open rejoined,
removed after a revocation (M27) and a cancellation (M28), removed with access still there rejoined, a rejoin
refused as ended, no loop when the status is still open, the window closing while waiting (M25). 390 px without
overflow, and nothing printed. Every M string against `COPY.md`, the iOS texts against `Info.plist`, the Android
permissions against the manifest.

**Checks.** `make mobile-analyze`: no issues. `make mobile-test`: 133 tests pass. `dart format --line-length 120
--set-exit-if-changed lib test`: no change. `make gen-clients`: no diff. `make lint`: 0 issues in both
modules, no migration from `main` changed. `make archcheck`: passes. `make test`: passes. `make cover`: every
package at 80% or more, total 94.9%. No Go, `openapi.yaml` or FHIR change, so `make sqlc` and `make
fhir-validate` are not affected.

**Not run.** The app on a device or an emulator (a manual step, README.md): there is no Flutter, Android SDK,
emulator or Xcode on this machine, and the `flutter` image has no Android or iOS toolchain. So `LiveKitMedia`
(the permissions, the previews, the visualizer's level and the room) is checked by the analyzer only, and the
`adb reverse` route in README.md has not been tried. The first device run should check: the M35 frame before
the prompt, "Open settings", the camera preview, the level against the 0.02 threshold, audio and video both
ways with the web app, and a reconnect.

**Deferred.**
- `BLUETOOTH_CONNECT` and its explanation, if the owner wants Bluetooth headsets on Android 12 and later.
- The main Android manifest has no `INTERNET` permission (only the debug and profile ones have it), so a
  release build reaches neither the API nor LiveKit. It predates this package; the release build is not set up
  yet.
- The Playwright and Go participant journeys are package 8.

**Dev stack left.** No migration and no data: the tests use the fake API and the fake room. The stack was up
when the package started and was left up, unchanged.

### 2026-11-05 — package 7.1: Owner changes after package 6

**What was built.**
- `migrations/00024_video_session_reason.sql`: `video_session.removal_reason` (`consent_withdrawn` | `cancelled`,
  `UPDATE` granted to `salus_app` for that column only), and the read-only policy `video_session_attended` (a
  clinic's scope narrowed to one appointment sees its session once its practitioner joined, whatever the
  consent). SPEC §5 has the column's line; the DPIA has a package 7.1 note.
- `videosession`: `Reason` and `Status.Reason` (Decisions above); `Repository.Removed` (first reason stays, room
  scope) and `SetAppointment`; `removeFromRoom` records the reason with its audit event (`EndAccess`
  `cancelled`, `EndPatientAccess` `consent_withdrawn`, the webhook's re-check from `accessGone`, which now asks the
  consent first); `Status` reads by the room rule (`attended`) when the caller's own rule finds nothing.
  `handler_http.go` answers `reason`.
- `appointment.Service.LookupPractitioner`: the practitioner of the appointment's role in the caller's
  transaction and scope, unaudited.
- `api/openapi.yaml`: `ConsultationStatus.reason` and the read's description; `make gen-clients`.
- Web: `CallScreen.tsx` reads the status once when the patient goes (W18, W18a, W18b), the end screen's W18a for a
  revocation before the start, W40 for a second tab; the join probe is gone. `messages.ts`: W40. `call.ts`:
  livekit-client's log level `silent`.
- Mobile: `call_screen.dart` reads the status on a removal or a deleted room (M27, M28) and no longer asks the
  join endpoint; `call.dart` turns hierarchical logging on before turning LiveKit's log off (it threw before);
  `pubspec.yaml`: `logging` 1.3.0 exact.

**Tests seen failing first.**
- migrations: TestVideoSession_RecordsWhyThePatientWasRemoved: column "removal_reason" of relation "video_session" does not exist (42703), 6 errors. TestVideoSession_OneAppointmentsScopeSeesItsSessionOnceThePractitionerJoined: clinic B with the cancelled appointment: sees [bPast], want [bPast bFuture].
- videosession repository: build failed, undefined: videosession.Reason, ReasonConsentWithdrawn, ReasonCancelled; r.Removed, got.RemovalReason, r.SetAppointment undefined.
- videosession service (fakes), with Status.Reason and the fake's LookupPractitioner in place: TestStatus_SaysWhyTheConsultationEndedOrThePatientWasRemoved failed on all 11 paths with a reason, e.g. `Status() = {… State:ended EndReason:practitioner Reason:}; want ended with the reason "ended"`, `{… State:open … Reason:}; want open with the reason "consent_withdrawn"` (revocation after the start), `{… EndReason:cancelled Reason:}; want … "consent_withdrawn"` (before the start), `want open with the reason "left"`. TestStatus_KeepsTheFirstRemovalsReason: `want ended cancelled, with the reason consent_withdrawn`. TestStatus_ThePractitionerWhoWasInTheRoomReadsItAfterTheClinicLostSightOfIt: `Status() by the practitioner = {}, videosession: not found; want {State:ended EndReason:cancelled Reason:consent_withdrawn …}`. TestStatus_TheRoomRuleFailsWithTheBackend: `Status() = videosession: not found, want the backend's error`. (TestStatus_TheRoomRuleShowsNobodyElse passed: it asserts the refusals that already held.)
- the reason table's removal paths then failed on a test bug: one shared slice for the room's participants, which the fake's removal changes, so later cases had nobody to remove (`Reason:left; want … consent_withdrawn`); each case now gets a fresh slice. Existing status and end tests gained the reason in their expected values (`Reason:left`, `practitioner_did_not_join`, `ended`).
- appointment: TestLookupPractitioner_GivesThePractitionerOfTheRoleInTheCallersScope: build failed, f.svc.LookupPractitioner undefined.
- api (real database, salus_app): TestStatus_ThePractitionerWhoWasInTheRoomReadsItAfterARevocationBeforeTheStart was written after the service code, so it was seen failing with the room rule switched off in `attended`: `Status() by the practitioner = {}, videosession: not found; want ended cancelled, with the reason consent_withdrawn` and `0 reads of the appointment by the practitioner and 3 refusals audited, want 1 and 3`; restored, it passes.
- videosession handler: TestStatusAndEndHandlers_AnswerTheState failed on the open and the ended cases: the body had no `"reason"` (`want 200 {…,"reason":"consent_withdrawn"}`).
- web CallScreen (Vitest): 5 failed. `says why when the patient left on their own, from the status alone`: `expected [ 'POST …/consultation/join', 'GET …/consultation' ] to deeply equal [ 'GET …/consultation' ]` (the join probe); withdrawn consent and cancellation: `Unable to find an element with the text` (the status was not read for the reason; W18b not shown); `says the consultation is open in another tab…`: W40 not found; `…when the room is deleted, by a revocation before the start`: W18a not found (W18b shown).
- web call.ts (Vitest, jsdom with a stub RTCPeerConnection so livekit-client tries the signalling connection to a closed port): `prints nothing, and no token, when the connection fails` failed, livekit-client printed `connection state changed: disconnected -> connecting …` and `connecting -> disconnected` (in a probe before the test, at its default info level it also printed `signal connecting to ws://…/rtc/v1?access_token=%3Credacted%3E&join_request=…` and `Couldn't connect to server, attempt 1 of 1`).
- web: `keeps saying the consultation is open elsewhere while it joins again from this tab` failed with `Unable to find an element with the text: The consultation is open in another tab or window.` while the busy state dropped otherTab; then once on a test bug (the held join answered before the request reached it; the test now waits for the request). `…by the practitioner after removing the patient for a withdrawn consent` (W28, not W18a) was written after endLine's first version and seen failing against it.
- mobile (flutter test): `LiveKit's own log is off` failed with `Unsupported operation: Please set "hierarchicalLoggingEnabled" to true if you want to change the level on a non-root logger.` from `lk.disableLogging` in `new LiveKitMedia` (call.dart:118): package 7's call threw, so LiveKitMedia could not be built at all (never run on a device). call_test.dart: 4 failed. `removed after withdrawing consent says so`: M27 not found (the status's reason was not read; the join probe rejoined); `says nothing about the practitioner while it asks why it was removed`: `Found 1 widget with text containing has left the` (the held status was not what it waited on); `removed after a cancellation says so`: M28 not found; `a deleted room says the consent was withdrawn when a revocation before the start cancelled it`: M27 not found (M28 shown).
- web, found in the browser walk (three status reads after one removal, from LiveKit's track events while the patient was gone), test first: `reads the status once when the patient goes, however many events LiveKit sends while they are gone` failed with `expected [ …(3) ] to deeply equal [ Array(1) ]`.

**What the tests cover.** The reason per path through the service: the practitioner's end, the window's close
with the room empty and without the practitioner, the hard cap, a cancellation, a revocation after and before
the start, the webhook's re-check (consent revoked, appointment out of scope, cancelled), a plain leave, nobody
joined; the first removal's reason kept. The room rule with fakes (the practitioner after a revocation before the
start; another practitioner, the admin, another clinic and the practitioner before they joined refused, audited;
a backend failure) and on the real database as `salus_app` (the practitioner reads it, audited; the colleague,
the admin and another clinic's practitioner get 404, audited; the patient reads `consent_withdrawn`). In SQL: the
new policy for every scope, read only; the column's grant and check. The handler's JSON. Web: W18, W18a, W18b
from the status alone with no join request, one status read however many LiveKit events, W18 when the read fails,
W40 and its rejoin, the end lines (W18a for the revocation before the start, W28 for the practitioner's end after a
removal), livekit-client printing nothing on a failed connection. Mobile: M27 and M28 from the status with no
join request, M27 on a deleted room after a revocation before the start, a rejoin when the status shows no
removal, LiveKit's log off with nothing reaching a listener.

**Browser walk** (`make up`, which applied migration 24; the Vite dev server on `SALUS_WEB_PORT=5175`; Chromium
from the Playwright 1.55.0 image, headless with fake devices, driven by a throwaway Python script; the patient was
a second browser context using livekit-client's UMD build with a token from the join endpoint and the patient's
bearer). Two synthetic consultations per run, made through the API by a temporary e2e helper (deleted before the
commit). The first run stopped on a script error (it revoked the consent before a step that needed it); the
second, on fresh data, passed every step:
- The patient left: W18, and the only request was the status `GET` (no join).
- A second tab: W40 in the first tab at 390 px with no horizontal scroll; "Join again" took the call back and the
  second tab showed W40.
- A join answer rewritten to a closed port: W34; livekit-client printed nothing.
- A revocation after the start: LiveKit removed the patient (reason 4), W18a, one status `GET`, no join; 390 px
  without horizontal scroll. "End consultation" then said "The appointment is marked as fulfilled."
- A revocation before the start, both in the room: the room was deleted and the practitioner saw W27 with W18a,
  read by the room rule (two status reads: the patient's removal and the room's deletion, no join); 390 px.
- The console: the app and livekit-client printed nothing but Vite's and React's development lines. Chrome's own
  network errors printed the failed signalling URL and the validate request with the token (Waiting on people).
- Found and fixed in the walk, test first: three status reads after one removal (LiveKit's track events while
  the patient was gone).
- The api's and the worker's logs from the walk held no name, e-mail, room, identity or token; LiveKit's held no
  name or e-mail.

**Checks.** `make gen-clients`: no diff after the regeneration in this package. `make lint`: 0 issues in
both modules, no migration from `main` changed. `make archcheck`: passes. `make test`: passes. `make cover`: every
package at 80% or more (`internal/videosession` 94.7%, `internal/appointment` 93.2%, total 94.8%). `make web-lint`:
0 warnings, 0 errors. `make web-typecheck`: passes. `make web-test`: 16 files, 209 tests pass. `make web-build`:
passes, with Vite's chunk-size warning (835 kB, SPEC §7). `make mobile-analyze`: no issues. `make mobile-test`: 135
tests pass. `dart format --line-length 120 --set-exit-if-changed lib test`: no change. W40 checked word for word
against `COPY.md`. No FHIR change, so `make fhir-validate` is not affected.

**Deferred.**
- The mobile app has still not run on a device (package 7's note); the logger fix is checked by the test only.
- Chrome's own network error lines with the token (Waiting on people).

**Dev stack left.** Migration 24 is applied (goose version 24). Besides package 6's walk data, the dev
database holds this walk's, made through the API: 4 clinics, each with an admin and a practitioner, 4 patients
and 4 appointments with their slots, jobs, audit events and e-mails in Mailpit. The first run stopped on a script
error after its revocation: its started appointment is booked with the consent revoked, its session open with
`removal_reason` `consent_withdrawn` (its window-close job will end it and delete the room), and its other
appointment is booked with an active consent and no session. The second run's: one fulfilled (session ended by
the practitioner, `removal_reason` `consent_withdrawn`) and one cancelled by the revocation before its start
(session ended `cancelled`, `removal_reason` `consent_withdrawn`). The Vite dev container was stopped and
removed; the Playwright image stays in the host's Docker cache.

### 2026-11-05 — package 8: End-to-end

**What was built.**
- `build/package/playwright.Dockerfile` and the `playwright` compose service (profile `e2e`, image
  `salus-playwright:1.55.0`, host user, `init`), with the `e2e` volume for the log check; `make down` stops it.
- `make e2e` now: `make up`, `web-install`, `e2e-install` (the image and `npm ci` in `test/e2e/web` when its
  lockfile changed), the whole Go suite in the playwright service, then LiveKit's log check.
- `test/e2e/consultation_test.go` (tag `e2e`): `TestConsultation_ThePractitionerEndsTheCallFromTheWebAppAndTheAppointmentIsFulfilled`
  and `TestConsultation_ARevocationDuringTheCallRemovesThePatient`; the Go participant (`connect`, the PCMU tone,
  the first-RTP-packet signal, LiveKit's disconnect reason); `startAt` and `ownerConn` moved here from the removal
  load test, which now uses `connect`. `booking_test.go`'s clinic keeps the practitioner's e-mail, name and
  authenticator for the web sign-in.
- `test/e2e/web`: `playwright.config.ts` (Chromium with `--use-fake-device-for-media-stream` and
  `--use-fake-ui-for-media-stream`, the Vite dev server as `webServer`, output in `/tmp`), `livekit-forward.ts`,
  `consultation.spec.ts`: sign in with the password and a TOTP code, the appointment's "Join consultation", the
  device check, "Join", W16 for the patient without camera and no W17, the patient's audio packets counted from
  the page's peer connections, then on the Go test's signal W19, W20, W21 and the end with W27 and W28.
- `scripts/e2e-livekit-log.sh`. CI: the Go workflow's e2e job has its own path filter, which adds `web/` and the
  node and playwright Dockerfiles. DPIA: a package 8 note.

**Tests seen failing first.**
- The revocation journey, the first run against the stack: `the practitioner was disconnected with "other
  reasons", want "room closed"`; the SDK maps `ROOM_DELETED` to "other reasons", so the journeys compare LiveKit's
  own reason (`Room.DisconnectReason()`).
- `make e2e`, first run: the journeys passed and the log check failed, `LiveKit's log holds the run's synthetic
  patient given name` (twice): "Ena" as a substring of other words. Whole-word matching fixed it; the log holds
  none of the 22 values.
- The browser journey with the forward pointed at a closed port (a probe, reverted): the spec failed at
  `getByText("Ena Syntheticova's camera is off")` (not found) and the Go test at `the Encounter in progress once
  the practitioner joined: not within 30s` after 30 s, which is why the Go side now fails as soon as the browser
  exits.
- Before this package, `make e2e` had no runner for the spec (no playwright service, no `test/e2e/web`).

**Login rate limit.** The suite now logs in 9 times per run from one container address (accounts 4, booking 2,
consultation 2 patients and the practitioner's web sign-in), against 30 per IP per 15 minutes (accounts SPEC), so
`make e2e` runs three times in a row in one window; my probe runs before the final ones used up the window once
(429 on every login), and the final runs waited for it to expire rather than deleting the key.

**Checks.** `make e2e` twice in a row: both pass (the Go suite in 11.7 s and 11.4 s, the consultation journeys about 4.7 s and 1.9 s of it; the browser journey itself about 1.6 s), with the existing accounts and booking journeys, and the log check after each: "none of the run's 22 synthetic values and each of its 2 rooms" in 2,764 and 2,823 lines. `make lint`: 0 issues in both modules, migrations unchanged.
`make archcheck`: pass. `make test`: pass. `make cover`: every package at 80% or more, total 94.8%. `make
fhir-validate`: pass (58 files). `make gen-clients`: no diff. `make web-lint`, `web-typecheck`, `web-test` (209
tests), `web-build`: pass. `make mobile-analyze`: no issues; `make mobile-test`: 135 tests pass.

**Deferred.** Nothing of this package. `@playwright/test` 1.55.0 has an npm audit advisory about its browser
download (`playwright install`), which the runner never calls: the browsers come with the pinned image of the same
version. Moving to 1.55.1 or later means moving the image too.

**Dev stack left.** No migration. Each `make e2e` run adds its synthetic data through the API, as before: 5
clinics with their admin and practitioner, 5 patients, 4 appointments (2 fulfilled consultations with finished
`Encounter`s, 2 cancelled), their slots, jobs and audit events. New: the `salus_e2e` volume (the last run's log
check) and the `salus-playwright:1.55.0` image; `test/e2e/web/node_modules` (git-ignored). The stack is up as it
was.

