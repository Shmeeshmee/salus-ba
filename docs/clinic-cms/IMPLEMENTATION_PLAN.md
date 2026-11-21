# Clinic system: implementation plan

Packages A to G of `docs/plans/clinic-cms.md` ("Tickets and packages"), unchanged. Step 0 is merged. This file says
how they run; where it differs from that document's "Waves" and "Checks per PR" on branches, worktrees and pull
requests, this file wins.

## How this plan runs

- One branch `feat/cms` per wave, cut from `origin/main`. Each package of a wave runs in its own worktree on
  `wt/cms/<package>`, opened by the owner's `wave` tool, and makes one commit `cms(<package>): <summary>`.
- A package never pushes and never opens a pull request. After the wave, one integration session merges the
  `wt/cms/*` branches into `feat/cms` in the wave's order, runs the checks and opens one pull request for the wave.
- Notes for one package are in `notes/<package>.txt`: read yours before you start.

## Waves

| Wave | Packages | Why they run together |
| --- | --- | --- |
| 1 | A, B, C, D | Each owns its own files (`docs/plans/clinic-cms.md`, the file ownership table); shared files get one appended block each; migrations are numbered apart (below). Integration order A, B, C, D |
| 2 | E, then F and G once unheld | E builds on A (modes) and C (receipts); F on C and Q1; G on C and Q6 |

Migration numbers (main's next free number is `00060` on 2026-11-21): A `00060`, B `00061`, C `00062`, D `00063`,
wave 2 from `00064`. A package that needs no migration leaves its number unused.

## Rules for every package

1. Read `CLAUDE.md`, `docs/plans/clinic-cms.md` whole, `docs/plans/clinic-cms.PROGRESS.md` (step 0, and what it left
   to the wave), this file, `PROGRESS.md` here, and `notes/<package>.txt`.
2. Build only your package: its row in "Tickets and packages", its row in the file ownership table and its entry in
   "Notes for the wave-1 packages" of `docs/plans/clinic-cms.md`.
3. Test first, as `CLAUDE.md` "TDD" says.
4. In shared files (`api/openapi.yaml`, `internal/api/app.go` and the router, `internal/worker`, `internal/config`,
   `cmd/salusctl`, `ownership.yaml`, `internal/retention/rules.go`, `docs/privacy/dpia.md`, the i18n catalogs,
   `sqlc.yaml`) add only your own block. Run `make gen-clients` and `make sqlc` for your own changes; the integration
   session regenerates them after the merges.
5. Your stack is your own: set `COMPOSE_PROJECT_NAME` to your worktree's folder name in lower case, and
   `SALUS_IMAGE_TAG`, in the git-ignored `deployments/.env`, with the override without published host ports, as
   `docs/plans/appointments.md` "Worktrees and stacks" describes. Other packages are running in sibling worktrees at
   the same time: never touch them, never `git stash`.
6. Where the plan conflicts with the code, stop and report instead of improvising.
7. Write your section in `docs/clinic-cms/PROGRESS.md` under your own heading only.
8. In the report, add after the usual fields: every new user-facing string for the owner, every place you departed
   from the contract, and anything the integration session must do for you (`INTEGRATION NOTES:`).

## Check sets

- `go`: `make lint`, `make test`, `make archcheck`, `make cover`, `make i18n-check`; `make fhir-validate` when a
  golden FHIR file changed; `make gen-clients` leaves no diff.
- `web`: `make web-lint`, `make web-typecheck`, `make web-test`.
- `mobile`: `make mobile-analyze`, `make mobile-test`.
- `e2e`: `make e2e`.

## Definition of done

The package's behaviour is built and tested as its notes in `docs/plans/clinic-cms.md` require; `go` passes, and `web`,
`mobile` and `e2e` pass where the package touched them; the package's part of the DPIA, the retention rules and
`ownership.yaml` is written; `PROGRESS.md` has its section; one commit `cms(<package>): <summary>`; the tree is clean;
the compose stack is stopped.

## Packages

| Package | Depends on | Check sets | Done when |
| --- | --- | --- | --- |
| A | step 0 | go, web, mobile, e2e | A patient books an in-person or a video visit; an in-person visit has no call; its e-mail and reminders carry the clinic's address |
| B | step 0 | go, web, mobile | An admin keeps the price list; a patient sees the price at booking; nothing is charged |
| C | step 0 (B's lines through the seam, faked) | go, web, e2e | Staff issue, reverse and print receipts with gapless numbers; the fake fiscal report runs once; the accountant's reports match the receipts |
| D | step 0 | go, web | A clinic has a plan, a trial and monthly invoices; unpaid stops new bookings and never care; the owner marks paid in `salusctl` |
| E | A, C | go, web, mobile, e2e | Hold until wave 1 is merged |
| F | C, Q1 | go, web | Hold for the owner |
| G | C, Q6 | go | Hold for the owner |
