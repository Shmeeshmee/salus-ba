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
