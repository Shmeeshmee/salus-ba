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
| Speaker | `doctor` or `patient`, as the ticket names them |
| Stream key | `consult:<encounter_id>:events`, from `transcript.StreamKey` |

## Contracts (P0)

| Event | `type`, `version` | Fields |
| --- | --- | --- |
| `transcript.Event` (TranscriptEvent) | `transcript`, 1 | `encounter_id`, `segment_id`, `speaker` (`doctor` \| `patient`), `lang` (BCP 47), `original`, `translation`, `is_final`, `ts_start`, `ts_end` |
| `suggestion.Event` (SuggestionEvent) | `suggestion`, 1 | `id`, `encounter_id`, `type` (`diagnosis` \| `question` \| `red_flag` \| `term`), `text`, `confidence` (0 to 1), `source_segment_ids` (at least one), `status` (`shown` \| `dismissed` \| `pinned`) |
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
  `docs/privacy/processors.md` before the code that calls it is merged (CLAUDE.md "GDPR and health data").
  The fakes are the only implementations until then.
- **Consent.** Translation, suggestions and drafting are processing beyond the video call itself: whether the
  care consent covers them or each needs its own `Consent` is a decision for the owner and legal before SAL-4.
- **Logs.** No transcript text, translation, suggestion or field value in a log line, error or trace.
- **Copy.** Every user-facing string goes to the owner first. Suggestions are labelled as suggestions.

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
