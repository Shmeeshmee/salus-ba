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
