# Clinic system: progress

Step 0's record is in `docs/plans/clinic-cms.PROGRESS.md` (merged, PR #107, `b18974f`). From wave 1 on, each package
writes under its own heading here.

| Package | Status | Commit |
| --- | --- | --- |
| A | done 2026-11-21 | `cms(A)` on `wt/cms/A` |
| B | done 2026-11-21 | `cms(B)` on `wt/cms/B` |
| C | done 2026-11-21 | `cms(C)` on `wt/cms/C` |
| D | not started | |
| E | hold (after wave 1) | |
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
