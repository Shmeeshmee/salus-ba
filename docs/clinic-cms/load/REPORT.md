# Load and slow queries (WAVE7.md RD2)

Measured 2026-11-25 on the package's own load stack, a busy year of synthetic data, with every hot path driven through
the api as `cmd/api` builds it (`api.New`, real HTTP on the loopback) and through the workers' own services. Before and
after are a pair of runs a few minutes apart on the same database: "before" is the base commit (`13db9fbc`, in a
detached worktree), "after" this package.

## The seed

`test/load/seed_test.go`, written in one pass of SQL as the schema's owner (about 9 minutes, 4.4 GB):

| What | Rows |
| --- | --- |
| Clinics, each with a location, an admin and a receptionist, a price list of 20 | 50 |
| Practitioners, each with an account and a weekday schedule (Monday to Wednesday at the clinic, Thursday and Friday by video, 08:00 to 16:00 in half hours) | 500 |
| Patients, each with an account and a care consent at one clinic (one in ten at two) | 200,000 |
| Visits from 300 days back to 64 days ahead, 8 a practitioner and weekday: past 80% fulfilled, 5% no-show, 15% cancelled; future 85% booked, 15% cancelled | 1,046,750 (with the load's own bookings) |
| Slots: one per visit, then 28 days ahead by the hourly generation | 1,109,055 |
| Receipts, fiscalised, one line each, one a fulfilled visit | 687,600 |
| Reminders (24 h and 1 h), kept 90 days after their due time as the retention rule keeps them | 890,500 |
| Encounters, drafts and one coded diagnosis a fulfilled visit; the clinic statistics of every clinic-day | 687,600 each; 18,250 clinic-days |

Names are `Load Patient <n>`, `Load Practitioner <clinic>-<n>`, `Load clinic <n>`; e-mails `@example.test`. Not seeded,
and so not measured at their real size: audit events (every read writes them; the busy year would hold tens of
millions), notification deliveries, chats, questionnaires, documents.

**Refused outside the load stack** (`test/load/guard_test.go`): the seeder and the driver run only when
`LOAD_CONFIRM=synthetic-busy-year` is given, `SALUS_ENV` is `dev`, the server's `cluster_name` is `salus-load` (set only
by `test/load/docker-compose.load.yml`) and the database holds no clinic but the seed's. The budget test seeds a
throwaway database of `pgtest` only.

## How to run

- `make load LOAD_CONFIRM=synthetic-busy-year`: starts `<project>-load` (Postgres and Redis only, no host ports, Postgres
  with production's settings plus pg_stat_statements), migrates, seeds on the first run (kept for the next), then the
  readings: 500 requests a path, 8 at a time (`LOAD_REQUESTS`, `LOAD_CONCURRENCY`). `make load-down` removes it and its
  volumes.
- `make load-budget`: the budgets below on a small seed (3 clinics, 30 practitioners, 6,000 patients, 26,000 visits) on a
  throwaway database, 25 to 35 s; a step of the Go workflow's test job.

## Before and after (busy year, ms)

"Statements" is pg_stat_statements' calls per request, those inside SQL functions included (`track = all`: the
restriction check `patient_record_restricted` runs two per patient); "database ms" their execution time per request.
The machine was shared with two other packages' checks (load average 9 to 24), so the p95 and p99 move from run to run
by tens of milliseconds; the medians, the statements and the database time are the steadier comparison.

| Path | p50 before | p50 after | p95 before | p95 after | p99 before | p99 after | Statements before | after | Database ms before | after |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Booking: a practitioner's schedule | 0.9 | 0.4 | 1.3 | 0.6 | 1.5 | 0.9 | 4 | 4 | 0.14 | 0.08 |
| Booking: a week's free slots | 10.9 | **1.4** | 16.3 | 2.1 | 43.9 | 2.6 | 4 | 4 | 10.78 | **0.61** |
| Booking: book a slot | 16.3 | 22.1 | 19.9 | 41.3 | 22.3 | 51.7 | 44.7 | 44.7 | 4.56 | 6.00 |
| The desk's day | 133.5 | **23.8** | 173.3 | 50.2 | 178.8 | 800.5 | 478 | 155 | 100.61 | **17.50** |
| Calendar: a practitioner's week | 84.5 | **34.3** | 93.5 | 130.8 | 97.3 | 370.3 | 340.5 | 104.5 | 32.52 | **21.26** |
| Calendar: the clinic's week (admin, every practitioner) | 739.0 | **139.2** | 1087.6 | 710.4 | 2455.8 | 1484.4 | 3132.6 | 791.6 | 303.16 | **199.84** |
| The patient's visits | 14.3 | 9.4 | 21.6 | 14.7 | 24.2 | 17.6 | 37.3 | 22.8 | 2.02 | 1.15 |
| Directory: listed clinics | 1.5 | 0.7 | 3.5 | 1.5 | 5.1 | 1.9 | 4 | 4 | 0.20 | 0.10 |
| Directory: a clinic's practitioners | 2.3 | 1.8 | 4.6 | 2.4 | 5.3 | 2.7 | 15 | 15 | 1.12 | 0.79 |
| Receipts: the last 30 days | 29.4 | 27.5 | 38.6 | 35.2 | 67.4 | 56.3 | 8 | 8 | 15.00 | 13.81 |
| Receipts report: a month | 26.2 | 24.5 | 40.7 | 32.7 | 1069.2 | 158.8 | 13 | 13 | 15.30 | 14.48 |
| Receipts report: a year (CSV) | 197.1 | 183.6 | 231.3 | 220.0 | 244.7 | 239.0 | 13 | 13 | 129.28 | 125.67 |
| Clinic statistics: a year's visits | 50.5 | 54.5 | 62.0 | 76.4 | 65.9 | 154.1 | 10 | 10 | 46.74 | 47.31 |
| Clinic statistics: a year's diagnoses | 13.1 | 14.9 | 16.5 | 23.9 | 19.7 | 27.8 | 7 | 7 | 8.90 | 8.70 |
| Slot generation: the hourly run, 500 schedules (3 runs) | 14,637 | **9,263** | 16,151 | 9,475 | | | | | | |
| Reminders: 1,000 overdue, swept, dispatched and sent (reminders a second) | 16 to 43 | 46 to 60 | | | | | | | | |
| Reminder dispatch job (River's run time) | 37 to 71 | 17 to 20 | 76 to 604 | 27 to 44 | | | | | | |

A second after run, with the machine less loaded (load average 9), gave the desk 22.2 / 50.4 / 88.5, a practitioner's
week 23.3 / 38.2 / 60.8 and the clinic's week 110.8 / 153.7 / 269.8 (p50 / p95 / p99). The reminders' path was not
changed; its spread is the machine's (four runs after: 43 to 60 a second; one of 16 before, under the heaviest load).

On the small seed of the budget test (p50 / p95, ms): the desk 69.2 / 120.5 before, 14.0 / 20.4 after; a practitioner's
week 78.7 / 108.1 before, 16.3 / 20.3 after; the clinic's week 585.1 / 674.4 before, 68.5 / 85.1 after; a week's free
slots 3.2 / 6.1 before, 0.7 / 2.2 after; the hourly slot generation of 30 schedules 1,202 before, 753 after.

## What was slow, and the fixes

1. **Range reads under row-level security read the clinic's whole year.** A condition on `lower(period)` is not
   leakproof (`lower()` of a range is not marked so), so Postgres applies it after the policies, row by row, and uses
   only `organization_id` of the index `(organization_id, lower(period))`: the desk's day read all 20,890 visits of the
   clinic's year and checked the consent policy's subquery 17,749 times (132 ms in `EXPLAIN ANALYZE`). The same held for
   the calendar's week, the practice report and, on `slot`, the booking's free slots and the generation's comparison.
   Fix: `start_at` as a generated column of `appointment` and of `slot` (the same value as `lower(period)`, never
   written by the app), indexed, and the staff's range reads compare it (`timestamptz` comparison is leakproof): the
   desk's day reads its 80 rows (1.2 ms). Tests: `internal/appointment/repository_postgres_plan_test.go` and
   `internal/slot/repository_postgres_plan_test.go` assert, from the plan of the statement the repository sent, that a
   range read drops at most the range's own rows (`pgtest.Explain`).
2. **The calendar and the desk read a row at a time.** For every patient of the page: the consent state (two
   statements), the restriction (a function of two), all of the patient's visits at the clinic, the profile; and an
   audit insert for every appointment and every name shown. A clinic's week was 3,133 statements. Fix: the page's
   consent states (`consent.Service.States`), restrictions (`patient.Service.RestrictedAll`), visits (`VisitsOfPatients`)
   and profiles (`patient.Service.LookupAll`) in one statement each, and the page's AuditEvents in one insert
   (`RecordAll`): the same events, in the same order, in the same transaction. Tests:
   `TestSearch_WritesItsEventsAtOnceAndReadsEachPatientOnce`, `TestDeskDay_WritesItsEventsAtOnce`, and the batch reads'
   repository and service tests.
3. **The hourly slot generation sent every period to the insert's conflict check.** Each run inserts the 28 days of
   every schedule with `ON CONFLICT DO NOTHING`; a re-run's 500 periods all met the exclusion constraint. Fix: a period
   the schedule has a slot of already is left out before the insert (by the new index). With fix 1 on its comparison,
   the run of 500 schedules went from 14.6 s to 9.3 s.

**Indexes added** (migration `00087_start_at.sql`; its two table rewrites took 86 s on the busy year's seed, under an
`ACCESS EXCLUSIVE` lock, so the migration belongs before go-live or in a quiet hour):

| Index | Table | Serves |
| --- | --- | --- |
| `appointment_organization_start_at (organization_id, start_at)` | appointment | the desk's day, the calendar's range, the practice report |
| `slot_schedule_start_at (schedule_id, start_at)` | slot | the booking's free slots, the generation's comparison and its insert |

## The slowest statements

By total time over the run, before (top 10):

| Calls | Total ms | Mean ms | Rows a call | Statement |
| --- | --- | --- | --- | --- |
| 206,475 | 75,287 | 0.36 | 6.3 | `SearchAppointments` by patient and clinic: access's facts, once a patient of every page |
| 527,862 | 69,189 | 0.13 | 1.0 | `InsertAuditEvent`, once a row |
| 1,500 | 46,007 | 30.67 | 5,536.7 | `InvoiceLines` of the receipts listed |
| 500 | 40,807 | 81.61 | 66.0 | `DeskAppointments` (the year read for a day) |
| 1,500 | 32,883 | 21.92 | 5,536.7 | `InvoicesIssued` |
| 260,523 | 31,798 | 0.12 | 1.0 | `PatientByID`, once a row |
| 1,500 | 16,373 | 10.92 | 0.0 | `InsertSlots` |
| 1,500 | 15,159 | 10.11 | 0.0 | `DeleteFreeSlotsExcept` |
| 500 | 8,603 | 17.21 | 11.0 | `AppointmentSeries` (clinic statistics) |
| 10,194 | 7,794 | 0.76 | 1.0 | `PractitionerRoleByID` (the hourly generation, once a schedule) |

After (top 10):

| Calls | Total ms | Mean ms | Rows a call | Statement |
| --- | --- | --- | --- | --- |
| 2,000 | 57,062 | 28.53 | 256.4 | `InsertAuditEvents`, one a page |
| 1,500 | 44,610 | 29.74 | 5,536.7 | `InvoiceLines` |
| 1,500 | 31,474 | 20.98 | 5,536.7 | `InvoicesIssued` |
| 2,000 | 27,262 | 13.63 | 119.4 | `PatientsByIDs`, one a page |
| 1,500 | 13,052 | 8.70 | 0.0 | `DeleteFreeSlotsExcept` |
| 1,000 | 12,435 | 12.44 | 1,076.7 | `VisitsOfPatients`, one a page |
| 500 | 8,548 | 17.10 | 11.0 | `AppointmentSeries` |
| 10,212 | 8,526 | 0.83 | 1.0 | `PractitionerRoleByID` |
| 1,500 | 7,073 | 4.72 | 149.5 | `SearchAppointments` |
| 15,250 | 6,236 | 0.41 | 1.0 | `InsertAuditEvent` (single events: refusals, bookings, the generation's reads) |

What is left is mostly the size of the answer: a page's audit events (about 0.1 ms an event, three indexes on
`audit_event`), a year's receipts and their lines (13,700 a clinic), the admin's week of every practitioner (400 visits).

## Budgets

`test/load/budget_test.go` (`make load-budget`, CI): the small seed, every path above, 60 requests 4 at a time after a
warm-up; each path's median within its budget (about four times its median after the fixes, under its median before
them where a fix changed it: the desk 50 ms, a practitioner's week 50 ms, the clinic's week 250 ms), every answer the one
wanted, the hourly generation under 5 s and 200 reminders dispatched and sent within 30 s. The median, because on 60
requests a p95 is the third slowest and one pause of a shared machine moved it past any budget in two of three runs.

## Left for later

- **The staff's appointment search without dates is unbounded**: `Appointment?` by an admin, or by a practitioner for
  their own role, returns the clinic's or the role's whole history (about 20,900 a clinic a year here), with the facts
  of every patient in it. The web never asks for one (the calendar always sends a week). Limiting it is a change of the
  FHIR search's contract (an `OperationOutcome` `too-costly`, or `_count` paging), so it is left to the owner.
- The hourly generation still reads each schedule's role on its own (`PractitionerRoleByID`, 500 a run, 0.4 s of the
  9 s), and generates the schedules one after the other.
- The receipts report of a year (13,700 receipts and their lines, 184 ms) and the clinic statistics (four aggregate
  reads of 10 to 17 ms) are the answer's size; neither was changed.
- A booking's p95 rose in the after run (41 ms against 20) with nothing changed on its path but the slot's index; the
  other after run and the budget test's runs give 25 to 58 ms. To watch, with the audit events at their real size.
