# 11. One Hetzner Cloud server with Docker Compose, deployed from GitHub Actions

Status: accepted, 2026-11-11 (docs/plans/deploy.md); amended the same day by the owner's decisions under "Decided
before real patient data". The domain, the server type and the points under "Open before real patient data" are
not decided yet; everything that depends on them is configuration, not code.

## Context

Salus has run only on developer machines (`deployments/docker-compose.yml`). The first customers are a few clinics
in Bosnia and Herzegovina; the load is small and not measured yet (docs/consultation/SPEC.md §8 "Production
sizing"). The stack is one application in several processes (ADR 0002) plus Postgres, Redis, LiveKit (ADR 0006),
Garage and ClamAV (ADR 0010). CLAUDE.md asks for the simplest thing that works until a measurement says otherwise,
data in the EU/EEA, TLS everywhere, encryption at rest, and keys for files from a KMS.

## Options

| | One server, Compose | Kubernetes (k3s or managed) | Managed services (PaaS, managed Postgres) |
| --- | --- | --- | --- |
| Moving parts | The compose file we already run, a reverse proxy, a bootstrap script | A cluster, manifests or charts, an ingress, a registry pull secret per namespace | One vendor per service, each a processor with its own DPA |
| Fits the load | Yes, by a wide margin for a pilot | Built for many nodes we do not need | Yes |
| LiveKit | Host ports for media and TURN, as LiveKit's own single-VM guide | Host networking on a pinned node, the same in the end | Not offered; self-hosted anyway |
| Cost | One server and a Storage Box | The same server plus the cluster's overhead | Several monthly bills |
| Way back | Move the compose project to a bigger server, or split it later | Back to compose, rewriting the manifests | Self-host again, a migration per service |

Hetzner Cloud (the owner's choice): EU data centres (Helsinki, Finland, the owner's choice of location), a DPA
under GDPR Art. 28 concluded in the console, no managed Postgres and no KMS.

## Decision

- **One Hetzner Cloud server**, Ubuntu 24.04, location Helsinki (`hel1`), server type `SALUS_SERVER_TYPE` (not chosen:
  the plan's sizing notes give the floor). Docker Engine and the compose plugin from Docker's apt repository.
- **One compose project, `salus`, in `/opt/salus`** (`deployments/production/`): Caddy, the Go binaries (`api`,
  `worker`, `scheduler`, `notifier`, `agent`, and `migrate` as a one-shot), the web app, the landing page, Postgres,
  Redis, LiveKit with its built-in TURN, Garage and ClamAV. Every service has a memory limit, a restart policy and
  rotated logs, and a healthcheck except the River binaries (`worker`, `scheduler`, `notifier`, `agent`): they listen
  on nothing, a fatal error ends their process and the restart policy starts it again, so a liveness endpoint in each
  would be code for no measured need (deployments/production/README.md "Healthchecks"; amended by DEP-2). Other
  compose projects on the server are left alone: every command names the project.
- **Images** are built by GitHub Actions only after CI passed on `main`, pushed to GHCR as
  `ghcr.io/shmeeshmee/salus-<image>` tagged with the commit SHA and `main`, with an SBOM, and scanned with Trivy
  (critical vulnerabilities fail the build). The server pulls by SHA, never by `main`.
- **Public hostnames** under one domain, `SALUS_DOMAIN` (not chosen): the apex for the landing page, `app.` for the
  web app and the API on its own origin (the same paths as the Vite proxy in `web/vite.config.ts`), `api.` for the
  mobile app, `livekit.` for LiveKit's signalling (`wss://`), `turn.` for TURN over TLS. Caddy obtains and renews
  every certificate (Let's Encrypt).
- **TURN over TLS on 443** (settled in docs/consultation/SPEC.md §8): Caddy is built with the `layer4` module,
  terminates the TLS connections whose SNI is `turn.SALUS_DOMAIN` with the certificate it manages for that name, and
  forwards the plain TCP stream to LiveKit's TURN listener (inside the compose network, not 443), which runs with
  `turn.external_tls: true`. Every other connection on 443 goes on to Caddy's HTTP server. LiveKit never holds a
  certificate, so a renewal needs no restart. The layer4 route has to be checked against LiveKit's current
  self-hosting guide when it is built (DEP-2).
- **Public ports**: 80/tcp and 443/tcp (Caddy, with TURN/TLS behind SNI), 443/udp (HTTP/3), 3478/udp (TURN/UDP),
  7881/tcp and 7882/udp (LiveKit's WebRTC media, one UDP port with muxing), and SSH. Nothing else binds to a public
  address: Postgres, Redis, Garage, ClamAV, the API and LiveKit's API port are on the compose network only. `ufw`
  allows exactly these; Docker's published ports are kept to the same list (Docker bypasses `ufw` for published
  ports, so the compose file is the real boundary and the bootstrap checks it).
- **Postgres in compose**, its data on the encrypted volume (below, "Decided before real patient data"), with a nightly `pg_dump` (custom format), encrypted on the server
  with `age` to a public key whose private half is kept offline (Bitwarden), and copied over SFTP to a Hetzner
  Storage Box (configurable, `BACKUP_*`); 14 days kept. A restore drill is documented and run before the first real
  data. A restore needs the dumps' `age` private key and the files' `BINARY_KEK`, both kept in Bitwarden. Garage's
  data (the attachments) is not backed up by this decision (open point 7).
- **Deploys** from GitHub Actions on every successful image build of `main`, and by hand (`workflow_dispatch` with a
  SHA): the `production` environment needs the owner's approval; the job connects over SSH as the `deploy` user,
  writes the release's image tags, pulls, runs `migrate` as a one-shot, runs `docker compose up -d`, waits for every
  healthcheck, and on a failure puts the previous release's tags back and brings them up without running `migrate`.
  Migrations are forward-only (ADR 0004): a rollback runs the previous code against the new schema, so a migration
  must stay compatible with the code before it (expand, then contract in a later release). Nothing checks that
  automatically yet; the reviewer of a PR with a migration does.
- **Secrets** live in the owner's Bitwarden and are copied into GitHub Actions secrets of the `production`
  environment; the deploy job writes them to `/opt/salus/.env` (mode 600, owner `deploy`) on each deploy. `BINARY_KEK` and `TOTP_KEY` are the
  exception: they live in a key file on the encrypted volume (below). Nothing
  secret is committed; `deployments/production/.env.production.example` lists every variable.
- **The server** is set up by an idempotent bootstrap script (`deployments/production/bootstrap/`): a `deploy` user
  with an SSH key only, no root login and no password login, `ufw`, `fail2ban`, unattended security upgrades,
  Docker, the `/opt/salus` layout and the backup timer.

## Decided before real patient data (owner, 2026-11-11)

Five of the points that were open when this ADR was accepted are settled; the work is in docs/plans/deploy.md
(wave 3: DEP-5, DEP-6, DEP-7).

- **Encryption at rest: a LUKS-encrypted Hetzner Volume, unlocked by hand.** Postgres's and Garage's data live on a
  Hetzner Volume attached to the server and encrypted with LUKS2 (`cryptsetup`); the server's own disk holds no
  personal data: no swap (the bootstrap checks), and container logs carry none (CLAUDE.md). The
  passphrase is in the owner's Bitwarden only, never on the server or in GitHub. After every reboot the owner unlocks
  and mounts the volume by hand with a script, then starts the stack; until then nothing that holds data runs, and
  the deploy refuses to start when the volume is not mounted. **Automatic reboots stay off**
  (`UPGRADE_AUTO_REBOOT=no`, and the bootstrap refuses `yes`): a kernel update that needs a reboot waits for the
  owner, who reboots and unlocks in one sitting.
- **The keys without a KMS: a root-only file on the encrypted volume.** `BINARY_KEK` and `TOTP_KEY` live in one key
  file on the LUKS volume, owned by root and readable by no other account on the host, and reach only the containers
  that need them. They are not in `/opt/salus/.env`, not in the `SALUS_ENV_FILE` GitHub secret, and not in the
  release directories; the deploy refuses an env file that contains them. Bitwarden keeps the only other copy, for a
  restore. Limits, accepted for the pilot: the `deploy` user is in the `docker` group and therefore root-equivalent,
  so the file protects the keys from the env file's copies (GitHub, releases), from a copy of the server's disk and
  from a detached volume, not from someone who already has `deploy` or root on the running server. This is the
  exception to CLAUDE.md's "keys from the KMS" for the single-server deployment; a KMS replaces it when the
  deployment outgrows one server or a customer asks for one.
- **Plaintext between containers on the one host.** Postgres, Redis, Garage, ClamAV, LiveKit's API and the Go
  binaries talk without TLS over the compose network, a Docker bridge that never leaves the machine; everything that
  leaves the host is TLS (Caddy, TURN over TLS, SMTP, SFTP to the Storage Box, which also carries only `age`-encrypted
  files). This is the exception to CLAUDE.md's "TLS inside the cluster" for one host: a second server, or any
  service moved off this one, brings TLS on that link with it.
- **E-mail through an EU SMTP relay, Brevo or Mailjet.** The notifier sends through an SMTP relay with
  authentication and TLS (STARTTLS or implicit TLS), configured by `SMTP_*` and provider-agnostic: Brevo (Sendinblue
  SAS, France) and Mailjet (Mailjet SAS, France) both qualify, and the choice is configuration. Whichever is used is
  a processor: its DPA is signed and it is recorded in docs/privacy/processors.md before the first e-mail to a real
  person. SES (ADR 0007) is not used. The sending domain gets SPF, DKIM and DMARC records (docs/deploy/README.md).
- **Live speech-to-text and the language model stay on the fakes in production** (`STT_ENGINE` and `LLM_ENGINE`
  unset or `fake`) until their DPAs are signed and recorded (docs/privacy/processors.md): until then live
  transcription, translation, suggestions and drafting do not work in production. The production env example says
  so, and nothing in the deploy sets a real engine.

## Open before real patient data

These do not block building or deploying the stack with synthetic data; each blocks real patient data, and each is
the owner's decision.

1. **Hetzner's DPA** signed (console, "Data Processing Agreement") and Hetzner moved from "Pending" to "Wired in" in
   docs/privacy/processors.md, with the Storage Box; and the chosen SMTP relay's DPA, the same way.
2. **Backups beyond Postgres and their retention.** Garage's attachments are not backed up: losing the server loses
   the files the database rows point to. The 14 days of dumps keep erased data for up to 14 days after an erasure;
   the retention rule and the DPIA note (docs/privacy/dpia.md) need the owner's confirmation.

## Consequences

- One machine is one point of failure: a server loss means a restore from the night's backup onto a new server
  (the drill measures how long). Acceptable for a pilot; revisit with the load test of SPEC §8.
- LiveKit shares the machine's CPU and bandwidth with everything else; the sizing in the plan is a floor, not a
  measurement.
- Moving to another provider or to several servers keeps the images, the compose file's shape and the workflows;
  only the bootstrap and the deploy target change.
