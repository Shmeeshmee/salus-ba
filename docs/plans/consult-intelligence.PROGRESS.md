# Consultation intelligence: progress

One section per ticket of docs/plans/consult-intelligence.md.

## C (SAL-6): event bus, persistence and the WebSocket gateway

2026-11-09. Branch `sal-6-bus`.

**Built.**

- `encounter_organization(uuid)` (migration 00033): the clinic of an encounter for the stream consumer, which has
  only the encounter's id; `encounter.Service.Lookup` (system path) and `Access` (a caller's attendance, unaudited).
- `access.ConsultationRecord`: the appointment's practitioner with an active consent.
- `transcript.Service`: `Store` (final segments only, once, not after a revocation), `Export`, `Subscribe`, `Next`
  (access checked again after every read of the stream). `suggestion.Service.Record`.
- `pkg/redis.Group`, a consumer group over every stream matching a pattern; `worker.ConsultStream` in `cmd/worker`
  (which now requires `REDIS_URL`).
- `GET /encounters/{id}/events` (WebSocket) and `GET /encounters/{id}/transcript`, in `api/openapi.yaml` with the
  generated clients (`ConsultFrame`, `CursorMessage`, `Cursor`, `Transcript`); the web dev server proxies
  `/encounters` with WebSocket upgrades.

**Wire protocol for SAL-9.** Connect to `/encounters/<encounter_id>/events` (same origin, session cookie). Frames
alternate: `{"type":"cursor","version":1,"payload":{"id":"<stream id>"}}`, then the entry's `ConsultMessage`
exactly as its producer wrote it. Keep a cursor's id once the message after it has arrived; on reconnect pass
`?after=<id>`. Close codes: 4403 (`consent_required` or `forbidden`), 4404 (`not_found`), 1011 (server error). The
encounter id comes from `GET /fhir/R4/Encounter?appointment=<appointment_id>`.

**Shared contract for the producers (SAL-4, SAL-7, SAL-8).** `XADD consult:<encounter_id>:events * envelope
<wire envelope bytes>` (`transcript.StreamKey`, `transcript.StreamField`).

**Decisions (agent decisions, for the owner to review).** In the plan, "Decisions taken for C". Also:

- No `after` replays every entry still on the stream, so a reload rebuilds the live view.
- The consumer writes suggestions (the owner's instruction for SAL-6); SAL-7 then only puts them on the stream.
  A suggestion's later event changes only its status; a re-delivered old event can set an older status back
  (only after a failure and a 30 s reclaim), which SAL-7 may want to guard with a version.
- The export is refused after a revocation, also for a consultation that started before it (unlike the
  questionnaire response's D9 exception). Whether the practitioner keeps their transcript as the care record
  after a revocation is a question for the owner and legal.
- A client resuming from a cursor the stream has trimmed (`MAXLEN ~`) gets what is left; the stored transcript
  covers the final segments.
- Each open WebSocket holds one Redis connection while it waits (XREAD BLOCK): go-redis's default pool
  (10 per CPU) bounds the concurrent streams per api process; to be raised when measured.

**Measurements** (`make bench-consult-stream`, compose stack on the dev machine, 2026-11-09):

| What | Result |
| --- | --- |
| Live, XADD to the envelope's frame through api.New (access check included), 500 entries | p50 3.8 ms, p95 5.5 ms, p99 6.2 ms, max 8.4 ms |
| Replay of 2,500 entries on a reconnect (25 pages, each checked) | 120 ms |
| One discovery sweep (`SCAN TYPE stream`) over 100,050 keys, 50 streams | 7 to 9 ms, run once a second |
| Revocation to socket closed, idle stream | under 2 s (tested; bound by `RecheckEvery`, 1 s) |

**Left.** Drafts (SAL-8). The patient's copy (the export feature). Producers' `MAXLEN` and expiry (SAL-4).

## SAL-4 (A): audio ingest from LiveKit and the streaming STT adapter

Branch `sal-4-stt`, 2026-11-09.

**Built.** The transcription agent in `cmd/agent` (plan, "SAL-4 as built"): `transcription_join` from the webhook,
`videosession.AgentListen` (grant, connection, admission asked again once connected, Opus and G.711 decoded to
16 kHz), `transcript.Service` (one recognition per microphone in the speaker's locale, partial and final events
on the encounter's stream), `internal/worker.NewAgent`, `pkg/stt/azure`, `config.LoadSTT`. Fake by default.

**Tests seen failing first.** Each behaviour's test failed before its code (compile errors for the new ports and
types; then, for example: `agent joins [], want [<session> <clinic>]` for the webhook's enqueue;
`the connection logged 6020 bytes, want none` for the SDK's logger, which needed both the connection's logger and
the package logger silenced; `LoadSTT is called by [], want [cmd/agent/main.go]`).
A flaky test found on the way: `TestStop_LeavesEveryRoom` failed 224 of 300 runs: on the call's end a
recognition returned before its results goroutine had ended, so `Stop` could return with a recognition still
open. Fixed by waiting for it; 0 of 200 runs fail since.

**Integration with synthetic audio.** `TestListen_HandsOverEachMicrophoneDecodedTo16kHzUntilTheAgentIsRemoved`:
on the compose LiveKit two publishers send a generated 440 Hz tone (Opus, G.711); the agent gets 20 ms frames of
16 kHz PCM whose pitch is 440 Hz (±40), and its tracks end once it is removed.
`TestTranscriptionJoin_JoinsTheRoomAndLeavesWhenTheConsentIsRevoked` (worker, real database and LiveKit): the job
puts the agent in the room, the revocation's job removes it, the next join is cancelled. `make e2e`: in the
revocation journey the patient's tone reaches the agent through the whole stack and the fake engine's partial and
final captions are on the encounter's stream; after the revocation the practitioner's room has nobody left, the
agent included.

**Partial caption latency (target under 1 s).** `make bench-stt`, i7-14700HX, compose Postgres, Redis, LiveKit:

| Hop | p50 | p95 | p99 |
| --- | --- | --- | --- |
| Pipeline, fake engine: frame handed to the agent to the partial read back from Redis (XREAD BLOCK), 1,000 frames, 3 runs | 0.04 ms | 0.11 to 0.13 ms | 0.15 to 0.24 ms |
| Pipeline, Azure adapter against its httptest server (TLS WebSocket, a hypothesis per audio message), 1,000 frames, 3 runs | 0.11 ms | 0.19 to 0.39 ms | 0.32 to 0.51 ms |
| LiveKit: a tone burst's first Opus frame written by a publisher to the agent's first decoded frame of it, 100 bursts, 3 runs | 0.59 to 0.80 ms | 1.1 to 1.5 ms | 1.5 to 1.7 ms |

Our part is under 3 ms at p99; the rest of the 1 s budget is the engine's own time to its first hypothesis, the
20 ms frame, and the network to an EU region, which only a run against real Azure can measure (blocked, below).

**Decisions (2026-11-09, agent decision, for the owner to review).**
- The agent joins on every person's join once the practitioner has joined (not on a token request), and
  `AgentJoin` now also refuses before the practitioner joined (`ErrNotStarted`): the Encounter must exist.
- The practitioner's speech language is the clinic's default language; the patient's their account's language
  (chosen, else detected), else the clinic's.
- Locales per language as in the plan; `en` as `en-GB`; Serbian in both scripts as `sr-RS`.
- Stream field `envelope`, `MAXLEN ~ 10000`, TTL 30 minutes after the last entry (shared contract: added to
  `transcript` for SAL-6 to read).
- When the agent is out of the room, recognitions end at once and results still pending are dropped (nothing
  is processed after a removal); when one person's track ends, its recognition is closed and its last results
  are published.
- A failed recognition is opened again after a second; the audio meanwhile is not recognised.
- Opus decoder: `github.com/tphakala/go-opus` v1.4.0 (BSD-3, pure Go and assembly, one dependency of its own);
  WebSocket: `gorilla/websocket`, already in the module through LiveKit's SDK. Neither is under 50 lines of
  standard library.

**Blocked.** The accuracy benchmark across bs, de, ar and tr (to confirm Azure and `ar-SA`), and the latency
against real Azure: both need real credentials and the DPA. Azure's protocol is implemented from the Speech
SDK's source and tested against our own server only; the first real run may need adjustments.

**Checks.** `make lint`, `make test`, `make archcheck`, `make cover`, `make e2e`, `make web-typecheck`,
`make mobile-analyze`: see the PR.

## SAL-9: the practitioner's call page (captions, transcript, suggestions, documents)

### 2026-11-09: built against the mocked event stream

What was built (web only; no Go, API or migration change):

- `web/src/consult.ts`: the two seams and the state. `LiveSource.subscribe(appointmentId, onMessage)` delivers the
  P0 envelopes (`ConsultMessage`: `transcript`, `suggestion`, `draft`, version 1, the generated types of
  `api/openapi.yaml`); `ConsultActions` holds `setSuggestionStatus`, `templates`, `generate` and `sign`.
  `decodeMessage` reads one frame and drops anything that is not a known type at version 1. `apply` folds a message
  into the state idempotently (segments by `segment_id`, ordered by `ts_start`; suggestions by `id`; drafts by
  `draft_id`), so a replay changes nothing. `useConsult` subscribes while the call is on (joined or reconnecting).
- `web/src/test/fakeConsult.ts`: the in-memory mock of both seams, and builders of the three envelopes with
  placeholder text.
- `web/src/ConsultPanels.tsx`: the captions, and the panel beside the video with three tabs (Transcript,
  Suggestions with the number not dismissed, Documents). `web/src/Documents.tsx`: template → Generate → review
  (sources, missing-field flags) → edit → Sign, and Regenerate.
- `web/src/CallScreen.tsx`: an optional `consult` prop (`{ live, actions }`); the captions go into the stage, the
  panel beside it.
- Copy: `docs/consult-intelligence/COPY.md` (W1 to W23), the English in `web/src/i18n/consultation/en.json` under
  `consultation.live.*`, drafted in bs, hr, sr-Latn and sr-Cyrl (`docs/i18n/TRANSLATIONS.md`).
- Tests: `consult.test.ts` (14), `ConsultPanels.test.tsx` (19), `Documents.test.tsx` (14), and 3 new in
  `CallScreen.test.tsx`. The layout was also checked in a browser (Playwright, 1280 and 820 px wide, a throwaway
  harness on the fakes, not committed): at 1280 px the panel sits to the right of the video, at 820 px below it,
  and the captions keep to the video's lower band, left of the own camera.

No performance claim is made, so nothing was measured.

### Decisions (2026-11-09, agent decision, for the owner to review)

- **"Nothing covers the video".** The panel is beside the video (below it under 960 px), never over it. The
  captions are the one live element on the video: a band along its lower edge, at most two lines (the latest two
  of a long sentence), 12 px from the edges and clear of the practitioner's own camera tile in the corner, so the
  patient's face stays clear. The tooltip with the original appears just above the band only while the
  practitioner hovers or focuses the caption.
- **What is captioned.** Only the patient's latest segment, in the practitioner's language (the translation; the
  original until a translation arrives), and only while the patient is in the room. The practitioner's own words
  are not captioned on their side. A final caption goes after 6 seconds without a new one; a partial one stays
  until it is revised.
- **Keyboard and screen readers.** The caption is focusable; its original is the tooltip, shown on focus as on
  hover, and is the caption's accessible description. The caption band is not a live region (a revised partial
  would be read out again and again); the transcript is the full text, in reading order.
- **One panel, three tabs**, rather than three columns: the practitioner sees one thing at a time (CLAUDE.md
  "UI direction"). A source link (W12) switches to the transcript, highlights the cited lines and brings the first
  into view (smoothly, or at once under `prefers-reduced-motion`). Every tab stays mounted, so the transcript keeps
  its scroll.
- **The transcript** shows partial lines greyed and revised in place; each line's text in the practitioner's
  language comes first (the translation of a patient line, the original of a practitioner line), the other below.
  Times are the practitioner's browser time zone, hours and minutes. It follows the latest line while the
  practitioner is within 24 px of the bottom; otherwise it does not move and offers "Show the latest".
- **Suggestions**: pinned first, then the newest first; a dismissed one leaves the list, with no undo; the
  confidence is not shown. A pin or a dismissal shows at once and is put back, with W14, if the action fails. The
  line that a suggestion is not a diagnosis (W9) is always at the top and is tagged `legal`.
- **Documents**: edits stay on the page until Sign, which sends every field's value; a field the practitioner has
  typed in loses its source link and its flag. Sign always asks to confirm (and says how many fields are empty);
  Regenerate asks only when it would replace edits. A draft `entered-in-error` is not shown.
- **No live source wired (the real app today)**: no panel and no captions at all; the call page is as before.
  `web/src/App.tsx` does not pass `consult` yet.
- **The strings** live in the existing `consultation` catalog area (keys `live.*`), not a new area, so the i18n
  plan's list of areas is unchanged.

### What the real backend must provide (for SAL-6, SAL-8 and SAL-10)

Wiring is `App.tsx` passing `consult={{ live, actions }}` to `CallScreen`, with `live` and `actions` built on the
endpoints below, each built once (not per render: the page resubscribes when `live` changes and reloads the
templates when `actions` changes). Nothing in `api/openapi.yaml` was added for them; SAL-6 and SAL-8 own those contracts.

- **Live source (SAL-6).** Keyed by the appointment id (the page knows the appointment, not the encounter); the
  practitioner of that appointment only. Each frame is one envelope as text, the stream entry's bytes unchanged;
  the page decodes it with `decodeMessage`. On every (re)connect it should send the consultation's messages from
  the start (or at least every final segment, the latest status of every suggestion and the latest version of
  every draft) before the live ones: the transcript, the source links and the drafts are only as complete as what
  the page has received. Duplicates are harmless. A suggestion's status change and every draft change are pushed
  as events too (the page also applies its own changes at once).
- **As SAL-6 landed (merged first, #35):** the gateway is `GET /encounters/<encounter_id>/events`, keyed by the
  encounter, not the appointment: the wiring resolves it with `GET /fhir/R4/Encounter?appointment=<id>`. Its frames
  alternate a `cursor` frame and the message; `decodeMessage` drops the cursor frames as an unknown type, so the
  wiring keeps the last cursor id itself and reconnects with `?after=<id>`. A connect without `after` replays the
  whole retained stream, as the page expects. (Orchestrator note at the merge, 2026-11-09.)
- **`setSuggestionStatus(id, status)`** (SAL-7/SAL-6): sets `shown`, `pinned` or `dismissed`; any 2xx is enough,
  a failure is shown as "could not be updated".
- **`templates()`** (SAL-8): the clinic's templates, each `id`, `title` and `fields` (`key`, `label`) in order.
- **`generate(appointmentId, templateId)`** (SAL-8): starts drafting (or redrafting) that template from the
  transcript and answers at once; the draft arrives on the live source as `draft` events. A redraft may reuse the
  draft's id or create a new one; the page shows the latest draft of the template.
- **`sign(draftId, fields)`** (SAL-8): takes `[{key, value}]` for every field as the practitioner left them, saves
  them (a value that differs from the drafted one has no sources any more), sets the status `final`, and answers
  with the signed `DraftEvent`. A draft that is not `preliminary` is refused.
- Every one of these reads or writes patient data, so each writes its `AuditEvent` server side. The page keeps
  the transcript, suggestions and drafts in memory only (no browser storage) and logs nothing.

### Left for later

- **Follow-up ticket to create:** the patient's consent notice for transcription at the start of the call (web and
  mobile), out of SAL-9's scope by the owner's decision of 2026-11-09.
- Native-speaker review of the drafted bs, hr and sr strings, and the lawyer's review of W9 in each language.
- An undo for a dismissed suggestion, and showing a suggestion's confidence, if the owner wants either.
- "Generating…" can clear early when a regenerate replays an unchanged draft (Copilot, PR #34, declined there):
  the page then shows the previous draft until the new one arrives. The fix depends on how SAL-8 versions a
  redraft over SAL-6's replay; for SAL-8/SAL-10.

## D (SAL-7): the live suggestions worker, and dismissing and pinning

Branch `sal-7-suggest`, 2026-11-09.

**Built.**

- `suggestion.Assistant` (`service_assist.go`, `service_assist_run.go`): the rules of the live suggestions (plan,
  "Decisions taken for D"): the window, the pace, the validation of the model's output, the repeats and the
  throttle; `Run` follows every stream whose lease the member holds. `suggestion.NewRedisLive` (streams, XRANGE,
  the lease) and `NewRedisPublisher`.
- `worker.Suggestions` in `cmd/worker`, beside the stream consumer, on the `llmtest` fake (no suggestions).
- `suggestion.Service.SetStatus` (the practitioner's path, under `transcript.Service.Allowed`, audited, then
  published) and `suggestion.CommandHandler`; the gateway (`transcript.HTTPHandler`) hands the client's text
  frames to `transcript.Commands` and serialises its writes.
- `suggestion.Event.Origin` (`ai`) and the 300-character text bound; `llm.SuggestRequest.Shown`.
- `api/openapi.yaml`: `SuggestionEvent.origin`, `ConsultCommand`, `SuggestionStatusMessage`, `SuggestionStatus`,
  `CommandErrorMessage`, `CommandError` (in `ConsultFrame`), with the generated clients.
- The stream consumer's `Record` stores a suggestion once and changes nothing after (SAL-6's note: an old event
  delivered again set an older status back).

**For SAL-9 / the web wiring.** `setSuggestionStatus(id, status)` is a text frame on the open event stream:
`{"type":"suggestion_status","version":1,"payload":{"id":"<suggestion id>","status":"dismissed"}}` (or `pinned`, or
`shown` to unpin). Success: the suggestion with its new status arrives on the stream, to every open view. Failure:
`{"type":"command_error","version":1,"payload":{"command":"suggestion_status","id":"<id>","error":"<code>"}}`, codes
in `CommandError`. `decodeMessage` drops the `command_error` frame as an unknown type today; the wiring needs to read
it to put a status back (W14).

**Tests seen failing first.** Compile errors for each new type and method first (`suggestion.OriginAI`,
`Service.SetStatus`, `Assistant`, `transcript.Commands`, `transcript.Service.Allowed`, `NewRedisLive`,
`worker.NewSuggestions`), then, for example: `stored {... Status:pinned ...}, want the first event as it was` and two
audit events for one (the consumer's insert-only `Record`); `audited 3, want 2` in the consumer's database test;
`told the model [...] were shown` in a different order than the fake's (fixed in the test: the order is not a rule);
the contract tests on the missing schemas.

**Measurements** (`make bench-suggest`, i7-14700HX, compose Postgres and Redis, the stream consumer running, the
model answering at once with six suggestions of which three are new): the worker's own cost of one window, from
reading the stream (a fresh session reading 330 entries: 30 final lines after ten partials each) to the third
suggestion published, through the encounter lookup, the consent, the audited read of what was shown, the clinic's
language, three audited stores and three publishes. 200 windows a run, 3 runs:

| p50 | p95 | p99 | max |
| --- | --- | --- | --- |
| 23.1 to 23.5 ms | 31.5 to 33.5 ms | 34.8 to 38.9 ms | 75 to 90 ms |

At most one run per encounter every 5 s: under 0.5% of the time per consultation (most of it waiting on Postgres
and Redis), besides the model's own.
In steady state a step reads only the entries since the last one. Not optimised: no claim needs it yet.

**Decisions (agent decisions, for the owner to review).** In the plan, "Decisions taken for D". Also:

- The window's text is each line's original, not its translation: what was said, in the language it was said in.
- A run that finds the consent withdrawn sends nothing and waits for new lines (none come once the agent has left).
- The `command_error` code and the suggestion's id go back to the client; nothing else of the failure.

**Left.** The engine (SAL-5's adapter, once merged: the composition root switches to its config). The web wiring of
`setSuggestionStatus` and of `command_error` (SAL-9's page, a later PR). An undo for a dismissal (SAL-9's list).
