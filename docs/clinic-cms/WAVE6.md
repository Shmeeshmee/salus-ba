# Clinic system: wave 6, hardening and the next features

Part of `docs/clinic-cms/IMPLEMENTATION_PLAN.md`: its "How this plan runs", "Rules for every package", "Check sets"
and "Definition of done" apply unchanged. Planned 2026-11-25, with waves 1 to 5 on main. Two halves that share no
file: fixes for what the walks, reviews and reports left open, and three features from the board (SAL-51 locations,
rooms and equipment; SAL-52 lab results; SAL-53 treatment plans; SAL-59 privacy follow-ups).

Run: first `wave docs/clinic-cms 6.0 BG1 BG2 BG3 PX --max 3` (the features' contract beside the four fix packages),
then, with all five merged into `feat/cms`, `wave docs/clinic-cms LC1 LC2 LC3 RM1 LB1 LB2 TP1 TP2 --max 3`.
Integration order of the second half: LC1, RM1, LB1, TP1, LC2, LB2, TP2, LC3.

Migration numbers: 6.0 takes main's next free number; every other package that needs one takes the next free after
it, in the order BG1, PX, LC1, RM1, LB1, TP1, and says which in its report. The integration renumbers.

## Decisions applied (agent's defaults, 2026-11-25, owner to confirm; record them in PROGRESS.md)

| # | Point | Default |
| --- | --- | --- |
| U1 | Locations | A clinic has one or more locations, each with a name, address, phone and time zone. Today's single address becomes the clinic's first location (migration). FHIR `Location`. A video visit has no location |
| U2 | Hours per location | An in-person availability rule names its location; its slots and the booked visit carry it. The patient picks the location before the time. Receipts, reminders and e-mails use the visit's location address |
| U3 | Rooms | Optional. A location may list rooms; an in-person rule may name one. Two practitioners' in-person hours cannot overlap in the same room |
| U4 | Equipment | Optional. A location may list devices; a price-list service may require one. Two visits that need the same device cannot overlap: the booking is refused (`resource_busy`) and the desk shows why |
| U5 | Lab results | Staff attach a result to a visit: a PDF, or values typed in (test, value, unit, reference range, a flag). FHIR `DiagnosticReport` with `Observation`s and the PDF as `Binary`. The practitioner reviews and releases it; the patient sees it in the app only once released. No direct interface to a laboratory yet |
| U6 | Treatment plans | The practitioner or admin lays out a plan for a patient: ordered steps, each a service, a day offset and a visit type. Booking a plan books every step in one transaction, or none. FHIR `CarePlan`. The plan carries its quote (X1). Moving a step is allowed and the plan shows the new order; cancelling the plan cancels its future steps |
| U7 | An open invite | The admin may withdraw one; the limit of 50 is enforced under a lock |
| U8 | Signup link | The 72 hours count from the latest verification e-mail; the verified signup row drops the admin's e-mail once the invite is accepted |

## First half

| Package | Check sets | Its own files | Touches |
| --- | --- | --- | --- |
| 6.0 | go, web, mobile | the migration, the skeletons of `location`, `diagnosticreport`, `careplan` (clinical or scheduling context as the code fits), the OpenAPI of the second half's routes, generated clients | `ownership.yaml`, retention rules, DPIA, `archcheck/config.yaml` |
| BG1 | go, e2e | new test files and small fixes across `auth`, `signup`, `invoice`, `consent`, `clinicexport`, `internal/worker` | no `api/openapi.yaml` schema that 6.0 adds |
| BG2 | web | fixes in existing web screens and `app.css` | no generated client |
| BG3 | go | `scripts/`, the `Makefile`, `deployments/` test overrides, `.claude/` runner files | nothing under `internal/`, `web/`, `mobile/` |
| PX | go, mobile | `internal/datarequest` export additions, `internal/auth` closure limit, the no-admin warning | `internal/worker` |

### 6.0: the contract for locations, rooms, equipment, lab results and treatment plans

As 3.0 and 5.0 (their records are in PROGRESS.md): one migration, types, refusing services with a test each, the
OpenAPI of every second-half route with `x-salus-package`, regenerated clients, a fresh subagent's review before
the commit, PROGRESS headings and rows for every wave 6 package. Tables and columns for U1 to U6; the first-location
backfill; routes for the clinic's locations, rooms and devices, the location on availability, slots, bookings and
the public directory; lab results on a visit (staff write, practitioner release, patient read); treatment plans
(write, book, read, cancel; the patient's read).

### BG1: backend hardening

Work from the records, not from this list alone: read every "Deferred", "Declined" and "Open" paragraph of
PROGRESS.md from "Wave 3 integration" on, and fix what is a defect; list what you leave and why. At least:

- Refusals that row-level security turns into a rollback leave no audit row; list and refusal audit events carry
  no entity; a staff account on `/me/data-requests` and a request for another clinic's export id are not audited.
- The invite limit has no lock (U7); the admin cannot withdraw an open invite (route, audited; the screen is BG2's:
  add the API only and say so).
- The clinic export job does not re-check that its requester is still the clinic's admin when it runs.
- A 500 when a profile disappears as a consent is recorded; any other 500 a table-driven probe of the staff routes
  with malformed ids and bodies can find.
- Signup: U8.
- The flaky test in `internal/worker` (seen twice under parallel load, never captured): reproduce it with the
  package run 100 times under load, fix the cause, not the timing.
- A Postgres concurrency test for two staff bookings of one slot (F2 left it to a fake).

### BG2: web polish

From the walks' "Still open" and "Deferred" notes (`docs/clinic-cms/walk/wave3`, `wave4`, `wave5` WALK.md) and the
reports: nested dialogs where Escape closes both; the patient search's name button at 390 px; the membership
invoices table at 390 px; the walk-in dialog's width at desktop; the phone on every screen in the grouped form; the
long-standing lint warning in `analytics/ClinicView.tsx`; a "Withdraw" action on an open invite (against BG1's
route, with a fake in tests). Then one pass over every staff screen at 1280 and 390 px for clipped text, controls
off screen and inconsistent button styles, fixed where found. Record it as `docs/clinic-cms/walk/wave6-polish/`.

### BG3: tooling

- `runner.sh start` gave two worktrees the same ports (R3's report): derive ports so that two live worktrees can
  never share them, and refuse to start on a port already in use, with a test of the allocation.
- `make e2e` on a long-lived dev stack fails on old data: it always runs on its own fresh compose project and
  volumes and cleans up only those.
- A `make check` that runs exactly the plan's check sets for the files changed, so every package runs the same
  thing; document it in CLAUDE.md "Commands".

### PX: privacy follow-ups (SAL-59)

Receipt and quote PDFs in the patient's export; a file the export cannot open is skipped and named in the manifest
instead of failing the build; `consent.Mine`, `invoice.Mine` and `MyQuotes` narrowed to the target clinic in a
clinic-target export; a rate limit on the password asked when closing an account; an e-mail to the operator when a
request targets a clinic with no admin. The notification erasers wait for the lawyer's retention periods: not here.

## Second half (after 6.0 is merged)

| Package | Ticket | Depends on | Check sets | Builds |
| --- | --- | --- | --- | --- |
| LC1 | SAL-51 | 6.0 | go, e2e | Locations on the server: the clinic's locations, the location on rules, slots, visits and the directory; the visit's location on receipts, reminders and e-mails |
| RM1 | SAL-51 | 6.0 | go | Rooms and devices: the room on a rule with its overlap rule (U3), the device a service requires and `resource_busy` (U4), concurrency tests as the booking's |
| LB1 | SAL-52 | 6.0 | go | Lab results on the server (U5): attach, values, release, the patient's read; in the record hand-over and the patient's export |
| TP1 | SAL-53 | 6.0 | go | Treatment plans on the server (U6): write, book all steps or none, move, cancel; the plan's quote |
| LC2 | SAL-51 | 6.0 | web | The admin's locations, rooms and devices screens; the location and room on availability; the desk filtered by location |
| LB2 | SAL-52 | 6.0 | web, mobile | Attaching and releasing a result on the visit page; the patient's results in the app |
| TP2 | SAL-53 | 6.0 | web, mobile | Laying out and booking a plan on the web; the patient's plan in the app |
| LC3 | SAL-51 | 6.0 | mobile | The patient picks a location when booking in person and sees its address on the visit |

The Go packages own their own domain's files; the three that touch `appointment` (LC1 the location, RM1 the device
check, TP1 the plan's booking) each add one file there and one appended block in the core files. The web and mobile
packages build against 6.0's generated client with a fake API and leave their wiring checks to the integration.
