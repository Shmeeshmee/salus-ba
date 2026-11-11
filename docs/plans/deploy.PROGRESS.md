# Deployment: progress

The running record of docs/plans/deploy.md. Nothing has been deployed: no server exists yet, and no secret is set.
GitHub Copilot's review quota was used up until 2026-12-01: every PR was reviewed by a fresh subagent that had not
seen the work, and its valid findings were fixed before the merge.

## Step 0 (PR #62, merged 2026-11-11)

The owner's decisions recorded (consent withdrawal keeps the patient's signed documents; a term is approved once
Latin and the five staff languages are; the review queue shows needs changes first, built test first), ADR 0011,
this plan. Review: eight findings, all fixed (among them an entry shown twice when a later page repeated it, and
TURN/TLS moved to Caddy terminating TLS with LiveKit's `external_tls`).

## DEP-1 images (PR #63, merged 2026-11-11)

`.github/workflows/images.yml` (`Images`): on a push to main, a gate waits for that commit's go, web, mobile and
landing runs to succeed; each image is pushed by digest (untagged, SBOM attestation), scanned from the registry with
Trivy (pinned image, any CRITICAL fails, exceptions only in `.trivyignore.yaml` with an id, a reason and an expiry),
and only the scanned digest is tagged with the SHA; the `release` job moves `main` only when the commit is still
main's head. New images: `salus-web` (`build/package/web.Dockerfile`, nginx on 8080, CSP from `SALUS_DOMAIN` at
start), `salus-caddy` (xcaddy with caddy-l4 v0.1.2, non-root uid 10001). Review: two must-fix (re-runs moving
`main` back; `--ignore-unfixed` against the ADR) and eight others, all fixed. The first real run on main
(a7f47f3) built, scanned, pushed and released all four images.

## DEP-2 stack (PR #64, merged 2026-11-11)

`deployments/production/`: the compose file, Caddyfile (layer4 TURN/TLS, routes checked against
`web/vite.config.ts` by `scripts/check-caddy-routes.sh` in `make lint`, no client IPs in logs), LiveKit's production
config, Garage, the env example; `cmd/healthcheck` for the distroless api (test first). Smoke test of the whole
stack on this machine, including the agent's media and TURN relays hairpinning through the node IP on the bridge
network (works; host networking is the documented fallback). Review: no blocker; eight findings fixed.

## DEP-3 server (PR #65)

`deployments/production/bootstrap/`: the idempotent bootstrap and the encrypted nightly backup with its systemd
units; `docs/deploy/backup-restore.md`; `make deploy-check` (shellcheck, the bootstrap twice in an Ubuntu 24.04
container, the backup and a restore of the real migrated schema against a throwaway Postgres and SFTP server).
Review: six must-fix (two SSH lockout paths, ufw blocking existing listeners, the restore procedure) and ten
others, all fixed.

## DEP-4 deploy

`.github/workflows/deploy.yml` (`Deploy`): after an Images run whose `release` job succeeded, or by hand with a
SHA (migrate optional); nothing runs until `DEPLOY_ENABLED` is `true`. Environment `production` with the owner as
required reviewer. `deployments/production/deploy.sh` on the server: check and pull the new release, switch files,
env and tag, `up -d --wait` (migrate through `up`), and on failure the previous release back without migrate;
the workflow then checks the public `/healthz` endpoints and rolls back if they fail. Tests:
`deployments/production/test/deploy-test.sh` (a fake docker: deploy, rollback, failed rollback, first deploy,
`--no-migrate`, refusals, pruning that leaves other images alone, a service that does not stay up, the lock, the
detached start), in `make deploy-check`. Review: one must-fix (a dropped SSH connection or a cancelled run could stop
the deploy half-way with no rollback: it now runs detached on the server under a lock, and the workflow follows its
status) and nine others, fixed (among them: services that crash after `--wait` fail the deploy; the workflow refuses
an environment without a required reviewer; the env file's line endings, quotes and image owner are checked). `docs/deploy/README.md`:
the first-time setup step by step. The landing image's nginx no longer writes an access log.
