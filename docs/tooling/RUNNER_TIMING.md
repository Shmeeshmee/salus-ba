# Package-runner timing

Where the time of a package run goes, measured before and after the runner rules of 2026-11-13
(`tooling(runner): time limits, layered checks, no CI watching, cleanup`).

## How it was measured

- **Package-runner sessions**: the eight latest `package-runner` subagent transcripts
  (`~/.claude/projects/-home-mesha-github-com-salus/*/subagents/agent-*.jsonl`, terminology 1.1–3.1 and i18n
  2.1–3.1, 2026-11-07/08). Each tool call is timed from its `tool_use` to its `tool_result` and put in a step
  type by its command; the rest of the wall time is the model's own (reading, thinking, writing code).
- **Headless runs**: the `duration_ms` of the eleven `run-feature.sh` sessions of the questionnaire plan
  (`.claude/runs/questionnaire/*.session.json`, 2026-11-06). The older `.claude/runs/{accounts,booking,consultation}`
  folders have reports but no session JSON.
- **Orchestrating sessions**: the eight latest top-level sessions that ran packages, opened the PRs and
  handled CI and Copilot (2026-11-09 to 11-13, SAL-28…36, history, analytics, deploy). Background shells are
  timed from their launch to their completion notice; a shell with no completion notice by the end of the
  transcript is counted to the last event and marked "never ended". Waits on a person (`AskUserQuestion`,
  490 min in one session) are left out.

The scripts that produced the numbers are not committed; they read private transcripts.

## Before

### One package, inside the package-runner

| Step type | 8 interactive runs (sum, min) | Share | Notes |
| --- | ---: | ---: | --- |
| Model (read, think, edit) | 77 | 64% | 4.6–13.0 min per run |
| Fast checks (`go test ./pkg`, vitest, tsc, oxlint) | 26 | 22% | 0.4–10.6 min per run |
| Waits and polls (`until grep …; sleep`) | 8 | 7% | three loops of 2–3 min on a log line, one on `git status` text |
| Other shell (grep, sed, cat) | 6 | 5% | |
| Build and generate (`make sqlc`, `gen-clients`) | 2 | 2% | |
| git | 1 | 1% | |
| Docker stack up/down | 0 | 0% | the shared `salus` stack was always up |
| **Wall time per package** | **6–27 (median 15)** | | |

Headless `run-feature.sh` sessions (questionnaire 1–11): 5.5–51.8 min, median 15 min; the 51.8 min one
(package 8) did a browser walk with six full passes.

So a package itself takes minutes. The hours are spent around it.

### Around the package: the orchestrating session

| Step type | Measured over 8 sessions | Worst single case |
| --- | --- | --- |
| Session span (first to last event) | 4.4–25.3 h | 25.3 h (SAL-28…36) |
| Session active time (gaps < 15 min) | 1.3–7.1 h | 7.1 h |
| CI watching (`gh pr checks --watch`, `gh run watch`, poll loops on CI) | 431 min in 58 background shells + 57 min foreground; 208 `gh pr checks` calls in one session | 18 min, one shell |
| Copilot review waits and fix rounds | 32 min background + 30 min foreground (one session) | 15 min "wait up to 15 minutes for Copilot" |
| Full check set reruns (`make test`, `make lint` after each fix) | 67 `make test` and 72 `make lint` runs; 65 min foreground + about 235 min in background shells that ended | 20 × `make test` and 20 × `make lint` in one session |
| Other waits (`sleep N`, "recheck in 15 minutes") | 33 min foreground, 56 min background; 194 `sleep` calls | 15 min |
| Docker stack | per-worktree project names and ports typed by hand each time (`-p salus-dep57`, `SALUS_API_PORT=28080` …), shared `salus` project otherwise | |
| Background shells that never ended | 5 of 107: 957 min and 916 min (`deploy-check` + full checks in two worktrees), 133 min ("run every check again"), 117 min (checks for three PRs), 18 min (CI on PR 60) | 16 h |

Left on the host on 2026-11-13 20:55: the `salus` compose project held containers of two checkouts
(`salus` and `salus-gate`), up 20 h to 3 days.

### Where the hours go

1. Blocking on CI and Copilot in the session that opened the PR: 7–10 min per watch, repeated per push.
2. Running the full check set after every small fix instead of the affected packages.
3. Background shells with no timeout, waiting on a line that never came, kept alive until the session or the
   machine went away.
4. No budget: nothing stopped a package or a fix loop that kept going.

Also measured: the Stop hook (`make arch-gate` at the end of every turn) ran 575 times in these transcripts,
169 min in all (about 18 s each, most of it Go's test cache).

## After

### The check run

One small package through `/run-package` on a throwaway plan (`docs/runner-check`, package 1: a leaf
`pkg/runnercheck` with a table-driven test), in its own worktree, on 2026-11-13, while two other worktree stacks ran
test suites on the same host. The plan and its commit were not merged.

| Step | Before (typical package) | After (check run) |
| --- | --- | --- |
| Wall time of the package-runner session | 6–27 min, median 15 | 5.3 min (TIMINGS: build 0, full 4, record 0; total 5) |
| Fast layer per change | `go test ./pkg` by hand, or the full suite | `runner.sh fast`: vet, gofmt and test of `./pkg/runnercheck` only, 2 s, no test services started |
| Full check set | 2–20 runs per package | 1 run + 1 allowed rerun (see FLAKY); the rerun took 24 s, Go's cache re-ran only the failed package |
| Flaky check | rerun until green, unrecorded | `cmd/worker` `TestRun_StartsRiverAndStopsCleanly` (15 s start deadline under load) failed once, passed on the rerun, recorded as FLAKY |
| Docker stack | shared `salus` project, ports typed by hand | own project `salus-runner-check`, ports 2xxxx from the worktree name; up for the full run, `down` by cleanup |
| Waits and polls | 2–3 min `until grep` loops, no deadline | none needed; `runner.sh bg`/`wait` with a pidfile and a timeout for long commands |
| CI and Copilot | blocking 7–18 min per watch, per push | not watched: the PR session requests Copilot and ends |
| Left running afterwards | 5 of 107 background shells, up to 16 h | CLEANUP: killed 0; stack: down salus-runner-check |

The check package is far smaller than a real one, so its 5 minutes are not a like-for-like comparison with the median
15. What it shows is that the overhead the rules remove is gone: no CI wait, no repeated full set, no unbounded wait,
nothing left behind. The orchestration hours above come from the steps the rules now forbid.

### Found while checking

- A worktree's own Postgres (max 400 connections) refused the uncached suite once with `53300 too many clients`
  while three stacks ran tests at the same time; the one rerun passed. Recorded as flaky, not fixed here.
- Before this change a worktree session cleared the shared stack to get its own: `docker compose -p salus down`
  from another worktree removed the main checkout's stack (api, worker, Postgres, up for days) at 20:59 on the day
  of the measurement. Per-worktree project names remove the reason to do that.
- In a worktree, compose warns on every run that the shared cache volumes (`salus_gomod`, …) "already exist but
  were created for project salus". Expected: the caches are shared on purpose so a new worktree starts warm.
