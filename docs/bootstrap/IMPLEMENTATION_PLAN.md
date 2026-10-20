# Bootstrap implementation plan

Read `SPEC.md` first. Packages run in order; each is one commit `bootstrap(<n>): <summary>` on
`chore/bootstrap`.

## Rules for every package

- Go is test-first: write the failing test, run it and record the failure, then the least code to pass.
- Leave `gofmt -l` empty and `go vet ./...`, `go build ./...`, `make test` passing (once they exist).
- No personal data anywhere; nothing beyond opaque ids in a log line.
- Update `PROGRESS.md` at the end of the package, in the same commit.
- Pin every version (Go modules, images, npm, pub, Actions).

## Check sets

Every check runs through `make`, in Docker. Never `go`, `npm` or `flutter` on the host.

- **Go**: `make test`; gofmt, `go vet ./...` and `go build ./...` in the `tools` container (from package 6:
  `make lint`, `make archcheck`, `make arch-gate`).
- **Stack**: `make up`, `/readyz` returns 200, `make down`.
- **Web**: `make web-lint`, `make web-typecheck`, `make web-test`, `make web-build`.
- **Mobile**: `make mobile-analyze`, `make mobile-test`.

## Packages

### 1. Go foundation

`go.mod` (Go 1.27). `internal/config`: `Load() (Config, error)` with required and defaulted values, tested
with `t.Setenv`. `pkg/clock`: `Clock` interface, `System`, `Fixed`.
`pkg/httpx`: `JSON(w, status, v)`, `InternalError` (logs via `slog`, fixed body). A
JSON decoder is added with the first endpoint that reads a body. `pkg/postgres`: `Open` (pgxpool), `Transactor.WithTx` (nested call joins the outer
transaction, rollback on error or panic), `Conn(ctx, db)`. `pkg/postgres/pgtest.NewDatabase`. Repository-style
tests for `WithTx` commit, rollback and nesting against a real database. `pgtest`: one container per test
package from `TestMain`, template database migrated once, `CREATE DATABASE ... TEMPLATE` per test, parallel
tests. Until package 2 the template has no migrations.
Checks: Go.

### 1b. Docker-only toolchain

Added 2026-10-20 after packages 1 and 2 had landed; committed after package 2 (pushed history is not
rewritten). Remove `.mise.toml` and the host mise install. `build/package/dev.Dockerfile` (Go 1.27.2; Go tools
via `go tool`), `build/package/flutter.Dockerfile`; `deployments/docker-compose.yml` with `postgres`,
`redis`, and the `tools`, `node` (Vite dev server on 5173) and `flutter` services behind profiles, caches in
named volumes. `Makefile` with `test` (via `docker compose run --rm tools`) and `down`. `pgtest`, test first:
no testcontainers fallback; without `TEST_DATABASE_URL` the tests fail with a clear message. README.md,
ADR 0009, CLAUDE.md "Checks".
Checks: `make test` passes with only Docker and make on the host.

### 2. Migrations

`migrations/` with the embed package; `00001_extensions.sql` (`btree_gist`, `pgcrypto`); River's schema as
goose migrations. `cmd/migrate` with `up`, `status`, `create` (goose as a library). `pgtest` applies the
migrations. `docs/architecture/archcheck/ownership.yaml` lists River's tables under owner `river`.
Checks: Go; `go run ./cmd/migrate up` and `status` against a local Postgres.

### 3. API binary

`cmd/api` with graceful shutdown on SIGINT/SIGTERM. `internal/api/app.go` builds dependencies;
`internal/api/router` mounts `pkg/middleware` (request id, access log without personal data, recover,
timeout) and the routes. `GET /healthz`, `GET /readyz` (Postgres and Redis ping through small ports so the
handler test stubs them), `GET /fhir/R4/metadata` (golden-file test, `application/fhir+json`).
`api/fhir/CapabilityStatement.json` lists only `metadata`. `api/openapi.yaml` covers `/healthz` and `/readyz`.
Redis ping is wired in package 4; until then `/readyz` checks Postgres only and the test for Redis lands in 4.
Checks: Go.

### 4. Redis and jobs

`pkg/redis` (`Open`, `Ping`) tested against a real Redis (`redistest`: one container per package from
`TestMain`, a logical database or key prefix per test). `/readyz` gains the Redis check.
`cmd/worker` runs River with one no-op job (`noop`) and a test that inserts it in a transaction and sees it
complete. `cmd/scheduler`, `cmd/notifier`, `cmd/agent` start, log, and stop cleanly on signal, each with a
test of its `run(ctx)`.
Checks: Go.

### 5. Local stack

`deployments/docker-compose.yml` (Postgres and Redis exist since 1b) gains LiveKit (dev mode), Mailpit,
`migrate` (runs once), then `api`, `worker`, `scheduler`, `notifier`, `agent`. `make up`, `make down`. `build/package/Dockerfile` (multi-stage, distroless
static, non-root). `configs/.env.example` with synthetic local values only.
Checks: Go, Stack.

### 6. Quality gates

`Makefile` targets, each `docker compose run --rm <service> ...`: `cover` (80% floor), `lint`, `archcheck`,
`arch-gate`, `e2e`, `fhir-validate`, `migrate`, `migrate-status`, `migrate-create`, `sqlc`, `gen-clients`
(`test`, `down` since 1b; `up` since 5). `.golangci.yml` with the linters CLAUDE.md
lists. `tools/archcheck`: `pkg/` never imports `internal/`; forbidden package names; table ownership
(every `CREATE TABLE` in `migrations/` appears in `ownership.yaml`); `<domain>db` packages imported only by their
domain's `repository_postgres.go`; empty `allowlist.yaml`. `sqlc.yaml` (root, one entry per domain, none yet).
`scripts/check-migrations.sh` in `make lint`. `test/e2e/`: stack up, `/readyz` returns 200 (build tag `e2e`).
`make fhir-validate` runs the HL7 validator (Docker) on the metadata golden file.
Checks: Go, Stack, `make lint`, `make archcheck`, `make arch-gate`, `make e2e`, `make cover`.

### 7. Web

`npm create vite` (React + TypeScript) in `web/`, run in the `node` service; ESLint; `typecheck` script (`tsc --noEmit`); generated
client from `api/openapi.yaml` (openapi-typescript, plus `openapi-fetch` to call it); one page that calls
`/healthz` and shows the status. Vitest for that page. No UI library.
`make web-lint`, `web-typecheck`, `web-test`, `web-build`, `web-dev` (Vite on 5173).
Checks: Web, `make gen-clients` leaves no diff.

### 8. Mobile

`flutter create --org ba.salus --platforms ios,android mobile` in the `flutter` service; Dart client generated by openapi-generator
(Docker image) into `mobile/packages/salus_api`; one screen that calls `/healthz`, with a widget test that
stubs the client. `make mobile-analyze`, `mobile-test`. Device and emulator runs are manual (README.md).
Checks: Mobile, `make gen-clients` leaves no diff.

### 9. CI

`.github/workflows/go.yml` (path filters: `cmd/`, `internal/`, `pkg/`, `migrations/`, `api/`, `tools/`,
`go.*`, `Makefile`, `sqlc.yaml`, `build/`, `deployments/`): the same `make` targets as locally, through
compose (lint, archcheck, migration immutability, sqlc and client staleness, test against the compose
Postgres and Redis, cover). No toolchain is installed on the runner beyond Docker. `web.yml`, `mobile.yml` with their own filters. `.github/dependabot.yml` for gomod, npm,
pub, docker, github-actions. Push the branch and open the PR. Add the workflows' checks as required
status checks on `main`.
Checks: all of the above, CI green on the PR.

## Definition of done

Every package committed, `PROGRESS.md` complete, and `make arch-gate`, `make lint`, `make e2e` pass locally
and in CI.
