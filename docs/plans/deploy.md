# Deployment: one Hetzner Cloud server

Status: planned, 2026-11-11; wave 3 added the same day. Decision record: docs/adr/0011-single-server-deployment.md. Nothing is deployed by
this plan's run: no server is touched, no secret is set.

## Not decided yet: configuration, never hard-coded

| Placeholder | What | Where it is set |
| --- | --- | --- |
| `SALUS_DOMAIN` | The domain; the stack serves the apex, `app.`, `api.`, `livekit.` and `turn.` under it | `.env` on the server (GitHub secret or variable) |
| `SALUS_SERVER_TYPE` | The Hetzner server type (sizing floor below) | Chosen when the server is created |
| `SALUS_ACME_EMAIL` | Let's Encrypt's contact address | `.env` |
| `BACKUP_*` | The Storage Box host, user, port, path and the backup's `age` public key | `.env` / the bootstrap's config |
| `SSH_PORT` | The SSH port `ufw` allows (default 22) | The bootstrap's config |
| `GHCR_OWNER` | The image namespace (default `shmeeshmee`) | Workflow env |

Sizing floor (not measured, ADR 0011): ClamAV holds its signature database in memory (ClamAV's documentation asks for
several GB on a host with clamd; `ConcurrentDatabaseReload no` keeps it from doubling during a reload), LiveKit and
the agent scale with the calls. 4 vCPU and 8 GB is the smallest sensible start; the memory
limits in `docker-compose.prod.yml` are variables with defaults for 8 GB.

## Work packages

```
Step 0 (one PR): the owner's decisions recorded, the review queue's order, ADR 0011, this plan
Wave 1, in parallel: DEP-1 images   DEP-2 stack   DEP-3 server
Wave 2, after all three are merged: DEP-4 deploy
Merge order in wave 1: DEP-1, DEP-2, DEP-3, main merged into the next branch first (no force-push)
Wave 3, in parallel: DEP-5 luks   DEP-6 smtp   DEP-7 ops   (merge order DEP-5, DEP-7, DEP-6)
```

Each wave-1 package runs in its own git worktree and branch, with its own compose project (`-p salus-dep1`, …) and
no published host ports when it runs the local stack.

### DEP-1 images (branch `dep-images`)

- `.github/workflows/images.yml`: builds every production image and pushes it to GHCR, only after CI passed for
  the commit on `main`; layer cache (GitHub Actions cache); an SBOM per image; a Trivy scan that fails on a
  critical vulnerability. Tags: the full commit SHA and `main`.
- Images: `salus` (every Go binary on distroless, `build/package/Dockerfile`, unchanged in shape), `salus-web`
  (`build/package/web.Dockerfile`: `web/` built, served by unprivileged nginx on 8080 with the SPA fallback),
  `salus-landing` (`build/package/landing.Dockerfile`), `salus-caddy` (`build/package/caddy.Dockerfile`: Caddy built
  with xcaddy and the `layer4` module, pinned versions).
- Contract for DEP-2 and DEP-4: image names `ghcr.io/${GHCR_OWNER}/salus`, `…/salus-web`, `…/salus-landing`,
  `…/salus-caddy`; the web image listens on 8080 and serves only static files; the Caddy image reads
  `/etc/caddy/Caddyfile` and keeps its state in `/data`; the workflow's name is `Images`.

### DEP-2 stack (branch `dep-stack`)

- `deployments/production/docker-compose.prod.yml` (project `salus`) on the GHCR images, tags from `SALUS_TAG`.
- `Caddyfile`: HTTPS for the apex (landing), `app.` (web, and the API paths of `web/vite.config.ts` to the api),
  `api.` (the api), `livekit.` (signalling); the `layer4` route that terminates TLS for SNI `turn.` and forwards it to
  LiveKit's TURN listener (`turn.external_tls: true`, the listener off 443 inside the network).
- `livekit.yaml` for production: `node_ip` or the external IP, TURN on behind Caddy's TLS (`external_tls`), no
  `allow_restricted_peer_cidrs`, log level `warn`, keys and webhook from the environment.
- Postgres with a persistent volume, Redis, Garage, ClamAV, the agent and the workers; memory limits per service,
  healthchecks, restart policies, rotated logs (`json-file`, size and count).
- `.env.production.example`: every variable the stack reads, one comment each; secrets come from GitHub Actions
  secrets (kept in Bitwarden).
- Only 80, 443, 443/udp and LiveKit's TURN and media ports bind publicly.

### DEP-3 server (branch `dep-server`)

- `deployments/production/bootstrap/`: an idempotent script for a fresh or an existing Ubuntu 24.04 server: the
  `deploy` user with an SSH key only, root and password login off, `ufw` with SSH and DEP-2's ports, `fail2ban`,
  unattended security upgrades, Docker, the `/opt/salus` layout. Leaves other compose projects alone.
- Nightly Postgres backups to a Hetzner Storage Box (configurable), encrypted with `age`, 14 days kept, and a
  documented restore drill.

### DEP-4 deploy (orchestrator, after wave 1)

- `.github/workflows/deploy.yml`: on a successful `Images` run of `main`, and by hand with a SHA. Environment
  `production` with the owner as required reviewer. SSH to the server, write the release's tags, pull, `migrate` as
  a one-shot, `docker compose up -d`, wait for the healthchecks, roll back to the previous tags on failure (without
  running `migrate`: forward-only migrations must stay compatible with the release before them).
- `docs/deploy/README.md`: the first-time setup step by step: what to create on Hetzner and GitHub, the secrets,
  the DNS records.

### Wave 3: the owner's decisions of 2026-11-11 (ADR 0011 "Before real patient data")

Three packages in parallel, each in its own worktree, branch and compose project (`-p salus-dep5`, …). Merge order:
DEP-5, DEP-7, DEP-6, main merged into the next branch first (no force-push). They share a few files (the restore
drill, `.env.production.example`, `docs/deploy/README.md`): each edits only its own part, and the merge resolves
the rest. DEP-7's alert e-mail does not wait for DEP-6: it sends with `curl` from the host through the same `SMTP_*`
variables (`SMTP_ADDR`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_TLS`, `SMTP_FROM`) that DEP-6 adds for the
notifier.

- **DEP-5 encrypted volume (branch `dep-luks`).** A documented one-time setup of a LUKS2 Hetzner Volume holding
  Postgres's and Garage's data and the key file (`BINARY_KEK`, `TOTP_KEY`, root-only); an unlock script the owner
  runs after a reboot; the bootstrap refuses automatic reboots and swap; the deploy refuses to start when the volume
  is not mounted, and refuses an env file that holds the keys; compose gives the keys to the containers from the key
  file. The restore drill restores onto the volume.
- **DEP-6 e-mail relay (branch `dep-smtp`).** An SMTP sender with authentication and TLS, a real from address and
  `PATIENT_APP_URL` from config, for any provider (Brevo or Mailjet); test first. SPF, DKIM and DMARC records for the
  domain in docs/deploy/README.md.
- **DEP-7 operations (branch `dep-ops`).** A nightly `age`-encrypted Garage backup to the Storage Box beside the
  Postgres dump, 14 days kept, in the restore drill; an alert e-mail when a backup fails or is missing; the HIGH
  vulnerabilities in `golang.org/x/net` and Caddy cleared where a fix exists; the flaky keyboard drag test in
  `web/src/Builder.test.tsx` fixed at its cause (no sleeps or retries), shown by 20 runs.

## Checks per PR

`make lint`, `make test`, `make archcheck`, `make e2e`; plus, for these packages, `actionlint` on the workflows,
`shellcheck` on the scripts, `docker compose config` on the production file, and a local build of each image.
GitHub Copilot's review quota is used up until 2026-12-01: each PR is reviewed by a fresh subagent that has not seen
the work.
