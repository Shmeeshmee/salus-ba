# Bootstrap progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Go foundation | done 2026-10-20 | see `git log` |
| 1b Docker-only toolchain | done 2026-10-20 (committed after 2) | see `git log` |
| 2 Migrations | done 2026-10-20 | see `git log` |
| 3 API binary | not started | |
| 4 Redis and jobs | not started | |
| 5 Local stack | not started | |
| 6 Quality gates | not started | |
| 7 Web | not started | |
| 8 Mobile | not started | |
| 9 CI | not started | |

## Decisions

- 2026-10-20, owner: `claude-setup/` moved to `.claude/`; database-migrations skill switched to goose with a
  migration immutability check; sqlc output `internal/<domain>/<domain>db/`; tests use env URLs or one
  container per package with template databases; private GitHub repo with protected `main`; native toolchain
  pinned in `.mise.toml`; `build/ci/` dropped for `.github/`.
- 2026-10-20, package 1: `config.Load()` reads `os.Getenv` itself and is tested with `t.Setenv`, so
  the environment is read only in `internal/config` (not even `main` passes `os.Getenv`).
- 2026-10-20, package 1: no `DecodeJSON` yet; nothing reads a request body in this plan (CLAUDE.md: no
  unused code).
- 2026-10-20, package 1: `pgtest` keeps the server handle in one package variable set by `Main` before
  any test runs; TestMain has no other way to reach the tests. `pgtest` reads `TEST_DATABASE_URL`
  directly: it is a test helper, not application config (the forbidigo rule in package 6 exempts it).
- 2026-10-20, package 2: River's schema is versions 002–008 from `river migrate-get` (River v0.49.0), one
  goose file per River version, verbatim. Version 001 (River's own migration ledger) is left out, as River
  recommends when another tool runs migrations; goose's ledger tracks the files. A River upgrade adds
  its new versions as new files.
- 2026-10-20, package 2: `pgtest.Main(m, migrations)` takes the goose files as an `fs.FS` (nil for an
  empty template), so `pkg/` stays free of product code; product tests pass `migrations.FS`.
  `pgtest.NewDatabaseURL` added for code that connects by URL.
- 2026-10-20, package 2: `cmd/migrate create` numbers files itself (next `NNNNN`): goose's `Create`
  numbers by timestamp unless a package-level switch is flipped.
- 2026-10-20, owner: nothing runs on the host but Docker and make (ADR 0009). mise and `.mise.toml`
  removed; toolchains pinned as image tags; tests use the compose Postgres and Redis; CI and Claude Code
  run checks only through `make`. Supersedes the mise and testcontainers decisions above. Added as package
  1b; it lands after package 2 because 2 was already pushed (owner chose not to rewrite history).
- 2026-10-20, package 1b: Flutter image built from the official 3.47.7 SDK archive; the newest
  `ghcr.io/cirruslabs/flutter` tag is 3.44.0.
- 2026-10-20, package 1b: Postgres and Redis publish no host ports; another project's container held
  5432 on the owner's machine.

## Waiting on people

- Branch protection on `main`: GitHub refused it (HTTP 403, "Upgrade to GitHub Pro or make this
  repository public") for a private repo on a free account. Owner is upgrading to GitHub Pro (2026-10-20);
  retried before package 2, still refused. Then: PR required, no force pushes, no deletions; CI checks
  required in package 9. Until then only `chore/bootstrap` is pushed.

## Log

### 2026-10-19 — plan

CLAUDE.md updated for the settled decisions; ADRs 0001–0008 written; this plan written. Waiting for owner
approval before package 1.

### 2026-10-20 — plan approved

Owner answered the open questions (see Decisions). mise 2026.10.14 installed; Go 1.27.2, Node 24.22.0 and
Flutter 3.47.7 installed through `.mise.toml`.

### 2026-10-20 — package 1, Go foundation

Built: `go.mod` (go 1.27, toolchain go1.27.2), `internal/config`, `pkg/clock` (`Clock`, `System`, `Fixed`),
`pkg/httpx` (`JSON`, `InternalError`), `pkg/postgres` (`Open`, `Querier`, `Transactor.WithTx` with nested
join and rollback on error or panic, `Conn`), `pkg/postgres/pgtest` (`Main`, `NewDatabase`), `.gitignore`.

Test first: each package's tests were run against an empty package and failed on the undefined API, then
passed with the code. `pgtest` failed once on a real bug (the quoted name was used as the connection's
database); fixed by using plain lowercase names.

Checks: `gofmt -l .` empty; `go vet ./...`, `go build ./...` pass; `go test ./...` passes both ways:
container per package (3.7 s wall, warm image) and `TEST_DATABASE_URL` on a shared Postgres 18.6 with
`-race` (2.3 s); no test databases left on the shared server afterwards. `make` targets come in package 6.

### 2026-10-20 — package 2, Migrations

Built: `migrations/` (`migrations.FS`; `00001_extensions.sql` with `btree_gist` and `pgcrypto`;
`00002`–`00008` River versions 002–008), `cmd/migrate` (`up`, `status`, `create [-dir] <name>`; anything
else, including `down`, is rejected), `pgtest` migrates its template, `ownership.yaml`.

Test first: migrations, pgtest and cmd/migrate tests failed on the missing API first. One expectation was
wrong, not the code: River 007 drops `river_client` and `river_client_queue`; the test now asserts the
tables River ends with and that those two are gone. Fixed on the way: package 1's `TestOpen` used
`pool.Config().ConnString()`, which is the server URL, not the test database's; it now uses
`NewDatabaseURL`.

Checks: `gofmt -l .` empty; `go vet ./...` passes; `go test ./...` passes with containers and with
`-race` against a shared Postgres 18.6. Manual: `go run ./cmd/migrate status` (8 pending), `up` (8
applied, 2–26 ms each), `up` again (nothing applied), `status` (8 applied), `down` (rejected, exit 1).

### 2026-10-20 — package 1b, Docker-only toolchain

Built: removed `.mise.toml`; uninstalled mise with the owner's approval (`~/.local/bin/mise`,
`~/.local/share/mise`, `~/.cache/mise`, `~/.local/state/mise`, `~/.config/mise`; 4.6 GB). Added
`build/package/dev.Dockerfile`, `build/package/flutter.Dockerfile`, `deployments/docker-compose.yml`
(`postgres`, `redis`; `tools`, `node`, `flutter` behind profiles; named volumes for module, build, lint,
npm and pub caches), `Makefile` (`test`, `tools-image`, `down`), README.md, ADR 0009. CLAUDE.md "Checks"
and the two skills now say: checks only through `make`, never host `go`/`npm`/`flutter`. The Stop hook
already calls `make arch-gate`, which will run in Docker; unchanged.

Test first: `TestMain_FailsClearlyWithoutTestDatabaseURL` re-runs the pgtest test binary without
`TEST_DATABASE_URL`. It failed first ("start container: ... rootless Docker not found"), then passed once
the testcontainers fallback was removed. `go mod tidy` dropped testcontainers and 50 indirect modules.

Go tools: `go get -tool` pinned sqlc v1.31.1, goose v3.29.0 and golangci-lint v2.15.0; each runs with
`go tool <name>` in the container. Their dependencies add about 280 indirect requirements to `go.mod` (module
graph only; none is linked into our binaries). `make test` takes `TESTFLAGS` (for example `-run`, `-count=1`).

Checks: `make test` passes with nothing on the host but Docker and make: run under `env -i` with
`PATH=/usr/bin:/bin` (no go, node, npm, npx, flutter, dart or mise found), `TESTFLAGS=-count=1`, 6.3 s
wall including container start. gofmt clean, `go vet ./...` and `go build ./...` pass in the container.
Files the container writes (go.mod, go.sum) are owned by the host user. Readings in the `tools` container,
warm caches, two runs each: whole suite 3.6–3.8 s wall with packages in parallel, 4.0–4.3 s with `-p 1`;
`pkg/postgres` alone 0.5 s (each test about 0.38 s, mostly the database clone). No contention from
packages sharing the compose Postgres.
