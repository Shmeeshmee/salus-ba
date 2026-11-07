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
| AWS SES (ADR 0007) | Sending e-mail from `cmd/notifier` | Recipient e-mail address; clinic name; a link into the app (invite links carry a one-time token in the URL fragment). Appointment e-mails add the date and time of the appointment in the clinic's zone; that a patient has an appointment with a clinic is health data (Art. 9, `docs/booking/SPEC.md` section 3). No practitioner, specialty, reason or other clinical content. The booking confirmation and the reminder add one fixed sentence when a questionnaire is waiting to be filled in (COPY E1, `docs/questionnaire`): that one is outstanding, never its title, a question or an answer. Patient e-mails go only to the patient | EU region, to be chosen | Not wired in. Needs the DPA; until then only SMTP to Mailpit exists, locally |
| Hosting provider (not chosen) | Compute, managed Postgres, object store, backups, KMS | Everything the platform stores: accounts, sessions, audit events, the directory, and health data (special category, Art. 9). Its EU object store will hold the questionnaire attachments' encrypted bytes (photos, documents, videos; `docs/attachments/SPEC.md` §8: bucket private, versioning off, its own encryption at rest on); its KMS will hold the key-encryption key that wraps each file's data key (`BINARY_KEK` locally) | EU/EEA | Not chosen. Needs the DPA, encryption at rest, and the KMS decision for `TOTP_KEY` (`docs/accounts/SPEC.md`) and `BINARY_KEK` (`docs/attachments/SPEC.md` §8) |
| LiveKit host (ADR 0006) | Running the self-hosted LiveKit server and TURN | Live consultation audio and video (health data), participant identities as opaque ids | EU/EEA | Not deployed. If we self-host on the hosting provider, it is covered by that DPA; a hosted LiveKit Cloud or separate TURN service would be its own processor |

A speech-to-text engine for live transcription (ADR 0006) is not chosen; a hosted engine would be a
processor holding patient audio and needs a DPA, an entry here and explicit SPEC approval first.

## Not processors

- Mailpit, the compose Postgres and Redis: local development only, synthetic data only.
- The compose object store (Garage): local development only, synthetic files only.
- ClamAV (clamd, `docs/attachments`): self-hosted beside the worker; files are scanned inside our stack and never
  leave it. Its signature updates (freshclam) download from ClamAV's CDN and send no personal data.
- GitHub (source code and CI): holds no personal data. Test data is synthetic (`@example.test` addresses,
  invented names); production data never reaches CI or a developer machine.
