# Clinic system: progress

Step 0's record is in `docs/plans/clinic-cms.PROGRESS.md` (merged, PR #107, `b18974f`). From wave 1 on, each package
writes under its own heading here.

| Package | Status | Commit |
| --- | --- | --- |
| A | not started | |
| B | not started | |
| C | not started | |
| D | done | `cms(D)` on `wt/cms/D` |
| E | hold (after wave 1) | |
| F | hold (Q1) | |
| G | hold (Q6) | |

## A: visit modes

## B: price list

## C: receipts and invoices

## D: clinic subscription

Status: done, 2026-11-21 (branch `wt/cms/D`, one commit `cms(D)`).

Built:

- `subscription` (step 0's skeleton filled): `Current` and `Invoices` for the clinic's admin (403 for its other staff,
  404 for anyone else and for a clinic without a membership), `StartTrial` (30 days of the plan `basic`, once),
  `IssueMonthly` (one invoice per clinic and month, seats times the plan's price, the month in the clinic's zone; the
  first invoice ends the trial), `Review` (past due after the due date, suspended 30 days past it), `RunAll` (the
  scheduler's daily run over every clinic, one clinic's failure not stopping the others), `MarkPaid`, `SetStatus` and
  `PutPlan` (salusctl, as the owner), `BookingsStopped`. Every status move, payment and trial start is audited
  (`salusctl` or `subscription_monthly` as the agent); reads are not (no personal data).
- Routes `GET /clinic/subscription` and `GET /clinic/subscription/invoices` (`subscription.HTTPHandler`), `GET` and
  `PUT /clinic/billing-details` (`organization.Billing`, `organization.BillingHandler`: the Go side of step 0's
  columns, the PUT audited also when refused).
- Migration `00063_subscription_numbers.sql` (a placeholder number): `subscription_invoice_sequence`, one row locked
  in the invoice's transaction (gapless numbers, as `invoice_sequence`), in `ownership.yaml`; the select policy
  `clinic_subscription_run` for the job's list of clinics (`app.subscription_run = 'all'`).
- The job: `subscription_monthly` worked by `cmd/worker` (`internal/worker/subscription.go`), enqueued by the
  scheduler daily at 01:00 UTC and when it starts leading.
- The stop of new bookings: `bookableClinics` in `internal/api/subscription.go` wraps the clinics the appointment
  service reads, so a suspended or cancelled clinic reads as not listed for a booking, a waitlist entry and an offer's
  acceptance. `internal/appointment` is unchanged (A's files). Care paths do not ask the membership.
- `practitionerrole.Service.ActiveCount` (the seats), unaudited: a count discloses no role.
- `salusctl subscription plan | start-trial | mark-paid | status`, and `create-clinic` starts the trial (it prints
  `trial none` while the owner has not set the plan `basic`).
- Web: `Subscription.tsx` at `/clinic/subscription` (plan, status, invoices, billing details form), a link from
  `ClinicSettings.tsx`, `SubscriptionBanner` in the shell for admins (from 14 days past due, and when suspended),
  `useFormat().money`, the `subscription` catalog in the five staff languages.
- DPIA: "Package D" under "Clinic system". Retention: nothing added, none of D's tables holds personal data.

Readings and decisions (agent, for the owner to review):

| Topic | Decision |
| --- | --- |
| "The status moves only forward" and a payment | Forward by rank (`trial`, `active`, `past_due`, `suspended`, `cancelled`) through the job and `salusctl subscription status` (`ErrStatus` otherwise). One way back: `mark-paid` returns a `past_due` or `suspended` clinic to `active` when no invoice of it is past due any more. Without it a clinic that pays could never book again. Owner to confirm |
| Where "forward" is enforced | In the service (step 0 left "the service or a trigger" to D); no trigger |
| When a clinic is `past_due` | From the first moment an open invoice is past its due date; the admin's banner shows from 14 days past due (the web computes it from `pastDueSince`); `suspended` at 30 days |
| Payment term | An invoice is due 14 days after it is issued (`subscription.PaymentTerm`). The plan names none: owner to confirm |
| Daily run, not only on the 1st | The 14 and 30 day thresholds need a daily look, and a run missed on the 1st heals the next day; the invoice is issued once per clinic and month either way. Seats are the active roles when the invoice is issued |
| A trial that ends mid-month | No invoice while the trial has not ended when the month starts: billed from the next 1st |
| Invoice numbers | Gapless, one global sequence row locked in the transaction (step 0's note: "if the law asks"; an invoice sequence with gaps is hard to defend to an accountant) |
| No plan seeded | Prices are `[PRICE]` (Q2): the owner sets `basic` (and `pro`) with `salusctl subscription plan`; nothing in the code holds a price or the seller's details |
| A clinic without a membership | Books as before (every clinic that existed before D); `start-trial` gives it one |
| The booking stop's place | A wrapper around `appointment.Clinics` in the composition root instead of a new port in `appointment`, so A's files are untouched in the wave |

Departures from the contract: `subscription.NewService` takes `Ports` (it needs the audit, the transactor, the
clinics and the seats); `Subscription` gained `Plan` and `PastDueSince` (the schema's `planName`,
`pricePerSeatMinor`, `pastDueSince`); the errors `ErrNoPlan`, `ErrInvalidPlan`, `ErrNotBilled`.

Deferred:

- "Public listing hidden" for a suspended clinic: a booking is refused (`slot_unavailable`), but the directory still
  lists the clinic and its slots. Hiding it needs `organization`'s listing to read the membership.
- The subscription invoice's e-mail and PDF ("Copy waiting for the owner" expects them): not built; the admin reads
  invoices on the screen. They need the `[SELLER]` details (Q3) to be worth sending.
- `api/openapi.yaml` still says "Built in package D (404 until then)" on D's four routes: left so the generated
  clients need no regeneration in the wave; the integration session can drop the phrase.
- The screen was not walked in a browser at desktop and 390 px (the session ran past its budget); its behaviour is
  covered by `Subscription.test.tsx`.
- Tests of the service were written right after the code of the same step, not before it, in this package.

Checks (2026-11-21, in the worktree's own compose project `salus-cms-d`): `make lint` (with `make i18n-check`; first
run failed on a test 82 lines long and a test stub's field order, fixed), `make test`, `make archcheck`, `make cover`
(90.8% total; `subscription` 87.3%, `organization` 92.2%, `practitionerrole` 96.0%, `cmd/salusctl` 89.0%,
`internal/api` 93.2%, `internal/worker` 90.7%), `make web-lint` (one warning, older than this package),
`make web-typecheck`, `make web-test` (first run failed on the catalog test's list of areas, which now names
`subscription`; then 68 files passed), `make web-build`. Not run: `make gen-clients` (`api/openapi.yaml` is unchanged),
`make fhir-validate` (no golden FHIR file changed), `make e2e` (not in D's check sets). The session ran past its
60-minute budget (an interruption, then a resume on the coordinator's instruction).

## E: online hours

## F: reception role and desk

## G: ESET adapter

## Facts

- Main's next free migration number on 2026-11-21: `00060`.
- Copilot is out of quota until 2026-12-01: the integration session has a fresh subagent review the wave's diff too.

## Waiting on a person

- The owner: confirm or change the Q1 to Q9 defaults; plan prices (Q2) and the invoicing company (Q3) before D ships.
- The owner: the ESET technical specification and test access (Q6); an accountant on VAT (Q9); the legal retention
  period for receipts.
