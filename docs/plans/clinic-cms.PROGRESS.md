# Clinic system: progress

The running record of docs/plans/clinic-cms.md, one section per merged package, written at merge time.

## Step 0: the contract (branch `cms-0-contracts`)

Phase A: the plan, CLAUDE.md "Product" (the monthly membership) and the `billing` context in its list, ADR 0016 (visit
modes, the billing context, the fiscal port). The owner had not answered Q1 to Q9: each default is applied and
recorded in the plan ("Decisions", "Defaults applied 2026-11-21, owner to confirm").

Phase B, built: migration 00059 (a placeholder) with the visit mode (a `mode` key on every rule of
`schedule.availability`, backfilled `video`; `slot.visit_mode`, `appointment.visit_mode`, default `video`, no `UPDATE`
grant on the appointment's), the `organization` columns (`address_line`, `postal_code`, `legal_name`, `tax_id`,
`vat_registered`, `billing_email`, granted to the app), `charge_item_definition` and its translations, `invoice`,
`invoice_line`, `invoice_sequence`, `subscription_plan`, `clinic_subscription`, `subscription_invoice`, each with
row-level security, grants, ownership and (for the two with personal data) retention; the append-only trigger of an
issued receipt. In Go: `schedule.VisitMode` and `Mode` on `Rule`, `Slot` and `Appointment` (stored, read, copied from
the held or offered slot at booking, in the JSON `Appointment` and the availability's rules), the FHIR visit-mode
extension on Slot and Appointment (golden files) with its StructureDefinition, CodeSystem and ValueSet in `api/fhir/`;
the `billing` context in archcheck with `chargeitemdefinition`, `invoice` and `subscription`, each with `model.go`,
`errors.go`, `repository.go`, `query.sql` and its sqlc package, and a `Service` whose methods refuse with
`errors.ErrUnsupported` (tested); the seam `invoice.Prices`, which `chargeitemdefinition.Service` satisfies as it is
(compile-time check in the test); the jobs' arguments `invoice_fiscalize` and `subscription_monthly`; `pkg/fiscal`
(`Reporter`, `Check`) with `fiscaltest`, and `config.LoadFiscal` (the fake only; `FISCAL_ENGINE` in
`configs/.env.example`). The OpenAPI of every wave-1 route with `x-salus-package` `cms-b`, `cms-c`, `cms-d`, a contract
test that the enums are the Go constants and that every route names its package, and the regenerated clients. The DPIA
section "Clinic system".

### Contract changes (where the code showed a better fit than the plan's draft)

| Plan's draft | As built | Why |
| --- | --- | --- |
| Main at `00057`, step 0 takes `00058` | `00059`; the wave's placeholders `00060` to `00063`, wave 2 from `00064` | `00058` is on main since #97 (appointment settings' version) |
| `visit_mode` on "`schedule_rule` or the schedule's rules" | A `mode` key on each rule in `schedule.availability`, backfilled `video` | The rules are a JSON array on `schedule`; there is no rule table |
| `visit_mode` as a type (`modes visit_mode[]`) | `text` with a check, `modes text[]` with a check | As every status column of the schema; sqlc and pgx would need an enum array registered for `visit_mode[]` |
| Extension URL `https://salus.ba/fhir/StructureDefinition/visit-mode` | `https://salus.invalid/fhir/StructureDefinition/visit-mode` (and `CodeSystem`, `ValueSet` beside it) | Every Salus extension uses `salus.invalid` until the canonical domain is chosen |
| `vat` (Q9) on the service and the line | `vat_rate_bp`, basis points, null for exempt | Q9's default (exempt unless a rate is given) in integer arithmetic, as the money |
| `charge_item_definition_translation (id, language, name)` | `(charge_item_definition_id, language, name)`, keyed by the first two | As `questionnaire_translation` |
| No sequence table | `invoice_sequence` (one row per clinic) | Package C's concurrency test needs the row it locks; gapless numbers cannot come from a Postgres sequence |
| Fiscal columns "leave `pending` once" | `pending` to `fiscalized` or `failed`, and `failed` to `fiscalized` | The report is retried after the job gives up; a fiscalized receipt never changes |
| Receipts kept 10 years (placeholder) | `ThenKeep` with the placeholder in its basis | A delete rule needs its applier (`TestRules_EveryRuleThatRemovesRowsCanBeApplied`); it comes with the legal decision |
| `/clinic/receipts/{id}.pdf` | `/clinic/receipts/{id}/pdf`, and `/me/receipts/{id}/pdf` for the patient's own | A `net/http` wildcard is a whole path segment; the patient's receipt in the app needs its own route |
| `/clinic/services` only for B | Also `PUT /clinic/services/{id}` and `GET /organizations/{id}/services` (the prices at booking) | Editing a service and showing its price to the patient are in B's scope and had no route |
| No route for the clinic's billing details | `GET`, `PUT /clinic/billing-details` (D) | D's "the clinic's billing details" needs one; the address goes on FHIR `Organization.address` (A), whose update exists |
| The organization additions in Go | Columns only in step 0 | The plan gives the Go side to A (address) and D (billing details); doing it here would only move their edits |
| "The mode on availability and booking (A)" | `Appointment.visitMode` in step 0 (the handler sends it); `AvailabilityRule.mode` sent in responses, taken in a PUT from A | The appointment's mode is the slot's, so booking takes none; a PUT naming a mode before A would make in-person rules whose slots are generated `video` |

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| Where `VisitMode` lives | `schedule` (the rule's), used by `slot` and `appointment`; the extension's URL is `slot.ExtensionVisitMode` |
| A rule without a mode | Saved `video` (the service's validation), an unknown mode refused (`ErrInvalidAvailability`); rules of any modes still never overlap |
| Slot generation | Unchanged: every generated slot is `video` until A carries the rule's mode (a PUT cannot name a mode before then) |
| A visit's receipt | Held to the visit's clinic by a key on `appointment (id, organization_id)` (new unique key) and, when it names a patient, to the visit's patient by one on `appointment (id, organization_id, patient_id)` |
| The owner and an issued receipt | The append-only trigger holds for every role, the table's owner too (as `communication`) |
| Plans | No plan is seeded: the prices are `[PRICE]` (Q2); D seeds `basic` and `pro` through `salusctl` |
| `x-salus-package` values | `cms-a` to `cms-d` (the plan's packages; several packages share one epic) |

Independent review of the branch (a fresh subagent, Copilot being out of quota until 2026-12-01), fixed before the
PR: a company receipt or cash sale (no patient) could name another clinic's visit, the three-column key being MATCH
SIMPLE (a second key on `appointment (id, organization_id)`, new); lines could be added to an issued receipt in a later
transaction, and could name another clinic's service, a foreign key bypassing row-level security (the
`invoice_line_guard_insert` trigger, for every role); the app could mark Salus's invoices paid (no `UPDATE` grant on
`subscription_invoice`; salusctl does it as the owner); `fiscal.Check` now checks each line's arithmetic, and the fake
refuses a second reversal of one receipt (`TestInvoice_RefusesLaterLinesAnotherClinicsServiceAndAnotherClinicsVisit`,
`TestSubscriptionInvoice_IsNotMarkedPaidByTheApp`, `TestCheck_RefusesAMalformedReceipt`,
`TestReporter_ReversesAFiscalizedReceiptOnly`). Left to the wave, recorded below: a slot carrying its rule's mode and a
reschedule keeping it (A); `clinic_subscription`'s status moving only forward, which D enforces with its status moves.

Second independent review (a fresh subagent, after CI's one failure), fixed before merge: the QR payload had no
column, and the guard's list of fiscal columns would have kept a later migration from filling one on a fiscalized
receipt (`fiscal_qr`, in the guard and in `invoice_fiscalized`); the seller's legal name, tax id and address are copied
onto the receipt at issue (`seller_legal_name`, `seller_tax_id`, `seller_address`), so a retried report or a reprint
after the clinic edits its details names the seller as it was; `fiscal.Check` refuses a sale with a negative quantity,
a reversal with a positive one and a discount above the line's amount; a receipt never reverses itself
(`invoice_reverses_other`) and the fake refuses to reverse a reversal; the directory's price-list policy checks
`o.listed` itself, as `practitioner_directory` does (defence in depth: no test reproduces a leak through the
organization's policies today). `TestReminder_SentOnceAtTheRightInstant` failed CI once on a random token holding "Cy";
the token is taken out before the check. Left to the wave, recorded below: a reschedule within the appointment's mode
(A); a receipt's patient being one the clinic cares for, and no reversal of a reversal (C).

Checks (2026-11-21, in the worktree's own compose project): `make lint` (with `make i18n-check`), `make test`,
`make archcheck`, `make cover` (90.9% total, the new packages 96.7 to 100%), `make e2e`, `make fhir-validate` (0 errors
in 122 files), `make web-typecheck`, `make web-test` (1066), `make mobile-analyze`, `make mobile-test` (571; its shared
appointment fixture gained `visitMode`), `make gen-clients` with no diff. CI and the review of the PR come after the
push.

Copy waiting for the owner: none. Step 0 adds no user-facing string; the API returns codes only.

Notes for the wave:

- **A.** A reschedule moves an appointment to another slot of its role; with modes, only to a slot of the same mode
  (the appointment's mode never changes; no `UPDATE` grant). Slot generation must carry each rule's mode, and the
  free-slot cleanup (`DeleteFreeSlotsExcept`) must compare the mode with the period.
- **A.** The second review asks for the reschedule rule in the database as well as the service, if it is cheap: the app
  may update `appointment.slot_id` (00054) but not `visit_mode`, so nothing in the schema stops a video appointment
  moving onto an in-person slot.
- **C.** A receipt without a visit may name any patient (patients are global, the foreign key bypasses row-level
  security), and that patient then sees it: the service issues one only to a patient the clinic cares for (a care
  relationship or a visit), tested. A reversal of a reversal is refused by the service (the schema allows it; the fake
  refuses it).
- **C.** The seller's columns are copied from `organization` at issue; issuing is refused while the clinic's legal name,
  tax id or address is missing.
- **C.** Lines are inserted only in the receipt's own transaction and from its clinic's price list (trigger). A
  reversal's sign (negative quantities and total) is the service's rule; the schema allows it.
- **D.** The app may update `clinic_subscription` (plan, status, period) in its clinic's scope; "only forward" is D's,
  in the service or a trigger. `subscription_invoice` has no `UPDATE` grant: marking paid is salusctl's, as the owner.
- **D.** `subscription_invoice.number` is unique; how Salus numbers is D's (a gapless sequence like C's, if the law
  asks).

## A: visit modes (branch `cms-a-visit-modes`)

Not started.

## B: price list (branch `cms-b-price-list`)

Not started.

## C: receipts (branch `cms-c-receipts`)

Not started.

## D: subscription (branch `cms-d-subscription`)

Not started.

## E: online hours (branch `cms-e-online-hours`)

Not started.

## F: reception (branch `cms-f-reception`), hold for Q1

Not started.

## G: ESET adapter (branch `cms-g-eset`), hold for Q6

Not started.

## Facts

- Main at migration 00058 on 2026-11-21 (#97); step 0 takes 00059.
- Copilot is out of quota until 2026-12-01: a fresh subagent reviews each PR until then.

## Waiting on a person

- The owner: to confirm or change the defaults applied for Q1 to Q9 (docs/plans/clinic-cms.md, "Decisions").
- The owner: the ESET technical specification and test access (Q6), the company that invoices clinics (Q3), an
  accountant's word on VAT (Q9), the legal retention period for receipts.
