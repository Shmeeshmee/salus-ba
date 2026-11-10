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
  Extended by the owner, 2026-11-10 (docs/plans/v1-finish.md "Decisions"; SAL-12), still pending Q16: a separate,
  granular, revocable FHIR `Consent` for transcription, translation and AI assistance, per patient and clinic
  (`transcription_consent`, migration 00038), asked in the patient app before the call. The agent joins, and the
  transcript, the suggestions and the drafting run, only while both the care Consent and this one are active
  (`access.Transcribed`). Declining, the call runs without the agent; withdrawing, the call goes on and the
  withdrawal's `transcription_consent_revoked` job removes the agent at once, which asks again once connected as
  above. A revocation of the care Consent ends the transcription Consent with it.
- **Logs.** No transcript text, translation, suggestion or field value in a log line, error or trace.
- **Copy.** Every user-facing string goes to the owner first. Suggestions are labelled as suggestions.

## SAL-4 (A): audio ingest and streaming STT, as built (2026-11-09)

- **When the agent joins.** Every join of a person with access into an open consultation the practitioner has
  joined enqueues `transcription_join` (queue `transcription`, worked only by `cmd/agent`) in the webhook's
  transaction; the agent joins a session once and the next join asks again (a restarted agent comes back). It
  needs the practitioner joined, because the transcript belongs to the `Encounter` that join starts. Since SAL-12
  it also needs the patient's transcription Consent with the clinic (Gates, "Consent"): without it no join is
  enqueued, and `AgentJoin` and `AgentAdmitted` refuse (`ErrConsentRequired`).
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
| Engine | `cmd/api` takes the drafting engine from `config.LoadLLM` (SAL-5): `LLM_ENGINE=fake` (the default) is `llmtest.Drafter{DiscardCalls: true}`, every field missing and no call kept; `anthropic` is SAL-5's adapter, off until its DPA |

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
| Engine | `config.LoadLLM` (SAL-5): `LLM_ENGINE=fake` (the default: `llmtest.Suggester` keeping no calls, suggesting nothing) or `anthropic` (not enabled before its DPA). The adapter puts `SuggestRequest.Shown` in a `<shown>` block, as data |
| Consent | The owner's interim decision applies: suggestions only while the patient's care consent with the clinic is active, checked before each model call and each store |

## E (SAL-8): document templates and drafting, as built (2026-11-09; agent decisions, for the owner to review)

| Topic | Decision |
| --- | --- |
| Tables | `composition` owns `document_template`, `document_draft` and the new `document_template_translation` (migration 00036). The signed draft is the FHIR `Composition`; its `DocumentReference` and `Provenance` are columns of the same row (one of each per signature), served by `composition`: no `documentreference` domain until a second source of documents needs one |
| Templates | Per clinic; fields `{key, type, label, instruction, required}`, types `text`, `list` (one item per line), `date` (YYYY-MM-DD), `coded` (one ICD-10 code, checked by its form, not against the term list); a LOINC document type (`11488-4` consult note, `57133-1` referral note); status `draft` (wording not reviewed yet) or `active`, both usable. Kept by every practitioner and admin of the clinic, like questionnaires, with `If-Match`. Translations as for questionnaires (original language plus a translation per language, the texts by field key) but complete or refused on every write: a template has no publishing step |
| Drafting | `POST /encounters/{id}/drafts` answers with the stored draft (synchronous; the model's call outside any transaction, at most 25 s within the 30 s request timeout). The model reads the stored final segments as they were said (originals) and the template in the language asked for (its original by default). A value is kept only with at least one source, every source a segment of the transcript, and of its field's type; otherwise the field is missing. No transcript: every field missing, no model call. One open draft per encounter and template (a unique index): a redraft replaces its fields as its next version. The draft keeps the labels, types and instructions it was drafted with |
| `version_id` | Added to `DraftEvent` (additive, optional in the schema): what a sign names in `If-Match`, and how a page tells a redraft from a replay (SAL-9's "Generating…" note) |
| Required | The model is told which fields are required and never guesses one; a sign with a required field empty is refused (`422 required_missing`); an empty optional field is signed as missing |
| Signing | `POST /drafts/{id}/sign` with `If-Match` and a value per field: a value as drafted keeps its sources, an edited one has none. The Composition becomes `final`, attested (`legal`) by the appointment's PractitionerRole; the PDF is rendered, encrypted (`pkg/envelope`, the attachments' key-encryption key) and put in the object store under `composition/<DocumentReference id>` before the final transaction; the access, status and version are checked again in it, and the object deleted if it fails. The attachments' purger now keeps keys that are no binary id |
| FHIR | Reads only: `Composition`, `DocumentReference` (the PDF's url is `GET /document-references/{id}/content`, `context.encounter`, `context.related` the Composition) and `Provenance` (targets the Composition's signed version and the DocumentReference; activity `LA` legally authenticated; agent the signer as author). A field without a value is a section with an `emptyReason` and a dash as its narrative (`cmp-1`) |
| PDF | `github.com/signintech/gopdf` v0.38.4 (MIT, one dependency, `phpdave11/gofpdi`) with the Go fonts (`golang.org/x/image` v0.46.0, BSD: Latin with the bs/hr/sr letters, Cyrillic), embedded as subsets, deterministic. A writer of our own with embedded Unicode TrueType (CID fonts, widths, wrapping, `ToUnicode`) is far over CLAUDE.md's 50 lines. Arabic script is not shaped, so a template is not written in Arabic (refused as `invalid_template`); a letter the fonts lack (CJK, emoji) would silently drop from the PDF, so a template text with one is refused, a drafted value with one is not kept and a signed one is refused (`invalid_values`). The PDF shows the title, the time of signing in UTC and each field under its label; no patient or practitioner name and no clinic header yet (below, "Left") |
| Access | `access.ConsultationRecord` for drafting, signing and every read of a draft or document (the encounter's practitioner with the patient's active consent, in the clinic's scope); templates for staff of the clinic |
| Consent | The owner's interim decision applied: drafting runs only while the patient's care consent with the encounter's clinic is active; checked before the model is asked and again before its answer is stored, and for a sign before the PDF is made and again before it is committed |
| Engine | The api wires `llmtest.EmptyDrafter{}` (every field missing, nothing kept of a request) until SAL-5's engine configuration is merged into the branch |
| Seeds | `make document-templates-seed clinic=<id>` (`salusctl document-templates seed`): a specialist findings report and a referral from `assets/document-templates`, in Bosnian with an English translation, status `draft`, titled "(primjer za pregled)", each file's note saying the wording is a placeholder for the doctor partner to review |

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
