# Bootstrap: repository skeleton

Status: approved decisions, 2026-10-19. Branch `chore/bootstrap`, one commit per package, one PR at the end.

## 1. Goal

A repository where every later feature can start with a failing test: the Go foundation, migrations, the
binaries, the local stack, the quality gates, the two client apps and CI. No business domain is built.

## 2. Settled decisions

See the ADRs in `docs/adr/` (0001 to 0008). In short:

| Topic | Decision |
| --- | --- |
| Priorities | Performance (measured), then simplicity, then elegance. GDPR never traded |
| Go | Latest stable (1.27), module `github.com/Shmeeshmee/salus`, `net/http` routing (1.22+ patterns), `log/slog` |
| Database | Postgres 18, pgx v5, sqlc (the one query generator) |
| Migrations | goose, SQL in `migrations/`, forward-only, embedded, run by `cmd/migrate`; River's schema included |
| Jobs | River (Postgres), enqueued in the causing write's transaction |
| Redis | Temporary state only, every key with a TTL |
| Video | Self-hosted LiveKit behind a port in `consultation`; `cmd/agent` for transcription. Stubs only now |
| E-mail | SES (EU) in production, Mailpit locally, behind a port in `notification` |
| Web | React + TypeScript (Vite) in `web/`, practitioners and admins |
| Mobile | Flutter in `mobile/`, org `ba.salus`, iOS and Android, patients only |
| Contract | `api/openapi.yaml`; TypeScript client by openapi-typescript, Dart client by openapi-generator |
| Binaries | `api`, `migrate`, `worker`, `scheduler`, `notifier`, `agent`, composition only |
| Repo | GitHub monorepo, GitHub Actions with path filters, Dependabot |

Decisions taken in this plan (reopen only with the owner):

- Everything runs in Docker (ADR 0009): the host needs Docker and make only; every `make` target runs in a
  compose service (`tools`, `node`, `flutter`), locally, in CI and in Claude Code sessions.
- Test databases are fast and never skip. `pkg/postgres/pgtest`: a test package calls
  `pgtest.Main(m, migrations)` from `TestMain`, which uses the server at `TEST_DATABASE_URL` (the compose
  Postgres inside the `tools` container) and fails with a clear message when it is unset. It migrates a
  template database once; `pgtest.NewDatabase(t)` then runs `CREATE DATABASE ... TEMPLATE <template>` so each
  test has its own database and tests run with `t.Parallel()`. Dropped on cleanup. The template name is unique
  per package run so parallel packages sharing one server do not collide. `pkg/redis/redistest` does the same
  with `TEST_REDIS_URL`, giving each test its own logical database (key prefix when databases run out).
- sqlc: one `sqlc.yaml` at the root, one entry per domain, output `internal/<domain>/<domain>db/`; only that
  domain's `repository_postgres.go` imports it (archcheck), and it maps rows to the domain model. No domain
  exists yet, so `sqlc.yaml` starts with no entries.
- Migrations are immutable once on `main`: `scripts/check-migrations.sh` diffs `migrations/` against
  `origin/main` (or `main`) and fails on any modified, renamed or deleted file. Run by `make lint` and CI.
- Toolchain, pinned as image tags: `build/package/dev.Dockerfile` (`golang:1.27.2-trixie`),
  `node:24.22.0-trixie-slim`, `build/package/flutter.Dockerfile` (Flutter 3.47.7 from the official SDK
  archive; no maintained image publishes 3.47.7). `go.mod` has `go 1.27` and `toolchain go1.27.2`; the
  image sets `GOTOOLCHAIN=local`.
- Compose: Postgres and Redis publish no host ports (another project on a developer machine may hold 5432);
  toolchain services reach them on the compose network. The Vite dev server publishes 5173. Running the
  mobile app on a device or emulator is manual (README.md).
- GitHub: private repo `Shmeeshmee/salus`; `main` protected (PR required, CI must pass; the required checks
  are added once the workflows exist in package 9).
- Migrations are embedded by a small Go package at `migrations/` (`migrations.FS`); the CapabilityStatement
  by one at `api/fhir/` (`fhir.CapabilityStatement`). Both are read-only `embed.FS` values.
- Middleware implementations (request id, access log, recover, timeout) live in `pkg/middleware`; they are
  mounted only in `internal/api/router`.
- Liveness `/healthz` returns 200 without touching dependencies. Readiness `/readyz` pings Postgres and Redis
  with a short timeout and returns 503 with a fixed JSON body naming the failing check (`postgres`, `redis`),
  never the error text.
- Container images: one multi-stage Dockerfile builds every binary into a distroless `static` image; the
  binary is chosen by the compose `command`.
- Go dev tools (sqlc, goose, golangci-lint) are pinned as `tool` lines in `go.mod` and run with `go tool` in
  the `tools` container; goose is also the library behind `cmd/migrate`.

## 3. Personal data

None. No table in this plan holds personal data; River's tables hold job arguments with opaque ids only (no
jobs exist yet). Logs carry request id, method, route pattern, status and duration; never query strings,
bodies, headers other than the request id, or client IP. Test data is synthetic. No processor is wired in:
Mailpit and LiveKit run locally; SES is not connected in this plan.

## 4. Out of scope

Any business domain (`identity`, `patient`, ...), authentication, TLS termination, production deployment
manifests, the speech-to-text engine choice, SES wiring, UI library choices, the HL7 validator content beyond
running it on the metadata golden file.

## 5. Done when

`make arch-gate`, `make lint` and `make e2e` pass on `chore/bootstrap`, CI is green on the PR, and
`PROGRESS.md` records every package.
