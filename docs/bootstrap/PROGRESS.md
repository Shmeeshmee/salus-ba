# Bootstrap progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Go foundation | done 2026-10-20 | see `git log` |
| 2 Migrations | not started | |
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

## Waiting on people

- Branch protection on `main`: GitHub refused it (HTTP 403, "Upgrade to GitHub Pro or make this
  repository public") for a private repo on a free account. Owner to choose: GitHub Pro/Team, or a public
  repo. Until then nothing enforces PR-only merges; I only push to `chore/bootstrap`.

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
