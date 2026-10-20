# Bootstrap progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Go foundation | not started | |
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

## Log

### 2026-10-19 — plan

CLAUDE.md updated for the settled decisions; ADRs 0001–0008 written; this plan written. Waiting for owner
approval before package 1.

### 2026-10-20 — plan approved

Owner answered the open questions (see Decisions). mise 2026.10.14 installed; Go 1.27.2, Node 24.22.0 and
Flutter 3.47.7 installed through `.mise.toml`.
