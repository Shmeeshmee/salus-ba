# Processors

Every third party that touches personal data on our behalf (GDPR Art. 28). A processor needs a signed DPA
and an entry here **before** it is wired in (CLAUDE.md "GDPR and health data"). Data stays in the EU/EEA.

Status as of 2026-11-04 (accounts feature, package 5): **no processor is wired in.** Everything that runs
today runs in our own compose stack: Postgres, Redis, Mailpit (local e-mail catcher, synthetic data only)
and LiveKit in dev mode. Since package 5 the notifier sends invite e-mails, through SMTP to Mailpit only;
there is no SMTP relay or SES configuration for any other environment. Since the booking feature's package 6
it also sends appointment e-mails (confirmation, cancellation, reminder) the same way: no new processor.
Since the deploy plan's DEP-6 the notifier can send through an SMTP relay with TLS and a login (`SMTP_*`); no
relay is configured with real recipients until its DPA is signed (the SMTP relay row below).
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
| AWS SES (ADR 0007) | Sending e-mail from `cmd/notifier` | Recipient e-mail address; clinic name; a link into the app (invite links carry a one-time token in the URL fragment). Appointment e-mails add the date and time of the appointment in the clinic's zone; that a patient has an appointment with a clinic is health data (Art. 9, `docs/booking/SPEC.md` section 3). No practitioner, specialty, reason or other clinical content. The booking confirmation and the reminder add one fixed sentence when a questionnaire is waiting to be filled in (COPY E1, `docs/questionnaire`): that one is outstanding, never its title, a question or an answer. The document e-mail (SAL-17) says that a document of the appointment is ready in the app, with the clinic and the appointment's date: never its title, type, signer or content. Patient e-mails go only to the patient | EU region, to be chosen | Not used in production: ADR 0011 point 5 chose an EU SMTP relay (below). Kept for the record |
| SMTP relay: Brevo (Sendinblue SAS, FR) or Mailjet (Mailjet SAS, FR), one of them, chosen by configuration (ADR 0011; production uses it instead of SES) | Sending e-mail from `cmd/notifier` | The same as SES above: recipient address, clinic name, links into the app, and the date and time of an appointment (health data, Art. 9) | EU/EEA; the data centres to confirm in the DPA | Chosen (ADR 0011), not wired in: needs the chosen provider's DPA and its sub-processors recorded here before the first e-mail to a real person |
| Hosting provider: Hetzner Online GmbH, Cloud server in Helsinki (FI), an encrypted Volume for its data and a Storage Box for backups (ADR 0011) | Compute, Postgres (self-hosted on the server), object store (Garage, self-hosted), encrypted backups; no KMS: the data on a LUKS-encrypted Volume, the keys in a root-only key file on it (ADR 0011 points 2 and 3) | Everything the platform stores: accounts, sessions, audit events, the directory, and health data (special category, Art. 9). The encrypted Volume holds the questionnaire attachments' encrypted bytes (photos, documents, videos) in Garage, and the Storage Box the nightly encrypted Postgres dumps and archives of the attachments (14 days) | EU/EEA: Helsinki (FI); the Storage Box in the location chosen with it | Chosen (ADR 0011), not wired in: needs the DPA (Hetzner console; ADR 0011 point 1) |
| Microsoft Azure AI Speech (`pkg/stt/azure`, SAL-4) | Live speech-to-text of consultations for the transcription agent (`cmd/agent`) | Each participant's microphone audio during the call (health data, Art. 9), streamed; the language spoken (a locale such as `bs-BA`); the recognised text it returns. No name, identity, room or id: the connection carries only the key and the language | An EU region only: the adapter refuses any other (`westeurope`, `northeurope`, `germanywestcentral`, `francecentral`, `swedencentral`, `italynorth`, `polandcentral`, `spaincentral`, `norwayeast`); to be chosen | Not enabled: needs the DPA, an EU region, zero data retention / no-logging in writing before it may be enabled. The code is merged off by default (owner, 2026-11-09): `STT_ENGINE` defaults to the fake; `azure` must not be set with real patient audio until then |
| Anthropic, Claude API (`pkg/llm/anthropic`, SAL-5) | The consultation pipeline's language model (`cmd/agent`; the suggestions worker in `cmd/worker`, SAL-7; the drafting in `cmd/api`, SAL-8): translating each final transcript segment, suggesting from the transcript, filling a document's fields from it | What was said in the consultation (health data, Art. 9): for a translation, one final segment, the two languages and the terms the terminology found in it; for suggestions and drafting, the transcript as `[segment id] speaker role: text` (for suggestions the latest 30 final lines, with the texts of the suggestions already shown) and, for drafting (SAL-8), every stored final segment of the encounter as said (its original language), the language to write in, and for each of the template's fields its key, label, type (text, list, date, coded), whether it is required and the clinic's instruction for it; what comes back are the values with the segment ids they cite, checked by type and against the transcript before anything is kept. Segment ids are random UUIDs. No name, account, patient, encounter or clinic id, and no audio. The fixed JSON schemas of the answers hold no patient data (the API caches them up to 24 hours) | None in the EU: the Messages API runs inference in `global` (any geography, the default) or `us` only (`inference_geo`), and stores data at rest in workspace geo `us` only (Anthropic's data residency documentation, checked 2026-11-09) | Not enabled, and may not be under the plan's gate as it stands: needs the DPA, zero data retention and an EU region. Zero data retention is an arrangement Anthropic's account team enables for the organisation (not a request flag); under it the Messages API and structured outputs keep no prompts or answers after the response (Covered Models such as Claude Fable are excluded; the default model, `claude-opus-5-5`, is not one). The adapter sends no beta and no retention-based feature. No EU region exists, so the adapter stays off until that condition is met or the owner decides otherwise. Code merged off by default (as the Azure adapter, owner 2026-11-09): `LLM_ENGINE` defaults to the fake; `anthropic` must not be set with real patient data |
| Google: Firebase Cloud Messaging (`pkg/push/fcm`, SAL-29) | Push notifications to the patient app on Android (`cmd/notifier`) | The device's FCM registration token (personal data: it identifies the device), and a fixed alert: the title "Salus" and "New message from your clinic" in the patient's language. No message text, name, clinic, id or other data payload. The request is authenticated with the project's service account | Outside the EU/EEA (Google's global infrastructure): needs a transfer basis (the EU-US Data Privacy Framework or standard contractual clauses) | Not enabled: the adapter is merged off; the notifier runs the fake until the owner adds the service account's key (docs/deploy/push.md), which needs the DPA (Google's Firebase data processing terms) and the transfer basis recorded here first |
| Apple: Apple Push Notification service (`pkg/push/apns`, SAL-29) | Push notifications to the patient app on iOS (`cmd/notifier`) | The device's APNs token (personal data: it identifies the device), and the same fixed alert as FCM. Nothing else | Outside the EU/EEA (Apple's infrastructure): needs a transfer basis | Not enabled: merged off; the fake until the owner adds the team's APNs key (docs/deploy/push.md), after the DPA (Apple Developer Program terms) and the transfer basis are recorded here |
| SMS provider (`pkg/sms`, docs/plans/appointments.md, SAL-33): not chosen; an EU/EEA-established provider (owner, 2026-11-13, Q2) | Appointment reminders and waitlist offers by SMS from `cmd/notifier`, only for patients who opted in, at clinics that turned the paid option on | The patient's phone number (E.164) and the text: the clinic's name, the date and time of an appointment or an offered time, and a one-time link (that a person has an appointment with a clinic is health data, Art. 9). No name, practitioner, specialty or other clinical content | EU/EEA for processing and storage; delivery necessarily passes the recipient's mobile network in Bosnia and Herzegovina (outside the EU/EEA): the transfer basis to record with the DPA | Planned, not wired in: `pkg/sms` runs its fake (`smstest`) in every environment and no adapter exists; the provider's adapter comes with the chosen provider, which needs its DPA, sub-processors and transfer basis recorded here, then the owner's credentials |
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

Since SAL-29 (2026-11-12) the push adapters for FCM and APNs exist (rows above), merged off by default like the
speech and language-model adapters: no push reaches Google or Apple until their DPAs and the transfer basis are
recorded and the owner adds the credentials. The clinics' webhook endpoints are not processors of ours: each is the
clinic's own system, and the clinic is the controller of what it receives (docs/privacy/dpia.md, SAL-29).

## Not processors

- Mailpit, the compose Postgres and Redis: local development only, synthetic data only.
- The compose object store (Garage): local development only, synthetic files only.
- ClamAV (clamd, `docs/attachments`): self-hosted beside the worker; files are scanned inside our stack and never
  leave it. Its signature updates (freshclam) download from ClamAV's CDN and send no personal data.
- A clinic's webhook endpoint (SAL-29): the clinic's own system; the clinic is the controller of what it receives.
- GitHub (source code and CI): holds no personal data. Test data is synthetic (`@example.test` addresses,
  invented names); production data never reaches CI or a developer machine.
