# Processors

Every third party that touches personal data on our behalf (GDPR Art. 28). A processor needs a signed DPA
and an entry here **before** it is wired in (CLAUDE.md "GDPR and health data"). Data stays in the EU/EEA.

Status as of 2026-11-04 (accounts feature, package 5): **no processor is wired in.** Everything that runs
today runs in our own compose stack: Postgres, Redis, Mailpit (local e-mail catcher, synthetic data only)
and LiveKit in dev mode. Since package 5 the notifier sends invite e-mails, through SMTP to Mailpit only;
there is no SMTP relay or SES configuration for any other environment. Since the booking feature's package 6
it also sends appointment e-mails (confirmation, cancellation, reminder) the same way: no new processor.
Since the consultation feature's package 1 LiveKit runs on our own configuration (`deployments/livekit.yaml`,
no longer dev mode), self-hosted, so it is no processor (SPEC.md §2 "Processor"): it receives random room
names and participant identities only.
Since the questionnaire attachments feature (`docs/attachments`, 2026-11-07) patients' files are stored, encrypted,
in an S3-compatible object store and scanned by ClamAV (clamd); both run self-hosted in our own stack (Garage in
compose), so neither is a processor and no processor is added.

## Wired in

None.

## Pending

Each of these needs a signed DPA, its region and sub-processors recorded here, and a review of this file
before the code that uses it is merged.

| Processor | Purpose | Personal data it will touch | Region | Status |
| --- | --- | --- | --- | --- |
| AWS SES (ADR 0007) | Sending e-mail from `cmd/notifier` | Recipient e-mail address; clinic name; a link into the app (invite links carry a one-time token in the URL fragment). Appointment e-mails add the date and time of the appointment in the clinic's zone; that a patient has an appointment with a clinic is health data (Art. 9, `docs/booking/SPEC.md` section 3). No practitioner, specialty, reason or other clinical content. The booking confirmation and the reminder add one fixed sentence when a questionnaire is waiting to be filled in (COPY E1, `docs/questionnaire`): that one is outstanding, never its title, a question or an answer. The document e-mail (SAL-17) says that a document of the appointment is ready in the app, with the clinic and the appointment's date: never its title, type, signer or content. Patient e-mails go only to the patient | EU region, to be chosen | Not wired in. Needs the DPA; until then only SMTP to Mailpit exists, locally |
| SMTP relay: Brevo (Sendinblue SAS, FR) or Mailjet (Mailjet SAS, FR), one of them, chosen by configuration (ADR 0011; production uses it instead of SES) | Sending e-mail from `cmd/notifier` | The same as SES above: recipient address, clinic name, links into the app, and the date and time of an appointment (health data, Art. 9) | EU/EEA (France) | Chosen (ADR 0011), not wired in: needs the chosen provider's DPA and its sub-processors recorded here before the first e-mail to a real person |
| Hosting provider: Hetzner Online GmbH, Cloud server in Helsinki (FI) and a Storage Box for backups (ADR 0011) | Compute, Postgres (self-hosted on the server), object store (Garage, self-hosted), encrypted backups; no KMS (ADR 0011 "Open before real patient data") | Everything the platform stores: accounts, sessions, audit events, the directory, and health data (special category, Art. 9). The server's disk holds the questionnaire attachments' encrypted bytes (photos, documents, videos) in Garage, and the Storage Box the nightly encrypted Postgres dumps (14 days) | EU/EEA: Helsinki (FI); the Storage Box in the location chosen with it | Chosen (ADR 0011), not wired in: needs the DPA (Hetzner console), encryption at rest, and the decision on where `TOTP_KEY` (`docs/accounts/SPEC.md`) and `BINARY_KEK` (`docs/attachments/SPEC.md` §8) are kept, as Hetzner has no KMS (ADR 0011 "Open before real patient data") |
| Microsoft Azure AI Speech (`pkg/stt/azure`, SAL-4) | Live speech-to-text of consultations for the transcription agent (`cmd/agent`) | Each participant's microphone audio during the call (health data, Art. 9), streamed; the language spoken (a locale such as `bs-BA`); the recognised text it returns. No name, identity, room or id: the connection carries only the key and the language | An EU region only: the adapter refuses any other (`westeurope`, `northeurope`, `germanywestcentral`, `francecentral`, `swedencentral`, `italynorth`, `polandcentral`, `spaincentral`, `norwayeast`); to be chosen | Not enabled: needs the DPA, an EU region, zero data retention / no-logging in writing before it may be enabled. The code is merged off by default (owner, 2026-11-09): `STT_ENGINE` defaults to the fake; `azure` must not be set with real patient audio until then |
| Anthropic, Claude API (`pkg/llm/anthropic`, SAL-5) | The consultation pipeline's language model (`cmd/agent`; the suggestions worker in `cmd/worker`, SAL-7; the drafting in `cmd/api`, SAL-8): translating each final transcript segment, suggesting from the transcript, filling a document's fields from it | What was said in the consultation (health data, Art. 9): for a translation, one final segment, the two languages and the terms the terminology found in it; for suggestions and drafting, the transcript as `[segment id] speaker role: text` (for suggestions the latest 30 final lines, with the texts of the suggestions already shown) and, for drafting (SAL-8), every stored final segment of the encounter as said (its original language), the language to write in, and for each of the template's fields its key, label, type (text, list, date, coded), whether it is required and the clinic's instruction for it; what comes back are the values with the segment ids they cite, checked by type and against the transcript before anything is kept. Segment ids are random UUIDs. No name, account, patient, encounter or clinic id, and no audio. The fixed JSON schemas of the answers hold no patient data (the API caches them up to 24 hours) | None in the EU: the Messages API runs inference in `global` (any geography, the default) or `us` only (`inference_geo`), and stores data at rest in workspace geo `us` only (Anthropic's data residency documentation, checked 2026-11-09) | Not enabled, and may not be under the plan's gate as it stands: needs the DPA, zero data retention and an EU region. Zero data retention is an arrangement Anthropic's account team enables for the organisation (not a request flag); under it the Messages API and structured outputs keep no prompts or answers after the response (Covered Models such as Claude Fable are excluded; the default model, `claude-opus-5-5`, is not one). The adapter sends no beta and no retention-based feature. No EU region exists, so the adapter stays off until that condition is met or the owner decides otherwise. Code merged off by default (as the Azure adapter, owner 2026-11-09): `LLM_ENGINE` defaults to the fake; `anthropic` must not be set with real patient data |
| LiveKit host (ADR 0006) | Running the self-hosted LiveKit server and TURN | Live consultation audio and video (health data), participant identities as opaque ids | EU/EEA | Not deployed. If we self-host on the hosting provider, it is covered by that DPA; a hosted LiveKit Cloud or separate TURN service would be its own processor |

A speech-to-text engine for live transcription (ADR 0006) is not chosen; a hosted engine would be a
processor holding patient audio and needs a DPA, an entry here and explicit SPEC approval first.
Since the consultation intelligence plan's P0 (`docs/plans/consult-intelligence.md`, SAL-2, 2026-11-08) the ports
for it exist (`pkg/stt`, `pkg/llm` for translation, suggestions and drafting, `pkg/tts` for dubbing), implemented
by in-memory fakes only: no processor is added. Each real engine is a processor of patient audio or text and needs
the above, plus zero data retention in writing, before it is wired in.
Since SAL-4 (2026-11-09) the Azure AI Speech adapter exists (row above): by the owner's decision of 2026-11-09 the
rule for it is that the DPA, the EU region and zero data retention come before it is **enabled with any real patient
audio**, not before its code is merged. Compose and every environment run the fake.
Since SAL-5 (2026-11-09) the Anthropic Claude API adapter exists (row above), under the same rule: merged off by
default, not enabled with real patient data before its DPA, zero data retention and an EU region. The API has no EU
region today, so it stays off; the owner decides whether the gate changes (for example Claude through a cloud
platform's EU region, which would be a different adapter and processor entry).

## Not processors

- Mailpit, the compose Postgres and Redis: local development only, synthetic data only.
- The compose object store (Garage): local development only, synthetic files only.
- ClamAV (clamd, `docs/attachments`): self-hosted beside the worker; files are scanned inside our stack and never
  leave it. Its signature updates (freshclam) download from ClamAV's CDN and send no personal data.
- GitHub (source code and CI): holds no personal data. Test data is synthetic (`@example.test` addresses,
  invented names); production data never reaches CI or a developer machine.
