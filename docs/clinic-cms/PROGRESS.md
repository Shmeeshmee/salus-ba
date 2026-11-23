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
| 3.0 | done 2026-11-23 | `cms(3.0)` on `wt/cms/3.0` |
| F1 | not started (wave 3, after 3.0) | |
| F2 | not started (wave 3, after 3.0) | |
| F3 | done 2026-11-23 | `cms(F3)` on `wt/cms/F3` |
| O1 | not started (wave 3, after 3.0) | |
| O2 | not started (wave 3, after 3.0) | |
| X1 | not started (wave 3, after 3.0) | |
| X2 | not started (wave 3, after 3.0) | |
| O3 | hold (after F2) | |
| X3 | hold (a payment provider) | |

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

Not started.

## F2: patients registered by the clinic and bookings by staff

Not started.

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

Not started.

## O2: price list import and the clinic data export

Not started.

## X1: treatment quotes

Not started.

## X2: clinic profile for patients abroad

Not started.

## Facts

- Main's next free migration number on 2026-11-21: `00060`. After wave 1: A `00060`, D `00061`; wave 2 takes E `00062`,
  H `00063`, I `00064`, Q1 `00065`.
- Copilot is out of quota until 2026-12-01: the integration session has a fresh subagent review the wave's diff too.

## Waiting on a person

- The owner: confirm or change the Q1 to Q9 defaults; plan prices (Q2) and the invoicing company (Q3) before D ships.
- The owner: the ESET technical specification and test access (Q6); an accountant on VAT (Q9); the legal retention
  period for receipts.
