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
- ~~"Generating…" can clear early when a regenerate replays an unchanged draft (Copilot, PR #34)~~: fixed in the
  wiring follow-up, below.

### 2026-11-09: wired to the real backend (branch `sal-9-wire`, the follow-up to #34)

What was built (web and the e2e journey; no Go, API or migration change):

- `web/src/live.ts`, `consultSocket({ fhir })`: the live source and the actions of one call page, sharing its one
  socket. The encounter comes from `GET /fhir/R4/Encounter?appointment=<id>` (it starts with the practitioner's
  join, so an empty search is tried again); then `GET /encounters/<id>/events` on the page's own origin. A cursor
  is kept once the message after it has arrived (a `command_error` frame does not count) and a reconnect passes it
  as `after`, so nothing is lost or repeated. Close codes 4403 `consent_required` / `forbidden` and 4404
  `not_found` end the source for good; anything else (1006, 1011, a refused handshake) is retried after 1, 2, 5,
  then every 10 s, back to 1 s once a frame arrives.
- Dismiss, pin and unpin (SAL-7): `setSuggestionStatus` sends `suggestion_status` on the open socket and is done
  when the suggestion comes back with that status; its `command_error`, or the socket closing first, fails it, and
  the page puts the status back with W14 (as built in #34). No socket open: it fails at once. `decodeMessage` now
  drops a suggestion whose `origin` is not `ai`, so the page shows only what it labels as a suggestion (W9, W10);
  that no approved string names AI is left open for the owner in COPY.md.
- `LiveSource.subscribe` takes an `onEnd(end)`; `useConsult` keeps the end. `App.tsx` builds the socket once per
  call page and passes it to `CallScreen`.
- `translation_failed` (SAL-5): the page shows the original of such a line, in its language, as it shows a line
  awaiting its translation (tested); no new string.
- Documents (SAL-8, after main with #39 was merged in): `templates(language)` is `GET /document-templates`, each
  template's title and labels in the UI language where it is written or translated in it, in its own otherwise;
  `generate` is `POST /encounters/<id>/drafts` with that language and answers with the stored draft; `sign` is
  `POST /drafts/<id>/sign` with `If-Match: W/"<version_id>"` of the draft as the page read it; a `422
  required_missing` fails as `SignRefused` with its `keys`. On subscribing, once the encounter is found,
  `GET /encounters/<id>/drafts` delivers the drafts the (capped) stream may no longer hold. The Vite dev server
  proxies `/document-templates` and `/drafts`.
- **Required fields.** A field the template marks required and the practitioner left empty blocks the sign: Sign
  then shows W24 (new, pending the owner, COPY.md) instead of the confirmation and marks the field; the same when
  the server refuses with `required_missing`. Optional empty fields are signed empty after the confirmation (W21).
- **"Generating…" (Copilot on #34), fixed at its cause.** It cleared on the first draft event of the template that
  was not the draft shown before, so a replay of the previous version (or any new object for it) cleared it. Now
  it ends only with the generation's own answer, the stored draft, which the page folds in; and `apply` never
  replaces a draft with an older `version_id`, so a late or replayed event of the previous version cannot bring
  it back after the redraft. Tested first (`keeps generating through an event of the previous draft...`, `never
  replaces a draft with an older version`); the guard's removal fails both.
- e2e: `test/e2e/web/consultation.spec.ts` now waits for the patient's placeholder line of the fake speech engine
  in the transcript beside the video: the tone in the room, the agent, SAL-4's fake engine, the stream, SAL-6's
  gateway and the page, through the real stack. Then, on a synthetic template the Go test creates (one required
  field, one optional), it generates a draft with the fake model (every field missing), is stopped by W24, fills
  the required field, signs and sees the document final. It failed first on the missing proxy of
  `/document-templates` (no template picker). The fake model suggests nothing, so suggestions are not proved end
  to end.

Tests seen failing first: `live.test.ts` (14, the module missing; then mutations: a cursor kept on its own frame
fails the reconnect and command_error tests, a reconnect after an end fails the three end tests), four new in
`CallScreen.test.tsx` (the panel's end state, a rejoin after an end), and the e2e journey (`element(s) not found`
for the transcript line before `App.tsx` wired the socket). The `translation_failed` tests passed at once: the
behaviour was already there.

No performance claim is made, so nothing was measured.

**Decisions (2026-11-09, agent decision, for the owner to review).**

- **An end drops what the page has.** On 4403 or 4404 the page clears the transcript, the suggestions, the drafts
  and the captions it holds in memory, and the panel shows one line instead of its tabs: for `consent_required` the
  approved `consultation.peerWithdrewConsent` ("The patient withdrew consent for this clinic, so the consultation
  has ended."), for `forbidden` and `not_found` `consultation.notAvailable` ("This consultation is not
  available."). Existing strings, no new one (COPY.md). The server already refuses the stream and the export after
  a revocation; showing what was received before it is the open question SAL-6 recorded for the owner and legal,
  and the page takes the stricter side until it is answered. A rejoin subscribes again from nothing.
- **A refused handshake is retried.** A browser cannot tell a 403 handshake from a dropped connection (both close
  1006), so the page retries, at most every 10 s; each attempt is audited server side as a refused subscription.
  The open socket's 4403 (a revocation during the call) is the common case and stops it.
- **No timeout on a command**: it is answered by its event or its error, or failed when the socket closes.
- **The draft's language** is the UI language when the template is written or translated in it, else the
  template's own; the labels the page shows are in that same language.
- **A refused sign other than `required_missing`** (`version_mismatch` after a redraft, `not_preliminary`,
  `invalid_values`) shows W23 "could not be signed"; the newer draft comes on the stream.

## B (SAL-5): translation of the final segments, and the Anthropic Claude API adapter

Branch `sal-5-translate`, 2026-11-09.

**Built.**

- `transcript.Live` translates each final segment in the background (`service_translate.go`) and publishes it
  once, with its translation or `translation_failed`, so `transcript.Service.Store` (SAL-6) stores it once; partial
  segments are published at once, untranslated. Budget 1.5 s (`TranslateBudget`), two attempts within it; after a
  removal nothing pending is published.
- `terminology.Service.Glossary` (the reference domain's service, not its repository) and its query on the index
  `term_label_language_folded` (migration 00034). `llm.TranslateRequest.Glossary` (`llm.Term`), additive; the
  `llmtest.Translator` records it and can fail (`Failures`), wait (`Delay`) or keep nothing (`DiscardCalls`, the
  agent's default engine). The `Suggester` and `Drafter` fakes take `DiscardCalls` too: a fake wired as a
  long-running process's default engine (SAL-7, SAL-8) is built with it, so it keeps no unbounded state (tested
  for the agent's: `TestNewTranslator_TheFakeKeepsNoCalls`).
- Shared contract, the smallest: `translation_failed`, optional on TranscriptEvent (Go `omitempty`,
  `api/openapi.yaml`, the generated TypeScript and Dart clients, the contract test) and stored
  (`transcript_segment.translation_failed`, migration 00034). Nobody who sends or reads the event had to change.
- `pkg/llm/anthropic`: `Translator`, `Suggester` and `Drafter` on the Messages API (`anthropic-version: 2026-10-21`,
  structured outputs, fixed schemas, the answers validated). `config.LoadLLM` (`LLM_ENGINE=fake|anthropic`,
  `ANTHROPIC_API_KEY`, `LLM_MODEL`), called by `cmd/agent`; `worker.NewAgent` takes the translator. Compose sets
  `LLM_ENGINE: fake`.
- Golden files: the glossary for a synthetic German, Arabic and Turkish line into Bosnian
  (`internal/terminology/testdata/glossary_{de,ar,tr}_bs.golden.json`, from synthetic terms on a real database) and
  the request the adapter sends for each (`pkg/llm/anthropic/testdata/translate_{de,ar,tr}_bs.golden.json`), plus
  the suggestion and draft requests (`suggest.golden.json`, `draft.golden.json`).

**Tests seen failing first.** Compile errors for each new port and type; then, for example:
`TestLiveEvents_MatchTheirGoTypes: properties [...] want the Go fields [... translation_failed ...]` for the
contract; the stored round trip without the flag; `LoadLLM is called by [], want cmd/agent/main.go` until the
agent loaded it; the adapter's golden files before they were written. Mutations checked: one attempt instead of two
fails the retry test (`one failure: ... TranslationFailed:true`); a three-letter stem or 400 words fails the
glossary's candidate tests.

**Measurements** (`make bench-translate`, i7-14700HX, compose Postgres and Redis, 2026-11-09):

| What | p50 | p95 | p99 |
| --- | --- | --- | --- |
| Pipeline, fake translator: final frame handed to the agent to the translated final read back from Redis (XREAD BLOCK), a fake glossary (archcheck: a domain's test builds no other domain's repository), 1,000 lines, 3 runs | 0.09 to 0.10 ms | 0.15 to 0.17 ms | 0.20 to 0.23 ms |
| Pipeline, Anthropic adapter against its httptest server (TLS, the request's JSON, an answer at once), same | 0.14 to 0.23 ms | 0.32 to 0.43 ms | 0.46 to 0.59 ms |
| Glossary alone, a 20-word German line with two terms, on synthetic terms of the seed's size (5,300 terms, 88,330 labels), 2,000 lines, 3 runs, one at a time | 1.10 to 1.79 ms | 1.56 to 2.42 ms | 2.08 to 2.96 ms |
| Same, 10 lines at a time | 1.38 to 1.93 ms | 2.59 to 4.01 ms | 15.5 to 18.2 ms |

Our part of a translated line, the pipeline plus the glossary, is under 4 ms at p99 one at a time and under 20 ms with
ten consultations' lines at once; with SAL-4's partial path (under 3 ms) and SAL-6's gateway (p99 6.2 ms) the stack adds about 30 ms at worst
to the 2 s target. The rest is the model's own time to answer, which only a run with real credentials can measure
(blocked: the DPA and the region, `processors.md`): its time per line for the chosen model and effort (the default
is `claude-opus-5-5` at effort `low`, which always thinks; `LLM_MODEL` can name a faster one, the owner's choice),
the first request's schema compilation (once per 24 hours per schema), the network to Anthropic's region, how often
it refuses medical content, and the translations' quality per language (a native speaker's review of the German,
Arabic and Turkish into Bosnian lines). If the model's p95 is over the 1.5 s budget, lines go out flagged rather
than late: the budget is one constant.

**Decisions (2026-11-09, agent decision, for the owner to review).** In the plan, "SAL-5 (B): translation, as
built". Also:

- The practitioner's lines are translated into the patient's language: the plan's goal ("each side sees what the
  other said in their own language") and the bilingual transcript in both directions. The patient's app does not
  show them yet.
- The translator is given the speakers' language tags (`tr`, `sr-Cyrl`), not the recognition locales, and the
  speaker's configured language, not one the engine may detect.
- A failed or slow translation is not translated later: the segment is stored once, flagged. A later re-translation
  of flagged segments would be a new ticket.
- The page (SAL-9) shows the original of a failed line as it shows one awaiting its translation; marking it
  ("translation unavailable") needs a string the owner approves first (Copy), so it is left for a follow-up.
- Adapter: plain `net/http` with the injected client, no SDK: one endpoint and three JSON shapes, the client in about 200 lines (`anthropic.go`);
  the SDK would add a dependency tree for that (CLAUDE.md "Do not add a dependency..."). No beta header and no
  server-side refusal fallback (a beta; a refusal is a failed call, so the line goes out flagged). Effort `low` for
  translation (the live path), `medium` for suggestions and drafting. The suggestion schema allows the plan's four
  kinds; a suggestion that is not valid (unknown kind, empty, confidence outside 0 to 1, a source not in the
  transcript) is dropped; a draft field without a source from the transcript is missing. The Suggester and Drafter
  request types are unchanged.
- The glossary includes draft terms (none is reviewed yet) and gives the target's preferred label; a term without a
  label in the target language is left out ("gaps stay gaps").

**Blocked.** Enabling the adapter: the DPA, zero data retention, and an EU region the API does not offer
(`processors.md`; the gate is the owner's). The model's latency and the translations' quality, as above.

**Checks.** `make lint`, `make test`, `make archcheck`, `make cover`, `make e2e`, `make web-typecheck`,
`make mobile-analyze`, `make gen-clients` and `make sqlc` with no diff: see the PR.

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

**After SAL-5's merge** (main merged in, 2026-11-09). `cmd/worker` takes its model from `config.LoadLLM`: the fake
keeping no calls by default (`DiscardCalls`, tested), Anthropic's adapter with its key; the compose worker sets
`LLM_ENGINE: fake`. `pkg/llm/anthropic`'s Suggest sends `Shown` in a `<shown>` block, escaped, marked as data in the
system prompt (golden request updated, the test seen failing first on the golden's difference). A race in the
worker's database test, seen once after the merge (the stream was read between a suggestion's store and its
publish), fixed by waiting for the published events; 5 runs in a row pass.

**Left.** The web wiring of
`setSuggestionStatus` and of `command_error` (SAL-9's page, a later PR). An undo for a dismissal (SAL-9's list).

## E (SAL-8): document templates and the drafting service

Branch `sal-8-draft`, 2026-11-09.

**Built.** Decisions in the plan, "E (SAL-8)".

- Migration 00036: a template's language, status and LOINC type; `document_template_translation`; a draft's drafted
  texts and its signature (DocumentReference, Provenance, the PDF's size, SHA-1 and wrapped key); one open draft per
  encounter and template.
- `composition.Service`: templates (`Templates`, `CreateTemplate`, `UpdateTemplate`, `SeedTemplate`), `Generate`,
  `Sign`, `Drafts`, `Composition`, `DocumentReference`, `Provenance`, `PDF`. `pkg/pdfdoc` renders the PDF.
- `llm.FieldSpec` gains `Type`, `Instruction`, `Required` (additive; the fake records its own copy of a request).
- JSON API (`api/openapi.yaml`, clients regenerated): `GET/POST /document-templates`, `PUT /document-templates/{id}`,
  `POST/GET /encounters/{id}/drafts`, `POST /drafts/{id}/sign`, `GET /document-references/{id}/content`;
  `version_id` on `DraftEvent`. FHIR reads of `Composition`, `DocumentReference`, `Provenance` (CapabilityStatement,
  golden files validated with `make fhir-validate`).
- Wired in `api.New` with the drafter `cmd/api` takes from `config.LoadLLM` (after main, with SAL-5, was merged in:
  the fake `llmtest.Drafter{DiscardCalls: true}` by default; SAL-5's Anthropic adapter now sends each field's type,
  required flag and instruction and keeps only values of the field's type, golden request updated); `make document-templates-seed clinic=<id>`; `make bench-drafting`.
- The attachments' purger keeps object keys that are no binary id (the PDFs are under `composition/`).

**Tests seen failing first.** Each behaviour's test failed before its code: most as compile errors for the new
types and methods; the migration's tests on the missing columns, table and index (`a second open draft: <nil>, want
unique_violation`); the purger's on the PDF's key (`Purge() = 4, want 3`); the contract test on `version_id` and the
new schemas (`no schema DraftValue`); the CapabilityStatement's on the three new resources; Arabic templates
(`an Arabic translation: ValidateTemplate() = <nil>`). A coverage test of `pkg/pdfdoc` found that gopdf leaves out a letter
its font lacks without an error (`meta: Render() = nil, want an error`): `pdfdoc.Supported` now checks every text, and
the template, the drafted values and the signed values are held to it (a sign with such a letter was a 500 before:
`Sign() = composition: sign: pdfdoc: a letter the fonts do not have, want composition: invalid values`). The existing migration test that inserted a second open draft
for the same encounter and template now inserts a final one (the new rule), keeping what it checks: the clinic's
write scope. The golden FHIR files failed the HL7 validator once (`cmp-1` on a section with only an `emptyReason`)
and were fixed by giving such a section a dash as its narrative.

**Measurements** (`make bench-drafting`, compose Postgres, Redis and Garage on the dev machine, 2026-11-09):

| What | Result |
| --- | --- |
| `POST /encounters/{id}/drafts` through api.New, 360 stored segments, fake model answering at once, 50 runs | p50 7.4 to 8.2 ms, p95 10.4 to 12.4 ms, max 16 ms (two runs) |
| `POST /drafts/{id}/sign` (PDF rendered, encrypted, put to Garage, final transaction), 50 runs | p50 18.2 to 18.3 ms, p95 22.0 to 22.9 ms, max 24 ms (two runs) |
| The request to the model for a 30-minute consultation (360 segments of about 130 characters, 7 fields) | 74 KB as JSON, 46,080 characters of transcript text |

`pdfdoc.Supported` (`go test -bench Supported ./pkg/pdfdoc/`, both fonts parsed per call, a 500-character value): 22 µs
per call, 3 runs, so about 0.15 ms for a seven-field draft. Kept as it is rather than parsing the fonts once into
package state (PR #39 review).

Our path is about 10 ms of the 15 s target. What only a run with real credentials can measure: the model's time
to answer a request of this size (some 12,000 to 15,000 input tokens by the usual 3 to 4 characters per token for
these languages, an estimate; the engine's tokenizer gives the real count) and the network to its EU endpoint,
and the quality of the fields it fills (blocked on SAL-5's adapter and the DPA).

**Left.**
- The PDF names no patient, practitioner or clinic and shows the signing time in UTC: which identifiers a signed
  document must carry (and the clinic's time zone) is for the owner and the doctor partner, and needs the patient's
  and practitioner's names through their services (audited reads).
- A coded field is checked by its form only; a link to the term list (`terminology`) and a FHIR `Condition` for a
  diagnosis come with the clinical feature.
- Discarding a draft (`entered-in-error`) and amending a signed document: not in the ticket.
- An orphan PDF if the api dies between the put and the commit: its key is never stored, so it cannot be read, but
  nothing deletes it yet (the attachments' purger now leaves `composition/` alone).
- The web wiring of `templates()`, `generate`, `sign` (SAL-9's page): a later PR. `sign` there sends no `If-Match`
  yet; the draft's `version_id` is now on every event for it.
