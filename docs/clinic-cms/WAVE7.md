# Clinic system: wave 7, readiness (an unattended overnight run)

Part of `docs/clinic-cms/IMPLEMENTATION_PLAN.md`: its "Rules for every package", "Check sets" and "Definition of
done" apply. Planned 2026-11-25 with waves 1 to 6 on main (`1cb56c5d`). No new product behaviour and no owner
decision is needed: six packages that prove and harden what is built, each finding fixed with a failing test first,
plus the demo the owner needs for clinic visits. One session runs them itself, with subagents in their own
worktrees and compose projects, at most three at a time (each `make check` peaks near 6 GB).

Migration numbers: a package that needs one takes main's next free number in the order RD2, RD6, RD1; the
integration renumbers.

| Package | Check sets | Its own files | Touches |
| --- | --- | --- | --- |
| RD1 | go | `internal/api/role_matrix_test.go` and fixes where it finds a hole | any handler or service it fixes |
| RD2 | go | `test/load/`, index migrations, query fixes | `query.sql` files and their sqlc output |
| RD3 | web | `web/e2e` accessibility specs, fixes in screens and `app.css` | web only |
| RD4 | go | `scripts/`, `docs/deploy/`, dependency files | no behaviour under `internal/` |
| RD5 | go, web | `cmd/salusctl` demo command or `scripts/demo-seed`, `docs/demo/` | nothing else |
| RD6 | go, web, mobile | small fixes named below | `schedule`, web desk and plan screens, one mobile screen |

### RD1: the role matrix

F1 proved route by route what a receptionist may reach. Do the same for every role: a table generated from
`api/openapi.yaml` and the FHIR CapabilityStatement, so a route added later fails until it is classified. For each
route and each of patient, practitioner, receptionist, admin and signed-out: the caller's own clinic and data,
another clinic's, and (for a patient) another patient's. A refusal reveals nothing: the same answer for a real id
of another clinic as for an unknown one. Also: a suspended clinic, a restricted record, a closed account, a
revoked consent. Every hole found is fixed test first and listed with its severity.

### RD2: load and slow queries

Seed synthetic data at the size of a busy year (50 clinics, 500 practitioners, 200 000 patients, 1 000 000 visits,
their receipts and reminders) on a stack of your own. Drive the hot paths with a load tool that runs in a container
(booking, the desk day, the calendar, the patient's visits, the directory, receipts and their report, the clinic
statistics, the hourly slot generation, the reminder dispatch). Record p50, p95 and p99 and the slowest statements
(`pg_stat_statements`). Fix what is slow: missing indexes, N+1 reads, unbounded lists. Set a budget per path as a
benchmark or test that fails on a regression, sized to run in CI in under two minutes on a small seed. Record the
before and after in `docs/clinic-cms/load/REPORT.md`.

### RD3: accessibility and keyboard

Every staff screen and dialog checked with an automated accessibility engine in Playwright at 1280 and 390 px, and
by keyboard alone: focus order, visible focus, focus returned when a dialog closes, labels on every control, error
messages announced, contrast on text and controls, targets of at least 44 px on touch. Fix what fails. The specs
stay in the suite so a regression fails. List what needs a design decision instead of guessing.

### RD4: operations drill

- Backup and restore by `docs/deploy/backup-restore.md` on a fresh stack: back up a seeded database and object
  store, destroy them, restore, and prove the data and the files are back (counts, a decrypted attachment, a
  receipt PDF). Record it in `docs/deploy/restore-drills.md`; fix the runbook where a step was wrong or missing.
- Migrations: from empty to head, and from a dump taken at `00059` (before the clinic system) to head, with data.
- Dependencies: `govulncheck`, the image scan and the web and mobile audits; update what has a fix, list what
  does not with its reach.
- Every setting in `.env.example` is documented and every placeholder (`[PRICE]`, `[SELLER]`, `[OPERATOR EMAIL]`,
  `[SUPPORT EMAIL]`, `[APP URL]`) is listed in one place, `docs/deploy/BEFORE-PRODUCTION.md`, with what refuses to
  start in production while it is unset.

### RD5: the demo

`make demo-seed`: one command that fills a fresh stack with a synthetic clinic the owner can show: two locations
with rooms and a device, four practitioners and a receptionist, a price list, hours, sixty patients, a month of
visits in every state, receipts and a reversal, quotes, lab results, a treatment plan, questionnaires with
branching, chat, a data request, the membership with an invoice. Names are plainly synthetic. Idempotent, and
refused outside development. `docs/demo/SCRIPT.md`: a fifteen-minute walkthrough in the order a clinic owner would
ask (the front desk, a visit, the receipt, online hours, the patient app, the numbers), each step with its sign-in
and a screenshot at 1280 px, and the three questions to ask the clinic afterwards. Draft copy, for the owner.

### RD6: wave 6 leftovers

A warning on a practitioner's working hours when a room clash means some slots are not generated; the desk names
a location that was taken out of use; "Add step" looks like a button; a disabled button looks disabled; the booking
date in the app's own format; the patient app shows a plan step's time beside its day.
