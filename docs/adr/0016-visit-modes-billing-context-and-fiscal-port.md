# 16. Clinic system: visit modes, the billing context and the fiscal port

Status: accepted, 2026-11-21 (docs/plans/clinic-cms.md, epics SAL-44 to SAL-47). The owner had not answered the plan's
Q1 to Q9 when this was written; each question's default applies (the plan's "Decisions", "Defaults applied"), for the
owner to confirm. They supersede docs/booking/SPEC.md's "in-person versus video types" out-of-scope line (now: both,
per schedule rule) and CLAUDE.md's "yearly subscription plus a small cut" (now: a monthly membership per practitioner).

## Context

Salus is sold to clinics as the system they run on, with telemedicine built in, for a monthly membership. A clinic
keeps no second program beside it only if Salus covers the front desk: visits at the clinic as well as by video, a
price list, and the receipts the law requires. The FBiH Law on Fiscalization of Transactions (Sl. novine FBiH 9/26, in
force 12 February 2026) moves fiscalization from cash-register hardware to software that reports each receipt to the
Tax Administration (ESET); private clinics are in scope. The ESET technical specification and test access are not yet
in hand (Q6).

What exists: every appointment is a video consultation of a clinic. A `schedule` holds a role's weekly rules as JSON
(`schedule.availability`, `{day, start, end}`), `slot`s are generated from them and an `appointment` holds one slot.
No price, service or invoice exists; `appointment.Prices` is nil and `pkg/payments` is a port with a fake: setting
`Prices` would charge patients in the app, which the pivot does not want. Ports with a fake that is the only engine
until a provider is chosen are an established shape (`pkg/sms` and `config.LoadSMS`, ADR 0015).

## Decision

### Visit modes

Two modes, `in-person` and `video`. A schedule rule has one mode (a key `mode` of the rule in
`schedule.availability`); a practitioner who works both has two rules, and rules of different modes never overlap
(the slots of one schedule never overlap, so the edit refuses it; package A). A slot carries its rule's mode
(`slot.visit_mode`) and an appointment copies its slot's at booking (`appointment.visit_mode`), which never changes
afterwards (no `UPDATE` grant). Everything that exists is `video`: the migration's default and the rules' backfill.

FHIR R4 has no element for it; R5 has `Appointment.virtualService` (and `Slot` none). Q8's default: an extension
`https://salus.invalid/fhir/StructureDefinition/visit-mode` on `Slot` and `Appointment`, a `valueCode` bound
(required) to the value set `https://salus.invalid/fhir/ValueSet/visit-mode` of the code system
`https://salus.invalid/fhir/CodeSystem/visit-mode` (`in-person`, `video`), all in `api/fhir/`. A `video` appointment
has the meaning of an R5 appointment with a `virtualService`; an `in-person` one of an appointment without. The
`salus.invalid` host is the one every Salus extension uses until the canonical domain is chosen; the plan's
`salus.ba` was corrected to it in step 0.

An in-person visit has no call, device check or join window; its documents, questionnaires, history and chat are as
for video. Reminders and the booking e-mail give the clinic's address (`organization.address_line`, `postal_code`)
instead of the join link (package A).

### The billing context

A new bounded context `billing`, three domains, each owning its tables:

| Domain | Owns | FHIR |
| --- | --- | --- |
| `chargeitemdefinition` | `charge_item_definition`, `charge_item_definition_translation`: the clinic's price list | `ChargeItemDefinition` (R4), served on the JSON API first (`/clinic/services`); no FHIR interaction until a SPEC asks |
| `invoice` | `invoice`, `invoice_line`, `invoice_sequence`: receipts and invoices a clinic issues to patients and companies | `Invoice` (R4), JSON API first (`/clinic/receipts`, `/me/receipts`) |
| `subscription` | `subscription_plan`, `clinic_subscription`, `subscription_invoice`: what a clinic owes Salus | none: plain JSON, nothing clinical |

Money is an integer in the currency's minor unit (`BAM`, fening) with its ISO 4217 code; VAT is a rate in basis points
on the service and copied to the line, null for exempt (Q9's default: exempt unless the service names a rate). A
receipt's lines copy the service's name, price and VAT at issue, so a price change never changes an issued receipt.

Prices are shown at booking and paid at the clinic (Q5's default). `appointment.Prices` stays nil; in-app charging is
SAL-48's, for prepaid international bookings.

### Receipts are append-only

An issued receipt never changes, as a chat message (`communication`): a mistake is a reversing receipt that names the
one it reverses (`reverses_id`, unique: a receipt is reversed once). Only the fiscal columns change, once, from
`pending` to `fiscalized` or `failed` (a trigger), and `failed` back to `fiscalized` by the retried job. Numbers are
the clinic's own gapless sequence: `invoice_sequence` holds one row per clinic, locked (`UPDATE … RETURNING`) in the
receipt's transaction, so two receipts issued at once get consecutive numbers (package C). A receipt to a patient is
health data (Art. 9: that a patient paid for a service at a clinic); a receipt to a company names the company only.

### The fiscal port

`pkg/fiscal` knows nothing of Salus: a `Receipt` (the seller's tax id, the receipt's reference, number, time, lines,
totals and payment method, and for a reversal the fiscal id of the receipt it reverses), and a `Reporter` with
`Report(ctx, Receipt) (Fiscalized, error)` and `Reverse(ctx, Receipt) (Fiscalized, error)`. `Fiscalized` is what the
Tax Administration returns: its identifier, the verification code and the QR payload, and when. The reference is the
receipt's opaque id, so a retried report of the same receipt is recognised. `fiscaltest` is the in-memory fake:
deterministic identifiers and codes from the reference, the same answer for a reference reported twice, and a count of
the calls (the "never reports twice" test of package C). `config.LoadFiscal` takes only the fake (`FISCAL_ENGINE`
empty or `fake`) and refuses `eset` until package G (Q6) writes its adapter, as `config.LoadSMS` does.

Reporting runs as a River job enqueued in the receipt's transaction (`InsertTx`, the receipt's id only) and runs after
it commits, retried; the receipt shows `pending` until it is fiscalized, and the clinic sees `failed`. The Tax
Administration is a public authority receiving data by law, not a processor; the DPIA records what it receives (no
patient name on a receipt to a patient unless the specification requires one).

### Subscription

Plans in `subscription_plan` (the owner edits them in `salusctl`, no UI), priced per active practitioner role per month
in `BAM`; `basic` and `pro` to start, prices `[PRICE]` (Q2). A clinic has one `clinic_subscription`, a 30-day trial from
its creation, then `active`, `past_due`, `suspended` or `cancelled`. The scheduler issues each clinic's
`subscription_invoice` for the month ahead from the active roles on the 1st; the owner marks it paid in `salusctl`
(Q3). Past due: a banner at 14 days, new bookings stop at 30; care never stops (Q4). Package D.

## Consequences

- Every existing schedule, slot and appointment is `video`; nothing changes for a clinic until it adds an in-person
  rule (package A).
- The extension and its value set are validated by `make fhir-validate` against `api/fhir/`.
- Receipts can be issued and shown before ESET exists: the fake fiscalizes them with test values. No receipt issued
  with the fake is a legal receipt; production waits for package G and the owner's certification (Q6).
- The retention of receipts is a placeholder of 10 years until the legal decision, as the booking's periods were.

## Alternatives

- **A visit type on the appointment only.** The patient chooses before the time, and the clinic plans rooms by the
  schedule; a mode only on the appointment would let a video-only practitioner be booked in person. Rejected.
- **A `serviceCategory` or `appointmentType` coding for the mode.** R4's `appointmentType` (v2 0276) means routine,
  walk-in, emergency; reusing it or `serviceCategory` for the channel would mislead other FHIR readers. Rejected for an
  extension shaped after R5's element.
- **Charging patients in the app.** The pivot keeps payment at the clinic; in-app charging stays for SAL-48.
- **Fiscalizing inside the receipt's transaction.** A slow or failing ESET call would hold the sequence row and block
  every receipt of the clinic. Rejected for a job after the commit.
