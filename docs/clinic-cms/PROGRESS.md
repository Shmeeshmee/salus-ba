# Clinic system: progress

Step 0's record is in `docs/plans/clinic-cms.PROGRESS.md` (merged, PR #107, `b18974f`). From wave 1 on, each package
writes under its own heading here.

| Package | Status | Commit |
| --- | --- | --- |
| A | done 2026-11-21, integrated 2026-11-22 | `cms(A)`, merged into `feat/cms` with `cms(wave 1)` |
| B | done 2026-11-21, integrated 2026-11-22 | `cms(B)`, merged into `feat/cms` with `cms(wave 1)` |
| C | done 2026-11-21, integrated 2026-11-22 | `cms(C)`, merged into `feat/cms` with `cms(wave 1)` |
| D | done 2026-11-21, integrated 2026-11-22 | `cms(D)`, merged into `feat/cms` with `cms(wave 1)` |
| E | wave 2, not started (after wave 1 is on main) | |
| H | wave 2, not started (after wave 1 is on main) | |
| I | wave 2, not started (after wave 1 is on main) | |
| L | wave 2, not started | |
| Q1 | wave 2, not started | |
| Q2 | wave 2, not started | |
| Q3 | wave 2, not started | |
| F | hold (Q1) | |
| G | hold (Q6) | |

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
| The accountant's CSV | Readable headers in the clinic's language, amounts as decimals |

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

## I: membership invoice document and the suspended listing

## L: landing page

## Q1: questionnaire branching, rules and server

## Q2: questionnaire branching, builder and preview

## Q3: questionnaire branching, patient app

## F: reception role and desk

## G: ESET adapter

## Facts

- Main's next free migration number on 2026-11-21: `00060`. After wave 1: A `00060`, D `00061`; wave 2 takes E `00062`,
  H `00063`, I `00064`, Q1 `00065`.
- Copilot is out of quota until 2026-12-01: the integration session has a fresh subagent review the wave's diff too.

## Waiting on a person

- The owner: confirm or change the Q1 to Q9 defaults; plan prices (Q2) and the invoicing company (Q3) before D ships.
- The owner: the ESET technical specification and test access (Q6); an accountant on VAT (Q9); the legal retention
  period for receipts.
