# Processors

Every third party that touches personal data on our behalf (GDPR Art. 28). A processor needs a signed DPA
and an entry here **before** it is wired in (CLAUDE.md "GDPR and health data"). Data stays in the EU/EEA.

Status as of 2026-11-04 (accounts feature, package 5): **no processor is wired in.** Everything that runs
today runs in our own compose stack: Postgres, Redis, Mailpit (local e-mail catcher, synthetic data only)
and LiveKit in dev mode. Since package 5 the notifier sends invite e-mails, through SMTP to Mailpit only;
there is no SMTP relay or SES configuration for any other environment.

## Wired in

None.

## Pending

Each of these needs a signed DPA, its region and sub-processors recorded here, and a review of this file
before the code that uses it is merged.

| Processor | Purpose | Personal data it will touch | Region | Status |
| --- | --- | --- | --- | --- |
| AWS SES (ADR 0007) | Sending e-mail from `cmd/notifier` | Recipient e-mail address; clinic name; a link into the app (invite links carry a one-time token in the URL fragment). No clinical content. Patient e-mails go only to the patient | EU region, to be chosen | Not wired in. Needs the DPA; until then only SMTP to Mailpit exists, locally |
| Hosting provider (not chosen) | Compute, managed Postgres, object store, backups, KMS | Everything the platform stores: accounts, sessions, audit events, the directory, and from later features health data (special category, Art. 9) | EU/EEA | Not chosen. Needs the DPA, encryption at rest, and the KMS decision for `TOTP_KEY` (`docs/accounts/SPEC.md`) |
| LiveKit host (ADR 0006) | Running the self-hosted LiveKit server and TURN | Live consultation audio and video (health data), participant identities as opaque ids | EU/EEA | Not deployed. If we self-host on the hosting provider, it is covered by that DPA; a hosted LiveKit Cloud or separate TURN service would be its own processor |

A speech-to-text engine for live transcription (ADR 0006) is not chosen; a hosted engine would be a
processor holding patient audio and needs a DPA, an entry here and explicit SPEC approval first.

## Not processors

- Mailpit, the compose Postgres and Redis: local development only, synthetic data only.
- GitHub (source code and CI): holds no personal data. Test data is synthetic (`@example.test` addresses,
  invented names); production data never reaches CI or a developer machine.
