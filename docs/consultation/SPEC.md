# Video consultations

Status: approved, 2026-11-05, with the owner's changes (section 2, "Changed on approval"). Branch `feat/consultation` from `main`, one commit
per package (`consultation(<n>): <summary>`), one PR at the end.

## 1. Goal

A booked appointment is attended as a one-to-one video consultation. Shortly before the start, the patient
(mobile) and the practitioner (web) see a "Join" button. Each checks their camera and microphone, asks our API
for a short-lived LiveKit token, and joins a room that only the two of them can enter. The patient waits until
the practitioner is there. When the practitioner joins, the FHIR `Encounter` starts; when the practitioner ends
the consultation, or the join window closes with the room empty, the `Encounter` finishes, the room is deleted,
and the `Appointment` becomes `fulfilled`, or `noshow` if the patient never joined. A revoked consent or a
cancelled appointment ends the patient's access at once. Video runs on our own LiveKit (ADR 0006); nothing is
recorded and LiveKit learns nothing about who the participants are.

## 2. Settled decisions

Given by the owner. Do not reopen them.

| Topic | Decision |
| --- | --- |
| Scope | Every booked appointment is a video consultation; there are no in-person types yet. One-to-one: the appointment's patient and its practitioner, nobody else. Live video and audio, a camera and microphone check before joining, mute and camera toggles, and an end-of-consultation step for the practitioner |
| Transcription | Not in this feature. It needs a speech-to-text engine decision for bs/hr/sr and its own SPEC. `cmd/agent` stays a stub |
| Out of scope | Recording (none is kept, ADR 0006), screen sharing, in-call chat, more than two participants, push notifications, in-person appointments, transcription |
| Domains | Context `consultation`: `videosession` (the LiveKit provider port and its implementation; creates rooms, issues tokens) and `encounter` (the FHIR `Encounter` of the consultation, class `VR`, referencing the `Appointment`) |
| LiveKit SDK | The official LiveKit Go server SDK, pinned exact, as the protocol client (§3). LiveKit types never leave `repository_livekit.go` |
| Join window | Opens 10 minutes before the start; closes 30 minutes after the scheduled end, or when the practitioner ends the consultation |
| Join token | A JSON endpoint issues a LiveKit join token only to the appointment's patient or its practitioner. The access rule and an active consent are checked in the service; every issued or refused token is audited. Tokens are valid 5 minutes for joining and grant only that one room |
| Opaque to LiveKit | Room names and participant identities are opaque random ids. No names, e-mails, specialties or appointment details go to LiveKit as room names, identities or metadata. The clients get the other person's display name from our API, not from LiveKit |
| Room creation | On the first token request, not when the appointment is booked |
| Waiting | The patient waits on a "waiting for the practitioner" screen until the practitioner is in the room |
| Webhooks | LiveKit webhooks go to the api, signed and verified, handled idempotently: a repeated or out-of-order event has no further effect |
| Encounter start | When the practitioner first joins, the `Encounter` starts (`in-progress`) |
| Encounter end | When the practitioner ends the consultation, or the window closes with the room empty: the `Encounter` finishes, the room is deleted, and the `Appointment` becomes `fulfilled`; if the patient never joined, `noshow` |
| Revocation, cancellation | A revocation or a cancellation during the window ends access at once: further token requests are refused, and the provider port removes the participant |
| Processor | Self-hosted LiveKit, so no processor; recorded in `docs/privacy/processors.md` and the DPIA notes |
| LiveKit logs | Configured to carry no participant names, checked in the e2e stack |
| Before production | Production TLS, TURN and the LiveKit domain (§8) |
| Latency | The token endpoint's latency is measured and recorded |
| Web client | `livekit-client` (npm, pinned), no LiveKit UI components. A custom call screen in the existing visual direction: light, clean, mint accents, flat background, minimal custom SVG icons, only the necessary controls |
| Mobile client | `livekit_client` (Flutter, pinned). Camera and microphone permissions on iOS and Android, with clear wording |
| Both clients | Handle a lost connection and reconnect, a denied permission, and "the consultation has ended" |
| Copy review | Every new user-facing string on the web and mobile screens goes to the owner on its own, before package 6. Nothing that depends on it merges until the owner has answered |

Decisions taken in this plan (reopen only with the owner):

- **Dependencies.** `videosession` → `encounter` → `appointment`, one way. `videosession` also uses
  `appointment` (the appointment and its access facts), `consent` (through those facts), `access`, `patient` and
  `practitioner` (display names) and `auditevent`. `encounter` knows nothing about LiveKit; `appointment`
  knows nothing about either. Both new domains go in the `consultation` context in `config.yaml`.
- **Tables.** Two, one per domain:
  - `video_session` (owner `videosession`): `id`, `appointment_id` (unique, references `appointment`),
    `organization_id`, `room` (random, unique), `patient_identity`, `practitioner_identity` (random, unique),
    `patient_joined_at`, `practitioner_joined_at`, `ended_at`, `end_reason` (`practitioner` | `window_closed` |
    `practitioner_absent` | `time_limit` | `cancelled`), timestamps. Created on the first token request (`INSERT … ON CONFLICT (appointment_id) DO
    NOTHING`, then read), so two simultaneous first requests share one room.
  - `encounter` (owner `encounter`): `id`, `appointment_id` (unique), `organization_id`, `patient_id`,
    `practitioner_role_id`, `status` (`in-progress` | `finished`), `period_start`, `period_end`, `version_id`,
    timestamps. `patient_id` and `practitioner_role_id` are the Encounter's own `subject` and `participant`,
    copied once from the appointment when it starts and never changed (they are references, not a copy of
    the appointment's data); RLS needs them.
  Both are row-level secured like `appointment`: patient scope, clinic scope with an active consent or for an
  appointment that started before the latest revocation, and the webhook scope below (`app.video_room`; on
  `encounter` it shows the encounter of that room's appointment, changed after package 5, so a revocation that
  cancels an appointment after the practitioner joined can still finish its `Encounter`, with `period.end` set).
  Grants to `salus_app`; `video_session` also `SELECT, DELETE` to `salus_retention`.
- **Ids.** `room` and both identities are 128 bits from `crypto/rand`, base32, prefixed `r_` and `p_` so a log
  reader can tell them apart; never derived from any id of ours. An identity is fixed per role per session, so
  a reconnect from a second tab or device replaces the first connection (LiveKit's rule for a repeated
  identity) and the room never holds more than two.
- **Token grant.** `roomJoin` for that one room; `canSubscribe`; `canPublish` with sources camera and
  microphone only (no screen share); `canPublishData: false` (no chat); no `name`, no `metadata`, no
  `attributes`; `ttl` 5 minutes. `roomCreate`, `roomList`, `roomAdmin` and `recorder` are never granted.
- **Rooms.** `CreateRoom` with `max_participants: 2`, `empty_timeout` long enough to outlast the window (so
  LiveKit does not close a room while a participant reconnects), no metadata. LiveKit's `room.auto_create` is
  off, so a token alone cannot create a room. The token request calls `CreateRoom` every time; it is idempotent
  in LiveKit, so a LiveKit restart during the window heals on the next request. Its cost is in the latency
  measurement (package 4).
- **Who may join.** `videosession.Service.Join(caller, appointmentID)`: the appointment is `booked`; the caller
  is its patient, or the practitioner of its role (`access.Appointment` is `Full`); the patient's consent with
  the clinic is active (for the practitioner too: without it the practitioner has no business in the room);
  the time is inside the window; the session has not ended. Otherwise: 404 `not_found` (not a participant, or
  no such appointment: existence is not disclosed), 409 `consent_required`, 409 `join_window_not_open`, 409
  `consultation_ended` (ended, or the appointment is `cancelled`, `fulfilled` or `noshow`). Every outcome writes
  one `AuditEvent`: type `rest`, subtype `restful-interaction` `operation`, action `E`, entity
  `Appointment/<id>`, purpose `PATRQT` for the patient and `TREAT` for the practitioner, outcome success or
  minor failure. Admins cannot join (404, audited).
- **Join response.** `POST /appointments/{id}/consultation/join` → `200 {"url", "token", "expiresAt",
  "peerName", "role"}`: the LiveKit URL for clients (`LIVEKIT_PUBLIC_URL`), the token, its expiry, the other
  person's display name, and the caller's role (`patient` | `practitioner`). For the patient `peerName` is the
  practitioner's public name (as in the directory); for the practitioner it is the patient's name at the
  appointment's access level, and that writes the `Patient` audit event too (booking SPEC §2 "FHIR reads").
  Token, identity and room are never logged.
- **Webhook endpoint.** `POST /webhooks/livekit`, no session, outside CSRF (that check applies to cookie
  sessions only). The body is read up to 64 KiB, and the provider port verifies the `Authorization` JWT (API
  key and secret, and the body's SHA-256 in the claim) before anything else is read; a bad or missing signature
  is 401, logged with the request id only. Valid events are answered 200 once handled, also when they have no
  effect; a handling error is 500 through `httpx.InternalError`, so LiveKit retries. The handler maps the event
  to a domain `Event` inside `repository_livekit.go`; nothing else sees LiveKit's types.
- **Idempotent, order-free handling.** No event log table: every effect is a monotonic state change guarded in
  SQL, so a repeat or a late event changes nothing.
  - `participant_joined`, practitioner identity: set `practitioner_joined_at` if null, and
    `encounter.Service.Start` (creates the `Encounter` `in-progress` with `period.start` the join time LiveKit
    reports, `ON CONFLICT (appointment_id) DO NOTHING`). Ignored after `ended_at`.
  - `participant_joined`, patient identity: set `patient_joined_at` if null; ignored after `ended_at`. Then the
    service checks the patient's access again (consent active, appointment `booked`); if it is gone, it removes
    the participant (a second guard behind the token check, for a join with a token issued just before the
    revocation committed).
  - `participant_left`, `room_finished`: when the window has closed and the room is empty, end the session
    (`window_closed`). Before the window closes they do nothing: a dropped connection is not the end.
  - `room_started`, `track_published` and every other kind: 200, no effect.
  - An identity or room we do not know: 200, no effect, logged as `unknown room` with no ids from the body.
- **Webhook scope.** A webhook carries no user. Its transaction sets `app.video_room` to the event's room; the
  `video_session` policy shows the one row with that room, and the service then sets `app.organization_id` to
  that row's clinic for the `Encounter` and `Appointment` writes, like the notifier's `app.appointment_id`
  (booking SPEC §2 "Row-level security scopes"). The writes are audited with agent name `livekit_webhook`,
  purpose `HOPERAT`.
- **Ending.** One function, `videosession.Service.end(session, reason)`, used by all three paths: in one
  transaction it sets `ended_at` and `end_reason` (only if null), calls `encounter.Service.Finish` and, through
  it, `appointment.Service.RecordConsultationOutcome`; after the commit it deletes the room through the port
  (a failure is retried by the path that called it; a room left behind expires on its `empty_timeout`).
  - **The practitioner ends it:** `POST /appointments/{id}/consultation/end`, the appointment's practitioner
    only (404 for anyone else, audited), at any time after they joined; repeated, it answers 200 again.
  - **The window closes:** the first token request enqueues `consultation_window_close` (session id, clinic id)
    with `ScheduledAt` the window's close, in the session's insert transaction. If the practitioner never
    joined, it ends the session (`practitioner_absent`, see below). Otherwise it asks the port how many
    participants the room has; empty: it ends the session (`window_closed`); not empty: it snoozes 1 minute, so
    a consultation that runs over continues until the practitioner ends it or both leave. No new token is
    issued once the window has closed.
  - **The hard cap (changed on approval):** 2 hours after the scheduled end the job ends the session
    (`time_limit`) and deletes the room whether or not anyone is in it; the `Encounter` finishes and the
    outcome is recorded as for any other end. The snooze never goes past the cap.
  - **Cancelled:** see "Removal".
- **Outcome.** `appointment.Service.RecordConsultationOutcome(ctx, agent, appointmentID, patientJoined)`: a new
  system method (no caller role), in the caller's transaction. A `booked` appointment becomes `fulfilled` if the
  patient joined, else `noshow`; any other status is left as it is (a practitioner who already marked it by hand
  wins, a cancellation stays a cancellation). Audited as an `Appointment` update by the agent. The existing
  manual `SetStatus` is unchanged.
- **The practitioner never joins.** Then there is no `Encounter`: the window-close job deletes the room and ends
  the session (`practitioner_absent`), and the appointment stays `booked`, for the practitioner or the clinic to
  mark or cancel by hand as today. `video_session.patient_joined_at` records whether the patient came (changed
  on approval), and the clinic sees it (below). The patient's app shows a "the practitioner did not join" ended
  state: a join request on an ended session answers 409 `practitioner_did_not_join` instead of
  `consultation_ended`, and the waiting screen asks again when the room is deleted under it.
- **Consultation status (changed on approval).** `GET /appointments/{id}/consultation` → `200 {"state",
  "patientJoinedAt", "practitionerJoinedAt", "endedAt", "endReason"}` (`state`: `not_started` | `open` |
  `ended`), for the appointment's patient, its practitioner and the clinic's admin under `access.Appointment`
  (`Name` or more); 404 for anyone else. Audited as a read of the `Appointment`. The web app's appointment
  detail shows whether the patient and the practitioner joined. **Changed after package 6:** the status also
  carries `reason`, why the session ended or the patient was removed: `consent_withdrawn`, `cancelled`, `left`,
  `ended`, `practitioner_did_not_join`, `time_limit`. The practitioner who was in the room keeps read access to
  the status under the room-based rule (as for `encounter`), so a revocation or cancellation that hides the
  appointment from the clinic still lets them read why. Neither client asks the join endpoint to learn why
  someone left; the status is the only source.
- **Encounter.** FHIR R4: `status` (`in-progress` → `finished`); `class` `http://terminology.hl7.org/CodeSystem/v3-ActCode`
  `VR` "virtual"; `subject` `Patient/<id>`; `participant` one, `type` `v3-ParticipationType` `PPRF`,
  `individual` `PractitionerRole/<id>`; `appointment` `Appointment/<id>`; `period` `start`, and `end` when
  finished; `serviceProvider` `Organization/<id>`; `meta.versionId`, `meta.lastUpdated`. A finished encounter
  never changes again. Read: whoever `access.Appointment` gives `Name` or `Full` for its appointment; everyone
  else 404. Search by `appointment`, `patient`, `practitioner` (the `Practitioner` behind the role, R4's
  `Encounter.practitioner`), with the same scoping as the `Appointment` search: a patient only for themselves, a
  practitioner only for themselves, an admin for the clinic. Every read and returned resource is audited
  (booking SPEC §2 "Audit codes"). No FHIR create or update: the lifecycle drives it.
- **No `Provenance` yet.** The `Encounter` holds no clinical content, only that a consultation took place and
  when; its writes are audited as `AuditEvent`s with the system agent. `Provenance` arrives with the first
  clinical write (observations, a later SPEC).
- **Removal.** `videosession.Service.EndAccess(ctx, appointmentID)` removes the patient from the room through
  the port and, if the appointment was cancelled, ends the session (`cancelled`, which deletes the room).
  - **Cancellation:** `appointment`'s cancel transaction (patient, staff, or the revocation job) also enqueues
    `consultation_access_end` (appointment id, clinic id), whose worker calls `EndAccess`. With no session it
    does nothing.
  - **Revocation:** the `consent_revoked` worker, after `CancelForRevokedConsent`, calls
    `videosession.Service.EndPatientAccess(ctx, patientID, clinicID)`, which finds the patient's open sessions
    in that clinic (through `appointment`'s facts: an appointment started before the revocation is not
    cancelled, but its room is still open) and removes the patient from each. The practitioner stays and ends
    the consultation as usual.
  - The token refusal takes effect when the revocation or cancellation commits; the removal follows as soon as
    the worker picks up the job (River's notify, normally well under a second; measured in package 5). The
    `participant_joined` check above covers a join that slips between the two.
- **Configuration** (`internal/config`): `LIVEKIT_URL` (server API, `http://livekit:7880` in compose),
  `LIVEKIT_PUBLIC_URL` (what clients connect to, `ws://localhost:7880` locally), `LIVEKIT_API_KEY`,
  `LIVEKIT_API_SECRET` (synthetic dev values in compose; production values from the secret store, never in a
  committed file). The api and the worker load them; the notifier and the scheduler do not.
- **LiveKit in compose.** `deployments/livekit.yaml` replaces `--dev`: the dev key pair, `webhook.urls`
  `http://api:<port>/webhooks/livekit` with that key, `room.auto_create: false`, JSON logs at `info`. We never
  send a name, so LiveKit has none to log; the e2e suite reads LiveKit's log after a run and asserts that no
  synthetic name, e-mail or appointment id appears in it.
- **Tests against a real LiveKit.** `repository_livekit_test.go` runs against the compose LiveKit
  (`TEST_LIVEKIT_URL`, key and secret in the `tools` service), never a fake, like Redis; without the URL the
  tests fail with a message. CI's Go workflow starts LiveKit with Postgres and Redis.
- **Retention.** `video_session`: `delete` 30 days after `ended_at` (sessions never ended: 30 days after the
  appointment's end), applied by a purger in `videosession/service_retention.go`. `encounter`: `keep`, with the
  clinical record. Both **need a legal decision before production**, like `appointment`.
- **JSON API** (`api/openapi.yaml`, `make gen-clients`): `POST /appointments/{id}/consultation/join`, `POST
  /appointments/{id}/consultation/end`, `GET /appointments/{id}/consultation`, and `POST /webhooks/livekit` (documented, called only by LiveKit).
  Errors keep `{"error": "<code>"}`. The `Encounter` is read through FHIR.
- **Join button.** Shown on the appointment (web: appointment detail and the calendar card; mobile: my
  appointments) while `now` is inside the window computed from the appointment's start and end and its status
  is `booked`. The client's clock only decides whether the button shows; the server decides whether a token is
  issued.
- **Client states.** Both clients: device check (preview, microphone level, camera and microphone pickers on
  web) → joining → waiting for the other person (the patient: "waiting for the practitioner") → in call →
  reconnecting (LiveKit's `Reconnecting`/`Reconnected`; on `Disconnected` with a network reason the client asks
  for a new token and rejoins, while the window is open) → ended (the room was deleted, or the end call
  succeeded; on mobile also "the practitioner did not join") or removed (access ended). A denied permission shows how to allow it and lets the user join with
  what remains (audio only, or neither, still hearing and seeing the other side). In-call controls: microphone,
  camera, leave; the practitioner also has "End consultation", which asks to confirm.
- **e2e LiveKit config (owner, on approval).** If one LiveKit configuration cannot serve both clients on the
  host (manual dev) and clients on the compose network (e2e), a separate config for e2e is acceptable; stop and
  report only if that fails too.
- **e2e driver.** A `playwright` compose service (the official image, pinned by tag, profile `e2e`) runs the
  web journey in Chromium with `--use-fake-device-for-media-stream` and `--use-fake-ui-for-media-stream`; the
  patient side is a Go test participant built with the LiveKit Go SDK, publishing a synthetic audio track
  (a generated tone), on the compose network.

Changed on approval (2026-11-05, owner):

- A patient "the practitioner did not join" ended state once the window closes without the practitioner, with
  its text in the copy review. `video_session` records that the patient joined, and the clinic sees it
  (`GET /appointments/{id}/consultation`, shown on the web appointment detail).
- A hard cap: the room is ended and deleted 2 hours after the scheduled end, whether or not anyone is in it
  (`time_limit`), tested.
- A separate LiveKit config for e2e is acceptable if one config cannot serve both.
- Everything else, decisions 2 and 3 included, approved as written.

Changed after package 5 (2026-11-05, owner):

- `encounter` gains the room-based (`app.video_room`) access rule `video_session` has, in a new migration, so
  an `Encounter` whose appointment a revocation cancelled after the practitioner joined finishes, with
  `period.end` set, instead of staying `in-progress`.
- Every removal of a patient from a room is audited (operation, action `E`, `Appointment/<id>`, the removing
  agent, `HOPERAT`).
- The open questions of packages 1–5 are approved as described (PROGRESS "Decisions").
- The web and mobile text is approved with the owner's changes: `COPY.md` is the source.

Changed after package 6 (2026-11-05, owner):

- The status endpoint carries the end or removal `reason` (above); the practitioner keeps read access to the
  status of a room they were in, under the room-based rule. The web app reads it when the patient leaves and
  shows W18a or W18b, including the revoked-before-start case. The join-endpoint probe is removed (from the
  mobile app too, which reads the same `reason`).
- A practitioner's second tab: W40 "The consultation is open in another tab or window.", detected locally from
  LiveKit's duplicate-identity disconnect, with no API change.
- `livekit-client` (web) and `livekit_client` (mobile) log at a level that never prints a token or the
  signalling URL with it to the console.

Changed for SAL-3 (2026-11-08, owner; Jira SAL-3, epic SAL-1 "Live Consult Intelligence"):

- **TURN.** LiveKit's built-in TURN is on in compose: TLS on 443 and UDP on 3478, under `turn.salus.test` (the
  livekit service's alias on the compose network, with a synthetic local certificate in
  `deployments/livekit-turn/`). `make turn-check` proves it: with UDP and LiveKit's ICE/TCP port blocked, a call
  does not connect without TURN and does through TURN over TLS on 443. The TURN ports are not published on the
  host, so a local browser or phone does not use TURN; checking TURN from a real browser and phone waits for a
  staging domain with a public certificate (owner: "automated test only").
- **The agent grant.** `videosession.Service.AgentJoin(sessionID, org)` gives the transcription agent
  (`cmd/agent`, ADR 0006; not a separate `cmd/consult`) a 5-minute token for the session's room: kind `agent`,
  `roomJoin` and `canSubscribe` only, no publishing, no data, no name or metadata. It is called in-process with the
  opaque ids a job carries; there is no HTTP endpoint for it. It needs what a person's join needs: the session open
  and inside its window, the appointment booked, the patient's consent with the clinic active; every outcome is
  audited (agent `transcription_agent`, `HOPERAT`). Its identity is `a_` and the room's random part, so nothing
  new is stored and no migration is needed.
- **Visible, not hidden.** The agent is not a hidden participant. Both apps show "Transcription is on" (W41,
  M36) while a participant of kind `agent` is in the room, and never treat it as the other person (a hidden
  participant cannot be seen by the clients, and the API would have had to be polled; owner's choice).
- **Still one-to-one.** LiveKit does not count an agent toward `max_participants` (tested against the compose
  LiveKit), so rooms stay at two people and a third person is still refused. `Participants` lists people only,
  so the agent never keeps a room "occupied" when the window closes; its webhook events have no effect.
- **Removal.** Whenever the patient is removed from a room (revocation, cancellation, the webhook's re-check), the
  agent is removed with them, after the commit, unaudited (it is not a person).
- **Not decided here (SAL-1's SPEC):** when the agent joins and who starts it; the speech-to-text engine and, if
  hosted, its DPA; the lawful basis for transcribing (§4); and the patient permission text, which says "The
  consultation is not recorded" (COPY.md, ADR 0006) and must be reviewed before any transcript is kept.
  Nothing calls `AgentJoin` yet; `cmd/agent` is still a stub.

Changed for SAL-4 (2026-11-09; owner decisions of that day and agent decisions recorded in
`docs/plans/consult-intelligence.PROGRESS.md`):

- **When the agent joins.** Every participant_joined of a person with access, once the practitioner has joined,
  enqueues `transcription_join` (queue `transcription`, `cmd/agent`) in the webhook's transaction. `AgentJoin`
  also refuses before the practitioner joined (`ErrNotStarted`) and gives the agent the Encounter and the room's
  identities; `AgentAdmitted` asks again once it is connected; `AgentListen` hands over each person's decoded
  audio by role (`docs/plans/consult-intelligence.md`, "SAL-4 as built").
- **Basis (interim).** It joins only while the patient's care consent with the clinic is active and leaves when it
  is withdrawn (owner, 2026-11-09, pending `docs/legal/BRIEF.md` Q16). The permission text "The consultation is not
  recorded" stays to be reviewed before any transcript is kept (SAL-6).

## 3. The LiveKit Go SDK

The settled decision is to use the official server SDK (`github.com/livekit/server-sdk-go/v2`, with
`github.com/livekit/protocol` for the access token and webhook types). Why it passes the "under ~50 lines"
rule: it is the protocol client, not a convenience. The room service is a Twirp API over protobuf messages that
LiveKit versions with the server; the access token is a JWT with LiveKit's grant schema; webhook verification
checks a JWT and a body hash against LiveKit's own claim layout. Writing these by hand means tracking
LiveKit's wire format on every server upgrade. The e2e patient participant (WebRTC) is far beyond 50 lines and
comes from the same SDK. Pinned exact in `go.mod`, versions recorded in `PROGRESS.md`; only
`repository_livekit.go` (and the e2e test) import it, which `depguard` enforces.

## 4. Lawful basis

- The consultation itself, and the fact and times of it, are health data (Art. 9). Care: Art. 9(2)(h), with the
  consent for sharing as in booking SPEC §3. Joining requires the active consent because the practitioner sees
  and hears the patient in the call, which is the pre-care sharing the consent covers. **Needs legal
  confirmation before production**, with booking's split.
- Live audio and video pass through our LiveKit server and are not stored (ADR 0006). TURN relays encrypted
  media only.
- Audit: as in accounts (Art. 5(2), 32).
- The transcription agent (SAL-3) receives the call's audio once it joins. Building the grant processes nothing
  (nothing calls it), but the basis for transcribing a consultation, and whether it needs its own granular
  `Consent`, is **open for SAL-1's SPEC and legal review** before anything calls `AgentJoin`.

## 5. Personal data

| Where | Field | Why it is needed |
| --- | --- | --- |
| `video_session` | `appointment_id`, `organization_id` | Which appointment the room is for, and the clinic, for RLS |
| `video_session` | `room`, `patient_identity`, `practitioner_identity` | Random ids that tie LiveKit's events back to the appointment; LiveKit never learns who they are |
| `video_session` | `patient_joined_at`, `practitioner_joined_at` | Whether each side came, for `fulfilled` versus `noshow` and the start of the `Encounter` |
| `video_session` | `ended_at`, `end_reason` | That the consultation is over, so no new token is issued, and why |
| `video_session` | `closes_at` (added in package 1, PROGRESS.md) | The join window's close (the appointment's end + 30 minutes), copied once: the retention of a session that never ended runs from it, as `salus_retention` may read only `video_session` |
| `video_session` | `removal_reason` (added in package 7.1, PROGRESS.md) | Why the patient was removed from the room (`consent_withdrawn` or `cancelled`), so the status can say why after the end, when a revocation before the start also ends the session `cancelled` |
| `encounter` | `appointment_id`, `patient_id`, `practitioner_role_id`, `organization_id` | The Encounter's `appointment`, `subject`, `participant` and `serviceProvider` |
| `encounter` | `status`, `period_start`, `period_end` | That the consultation took place and when: the care record |
| LiveKit (memory, during the call) | Opaque identities, the room name, live media | To run the call. No name, e-mail, metadata or recording |
| Join response | The other person's display name | So each side knows who they are talking to, from our API under its access rule |
| River jobs | `consultation_window_close`: session id, clinic id; `consultation_access_end`: appointment id, clinic id | To close the room and to remove a participant; opaque ids only |

Logs carry the request id, route, status, duration, opaque ids of our own (appointment, session) and error codes.
Never a token, a room name, a participant identity, a display name or media details. Test data is synthetic, as
in booking.

Retention (defaults, **need a legal decision before production**):

| Table | Rule |
| --- | --- |
| `video_session` | `delete` 30 days after `ended_at` (or after the appointment's end if never ended) |
| `encounter` | Kept with the clinical record (`keep`) |

## 6. Out of scope

Recording, screen sharing, in-call chat, more than two participants, push notifications, in-person
appointments, live transcription beyond the agent's audio and its transcript stream (SAL-4, `docs/plans/consult-intelligence.md`), a practitioner joining from mobile, a patient
joining from the web, break-glass, `Provenance`, FHIR writes to `Encounter`, network-quality indicators beyond
"reconnecting", bandwidth settings, call-quality analytics.

## 7. Up next

- **Live transcription.** A speech-to-text engine for bs/hr/sr, its own SPEC, and DPA review if hosted.
- **Clinical notes in the consultation.** `Observation` and `Provenance` (clinical feature).
- **Web bundle size.** `livekit-client` makes the web bundle 835 kB minified (233 kB gzip); split it out of the
  first page load if a measurement shows the cost (owner, after package 6).
- **Removal audit race.** A removal is audited when LiveKit lists the patient in the room, then the patient is
  removed (package 5.1): a patient who connects between the two is removed without an event of their own, and
  a retry after a failed removal can audit twice. A column on `video_session` recording each removal would close
  it (owner, on review of PR #20).
- **M35's continue button.** The Android explanation before the system prompt gets its own continue button,
  with approved text.
- **Bluetooth headsets on Android 12 and later.** `BLUETOOTH_CONNECT` and approved text for its prompt.
- **A patient's second device.** Text for the device that is replaced when the patient joins from another one,
  like the web's W40.

## 8. Before production

- **Production sizing (open question, SAL-3).** Not measured. To decide with the hosting provider, from a load
  test of LiveKit on the chosen machine (LiveKit's `lk load-test`, figures into `PROGRESS.md`): concurrent
  consultations at the peak (2 people, 2 audio and 2 video tracks each way, plus one subscribe-only agent per
  transcribed call); the share of calls relayed through TURN, which carry each stream twice through the server
  (in and out of the relay); egress bandwidth per call at the video resolution we allow; CPU per call for the
  SFU; whether one node suffices or rooms need LiveKit's Redis-routed multi-node setup (compose already shares
  Redis); and where the agent runs (beside LiveKit or apart), which depends on SAL-1's speech-to-text engine.
- LiveKit's public domain, TLS (`wss://`) on the signalling port, and TURN over TLS (443) for networks that
  block UDP; certificates and the domain chosen with the hosting provider. Locally TURN is on and tested
  (SAL-3); production needs the public domain and certificate, and no `allow_restricted_peer_cidrs`.
- The LiveKit API key and secret in the secret store; the dev pair never leaves compose.
- LiveKit's node IP and ports for the deployed network (compose uses the container network).
- Retention of `video_session` and `encounter`, and the lawful basis (§4): legal decision.
- The new web and mobile text, and the iOS and Android permission wording, localised (bs/hr/sr) with the rest
  of the patient app.
- LiveKit's log level `warn` in production, with short log retention; `info` stays in dev. At `info` LiveKit
  logs each client's IP address and port, SDK and browser, and a client IP is personal data (owner, on review
  of PR #20).
- The mobile app runs on a real Android device and an iOS build before any pilot. It has not run on a device
  yet. On iOS, `permission_handler` may report the camera and microphone as denied without its build flags
  (the project uses Swift Package Manager and has no Podfile); that is checked on the iOS build (owner, on
  review of PR #20).
- Chrome's own network log can show the signalling URL with the 5-minute token on a failed connection; no log
  level reaches it. Accepted by the owner on review of PR #20.

## 9. Done when

All eight packages are committed on `feat/consultation`; `make lint`, `make test`, `make archcheck`, `make
cover` (every package at 80% or more), `make fhir-validate` and `make e2e` pass locally and in CI; `make
gen-clients` leaves no diff; the web and mobile checks pass; `docs/privacy/` describes what exists;
`PROGRESS.md` records every package with its failing tests, the SDK versions and the token latency.
