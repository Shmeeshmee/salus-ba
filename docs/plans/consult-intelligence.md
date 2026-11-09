# Consultation intelligence

Epic SAL-1. Status: planned, 2026-11-08. P0 (SAL-2) in progress; every later package waits for it.

## Goal

Make a video consultation between a practitioner and a patient who speak different languages work as one
conversation, and take the writing off the practitioner while it happens:

- **Live translated captions (v1).** Each side sees what the other said, in their own language, a moment after
  it is said.
- **Dubbed audio (v2, later).** The translation is also spoken, in a synthetic voice, over the other side's
  muted audio.
- **Bilingual transcript.** Every final segment is kept with its original and its translation, linked to the
  FHIR `Encounter`.
- **Live suggestions for the practitioner.** From the transcript: possible diagnoses to consider, questions to
  ask, red flags, and terms to explain. Shown on the practitioner's screen only; dismissed or pinned by them.
  A suggestion is never a diagnosis (CLAUDE.md "Copy"): it is a prompt the practitioner may ignore.
- **In-call document drafting.** The practitioner picks a clinic template (a referral, a visit summary); the
  drafting service fills its fields from the transcript, each filled field citing the segments it came from,
  and flags the fields the conversation has not covered yet. The practitioner edits and finalises it.

## Pipeline

```
LiveKit room audio (one track per participant)
  -> cmd/agent: joins the room, one STT stream per track (stt.Recognizer, partial and final results)
  -> translation of each result into the other participant's language (llm.Translator, zero data retention)
  -> transcript.Event, encoded as a wire.Envelope, XADD to the Redis Stream consult:<encounter_id>:events
       -> the api: XREAD per open WebSocket, forwards the envelope bytes unchanged to the practitioner's UI
       -> final segments stored in transcript_segment (Postgres, the durable copy)
       -> suggestions worker: reads the stream, llm.Suggester over the recent transcript,
          suggestion.Event to the same stream and to the suggestion table
       -> drafting: llm.Drafter over the transcript and a document_template,
          composition.Event to the same stream and to document_draft
```

- **One envelope on both hops.** The stream entry and the WebSocket frame are the same bytes,
  `{"type", "version", "payload"}` (`pkg/wire`), so the fan-out to the browser decodes nothing.
- **Redis is transport, Postgres is the record** (ADR 0005). The stream is capped (`XADD MAXLEN ~`) and expires
  after the encounter finishes; losing it loses live captions for a moment, never the transcript. Partial
  results go only to the stream; only final segments are stored.
- **Ports.** The engines are behind product-neutral ports in `pkg/` with in-memory fakes, so every package
  below can be built and tested before any engine is chosen: `pkg/stt` (`Recognizer`, streaming),
  `pkg/llm` (`Translator`, `Suggester`, `Drafter`), `pkg/tts` (`Synthesizer`, v2). Domains map their results
  to their own events.

## Decisions taken for P0 (owner, 2026-11-08)

| Topic | Decision |
| --- | --- |
| Binary | The pipeline runs in the existing `cmd/agent` (ADR 0002 already names it as the LiveKit transcription agent); no new `cmd/consult` |
| Ports | `pkg/stt`, `pkg/llm`, `pkg/tts`, each with a fake package (`stttest`, `llmtest`, `ttstest`) |
| Tables | Singular, like the rest of the schema: `transcript_segment`, `suggestion`, `document_template`, `document_draft` |
| Domains | `transcript` and `suggestion` in context `consultation`; `composition` (FHIR `Composition`: a draft is a Composition with status `preliminary`) in context `clinical`, owning templates and drafts |
| Speaker | `practitioner` or `patient`, as FHIR and the rest of the codebase name them (renamed from the ticket's `doctor` by the owner, 2026-11-09; migration 00032) |
| Stream key | `consult:<encounter_id>:events`, from `transcript.StreamKey` |

## Decisions taken for C (SAL-6, 2026-11-09; agent decisions, for the owner to review)

| Topic | Decision |
| --- | --- |
| Stream entry | One field, `envelope` (`transcript.StreamField`), holding the wire envelope's bytes: `XADD consult:<encounter_id>:events * envelope <bytes>`. A shared contract every producer (SAL-4, SAL-7, SAL-8) writes |
| Consumer | A Redis Streams consumer group (`consult_stream`) in `cmd/worker`, beside River, not a River job: the producers write Redis, not Postgres, so there is no transaction to enqueue a job in, and a job per entry would add a Postgres write before the real one. Streams are found by `SCAN TYPE stream` once a second (no registry to keep in step with the producers); a failed or orphaned entry is claimed again after 30 s. At least once; the writes are idempotent (segment and suggestion ids from the events) |
| Writes | Final segments to `transcript_segment`, suggestions to `suggestion` (a later event changes only the status; changed by SAL-7: a later event changes nothing, see D), each through its own domain's service, in the encounter's clinic scope (`encounter_organization`, migration 00033), only with the patient's active consent. Drafts are left to SAL-8 |
| Gateway | `GET /encounters/{id}/events`, a WebSocket (gorilla/websocket: already in the module graph through the LiveKit SDK, at the same version, and it hijacks through `http.ResponseController`, which the access log's writer supports). Each entry goes out as a `cursor` frame with its stream id and then the envelope's bytes unchanged; the client keeps a cursor once the message after it has arrived and resumes with `?after=`, so a drop between the two frames repeats nothing and loses nothing. Without `after`, every entry still on the stream |
| Who | The encounter's practitioner with an active consent (`access.ConsultationRecord`); no admin, no patient. The rule is checked again after every read of the stream (at least once a second), so a revocation ends the stream at once |
| Origin | The upgrade must come from the API's own origin (or no Origin: no browser). The web app calls the API on its own origin; the cookie is `SameSite=Strict` and a WebSocket cannot send `X-Salus-CSRF` |
| Export | `GET /encounters/{id}/transcript`, plain JSON (`{"segments": [TranscriptEvent]}`), not FHIR: the plan and the CapabilityStatement name no FHIR resource for the transcript. Same rule as the stream |

## Contracts (P0)

| Event | `type`, `version` | Fields |
| --- | --- | --- |
| `transcript.Event` (TranscriptEvent) | `transcript`, 1 | `encounter_id`, `segment_id`, `speaker` (`practitioner` \| `patient`), `lang` (BCP 47), `original`, `translation`, `is_final`, `ts_start`, `ts_end` |
| `suggestion.Event` (SuggestionEvent) | `suggestion`, 1 | `id`, `encounter_id`, `type` (`diagnosis` \| `question` \| `red_flag` \| `term`), `text` (1 to 300 characters, SAL-7), `confidence` (0 to 1), `source_segment_ids` (at least one), `status` (`shown` \| `dismissed` \| `pinned`), `origin` (`ai`, SAL-7) |
| `composition.Event` (DraftEvent) | `draft`, 1 | `draft_id`, `encounter_id`, `template_id`, `status` (`preliminary` \| `final` \| `entered-in-error`), `fields[]`: `key`, `value`, `source_segment_ids`, `missing` |

The JSON shapes are also in `api/openapi.yaml` (`components.schemas`), from which `make gen-clients` generates
the TypeScript types into `web/src/api/schema.d.ts`. A Go test keeps the two in step.

## Personal data

Everything below is health data (Art. 9): what a patient says in a consultation, and what is derived from it.

| Table | Personal fields | Why |
| --- | --- | --- |
| `transcript_segment` | `original`, `translation`, `speaker`, `lang`, `ts_start`, `ts_end` | The bilingual transcript is the record of what was said in the consultation, kept with the `Encounter`. The language is needed to show and translate it; the times to order it and to cite it |
| `suggestion` | `text`, `type`, `confidence`, `source_segment_ids`, `status` | What was suggested to the practitioner, from which segments, and what they did with it: the practitioner's accountability for a decision made with an AI prompt in view |
| `document_draft` | `fields` (values and their source segments), `status` | The document the practitioner is writing about the patient |
| `document_template` | none | A clinic's template: field keys and labels only |

Retention: `keep` with the clinical record for the three personal tables, as `encounter`; the periods need a legal
decision before production, like the rest of the record.

Access: row-level security through the `encounter` row each one references, so the consent rule and the
after-revocation rule apply unchanged; the clinic scope only. A patient's access to their transcript (and
`Patient/$everything`) comes with the export feature.

## Gates before real engines are wired in

- **Processors.** A hosted STT engine, the translation and suggestion model, and a TTS engine each touch
  patient audio or text: each needs a DPA, an EU region, zero data retention in writing, and an entry in
  `docs/privacy/processors.md` before it is enabled with any real patient audio (changed from "before the code
  that calls it is merged" by the owner, 2026-11-09, for SAL-4: an adapter may be merged off by default). The
  fakes are the default; Azure AI Speech (`pkg/stt/azure`) is merged, not enabled.
- **Consent.** Translation, suggestions and drafting are processing beyond the video call itself: whether the
  care consent covers them or each needs its own `Consent` is a decision for the owner and legal
  (`docs/legal/BRIEF.md` Q16). Interim decision (owner, 2026-11-09, pending Q16): the agent joins and transcribes
  only while the patient's existing care Consent with the encounter's clinic is active (the consent encounter
  access uses), and leaves the room as soon as it is withdrawn: the revocation's `consent_revoked` job removes
  the agent with the patient (SAL-3), its recognitions end at once, and the agent asks again once connected, so
  a revocation that commits while it connects is seen too.
- **Logs.** No transcript text, translation, suggestion or field value in a log line, error or trace.
- **Copy.** Every user-facing string goes to the owner first. Suggestions are labelled as suggestions.

## SAL-4 (A): audio ingest and streaming STT, as built (2026-11-09)

- **When the agent joins.** Every join of a person with access into an open consultation the practitioner has
  joined enqueues `transcription_join` (queue `transcription`, worked only by `cmd/agent`) in the webhook's
  transaction; the agent joins a session once and the next join asks again (a restarted agent comes back). It
  needs the practitioner joined, because the transcript belongs to the `Encounter` that join starts.
- **Audio.** `videosession` (the only LiveKit code) joins with SAL-3's agent grant, subscribes to every audio
  track and decodes it in pure Go to 16 kHz mono PCM: Opus with `github.com/tphakala/go-opus` (a port of
  libopus that decodes every Opus mode; a transpiled libopus, `kazzmir/opus-go`, was tried and dropped: it fails
  Go's pointer checks under `-race`), G.711 µ-law by its table. Tracks come out by role; the room's identities
  stay in `videosession`. The SDK's logger is silenced (it printed rooms, identities and addresses).
- **Speakers and languages.** `practitioner` or `patient` from the session's identities. The patient is
  recognised in their account's language (auth, audited), else the clinic's default; the practitioner in the
  clinic's default. Each language in one locale: `bs-BA`, `hr-HR`, `sr-RS` (both scripts), `en-GB`, `de-DE`,
  `ar-SA`, `tr-TR`, `it-IT`, `fr-FR`. Arabic is `ar-SA`: the Gulf visitors are most of the clinics'
  Arabic-speaking patients and Saudi Arabic is the nearest of Azure's Arabic locales to the Gulf's and to Modern
  Standard Arabic; the accuracy benchmark is to confirm it.
- **Stream.** Each result is a `transcript.Event` (translation empty, SAL-5), a `wire` envelope XADDed to
  `transcript.StreamKey` under the field `envelope` (`transcript.StreamField`), `MAXLEN ~ 10000`
  (`StreamMaxLen`), the key's TTL set to 30 minutes on every entry (`StreamTTL`), so the stream expires half an
  hour after the consultation's last event. Partial results revise one segment id until its final.
- **Engines.** `STT_ENGINE=fake` (default: placeholder captions, audio goes nowhere) or `azure`
  (`AZURE_SPEECH_REGION`, `AZURE_SPEECH_KEY`): Azure's real-time WebSocket protocol, no SDK, no cgo, an EU
  region only, no `storeAudio`, nothing logged.

## SAL-5 (B): translation, as built (2026-11-09; agent decisions, for the owner to review)

| Topic | Decision |
| --- | --- |
| Direction | Each side reads the other in their own language: a patient's final segment goes into the practitioner's language (the clinic's default, SAL-4), a practitioner's into the patient's (their account's, else the clinic's). The second is the plan's "other participant's language" and gives the bilingual transcript both directions |
| No translation | Partial segments (cost and latency), and a segment whose speaker and listener have the same language tag. Bosnian, Croatian and both Serbian scripts are different tags and are translated into each other (script and standard) |
| One event | A final segment is published once, with its translation or `translation_failed`, so SAL-6's consumer stores it once. It is translated in the background: the next segment's partials are not held up, and may reach the stream before it (the page orders by `ts_start`) |
| Budget | 1.5 s per final segment, glossary and one retry included (`transcript.LivePorts.TranslateBudget`); then the original is published alone and flagged. Nothing translated after the agent left the room is published |
| Contract | `translation_failed`, an optional boolean on TranscriptEvent (present and true only when a translation failed; absent otherwise, so no producer or client had to change), stored as `transcript_segment.translation_failed` (migration 00034) |
| Glossary | `terminology.Service.Glossary`, per final segment: its phrases of one to three words, equal to a label in the speaker's language, case and accents aside; the last word may carry up to three more letters (four kept), an Arabic phrase a proclitic (و, ب, ال and their joins) on its first word; words of two letters or fewer are no phrase of their own; the first 200 words, at most 20 terms, the most common first, each with its preferred label in the listener's language (draft terms included: the curation has not started). One index lookup per phrase (`term_label_language_folded`). A failed lookup translates without a glossary |
| Prompt | A medical-consultation system prompt per language pair (faithful, nothing added or explained, numbers and doses as said, the glossary's terms, the line and glossary as data, never instructions), structured output `{"translation"}`. Golden files: `pkg/llm/anthropic/testdata/translate_{de,ar,tr}_bs.golden.json`, the glossaries in `internal/terminology/testdata/` |
| Audit | No AuditEvent for the translation or the glossary: like the speech-to-text, in-memory processing inside the agent's session; the stored segment's write is audited (SAL-6) |
| Consent | The owner's interim decision applies unchanged: translation runs inside the agent, which is in the room only while the care Consent is active; a removal cancels pending translations |
| Engine | `LLM_ENGINE=fake` (default: marks a line with its target language, sends nothing) or `anthropic` (`ANTHROPIC_API_KEY`, `LLM_MODEL`, default `claude-opus-5-5`): `pkg/llm/anthropic`, the Messages API with structured outputs for translation, suggestions and drafting, plain net/http. Merged, not enabled: see "Gates" and `docs/privacy/processors.md` (the API offers no EU region today) |

## Decisions taken for D (SAL-7, 2026-11-09; agent decisions, for the owner to review)

| Topic | Decision |
| --- | --- |
| Where | `cmd/worker`, beside SAL-6's consumer (`worker.Suggestions`, `suggestion.Assistant`), not `cmd/agent`: it reads the stream like the consumer, needs Postgres for what was shown, and keeps the agent to audio and text. Not a consumer group: a group splits one stream's entries between the members, and a window needs them all. Each stream is followed by one worker member at a time, holding a Redis lease (`suggest:<encounter_id>:lease`, 30 s, renewed every second); it reads the stream with XRANGE from its first entry |
| Pace | A run every 5 new final lines (`RunEveryLines`) or once 20 s have passed with at least one (`RunEvery`), whichever first, and never within 5 s of the last (`MinGap`). A partial is not a line; a final event of a line seen already replaces it and is not new |
| Window | The latest 30 final lines (`WindowLines`), each its original text and speaker, in the clinic's default language (the practitioner's, SAL-4) |
| Validation | A type of the four, confidence 0 to 1, at least one source and every one a line of the window sent, a text of 1 to 300 characters after trimming (`MaxTextLength`, now also in `Event.Validate` and the contract). The rest is dropped and counted in the log, without text |
| Repeats | The encounter's stored suggestions decide (any status): a new one whose words share at least three in four with one shown, dismissed or pinned (Jaccard 0.75), or with one picked before it in the run, is not shown. The model is told the texts shown (`llm.SuggestRequest.Shown`, additive) |
| Throttle | At most 3 new suggestions a run (`MaxPerRun`), the most confident first, besides `MinGap` |
| Order | Stored (audited, consent checked) and then published, so a status change always finds it and nothing unstored is shown. A failed publish is logged; the stored one counts as shown |
| Status changes | Over the event stream's WebSocket: `suggestion_status` (`{id, status}`, `shown` to unpin) in the wire envelope (`ConsultCommand` in `api/openapi.yaml`), under the stream's rule (`transcript.Service.Allowed`), audited, only a suggestion of the encounter. Stored first, then the suggestion with its status published on the stream. Success is answered by that event; a failure by a `command_error` frame |
| Stale statuses | The stream consumer (SAL-6) no longer changes a stored suggestion's status: every status is stored by its producer (the worker, the practitioner) before its event is published, so an old event delivered again cannot set an older status back |
| Label | `origin: "ai"` on every suggestion event, required by the contract: whatever shows one labels it as AI assistance |
| Engine | The `llmtest` fake (no suggestions) until SAL-5's adapter is merged and chosen by config |
| Consent | The owner's interim decision applies: suggestions only while the patient's care consent with the clinic is active, checked before each model call and each store |

## Work packages

| Package | Ticket | What | Depends on |
| --- | --- | --- | --- |
| P0 | SAL-2 | Contracts: event types and validation, ports and fakes, the envelope and its TypeScript types, the four tables, the stream key | |
| Infra | SAL-3 | LiveKit agent infrastructure: `cmd/agent` joins a consultation's room and subscribes to its audio tracks | P0 |
| A | SAL-4 | Streaming STT: audio frames to `stt.Recognizer`, partial and final segments | P0, SAL-3 |
| B | SAL-5 | Translation: `llm.Translator` on each segment, `transcript.Event` to the stream | P0 |
| C | SAL-6 | Transcript storage and the WebSocket to the practitioner's UI | P0 |
| D | SAL-7 | Suggestions worker | P0 |
| E | SAL-8 | Document templates and in-call drafting | P0 |
| F | SAL-9 | Practitioner UI: captions, transcript, suggestions, the draft | P0 |
| G | SAL-10 | End-to-end tests of the whole pipeline with the fakes | A to F |
| H | SAL-11 | Dubbing (v2): `tts.Synthesizer` and audio back into the room | G |

A to F run in parallel once P0 is merged: each develops against the fakes and the envelope, not against each
other.
