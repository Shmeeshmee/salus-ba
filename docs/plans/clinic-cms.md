# Clinic system: in-person visits, price list, receipts, clinic subscription, online hours

Epics SAL-44 (clinic subscription), SAL-45 (fiscalization and invoicing), SAL-46 (clinic management core), SAL-47
(after-hours remote practice). Status: planned 2026-11-21 from the owner's pivot. The owner had not answered the open questions
when step 0 began: each question's default was applied on 2026-11-21 ("Decisions", "Defaults applied"), for the owner
to confirm. Design in ADR 0016. SAL-48 (diaspora and international patients) and SAL-49 (onboarding
and migration) are later plans; this one leaves their seams only where they cost nothing.

## Why (the pivot, owner, 2026-11-18 to 2026-11-21)

Salus stops being sold as telemedicine with a cut of in-app payments. Clinics pay a **monthly membership** and use
Salus as the system they run the clinic on, with telemedicine built in: their doctors can also work online after
hours, from home, with every Salus feature, and the clinic earns from those hours without more space or staff. A
clinic that has to keep its old software beside Salus will not switch, so Salus has to cover the front desk: in-person
visits, a price list and the receipts the law requires. The new FBiH Law on Fiscalization of Transactions (Sl. novine
FBiH 9/26, in force 12 February 2026) moves fiscalization from cash-register hardware to software reporting through
ESET; private clinics are in scope, so every clinic changes its fiscal setup anyway. That is the moment to switch.

## Corrections (what the code says against the epics' text)

| Epic text | The code today |
| --- | --- |
| "Online consultations tied to the clinic" (SAL-47) | Every appointment is already a video consultation of a clinic: there is no other kind. In-person visits are out of scope in docs/booking/SPEC.md ("in-person versus video types"), so the gap is the in-person visit, not the online one |
| "Price list", "fiscal receipt for every paid service" (SAL-45) | No price, service or invoice exists anywhere. `appointment.Prices` is nil and `pkg/payments` is a port with a fake; setting `Prices` would start charging patients in the app through `purchase`, which the pivot does not want. This plan does not set it |
| "Staff and roles: owner, admin, receptionist, doctor, nurse" (SAL-46) | `auth.Role` is `admin`, `practitioner`, `patient`, an owner decision in docs/accounts/SPEC.md §1. A reception role reopens it: Q1 |
| "Multiple locations, rooms and equipment" (SAL-46) | `organization` has one location (region and city, for analytics). Several sites per clinic are a later plan, not this one |
| "Findings on the clinic's letterhead from clinic templates" (SAL-46) | Largely there: `composition` has clinic templates, drafting and signing, and `pkg/pdfdoc` renders PDFs. Only the letterhead is missing; it goes with the receipt's PDF (C) |
| "Yearly subscription plus a small cut" (CLAUDE.md "Product") | Superseded: CLAUDE.md "Product" is amended in step 0's phase A |

## Tickets and packages

One branch and one PR per package, as docs/plans/appointments.md. Each package names its epic and is checked off in
it when it merges.

| Package | Branch | Epic | What |
| --- | --- | --- | --- |
| Step 0 | `cms-0-contracts` | all four | Phase A: this plan, CLAUDE.md "Product", ADR 0016 (visit modes, billing context, fiscal port), the owner's answers. Phase B: the shared contract below (migration, the `billing` context and its domains' skeletons, the visit mode on schedules, slots and appointments, `pkg/fiscal` with its fake, the OpenAPI of every wave-1 route, generated clients) |
| A | `cms-a-visit-modes` | SAL-46 | In-person and video visits end to end: availability per mode, slots carry it, the patient books one, the calendar shows it, an in-person visit has no call, its reminders and e-mails say where to come |
| B | `cms-b-price-list` | SAL-46 | The clinic's price list: services with price, duration, VAT treatment and the modes they are offered in, translations for patients, the admin's screen, prices shown at booking (never charged in the app) |
| C | `cms-c-receipts` | SAL-45 | Receipts and invoices at the clinic: a receipt for a visit from price-list lines, payment method, cancellation by a reversing receipt, sequential numbering, the PDF on the clinic's letterhead, reporting through `pkg/fiscal` (the fake until D2), the daily and period reports for the accountant |
| D | `cms-d-subscription` | SAL-44 | The clinic's membership: plans priced per active practitioner per month, trial, status and grace period, the monthly invoice to the clinic, the admin's subscription screen, the owner's operations in `salusctl` |
| E | `cms-e-online-hours` | SAL-47 | After-hours online practice: online-only hours per practitioner set by the clinic, which services may be done online, the online consultation receipted by the clinic (C), the admin's view of online hours, consultations and revenue per practitioner |
| F | `cms-f-reception` | SAL-46 | Hold for Q1. The reception role and desk: today's list, check-in, walk-in booking, receipts at the desk |
| G | `cms-g-eset` | SAL-45 | Hold for Q6. The ESET adapter of `pkg/fiscal` against the Tax Administration's test environment, then production |

## Waves

```
Step 0 (one PR): phase A (documents and the owner's answers), then phase B (the contract)
Wave 1, in parallel, each in its own worktree and compose project:   A   B   C   D
Wave 2, after wave 1 has merged:                                     E   F (hold Q1)   G (hold Q6)
```

- **Worktrees.** `git worktree add ../salus-cms-a -b cms-a-visit-modes origin/main` (and `salus-cms-b`, `-c`, `-d`),
  each with its own `COMPOSE_PROJECT_NAME` (`salus-cms-a` …) and `SALUS_IMAGE_TAG` in its git-ignored
  `deployments/.env` and the override without published host ports, as the appointments wave did. Never
  `git stash`; one worktree never touches another.
- **Merge order** in wave 1: A, B, C, D, one at a time, main merged into the next branch first (no force-push). A
  first: B's "modes a service is offered in" and C's receipt for a visit read A's mode. C after B: its lines come from
  the price list (built against step 0's seam until B merges). D last and independent: it touches only its own
  domain, `organization`'s admin screen and `salusctl`.
- **Migrations.** Main is at `00058` (2026-11-21, after #97; corrected in step 0, the plan said `00057`). Step 0
  takes `00059`; the wave's placeholders follow: A `00060`, B `00061`, C `00062`, D `00063`, wave 2 from `00064`. Each branch renumbers to the next free number when it merges
  main before its own merge, and its migration test file follows.
- **Shared files.** `api/openapi.yaml`, the generated clients (regenerated after each merge with `make gen-clients`,
  never hand-merged), `internal/api/app.go` and the router, `internal/worker` (registration lines), `internal/config`,
  `cmd/salusctl`, `docs/architecture/archcheck/config.yaml` and `ownership.yaml`, `internal/retention/rules.go`,
  `docs/privacy/dpia.md`, the i18n catalogs, `sqlc.yaml` and the sqlc output: the later merge resolves.

| Package | Its own files | Touches, appending its own block |
| --- | --- | --- |
| A | `internal/schedule/service_mode.go`, `internal/slot/service_mode.go`, `internal/appointment/service_mode.go`; web `AvailabilityModes.tsx`; mobile `visit_mode.dart` | `schedule`, `slot`, `appointment` core files (model, query.sql, repository_postgres.go, handlers); web `Availability.tsx`, `Calendar.tsx`, `AppointmentDetail.tsx`; mobile booking screens; `notification` templates (where to come) |
| B | `internal/chargeitemdefinition/*`; web `PriceList.tsx`; mobile `service_price.dart` | `organization` none; web `ClinicSettings.tsx` (a link); mobile booking confirm screen |
| C | `internal/invoice/*`, `pkg/fiscal/*` (beyond step 0's port and fake: none, the fake is step 0's); web `Receipts.tsx`, `ReceiptDialog.tsx`, `ClinicReports.tsx` | web `AppointmentDetail.tsx` (the "Issue receipt" action, after A's merge); `composition` none; `pkg/pdfdoc` (the letterhead, if it needs a new primitive) |
| D | `internal/subscription/*`; web `Subscription.tsx`; `cmd/salusctl` subscription commands | `organization` (the clinic's billing details: legal name, ID number (JIB/IDB), address, e-mail for invoices); web `ClinicSettings.tsx` (a link) |

## Decisions

### Owner, 2026-11-18 to 2026-11-21

| Topic | Decision |
| --- | --- |
| Revenue | A monthly membership charged to clinics. Live translation stays a paid add-on per use. An optional transaction fee is kept only for prepaid bookings by diaspora and international patients (SAL-48, later) |
| Positioning | Salus is the clinic's management system with telemedicine; doctors may work online after hours from home with every feature |
| Market | Private clinics in BiH first; RS and Brčko rules after FBiH |

### Defaults applied 2026-11-21, owner to confirm

The owner had not answered Q1 to Q9 ("Open questions") when step 0 began; each question's default applies until the
owner answers. An answer that differs is a change to this plan, made before the package it touches.

| # | Topic | Default applied |
| --- | --- | --- |
| Q1 | Reception role | Yes: a `reception` role (books, checks in, issues receipts; sees no clinical content), added in package F, which stays on hold until the owner confirms because it reopens docs/accounts/SPEC.md §1. Until then admins do the desk |
| Q2 | Plans and prices | Two plans, `basic` and `pro`, priced per active practitioner role per month in `BAM`, prices `[PRICE]` placeholders; a 30-day trial from the clinic's creation |
| Q3 | Who invoices clinics, how they pay | The owner's company as `[SELLER]` placeholders; bank transfer, marked paid by the owner in `salusctl`; card payment later |
| Q4 | A clinic that does not pay | A banner for the admin at 14 days past due; new bookings stop (public listing hidden) at 30 days; care never stops |
| Q5 | Where patients pay | At the clinic, not in the app (`appointment.Prices` stays nil); SAL-23 is rescoped to prepaid international bookings (SAL-48) |
| Q6 | ESET specification, test access, certification | An owner task; package G stays on hold and `pkg/fiscal`'s fake runs everywhere until then |
| Q7 | Jurisdictions | FBiH first; RS and Brčko later |
| Q8 | Visit mode in FHIR | An R4 extension on `Slot` and `Appointment` with the meaning of R5's `virtualService` (ADR 0016) |
| Q9 | VAT on services | Exempt by default; a per-service VAT rate for taxable items. An accountant confirms before production |

### Agent, for the owner to review (2026-11-21)

| Topic | Decision |
| --- | --- |
| Billing context | A new bounded context `billing`: `chargeitemdefinition` (the clinic's price list, FHIR `ChargeItemDefinition`), `invoice` (receipts and invoices a clinic issues to patients and companies, FHIR `Invoice`), `subscription` (what a clinic owes Salus: plain JSON, not FHIR, nothing clinical). CLAUDE.md's context list gains it in step 0 |
| Visit mode | Two modes, `in-person` and `video`, on a schedule rule, carried by its slots and copied to the appointment at booking. A rule has one mode; a practitioner with both has two rules. FHIR R4 has no field for it (R5's `virtualService` does): `Appointment.serviceCategory`-free, an extension on `Appointment` and `Slot` with the URL `https://salus.ba/fhir/StructureDefinition/visit-mode` and a value set in `api/fhir/`, the R5 element's meaning, written in ADR 0016. Existing schedules, slots and appointments are `video` (the migration's default) |
| In-person visits | No call, no device check, no join window; the consultation's documents, questionnaires, history and chat work as for video. Reminders and the booking e-mail say the clinic's address instead of the join link. The clinic's address is added to `organization` (A) |
| Prices are not charged in the app | `appointment.Prices` stays nil: prices are shown at booking and paid at the clinic, receipted by C. In-app charging is SAL-48's, for prepaid international bookings, and SAL-23 is rescoped to that |
| Price list | A service: name (clinic's language, optional translations for patients as questionnaires do), an optional code, price in the currency's minor unit (`BAM`), duration (default the slot length), VAT treatment (Q9), the modes it is offered in, active. A booking may name a service (optional, shown to the clinic) |
| Receipts | Issued by staff for a visit or on its own (walk-in, product), from price-list lines with quantity and discount; payment method (`cash`, `card`, `transfer`); issued receipts never change (append-only, as `communication`): a mistake is a reversing receipt. Numbering, the seller's data and the fiscal fields (ESET identifier, verification code and QR) follow the ESET technical specification; the model keeps them as fields the adapter fills, the fake fills test values |
| Fiscal port | `pkg/fiscal`: `Report(ctx, Receipt) (Fiscalized, error)` and `Reverse`, with `fiscaltest` (the fake). Reporting runs as a River job after the receipt's transaction, retried; a receipt shows "reporting" until it is fiscalized, and the clinic sees failures. The ESET adapter is package G, held for Q6. The fake runs everywhere until then |
| Who issues receipts | Today: the admin and the practitioner of the visit. The reception role waits for Q1 (package F) |
| Subscription plans | Priced per active practitioner role per month, in `BAM`, from a plan table the owner edits in `salusctl` (no UI for plans). Two plans to start, `basic` and `pro` (video, transcription and AI), prices `[PRICE]` until Q2. A 30-day trial from the clinic's creation |
| Subscription invoices | Generated monthly by the scheduler for the month ahead, from the active roles on the 1st; a role added mid-month is billed from the next month. Paid by bank transfer, marked paid by the owner in `salusctl` (Q3); card payment later through `pkg/payments` |
| Not paying | Q4's default: 14 days past due the admin sees a banner; 30 days past due new bookings stop (public listing hidden); care never stops: existing appointments, records, documents, exports and subject rights keep working, always |
| Data | Nothing in the price list or a subscription is personal data. A receipt to a patient is personal data (who paid for what, at which clinic: health data, Art. 9); a receipt to a company names the company only |

## Contract (step 0, phase B, migration 00059 as a placeholder)

As built; every change from the first draft is listed in `docs/plans/clinic-cms.PROGRESS.md` ("Step 0", "Contract
changes").

### Tables and columns

| Table | Columns |
| --- | --- |
| `schedule.availability` (the rules are JSON), `slot`, `appointment` (added) | A key `mode` on each rule (`in-person` \| `video`, every existing rule backfilled `video`); `slot.visit_mode` and `appointment.visit_mode` (`text`, the same check, default `video`, not null). An appointment copies its slot's at booking (direct and waitlist) and has no `UPDATE` grant on it |
| `organization` (added) | `address_line`, `postal_code` (the clinic's address for in-person visits and receipts); `legal_name`, `tax_id` (JIB/IDB), `vat_registered` (bool), `billing_email` (Salus's invoices to the clinic). Not personal data |
| `charge_item_definition` | `id`, `organization_id`, `name`, `code`, `price_minor`, `currency` (default `BAM`), `duration_minutes` (null: the slot's length), `vat_rate_bp` (basis points, null for exempt: Q9), `modes` (`text[]`, at least one of `in-person`, `video`), `active`, `version_id`, timestamps. Row-level security: the clinic's scope; the directory sees the active services of listed clinics. Never deleted. `charge_item_definition_translation` (`charge_item_definition_id`, `language`, `name`; key the first two, as `questionnaire_translation`) |
| `invoice` | `id`, `created_at`, `organization_id`, `patient_id` (nullable), `company_name` and `company_tax_id` (nullable; a patient, a company or neither for an anonymous cash sale, never both), `appointment_id` (nullable; held to the visit's clinic, and to its patient when the receipt names one, by two composite foreign keys), `kind` (`receipt` \| `invoice`), `reverses_id` (nullable, unique: reversed once, by a receipt of the same clinic), `number` (unique per clinic), `issued_at`, `issued_by_account_id`, `payment_method` (`cash` \| `card` \| `transfer`), `total_minor`, `currency`, `fiscal_status` (`pending` \| `fiscalized` \| `failed`), `fiscal_id`, `fiscal_code`, `fiscalized_at` (all three set exactly when fiscalized). Append-only after issue except the fiscal columns, for every role (trigger): `pending` to `fiscalized` or `failed`, `failed` to `fiscalized` (the retried report), a fiscalized receipt never changes. No `DELETE` grant. Row-level security: the clinic's scope; the patient sees their own |
| `invoice_line` | `id`, `invoice_id`, `charge_item_definition_id` (nullable: an item off the price list), `name` (copied at issue), `quantity` (not zero; negative on a reversal), `unit_price_minor`, `discount_minor`, `vat_rate_bp` (copied at issue), `total_minor`. No `UPDATE` or `DELETE` grant, a trigger refuses an update for every role; inserted only in the receipt's clinic, in the transaction that issues the receipt and from that clinic's price list (trigger), seen where the receipt is |
| `invoice_sequence` (added in step 0) | `organization_id` (key), `last_number`: the clinic's gapless receipt numbers, the row locked by C's `UPDATE … RETURNING` in the receipt's transaction |
| `subscription_plan` | `code`, `name`, `price_per_seat_minor`, `currency`, `features` (`text[]`), `active` |
| `clinic_subscription` | `organization_id` (key), `plan_code`, `status` (`trial` \| `active` \| `past_due` \| `suspended` \| `cancelled`), `trial_ends_at`, `current_period_start`, `current_period_end`, timestamps |
| `subscription_invoice` | `id`, `organization_id`, `number` (Salus's sequence, unique; how it is numbered is D's), `period_start`, `period_end` (dates), `seats`, `amount_minor`, `currency`, `issued_at`, `due_at`, `paid_at` (set exactly when paid), `status` (`open` \| `paid` \| `void`); unique per clinic and `period_start` (the monthly job runs once) |

Each table goes in `ownership.yaml` (owners: the domain named after it; `subscription` owns the last three) with
`personal_data` (`invoice_sequence` by `invoice`), and in `internal/retention/rules.go`: receipts and their lines kept
(`ThenKeep`) until the legal period is decided (placeholder 10 years after issue, needs the legal decision before
production): a delete rule needs its applier, which comes with the decision. The rest is not personal data.

### Go

- `billing` in `docs/architecture/archcheck/config.yaml` and CLAUDE.md's list: `chargeitemdefinition`, `invoice`,
  `subscription`, each a package with `model.go`, `errors.go`, `repository.go`, `query.sql` and its sqlc entry, and
  the service's exported method signatures the wave fills (returning `errors.ErrUnsupported` until then, with a test
  that says so).
- `schedule`, `slot`, `appointment`: `schedule.VisitMode` (`ModeInPerson`, `ModeVideo`) and the field on `Rule`
  (`Mode`; a rule without one is saved `video` until A lets the edit name it), `Slot` and `Appointment` (`Mode`); the
  FHIR extension `https://salus.invalid/fhir/StructureDefinition/visit-mode` (`slot.ExtensionVisitMode`) on Slot and
  Appointment and the golden files; the StructureDefinition, CodeSystem and ValueSet in `api/fhir/`. The JSON
  `Appointment` carries `visitMode`, the availability's rules `mode` (read only until A). Slot generation still writes
  `video` for every rule: A carries the rule's mode into its slots.
- `organization`: the columns only; the Go fields, reads and writes are A's (address) and D's (billing details).
- `pkg/fiscal` and `pkg/fiscal/fiscaltest`: the port (`Reporter` with `Report` and `Reverse`, `Receipt`, `Line`,
  `Fiscalized`, `Check`), the fake (deterministic `TEST-` identifiers, codes and QR from the reference, the same
  answer for a reference reported twice, `Reported` counting calls), `config.LoadFiscal` (`FISCAL_ENGINE` empty or
  `fake`; `eset` named and refused until G, as `LoadSMS`).
- The jobs' arguments: `invoice.FiscalizeArgs` (`invoice_fiscalize`) and `subscription.MonthlyArgs`
  (`subscription_monthly`), opaque ids only; their workers are C's and D's.
- The seam `invoice.Prices` → `chargeitemdefinition.Service.Lines` (C builds against a fake until B merges).

### HTTP (`api/openapi.yaml`)

Every wave-1 route with its schema and `x-salus-package` naming its builder (`cms-a` to `cms-d`), as the appointments
contract did: `GET`, `POST /clinic/services`, `PUT /clinic/services/{id}`, `GET /organizations/{id}/services` (the
prices at booking, public as the directory) (B); `GET`, `POST /clinic/receipts`, `GET /clinic/receipts/{id}`,
`POST /clinic/receipts/{id}/reverse`, `GET /clinic/receipts/{id}/pdf` (not `{id}.pdf`: a `net/http` wildcard is a whole
segment), `GET /clinic/reports/receipts` (JSON or CSV), `GET /me/receipts`, `GET /me/receipts/{id}/pdf` (C);
`GET /clinic/subscription`, `GET /clinic/subscription/invoices`, `GET`, `PUT /clinic/billing-details` (D); the mode on
availability (`AvailabilityRule.mode`, A) and on the appointment (`Appointment.visitMode`, step 0). The clinic's address
is A's: on FHIR `Organization.address` with the existing update. New error codes `invalid_service`,
`invalid_receipt`, `already_reversed`. Then `make gen-clients`.

## Notes for the wave-1 packages

- **A.** The booking SPEC's out-of-scope line is amended. Test: an in-person appointment never gets a video session,
  a join token or the call screen (the api refuses them), a video one is unchanged; slot generation for a role with
  both modes on overlapping hours is refused at the rule edit. The patient picks the mode before the time in the app.
  e2e: book in person, the e-mail carries the address and no join link.
- **B.** Prices in `BAM` minor units, shown with `Intl` in the UI language. A service offered only in person is not
  shown for a video slot and the reverse. Changing a price never changes an issued receipt (C copies the line).
- **C.** Concurrency test: two receipts issued at the same instant get consecutive numbers, never the same or a gap
  (the sequence row locked in the receipt's transaction). A reversing receipt reverses exactly one issued receipt, once.
  The fiscal job: run twice, rolled back once, never reports twice (the fake counts). The PDF: the letterhead (clinic's
  name, legal name, tax id, address), lines, totals, the fiscal code and QR; tested on the PDF's text. Reports: per day
  and period, by payment method and service, CSV export; equality tests against the raw receipts.
- **D.** The monthly job is unique per clinic and month (run twice, one invoice). Seat count from active practitioner
  roles on the 1st in the clinic's zone. Status moves only forward through the job or `salusctl`, each move audited.
  The 30-day stop of new bookings goes through `organization.Listed`-like checks in `appointment`'s booking, tested;
  care paths are tested to keep working for a suspended clinic.

## Checks per PR

`make lint`, `make test`, `make archcheck`, `make cover`, `make e2e`, `make i18n-check`; `make fhir-validate` where the
golden files changed (step 0, A); `make web-typecheck`, `make web-test`, `make mobile-analyze`, `make mobile-test`
where they changed; `make gen-clients` leaves no diff; CI green. Copilot is requested on each PR
(`gh pr edit <url> --add-reviewer @copilot`); while it is out of quota (until 2026-12-01) a fresh subagent that has
not seen the work reviews the diff. Valid findings are fixed before the merge, declined ones listed in the PR. Each
merge writes its section in `docs/plans/clinic-cms.PROGRESS.md` and is checked off in its Jira epic.

## Copy waiting for the owner

Every new user-facing string goes to the owner first, listed per PR. Expected: A (the mode picker, the in-person
booking e-mail and reminder with the address, in ten languages), B (the price list screen, the price at booking), C
(the receipt dialog, the receipt PDF's labels, the reports; `legal`: the patient's receipt in the app), D (the
subscription screen, the past-due banner, the subscription invoice e-mail and PDF).

## Personal data

| Field | Why |
| --- | --- |
| `appointment.visit_mode` | Whether the patient comes to the clinic or joins by video: the clinic plans its rooms and the patient gets the right instructions |
| `invoice.patient_id`, `invoice_line` (for a patient) | The receipt the clinic must issue and keep by law for a paid service; that a patient paid for a service at a clinic is health data (Art. 9). Kept as the fiscal and accounting law requires (Q-retention, placeholder) |
| `invoice.company_name`, `company_tax_id` | A company's invoice (systematic exams): not personal data |
| Everything in `charge_item_definition`, `subscription*`, the `organization` additions | Not personal data |
| River job arguments | Opaque ids only, as today |

`pkg/fiscal`'s ESET adapter sends receipts to the Tax Administration: a public authority by law, not a processor, but
the DPIA records what it receives (no patient name on a receipt to a patient unless the specification requires one;
Q6). Each package adds its part to `docs/privacy/dpia.md` (section "Clinic system (`docs/plans/clinic-cms.md`)").

## Open questions for the owner (defaults applied 2026-11-21, owner to confirm; see "Decisions")

| # | Question | Default if not answered |
| --- | --- | --- |
| Q1 | A `reception` role (books, checks in, issues receipts; sees no clinical content)? It reopens the accounts SPEC's roles | Yes, added in package F; until then admins do the desk |
| Q2 | Plan names and prices per practitioner per month | `basic` and `pro`, `[PRICE]` placeholders; trial 30 days |
| Q3 | Which company invoices the clinics, and how they pay | Your company's details as `[SELLER]` placeholders; bank transfer, marked paid in `salusctl`; cards later |
| Q4 | What happens when a clinic does not pay | Banner at 14 days past due, new bookings stop at 30, care never stops |
| Q5 | Patients pay at the clinic, not in the app (except SAL-48 later)? | Yes: SAL-23 is rescoped to prepaid international bookings |
| Q6 | ESET: who obtains the technical specification, the test access and any certification of Salus as fiscal software | You (owner task); G stays on hold, the fake runs until then |
| Q7 | FBiH first, RS and Brčko later | Yes |
| Q8 | Visit mode as an R4 extension shaped like R5's `virtualService` | Yes |
| Q9 | VAT on medical services: exempt by default with a per-service rate for the rest? An accountant confirms | Exempt by default, a per-service VAT rate field for taxable items |
