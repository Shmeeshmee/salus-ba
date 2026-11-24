# Clinic system: progress

Step 0's record is in `docs/plans/clinic-cms.PROGRESS.md` (merged, PR #107, `b18974f`). From wave 1 on, each package
writes under its own heading here.

| Package | Status | Commit |
| --- | --- | --- |
| A | done 2026-11-21, integrated 2026-11-22 | `cms(A)`, merged into `feat/cms` with `cms(wave 1)` |
| B | done 2026-11-21, integrated 2026-11-22 | `cms(B)`, merged into `feat/cms` with `cms(wave 1)` |
| C | done 2026-11-21, integrated 2026-11-22 | `cms(C)`, merged into `feat/cms` with `cms(wave 1)` |
| D | done 2026-11-21, integrated 2026-11-22 | `cms(D)`, merged into `feat/cms` with `cms(wave 1)` |
| E | done 2026-11-23, integrated 2026-11-23 | `cms(E)`, merged into `feat/cms` with `cms(wave 2)` |
| H | done 2026-11-23, integrated 2026-11-23 | `cms(H)`, merged into `feat/cms` with `cms(wave 2)` |
| I | done 2026-11-23, integrated 2026-11-23 | `cms(I)`, merged into `feat/cms` with `cms(wave 2)` |
| L | done 2026-11-23, integrated 2026-11-23 | `cms(L)`, merged into `feat/cms` with `cms(wave 2)` |
| Q1 | done 2026-11-23, integrated 2026-11-23 | `cms(Q1)`, merged into `feat/cms` with `cms(wave 2)` |
| Q2 | done 2026-11-23, integrated 2026-11-23 | `cms(Q2)`, merged into `feat/cms` with `cms(wave 2)` |
| Q3 | done 2026-11-23, integrated 2026-11-23 | `cms(Q3)`, merged into `feat/cms` with `cms(wave 2)` |
| F | split into F1, F2, F3 (wave 3; the owner said yes to the reception role on 2026-11-23) | |
| G | hold (Q6) | |
| 3.0 | done 2026-11-23, integrated 2026-11-23 | `cms(3.0)`, merged into `feat/cms` with `cms(wave 3)` |
| F1 | done 2026-11-23, integrated 2026-11-23 | `cms(F1)`, merged into `feat/cms` with `cms(wave 3)` |
| F2 | done 2026-11-23, integrated 2026-11-23; the staff move deferred to wave 4 | `cms(F2)`, merged into `feat/cms` with `cms(wave 3)` |
| F3 | done 2026-11-23, integrated 2026-11-23 | `cms(F3)`, merged into `feat/cms` with `cms(wave 3)` |
| O1 | done 2026-11-23, integrated 2026-11-23 (migration renumbered `00066`) | `cms(O1)`, merged into `feat/cms` with `cms(wave 3)` |
| O2 | import done 2026-11-23, integrated 2026-11-23; the clinic data export deferred to wave 4 | `cms(O2)`, merged into `feat/cms` with `cms(wave 3)` |
| X1 | done 2026-11-23, integrated 2026-11-23 | `cms(X1)`, merged into `feat/cms` with `cms(wave 3)` |
| X2 | done 2026-11-23, integrated 2026-11-23 | `cms(X2)`, merged into `feat/cms` with `cms(wave 3)` |
| O3 | hold (after F2) | |
| X3 | hold (a payment provider) | |
| R1 | done 2026-11-24, integrated 2026-11-24 | `cms(R1)`, merged into `feat/cms` with `cms(wave 4)` |
| R2 | done 2026-11-24, integrated 2026-11-24 (migration `00067`) | `cms(R2)`, merged into `feat/cms` with `cms(wave 4)` |
| R3 | done 2026-11-24, integrated 2026-11-24 (migration `00068`) | `cms(R3)`, merged into `feat/cms` with `cms(wave 4)` |
| M1 | done 2026-11-24, integrated 2026-11-24 (migration `00069`) | `cms(M1)`, merged into `feat/cms` with `cms(wave 4)` |
| M2 | done 2026-11-24, integrated 2026-11-24 (migration `00070`); the patients' records and quotes.csv joined at the integration | `cms(M2)`, merged into `feat/cms` with `cms(wave 4)` |
| M3 | done 2026-11-24, integrated 2026-11-24; the consent for an imported patient joined at the integration | `cms(M3)`, merged into `feat/cms` with `cms(wave 4)` |
| M4 | done 2026-11-24, integrated 2026-11-24 | `cms(M4)`, merged into `feat/cms` with `cms(wave 4)` |
| 5.0 | done 2026-11-24 (migration `00071`) | `cms(5.0)` |
| FX | not started | |
| D1 | done 2026-11-24 (migration `00073`) | `cms(D1)` |
| D2 | done 2026-11-24 (migration `00074`) | `cms(D2)` |
| D3 | not started | |
| D4 | not started | |
| D5 | not started | |
| D6 | not started | |

## A: visit modes

Done 2026-11-21, one commit `cms(A)` on `wt/cms/A`. Built test first where the session allowed it; several Go tests
were written with their code and first run after it (the missing golden files were their only failures).

**Built.**

- `schedule`: the availability PUT takes `mode` on each rule (`in-person` or `video`); a rule that names none is a
  video rule, an unknown mode is `invalid_availability`, rules of any modes still never overlap (step 0's validation,
  which is the "both modes on overlapping hours is refused at the rule edit" of the plan's notes).
- `slot`: generation carries each rule's mode into its slots (`Period.Mode`); a rule that changes its mode replaces its
  future free slots (the cleanup compares the mode with the period) and leaves a busy slot, in its old mode, with its
  appointment.
- `appointment`: a reschedule takes only a slot of the appointment's mode (`ErrNotReschedulable`, `not_reschedulable`,
  as another role's slot); migration `00060` adds the trigger `appointment_guard_mode`, so the database refuses it too
  (step 0's second review). `ReminderNotice` carries the mode.
- `videosession`: a join of an in-person visit is `ErrNotFound` (404) for both participants: no session, room or
  token, the refusal audited.
- `organization`: `AddressLine` and `PostalCode` on the model, read and written through the existing FHIR update as
  `Organization.address[0].line` (at most one) and `postalCode`; left out keeps them, empty clears them; trimmed, at
  most 200 and 20 characters, and only beside a city (`ErrInvalidLocation` otherwise). `Organization.Address()` writes
  it on one line.
- `notification`: the confirmation, both reminder e-mails and the e-mail after a reschedule say that an in-person
  visit is at the clinic and give the address; the SMS reminder has the address after the clinic's name; the push is
  unchanged. Ten languages. A video visit's messages are unchanged (their golden files did not move).
- `web`: a visit type per row of working hours (`AvailabilityModes.tsx`), an "At the clinic" tag on the calendar card
  and the appointment, no join button, hint or consultation status for an in-person visit, and the street address and
  postal code in Clinic settings under the city.
- `mobile`: the patient picks video or the clinic before the time on a day that has both (`visit_mode.dart`), the mode
  is named on a day with one, on the booking screen and on the appointment; no Join for an in-person visit; a new time
  is offered only in the appointment's mode.
- Docs: the booking SPEC's out-of-scope line amended; the DPIA's "Package A"; `AvailabilityRule.mode` in
  `api/openapi.yaml` (description only) and the regenerated clients.

**Decisions.**

| Topic | Decision | Why |
| --- | --- | --- |
| Join of an in-person visit | 404 `not_found`, no new error code | The contract lists no code for it and the apps never offer the join; a new code would change the contract and every client catalog |
| A slot of the other mode at a reschedule | `not_reschedulable` | The same class as another role's slot: the clinic's rules never allow it |
| The address on FHIR | `line` and `postalCode` on the one `Organization.address` that holds the location; needs a city | The patient needs street, postal code and city together; the city is the location's |
| An in-person visit of a clinic without an address | The messages still say it is at the clinic, without an address | Nothing to give; the admin's screen says what the address is for |
| The public directory | Still shows no address | Not asked by the plan; the patient gets it in the messages. Showing it in the app at booking is left open (below) |
| The consultation status of an in-person visit | Not shown in `web`; the API still answers it (not started) | Only the join creates a session; refusing the status read is not in the plan's test list |
| Web PUT of the address | Always sends `line` and `postalCode` (empty clears) | The form shows both; one existing test's expected body gained the two keys on purpose |
| The city in the e-mail's address | The notifier's clinic service gets the places catalog (`internal/worker/notifier.go`) | It was built without one, so its read named no city and the e-mail gave street and postal code only; `make e2e` caught it |

**Checks.** Recorded in the package's report (`.claude/runs/cms/A.report.md`): the fast layer per change, then the
full set once.

**Deferred or open.**

- The clinic's address is not shown in the patient app (booking screen, appointment card): the public directory does
  not send it. For the owner to decide; the e-mail and reminders carry it.
- No browser walk of the changed screens at desktop width and 390 px: the session had no browser. The 390 px layouts
  are covered only by the component tests and the mobile golden screens.
- New copy (the mode picker, "At the clinic", the address fields and hint, the in-person line of the e-mails, in every
  language) is a draft for the owner; it is listed in the report.

**Integration notes.** Migration `00060` is a placeholder (`migrations/00060_visit_mode_reschedule.sql`, test
`migrations/visit_mode_reschedule_test.go`, which names no number). `web/src/ClinicSettings.tsx` gained two fields in
its `Location` part (B and D add a link each elsewhere in the file). `make gen-clients` and `make sqlc` were run for
these changes; regenerate after the merges.

## B: price list

Done 2026-11-21, one commit `cms(B)` on `wt/cms/B`. No migration: `00061` stays unused (step 0's tables were enough).

Built:

- `internal/chargeitemdefinition`: the service (`List`, `Create`, `Update`, `Offered`, `Lines`), the Postgres
  repository (services with their translations, optimistic locking on `version_id`, translations replaced on a save),
  the HTTP handler (`GET`, `POST /clinic/services`, `PUT /clinic/services/{id}` with `If-Match`,
  `GET /organizations/{id}/services`), wired in `internal/api/pricelist.go`. Only the clinic's admin writes; each write
  is an `AuditEvent` on `ChargeItemDefinition/<id>`, a refused one (not the admin, not found, a stale version) is
  audited without an entity. Reads are not audited: nothing in a price list is personal data.
- `Offered`: the active services of a listed clinic in the visit's mode, each named in the language asked for where
  translated; a clinic that is not listed is 404. A service offered only in person is not shown for a video slot and
  the reverse (tested in the service and through the app).
- `Lines` (the seam `invoice.Prices`): the picks priced from the clinic's active services in the caller's transaction;
  a line copies the name, unit price and VAT rate, so a later price change never changes an issued receipt.
- Web: `PriceList.tsx` at `/price-list` (admin only), linked from the clinic settings; the catalog area `prices` in the
  five staff languages. Prices through `Intl` in the UI language.
- Mobile: `service_price.dart` (`priceText`, `ServicePrices`) and the booking confirm screen shows the clinic's
  services and prices with "You pay at the clinic"; two strings in the ten languages. Nothing is charged:
  `appointment.Prices` stays nil.
- DPIA: "Package B: the price list". Ownership and retention were step 0's (no personal data, no retention rule).

Decisions and readings:

- `NewService` takes `Ports` (repository, auditor, transactor, clock, the directory of listed clinics): step 0's
  `NewService(repo, clk)` had no auditor or transactor. The one caller, `internal/invoice/service_test.go` (C's file),
  changed by one line.
- `Repository.List` takes the clinic's id: the directory's scope shows every listed clinic's services.
- The currency is `BAM` only (the plan's decision); another is `invalid_service`. A price is at most 1,000,000,000
  minor units (ten million BAM), so a line's total stays inside an int64; the OpenAPI has no maximum.
- `Lines` refuses a quantity below one and a discount below zero or above the line's amount: a reversal's negative
  lines are C's, copied from the receipt it reverses, not priced again.
- A validation refusal (`invalid_service`) is not audited, as the appointment settings': only authorization, not-found
  and version refusals are.
- The screen is at `/price-list`, not under `/clinic/`: Vite's proxy and the production Caddyfile send every
  `/clinic/*` path to the API.
- `Offered` asks `organization.Service.ListListed` whether the clinic is listed (no new method on `organization`,
  which is not B's).
- Mobile asks for the `video` mode: the app's `Slot` has no mode until A. See "Left for the integration".

Deferred:

- "A booking may name a service (optional, shown to the clinic)": the contract has no column on `appointment` and no
  field in the booking request; it touches A's files. Not built.
- The OpenAPI descriptions still say "Built in package B (404 until then)": changing them means regenerating both
  clients, which the integration session does anyway.
- The screen was not walked in a browser (the session's time went to an interruption); its behaviour is covered by
  `PriceList.test.tsx` and the app test.

Left for the integration:

- After A: `mobile/lib/slots_screen.dart` `_BookScreenState.didChangeDependencies` passes `api.VisitMode.video`; pass
  the slot's mode once `Slot` carries it.
- After C: `internal/invoice/service_test.go` builds `chargeitemdefinition.NewService(chargeitemdefinition.Ports{…})`;
  C's wiring of `invoice.NewService` in `internal/api` takes the price list service built in `pricelist.go` (export it
  from there or build it once in `app.go`).

Copy waiting for the owner: the web area `prices` (38 strings, `web/src/i18n/prices/en.json`), and mobile
`pricesHeading` ("Prices at this clinic") and `pricesPaidAtClinic` ("You pay at the clinic. Nothing is charged in the
app.").

Checks (2026-11-21, the worktree's own compose project): `make lint` (with `make i18n-check`), `make test`,
`make archcheck`, `make cover` (90.9% total, `chargeitemdefinition` 93.7%), `make web-lint`, `make web-typecheck`,
`make web-test` (1078), `make web-build`, `make mobile-analyze`, `make mobile-test` (576): all pass. `make gen-clients`
was not run: `api/openapi.yaml` is unchanged. The first full run failed on lint (struct layouts, two long tests, a
request without a context) and the second on the catalog's list of areas (`prices` added); both fixed. The session
was interrupted once and resumed, so it ran past the 60-minute budget on the clock.

## C: receipts and invoices

Status: built, 2026-11-21, on `wt/cms/C`. No migration (`00062` unused). No pull request: the wave's integration opens it.

Built:

- `internal/invoice`: the service on `Ports` (`Issue`, `Reverse`, `Read`, `List`, `Mine`, `Report`, `PDF`, and the
  job's `Fiscalize` and `GiveUp`), the Postgres repository and its queries, the River queue (`invoice_fiscalize` in the
  receipt's transaction), the JSON handler for the eight routes of the contract, the receipt's PDF with its labels in
  the five staff languages.
- Numbers come from `invoice_sequence` by an upsert that keeps the clinic's row locked until the receipt's transaction
  ends. The seller's legal name, tax id and address are copied at issue; issuing is refused (`409
  billing_details_missing`, new in `api/openapi.yaml`) while one is missing.
- Rules from step 0's review: a receipt on its own names only a patient the clinic treats (an active care consent or a
  visit there); a visit's receipt only the visit's patient; a receipt is reversed once and a reversal never.
- The fiscal job (`internal/worker/invoice.go`): reports through `pkg/fiscal`'s fake with the receipt locked, skips a
  fiscalized receipt, marks a rejected one `failed` at once and a pending one `failed` when River's last attempt
  fails; `failed` still becomes `fiscalized` on a later run. `cmd/worker` calls `config.LoadFiscal`.
- Web: `Receipts.tsx` (`/receipts`, admin), `ReceiptDialog.tsx`, `ClinicReports.tsx` (`/receipts/report`),
  `ReceiptPeriod.tsx`, `receipts.ts`, the `receipts` catalog in five languages, a nav link and an icon.
- `docs/privacy/dpia.md`: the package's part. Ownership and retention were complete in step 0 (no new table).

Decisions and readings:

- A receipt for a visit that names no company is the visit's patient's: the request need not carry the patient.
- An invoice is exactly a receipt with a company; a company's tax id needs its name.
- A practitioner issues only for their own visit and reads only the receipts they issued; list, report and reversal
  are the admin's (the contract's wording).
- A reversal copies the seller from the receipt it reverses, not from the clinic as it is now.
- The report is summed in Go from the receipts `List` returns, so it equals them by construction (tested).
- Lines have no position column: they are read by name. A position needs a migration; not taken.
- The PDF is in the clinic's default language (Bosnian for a language outside the five) and its time zone, names no
  patient, and prints the QR payload as text: `pkg/pdfdoc` draws no image. The drawn QR is left to package G, with
  the payload's specification.
- The report's CSV has code headers and minor units; a name that starts like a formula is quoted.
- The receipt is locked while it is reported (one row, for the length of the call), so two runs never report at
  once. A commit lost after the report leaves it pending; the retry sends the same reference.
- The fake keeps what it fiscalized in memory: after a worker restart a reversal of an earlier receipt is rejected
  by the fake and marked `failed`. Development only.
- `worker.New` builds the fake itself; a parameter for the engine comes with the second engine (G).
- The session ran past its 60-minute budget (it was interrupted and resumed on the coordinator's instruction).

Not wired in this worktree (see the report's INTEGRATION NOTES): the price list's service (B), the seller's fields on
`organization.Organization` (A's address, D's billing details), the "Issue receipt" action in `AppointmentDetail.tsx`
(A). Until then `POST /clinic/receipts` answers `billing_details_missing` on the real stack.

Deferred: `make e2e` for the receipt journey (it needs B, A and D merged); the browser walk at desktop and 390 px
(same reason, and no browser in this session); a retry of `failed` receipts after the job gave up (a sweep or an
admin action: not in the contract); the patient's receipts in `mobile/` (the API is there; the screen is in no
package's file list); the drawn QR on the PDF.

Checks (2026-11-21, compose project `salus-cms-c`): `make lint` (with `make i18n-check`) passed after one round of
fixes (function length, field alignment, `noctx` in the new tests and `cmd/worker`); `make archcheck` passed;
`make cover` passed (every test, 90.8% total, no package under 80%); `make web-typecheck`, `make web-lint`,
`make web-test` (1074 tests; the catalog's area list gained `receipts`) passed; `make gen-clients` run after the last
contract change (a new error code, `billing_details_missing`). Not run: `make e2e`, `make mobile-analyze` and
`make mobile-test` (only the generated Dart client changed), the browser walk.

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
- Migration `00061_subscription_numbers.sql` (00063 on the branch, renumbered at the merge): `subscription_invoice_sequence`, one row locked
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
- Web: `Subscription.tsx` at `/clinic/subscription`, `/membership` since the integration (plan, status, invoices, billing details form), a link from
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

## Wave 1 integration

Done 2026-11-22 on `feat/cms`: `wt/cms/A`, `B`, `C`, `D` merged with `--no-ff` in that order, then one commit
`cms(wave 1): integrate A, B, C, D`.

- **Merges.** Conflicts only in the shared files (this file, the DPIA, the mobile ARB catalogs, `internal/api/app.go`,
  `internal/worker/worker.go`, `web/src/App.tsx`, `app.css`, `web/src/i18n/catalog.test.ts`); every package's block
  kept. Generated code was not hand-merged: `make sqlc`, `make gen-clients` and `flutter gen-l10n` ran after the
  merges. D's migration `00063` is `00061`: main ends at A `00060`, D `00061`. D has no migration test file of its own
  in `migrations/` (its tests are `internal/subscription`'s repository tests), so none was renamed.
- **Price list in the receipts.** C's stand-in in `internal/api/invoice.go` is gone: `priceList` builds B's service
  once in `app.go`, for its own routes and the receipts' prices. An unknown, inactive or another clinic's service is
  400 `invalid_receipt` (tested through the api against Postgres).
- **Seller.** `clinicSeller` reads the legal name and tax id through the new `organization.Billing.ForClinic` (the
  clinic's own scope, for whoever of its staff issues, unaudited like `Details`) and the address through
  `organization.Service.Read`, which names the city. A receipt is issued once both are set; 409
  `billing_details_missing` remains while either is missing (tested through the api, and in the e2e).
- **Receipts see the clinic's visits.** Tested against Postgres through the api: the admin issues for a visit they
  are not part of; another clinic's visit of the same patient is 404.
- **Booking stop.** D's api test now books an in-person slot of a suspended clinic too (A's booking path reads the
  clinic through the same `Clinics` port): 409 `slot_unavailable`.
- **Web.** "Issue receipt" on the appointment for the admin and the visit's own practitioner, for a visit that is
  booked or fulfilled (not cancelled, not a no-show); it opens C's dialog with the visit's patient as the buyer.
- **Membership screen at `/membership`.** D's screen was at `/clinic/subscription`, which the Vite proxy and the
  production Caddyfile send to the API: a reload or a link showed the API's JSON. It moved to `/membership` (as B's
  price list is at `/price-list`); the API route `GET /clinic/subscription` is unchanged.
- **Mobile.** The booking screen asks for the prices of the slot's own mode.
- **API descriptions.** "(404 until then)" removed from every built route (B, D and SAL-33's).
- **e2e.** `TestReceipts_AVisitAtTheClinicIsReceiptedReportedAndReversed`: price list, an in-person visit, the
  receipt refused before the billing details, issued, reported by the fake, reversed, the reversal reported, the
  report summing to zero and the patient seeing both. A's e2e setup of an in-person clinic is the shared helper
  `atTheClinic`.

- **Review round** (a fresh subagent over the whole diff against `origin/main`; A had none). Fixed:
  - A receipt whose fiscal report failed is not reversed: 409 `fiscal_report_failed` (new error code), and no Reverse
    button for it. Its reversal would name a fiscal number the receipt never got, so it could not be reported either
    and would hold a number of the sequence for good. A pending receipt may still be reversed (its reversal is
    reported after it).
  - A receipt has at most 100 lines and a quantity of at most 10,000 (service and OpenAPI), so its total cannot
    overflow an int64.
  - The membership screen and banner show dates in the clinic's time zone, not the browser's.
  - The reversal question takes the keyboard focus and gives it back to Reverse on Keep; Issue receipt is disabled
    while a line is incomplete.
  - The membership screen's path (above).

  Declined, for later packages: the fake fiscal reporter forgets its receipts when the worker restarts, so a later
  reversal is rejected (development only; package G replaces the fake); `POST /clinic/receipts` has no idempotency key,
  so a retry after a lost response issues a second receipt (package H); the fiscal job holds its transaction during the
  report call (fine for the in-memory fake; package G, with the real authority's latency).
- **Browser walk** (headless Chromium from the e2e Playwright service, desktop 1280 px and 390 px, a fresh stack): working
  hours with visit types, `/price-list`, `/receipts`, its report, the receipt dialog on the appointment, the reversal
  question, `/membership` and the past-due banner. Fixed: the report overflowed at 390 px and C's table styles reached
  D's invoices table (scoped to `report-table` and `report-facts`); the membership invoices table now scrolls inside its
  panel at 390 px; spacing on the price list and above Issue receipt.
- **e2e on the main checkout's stack.** Two journeys unrelated to the wave (the consultation pipeline's translated
  transcript and the term review's first entry) failed twice against the main checkout's long-lived dev database and
  passed on a fresh stack, both on `origin/main` and on this branch; the whole suite passed on a fresh stack.

## Decisions for wave 2

Applied as the agent's defaults on 2026-11-22, owner to confirm (`IMPLEMENTATION_PLAN.md`, "Wave 2 packages"):

| Topic | Decision |
| --- | --- |
| A payment of a past-due or suspended clinic | Returns it to active |
| Membership payment term | 14 days after issue |
| First membership invoice | On the first 1st after the trial ends |
| Membership invoice numbers | Gapless |
| Currency | BAM only |
| A company's invoice in Croatian | Headed "Račun" |
| The accountant's CSV | Readable headers in the clinic's language, amounts as decimals; semicolons and decimal commas, as Excel reads a file with Bosnian settings (owner's default, wave 2 integration) |

## E: service on a booking and the online practice report

Status: built, 2026-11-23, on `wt/cms/E`, one commit `cms(E)`. Migration `00062_appointment_service.sql`. No pull
request: the wave's integration opens it.

Built:

- A booking may name a service: `appointment.service_id` (nullable; held to the appointment's own clinic by a composite
  foreign key to `charge_item_definition (id, organization_id)`, which gains that unique key; no `UPDATE` grant, so a
  reschedule keeps it), `serviceId` on `POST /appointments` and on the JSON `Appointment`. `appointment.Service.Book`
  takes the service id and asks `chargeitemdefinition.Service.Offered` (the new `Services` port, the api's only) in the
  directory before the booking's transaction, as the slot is: a service the clinic does not offer, active, in the
  slot's mode is `400 invalid_service`, audited. Nothing is charged.
- The clinic sees it: the FHIR Appointment carries the extension
  `https://salus.invalid/fhir/StructureDefinition/appointment-service` (a reference to `ChargeItemDefinition/<id>`,
  `api/fhir/StructureDefinition-appointment-service.json`, golden `appointment_service.json`); left out at the name
  level. Web `AppointmentDetail.tsx` names the service ("Service: …", from the clinic's price list) and passes it to
  C's `ReceiptDialog.tsx`, whose first line starts with it (`serviceId` prop, the only change there).
- The online practice report, `GET /clinic/reports/practice?from=&to=` (RFC 3339, half-open, as the receipts report):
  `clinicstats.Service.Practice` (`service_practice.go`, `handler_practice.go`), per practitioner role and visit mode:
  minutes offered by the role's availability rules, visits held (fulfilled appointments starting in the period), and
  the receipted amount of those visits. Sources through their services: `organization` (the zone),
  `schedule.Service.OfClinic` (new, the clinic's schedules in its scope), `appointment.Service.Visits` (new, opaque ids,
  modes, statuses), `invoice.Service.List` (C's, admin only, audited by invoice). Admin only; the read is audited with
  the clinic as the entity, a refusal too. Wired in `internal/api/practice.go`.
- Web `OnlinePractice.tsx` at `/receipts/practice` (admin), linked from the receipts report; `practice.*` strings in the
  `receipts` catalog, `calendar.bookedService`, in the five staff languages.
- Mobile `service_picker.dart`: the booking screen's prices became a choice, "I will choose at the clinic" first and
  the default, then each service with its price; `Booking.book` sends the pick. B's `ServicePrices` widget, now
  unused, is deleted (`priceText` stays). One string, `serviceAtClinic`, in the ten languages.
- `docs/privacy/dpia.md`: "Package E". Ownership unchanged (a column on `appointment`); its retention is the row's.

Decisions and readings:

- "Hours offered" are the availability rules as they are now projected over the period (rule history is not kept):
  a past period shows today's rules, not the ones then. Slots are not used: free slots are deleted once they end.
- "Visits held" are fulfilled appointments; the receipted amount counts only receipts of those visits, issued from
  the period's start until now; a reversed receipt and its reversal are both left out. A no-show's receipt is not
  counted.
- The report is the admin's only: it is revenue per practitioner. A practitioner gets 403, audited.
- The service on the FHIR Appointment is an extension with a Reference, not `serviceType`: R4's `serviceType` is a
  CodeableConcept and a clinic's own services have no code system (CLAUDE.md "no free-text codes").
  `ChargeItemDefinition` is not served over FHIR; the reference is logical.
- A service made inactive later stays on the appointments that named it; the receipt dialog then starts with an empty
  pick (its list holds active services only).
- `Book(ctx, c, slotID, serviceID)`: every caller passes `""` but the handler.
- The report's sources are read outside its audited transaction except the visits (schedules and the clinic hold no
  personal data; the receipts' read is audited by invoice in its own transaction).

Departures from the file table: `internal/schedule/service.go` gains `OfClinic` (no new query: the existing `Listed`
in the clinic's scope); `web/src/api/fhir.ts` `Extension` gains `valueReference`; `web/src/ClinicReports.tsx` gains the
link; `mobile/lib/booking.dart` and `service_price.dart` (above); `web/src/appointmentService.ts` is new.

Copy waiting for the owner: web `calendar.bookedService` ("Service: {name}"), `receipts.practice.*` (14 strings,
`web/src/i18n/receipts/en.json`); mobile `serviceAtClinic` ("I will choose at the clinic").

Deferred: an `internal/api` test of the report against a real database (the service's equality test runs on fakes of
the raw appointments and receipts); an e2e journey for the service at booking; the browser walk at desktop and 390 px
(no browser in this unattended session; the screens are covered by `OnlinePractice.test.tsx`,
`AppointmentDetail.test.tsx` and the mobile `book_test.dart`, whose 390 px layout test now holds the picker).

Tests first: each behaviour's test was written and run before its code, except the repository's `Visits` and the
FHIR extension, whose tests were written in the same step as the code.

Checks (2026-11-23, compose project `salus-cms-e`): `runner.sh full` (`make gen-clients lint test archcheck web-lint
web-typecheck web-test web-build mobile-analyze mobile-test`, `make i18n-check` inside lint) passed, `gen-clients` left
no further diff; `make cover` passed (90.9% total; appointment 85.3%, clinicstats 89.7%, schedule 91.7%);
`make fhir-validate` passed (the new golden included); `make e2e` passed. In the fast layer `cmd/scheduler`'s
`TestRun_StartsRiverAndStopsCleanly` timed out once while every package ran at once beside the Flutter tests, and
passed on its rerun and in the full run: load, not this package.


## H: receipts follow-ups

Status: built, 2026-11-23, on `wt/cms/H`. Migration `00063`. No pull request: the wave's integration opens it.

Built:

- **Line order.** `invoice_line.position` (`00063`, the existing lines numbered by name, the order they were read in,
  with the no-update trigger disabled for that one statement); `InvoiceLines` reads by position; `Insert` writes each
  line's index. Tested against Postgres through `ByID`, `Locked` and `Issued` (the report and the PDF read those).
- **PDF labels in catalogs.** `internal/invoice/templates/pdf/<language>.json` in the five staff languages (embedded,
  ICU with `{number}` and `{reversed}`), replacing the Go map in `service_pdf.go`; `scripts/i18n-check.mjs` checks them
  against `en.json` (`checkFlat`, shared with the one-time link's page; a test in `i18n-check.test.mjs`). A company's
  invoice is "Račun br." in Croatian, "Faktura" in Bosnian and Serbian (tested).
- **Retry of a failed report.** `Service.Retry` (`service_retry.go`), `POST /clinic/receipts/{id}/retry` (202, the
  receipt as it is), admin only, `failed` only (409 `fiscal_report_not_failed`, new code), enqueues the same
  `invoice_fiscalize` job; audited as an update, refusals too. Web: "Report again" on a failed receipt in `Receipts.tsx`.
- **Idempotency key.** `invoice.idempotency_key` (uuid, unique per clinic where set). `POST /clinic/receipts` takes an
  optional `Idempotency-Key` header (a UUID, else 400 `invalid_request`); the service answers the receipt already issued
  with that key (201, audited as a read) and looks again after `NextNumber` locks the clinic's sequence, so two
  concurrent requests with one key issue one receipt (the loser rolls back, giving its number back, and answers the
  winner's). A key issued by another account is `invalid_receipt`. `ReceiptDialog.tsx` makes one key per dialog
  (`crypto.randomUUID()`) and sends it with every attempt.
- **The accountant's CSV.** Headers, section names and payment methods in the clinic's default language (from the same
  catalogs, `csv_*` keys; `Report.Language` from `Clinics`), amounts as decimals with a point (`106.70`).
- **Mobile.** `receipts_screen.dart` ("My receipts" on the home screen): the patient's receipts, newest first, with
  number, date (Europe/Sarajevo: a receipt carries no zone), lines, total, a reversal marked, and the PDF to save or
  share (`Booking.receipts`, `Booking.receiptPdf`). Seven strings in the eleven ARB files; `receiptsIntro` is `legal`.
- `docs/privacy/dpia.md`: package H's part. `ownership.yaml` and the retention rules unchanged (no new table; the new
  columns live and go with their rows).

Decisions and readings:

- Any `failed` receipt may be retried, rejected ones too: the status does not say why it failed, and a rejected one
  rejected again stays failed (the job ends). A pending one is refused (its job is still running).
- The CSV's amounts use a decimal point in every language, the CSV a comma separator (RFC 4180): unambiguous for any
  spreadsheet import. A semicolon CSV with decimal commas for a Bosnian Excel is the owner's call.
- The CSV's labels sit in the PDF catalog files (one catalog for the receipt's printed words), not in a second
  directory.
- The idempotency key is kept on the receipt (Postgres), not in Redis: it must hold as long as a retry may come and
  never be the only copy of anything; it adds no personal data.
- The catalog is read once into a package-level value from the embedded files (as the link page's are read in its
  constructor); it never changes after.
- TDD: the repository, service and handler tests, the mobile tests and the i18n-check test were written before the
  code; the Go, web and i18n-check ones were not run red before the code was written (they were run once, green, after
  both); the mobile tests were seen failing first.

- `migrations/clinic_system_contract_test.go`'s `addLine` writes `position` 0: the column has no default (a line's
  index is the repository's to give, and a default would hide a forgotten one).

Checks (2026-11-23, compose project `salus-cms-h`): `make lint` (with `make i18n-check`) passed after one fix (field
alignment in a new test); `make archcheck` passed; `make cover` passed on its rerun (90.8% total, `invoice` 88.2%): its
first run failed in `internal/schedule`'s `TestHTTPHandler_SendsAFixedErrorAndLogsNoAvailability`, which checks that a
log line does not contain "09:00" and failed because the log's own timestamp was 08:09:00 (a wall-clock flake, not
this package's); `make gen-clients` run after the contract change; `make web-lint`, `make web-typecheck`,
`make web-test` (1111 tests), `make web-build`, `make mobile-analyze`, `make mobile-test` passed. Not run: `make e2e`
(not in H's check sets), `make fhir-validate` (no golden FHIR file changed), the browser walk.

Deferred: a migration test of `00063`'s numbering of lines issued before it (the dev databases hold none from
production; covered by reading the SQL); the browser walk of "Report again" and the mobile screen on a device (see
the report); the drawn QR (package G).


## I: membership invoice document and the suspended listing

Status: built, 2026-11-23, on `wt/cms/I` (one commit `cms(I)`). Migration `00064_directory_membership.sql`. No pull
request: the wave's integration opens it.

Built:

- `internal/subscription/service_document.go`: `PDF` (the admin's download, `GET /clinic/subscription/invoices/{id}/pdf`,
  403 for the clinic's other staff, 404 for anyone else and another clinic's invoice) and `SendInvoice` (the
  `subscription_invoice_email` job): the invoice laid out with `pkg/pdfdoc` in the clinic's default language (the five
  staff languages, Bosnian for any other), the dates in its zone: the seller, the invoice's number, issue, due and paid
  dates, the clinic as the buyer (legal name or name, tax id, address), the month's line (seats times the price per
  seat), the total and the bank transfer with the number as the reference.
- The e-mail: `issue` enqueues `subscription_invoice_email` (`InvoiceEmailArgs`, opaque ids, the notifier's `email`
  queue, `subscription.NewRiverJobs`) in the invoice's transaction, so a second run of the month enqueues nothing. The
  notifier works it (`invoiceEmailWorker` in `internal/worker/subscription.go`): the PDF goes to the clinic's billing
  e-mail through `notification.Service.SendMembershipInvoice` (template `membership_invoice.txt` in all ten languages)
  as an attachment; a clinic without a billing e-mail gets none. `notification.Message` gained `Attachment` (a pointer,
  so `Message` stays comparable), sent by the SMTP sender as `multipart/mixed` (tested against Mailpit).
- The seller: `config.LoadSeller` (`SELLER_NAME`, `SELLER_TAX_ID`, `SELLER_ADDRESS`, `SELLER_ACCOUNT`), each its
  `[SELLER …]` placeholder when unset, in `config.Config.Seller` for the api and the notifier; `.env.example` lists them.
- The suspended listing: migration 00064 rewrites the directory policy `organization_directory`: a listed clinic whose
  membership is `suspended` or `cancelled` is not in the directory scope. The directory policies of practitioners,
  roles, services, schedules and slots read `organization` through it, so all of them hide with it. A new select
  policy `clinic_subscription_directory` lets the directory scope see the membership of a stopped clinic only. No grant
  changes: the app still has no `UPDATE` on `subscription_invoice`.
- Web: `/membership`'s invoices table has a PDF column (a link per invoice, with an accessible name naming its number);
  at 480 px and less it is a card per invoice, every cell under its column's name (`data-label`), instead of the
  sideways scroll of the wave 1 walk. Catalog `subscription`: `col.document`, `download`, `downloadLabel` in the five
  staff languages.
- `api/openapi.yaml`: the PDF route (`x-salus-package: cms-i`); `make gen-clients` run. DPIA: "Package I".

Readings and decisions (agent, for the owner to review):

| Topic | Decision |
| --- | --- |
| Where the directory hides a stopped clinic | Row-level security (migration 00064), not a Go filter: one policy hides the clinic and everything the directory shows under it, and `organization` reads no table of `subscription` in Go (ownership). D's `bookableClinics` wrapper stays for the booking's own refusal |
| Which statuses hide | `suspended` and `cancelled`, as D's `BookingsStopped`; `past_due` stays listed (Q4: the stop is at 30 days) |
| The api test of the hiding | The api tests connect as the owner, which bypasses row-level security, so the hiding is tested as `salus_app` in `migrations/directory_membership_test.go`; the api test checks the PDF route and that care goes on |
| The e-mail is not recorded on the invoice | The owner's note: no `UPDATE` grant on `subscription_invoice`. River keeps the job's state; a failed send is retried by River, a missing invoice or an address SMTP cannot carry cancels it |
| A clinic without a billing e-mail | No e-mail; the admin downloads the PDF from `/membership` |
| PDF labels | A Go map in `service_document.go` for the five staff languages (as C's receipt before H moves it); the invoice is for the clinic's staff |
| The seller unset | Prints its placeholder (`[SELLER]` …) instead of refusing to start: the dev stack runs, and no invoice can look finished before Q3 |
| One invoice by id | Read from the clinic's list (a clinic has one a month), no query of its own |
| The PDF's file name | `salus-invoice.pdf` on the download, `salus-invoice-<number>.pdf` on the e-mail |

New user-facing strings (for the owner): the invoice PDF's labels (five languages, `service_document.go`); the e-mail
`membership_invoice.txt` (ten languages; subject "Salus membership invoice no. {number}"); web `subscription.col.document`
("Invoice"), `subscription.download` ("PDF"), `subscription.downloadLabel` ("Download invoice {number} (PDF)").

Tests first: each step's test was written and seen failing before its code (the service, the document, the SMTP
attachment, the notification, the handler, the migration's policy, the web link).

Checks (2026-11-23, compose project `salus-cms-i`): `runner.sh full` (`make gen-clients lint test archcheck web-lint
web-typecheck web-test web-build mobile-analyze mobile-test`) passed on its second run; the first failed on four struct
layouts (`fieldalignment`), fixed. `make cover` passed (90.8% total; `subscription` 86.4%, `config` 98.3%, `worker`
90.1%). `make i18n-check` passed inside lint. Not run: `make e2e` (not in I's check sets), `make fhir-validate` (no
golden FHIR file changed).

Layout at 390 px: the invoices table's markup and `app.css` rendered in headless Chromium (Playwright 1.65) at 1280 and
390 px: no sideways scroll at 390 (table 326 px in a 358 px panel). Not walked against a running stack with a signed-in
admin.

Deferred:

- A browser walk of `/membership` against the full stack (seeded membership and invoice, admin sign-in with MFA): the
  integration session's walk covers it.
- No worker-level test runs `subscription_invoice_email` through River into Mailpit; the parts are tested (the service
  with fakes, the SMTP attachment against Mailpit, the notification's message).
- The PDF labels stay in Go; if H's `templates/pdf/<language>.json` pattern is wanted here too, a follow-up moves them.


## L: landing page

Done 2026-11-23, one commit `cms(L)` on `wt/cms/L`. Only `landing/` changed, plus this entry. Tests first: the
headline, section order, prerendered headings, pricing and the new section's tests were changed or written and seen
failing before the copy and components.

**Built.**

- The page sells Salus as the clinic system with telemedicine built in: new hero eyebrow and headline ("Your clinic,
  in person and online."), highlights, four product cards (front desk, doctors and admins, patients, online hours from
  home; a new receipt icon in the same stroke), "From booking to receipt" steps, and the page title and description in
  `index.html`.
- A new short section "Patients abroad" (`sections/Abroad`, `content/abroad.ts`, anchor `#abroad`) between "How it
  works" and "Standards": ten app languages, live translation as a paid add-on per use, and prepaid booking from abroad
  tagged "Coming".
- Pricing: two plans, Basic and Pro, `[PRICE]` per practitioner per month, a note on the 30-day trial and paying only
  for active practitioners, and that patients pay at the clinic. The yearly plan and the `[X]%` share are gone; a test
  keeps "year", "%" and "share" out of the pricing copy.
- Structure, look and language (English) unchanged; the nav keeps its four links.

**Decisions.**

| Topic | Decision | Why |
| --- | --- | --- |
| What Basic and Pro hold | Pro adds video visits and online hours, live transcription and AI drafts | The plan's Q2 default: `pro` is video, transcription and AI |
| Fiscal wording | "Built for the new FBiH fiscalization law; reporting to the Tax Administration through ESET is coming" | No certification claim; G (ESET adapter) is on hold |
| No nav link for the new section | Nav unchanged | "Structure stays"; four links fit at 390 px |
| Product grid | Two columns (`minmax(min(100%, 440px), 1fr)`) | Four cards in three columns left one alone |

**Checks.** `make landing-lint`, `landing-typecheck`, `landing-test` (18 files, 48 tests), `landing-build`: pass. Walked
the built page at 1440 and 390 px (headless Chromium, reduced motion): no horizontal scroll; the plan's unit wrapped
mid-phrase at 390 px and was fixed.

**Left.** CLAUDE.md's landing row still lists the `[X]` placeholder, which the page no longer has (outside this
package's files). The new copy waits for the owner's approval.

## Q1: questionnaire branching, rules and server

**Status: done, 2026-11-23** (worktree `wt/cms/Q1`). Ticket SAL-43. No migration: `00065` is left unused.

**Built.**
- **The model is FHIR R4 `enableWhen`.** `questionnaire.Item.EnableWhen` is a list of `Condition`
  (`question`, `operator`, one `answer[x]`: `answerBoolean`, `answerString`, `answerInteger`, `answerDecimal`,
  `answerDate`) plus `enableBehavior` (`all`, `any`). The FHIR read serves the model's conditions as they are; the
  JSON API (`api/openapi.yaml` `QuestionnaireItem`) now takes and returns the FHIR array. The single object of before
  (`{question, answerBoolean | answerString}`, operator `=`) is still decoded, from stored rows and from requests,
  so existing drafts and published versions keep working without a migration.
- **Publishing refuses broken logic** (`internal/questionnaire/validate_logic.go`, on every save and so on
  publish, 422 `invalid_questionnaire` naming the item): a condition on a missing, later or own-descendant question,
  on a section, instructions or a file upload; an operator that does not fit the type (`=`/`!=` on yes/no and choice,
  every comparison on integer, decimal and date, `exists` on any question, string and text only `exists`); not exactly
  one answer of the type (`exists` takes `answerBoolean`); an option that no longer exists; a malformed decimal or a
  date that does not exist; two or more conditions without `enableBehavior`, an unknown behavior, a behavior without
  conditions. Sections now nest (a branch is a section carrying the condition); an empty one at any depth is refused
  on publish (que-1).
- **The server evaluates the conditions on submit** (`internal/questionnaireresponse/service_enable.go`, was
  `enabled.go`) and refuses saved answers to hidden items: 422 `answer_not_enabled` with their `linkIds` in document
  order, audited as a refused submit. Files of hidden attachment items are still removed on submit.
- **Shared test vectors** `test/testdata/enablewhen/*.json` (README there): today's `is` conditions, every operator
  and type, `all` and `any`, nesting with a join, the ticket's gynaecology example, the changed-answer case. Both Go
  packages run every case (`TestPublish_AcceptsEveryEnableWhenVector`, `TestSubmit_EvaluatesEveryEnableWhenVector`);
  `questionnaire_branching.json` is a new golden FHIR read.

**Readings and decisions.**
- Every operator but `exists` is false on an unanswered or hidden question, `!=` included (R4 leaves it open; R5
  says so). `!=` on a multiple choice holds when no answer equals.
- `answer_not_enabled` refuses rather than drops, as the plan says. Today's mobile app saves answers of hidden items
  and relied on the server dropping them: after Q3 it saves only the shown answers; until then a patient who changed
  an answer that hides a branch gets the refusal on submit.
- A condition on a file upload is refused (its answers live in `binary`, outside the answers the conditions read);
  the ticket's "any question: answered" covers every other question type.
- The JSON API's `enableWhen` changed to the FHIR array (the plan's contract); `web/src/questionnaire.ts` and its
  test fixture were adapted in two lines each so the web app keeps building until Q2 replaces them.
- `docs/questionnaire/SPEC.md` §2 "Item model" (one `=` condition, sections top level only) is superseded by this
  package; that plan's docs are not edited here.
- TDD: the tests were written before the code each time; the first runs failed on the build (the model change)
  rather than on an assertion.

**Checks.** Fast layer after each change; `make gen-clients lint test archcheck web-lint web-typecheck web-test
web-build mobile-analyze mobile-test` passed (lint includes `i18n-check`), `make cover` passed (questionnaire 93.6%,
questionnaireresponse 93.1%), `make fhir-validate` passed (the new golden: 0 errors, the usual dom-6 warning).
`gen-clients` dropped the stale generated `questionnaire_item_enable_when.dart` (git rm: the generator leaves it).

**Integration notes.** Run the vectors against `enableWhen.ts` and `enable_when.dart`. Q2 and Q3 should read and
write `enableWhen` as the array (the JSON API's response no longer carries the single object), Q3 should save
without hidden answers before submit and handle `answer_not_enabled`. Regenerate the clients after the merges.
`openapi.yaml`'s `Error.error` enum gained `answer_not_enabled` at its end.

## Q2: questionnaire branching, builder and preview

Done 2026-11-23 (ticket SAL-43, web only; no Go, no mobile, no `api/openapi.yaml` change).

- **Model.** An item's condition is now FHIR R4 `enableWhen[]` (`question`, `operator` of `exists = != > < >= <=`,
  the value) with `enableBehavior` (`all`/`any`) when it has more than one (`builder.ts`). Operators per type
  (`operatorsFor`): yes or no and choices is / is not / answered; numbers every comparison; dates before / after / on;
  every other question answered / not answered. Any earlier question can be depended on (file uploads only by
  answered). W28 checks each condition: the question is above (`conditionOrder`), the operator fits its type
  (`conditionOperator`, new), the value fits (`conditionAnswer`, now any type).
- **Evaluator** `enableWhen.ts`, pure and tested: `test` (one condition), `enabled` (document order, a hidden
  section hides its items, a hidden question counts as unanswered), `enabledAnswers` (the path taken),
  `discarded` (answers a change would hide). `responses.ts` (the practitioner's view of a submitted response) now
  uses it too instead of its own `is`-only check.
- **Builder** (`builder-logic.ts`, `BuilderBranch.tsx`, `ItemCard.tsx`, `Builder.tsx`): on a yes or no or choice
  question, "Branches": per answer "If the answer is X", the questions it asks, and "Ask a question", which adds a
  short answer shown only on that answer at the end of the question's branches (`addFollowUp`). Cards are indented
  by branch depth (`branchDepths`, a mint left border), tagged "Conditional", and "Branches join here" marks the
  first item back at a shallower depth (`joins`). The condition editor ("More options") has several conditions,
  "Show when all / any", a comparison and a value control per type. W19 deleting names the affected items; an
  opened question others depend on says which ("Shown only through this question: ...").
- **Preview** (`BuilderPreview.tsx`): "Preview" beside Publish shows the draft as the patient fills it in, branches
  live; a change that hides answers already given asks "Change this answer?" (Keep answer / Change answer); "Start
  again" clears it. Nothing is saved.
- **Wire (decision).** A single `=` condition on a yes or no or a choice is still sent in the JSON API's legacy
  object shape (`{question, answerBoolean|answerString}`), so today's forms save unchanged on today's server; any
  other logic is sent as FHIR `enableWhen[]` (`answerBoolean|answerString|answerInteger|answerDecimal|answerDate`)
  plus `enableBehavior`. Reading takes both. The request type widens the generated one (`questionnaire.ts`), with one
  cast in `Builder.tsx` until Q1's API and `make gen-clients` land.
- **Not built (decision).** "Any of" on a choice is several `=` conditions with "any" (FHIR has no `in`). Branches
  are items carrying the condition or a top-level section; sections still do not nest (API rule), so a nested
  branch is a conditional question inside a conditional branch. "Ask a question" adds a question, not a section.
- **Checks.** `make web-lint`, `web-typecheck`, `web-test`, `web-build`, `make i18n-check`, and the runner's full
  set: see the report. TDD: the evaluator, builder-logic, the wire tests and the Builder tests were written before
  or with their code in this session; the old tests that stated the single-`is` model (answer label "is", "a short
  answer cannot be depended on", the W19 text) were changed on purpose.
- **Browser walk** (headless Chromium from the playwright service, this worktree's stack, a synthetic clinic,
  1280 px and 390 px): built the gynaecology example with "Ask a question if the answer is Yes" twice (whole numbers)
  and a main question; the join marker shows; "Save draft" saved (legacy shape accepted by today's API); the preview
  showed 2 items, 4 on Yes, back to 2 on No; the warning appeared after a month was entered, Keep answer kept 3,
  Change answer discarded it. No horizontal scroll at 390 px. Fixed on the way: the preview's `.preview` class
  collided with the device check's video box (renamed `q-preview`).

INTEGRATION NOTES: Q1 must accept the FHIR `enableWhen[]` + `enableBehavior` on the JSON save (and keep the legacy
object); after it, `make gen-clients` and drop the widened types in `questionnaire.ts` and the cast in `Builder.tsx`.
Run Q1's `test/testdata/enablewhen/` vectors against `enableWhen.ts` (`enabled`, `test`). The web e2e of both paths
can reuse the walk's steps (labels above).

## Q3: questionnaire branching, patient app

Done 2026-11-23, one commit `cms(Q3)` on `wt/cms/Q3`. Mobile only (ticket SAL-43, its patient-app part).

**Built.**

- `mobile/lib/enable_when.dart`, pure: `Condition` (question, `Operator`, answer in the question's type),
  `EnableBehavior` (all, any), `enabled()` over several conditions with the R4 operators `exists`, `=`, `!=`, `>`, `<`,
  `>=`, `<=` on yes/no, choice, integer, decimal, date and text answers, nested groups, and `discarded()` (the answered
  items a change hides). `questionnaire.dart` reads every `enableWhen` and `enableBehavior` (the old single
  `condition` is gone; today's single `=` conditions read as one condition and behave as before) and re-exports it.
- `toApi` sends only enabled questions' answers, so a save or submit holds only the path taken (Q1's
  `answer_not_enabled`). A response opened with answers to hidden items drops them.
- The form shows and hides branches live. Before a save that would drop answers already given, the patient is asked
  ("Change this answer?", how many answers go): Cancel puts the answers back to the ones the server has (the field's
  text too) and nothing is sent; "Change and remove" removes them, the hidden items' files too, then saves. A typed
  answer asks when typing pauses or the field is left, not per keystroke; a choice asks at once.
- Three strings (`discardTitle`, `discardBody` with a plural, `discardConfirm`) in all ten languages (eleven ARB
  files, `sr` mirrors `sr_Cyrl`).

**Checks.** `runner.sh full` (`make lint test archcheck mobile-analyze mobile-test`): pass, 622 mobile tests, no
analyzer issues, `i18n-check` pass. No device walk: running the app on a device is a manual step (CLAUDE.md "Checks").

**Decisions.**

| Topic | Decision | Why |
| --- | --- | --- |
| `!=` on an unanswered question | Holds (shown) | R4 4.0.1: "true if no answer has a value that is equal"; the integration session checks Q1's vectors agree |
| Comparison of mismatched types (a number against text) | Never holds; `>`/`<` on text or yes/no never hold | Publishing refuses such an operator (Q1); the app does not guess |
| Unknown operator | Never holds (hidden) | As an unknown item type is not shown |
| When the warning shows | At the save that would drop answers (a choice at once, typing after its pause) | Hiding is live, so a half-typed number does not ask on each keystroke |
| Declining | Back to the server's answers, not only the changed field | One snapshot; edits since the last save are at most the change being taken back |

**Test first.** The evaluator and screen tests were written before their code, but the first run came after the code
(they compiled only with it); the `toApi` test was seen failing first.

**Left to the integration session.** Run Q1's `test/testdata/enablewhen/*.json` vectors against `enable_when.dart`; the
mobile e2e of both paths of the gynaecology example and the changed-answer case.

## Wave 2 integration

Done 2026-11-23 on `feat/cms`: `wt/cms/Q1`, `Q2`, `Q3`, `H`, `I`, `E`, `L` merged with `--no-ff` in that order, then
one commit `cms(wave 2): integrate E, H, I, L, Q1, Q2, Q3`. `origin/main` had not moved (`df5480d`), so the migrations
keep their numbers: E `00062`, H `00063`, I `00064`; `00065` (Q1's) is unused.

- **Merges.** Conflicts only in shared files: `web/src/questionnaire.ts` (Q2's taken, then reworked below),
  `api/openapi.yaml` (both error codes kept), the mobile ARB catalogs (merged as JSON: every package's keys), the DPIA
  (every package's block), and generated code. Generated code was not hand-merged: `make sqlc` (E's column changes
  every domain's `models.go`), `make gen-clients` and `flutter gen-l10n` ran after the merges.
- **Branching vectors in every evaluator.** `test/testdata/enablewhen` now runs as tests in the repo for web
  (`web/src/enableWhen.vectors.test.ts`, the items read through the builder's own FHIR read) and mobile
  (`mobile/test/enable_when_vectors_test.dart`, read as the app reads the API), beside Q1's two Go packages. Web
  passed all 37 cases as merged; mobile failed 11, all from Q3's `!=` holding on an unanswered question. Fixed to
  Q1's rule (every operator but `exists` is false on an unanswered or hidden question); Q3's unit case changed with it.
- **Web writes FHIR only.** Q2's widened `JSONItem`/`SaveRequest` types and the cast in `Builder.tsx` are gone: the
  builder reads and writes the generated `enableWhen` array and `enableBehavior`, a single `=` too (no legacy object);
  the server still accepts the legacy object on input. Two tests that pinned the legacy wire shape changed on purpose.
- **Mobile.** Saves send only shown answers (Q3); a submit refused with 422 `answer_not_enabled` now names the
  questions (new string `answersNotEnabled`, ten languages) and saves the shown answers again, so the next submit
  passes. Groups inside groups are indented one step per level, in the form and the read-only view.
- **The practitioner's view of a response** (`web/src/responses.ts`) walks sections at any depth: a branch inside a
  section was dropped with its answers (review finding).
- **e2e.** `TestQuestionnaire_TheGynaecologyExampleKeepsOnlyThePathTaken` (the yes path at 17, the no path, and yes
  changed to no: refused with `answer_not_enabled` naming the branch's two questions, accepted once the app's save
  holds the path taken) and `TestQuestionnaire_TheBuilderBranchesTheGynaecologyExample` (the web builder builds it
  with "Ask a question if the answer is Yes", the preview takes both paths and the changed answer, publish; its FHIR
  read has the `enableWhen` arrays and the unconditional join). The older journey that saved an answer of a hidden
  question now expects the refusal, then the re-save. Mobile's side is widget journeys against the fake API
  (`questionnaire_screen_test.dart`, the shared `gynaecology.json`): there is no device e2e in this repository.
- **Receipts and booking.** `ReceiptDialog` keeps E's first line from the booked service and H's `Idempotency-Key`;
  the key now belongs to the receipt as drafted, so a double click or a retry issues it once and a receipt changed
  after a failed attempt gets a new key (review finding; before, the server answered the changed receipt with the
  first one). New: `TestNew_ThePracticeReportCountsOfferedHeldAndReceiptedWorkByMode` (internal/api, Postgres),
  `TestReceiptLinePosition_TheBackfillNumbersExistingLinesByName` (00063 on lines issued before it),
  `TestReceipts_AServicePickedAtBookingStartsTheVisitsReceipt` (e2e), and a hint beside "Hours offered".
- **Retry.** `Retry` locks the receipt and `invoice_fiscalize` is unique by receipt while one is queued or running, so
  two clicks report it once (`TestNew_RetryingAFailedReceiptQueuesOneReport`); the web hides "Report again" for a
  receipt retried from the page (review finding).
- **Accountant's CSV.** Semicolons and decimal commas in every language (owner's default for Excel with Bosnian
  settings); a field with a semicolon, a quote or a line break is quoted (RFC 4180 with `;`).
- **Membership.** The invoice PDF's labels moved from the Go map to `internal/subscription/templates/pdf/<language>.json`,
  checked by `make i18n-check` as the receipt's are; `TestNotifier_SendsAnIssuedMembershipInvoiceWithItsPDF` runs the
  monthly job and the notifier to the compose Mailpit and checks the subject and the PDF attachment.
- **Flaky tests.** `cmd/scheduler`: River gives its notifier's listener 10 s to connect, so under load `Start` itself
  returned a timeout; `run` now tries River's start up to 3 times, 2 s apart (River resets a client whose start
  failed), with tests of its own. Measured under the full suite: the test body took under 3 s; the rest of its 16–37 s
  was the test database's drop. `internal/schedule`: the handler test's logger left the record's time in the log,
  which held "09:00" at 08:09:00; the test logger now drops the time, and the check also looks for "12:00".
- **Docs.** `docs/questionnaire/SPEC.md` (§1, item model, linkIds, submit, D4, out of scope, up next, personal data)
  states the branching model; CLAUDE.md's landing row no longer lists `[X]`.
- **Browser walk** (headless Chromium from the playwright service, this integration's own stack, a synthetic clinic,
  1280 and 390 px; the temporary spec and its Go wrapper are not committed): the appointment with "Service:
  Placeholder examination" and the receipt dialog starting from it; `/receipts` with "Report again" on a failed
  receipt and its notice after the click; `/receipts/practice` with the hint (at 390 px it ran off the screen's left
  edge: now centred under its heading and capped to the screen); `/membership` with the PDF download
  (`salus-invoice.pdf`); the builder with two branch questions, "Conditional", "Branches join here" and the preview on
  Yes; the landing page (with reduced motion). No horizontal scroll on any page at either width. On a phone a palette
  type is tapped, not dragged.

**Checks** (this integration's own compose project, `salus-cms-w2`; e2e on a fresh one: stack down and its own
volumes removed first). After the review's fixes: `make lint` (gofmt, vet, golangci-lint, i18n-check, migrations),
`make test`, `make archcheck`, `make cover` (90.8% total, every package at 80% or more), `make web-lint`,
`web-typecheck`, `web-test` (1201), `web-build`, `make mobile-analyze`, `mobile-test` (671), `make e2e` (346 s): all
pass; `make gen-clients` leaves no diff. Before them, on the same tree but for the review's fixes: `make fhir-validate`
and the landing set (`landing-lint`, `-typecheck`, `-test`, `-build`) pass; neither area changed after.

**Decisions taken here.**

| Topic | Decision | Why |
| --- | --- | --- |
| The CSV | `;` and decimal comma for every language | Owner's default for Excel with Bosnian settings |
| The JSON API's questionnaire writes from web | FHIR `enableWhen` array always | Q1's API takes it; the legacy object stays accepted on input |
| A submit holding hidden answers on mobile | Name the questions, save the shown answers again, submit again | The server refuses rather than drops (Q1) |
| The idempotency key | One per receipt as drafted | A changed receipt is another receipt |
| Retrying a failed report | One queued report per receipt; the button hidden after a retry from the page | One click, one report to the authority |
| The scheduler's start | Up to 3 tries, 2 s apart | River's 10 s listener connect times out under load |

**Review** (a fresh subagent over `origin/main...HEAD`). Fixed: the nested sections in the practitioner's view, the
idempotency key of a changed receipt, the retry's lock and duplicate jobs, two stale SPEC lines. Declined, with
reasons in the PR: the server also takes `!=`, `>=`, `<=` on a date (FHIR allows them; the builder offers before,
after and on; the SPEC now says so); older app builds blocked by the refusal (the patient app is not released); the
app's record of the server's answers on a response saved before Q1 (handled by the refusal and the re-save); Go's
`!=` on mismatched types (validation makes it unreachable).

**Open.** No device run of the mobile journeys (a manual step). The builder still adds sections at the top level
only (a branch within a branch is built from conditional questions). The practice report's amount column wraps at
390 px.

## F: reception role and desk

Split into F1, F2 and F3 for wave 3 (below).

## G: ESET adapter

## 3.0: the wave 3 contract

Status: done, 2026-11-23, on `wt/cms/3.0`, one commit `cms(3.0)`. Migration `00065_wave3_contract.sql`. Contract only,
as step 0: no behaviour; every new service method refuses with `errors.ErrUnsupported`, with a test that says so.

**Built.**

- **Migration 00065.** `reception` in the role checks of `account` and `invite` (no name on its invite, as an admin's).
  `patient.account_id` nullable and `patient.registered_by_organization_id` (exactly one of the two, `patient_owner`),
  with `patient_registered` (the clinic scope reads, registers and corrects the profiles it registered) and the
  restrictive `patient_registered_only_there` (no other scope sees them). `consent.recorded_by_account_id` and
  `given_in_person` (both or neither), the clinic-scope insert `consent_clinic_in_person`, and the trigger
  `consent_guard_in_person` (for every role: an in-person consent only for a patient its clinic registered).
  `appointment` status `arrived`, `arrived_at` (an in-person visit only; kept when it is then fulfilled),
  `booked_by_account_id`, the clinic-scope insert `appointment_clinic_insert` (with the clinic's active care consent);
  `appointment_change` takes `arrived` (booked to arrived, arrived to fulfilled, no-show or cancelled). The trigger
  `appointment_guard_staff` (who booked is an account of the row's clinic, in its scope, and never changes; an arrival
  is set once, in the clinic's scope, from booked) and 00054's `appointment_guard_change` replaced so the clinic's
  scope moves a booked visit as its patient does (F2's "move"; the one existing test that pinned "the clinic never
  moves one" now expects the move, on purpose).
  `clinic_signup` (token hash of 32 bytes). `quote` (written for a patient with the clinic's active care consent, by
  its own staff) and `quote_line` (`quote_line_guard_insert`: only in the quote's transaction, from its clinic's price
  list, as `invoice_line`). `organization.languages`, `profile_description`, `welcomes_international`
  and `organization_profile_translation`. Ownership and retention for each new table.
- **Go.** `auth.RoleReception` (an invite naming it is still refused, `ErrInvalidRole`, until F1);
  `appointment.StatusArrived` and `service_staff.go` (`BookForPatient`, `MoveForPatient`, `CancelForPatient`, `Arrive`,
  `DeskDay`); `patient` `service_clinic.go` (`SearchClinic`, `Register`); `consent` `service_inperson.go`
  (`RecordInPerson`); `invoice` `Quote` and `service_quote.go` (`SendQuote`, `Quotes`, `ReadQuote`, `MyQuotes`,
  `QuotePDF`); `organization` `Profile` and `service_profile.go` (`Profile`, `SaveProfile`); `chargeitemdefinition`
  `service_import.go` (`Import`); the new domain `signup` (context `identity`) with `model.go`, `errors.go`,
  `repository.go`, `service.go`, `query.sql` and `signupdb`.
- **OpenAPI.** Every wave 3 route with `x-salus-package` (`cms-f2`, `cms-o1`, `cms-o2`, `cms-x1`, `cms-x2`), the role
  enums with `reception`, `Appointment.status` with `arrived` and `arrivedAt`, six error codes (`invalid_signup`,
  `invalid_verification`, `invalid_import`, `invalid_quote`, `invalid_profile`, `not_in_person`); a contract test
  (`api/clinic_wave3_contract_test.go`) that the enums are the Go constants and every route names its package;
  `make gen-clients`. Web's hand-written FHIR `AppointmentStatus` gains `arrived`.
- **Docs.** CLAUDE.md "Product" and `docs/accounts/SPEC.md` §2 (roles, account scope, MFA) name the receptionist; the
  DPIA's "Package 3.0" block.

**Contract changes** (where the code fit better than the plan's text):

| Plan's text | As built | Why |
| --- | --- | --- |
| `/clinic/staff/invites` for a receptionist | The existing `POST /invites` with `role: reception` | Invites already are the clinic's staff invites (admin, practitioner); a second route would duplicate them |
| `invoice`: kind `quote`, `valid_until` | Own tables `quote`, `quote_line`, owned by `invoice` | A quote has no number, payment method, seller copy or fiscal status, all `NOT NULL` on `invoice`; as a kind, every receipt query, the fiscal job, the receipts report and `/me/receipts` would need a filter to keep quotes out. Separate tables cannot be numbered or reported by mistake |
| `organization.languages` as `language[]` | `text[]` checked against `enum_range(NULL::language)` | As step 0's `modes`: an enum array needs its type registered in pgx and sqlc |
| "the profile on the public clinic read" | `GET /organizations/{id}/profile` (public, as `/organizations/{id}/services`) | The directory's clinic read is FHIR `Organization`, which has no element for languages spoken or a welcome; the language filter of the clinic list is X2's (a search parameter in the CapabilityStatement) |
| `/clinic/appointments` (books, moves, cancels) | `POST /clinic/appointments`, `POST /clinic/appointments/{id}/reschedule`, `/cancel`, `/arrive` | As the patient's own `/appointments/{id}/reschedule` and `/cancel` |
| `/clinic/desk/today`, the desk's server side | Tagged `cms-f2` | F3 is the web screen only; F2 owns `appointment/service_staff.go` and its handler |
| `/clinic/export` | `POST /clinic/export`, `GET /clinic/export/{id}`, `GET /clinic/export/{id}/archive` | The archive is built by a job: ask, poll, download |
| A consent policy for "registered by this clinic" | A trigger (`consent_guard_in_person`) | A policy on `consent` reading `patient`, whose policies read `consent`, is refused by Postgres as recursive (`42P17`) |

**Decisions** (agent's, for the owner to review):

| Topic | Decision |
| --- | --- |
| A receptionist's invite | No name, as an admin's; no Practitioner |
| A clinic-registered profile | Exactly one of account and registering clinic; linking an account later relaxes `patient_owner` in its own package |
| An in-person consent | Only for a patient the clinic registered: a clinic never records consent for a patient who has an account |
| `arrived_at` | Kept when the visit is then fulfilled or a no-show; never on a video visit |
| Go fields for the new columns | Columns only (as step 0 did for `organization`): the model fields come with the reads in F2, X2, O1, so no field is always zero |
| Error codes | Six new ones; the rest reuse existing codes (`invalid_name`, `invalid_birth_date`, `invalid_phone`, `slot_unavailable`, `consent_required`, `too_large`) |

**Left to the packages** (their migrations are reserved: F1 `00066`, F2 `00067`, O1 `00068`, O2 `00069`, X1 `00070`,
X2 `00071`):

- **F1.** `validInvite` accepts `reception`; `ClinicMember` gives a receptionist's member (today it returns an empty
  `Member` for one); the `access` rules of W2 and the table-driven route test. A status label for `arrived` in web
  (none exists; nothing sets it before F2).
- **F2.** The Go fields and reads of `patient.registered_by_organization_id`, `consent.given_in_person` and
  `recorded_by_account_id`, `appointment.arrived_at` and `booked_by_account_id`; revoking an in-person consent from the
  clinic (an `UPDATE` policy); the slot hold for a staff booking (slot's policies are the patient's today); reminders by
  SMS or none for a patient without an account (the notifier reads `account_id`).
- **O1.** Where the wizard's progress is stored (a table or columns, its migration); deleting unverified signups (a
  retention rule with its applier); `salusctl signup list | approve`.
- **O2.** The export's table (status, object key, expiry) and its job.
- **X1.** The retention period of quotes; the quote e-mail.
- **X2.** The language filter of the clinic list.

**Review** (a fresh subagent over the diff and the plan docs, Copilot being out of quota). Fixed: the staff move
refused by 00054's guard (now allowed in the clinic's scope); lines could be added to a sent quote and the line policy
had no scope guard (`quote_line_guard_insert`, the guard); a quote could be written for a patient after a revocation
(the clinic's active consent is required) and by another clinic's account; `booked_by_account_id` and
`recorded_by_account_id` could name any account, and a patient's own scope could set them (both triggers check the
clinic's staff and the scope); a patient could mark their own arrival (`appointment_guard_staff`); the `patient`
retention basis and the DPIA did not cover a profile without an account (both amended: the clinic takes the
subject-rights requests in person, F2 records how); the profile translations' policy had no scope guard; the signup
token hash had no length check. Left, recorded above: the wizard's and the export's tables are O1's and O2's, in their
reserved migrations.

**Test first.** The Go skeleton tests and the contract test were written and run red (no such method or type) before
the code. The migration was written before its tests (`migrations/wave3_contract_test.go`); the review's fixes came
with their failing cases first only for the staff move (the existing test failed, then was changed on purpose).

**Checks** (this worktree's compose project, `salus-cms-3-0`): `make lint` (gofmt, vet, golangci-lint, i18n-check,
migrations), `make test`, `make archcheck`, `make cover` (90.8% total; `signup` 80%), `make web-lint`,
`web-typecheck`, `web-test` (1201), `web-build`, `make mobile-analyze`, `mobile-test` (671), `make gen-clients`: all
pass. Two existing tests changed on purpose with the staff move (the clinic case of
`TestAppointment_OnlyThePatientMovesOrConfirmsTheirBookedAppointment`, `TestReschedule_MovesTheAppointmentOnceAndTheExclusionConstraintGuards`'s last
step). The Dart generator cannot take `const: true` on a boolean, so `consentGivenInPerson` says "must be true" in its
description. Not run: `make e2e` and `make fhir-validate` (no route is served and no golden file changed).

**Copy waiting for the owner:** none. 3.0 adds no user-facing string; the API returns codes only.

## F1: reception role and what it may see

Status: done, 2026-11-23, on `wt/cms/F1`, one commit `cms(F1)`. No migration (00066 left unused): 3.0's role checks
already take `reception`.

**Built.**

- **Invite.** `POST /invites` with `role: reception` (e-mail alone, as an admin's; a name is `invalid_name`);
  acceptance creates the account with a pending TOTP secret and no Practitioner (`service_invite_reception.go`).
  `ClinicMember` gives a receptionist `Member{Reception: true}`, never admin, practitioner or term reviewer.
- **access.** `RoleReception`, outside `staffOf`, so every rule of the care record refuses it; `access.Desk`
  (`reception.go`) is W2's desk view (name, birth date, phone, appointments with service and visit type) for the
  clinic's receptionist and admin with an active care consent or a visit before a revocation, for F2's routes.
- **Composition root.** `appointmentCaller` gives a receptionist no clinic, so every domain built on it
  (appointments, compositions and document templates, receipts, the clinic waitlist, binaries) refuses the
  receptionist as a stranger. The route test found the two that answered 200 before this: `GET /clinic/waitlist` and
  `GET /document-templates`.
- **The route test** (`internal/api/reception_routes_test.go`). `receptionRoutes` classifies every operation of
  `api/openapi.yaml`, parsed at test time: `public` (must match `security: []`), `own`, `desk`, `refused`, `stranger`
  (a patient's own routes), `machine` (the LiveKit webhook). An operation not in the table, or classified but gone
  from the contract, fails `TestReceptionRoutes_EveryRouteOfTheContractIsClassified`. The second test runs every
  route as a signed-in receptionist (row-level security on, the platform analytics on) against a clinic with a
  patient, a care consent and a booked visit: `refused` answers 403 or 404, byte for byte the same for the clinic's
  appointment or patient as for an unknown id, with none of the patient's data; `stranger` is refused or gets what a
  signed-in account without a patient profile gets; `own` and `desk` are not refused; refusals are audited with the
  receptionist as agent. `notServedYet` lists the wave 3 routes no package has mounted (they answer the router's
  404 or 405); mounting one fails the test until it is taken off the list.
- **Web.** The invite form has a Reception option (no name fields) and the open invites show "Reception · Expires";
  a receptionist gets no navigation, no cheatsheet, and every clinical screen says the page does not exist (five
  staff languages, `admin.reception`, `admin.inviteReception`).
- **DPIA.** "Package F1" block in `docs/privacy/dpia.md`.

**Decisions** (agent's, for the owner to review):

| Topic | Decision |
| --- | --- |
| Deny by default | The receptionist is refused in the composition root's appointment caller, not in each domain: one place, and a domain added later refuses them too. The desk's routes open explicitly (F2) |
| Receipts | Classified `refused` for now: `invoice` allows the admin and the visit's practitioner. W2 gives the desk the receipts and F3 issues them from the row, so the integration opens `POST/GET /clinic/receipts`, `GET /clinic/receipts/{id}` and `/pdf` to the receptionist (a `Reception` flag on `invoice.Caller`, admin-like for the clinic's visits) and reclassifies them `desk` |
| Price list | `GET /clinic/services` is `desk` (any staff of the clinic reads it); editing stays the admin's |
| A patient's routes | `stranger`: the patient and consent domains refuse staff with 403, the others answer as to an account without a profile (empty lists, defaults); neither reveals anything |
| 400 before access | A write without a body (`invalid_request`) and a live stream without the WebSocket upgrade (`websocket_required`) are rejected before any access check; the test accepts that, still compared with an unknown id. Writes with valid bodies are left to each domain's own authorization tests |
| Directory ids | The clinic and its PractitionerRole are public in the directory, so their routes are not compared with an unknown id (an admin-only route there answers 403 to staff and 404 to an unknown id) |
| The receptionist's home | No screen until F3: `/desk` and its link are F3's, which should make it the receptionist's home |
| `GET /terminology/explanations` | `own`: term explanations are reference data any signed-in account reads |

**For the integration** (F2, F3, X1, O1, O2, X2 mount routes that are on `notServedYet`):

- Each mounted route comes off `notServedYet`; its class in `receptionRoutes` then holds.
- F2's desk routes need the receptionist's clinic: `appointmentCaller` returns none for a receptionist, so F2's handler
  takes a caller with `Reception` (or `appointment.Caller` gains the flag) and `access.Desk` for the patient reads.
- Receipts for the desk, as in the decisions above.

**Test first.** Red before green: the auth invite and member tests (refused role, missing field), the `access` tests
(no `Desk`, no `RoleReception`), and the route test (unclassified operations, then `GET /clinic/waitlist` and
`GET /document-templates` answering 200 to a receptionist) before the `appointmentCaller` change. The web tests were
written before the code but not run red first.

**Checks** (compose project `salus-cms-f1`): the fast layer (vet and tests of the 23 affected Go packages, oxlint,
vitest related, i18n-check) and `runner.sh full` (`make lint test archcheck web-lint web-typecheck web-test web-build`;
web 1203 tests) pass. The first full run failed on gocognit (32) in the route test, fixed by moving the per-class
check into `checkClass`. Not run: `make e2e`, `make fhir-validate` (no FHIR route or golden file changed), `make cover`.
No screen walk in a browser: the only web changes are an invite radio option and a hidden menu, covered by Vitest.

**Copy waiting for the owner:** "Reception" / "Recepcija" / "Рецепција" and "Reception · Expires {date}" in the
five staff languages.

## F2: patients registered by the clinic and bookings by staff

Status: done 2026-11-23 on `wt/cms/F2`, one commit `cms(F2)`, with the staff move and the SMS reminders deferred
(below). No migration (00067 unused): 3.0's columns, policies and guards were enough.

**Built.**

- **patient** (`service_clinic.go`, `handler_clinic.go`, `GET`/`POST /clinic/patients`). `SearchClinic`: name (trimmed,
  2 characters at least, `%`, `_`, `\` literal) contained in "given family" or "family given", optional birth date, 50
  rows, by family and given name; the clinic's registered patients and those with its active care consent; the admin
  and receptionist see all of them, a practitioner only those with an appointment with their role; every patient
  returned is audited (search-type), an empty search once. `Register`: admin or receptionist only; the profile with
  `registered_by_organization_id` and the in-person care consent (`consent.RecordInPerson`) in one transaction.
  `Patient.RegisteredBy` is read on every query; `AccountID` is empty for a registered profile.
- **consent** (`service_inperson.go`). `RecordInPerson`: an active care consent `given_in_person` with the recording
  account, in the clinic's scope; `ErrActive` for one active already; `ErrNotFound` when the database's guard refuses
  (not a patient the clinic registered). `Consent.GivenInPerson` and `RecordedBy` are set by this write only.
- **appointment** (`service_staff.go`, `handler_staff.go`). `BookForPatient` (`POST /clinic/appointments`): a free slot of
  the caller's clinic as the directory shows it, the service checked as the patient's booking, the patient read and
  their active care consent checked in the clinic's scope, the slot held, `booked_by_account_id` stored; the desk books
  for every role, a practitioner for their own. For a patient without an account only the statistics are enqueued (no
  confirmation, questionnaire request or reminders). `CancelForPatient` (`/cancel`, `If-Match` required): the existing
  cancellation with the desk acting with the admin's access, rolled back on a stale version (`version_conflict`).
  `Arrive` (`/arrive`): desk only, an in-person booked visit, `arrived_at`, `not_in_person` for video. `DeskDay`
  (`GET /clinic/desk/today?date=`): the clinic's day in its zone, cancelled visits left out, by role and start, each with
  its patient (each patient read audited once, the search once). `Appointment.BookedBy`, `ArrivedAt` read on every
  query; `arrivedAt` in the JSON.
- **notification**: an appointment e-mail whose notice has no account sends nothing.
- **Callers**: `patient.Caller` and `appointment.Caller` gain `Reception`; "the desk" is admin or receptionist.

**Readings and decisions** (agent's, for the owner to review):

| Topic | Decision |
| --- | --- |
| The receptionist flag | `Reception` on the callers is never set yet: `auth.Member` has no reception field before F1. Until the integration sets it, only the admin acts at the desk |
| Who registers | Admin and receptionist; a practitioner searches (their own patients) and books (their own slots) but does not register |
| A staff booking's slot | Resolved as the public directory shows it (listed clinic, active role), as the patient's booking: an unlisted clinic books once listed (O1) |
| `consent_required` on a staff booking | 403, as `api/openapi.yaml` says (the patient's booking answers 409) |
| A receptionist's cancel | The admin's access to the appointment (W2), audited as the receptionist's account |
| Contract changes | `CancelForPatient` takes the `If-Match` version; `DeskDay` returns `[]DeskVisit` (appointment and patient) rather than `[]Appointment`, since the screen needs the patient; `serviceName` of `DeskVisit` is not filled (the screen has the price list) |
| The desk's "today" | Without `date`, the handler's UTC date: in the first hour or two after midnight in Sarajevo it is still the day before. Pass `date` from the screen |

**Deferred.**

- The staff move (`MoveForPatient`, `/clinic/appointments/{id}/reschedule`): not served and still refusing. The
  patient's `Reschedule` counts the move against the patient's own limit and confirms the visit in the app; a staff move
  needs neither, so it needs its own query (no reschedule count, history `via staff`).
- SMS reminders for a patient without an account: deliveries are per account (`notification_delivery.account_id`); an
  SMS to the profile's phone under the clinic's SMS option needs a delivery without an account (a migration). Until then
  such a patient gets no reminder at all, which the plan allows ("by SMS or not at all").
- Revoking an in-person consent from the clinic (an `UPDATE` policy) and correcting a registered profile.
- The concurrency test against Postgres of two staff bookings of one slot: the fake shows the second is refused, and
  the existing `TestCreate_TwoConcurrentOverlappingBookingsOfOneRoleGiveOne` covers the constraint both bookings share.

**Test first.** The service and handler tests were written before their code in each domain, but run together with it
rather than red first; the repository tests after the queries.

**Checks** (compose project `salus-cms-f2`): `make lint` (gofmt, vet, golangci-lint, i18n-check, migrations), `make
test`, `make archcheck`: pass (the third run; the first two failed on fieldalignment of new test structs, fixed). Not
run: `make cover` (the time budget), `make gen-clients` (no OpenAPI change), `make fhir-validate` (no golden file
changed), `make e2e` (not in F2's check set).

**INTEGRATION NOTES.** After F1: set `Reception` from F1's member in `patientReadCaller` and `appointmentCaller`
(`internal/api/app.go`), and check a receptionist reaches `/clinic/patients`, `/clinic/appointments*` and
`/clinic/desk/today` and nothing clinical. F3's screen should pass `date`.

## F3: the front desk screen

Status: done, 2026-11-23, on `wt/cms/F3`, one commit `cms(F3)`. Web only, built against 3.0's generated client with a
fake API in the tests; no Go, no migration (`00066`-`00071` untouched). The integration wires it to F1 and F2.

**Built.**

- `/desk` (`Desk.tsx`) for the clinic's admin and its receptionists (`App.tsx`: the route and a "Front desk" link first
  in the menu for those two roles; a practitioner gets neither). Today's visits from `GET /clinic/desk/today` (no
  `date`: the server's today in the clinic's zone), grouped by practitioner in the server's order, each with its time,
  the patient's name, birth date and phone (W2), the visit type, the booked service's name (`serviceName`) and the
  status; a role the clinic no longer has shows as "Former practitioner".
- `DeskCheckIn.tsx`: "Arrived" on a booked in-person visit only, `POST /clinic/appointments/{id}/arrive`; the row then
  shows "Arrived HH:MM" from the answer's `arrivedAt`, without reloading the day. `not_in_person`,
  `appointment_closed` and anything else each have their message.
- "Issue receipt" on every booked, arrived or fulfilled row opens package C's `ReceiptDialog` unchanged, with the
  visit's `appointmentId` and the booked service (package E's `appointment.serviceId`) as the first line.
- `DeskWalkIn.tsx`: a dialog that picks or registers the patient (`PatientPicker.tsx`), then a practitioner, one of
  today's free in-person times (FHIR `Schedule?actor=PractitionerRole/…`, then
  `Slot?schedule=…&status=free&start=ge<now>&start=lt<end of the clinic's day>`) and a service (active, offered at the
  clinic, or none), and books with `POST /clinic/appointments`. `slot_unavailable` reads the times again;
  `consent_required`, `invalid_service` and anything else each have their message.
- `PatientPicker.tsx`: search by name (from two letters) and an optional birth date (`GET /clinic/patients`); register
  a patient without an account (`POST /clinic/patients`: given and family name, birth date, optional phone, and the
  checkbox that the patient gave their care consent in person, which the button needs; `consentGivenInPerson: true`).
- The `desk` catalogs in the five staff languages (the consent line tagged `legal`); a `DeskIcon`; the desk's own
  block at the end of `app.css`.

**Decisions** (agent's, for the owner to review):

| Topic | Decision |
| --- | --- |
| A walk-in's times | Today's free in-person slots only, from now: a walk-in is at the desk. Booking a later visit for a patient is the calendar's or F2's API, not this dialog |
| Which day | Today only, as the plan's route says; the API's `date` is not used |
| The check-in button | Labelled "Arrived" as the plan says; the status tag reads "Arrived HH:MM". In bs, hr and sr "Prijavi dolazak" / "Dolazak u HH:MM" (gender-neutral) |
| Receipt on which rows | Booked, arrived and fulfilled; not on a cancelled visit or a no-show (as `AppointmentDetail`) |
| Phone on the row | Shown (W2 lists it for the desk), plain text |
| Free slots | Read through the FHIR directory search, as the mobile app does; there is no staff slot route in the contract |

**Test first.** `Desk.test.tsx`, `DeskWalkIn.test.tsx`, `PatientPicker.test.tsx` and the App route test were written
and run red (no such modules) before the code. `catalog.test.ts`'s list of areas gained `desk` on purpose.

**Walk.** No browser tool in this session and no F2 backend (its routes refuse), so the screen was walked with
Playwright (the `playwright` compose service, `--no-deps`) against `make web-build`'s output with the API faked by
route interception, at 1280 px and 390 px: the day, a check-in, the walk-in's picker with the registration form, the
booking step. No horizontal overflow at either width. The run fixed the row layout (the tag after the name, the actions
below on a phone) and the walk-in dialog's spacing; the date and phone inputs of the picker are styled in the desk's
block (the global input rule has no `date` or `tel`, so `ReceiptPeriod`'s date inputs are unstyled too: left alone).
The throwaway script and screenshots are not committed.

**Checks** (compose project `salus-cms-f3`): `make lint`, `make test`, `make archcheck`, `make web-lint` (1 warning, in
`analytics/ClinicView.tsx`, not this package's), `make web-typecheck`, `make web-test` (78 files, 1231 tests),
`make web-build`: pass. Not run: `make e2e` (the desk's routes are F2's and refuse until it lands; the e2e journey is
the integration's, below).

**For the integration session** (F1 and F2 are not wired here; everything below is what the fakes assume):

- `GET /clinic/desk/today` answers `DeskDay` with `visits` ordered by practitioner, then start; each `DeskVisit` carries
  the full `Appointment` (with `visitMode`, `status`, `arrivedAt`, `serviceId`) and `patient` (`ClinicPatient`), and
  `serviceName` when the booking named a service. 200 for admin and reception, 403 `forbidden` for a practitioner.
- `POST /clinic/appointments/{id}/arrive` answers 200 with the `Appointment`, `status: arrived` and `arrivedAt` set;
  409 `not_in_person` for video, 409 `appointment_closed` when not booked; no `If-Match` is sent (none is in the
  contract).
- `GET /clinic/patients?name=…[&birthDate=…]` answers a `ClinicPatient[]` (200, `[]` for none), for reception too.
  `POST /clinic/patients` answers 201 `ClinicPatient`; the body omits `phone` when empty; refusals `invalid_name`,
  `invalid_birth_date`, `invalid_phone` (400) are shown by code.
- `POST /clinic/appointments` with `{patientId, slotId, serviceId?}` answers 201 `Appointment`; refusals
  `slot_unavailable` (409), `consent_required` (403), `invalid_service` (400) are shown by code. A patient the desk just
  registered must be bookable at once (the in-person consent counts as the clinic's active care consent).
- Free times: the walk-in reads FHIR `Schedule?actor=PractitionerRole/<id>` and `Slot?schedule=…&status=free&start=ge…&
  start=lt…`, which the CapabilityStatement documents as the public directory's: **an unlisted clinic (or one O1 has
  not approved yet) has no schedules there**, so its desk would show no free time. Check it, and if so, F2 or the
  integration opens these two searches to the clinic's own staff (or adds a staff slot read) and the CapabilityStatement
  says so. The slot's visit mode is read from the `visit-mode` extension.
- Receipts by a receptionist: `POST /clinic/receipts` says "by the clinic's admin or the practitioner of the visit (the
  reception role waits for Q1)", and `GET /clinic/services` (the dialog's price list) may be admin- and
  practitioner-only too. F1 must let the reception role issue a visit's receipt and read the price list, or the desk's
  "Issue receipt" is refused for a receptionist.
- The desk also reads `GET /fhir/R4/Organization/<id>` (time zone) and `PractitionerRole?organization=…&_include=…`
  (names): F1 must let a receptionist read both.
- The menu: F3 adds only the "Front desk" link (admin and reception); F1 decides the rest of a receptionist's menu in
  `App.tsx` (today a receptionist sees every staff link). Merge the two `Navigation` changes by hand.
- An e2e journey (`test/e2e`): admin invites a receptionist (F1); the receptionist registers a walk-in, books today's
  in-person slot, checks the patient in, and the admin (or the receptionist, after F1) issues the receipt from the
  desk. Not written here: it needs F1 and F2 to pass.

**Copy waiting for the owner:** every string in `web/src/i18n/desk/en.json` (55 keys), in particular the consent line
"The patient gave the clinic their consent to care, in person." (`legal`) and its translations.

## O1: clinic signup and setup wizard

Status: done, 2026-11-23, on `wt/cms/O1`, one commit `cms(O1)`. Migration `00066_clinic_signup.sql`. 3.0 was taken as
done on the strength of its commit `2ea853c` (merged into this branch by `80e7769`).

**Built.**

- **One path for a clinic.** `signup.Service.CreateClinic` creates the clinic (unlisted), its admin's invite and its
  30-day trial in one transaction; `salusctl create-clinic` now calls it, and a verified signup calls it from the API.
  `organization`'s repository `Create` sets the new clinic's scope first (an id from `gen_random_uuid()`), and
  migration 00066 grants `salus_app` `INSERT (id, name)` on `organization`: `organization_clinic`'s check lets the app
  insert only the clinic of its scope, so the invite and the trial that follow write into that clinic alone.
- **Signup.** `POST /signup` (202 for every valid request, so nothing is enumerated: each is a new signup), rate limited
  per client address (5 an hour) and per e-mail (3 a day), counters named by an HMAC of `RATE_LIMIT_KEY`; the
  `signup_email` job issues a new token (only the newest link works) and the notifier sends the link
  `<WEB_BASE_URL>/signup/verify#<token>` in the signup's language (ten templates, golden files).
  `POST /signup/verify` (20 an hour per address): the signup row is locked, a used, replaced or 72-hour-old token is
  `invalid_verification`, and a concurrent second verification rolls its clinic back.
- **Operator.** `salusctl signup list` (opaque ids and dates only) and `salusctl signup approve <organization-id>`
  (sets `approved_at` and lists the clinic: `organization.Service.List`, audited as salusctl).
- **Wizard.** `clinic_setup_step` (00066, owned by `signup`, not personal data, clinic-scoped): `GET /clinic/setup` for
  the clinic's staff, `PUT /clinic/setup` for its admin; a step without a row is open. Web: `SetupWizard.tsx` (`/setup`,
  each step links to its own screen: clinic settings, membership for the billing details, working hours,
  practitioners, price list; done, skip or reopen) and `SetupBanner` above an admin's screens while a step is open.
- **Web signup.** `Signup.tsx`: `/signup` (clinic name, e-mail, the UI language) and `/signup/verify` (the token from
  the fragment, removed from the address bar). New catalog area `signup` in the five staff languages.
- **Retention.** The `clinic_signup` rule now deletes unverified signups 30 days after they were made
  (`signup.Purger`, as `salus_retention`, 00066's grant); a verified one stays with its clinic.
- **Landing.** The header's button is "Sign up" to `/signup` (only that link changed).
- **DPIA**: "Package O1" block.

**Contract changes.** `Start` and `Verify` take the client address (for the rate limits); `NewService` takes `Ports`
(the skeleton took the repository only); `IssueVerification`, `CreateClinic`, `Signups` and `Approve` are new.
`signup` imports `organization`, `auth` and `subscription` (service to service). The notifier's
`SendSignupVerification` takes the address, clinic, language and token as values: a `Signups` port in
`notification` would import `signup`, which imports `subscription`, which imports `notification` (a cycle); the
worker calls `signup.Service.IssueVerification` and then the notifier.

**Decisions** (agent's, for the owner to review):

| Topic | Decision |
| --- | --- |
| Link lifetime | 72 hours from the signup, as an invite; each e-mail replaces the token |
| Unverified signups | Deleted 30 days after they were made |
| The clinic's audit on creation from a signup | `organization.Create` records it as the operator command (`salusctl`), as before: one path |
| Language of the admin's invite | The clinic's default language (`bs`), as for `create-clinic`; the signup's language is used for the verification e-mail only |
| Wizard at first sign-in | A banner above every admin screen while a step is open, linking to `/setup`; no forced redirect |
| Landing link | Relative `/signup`: assumes the web app is served on the landing page's origin (see integration notes) |

**Copy waiting for the owner:** the `signup` web catalog (31 strings, five languages), the verification e-mail
(`internal/notification/templates/<language>/signup.txt`, ten languages), and the landing button "Sign up".

**Review** (one round, a fresh subagent over the diff; Copilot is out of quota). Fixed, each with its failing test
first: a signed-up clinic's admin could list it through the clinic settings before approval (00066's trigger
`organization_guard_signup_listed`: a clinic with a waiting signup is listed only by the operator; a clinic from
`create-clinic` is unaffected); a clinic name with a line break or a link went into the e-mail (refused,
`invalid_signup`). Checked and not reproduced: a double verification under React's StrictMode (a test pins one
request). Left, below under "Deferred".

**Deferred.**

- The audit events of a clinic created from a signup name the operator (`salusctl`) as agent, since `organization`,
  `auth` and `subscription` hard-code it; an agent for a self-signup needs a parameter through the three services.
- A verified `clinic_signup` row (the admin's e-mail) is kept while the clinic exists; when a clinic is closed is a
  later feature, and its deletion goes with it (owner to confirm).
- The 72 hours count from the signup, not from the latest e-mail: a much-delayed job sends a link with less time.
- Errors from the services `CreateClinic` calls that `validRequest` does not already exclude would be a 500 (none
  known: the e-mail and name checks match).
- The browser walk at desktop width and 390 px, and `make e2e` (`test/e2e/signup_test.go` is written, not run): the
  time budget went to the Go checks.

**Test first.** Red before green for the organization `Create` as `salus_app`, the signup service (compile-red), the
listing guard, the name check and the landing link; the Postgres repository, the handler, the worker and the web
screens' tests were written before their code but first run with it.

**Checks.** `make lint`, `make archcheck`, `make arch-gate`, `make test` (the full run, then the fast layer after the fixes),
`web-lint`, `web-typecheck`, the web tests of the touched screens, `make landing-test`, `check-caddy-routes`.

**Integration notes.** `/signup` is a screen of the web app and its writes go to the API: `web/vite.config.ts` and the
production Caddyfile gain the route (GETs to the app). `make gen-clients` gave no diff (the routes were in 3.0's contract). The landing's "Sign up"
points at `/signup` on its own origin: when the landing and the web app are served on different hosts, the link needs
the web app's address. `internal/api/router` needs nothing: `/signup` and `/signup/verify` take the CSRF header like
every write.

## O2: price list import and the clinic data export

Status: the import is done, 2026-11-23, on `wt/cms/O2`, one commit `cms(O2)`. **The clinic data export is not built**
(see "Deferred"). No migration: 00069 stays unused by the import; the export needs it.

**Built.**

- **`pkg/xlsx`** (new, standard library only: `archive/zip` and `encoding/xml`, about 200 lines with its limits). Reads
  the first sheet of a workbook (through `xl/workbook.xml` and its relationships, not by assuming `sheet1.xml`) as text:
  shared strings, rich text runs, inline strings, formula results, numbers as written. Each part is read up to 16 MiB
  inflated (a zip bomb stops there), rows up to 10000. No dependency added: excelize would bring a large module tree for
  what the import needs, and the reader is small and tested.
- **`chargeitemdefinition.Import`** (`service_import.go`). A CSV (UTF-8, a BOM allowed; the delimiter the header line uses
  most of `;`, `,` and tab) or an XLSX file (told by its zip signature). A header row names the columns, in English or the
  staff languages (`name`/`naziv`/`назив`, `code`/`šifra`, `price`/`cijena`/`cena`, `duration`/`trajanje`, `vat`/`pdv`,
  `modes`/`način`); name and price are required, others optional, unknown columns ignored, a column named twice refused.
  Per row: the codes `name_missing`, `name_invalid`, `code_invalid`, `price_invalid` (a decimal with a point or a comma,
  at most two decimals, a trailing `KM`/`BAM` allowed), `duration_invalid` (5 to 480), `vat_invalid` (0 to 100, `%`
  allowed; empty is exempt), `mode_invalid` (`in-person`/`uživo`/`video`/`online`..., separated by spaces, commas,
  slashes or bars; empty is in-person), `duplicate` (a name or code, case ignored, already on the clinic's list or on an
  earlier row). Blank rows are skipped; the line is the file's. With `save`, all rows are created in one transaction,
  each audited as a create, or none when any row has an error. Admin only (others refused and audited as the other
  writes are). `ErrInvalidImport` for a file it cannot read, no header with name and price, no rows, or more than 1000.
- **`POST /clinic/services/import`** (`handler_import.go`): the multipart part `file`, at most 2 MiB (`413 too_large`),
  `save` a boolean; `400 invalid_import`; a row with errors comes back without its `service`, so every `service` sent is a
  valid `SaveServiceRequest`. OpenAPI: the row error enum gains `name_invalid` and `code_invalid`; `make gen-clients`.
- **Web.** `PriceListImport.tsx`: choose a file, the preview (every service with price, duration, VAT and modes, or the
  line with what is wrong), "Import N services" only when no row has an error, then the list reloads with "N services
  imported." `PriceList.tsx`: the "Import from a file" button; its line of details moved to `serviceDetails.ts` to be
  shared. Strings in `prices` for the five staff languages.

**Decisions** (agent's, for the owner to review):

| Topic | Decision |
| --- | --- |
| XLSX | Read with the standard library (`pkg/xlsx`), no excelize |
| An import adds | New services only, active; a name or code already on the list is a `duplicate`, never an update |
| Empty modes cell | In-person (a clinic's existing list rarely has the column) |
| CSV encoding | UTF-8 only; a Windows-1250 export from Excel is refused as unreadable (`invalid_import`); the XLSX path has no such problem |
| Over-long name or code | Two codes added to the contract's enum (`name_invalid`, `code_invalid`) instead of calling them `name_missing` |
| Translations | Not imported (no column); added per service afterwards |

**Test first.** `pkg/xlsx`, the service and the handler tests were written and run red (undefined) before the code; the
web tests of `PriceListImport` were written before the component, the `PriceList` import test with its wiring.

**Checks** (compose project `salus-cms-o2`): the full set (`make gen-clients`, `lint`, `test`, `archcheck`,
`web-lint`, `web-typecheck`, `web-test` 1206, `web-build`, `mobile-analyze`, `mobile-test` 671) passes; `make cover`
90.9% total (`chargeitemdefinition` 95%, `pkg/xlsx` 84.7%); `make i18n-check` inside lint. Not run: `make e2e` (not in
O2's sets), `make fhir-validate` (no golden file changed). Not walked in a browser: the session had no browser tool;
the integration session walks the import at desktop width and 390 px.

**Deferred.** The clinic data export (`POST /clinic/export`, `GET /clinic/export/{id}`, `/archive`; web
`ClinicExport.tsx` and its link in `ClinicSettings.tsx`): its table (status, object key, expiry; migration 00069,
ownership, retention), the River job that builds the archive (price list, staff, appointments and receipts as CSV, each
patient's record as a FHIR Bundle, each read audited), the encrypted archive in the object store, the audited download and
its expiry, and the DPIA block. It did not fit the session's budget next to the import, and it is a package of its own
size; the routes stay as 3.0 declared them (not served).

**Copy waiting for the owner:** the new `prices.import*` and `prices.imported` strings in `web/src/i18n/prices/`.

## X1: treatment quotes

Status: done, 2026-11-23, on `wt/cms/X1`, one commit `cms(X1)`. No migration (3.0's `00065` holds `quote` and
`quote_line`); `00070` stays unused.

**Built.**

- **Go (`invoice`).** `service_quote.go`: `SendQuote` (at a visit, by the clinic's admin or the visit's practitioner,
  for the visit's patient while the clinic has their active care consent; lines priced from the price list as a
  receipt's, a note of at most 2000 characters, valid 30 days), `Quotes` (the admin every quote, a practitioner
  theirs), `ReadQuote`, `MyQuotes`, `QuotePDF`; the state (`sent`, `expired`) is read from `valid_until` and the clock,
  never stored. Every send, list, read and PDF is audited on `Quote/<id>`, also when refused. The repository's
  `InsertQuote`, `QuoteByID`, `QuotesOf`, `QuotesOfPatient` (sqlc); the `quote_email` job (`QuoteEmailArgs`, the
  visit's and the clinic's ids only) enqueued in the quote's transaction on the email queue. Handlers for
  `/clinic/quotes`, `/clinic/quotes/{id}`, `/me/quotes`, `/me/quotes/{id}/pdf` (`invalid_quote` for a draft that is
  no quote). The PDF uses package H's catalogs (`templates/pdf/<language>.json` gained `quote`, `quote_valid_until`,
  `quote_note`, `quote_not_receipt`): no second set of labels.
- **E-mail (`notification`).** `SendQuote` through the appointment e-mails (the clinic, the visit's local time, the link
  `PATIENT_APP_URL/quotes`, that nothing was charged; no service, amount or practitioner), `templates/<language>/quote.txt`
  in all ten languages with their golden files; the notifier's worker for `quote_email`.
- **Web.** `QuoteDialog.tsx` (the receipt dialog's line fields and words, a note, the effect before the button) and
  `VisitQuotes` (the visit's quotes with their state and validity, on request); `AppointmentDetail.tsx` gains "Write a
  quote" and "Show quotes" for the admin and the visit's practitioner; the `quote.*` keys of the `receipts` catalog in the five staff languages (no new area).
- **Mobile.** `quotes_screen.dart` (date, validity or "Expired", lines, note, total, the PDF shared) behind "My quotes"
  on the home screen; `Booking.quotes` and `quotePdf`; six strings in all ten languages.
- **Docs.** The DPIA's "Package X1" block; the `quote` retention rule's basis states the open period.

**Not a receipt, tested both ways.** `TestSendQuote_TakesNoReceiptNumberAndEnqueuesNoFiscalReport` (service: three
quotes leave the fake sequence at 0 and enqueue no fiscal report; the next receipt is number 1) and
`TestRepository_AQuoteTakesNoReceiptNumberAndNoFiscalJob` (Postgres and River: after a quote's transaction
`invoice_sequence` and `invoice` are empty and the only job is `quote_email` with the two ids).

**Decisions** (agent's, for the owner to review):

| Topic | Decision |
| --- | --- |
| A quote without a visit | Refused (`invalid_quote`): the plan writes a quote "from an appointment", and the e-mail and the practitioner's right rest on the visit. `appointmentId` stays optional in the schema; its 400 says so |
| Who sends | The clinic's admin and the visit's practitioner, as a receipt for a visit |
| A practitioner's list | The quotes they sent (as receipts: the ones they issued) |
| No active care consent | `not_found`, audited: the patient would see the quote, and `quote_clinic` refuses the row anyway |
| The PDF's language | The clinic's default language, as the receipt (the patient's is the same document) |
| The clinic's list on the web | Per visit (`?appointmentId=`), behind "Show quotes" in the appointment; a clinic-wide page needs a route and a menu entry in `App.tsx`, which F1 and F3 change in this wave (left to the integration, below) |
| Retention | Kept: no period without a legal decision (waiting on a person); the removal rule, its applier and a `DELETE` grant come together in a later migration |

**Review** (one round, a fresh subagent over the diff; Copilot is out of quota). Fixed: the visit's list read every
quote of the clinic into the browser (now `GET /clinic/quotes?appointmentId=`, a query parameter added to the
contract, filtered in the service); `ReadQuote` returned the refused quote beside `ErrNotFound` (now nothing). Left:
the quote e-mail for a patient without an account (F2's profiles: `sendAppointment` asks the account's e-mail of an
empty id and the job retries; the same holds for every appointment e-mail, so F2 or the integration decides it once);
refusals and list audits carry no entity (as the receipts'), and `Quote/<id>` is no FHIR type (as `Invite/<id>`); a
practitioner not the visit's gets 403 where a wrong visit gets 404 (as receipts); the app's quote card does not name
its clinic (the `Quote` schema has only `organizationId`; a clinic name needs a field or a lookup); `appointmentId`
stays optional in `SendQuoteRequest` (3.0's contract), its 400 says it is needed.

**Test first.** The Go service, repository, handler, PDF and e-mail tests were written first and run red (missing
methods and `quoteDocument`), then the code. The web and mobile tests were written before their components but run
only once the components existed (red not observed).

**Checks** (this worktree's compose project, `salus-cms-x1`): `runner.sh full` on the final tree (`make gen-clients`,
`lint`, `test`, `archcheck`, `web-lint`, `web-typecheck`, `web-test` (1207), `web-build`, `mobile-analyze`,
`mobile-test` (675)): pass. `make cover`: pass (total 90.8%; `invoice` 87.2%, `notification` 84.9%), run before the
review's two fixes, which only add tested lines. Not run: `make e2e` (no e2e journey changed), `make fhir-validate` (no
golden FHIR file changed). Not done: the browser walk of the dialog at desktop width and 390 px (no browser in this
unattended session; the integration session walks it).

**Copy waiting for the owner:** the PDF labels (`Quote`, `Valid until`, `Note`, "This is a quote, not a receipt.
Nothing has been charged."), the e-mail `quote.txt` in ten languages, the web `receipts.quote.*` keys (the dialog's effect is
tagged legal), the mobile `myQuotes`, `quotesIntro` (legal), `noQuotes`, `quoteTitle`, `quoteValidUntil`,
`quoteExpired`. Bosnian, Croatian and Serbian use "ponuda"; German "Kostenvoranschlag", French "devis", Italian
"preventivo", Turkish "fiyat teklifi", Arabic "عرض سعر".

**For the integration:** a clinic-wide quotes page (`GET /clinic/quotes` is served) if the owner wants one beside
the receipts, once `App.tsx` settles; the e-mail's link `/quotes` assumes the patient app opens there (today it opens
the app); `make gen-clients` regenerates the clients (two response descriptions and the `appointmentId` query parameter of `GET /clinic/quotes`).

## X2: clinic profile for patients abroad

Done 2026-11-23, one commit `cms(X2)` on `wt/cms/X2`. No migration: 3.0's columns and `organization_profile_translation`
are enough, so `00071` stays unused.

**Built.**

- `organization`: `service_profile.go` (`Profile` for the public read, `ClinicProfile` for the clinic's staff,
  `SaveProfile` for its admin, `ListListedSpeaking` for the directory's language filter), the repository's
  `ProfileOf`, `PutProfile` and `ListedSpeaking` (in `repository_postgres.go`, which archcheck requires for the sqlc
  package; six queries), and
  `handler_profile.go` (`GET` and `PUT /clinic/profile`, `GET /organizations/{id}/profile`). `Profile` gains
  `VersionID` (the organization's version, for the ETag); `ErrInvalidProfile` (`invalid_profile`).
- FHIR: `Organization?listed=true&language=<tag>`; the CapabilityStatement's `language` search parameter and
  `api/fhir/SearchParameter-organization-language.json`.
- Web: `ClinicProfile.tsx` on the clinic settings page (`/clinic`, admin): the ten languages by their own names, the
  welcome, the description, and its translations behind a disclosure; the new catalog area `clinicProfile` in the five
  staff languages.
- Mobile: the clinic list's language filter (a drop-down, "Any language" first), and `clinic_profile.dart` above a
  clinic's practitioners: the description in the app's language where translated (else as written), the welcome, the
  languages spoken, and the services with their prices (in person and video, each once). Six strings in all ten
  languages (eleven ARB files with `sr`).

**Decisions** (agent's, for the owner to review):

| Topic | Decision |
| --- | --- |
| A suspended clinic (owner's note, package I) | The public profile reads in the directory scope, so 00064's `organization_directory` policy hides a suspended or cancelled clinic, and the service checks `listed` again; it answers `not_found`, as the directory. Proved by `TestProfile_TheDirectoryHidesAnUnlistedOrSuspendedClinic` against the database (also the language filter) |
| "In the patient's language when translated" | The public read returns the description and every translation; the app picks the one in its language. No `Accept-Language` handling on the server |
| The language search parameter | A token `language` with no FHIRPath expression: the languages are the profile's, not an element or extension of the Organization resource, and adding an extension would change every directory response |
| The FHIR search's refusal text | Now "Search with listed=true and at most one language." (`unsupported_search.json` updated on purpose) |
| Translations without a description | Refused (`invalid_profile`): a translation of nothing |
| Order | Languages and translations are stored and returned in the enum's order |
| `GET /clinic/profile` | Any staff of the clinic (the OpenAPI says "for the clinic's staff"); `PUT` the admin only. A patient gets `not_found` |
| The web screen's place | A section of `/clinic` after the waitlist settings; no new menu entry |

**Test first.** The service tests, the mobile widget test and the web test were written and run red before the code.
The repository and handler tests were written with their code, then run.

**Checks** (this worktree's compose project, `salus-cms-x2`): the fast layer after each change; then `make lint`,
`make test`, `make archcheck`, `make web-lint`, `web-typecheck`, `web-test` (1206), `web-build`, `make mobile-analyze`,
`mobile-test` (675): all pass. The first full runs stopped on findings, fixed before the next: govet fieldalignment
and staticcheck in the test fakes, gofmt, the router's `metadata.json` golden (the CapabilityStatement's new
parameter, updated on purpose with `-update`), archcheck (the sqlc package may be imported only by
`repository_postgres.go`, so the profile's repository code moved there), and the web catalog test's list of areas
(`clinicProfile` added). Not run: `make e2e`, `make fhir-validate` (the new golden files are a searchset and an
OperationOutcome of shapes already validated; the new SearchParameter has no expression, which the validator may
warn on), `make cover`.

**Not walked in a browser**: this session had no browser tool. The web section and the mobile screens are covered by
their component and widget tests only; the integration session should walk `/clinic` at desktop width and 390 px.

## Wave 3 integration

Done 2026-11-23 on `feat/cms`: `wt/cms/F1`, `F2`, `F3`, `O1`, `O2`, `X1`, `X2` merged with `--no-ff` in that order onto
3.0, then one commit `cms(wave 3): integrate F1, F2, F3, O1, O2, X1, X2`. The wave's migrations follow `00065` without
gaps: O1's `00068_clinic_signup.sql` is now `00066_clinic_signup.sql` (with every reference to it); no other package
needed one, so `00067` to `00071` are not used.

- **Merges.** Conflicts only in shared files: this file and the DPIA (every package's block kept), the organization
  queries and port (X2's profile and O1's create and list, both kept), the web catalog area list, and the mobile ARB
  catalogs (merged as JSON: every package's keys). Generated code was not hand-merged: `make sqlc`, `make gen-clients`
  and `flutter gen-l10n` ran after the merges. `web/src/App.tsx` was merged by hand: a receptionist's menu holds only
  "Front desk", `/` sends them to `/desk`, and every other path is not found (F1 had them land on "not found"
  everywhere, with F3's `/desk` route behind that check).
- **The receptionist's clinic.** `internal/api/app.go` gives the care record's domains `appointmentCaller`, in which a
  receptionist is still no one of the clinic (refused as a stranger), and the desk's own routes and the receipts
  `deskCaller`, which carries the clinic and `Reception`. `patient.Caller` takes `Reception` from F1's member, and a
  receptionist's FHIR Patient read is decided by `access` as `RoleReception` (refused). The consent routes treat a
  receptionist as a stranger, so FHIR Consent is refused too.
- **Receipts at the desk.** `invoice.Caller.Reception`: a receptionist issues a receipt for any visit of the clinic and
  reads and prints the receipts they issued; the list of all receipts, a reversal, a retried report, the reports and
  quotes stay the admin's (and the practitioner's where they were). `GET /clinic/services` was already the desk's.
- **The desk's FHIR reads.** Organization/<own clinic> (its time zone), PractitionerRole?organization=<own clinic> with
  its practitioners, Schedule and Slot. A receptionist's read of their clinic is now audited (`organization.Read`
  records it when the caller is the receptionist; other staff reads of a clinic stay unaudited, nothing in a clinic is
  personal data); the role search and the practitioner reads were audited already.
- **A clinic that is not listed or not approved.** The desk works through the clinic's own scope: the FHIR Schedule
  and Slot searches give a clinic's own staff its schedules and free slots in its scope, listed or not
  (`schedule.ReadFor`/`ByActorFor`, `slot.SearchFor`; everyone else still gets the public directory), and a staff
  booking resolves its slot with `slot.BookableInClinic`, checks the role is active, and checks its service against the
  clinic's own price list (`chargeitemdefinition.OfClinic`). The suspended clinic's booking stop holds: a staff booking
  asks `subscription.BookingsStopped` (the clinic's scope does not hide such a clinic as the directory does) and is
  refused with `slot_unavailable`. The CapabilityStatement's Schedule, Slot and Organization texts say so.
- **The desk's day.** Without a date it is today in the clinic's zone (the service decides; the handler no longer
  takes a clock); each visit carries `serviceName` from the clinic's own price list; visits are ordered by practitioner
  role, then start, as F2 built. A patient registered at the desk is bookable at once: the in-person consent is the
  clinic's active care consent (`TestReceptionDesk_…` books them in the same minute).
- **F1's route test, every route.** `notServedYet` is now the staff move and the clinic export only; the receipts are
  reclassified (issue, read, PDF: desk; the rest refused); `GET /clinic/profile` is desk (the clinic's public profile,
  open to any staff since X2); `GET`/`PUT /clinic/setup` are refused and now are (admin only). PUT routes get an
  If-Match so they reach the access check. New `reception_fhir_test.go` holds every FHIR interaction of the
  CapabilityStatement to a class the same way: the desk's reads answer, every resource with a patient's data refuses
  (403, 404 or an empty search, the same for the clinic's own patient as for an unknown one, with none of their data).
  New `reception_desk_test.go` runs the desk at an unlisted clinic against Postgres.
- **Patients without an account.** No e-mail or push job is enqueued for them anywhere, and a job already queued
  completes without sending or retrying: booking (F2), cancellation by staff, the clinic or a revocation (new), SAL-33
  reminders (no delivery is created; the reminder is marked dispatched), the waitlist (such a patient is not
  reachable, so passed over), quotes (`quote_email` is not enqueued; `appointment.PatientHasAccount`), a signed
  document's e-mail (`composition` asks the same port), and chat (no unread e-mail and no push). A test per job kind.
  **Their reminders by SMS are a later package** (wave 4): until then they get no reminder at all.
- **Signup (O1).** The audit events of a clinic created by signup name the signup (`auditevent.AgentSignup`), those of
  `salusctl create-clinic` salusctl; `organization.Create`, `auth.InviteAdmin` and `subscription.StartTrial` take the
  agent. Every signup service error maps to a client code the contract documents for its route
  (`handler_http_errors_test.go` checks it against `api/openapi.yaml`); `PUT /clinic/setup` gained its documented 404.
  The wizard is the admin's only (403 to other staff). The landing's sign-up link is `VITE_APP_URL` + `/signup`, set at
  build time, and `[APP URL]/signup` while unset (CLAUDE.md's landing row lists the placeholder). The e2e journey
  `TestClinic_SignsItselfUpAndIsListedOnceApproved` found a real bug: `PUT /clinic/setup` wrote before it set the
  clinic's scope (a 500 on the first save); fixed with a test.
- **Price list import (O2).** A CSV that is not valid UTF-8 is decoded as Windows-1250 before it is refused (a
  128-entry table, no dependency); bytes Windows-1250 leaves undefined are still refused. Tested with š, đ, č, ć, ž and
  their capitals.
- **Quotes and profile (X1, X2).** `make fhir-validate` over X2's golden files and SearchParameter: 0 errors (only the
  general dom-6 narrative warning); the SearchParameter keeps no expression, which R4 allows, because the languages
  are profile data, not an Organization element. The mobile quote card names its clinic from the directory, or says
  it is no longer listed.
- **Web.** One global style for `date` and `tel` inputs; the desk's own copy of it is gone, so the desk and the
  receipts period match.
- **e2e.** New `TestDesk_AReceptionistRegistersAWalkInBooksChecksInAndIssuesTheReceipt` (the receptionist finds a free
  in-person time through the clinic's own searches, registers the walk-in with the in-person consent, books with a
  service, sees it on the desk's day with its name, marks the arrival, issues and prints the receipt, and is refused the
  patient's questionnaire answers, documents, history and chat) and O1's signup journey.

**Checks** (this integration's own compose project, `salus-cms-w3`; e2e on fresh volumes, twice: before and after the
review's fixes). On the final tree: `make lint` (gofmt, vet, golangci-lint, i18n-check, migrations, Caddy routes),
`make test`, `make archcheck`, `make cover` (90.6% total, every package at 80% or more), `make fhir-validate` (129
files, 0 errors), `make gen-clients` (no diff), `make e2e` (362 s); before the review's fixes, which touched no web,
mobile or landing file: `make web-lint` (the one warning in `analytics/ClinicView.tsx` that was there), `web-typecheck`,
`web-test` (1266), `web-build`, `make mobile-analyze`, `mobile-test` (680), `landing-lint`, `-typecheck`, `-test` (51),
`-build`: all pass. Tests changed on purpose: F1's App test (the receptionist's menu and landing), F1's route table
(receipts, `notServedYet`, the profile), the staff booking's "another clinic's slot" row (`slot_unavailable`), the
desk handler's day without a date, O2's undecodable-byte case (0x9a is "š" in Windows-1250), and the golden files of
the organization's invalid name and the router's CapabilityStatement.

**Review** (a fresh subagent over `origin/main` and the working tree; Copilot is out of quota). The one question —
can a receptionist, or one clinic, reach another's data or clinical data — found no path. Fixed:

- An XLSX "cell bomb" (cells referencing column XFD) could expand a few KB of XML into gigabytes in `pkg/xlsx`; a
  sheet may now expand to at most a million cells (`ErrTooLarge`, tested), and self-signup made the import public.
- An `arrived` visit could never be closed: its practitioner now marks it fulfilled or a no-show and staff cancel it
  (00065's transitions), and the arrival is written to the visit's change history.
- One patient the clinic's scope hides (a revocation before the job cancels the visit) gave the whole desk day a 500;
  their visit is now left off the day.
- A signed-up clinic's admin listing it before approval got a 500 from 00066's trigger: now 409 `business-rule`
  (`organization.ErrNotApproved`, audited as refused).
- A clinic could be renamed, after signup's check, to a text with a link or a line break that then went into every
  invite e-mail it sends: a clinic's name is one line without `://` on create and on rename too.
- FHIR `Appointment?status=arrived` was refused; it is a search value now (CapabilityStatement updated).
- `access.Desk` had no caller (the desk rests on its SQL and row-level security, and `access` refuses a receptionist
  every care-record rule): deleted with its test.

Declined, with reasons in the PR: a cap on invites per clinic (the name rule removes the attacker's text; a signed-up
clinic must invite its first practitioner in the wizard before approval, so gating invites on approval would break
the wizard; a per-clinic invite limit is a wave 4 item); the clinic revoking an in-person consent and the subject
rights of a clinic-registered profile (deferred to wave 4 above, needed before production); the import reading the
2 MiB upload before the admin check (bounded, refused and audited).

**Browser walk** (headless Chromium from the playwright service on this integration's own stack, a synthetic clinic,
1280 and 390 px; the temporary spec and its Go driver are not committed): as a receptionist, `/desk` with the only
menu entry "Front desk", "Arrived" and its time, the receipt dialog from the row, the walk-in dialog, and
`/practitioners` not found; as an admin, the home banner of the setup, the staff invite with Reception, the desk, the
price list import with a file that has errors (three lines named with their errors), the quote dialog, the clinic
settings with the profile for patients abroad, `/setup`; signed out, `/signup`. No horizontal scroll on any page at
either width. At 390 px the receptionist's run stopped at "Arrived" because the 1280 run had already marked the visit
arrived (the script's order, not the screen). Seen: at desktop width the walk-in dialog's fields take half its width.
The patient app's quote and clinic profile screens are covered by widget tests (`quotes_test.dart`, the clinic
profile's), not walked: there is no device run in this repository.

**Decisions taken here.**

| Topic | Decision | Why |
| --- | --- | --- |
| A receptionist in the care record's domains | No clinic at all (a stranger), as F1 built; the clinic only on the desk's routes and receipts | One place refuses by default; a new clinical route cannot open to the desk by accident |
| A receptionist's receipts | Issue for any visit; read and print the ones they issued | W2 gives the desk receipts; the list of all of them is the admin's |
| The desk's free times | The existing FHIR searches, opened to the clinic's own staff in its scope; no new route | The screen already reads them; nothing in a slot is personal data |
| Another clinic's slot in a staff booking | `slot_unavailable`, as an unknown one (was `not_found`) | Resolved in the clinic's own scope, it does not exist; one existing test row changed on purpose |
| A suspended clinic at the desk | Refused with `slot_unavailable` | The same answer the app gets for such a clinic |
| Auditing a clinic read | The receptionist's only | Asked for the desk; a clinic holds no personal data |
| `GET /clinic/profile` | Any staff of the clinic, as X2 built | The same data is public at `/organizations/{id}/profile` |

**Deferred to wave 4.**

- Staff moving a visit: `POST /clinic/appointments/{id}/reschedule` stays declared and unserved, `MoveForPatient`
  refuses, and the desk shows no move action.
- SMS reminders for patients without an account (deliveries are per account today: a migration).
- The clinic revoking an in-person consent, and correcting a profile it registered.
- The clinic data export (O2; its migration number is reserved, the three routes stay declared and unserved).
- A clinic-wide quotes page in web.
- Patient import.
- A limit on the invites a clinic sends (review).
- Also open: the hourly slot generation covers listed clinics only, so an unlisted clinic's free times run 28 days ahead
  of its last change of hours and are not rolled forward until it is listed. (Done in wave 4, R1.)

## R1: free times for every clinic

Done 2026-11-24, one commit `cms(R1)` on `wt/cms/R1` (wave 4, `docs/clinic-cms/WAVE4.md`). No migration.

**Built.**

- `schedule.Service.Generating` replaces `Listed`: every clinic's id from `organization.Stats.Locations` (new port
  `schedule.Clinics`), then each clinic in its own scope and transaction: its schedules (`OfClinic`) whose role is
  active (`Roles.Lookup`, added to the `schedule.Roles` port; `practitionerrole.Service` already has it). The directory
  scope is no longer used by the generation.
- `slot.Service.GenerateAll` asks `Generating`, so the hourly run covers unlisted and suspended clinics too. The
  directory's reads (`Search`, the FHIR Slot and Schedule searches) are unchanged: row-level security still shows only
  listed, not suspended clinics.
- `internal/worker`: `newSlots(pool, statsPool, clk)` builds the role lookup and the clinic list; `New` passes the stats
  pool (`newRevocations` passes none: it only holds and frees slots). `TestNew_HandsTheStatsPoolToTheAggregateBuildsOnly`
  now lists `newSlots` on purpose.
- Tests: `internal/worker/slots_unlisted_test.go` against the database with the worker's three logins and the service
  `newSlots` builds (an unlisted clinic's slots roll forward a week later, a suspended clinic's are generated, an
  inactive role's are not, the directory's `Search` gives only the listed clinic's); `TestGenerating_…` in `schedule`
  with fakes; the existing `GenerateAll` and worker tests renamed or kept. WAVE4 named
  `internal/slot/service_generate_unlisted_test.go`: a database test in a `service_*` file breaks depguard (no pgx in a
  service file) and needs the worker's composition, so it lives beside the worker's other slot tests.

**Decisions** (agent's, for the owner to review):

| Topic | Decision | Why |
| --- | --- | --- |
| Listing every clinic without a migration | `organization.Stats.Locations` on the `salus_stats` login (it reads only `organization.id` and `time_zone`, no personal data); everything else on `salus_app` in each clinic's scope | `salus_app` sees no clinic outside a scope, and a policy for a "generation" scope would need a migration number (WAVE4 gives R1 none; `00067` on are taken) |
| "Active clinic" | Every clinic, with the schedules of its active roles (as before for listed clinics); `organization.active` is not consulted | The old run did not check it either (the directory policy never did), and the app never sets it |
| A suspended clinic | Generated (V1); its bookings are stopped where they are made | WAVE4 R1 |
| A clinic whose listing fails mid-run | The run stops with the error; a schedule's own failure still does not stop the others | As before for the list; the next hourly run retries |

**Measured** (`BenchmarkGenerateAll1000Clinics` in `internal/worker`, `-benchtime 3x`, i7-14700HX, the compose
Postgres): 1 000 clinics, half unlisted, one schedule each (weekdays 08:00 to 16:00, hour slots, 159 slots a schedule).
First run (from no slots) 28.5 s a run; the hourly job's steady state (nothing changes) 20.8 s a run, about 21 ms a
schedule, against 17 ms a schedule for 50 schedules of one clinic in `docs/booking/PROGRESS.md` (package 5): the extra
is the per-clinic transaction and the role lookup. Well inside the hour; not optimised.

**Checks** (this worktree's compose project, `salus-cms-r1`): the fast layer after each change; the full Go set (`make lint`, `make test`, `make archcheck`) twice: the first stopped on lint (depguard and forbidigo in the database test, then in a `service_*` file, moved to `internal/worker`), the second passed. Not run: `make e2e`, `make cover`, `make fhir-validate` (no FHIR response changed).

## R2: rights of patients the clinic registered

Status: done 2026-11-24 on `wt/cms/R2`, one commit `cms(R2)`. Migration `00067_clinic_consent_withdraw.sql`. Built to
V2 of `WAVE4.md`, which **needs the lawyer's word before production** (the list is under "Waiting on a person").

**Built.**

- **patient** (`service_clinic_rights.go`, `handler_clinic_rights.go`). `Correct` (`PUT /clinic/patients/{id}`,
  `If-Match`): the clinic's admin or receptionist replaces the names, birth date and phone of a profile **their clinic
  registered**, at the version read; validated as a patient's own profile; audited as an update, refusals included.
  `WithdrawConsent` (`POST /clinic/patients/{id}/consent-withdrawal`, 204): reads the profile (audited) and has
  `consent.WithdrawInPerson` revoke the in-person consent, in one transaction. `ClinicRecord`
  (`GET /clinic/patients/{id}/record`): the admin only; a FHIR R4 Bundle of type `collection` (Patient and the clinic's
  Consents of the patient, `urn:uuid` full URLs, `Content-Disposition: attachment`); audited with `PATRQT` (the
  patient's request) as the purpose. Every route answers `not_found` for a profile the caller's clinic did not register:
  another clinic's, or a patient's own (checked in the service on `RegisteredBy`, not on row-level security alone,
  since the clinic's scope also shows patients with its consent).
- **consent** (`service_inperson_withdraw.go`). `WithdrawInPerson`: in the clinic's scope, revokes the patient's
  active care consent given in person to that clinic and then does exactly what `Revoke` does for a care consent
  (`revoked`: the transcription and history consents with the clinic, then `consent_revoked` enqueued in the same
  transaction), audited with the staff account; `ErrInactive` (409 `consent_inactive`) when there is none.
  `OfClinicPatient`: the clinic's consents of a patient, each read audited. `FHIR(c)` exports the Consent mapping for
  the record's Bundle.
- **Migration 00067.** `consent_clinic_withdraw_in_person`: an `UPDATE` policy for the clinic's scope, on its own
  `given_in_person` consents, from active only and to inactive only (00016's column grant already limits the columns).
  An app patient's consent stays theirs to revoke; a clinic cannot reinstate a withdrawn consent (tested against
  Postgres).
- **What the withdrawal sets off** is the existing `consent_revoked` job, unchanged: `CancelForRevokedConsent` (the
  visits ahead, with their notices; a patient without an account gets none, wave 3), `WithdrawForRevokedConsent` (the
  waitlist), `EndPatientAccess` (video). Access ends at the commit: `access.Profile` needs an active consent, so the
  clinic's practitioners no longer see the profile; the desk still finds the registered profile (the clinic holds it).
  The e2e journey checks the visit leaves the desk's day.
- **Web** (`DeskPatient.tsx`; `Desk.tsx` only renders it in place of the name, two lines). A registered patient's name
  on the desk row is a button that opens the patient's card: the profile form (Save, `If-Match`), "Record withdrawal of
  consent" with a confirmation that states its effects (`legal`), and, for the admin only, "Download the record" (a
  same-origin link with `download`). An app patient's name stays text. The card asks `GET /auth/me` for the role
  rather than threading the account through `Desk.tsx`'s rows (M1 edits them in parallel). The `desk` catalogs in the
  five staff languages (`desk.patient.*`, 19 keys; five tagged `legal`).
- **Erasure: not built, by the plan's rule.** There is no erasure path for app patients either: `patient`, `consent`
  and `account` are `keep` in `internal/retention/rules.go`, "erasure comes with the subject-rights feature". The
  registered profile already follows those rules (the `patient` rule names it). Nothing was invented; see "Waiting on
  a person".
- **Contract**: `api/openapi.yaml` gains the three routes and `CorrectPatientRequest`; clients regenerated.
  `internal/api/reception_routes_test.go` classifies them (correct and withdrawal: desk, record: refused) and maps
  `/clinic/patients/{id}` to a patient the receptionist registered. DPIA block "Package R2".

**Readings and decisions** (agent's, for the owner):

| Topic | Decision |
| --- | --- |
| The record's content | Patient and the clinic's Consents only: what the admin may read today. Appointments, encounters and documents are left out (the admin has no access rule to the clinical record; a full `$everything` is not in any SPEC). M2 builds "each patient's record as a FHIR Bundle" too: one builder for both is the integration's call |
| Who downloads | The admin only (as the plan says); the receptionist corrects and records the withdrawal |
| Withdrawal "in one transaction" | The revocation, its audit and the `consent_revoked` job are one transaction, as an app patient's; the cancellations run in the job, as they do for an app patient |
| A second consent after a withdrawal | Not built: the desk records an in-person consent only at registration (F2). Booking such a patient again is refused (`consent_required`) |
| Purpose of the record's audit | `PATRQT` (the patient's request); the other rights routes `TREAT`, as the rest of the desk |
| The withdrawal's route | Served by `patient` (the desk caller, the registered check) calling `consent`; the consent domain has the rule and the write |

**Test first.** The patient service and handler tests, the web test and the reception route classification were run
red first (no such methods, routes or module). The consent service test was written first but run only after its code;
the Postgres tests after their queries. `DeskPatient.test.tsx` was written before the component but run after it;
`Desk.test.tsx`'s "no receipt on a cancelled visit or a no-show" row changed on purpose (the registered patient's name
is now a button) and gained the card's opening.

**Checks** (compose project `salus-cms-r2`). `make gen-clients`, `make lint` (gofmt, vet, golangci-lint 0 issues,
i18n-check, migrations), `make test`, `make archcheck`, `make web-lint` (the one warning in `analytics/ClinicView.tsx`
that was there), `make web-typecheck`, `make web-test` (1281), `make web-build`, `make mobile-analyze`,
`make mobile-test` (680), `make fhir-validate` (the new `clinic_record.json` included, 0 errors), `make e2e` (356 s,
new `TestDesk_TheClinicActsOnTheRightsOfAPatientItRegistered`): pass on the final tree. Earlier full runs failed on
gofmt of the e2e file and on fieldalignment of four structs (fixed; the golden file regenerated for the reordered
entry). Not run: `make cover` (the time budget).

**Deferred.**

- The browser walk at 1280 and 390 px: not done in this session (time budget, and the stack was busy with `make e2e`).
  The card is a dialog with the walk-in dialog's classes; the integration's walk should open a registered patient's
  card at both widths.
- The record's visits and clinical documents, and one record builder shared with M2's export.
- Recording a new in-person consent for a patient who withdrew (today only at registration).
- Erasure for any patient (the subject-rights feature).

**INTEGRATION NOTES.** `Desk.tsx`: R2 replaces the name `<span>` in `Row` with `<DeskPatient api={api} patient={p} />`
and adds its import; M1's "Move" goes in the row's actions, so the two should merge without overlap. `Desk.test.tsx`:
one assertion changed and one test added at the end of the cancelled/no-show test. Regenerate the clients after the
merges (R2 adds three routes and `CorrectPatientRequest`).

## R3: SMS reminders for patients without an account (wave 4, 2026-11-24)

Built to `docs/clinic-cms/WAVE4.md` R3 and decision V3 (agent's default, owner to confirm; recorded here as the wave
asks): SMS only, to the profile's phone, while the clinic's SMS option is on; `pkg/sms` stays the fake.

- **Migration `00068_patient_sms_delivery.sql`.** `notification_delivery.account_id` nullable, `patient_id` (references
  `patient`, `ON DELETE CASCADE` as the account's), exactly one of the two (`notification_delivery_recipient`), and a
  patient's delivery is an SMS only (`notification_delivery_patient_sms`). No new table: ownership unchanged.
  `TestNotificationDelivery_ToAnAccountOrBySMSToAPatient`.
- **appointment** (`service_reminder_patient.go`). A desk booking for a patient without an account now schedules the
  visit's reminders (`bookedAtDesk`: the statistics and `scheduleReminders`; still no confirmation or questionnaire
  request); a cancellation cancels them as any other. `DeskVisit.Reminder` (`reminder` in the desk's JSON and the
  OpenAPI `DeskVisit`): true for a patient with an account, else only while the clinic offers SMS and the profile has
  a phone. The screen is not this package's.
- **notification** (`service_patient_sms.go`). `DispatchReminder` hands a notice without an account to
  `deliverPatientSMS`: nothing unless the clinic offers SMS, then one SMS delivery to the patient, held by the default
  quiet hours (22:00 to 08:00) in the clinic's zone or dropped when they end less than 30 minutes before the start.
  `SendDelivery` composes it with `composePatientSMS`: the reminder re-read (gone when cancelled or moved, or the
  profile has an account now), the clinic's option checked again, the phone read through `patient.NoticePhone`
  (audited, the send's job `reminder_sms` as agent), then sent once and counted in the clinic's `sms_usage` by the
  existing `finish`. The SMS text's data is shared with the app patient's (`reminderSMS`). Every `reminder_sms.txt`
  (ten languages) prints its link only when there is one; no wording changed.
- **Retention.** The `notification_delivery` rule's basis names the profile; period unchanged (90 days).
- **DPIA.** `docs/privacy/dpia.md` "Package R3".
- **Tests.** `TestReminder_APatientWithoutAnAccountGetsOneSMSAtTheRightInstant` (`internal/worker`, Postgres, River's
  inserts, fake SMS): `TestReminder_SentOnceAtTheRightInstant`'s cases for a profile at a Sarajevo clinic around the
  25 October change (the quiet hours, a dispatch rolled back once, every job run twice, one SMS counted once, a
  cancellation and a move between scheduling and dispatch), plus no option and no phone. Service tests in
  `service_patient_sms_test.go` and `service_reminder_patient_test.go`; `TestDeliveries_AnSMSToAPatientWithoutAnAccount`.
  `TestDispatchReminder_DeliversNothingToAPatientWithoutAnAccount` now runs at a clinic without the SMS option, and
  `TestBookForPatient_SendsAPatientWithoutAnAccountNothing` became `…NoEmail` (reminders are now scheduled): both
  changed on purpose, as R3 changes that behaviour.

**Readings and decisions** (agent's, for the owner to review):

| Topic | Decision | Why |
| --- | --- | --- |
| The one-time link | Not in a profile's SMS | A link's use acts as the patient's account (audit, the patient's scope), and a registered profile has none; the patient scope does not see a registered profile (`patient_registered_only_there`). Confirming or cancelling by SMS link for them is left open |
| Quiet hours | The default 22:00 to 08:00 in the clinic's zone | A profile has no settings; the same default an app patient gets, so the existing cases apply |
| Language | The clinic's default language (`organization.DefaultLanguage`, else Bosnian) | A profile stores no language |
| The phone at dispatch | Not read; checked at the send (no phone cancels the SMS) | WAVE4: read "only for the send" |
| The desk's field | `reminder` optional in the OpenAPI (always sent) | Required would make the generated web type require it in `web/src/test/desk.ts`; this package is Go only. The screen's package can make it required |
| The desk's field for an app patient | Always true | Their channels are their own choice; the desk cannot see them |

**Checks** (2026-11-24): `runner.sh fast` passed; `runner.sh full` passed (`make gen-clients lint test archcheck web-lint
web-typecheck web-test web-build mobile-analyze mobile-test`: lint 0 issues, archcheck clean, every Go, web and mobile
test green). The migration test's first run failed on a port clash with another worktree's stack (no test ran) and
passed on its rerun after this worktree moved to free ports.

**Deferred**: a one-time link for a profile's SMS (above); showing `reminder` on the desk's screen (a web package).

## M1: staff moving a visit

Status: done 2026-11-24 on `wt/cms/M1`, one commit `cms(M1)`. Migration `00069_staff_move.sql`.

**Built.**

- **appointment** (`service_staff_move.go`, `MoveForPatient`; served at `POST /clinic/appointments/{id}/reschedule`,
  `If-Match` required, body `{slotId}`): the clinic's admin and receptionist move a booked visit to any free slot of the
  clinic (resolved in the clinic's own scope, listed or not, of an active role) of the visit's type, another
  practitioner's too (V4). In one transaction in the clinic's scope: the appointment read and checked (booked, at the
  `If-Match` version, the same visit mode, a video visit only before its join window opens), the patient's active care
  consent checked, the slot held, the appointment moved by its own query (`MoveAppointmentByStaff`: period, slot and
  practitioner role, `version_id` in the `WHERE`; no reschedule count, no confirmation), the old slot freed and handed
  to the waitlist, the reminders cancelled and, for a patient with an account, scheduled again with the
  `appointment_rescheduled` e-mail, both days' statistics refreshed, the history `rescheduled` `via staff` with the
  account, and the move audited (each refusal too). Refusals: `forbidden` (a practitioner or a patient),
  `not_found`, `appointment_closed` (arrived, fulfilled, cancelled, no-show), `version_conflict`, `not_reschedulable`,
  `consent_required`, `slot_unavailable`. `Repository.MoveByStaff` is the new port.
- **Migration `00069`**: `GRANT UPDATE (practitioner_role_id)` to the app; `appointment_guard_change` lets the clinic's
  scope move a visit without counting it (the patient's own scope still counts each move exactly once); a new
  `appointment_guard_role` lets only the clinic's scope change the practitioner, with a move of a booked visit, to a
  role of its own clinic, and refuses any statement naming the column in another scope.
- **web**: `DeskMove.tsx`, a dialog from the desk row's new "Move" action (booked visits only; `Desk.tsx` gained only the
  action, its dialog and the notice): the practitioner (the visit's by default, any active one), today's and the next
  14 days' free times of the visit's type grouped by day, `If-Match` from the row's version; after a move the day is
  read again. The `desk` catalogs gained `move.*` (10 keys, five languages).
- **Contract**: the route's description and refusals in `api/openapi.yaml` now say what is built (V4: another
  practitioner allowed, `consent_required`, `not_reschedulable`); clients regenerated. `x-salus-package` stays
  `cms-f2` (the wave 3 contract test pins it).
- **e2e**: `test/e2e/desk_move_test.go`: a receptionist books a walk-in and moves the visit; the old time is free again,
  a move from the old version is `version_conflict`.

**Readings and decisions** (agent's, for the owner to review):

| Topic | Decision |
| --- | --- |
| V4 applied | As WAVE4 says (agent's default, owner to confirm). A practitioner cannot move a visit, their own either: V4 names admin and reception |
| When a visit stops moving | In person: while booked, also after its start (a late patient is moved to a later time). Video: until its join window opens (10 minutes before), so no consultation session exists yet and the window follows the new start |
| A clinic whose bookings are stopped | The desk still moves its existing visits (a move is not a new booking); only `BookForPatient` checks the stop |
| A patient without an account | Reminders cancelled and none scheduled, no e-mail, as `BookForPatient` does: R3 gives such a patient SMS reminders, and the integration should call R3's scheduling here too |
| The guard's count rule | The clinic's scope may also count a move (+1), as 00065 allowed, so the wave 3 contract tests stay as they were; the service never does. One test changed on purpose: `TestAppointment_OnlyThePatientMovesOrConfirmsTheirBookedAppointment`'s SECURITY DEFINER case now runs in another clinic's scope, since the appointment's own clinic may now move it without counting |
| The placeholder test | `TestMoveForPatient_RefusesUntilItIsBuilt` removed (it asserted the unbuilt state); `notServedYet` in `reception_routes_test.go` lost the route, which is now held to the desk class |
| Free times offered | Today and the 14 days after it, through the existing FHIR Schedule and Slot searches (no new route) |

**Test first.** The service, repository, migration and handler tests were written and run red first (the migration
test with the file moved aside). `DeskMove.test.tsx` was written before the component but not run red; the Desk row's
test was run red.

**Checks** (compose project `salus-cms-m1`): `make gen-clients`, `make lint` (gofmt, vet, golangci-lint,
i18n-check, migrations), `make test`, `make archcheck`, `make web-lint`: pass on the second full run (the first failed
on gocognit and fieldalignment in the new service test, fixed). `make web-typecheck` then failed on `not_reschedulable`
missing from the `Error` enum: added to `api/openapi.yaml` (the patient's reschedule answers it too), clients
regenerated; then `make web-typecheck`, `make web-test` (86 files, 1279 tests), `make web-build`, `make mobile-analyze`,
`make mobile-test` and `go test ./api/... ./internal/api/...`: pass. e2e: `make e2e-up`, then `go test -tags e2e
-run TestDesk_ ./test/e2e/...` (the new move journey and F3's desk journey with its browser run): pass; the whole
`make e2e` suite was not run (the time budget). The fast layer's Go step timed out at 10 minutes (every package runs
when `migrations/` changes); the full set covered it. Not run: `make cover` (the budget), `make fhir-validate` (no
golden file changed).

**Not walked in a browser by hand**: no browser tool in this session. The move dialog is covered by its component tests
and the Desk row test; F3's Playwright desk journey still passes with the new action on the row. The integration's walk
should open "Move" at desktop width and 390 px.

**Copy waiting for the owner:** `web/src/i18n/desk/en.json` `move.*`: "Move", "Move visit", "Moving", "New time", "This
practitioner has no free time of this visit type in the next two weeks.", "The visit was changed in the meantime.
Close this and reload the page.", "A video visit cannot be moved once its call can be joined.", "The visit could not be
moved. Try again.", "The visit was moved.", and their bs, hr, sr-Latn, sr-Cyrl translations.

**INTEGRATION NOTES.** `Desk.tsx`: M1 adds the row's "Move" button, the `DeskMove` dialog, the `move`/`onMoved` props
of `Row` and the notice; R2 adds the patient's card there too: merge by hand. The `desk` catalogs: M1's `move.*` block
is appended at the end. After R3: schedule a moved visit's SMS reminders for a patient without an account in
`movedByStaff` as R3 does in `bookFor`.

## M2: the clinic data export

Status: done 2026-11-24 on `wt/cms/M2`, one commit `cms(M2)`. Migration `00070_clinic_export.sql`. Specification:
`WAVE4.md` "M2" (what O2 left, its "Deferred" above).

**Built.**

- **Go, a new domain `internal/clinicexport`** (directory context, `archcheck/config.yaml`), owner of `clinic_export`
  (`ownership.yaml`, personal data). `Request` (the admin only, audited also when refused; one queued export per clinic,
  a second ask returns it; the build enqueued in the same transaction), `Get` (status; a ready export past its expiry
  reads `expired`), `Download` (ready and unexpired only, audited, decrypted as it streams), `Build` (the job: reads the
  price list, the staff with their names, the appointments with the patient's id and name, the receipts; writes
  `prices.csv`, `staff.csv`, `appointments.csv`, `receipts.csv` into one zip; seals it with `pkg/envelope` under a data key
  bound to the export's id; stores it at `clinicexport/<id>` outside any transaction; marks it ready), `Fail` (the job's
  last attempt), the `Purger` (the archive, then the row, 7 days after the request). CSV cells that a spreadsheet would
  read as a formula are prefixed with an apostrophe.
- **Routes** `POST /clinic/export`, `GET /clinic/export/{id}`, `GET /clinic/export/{id}/archive` are served
  (`internal/api/clinicexport.go`; off `notServedYet`; reception stays refused).
- **Worker**: `clinic_export_build` on the default queue (3 attempts, then failed), built in `internal/worker/clinicexport.go`
  on the default queue's appointment and receipts services; the `clinic_export` retention rule (`Keep` 7 days, delete) and
  its purger on `salus_retention` and the object store.
- **Web**: `ClinicExport.tsx`, a section of the clinic settings (ask, the status read every 5 s while it is built, the
  download link and its expiry, failure and expiry notices); its strings are `admin.export*` in the five staff
  languages (in `admin`, not an area of their own, so `catalog.test.ts`'s list of areas is not touched by two packages).
- **DPIA**: "Package M2" in `docs/privacy/dpia.md`.

**Test first.** The service tests (fakes for every port, `clock.Fixed`) and the repository and River tests were written
and run red (undefined `Service`) before the code; the handler's and the web component's tests were written with them.

**Checks** (compose project `salus-cms-m2`): `make lint` (gofmt, vet, golangci-lint, `check-migrations`, i18n-check),
`make test`, `make archcheck` pass (the second full run; the first failed on `internal/retention`'s applier lists, fixed);
`make web-lint`, `web-typecheck`, `web-test` (1278), `web-build` pass; `make cover` 90.4% total (`clinicexport` 83.0%, `worker` 89.9%, `retention` 100%). Not run: `make e2e`
(not in M2's sets), `make fhir-validate` (no golden file), `make gen-clients` (the contract is unchanged).

**Readings and decisions.**

| Point | Decision | Why |
| --- | --- | --- |
| Where the code lives | `internal/clinicexport`, not `internal/organization/service_export.go` | `organization` is imported by `appointment`, `invoice`, `practitionerrole` and `chargeitemdefinition`, whose services the build reads: an import cycle. WAVE4 allowed "where the contract put the skeleton"; there was none |
| The screen | A section of `ClinicSettings.tsx`, not a route of its own | WAVE4 gives M2 `ClinicSettings.tsx` and not `App.tsx` (M4's); a link would have needed a route |
| The reads' agent | Each owning service audits its read as the admin who asked (their `Caller`); the job's own writes carry the agent `clinic_export_build` | The services audit with a `Caller`'s account; giving them a job agent would change four services outside the package |
| Expiry | 7 days after the request (`expires_at`), the same cut-off as the retention rule | One window for the download and the deletion; a build takes minutes |
| One in progress | A partial unique index (`status = 'queued'`) and `ON CONFLICT DO NOTHING`; a second ask returns the queued one (202) | The contract has no 409 for it |
| Size | The archive is built in memory | No measurement yet; a clinic's CSVs are small. Stream it when a clinic's export is measured to need it |

**Deferred.**

- **The patients' records as FHIR Bundles.** There is no `Patient/$everything` or record Bundle in the code today; R2
  builds "the patient's record as a FHIR Bundle" in this wave. The export should add one `patients/<id>.json` per patient
  through it once R2 is integrated (the same `archive` list in `service_build.go`), each read audited by R2's service.
- **Quotes as CSV.** `invoice` has no clinic-wide quote list yet; M4 builds the quotes page and with it the list. Add
  `quotes.csv` from it after M4 is integrated.
- A worker test of `clinicExportWorker` (fail on the last attempt) against River; the service's `Fail` is tested.
- Not walked in a browser (no browser tool in this session).

## M3: patient import

Status: done, 2026-11-24, on `wt/cms/M3`, one commit `cms(M3)`. No migration: 00071 stays unused.

**Decisions applied** (WAVE4.md, agent's defaults of 2026-11-24, owner to confirm): V5, imported patients are
clinic-registered profiles without a consent and appointments are not imported. Taken here:

| Topic | Decision | Why |
| --- | --- | --- |
| A likely duplicate | A flag on the row, not an error: shown in the preview and saved all the same | "Likely": the admin decides, by removing the row from the file; same name and birth date can be two people |
| Who imports | The clinic's admin only (receptionist and practitioner refused, audited) | M3 says "the admin uploads" |
| The audit | Each likely duplicate found audited as a search of that patient, each saved profile as a create, and the import once with `entity.name` `patient` and the count (`HOPERAT`) | "With the count, never the rows"; the per-profile create keeps every write of a profile audited, as F2's registration |
| Birth date cells | `1990-01-31`, `31.01.1990` (final point optional), `31/01/1990`, or an Excel date number (an XLSX date cell) | What a clinic's spreadsheet holds |
| Phone cells | Spaces, dashes, slashes, points, parentheses dropped; `00` read as `+`; a single leading `0` as `+387` | The clinics are in Bosnia and Herzegovina; the result must be E.164 or the row has `phone_invalid` |
| The preview on the screen | Only rows with errors and likely duplicates are listed; the others are counted | Up to 5 000 rows: show what needs a look |
| The shared reader | `pkg/tabular` (CSV in UTF-8 or Windows-1250, or XLSX through `pkg/xlsx` with its million-cell cap); the row limits stay with each caller; the multipart upload reader moved to `httpx.FormFile` | Two real uses now (O2's price list, M3) |

**Built.**

- **`pkg/tabular`** (new, product-agnostic): O2's reader moved out of `chargeitemdefinition/service_import.go` unchanged
  (encoding, delimiter, blank rows, lines); `chargeitemdefinition` keeps its 1000-row limit and `ErrInvalidImport`.
  **`httpx.FormFile`**: O2's `readUpload`, the part's name and bytes or the 400/413 status.
- **`patient.Import`** (`service_import.go`): admin only; a header in English or bs/hr/sr (`given name`/`first name`/`ime`/
  `име`, `family name`/`last name`/`surname`/`prezime`/`презиме`, `birth date`/`date of birth`/`datum rođenja`/`датум
  рођења`, `phone`/`mobile`/`telefon`/`mobitel`/`mobilni`/`телефон`), the three first required, a column named twice
  refused; at most 5 000 rows; per row `given_name_invalid`, `family_name_invalid`, `birth_date_invalid` (not a date, before
  1900 or not in the past), `phone_invalid`, and the `duplicate` flag (case ignored, against the clinic's patients and the
  earlier rows). Saving registers every row in one transaction or none. `ErrInvalidImport` for a file it cannot read.
- **`Repository.Matching`** (`MatchingClinicPatients`): the clinic's patients (registered by it or with its active care
  consent) whose lower-case names and birth date are one of the keys sent; tested against Postgres for another clinic's
  registered profile, a patient without a consent, another birth date and case.
- **`POST /clinic/patients/import`** (`handler_import.go`): the multipart part `file`, at most 2 MiB (`413 too_large`),
  `save` a boolean, `400 invalid_import`, `403 forbidden`; a row with errors comes back without its `patient`. OpenAPI
  (`PatientImport`, `PatientImportRow`, `ImportedPatient`), `make gen-clients`. The receptionist route table classifies it
  `refused`.
- **Web.** `PatientImport.tsx` (choose a file, the preview, "Import N patients" only when no row has an error) opened from
  a button in `ClinicSettings.tsx`, which then says "N patients imported." Strings in the new `patientImport` area for
  the five staff languages.
- DPIA: "Package M3: patient import".

**Checks** (this worktree's own stack): `make gen-clients` (then no diff), `make lint` (gofmt, vet, golangci-lint,
i18n-check, migrations, Caddy routes), `make test`, `make archcheck`, `make cover` (90.6% total; `pkg/tabular` 100%,
every package at 80% or more), `make web-lint`, `web-typecheck`, `web-test` (1279), `web-build`, `make mobile-analyze`,
`mobile-test` (680): all pass. One test changed on purpose: `web/src/i18n/catalog.test.ts`'s list of areas gains
`patientImport`. No golden FHIR file changed, so no `fhir-validate`. Test first: the tests were written before the code
in each layer, but run red only as a failing build (the package or method did not exist yet), not as a failing assertion.

**Not walked in a browser**: the session's time budget went to the full check set. `PatientImport` and its opening from
the clinic settings are covered by component tests; the integration session should walk `/clinic` (the clinic settings)
at desktop width and 390 px with a file that has errors, a likely duplicate, and a clean one.

**Not measured**: a 5 000-row save is 5 000 profile inserts and 5 000 audit inserts in one transaction, one statement
each; the integration session's stack can time it. The duplicate check is one query for the whole file.

**Gap for the integration (V5).** V5 says the desk records the consent of an imported patient at the first booking, as
for a walk-in. No path does that today: the in-person consent is recorded only by `POST /clinic/patients` when it
registers a walk-in, and `POST /clinic/appointments` books only for a patient with an active care consent. Until a route
records the in-person consent of an existing clinic-registered profile (and the desk's picker offers it), an imported
patient is found by the desk's search but cannot be booked. That is the patient and consent core files and the desk
screens (R2's and F3's), so it is left out of M3.

## M4: quotes page and invite limit

Done 2026-11-24 (wave 4, `docs/clinic-cms/WAVE4.md`). No migration.

**Built.**

- `web/src/Quotes.tsx` at `/quotes`, the admin's only: the clinic's quotes newest first with their state (sent,
  expired), total and validity in the clinic's zone, a state filter (all, sent, expired) and each quote's PDF. Linked
  from the receipts page's toolbar and from the admin's menu (`QuoteIcon`). A practitioner and a receptionist get "This
  page does not exist." on `/quotes`, no menu item, and no request to `/clinic/quotes` (`App.test.tsx`; `/quotes` is in
  F1's list of screens a receptionist does not reach).
- `GET /clinic/quotes/{id}/pdf` (`invoice.ClinicQuotePDF`): the PDF the patient gets, to those `ReadQuote` shows the
  quote to (the clinic's admin, the practitioner who sent it), `not_found` to anyone else, audited as the read, also
  when refused. Classified `refused` for the receptionist in `internal/api/reception_routes_test.go`.
- At most 50 open invites per clinic (V6), `internal/auth/service_invite_limit.go`: the next invite for a new e-mail is
  refused with `ErrInviteLimit`, `409 invite_limit`, audited as a refused create; replacing an open invite for the same
  e-mail stays allowed; accepted and expired invites do not count. The invite form says "The clinic has 50 open
  invites. Wait until some are accepted or expire before sending another." in the five staff languages.

**Decisions taken here** (V6 is the agent's default of 2026-11-24, owner to confirm).

| Topic | Decision | Why |
| --- | --- | --- |
| Opening a quote's PDF from `/quotes` | A new staff route `GET /clinic/quotes/{id}/pdf` in `invoice` | Only the patient's `/me/quotes/{id}/pdf` existed; the spec asks the admin to open the PDF |
| The invite limit's status | `409 invite_limit` | A state of the clinic, not a malformed request; the screen translates the code |
| The limit and a replaced invite | Not counted when it replaces an open invite for the same e-mail | It adds no open invite |
| The limit and the clinic's first admin (`InviteAdmin`) | Not checked | A clinic being created has no invites |
| Two concurrent invites at 49 | May both pass (51 open) | The count is read in the transaction without a lock; the limit bounds abuse, it is not a legal limit. A per-clinic advisory lock is the fix if it ever matters |
| The quotes list's columns | No patient name: date, total, validity, state, PDF | `Quote` carries only the patient id; a name needs a patient read per row (audited), not in the spec |

**Deferred.** The patient's name on `/quotes` (see above). No way for an admin to withdraw an open invite.

## Wave 4 integration

Done 2026-11-24 on `feat/cms`: `wt/cms/R1`, `R2`, `R3`, `M1`, `M2`, `M3`, `M4` merged with `--no-ff` in that order,
then one commit `cms(wave 4): integrate R1, R2, R3, M1, M2, M3, M4`. The wave's migrations follow main's `00066` without
gaps as the packages numbered them: R2 `00067_clinic_consent_withdraw.sql`, R3 `00068_patient_sms_delivery.sql`, M1
`00069_staff_move.sql`, M2 `00070_clinic_export.sql` (R1, M3 and M4 needed none), so nothing was renumbered.

- **Merges.** Conflicts only in shared files: this file and the DPIA (every package's block kept; R3's heading level
  aligned), `api/openapi.yaml` (M3's import path block and R2's rights blocks, rebuilt path by path), the patient
  queries, port and repository test (R2's correction and M3's matching, both kept), the desk catalogs (merged as JSON),
  `internal/worker/worker.go` (R1's stats pool and M2's services), F1's route test (`notServedYet` is now empty: the staff
  move and the export are both served), and `web/src/ClinicSettings.tsx` (M2's export and M3's import, both). `Desk.tsx`
  by hand: R2's patient card on the name, M1's "Move" in the actions. Generated code was not hand-merged: `make sqlc`,
  `make gen-clients` and `flutter gen-l10n` ran after the merges (no ARB changed this wave).
- **Consent for an existing profile.** `POST /clinic/patients/{id}/consent` (`patient.Service.RecordConsent`, through
  `consent.RecordInPerson`): admin and reception, for a profile the clinic registered that has no active care consent
  with it; the read and the consent audited with the staff account; another clinic's profile or an app patient's is not
  found; one already active is 409 `consent_active`. `ClinicPatient.consentActive` (required) is read from the consent
  domain in the desk's search, a registration, a correction and the desk's day. The walk-in's booking step asks for the
  consent first with the walk-in's legal line (`DeskConsent.tsx`), and the patient card offers it while none is active
  (after an import or a withdrawal). e2e `TestDesk_AnImportedPatientGivesTheConsentAtTheDeskAndIsBooked`: import, find at
  the desk without a consent, booking refused, another clinic refused, the consent recorded, booked.
- **Staff move and SMS.** `movedByStaff` schedules the reminders again for every patient (R3's notifier sends a
  profile's by SMS where the clinic offers it); the e-mail with the new time stays for account holders. Tested in the
  service and against Postgres with R3's sent-once cases (`checkProfileMovedByStaff`: the old time's reminders cancelled
  and silent, the new day-before one's first dispatch rolled back, every job twice, one SMS with the new time, counted
  once). Both tests failed against M1's code first.
- **The desk row's reminder.** `DeskVisit.reminderOff` (`sms_off` or `no_phone`, absent when a reminder goes); the row of
  a booked visit says "Reminder will be sent", "No reminder: no phone number" or "No reminder: SMS is off for the clinic".
- **One record builder.** `clinicexport.Service.PatientRecord` builds a registered patient's record: R2's profile and
  consents (`patient.ClinicRecord`, which keeps the admin and registered checks and their audit), the patient's
  appointments at the clinic (`appointment.Search` with the patient) and the receipts the clinic issued to them (new
  `invoice.OfClinicPatient`), each kept only when it is the clinic's. `GET /clinic/patients/{id}/record` moved from
  patient's handler to clinicexport's (same route, same answers; its golden file moved and now holds a visit and a
  receipt, validated against R4); the export writes the same file as `patients/<id>.json` for each patient the clinic
  registered (`patient.Service.Registered`). The receipts are a minimal FHIR R4 Invoice (`invoice.FHIR`). The export also
  writes `quotes.csv` (M4's clinic-wide list; no note, no lines) and a `manifest.txt`. Clinical documents, questionnaire
  answers and history are in neither until the owner and the lawyer decide who may hand them over: the manifest, the
  record's and the export's hints, the DPIA and "Before production" below say so. The routes' "Built in package O2" texts
  are rewritten.
- **5 000 patients.** An import of 5 000 clean rows (222 KB CSV) through the API on the walk's stack, wall clock from the
  client: save 1.50 s, 1.69 s, 1.27 s; the preview 15 ms. Under 30 s, so the inserts are not batched.
- **Tests changed on purpose.** M1's `TestMoveForPatient_SendsAPatientWithoutAnAccountNothing` is now
  `…SchedulesAPatientWithoutAnAccountsRemindersButSendsNoEmail`; M1's `TestMoveByStaff_ConcurrentMovesGiveOne` ran its
  cases from a map although the second depends on the first (it failed one run in a few): an ordered slice now; R2's
  handler tests for the record moved to clinicexport; the golden
  `clinic_record.json`; the bad phone of `TestRegister_RefusesAPractitionerAPatientAndABadField` (`061 000` is now read
  as `+38761000`; `061 ABC` is still refused); two desk tests that built their own ports got the consents port.

**Review** (a fresh subagent over `origin/main` and the working tree; Copilot is out of quota). The two questions —
can one clinic reach another's patients through the import, the export, the record download or a staff move, and does
anything personal reach a log, a job argument or an audit detail — found no path (each checked in the services, the
queries and the row-level security policies). Fixed:

- The record's refusals (403, 404) were audited inside an outer transaction that the refusal rolled back: no audit row
  for another clinic's probe. The builder no longer wraps the owning services' own transactions.
- A withdrawal without an active in-person consent lost its two audit events the same way: now kept (as `RecordConsent`
  keeps `consent_active`'s).
- A passing failure reading the phone or the clinic's rules cancelled a profile's SMS for good: it is retried now.
- One clinic that could not be read stopped the hourly slot generation for all: the others are generated, the failure
  is returned.
- `appointment_guard_role` (00069, not yet on main) let a NULL scope through `IF NOT clinic`: `IF clinic IS NOT TRUE`.
- An import cell with a year alone (`1985`) was read as an Excel date (1905): a date number counts above 2100 only.
- `MoveForPatient`'s and `WithdrawInPerson`'s comments said what the code no longer does.

Declined, with reasons in the PR: the export job not re-checking that the asking admin is still the admin (a job runs
minutes after the request, its reads are audited under that account, and removing an admin while an export is queued is
rare; listed below); the urn:uuid `fullUrl`s with `Patient/<id>` references in the collection Bundle (R2's form, valid R4,
the validator passes; a collection need not resolve its references); a refused consent in an aborted transaction giving
a 500 (only when the profile disappears in the same instant); `Get` and `Download` of another clinic's export not audited
(nothing of a patient is read); the invite limit's unlocked check (M4's accepted bound). Reading 3 (a withdrawn patient's
profile stays reachable at the desk) is the decision below.

**Browser walk** (`docs/clinic-cms/walk/wave4/WALK.md`, its own worktree and compose project, synthetic data, headless
Chromium at 1280 and 390 px, each width its own patients): all ten steps pass at both widths. Fixed after it: the desk's
phone read as the import reads it, the row read again after the card changes something, the withdrawal's buttons, and
the record's and export's hints. Stated, not fixed: visits a withdrawal cancelled are not in the export or the record
(the clinic's scope shows only the visits before the withdrawal; the manifest says so). Left open: the card is reached
only from a desk row (after a withdrawal the admin downloads the record by URL), the phone shown unspaced, the move's one
long select, the long clinic settings page.

**Checks** (this integration's own compose project, `salus-cms-w4`; e2e on fresh volumes): on the final tree `make gen-clients` (no diff),
`make lint` (gofmt, vet, golangci-lint, i18n-check, migrations), `make archcheck`, `make test`, `make cover` (90.4% total,
every package at 80% or more), `make fhir-validate` (131 files, 0 errors; the new `clinic_record.json` included),
`make web-lint` (the one warning in `analytics/ClinicView.tsx` that was there), `web-typecheck`, `web-test` (1308),
`web-build`, `make mobile-analyze`, `mobile-test` (680): all pass. `make e2e` on fresh volumes: one failure, R2's journey,
which this integration had changed to expect the withdrawn patient's cancelled visit in the record (the walk's finding 9:
it is not there); the expectation put back with the reason, the desk journeys rerun, then the whole `make e2e` again on
that stack: pass (403 s). After the review's and the walk's fixes the Go, web and mobile sets were run again (above).

**Decisions taken here.**

| Topic | Decision | Why |
| --- | --- | --- |
| Which patients the export writes a record for | Those the clinic registered; an app patient's visits, receipts and quotes are in the CSV files | R2's builder is the clinic acting for a patient without an account (V2); an app patient exports their own record |
| Where the record builder lives | `clinicexport`, which serves the hand-over's route too | `appointment` and `invoice` import `patient`, so `patient` cannot read visits or receipts |
| The audit purpose of a record read | `PATRQT` for the hand-over, `HOPERAT` for the export | The hand-over is the patient's request; the export is the clinic's operations |
| A receipt in the record | FHIR Invoice with number, status, date, clinic, patient and the gross total | Enough to hand over; the PDF stays the receipt |
| `quotes.csv` | No note and no lines | A note is a practitioner's free text |
| A withdrawn patient the clinic registered | The desk still finds the profile, corrects it, hands over its record and records a new consent; practitioners lose access and the visits ahead are cancelled | The record is kept under retention law and the patient may come back; reading the review's point 3 this way, owner and lawyer to confirm |
| The desk's phone | `061 …` and `00387 …` read as `+387 …` at registration and correction, as the import reads them; the app patient's own profile unchanged | Desks in Bosnia and Herzegovina type local numbers; a diaspora patient's local number is not Bosnian |
| `consent_active` at the screen | Taken as recorded | Another desk recorded it meanwhile |
| The 5 000-row import | Not batched | 1.3 to 1.7 s measured |

## FX: desk fixes from the wave 4 walk

Status: done 2026-11-24 on `wt/cms/FX`, one commit `cms(FX)`. Migration `00072_clinic_record_scope.sql`. Specification:
`WAVE5.md` "FX" and S11; `walk/wave4/WALK.md` findings 5 to 9.

**Built.**

- **The record keeps the visits a withdrawal cancelled (S11, finding 9).** `access.ClinicRecord` (the clinic's admin: all
  of an appointment with an active consent, the name, times and status of every other one of the clinic; nobody else
  anything) and `appointment.Service.ClinicRecord` (the admin only, refused and audited otherwise; the clinic scope plus
  `Repository.SetRecord`, the setting `app.clinic_record`; each appointment and patient name audited as `Search` audits).
  Migration 00072 adds two read-only policies, `appointment_clinic_record` and `patient_clinic_record`. `clinicexport`
  reads its visits through it for both the hand-over (`PatientRecord`) and the export's `appointments.csv` and
  `patients/<id>.json`; the manifest now says the archive holds the cancelled visits too. Every other screen keeps
  `Search` and the consent rule. The e2e journey `desk_rights_test.go` expects the cancelled visit in the record again.
- **Web.** `DeskPatients.tsx`: a "Patients" button on the desk's toolbar opens a search of the clinic's patients (name,
  optional birth date; the existing `GET /clinic/patients`), whatever their consent, each one opening the card of a
  patient the clinic registered (finding 5). `phone.ts` `groupPhone`: a stored +387 number shown as `+387 61 000 399`
  on the desk row, in the search and in the card's field (finding 6; saving strips the spaces as before). `DeskMove.tsx`:
  a "Day" select of the days with free times, then that day's times (finding 7). `ClinicSettingsNav.tsx`: "On this
  page", eight links that scroll to and focus each section of `/clinic` (finding 8); the sections carry ids
  (`ClinicSettings.tsx`, and one `id` each on `ClinicSettingsWaitlist.tsx` and `ClinicProfile.tsx`, which `App.tsx`
  renders below it).
- **New strings** (five staff languages): `desk.patients.title`, `desk.patients.hint`, `desk.move.day`,
  `admin.settingsNav`, `admin.settingsNavDetails`.
- **e2e web journey** `test/e2e/web/desk-rights.spec.ts` (started at the end of the rights journey as a second admin):
  the search finds the withdrawn patient and opens their card; the move dialog picks a day then a time; the settings
  navigation reaches its last section; at 1280 and 390 px with no sideways scroll. `E2E_WALK_DIR` keeps screenshots.
- **DPIA**: "Package FX" in `docs/privacy/dpia.md`.

**Test first.** Each Go and web test was written and run red before its code: `TestClinicRecord_…` in `access` (undefined
function) and `appointment` (undefined method, then the fake's RLS hiding the cancelled visit, which asked for
`SetRecord`), `TestSetRecord_…` against Postgres, the clinicexport fake moved to `ClinicRecord` and the manifest's
assertion; `phone.test.ts`, `DeskPatients.test.tsx`, the Day test in `DeskMove.test.tsx`, `ClinicSettingsNav.test.tsx`
and the new cases in `Desk.test.tsx`, `DeskPatient.test.tsx`, `ClinicSettings.test.tsx`.

**Checks** (compose project `salus-cms-fx`): the fast layer after each change; `runner.sh full` (`make lint test archcheck
web-lint web-typecheck web-test web-build`): the first run failed `make lint` on two findings of this package (funlen on the
rights journey, fieldalignment on the appointment fake), fixed; the second passed (web-test 1325, web-lint with the one
warning in `analytics/ClinicView.tsx` that was there plus one of `ClinicSettingsNav.tsx`, fixed after and checked with the
fast layer's oxlint: 0 warnings). `make e2e`: every journey passed except the rights journey, whose record was read before
the revocation's job had cancelled the visit (the desk's day hides it at once by the consent rule, so the existing wait
passed early); it now waits for the record to show the visit cancelled, and the `TestDesk_` journeys rerun on the same
stack passed (with the new web journey at both widths). `make cover`: pass (90.4% total, every package at 80% or more). Not run: `make fhir-validate` (no golden FHIR file
changed); `make i18n-check` ran in the fast layer and in `make lint`.

**Decisions taken here.**

| Topic | Decision | Why |
| --- | --- | --- |
| How the record reads a withdrawn patient's cancelled visits | A scope of its own (`app.clinic_record`) with read-only policies, set only by `appointment.Service.ClinicRecord`, which only the clinic's admin passes | S11 says "the job's own scope, audited"; the everyday clinic scope keeps the consent rule, so no screen changes |
| The level of such a visit | Name (times, status, practitioner, patient's name), as a past visit after a revocation | What the hand-over needs; nothing clinical |
| The patient search | Its own dialog on the desk, not the walk-in's picker | The picker registers and books; the search only opens cards |
| The phone's grouping | `+387 NN NNN NNN(N)`; other countries as stored | The desk's numbers are Bosnian; another country's grouping is not known here |
| No row in the table above | 5.0 adds the rows for 5.0, FX and D1 to D6 | Its contract, to keep the merge clean |

**Deferred.** Nothing of FX. The move's long single select is replaced; a calendar-style day picker was not asked for.

## Facts

- Main's next free migration number on 2026-11-21: `00060`. After wave 1: A `00060`, D `00061`; wave 2 takes E `00062`,
  H `00063`, I `00064`, Q1 `00065`.
- Copilot is out of quota until 2026-12-01: the integration session has a fresh subagent review the wave's diff too.

## Before production

- **The lawyer's five questions** (R2): (1) whether a clinic may act as the channel for the GDPR rights (rectification,
  withdrawal, access) of a patient it registered without an account, and what identity check the desk must make first;
  (2) whether an in-person withdrawal needs a signed or written record beyond the staff account and time Salus stores;
  (3) whether a withdrawal may cancel the visits ahead at once, as an app patient's revocation does; (4) whether the
  handed-over file (profile, consents, visits and receipts as FHIR JSON; no clinical documents, questionnaire answers or
  history) meets Art. 15 and Art. 20, in what form the clinic must hand it over, and who may hand over the clinical
  content; (5) erasure: how long the registered profile and its consent must be kept under the medical-record retention
  law.
- **Erasure and subject rights for every patient.** There is no erasure path for any patient (`patient`, `consent` and
  `account` are `keep` in `internal/retention/rules.go`); restriction and objection are not built either. A
  subject-rights feature, for app patients and registered profiles alike.
- **An SMS provider** with a DPA and an entry in `docs/privacy/processors.md`; until then `pkg/sms` is the fake and the
  clinic's SMS option cannot be on, so a patient without an account gets no reminder.
- **The ESET adapter** (package G, Q6): the fiscal reporting of receipts.
- Also open from wave 4: what the clinic may read of a withdrawn patient's cancelled visits (export, record); the export
  job re-checking the admin; the patient card reachable from a desk search.

## Waiting on a person

- The lawyer: the five questions on V2 in "Before production" above.
- The owner: confirm or change the Q1 to Q9 defaults; plan prices (Q2) and the invoicing company (Q3) before D ships.
- The owner: the ESET technical specification and test access (Q6); an accountant on VAT (Q9); the legal retention
  period for receipts.
- The owner: confirm WAVE4's V1 to V6 and the readings the packages and this integration recorded (above), and read the
  wave's new copy (desk `patient.*`, `move.*`, `consent.*`, `reminder.*`; `patientImport`; the quotes page; the export
  and record hints); sr-Cyrl was transliterated from sr-Latn and wants a native reader.

## 5.0: the wave 5 contract

Status: done, 2026-11-24, on `wt/cms/5.0`, one commit `cms(5.0)`. Migration `00071_wave5_contract.sql`. Contract only,
as 3.0: no behaviour; every new service method refuses with `errors.ErrUnsupported`, with a test that says so.

**Built.**

- **Migration 00071.** `data_request` (S1 to S3): `account_id` (the patient's own account, when they filed it in the
  app), `patient_id` (their profile), `organization_id` (the clinic target; null is the account target), `kind`,
  `status`, `received_at`, `due_at`, `extended_until` with `extension_reason` (a code), `filed_by_account_id`,
  `answered_at` with `answer_code` (a code; fulfilled takes `done` or `done_partly_kept`, refused the others), `version`.
  Row-level security: `data_request_own` (the patient's account scope: their own, filed by themselves, naming only
  their own profile) and the clinic's `data_request_clinic`, `_update`, `_insert` (those targeting it; it files only
  for a patient it registered, by its own staff, never in a patient's name, S10); the operator reads and writes as the
  table's owner (salusctl, D5). The trigger `data_request_guard` (every role, the owner too): filed received,
  unanswered, not extended (and, but for the owner, received now); never deleted, moves forward only, final once
  answered or withdrawn, extended once, who/target/kind/filing/due never change; the patient's scope only withdraws, and
  a request the patient filed is withdrawn by the patient only. `data_request_change`: the append-only history of
  status changes (a trigger refuses update, delete, truncate), seen and written where its request is seen and with its
  current status; `changed_by_account_id` is the patient's own account in their scope, staff of the clinic or null (a
  job) in the clinic's. `account.closed_at`, `erase_after` (both or neither, erasure after closure).
  `patient.analytics_objected_at` (salus_app may update it). `patient_restriction` (patient and clinic, when, by which
  staff of that clinic or null for the erasure job, optionally for a request of that clinic and patient): the clinic's
  scope only; salus_app selects, inserts and deletes (lifting), and the clinic never lifts the erasure's restriction (a
  restrictive policy). Ownership: `data_request`, `data_request_change` by `datarequest`;
  `patient_restriction` by `patient`. Retention: all three kept (`dataRequestRules`), pending the lawyer.
- **Go.** The new domain `datarequest` (context `privacy` in `archcheck/config.yaml`): `model.go` (kinds, targets,
  statuses, answer codes, extension reasons, the deadlines `DueMonths`, `ExtensionMonths`, `EraseAfter`, `ExportWindow`,
  `Request`, `Change`, `Filing`, `Answer`, `Export`, the callers `Patient` and `Staff`), `errors.go`, `repository.go`
  (the port D1 implements) and `service.go`: `File`, `Mine`, `Withdraw`, `FileOnBehalf`, `ClinicRequests`,
  `ClinicRequest`, `Answer`, `Extend` (D1), `Export`, `ExportArchive` (D2), `CloseAccount`, `ReopenAccount` (D3).
  `patient/service_restriction.go`: `Restrict`, `LiftRestriction` (D4).
- **OpenAPI.** Every wave 5 route with `x-salus-package`: `/me/data-requests` (list, file), `/me/data-requests/{id}/withdraw`
  (cms-d1), `/me/data-requests/{id}/export` and `/export/archive` (cms-d2), `/me/account/close` (cms-d3),
  `/clinic/data-requests` (list), `/{id}` (read), `/{id}/answer`, `/{id}/extend`, `/clinic/patients/{id}/data-requests`
  (cms-d1), `PUT` and `DELETE /clinic/patients/{id}/restriction` (cms-d4); the schemas `DataRequest*`,
  `AccountClosure`; three error codes (`invalid_data_request`, `data_request_closed`, `already_extended`); the tag
  `privacy`. A contract test (`api/clinic_wave5_contract_test.go`): the enums are the Go constants, every route names its
  package, no operator route. `make gen-clients` (the Dart `PrivacyApi` and models, web `schema.d.ts`).
- **Docs.** The DPIA's "Package 5.0" block; this record and the table rows for 5.0, FX and D1 to D6.

**Contract changes** (where the code fit better than the plan's text):

| Plan's text | As built | Why |
| --- | --- | --- |
| "the patient (account or clinic-registered profile)" | `account_id` and `patient_id`, checked: the account target needs the account, a clinic target the profile | A clinic must see which of its patients asked; an account target may come from an account with no profile yet |
| `patient` at a clinic: `restricted_at`, per clinic | Its own table `patient_restriction` (patient, clinic), owned by `patient` | A patient with an account has one profile for every clinic; a column could not be per clinic |
| analytics objection per patient | `patient.analytics_objected_at` | One timestamp, read by `clinicstats` (D4) |
| `/me/data-requests/{id}/export` (status and download) | `/export` (status) and `/export/archive` (download) | As `/clinic/export/{id}` and `/archive`: the archive is built by a job, ask, poll, download |
| `/clinic/patients/{id}/restriction` | `PUT` restricts (optionally naming the request), `DELETE` lifts, both idempotent | Restricting and lifting are the two actions of D4's detail |
| "extended_until and its reason" | `extension_reason` a code: `complex`, `many_requests` | Art. 12(3)'s two grounds; no free text about health |

**Decisions** (agent's, for the owner to review):

| Topic | Decision |
| --- | --- |
| Answer codes | `done`, `done_partly_kept` (erasure with the record kept, S6), `identity_unverified` (Art. 12(6)), `manifestly_unfounded`, `excessive` (Art. 12(5)), `retention_law` (Art. 17(3)), `no_data` |
| The patient's scope | Matches on `app.account_id` only: a clinic-registered patient has no app session, so their requests are the clinic's |
| Wrong password at closure | `403 wrong_credentials`, as the login's |
| Restriction by the erasure job | `restricted_by_account_id` null; staff restrictions name staff of the same clinic |
| Go fields for the new `account` and `patient` columns | Columns only (as 3.0): the fields come with the reads in D3 and D4 |

**Left to the packages** (their migrations are reserved: D1 `00073`, D2 `00074`, D3 `00075`, D4 `00076`, D5 `00077`):

- **D1.** `query.sql`, `datarequestdb` (its entry in `sqlc.yaml`) and `repository_postgres.go` implementing the port;
  how the deadline job reads open requests across clinics (a role or policy, in its own migration if needed); the
  handlers and their wiring in `internal/api/app.go`.
- **D2.** Where the export is stored (a table or columns: status, object key, wrapped key, expiry), its job and its
  retention rule.
- **D3.** The erasure record on the erasure request (what was erased, what was kept, under which rule): the guard
  freezes an answered request, so either closure files an account `erasure` request kept `in_progress` until the job
  answers it (its `due_at` at least `erase_after`: one month can be 28 days, `EraseAfter` is 30) or the record is its own
  table; `/me/account/close` files no request in the contract. How the job restricts the kept record and erases
  (`salus_retention` has no INSERT or UPDATE by 00014's rule: a grant, or a scope); erasure blanks the profile and account a request names
  (both are referenced by `data_request`, which is never deleted).
- **D4.** The restrictive reads (every staff read of a restricted patient at that clinic), the admin's audited opening
  with a reason, and `clinicstats` leaving objected patients out.

**Test first.** The migration tests (`migrations/wave5_contract_test.go`) were written first and run red without the
migration (every table and column missing), then green with it. The `datarequest` skeleton test was run red (no
package) before the code; the retention rule test red before the rules; the patient restriction test was written with
its two refusing methods.

**Review** (a fresh subagent over the diff and the plan docs, Copilot being out of quota). Fixed, each with its failing
case: a clinic could insert a request in a patient's name (the clinic's policy split, its insert needs no account); a
request could be inserted answered, extended or backdated (the guard on insert); history rows could carry any status or
claim a job (the request's current status, a job only in the clinic's scope); a clinic could withdraw a request the
patient filed; a clinic could lift the erasure's restriction or name another clinic's request; the export's status read
was marked unaudited. Dropped: the INSERT grant on `patient_restriction` to `salus_retention` (00014: that role has no
INSERT; D3 decides). Left, recorded under D3: the erasure record and the one month against 30 days.

**Checks** (this worktree's compose project, `salus-cms-5-0`): `make gen-clients`, `make lint` (gofmt, vet,
golangci-lint, i18n-check, migrations), `make test`, `make archcheck`, `make web-lint`, `web-typecheck`, `web-test`,
`web-build`, `make mobile-analyze`, `mobile-test` (680): all pass. The first full run failed twice, both fixed: funlen
on a migration test (split) and `TestReceptionRoutes_EveryRouteOfTheContractIsClassified` (the 13 new routes classified
for the receptionist, the desk files on behalf, and listed in `notServedYet` until D1 to D4 mount them). An
order-dependent case in the closure test (map order) was fixed by resetting the row per case. Not run: `make cover`,
`make e2e` and `make fhir-validate` (no route is served and no golden file changed).

**Copy waiting for the owner:** none. 5.0 adds no user-facing string; the API returns codes only.

## D1: requests and deadlines

Status: done, 2026-11-24, on `wt/cms/D1`, one commit `cms(D1)`. Migration `00073_datarequest_deadlines.sql`.

**Built.**

- **Service** (`internal/datarequest/service.go`): `File` (the patient's account, or a clinic where they gave a consent,
  active or revoked: 404 otherwise), `Mine`, `Withdraw` (open only, at its version), `FileOnBehalf` (the clinic's admin
  or receptionist, for a patient the clinic registered, S10; row-level security holds the registration), and the
  admin's `ClinicRequests`, `ClinicRequest` (with the history), `Answer` (fulfilled with `done`/`done_partly_kept`,
  refused with a refusal's code) and `Extend` (once, with a coded reason). Every filing, list, read and change is
  audited in its transaction (entity `data_request`, `Patient/<id>` when the request names one; `PATRQT` for the
  patient's own, `HOPERAT` for the clinic's); refusals to other staff are audited in their own transaction.
- **Deadlines** (`service_deadline.go`): due at the end (23:59:59) of the same date one month after receipt in the
  answerer's time zone (the clinic's `time_zone`; the operator's `Europe/Sarajevo` for the account), clamped to the
  month's last day (31 January → end of February, as Regulation 1182/71 counts months). An extension ends on the same
  date two months after the due date, capped at the database's `due_at + interval '2 months'` (UTC), an hour sooner
  when the clocks went back in between. `QueueWarnings`: the 7-day and the 1-day warning, each once per deadline (an
  extension's deadline is warned of again), none once the deadline has passed.
- **Job**: `data_request_deadlines` (hourly, also when the scheduler starts leading) on the worker's default queue; it
  queues one `data_request_deadline_warning` (`{request_id, days}`) on the notifier's `email` queue per warning, marked
  on the request (`deadline_warned_at`) and audited as the job's (agent `data_request_deadlines`). Its sending is D5's:
  until D5 registers that worker, the notifier fails those jobs as an unknown kind (they retry, then are discarded).
- **Migration 00073**: `data_request.deadline_warned_at` (salus_app may update it); the job's scope
  `app.data_request_deadlines = 'all'` (two policies: open requests only, select and update), and a trigger that lets
  that scope change nothing but the mark.
- **Repository** (`query.sql`, `datarequestdb` and its `sqlc.yaml` entry, `repository_postgres.go`), the River port
  (`repository_river.go`), the handler (`handler_http.go`) for the eight cms-d1 routes, mounted in `internal/api/app.go`
  (`internal/api/datarequest.go`), the worker and the scheduler's periodic job. The eight routes are off
  `notServedYet`; the reception test holds them to their classes.
- **Docs**: the DPIA's "Package D1" block.

**Contract changes.**

| Contract | As built | Why |
| --- | --- | --- |
| `ClinicRequest` returns the request | Returns the request and its history (`[]Change`) | The read of one carries its history (`DataRequest.history`); one transaction, one audit event |
| `Staff` has `Admin` | Also `Reception` | The receptionist files on behalf (the contract's 403 says "not the clinic's admin or receptionist") |
| `Repository` | Also `SetDeadlines`, `OpenDeadlines`, `MarkWarned`; a `Jobs` port; `Deadline`, `DeadlinesArgs`, `WarningArgs`, `WarnDays`, `LastWarnDays` in `model.go` | The deadline job |
| `Ports` has `Repo`, `Clock` | Also `Tx`, `Audit`, `Profiles`, `Consents`, `Clinics`, `Jobs` | What the methods consume |

**Decisions** (agent's, for the owner to review):

| Topic | Decision | Why |
| --- | --- | --- |
| "A clinic where the patient has a record" | A clinic they gave a consent, active or revoked | Every booking needs one; a revoked one still left a record there |
| The due date's time | 23:59:59 of the due date in the answerer's time zone | "Within one month": the whole last day counts |
| The account target's time zone | `Europe/Sarajevo` (`OperatorTimeZone`) | Salus's own; no setting exists |
| Warning after a missed run | The latest warning whose time has come, once; none after the deadline | The answerer is warned before, never told late that a day is left |
| How the job reads across clinics | Its own scope with two policies and a guard trigger, not a role | As `app.subscription_run`: the app role, no new login |
| A request filed on behalf by a profile the clinic did not register | 404 from row-level security (42501) or a missing row (23503) | The service has no read of another clinic's patients; the database holds S10 |
| A staff session on `/me/data-requests` | 403 in the handler | As the contract's 403 |
| Answer or extend by non-admins | Refused (and audited) before the If-Match and body are read | Every route answers a stranger the same (the reception test) |
| Withdrawal's history time | The transaction's time (`now()`) | The port carries no time for it; the answer's is `answered_at` |
| History order at the same instant | By the status's progression | Tied timestamps came back in id order |

**Test first.** The migration test ran red (no column, no policy) before 00073, then green. The service tests
(`service_requests_test.go`, `service_deadline_test.go`) ran red (no ports, no `QueueWarnings`) before the code. The
repository, River and worker tests were written with their code and first run after it; the handler tests with the
handler. The API journey (`app_datarequest_test.go`) failed once on its own wrong version (3, not 4).

**Review** (a fresh subagent over the diff, Copilot being out of quota). Fixed: two overlapping runs of the job could
queue one warning twice (`MarkWarned` now matches the mark it read); a practitioner's malformed filing on behalf got a
400 before the role check (now refused and audited first). Left, for the owner or later packages: an overdue request
can still be extended (Art. 12(3) wants the patient told within the first month: a SPEC question); warnings queued
before D5 is deployed are lost (deploy D1 with D5); staff on `/me/data-requests` get an unaudited 403 (the contract's
withdraw lists no 403).

**Checks**: `make lint`, `make test`, `make archcheck` (the full set, twice: the first failed on lint, fixed: funlen of
`api.New`, field alignment of test structs, `time.Now` in tests, a test's complexity); see the package's report.

**Copy waiting for the owner:** none. D1 adds no user-facing string; the API returns codes only.

## D2: the patient's export

Status: done 2026-11-24 on `wt/cms/D2`, one commit `cms(D2)`. Migration `00074_data_request_export.sql`. Specification:
`WAVE5.md` "D2" and S4, S10, S11; the contract of 5.0 above.

**Built.**

- **Migration 00074.** `data_request_export` (the request it answers as its key, the account, requested, ready and
  expiry times, status `queued`/`ready`/`failed`, size, KEK id and wrapped data key): the patient's own scope only
  (`app.account_id`), for an access or portability request of theirs; one queued per patient (partial unique index);
  salus_app inserts and updates the build's columns, salus_retention selects and deletes. `data_request_export_fulfil`:
  a SECURITY DEFINER function that answers the scope's own open access or portability request `done` once its export
  is ready, with a history entry by the job (null), and returns whether it did. Ownership: `datarequest`. Retention
  rule `data_request_export`: deleted with its archive 7 days after it was ready (a failed one 7 days after it was
  asked for).
- **`internal/datarequest`.** `service_export.go`: `QueueExport` (an access or portability request filed in the app
  queues its export and the build job in the caller's transaction, audited; `ErrExportInProgress` while one is
  queued; nothing for another kind or a request the clinic filed, S10), `Export` (status, audited as the patient's
  read; a ready one past its 7 days reads expired), `ExportArchive` (decrypted as it is read, audited), `FailExport`.
  `service_export_build.go`: `BuildExport`, the job: the patient's record as a FHIR Bundle (`record.json`), their
  account data for the account target (`account.json`: id, e-mail, language, when MFA was set up; never a credential)
  and `manifest.txt`, one zip encrypted with its own data key (`pkg/envelope`, bound to the request id), stored under
  `datarequest/<request id>`, then ready for `ExportWindow` and the request fulfilled.
  `service_export_retention.go`: `ExportPurger`. `repository_export.go` (the port `ExportRepository`, `Store`,
  `ExportJobs`), `repository_export_postgres.go`, `repository_export_river.go`, `query_export.sql` → `datarequestexportdb`
  (its own sqlc entry and package, read only by `repository_export_postgres.go`), `handler_http_export.go` (`GET /me/data-requests/{id}/export`
  and `/export/archive`, `Cache-Control: no-store`, `not_found` for another's, not ready or expired).
- **`internal/clinicexport/service_own.go`.** The record builder in the patient's own scope: `OwnRecords.Bundle` reads
  the profile (`patient.Me`), consents (`consent.Mine`), appointments (`appointment.Search` as the patient: every one of
  theirs, the visits a withdrawal cancelled included, S11) and receipts (`invoice.Mine`), each audited by its service as
  the patient's own request, keeps one clinic's for a clinic target, and writes the hand-over's Bundle (`bundleJSON`).
- **Wiring.** `internal/worker/datarequestexport.go` (the `data_request_export_build` job, 3 attempts, the last failure
  marks the export failed and logs the request id only; registered in `worker.go`), the purger in `retention.go`;
  `internal/api/datarequestexport.go` mounts the two routes (`app.go`), and they leave `notServedYet` in
  `reception_routes_test.go`. DPIA: "Package D2" in `docs/privacy/dpia.md`.

**Left for the integration with D1** (parallel packages, each writing only its own files):

- D1's `File` calls `s.QueueExport(ctx, r)` inside its transaction after inserting an access or portability request
  (the request's own id and account), and maps `ErrExportInProgress` to a 409 (`invalid_data_request` is the
  contract's nearest code; or add one to the OpenAPI). D1's service in the API is built with
  `Export: dataRequestExportsOn(...)` so the queue has its ports; the two routes can then share D1's service.
- The export's queries are their own sqlc package, `datarequestexportdb`, beside D1's `datarequestdb`: archcheck lets
  only `repository_postgres.go` (D1's file) import `datarequestdb`. The integration may fold `query_export.sql` into
  D1's `query.sql` and `repository_export_postgres.go` into `repository_postgres.go`, and drop the package and its
  `sqlc.yaml` entry. `scripts/cover.sh` now leaves out every sqlc package (`internal/*/*db` with sqlc's `db.go`), not
  only `<domain>db`.
- `datarequest` test files: both packages may declare `TestMain` (`pgtest.Main`) and constants like `accountA`; keep
  one of each.
- `ExportTransactor`/`ExportAuditor` are this package's names for the two ports so as not to clash with D1's; if D1's
  `Transactor`/`Auditor` have the same methods, replace these with them.
- `make sqlc` here regenerated every domain's `models.go`: 5.0 had not regenerated them after 00071, so they gain the
  wave 5 tables. Generated, deterministic; no hand edit.

**Test first.** The datarequest service tests (`service_export_test.go`, fakes for every port, `clock.Fixed`) and the
clinicexport builder tests (`service_own_test.go`) were written before their code; the retention rule's test ran red
(no rule) before the rule; the purger's test ran red (undefined) before it. The migration test
(`migrations/data_request_export_test.go`: own scope, one queued, fulfil only when ready and only in the account's
scope, once, with its history entry) was written with the migration and run green; the repository and River tests
run against Postgres. The handler tests were written alongside the handler.

**Review** (a fresh subagent over the diff, Copilot being out of quota). Fixed, each with a test: a request withdrawn or
answered while its archive was built still got a downloadable export (`MarkExportReady` now needs the request open; the
build then marks the export failed); an export stuck `queued` (a discarded job) blocked every later export for good
(the purger now also deletes a never-ready export 7 days after it was asked for, with its archive). Confirmed: the
SECURITY DEFINER function cannot answer another account's request or escalate. Left, recorded below: a clinic target
reads (and audits) the patient's consents, visits and receipts of every clinic before keeping that clinic's; the
export's own audit events name the table, not the request; two overlapping builds of one export could race on its
object key; the clinical content (S4).

**Checks** (compose project `salus-cms-d2`): the fast layer after each change (`runner.sh fast` over every package, as
`migrations/` changed: its failures were the retention test lists and the worker's retention-pool list, which name
every purged table and gained `data_request_export`); targeted reruns of `datarequest`, `clinicexport`, `retention`,
`worker` and golangci-lint on the touched packages. `runner.sh full` (`make lint test archcheck`): the first run failed
`make lint` on five findings of this package (`time.Now` in three tests, fieldalignment on two test structs), fixed; the
second passed `make lint` and `make test` and failed `make archcheck` (the export repository imported
`datarequestdb`, which only `repository_postgres.go` may), fixed with the package of its own; then `make archcheck`,
`go vet ./...` and the `datarequest` tests and golangci-lint passed on the final tree. Not run: `make cover`, `make e2e` (no journey changes before D1 files
requests), `make fhir-validate` (no golden file changed).

**Decisions taken here.**

| Topic | Decision | Why |
| --- | --- | --- |
| Where the export is stored | Its own table `data_request_export`, keyed by the request | The request table is append-only in spirit and guarded; the export is temporary and deleted after 7 days |
| Who marks the request fulfilled | `data_request_export_fulfil` (SECURITY DEFINER) in the patient's scope | 00071's guard lets the patient's scope only withdraw and no salus_app scope reaches an account-target request; the function answers only the scope's own request with a ready export |
| When the export is queued | When the access or portability request is filed (`QueueExport`, called by D1's `File`) | The OpenAPI has only the status `GET`; a `GET` that creates would be a side effect |
| The 7 days | From when the archive is ready (`expires_at = ready_at + 7 days`) | S4: "downloadable in the app for 7 days" |
| A failed build | Export `failed`, the request stays open for its answerer; deleted after 7 days | No retry route in the contract; the deadline warnings (D5) still apply |
| A clinic target | The record of that clinic only (consents, visits, receipts there; the shared profile), no account data | S1: the clinic answers for its record; the account is Salus's |
| A patient the clinic registered | No export from Salus; the hand-over (wave 4) is theirs | WAVE5 D2's last sentence |
| The manifest's text | English plain text in the archive, as the clinic export's | A file, not a screen; the app's own copy is D6's |
| Audit | The patient's own reads by their services (PATRQT); the export's create, status read and download as `data_request_export`; the job's writes under `AgentExport` | "every read audited as the patient's request" (S4) |

**Deferred.**

- **Clinical content in the archive.** S4 says everything the patient can read in the app; this archive holds the
  profile, consents, visits, receipts and account data, not yet the encounters, signed documents, questionnaire
  answers and their files of the app's history. They need the history's sources (encounter, composition,
  questionnaireresponse, binary) built in the worker and their FHIR mappers exported; the manifest says they are not
  in it yet. A follow-up package.
- No retry after a failed build (no route in the contract).
- A clinic target reads the patient's own consents, visits and receipts of every clinic, each audited, then keeps that
  clinic's: the patient's own reads have no clinic filter (`appointment.Filter.OrganizationID` is the service's). A
  clinic filter on `consent.Mine`, `appointment.Search` and `invoice.Mine` for the patient would make it minimal.
- The export's own audit events (`data_request_export`: create, status read, download, the job's update) carry the
  table and the patient's account as the agent (the job's: `AgentExport`), not the request's id: AuditEvent's entity is
  a FHIR reference and a request is no FHIR resource.
- Two overlapping attempts of one build (River runs a retry only after the attempt ends, so only after a rescue of a
  stuck job) would write the same object key with different data keys; a per-build object key would rule it out.
- A browser walk: D2 has no screen (D6 shows the download).
