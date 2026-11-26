# Wave 7, readiness: the morning report

Run overnight 2026-11-25 to 26, unattended. Details are in `docs/clinic-cms/PROGRESS.md` (RD1 to RD6 and "Wave 7
integration") and in each package's report under `.claude/runs/cms/` (git-ignored).

## The pull request

https://github.com/Shmeeshmee/salus-ba/pull/126, "cms: wave 7, readiness: role matrix, load, accessibility, operations
drill, demo", from `feat/cms` (force-pushed with lease over wave 6's pre-squash commits, whose content was on main).
Copilot was requested with `gh pr edit --add-reviewer @copilot`.

CI: **green** on `886998a0`. All 14 checks passed: lint, test (with the new `make load-budget` step), archcheck,
generated, fhir-validate, e2e, web, mobile, apk, landing, and the path-filter jobs. No review has come: the Copilot
request no longer shows on the PR, which fits its quota being out until 2026-12-01. The PR is not merged.

Locally, `make check BASE=origin/main SETS=go,web,mobile,landing,e2e` passed on the merged tree, and again on the
final tree with the review's fixes (cover 89.7%, `make e2e` included). Nothing was flaky.

## Per package

All six finished on their first run; none was stopped.

| Package | Done | Commit | What it found |
| --- | --- | --- | --- |
| RD1 role matrix | done | `bcc08bc3` | One low hole (the desk could check in a restricted record's visit by its id), fixed. The integration found a second, narrower one and fixed it too. |
| RD2 load | done | `4368a1ba` | Range reads under row-level security read a clinic's whole year for one day, because `lower(period)` is not leakproof. A generated `start_at` column and its indexes fix it. |
| RD3 accessibility | done | `0aeb922f` | axe found nothing. The keyboard and touch checks found mint focus rings at 1.9:1, targets under 44 px on touch screens, a dialog that could not be closed, and unannounced errors. |
| RD4 operations drill | done | `ec4ce2fb` | The restore drill passed. The runbook missed closing sessions after a restore. Go 1.27.4 fixes an HTTP server crash reachable from `cmd/api`. There are four production wiring gaps. |
| RD5 demo | done | `7554b07c` | The seed works through the api in about 30 s and is idempotent. The invite job races the seed's own invite token, so the seed retries. |
| RD6 leftovers | done | `2f33eef8` | All six items done. Item 5 was the web plan booking's date field, not the patient app (see the decisions below). |
| Integration | done | `ff90bde0`, `886998a0` | The review found 8 low findings and none higher; the valid ones are fixed (below). |

## RD1: authorization holes

| Hole | Severity | Fixed |
| --- | --- | --- |
| `POST /clinic/appointments/{id}/arrive` let the desk and the admin check in a visit of a patient whose record is restricted (S7), by the visit's id. No data was shown, but the visit changed and counted in the statistics. | low | yes, test first (RD1) |
| The same route answered `not_in_person` for a restricted record's video visit, which tells that the hidden visit exists and is a video visit: the mode was checked before the restriction. | low | yes, test first (integration) |

The matrix found no other hole. Another clinic's and another patient's real ids get the same answer as unknown ids, and
return none of their data, on every route and FHIR interaction. Closed accounts, restricted records and revoked
consents show nothing on staff reads, except the admin's deliberate openings (the hand-over and the data-request
queue).

The integration also tightened the matrix itself. It had counted any 400 as a refusal. That hid four routes classed
"allowed" for roles the server refuses: `GET /clinic/patients` for the patient, and `POST /invites` and both imports
for the patient and the practitioner. With requests that pass validation, the server refuses each one correctly, so
these were not holes. Their classes are corrected, and a test holds them with real requests.

Open for you, from RD1:
1. Receipts can still be issued for a restricted record's visit: fiscal law against GDPR Art. 18.
2. A suspended clinic is hidden from its own staff and patients too (its locations, profile and FHIR Schedule), so the
   app cannot name a visit's location.

## RD2: hot paths before and after (p50, ms, on the busy year)

| Path | Before | After |
| --- | --- | --- |
| A week's free slots | 10.9 | 1.4 |
| The desk's day | 133.5 | 23.8 |
| A practitioner's week | 84.5 | 34.3 |
| The clinic's week (admin) | 739.0 | 139.2 |
| The patient's visits | 14.3 | 9.4 |
| Hourly slot generation, 500 schedules | 14,637 | 9,263 |
| Book a slot | 16.3 | 22.1 (slower; one run on a shared machine, within its budget) |
| Directory, receipts, receipts report, clinic statistics | unchanged (the cost is the size of the answer) | |

Statements per request: the desk's day went from 478 to 155, the clinic's week from 3,133 to 792. The p95 and p99 are
noisy (load average 9 to 24), so the CI budgets are on medians. The full table is in
`docs/clinic-cms/load/REPORT.md`.

Indexes added (migration `00087_start_at.sql`, plus a generated column `start_at = lower(period)` on both tables):
- `appointment_organization_start_at (organization_id, start_at)`: the desk's day, the calendar, the practice report.
- `slot_schedule_start_at (schedule_id, start_at)`: free slots, and the generation's comparison and insert.

Other fixes: the appointment search and the desk's day read each page's consents, restrictions, visits and profiles in
one statement each, and write the page's audit events in one insert. Slot generation skips the periods it already has.

Open for you:
1. A staff FHIR Appointment search without dates is unbounded. Should it be refused (`too-costly`) or paged?
2. `00087` rewrites `appointment` and `slot` under an exclusive lock (86 s on the busy year). That is fine before
   go-live; afterwards it needs a quiet hour.

## RD3: fixed, and what needs a design decision

Fixed:
- Every focus ring is now the brand green (6.3:1) instead of the mint.
- The medical terms search had no focus ring; it has one now.
- Every control is at least 44 px under `(pointer: coarse)`.
- The "Add a doctor" dialog with no other doctor in the clinic can be closed now: it takes the focus, closes on Escape
  and has a Cancel button.
- The dialog's Tab trap leaked past a disabled last button; it no longer does.
- Camera and microphone problems in the call, and a refused chat upload, are announced as alerts.
- On the analytics heatmap, the focused cell keeps its ring when it is also the cell shown.

The specs stay in `make e2e`: 26 screens and 12 dialogs, as admin, practitioner and receptionist.

Needs your decision:
1. The focus ring colour. The brand green was the smallest change that passes; the mint alone cannot be a ring.
2. Mint hover borders, the confirmed tag's border and the calendar's "today" underline are 1.9:1. They are decorative
   now, and must never be the only sign of a state.
3. On a phone, the header shows 9 to 12 links in wrapped rows, about 260 px of the admin's first screen. A menu button
   would give that space back.
4. Disabled buttons: RD6 gave them one grey look with a 3.5:1 outline. Confirm it.
5. Chart marks are 24 px on a phone, with "Show as table" as their touch equivalent. Larger marks would be a chart
   design change.

Not walked: the live call screen (it needs LiveKit and a started consultation) and the questionnaire file viewer.

## RD4: restore drill, vulnerabilities, placeholders

**Restore drill: passed** (`scripts/restore-drill.sh`, recorded in `docs/deploy/restore-drills.md`):
- 95 tables and 5,018 rows, every count equal; 3 objects, none missing on either side.
- Staff signed in again with password and TOTP. The attachment's SHA-1 matches, and the receipt PDF is byte-identical.
- Backup 15 s, restore 43 s.
- Runbook fixes: close every session after a restore (an old session answered 200), the erasure paragraph, and the
  bucket's contents.
- Migrations from empty to `00086`, and from a `00059` dump with data, passed with every count equal.

**Fixed:**
- Go 1.27.3 to 1.27.4: GO-2026-6859, an HTTP server crash reachable from `cmd/api`.
- postgres 18.6 to 18.7 in production (55 fewer HIGH).
- clamav 1.5.4 to 1.5.5 (libxml2).
- redis 8.10.3 repinned to its rebuilt digest (zlib).
- npm (3 lock files) and osv-scanner (mobile): 0 findings.

**Left, with reach:**

| Where | Finding | Reach |
| --- | --- | --- |
| Caddy | go-jose CVE-2026-116587 HIGH, no upstream fix | not reached (no JOSE use beyond ACME) |
| LiveKit 1.13.11 | x/net CVE-2026-78661 and -78669 HIGH, no LiveKit release with the fix | not reached (parses no HTML) |
| postgres 18.7 (Debian) | libxml2 CVE-2026-6653 CRITICAL, about 70 HIGH | low: compose network only, no XML |
| postgres 18.7 (gosu) | Go 1.24.6 stdlib CVE-2025-68121 CRITICAL | not reached (gosu only drops privileges) |
| redis 8.10.3 (Debian) | 51 HIGH, no fix | low: compose network only |

**Placeholders** (`docs/deploy/BEFORE-PRODUCTION.md` §2). Only `[OPERATOR EMAIL]` stops anything from starting:
- `[OPERATOR EMAIL]`: the notifier refuses to start outside dev.
- `[SELLER]`, `[SELLER TAX ID]`, `[SELLER ADDRESS]`, `[SELLER ACCOUNT]`: printed on membership invoices; nothing
  refuses.
- `[PRICE]`: the landing page's pricing, and the web reminders' SMS hint.
- `[APP URL]`: the landing page's sign-up link (`VITE_APP_URL`).
- `[YOUR EMAIL]`: the landing page's demo link.
- `[Hosting]` / `[Where data is stored]` and `[Certification]` / `[Audit or standard]`: the landing page's standards
  tiles.
- `[SUPPORT EMAIL]`: the mobile app (`--dart-define`).
- `PRIVACY_NOTICE_URL`: the mobile app (`salus.invalid` today).
- Plan prices: none seeded.

**Deploy blockers, not fixed** (production wiring was outside RD4; the exact lines are in BEFORE-PRODUCTION.md §1):
1. `docker-compose.prod.yml` passes the notifier neither `SALUS_OPERATOR_EMAIL` nor `SALUS_ENV`, so it cannot start.
2. `SELLER_*`, the analytics, retention, push, SMS, fiscal and chat-summary settings reach no production container.
3. The landing image is built without `VITE_APP_URL`.
4. No mobile release build sets `SUPPORT_EMAIL`, `PRIVACY_NOTICE_URL` or `API_BASE_URL`.

Questions: should RD4 wire those in? Will you keep a list, off the server, of the last 14 days' account closures, as
the runbook now asks? Is everyone signing in again after a restore acceptable?

## RD5: the demo

Run it on a stack with an empty database (a checkout of its own, or an emptied postgres volume; it refuses a database
with other clinics, so it will not run on your `salus` stack as it is):

```
make demo-seed      # about 30 s; prints the sign-in sheet (TOTP secrets on the first run only)
make web-dev        # then http://localhost:5173
```

The script is `docs/demo/SCRIPT.md`: 15 minutes in the order front desk, a visit, the receipt, online hours, the
patient app, the numbers. Screenshots are in `docs/demo/screenshots/`, retaken with `docs/demo/screenshots.sh`. The
patient app is shown live on an emulator, with no screenshots.

Seed on the morning of a demo: receipts carry the seed's time, because fiscal receipts are never backdated. The staff
UI is in Bosnian. The plan price is a labelled placeholder (2.00 BAM). The copy and the three closing questions are a
draft for you. RD6 restyled disabled buttons and "Add step" after the screenshots were taken; retake them before
showing.

## Decisions I took without you

- Recreated `feat/cms` from origin/main after checking the old branch had no diff against main.
- Ran every check, the integration's included, in worktrees with their own compose projects. The integration ran on
  `wt/cms/int` and was fast-forwarded into `feat/cms`, so no command touched your `salus` stack.
- Gave packages an unlimited budget (`RUNNER_BUDGET_MIN=720`), as you asked; none needed stopping.
- RD3's specs live in `test/e2e/web` (the existing Playwright suite), not `web/e2e`.
- RD3 added `@axe-core/playwright` and `axe-core` 4.14.0 as e2e dev dependencies.
- RD3 raised the e2e `go test -timeout` to 30 minutes (the suite takes 7 to 8 minutes with three accessibility passes).
- RD3 changed focus rings from mint to brand green.
- RD6 read item 5 as the web plan booking's date field (the wave 6 walk's finding). The app already formats dates
  through intl.
- RD6 chose the disabled look: flat grey, with a 3.5 to 3.7:1 inset outline.
- RD2 put the CI budgets on medians, not p95, because shared runners are noisy.
- RD2 used a generated column rather than a LEAKPROOF wrapper, which would need a superuser.
- RD2 added a `make load-budget` step to the Go workflow.
- RD4 treated image pins as dependency files and updated them (Go, postgres, clamav, redis).
- RD4 added no start-up refusals; the gaps are listed as findings instead.
- RD5 prints the synthetic staff's TOTP secrets on the first run, since you need them to sign in.
- RD5 added `config.IsDev` in `internal/config`, since only that package may read the environment.
- The dev compose Postgres now runs with `cluster_name=salus-dev`, and the demo seed requires it. Your `salus` stack
  gets it on its next `make up`: Postgres is recreated, the data is kept. Until then `make demo-seed` refuses there,
  which it would anyway since your database has clinics. The tests don't depend on it: they pass on a Postgres without
  the setting.
- I corrected the role matrix's classes of four routes, to match what the server does with valid requests.
- I declined, and recorded: `00087`'s lock, the remaining `HistoryShared` per-patient read, the load driver's Redis URL
  check, and the timing risk of the CI budget test.
- I did not retake the demo screenshots after RD6's restyle.

## Left running or on disk

- Running: nothing of this run. No containers of the wave 7 projects.
- Your `salus` stack: no command of mine ran against it. This session's Stop hook (`scripts/runner.sh gate`, `make
  arch-gate`) ran once in the main checkout. It started nothing new (`up --no-recreate`, everything was already
  running) and ran `go test ./...`, which makes throwaway databases on that Postgres and drops them. It failed on the
  demo tests, which led to the follow-up commit `886998a0`. Your data volumes were not touched.
- Removed: the seven worktrees (`../salus-cms-RD1` to `RD6`, `../salus-cms-int`), their `wt/cms/*` branches, and 28
  volumes of the wave 7 compose projects (each project's own; `salus_gomod`, `salus_gobuild` and `salus_golangci`
  kept).
- Left: the images `salus:cms-rd1` to `cms-rd6` and `salus:cms-int`. They can be removed with
  `docker image ls 'salus:cms-*'`, after checking the list.
- Left: the shared tags `salus-tools:dev`, `salus-playwright:1.65.1` and `salus-landing:dev`, rebuilt by the packages
  (now Go 1.27.4).
- Left: fresh pulls of `postgres:18.7`, `redis:8.10.3`, `clamav:1.5.5` and `golang:1.27.4-trixie`.
- Left: volumes from earlier waves (`salus-cms-0`, `-d5`, `-w3`, `-w4` and others); not this run's, so not touched.
- Left: package reports in `.claude/runs/cms/RD*.report.md` (git-ignored). No `.wip.patch` was needed.
