# 9. Docker is the only host requirement

Status: accepted, 2026-10-20

## Context

Toolchains installed on a developer machine drift from CI and from each other, and a version manager on
the host is one more thing to install and keep in step. The stack (Postgres, Redis, LiveKit, Mailpit)
already runs in Docker.

## Decision

The host needs Docker and make only. Every `make` target runs its tool in a compose service: `tools`
(`build/package/dev.Dockerfile`: Go 1.27.2; sqlc, goose and golangci-lint pinned as `tool` lines in a
separate `tools/go.mod` and run with `go tool -modfile=tools/go.mod`, so the app's `go.mod` holds only what
the app needs), `node` (24.22.0)
and `flutter` (3.47.7, built from the official SDK archive), the last two behind compose profiles. Versions
are image tags and `tool` lines in `tools/go.mod`. Tests use the compose Postgres and Redis through
`TEST_DATABASE_URL` and `TEST_REDIS_URL`; there is no testcontainers fallback. CI runs the same `make`
targets. Claude Code sessions run checks only through `make`.

## Consequences

- Local, CI and agent runs use identical toolchains; upgrading one is a tag change in one place.
- The first run pulls images and fills the module and build caches; later runs reuse the named volumes.
- Files the containers write into the repo are owned by the host user (`HOST_UID`/`HOST_GID` from make).
- Running the mobile app on a device or emulator still needs a host Flutter SDK; that is a manual step,
  not a check.
