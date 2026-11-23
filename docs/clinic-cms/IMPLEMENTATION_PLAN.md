# Clinic system: implementation plan

Packages A to G of `docs/plans/clinic-cms.md` ("Tickets and packages"), unchanged. Step 0 is merged. This file says
how they run; where it differs from that document's "Waves" and "Checks per PR" on branches, worktrees and pull
requests, this file wins.

## How this plan runs

- One branch `feat/cms` per wave, cut from `origin/main`. Each package of a wave runs in its own worktree on
  `wt/cms/<package>`, opened by the owner's `wave` tool, and makes one commit `cms(<package>): <summary>`.
- A package never pushes and never opens a pull request. After the wave, one integration session merges the
  `wt/cms/*` branches into `feat/cms` in the wave's order, runs the checks and opens one pull request for the wave.
- Notes for one package are in `notes/<package>.txt`: read yours before you start.

## Waves

| Wave | Packages | Why they run together |
| --- | --- | --- |
| 1 | A, B, C, D | Built 2026-11-21. Each owns its own files (`docs/plans/clinic-cms.md`, the file ownership table); integration order A, B, C, D |
| 2 | E, H, I, L, Q1, Q2, Q3 | Disjoint files (the table under "Wave 2 packages"); shared files get one appended block each; migrations numbered apart. Integration order Q1, Q2, Q3, H, I, E, L |
| later | F (Q1 of the owner's questions), G (Q6) | On hold for the owner |

Migration numbers after wave 1's integration (A `00060`, D `00061`): E `00062`, H `00063`, I `00064`, Q1 `00065`. A
package that needs no migration leaves its number unused.

## Rules for every package

1. Read `CLAUDE.md`, `docs/plans/clinic-cms.md` whole, `docs/plans/clinic-cms.PROGRESS.md` (step 0, and what it left
   to the wave), this file, `PROGRESS.md` here, and `notes/<package>.txt`.
2. Build only your package: its row in "Tickets and packages", its row in the file ownership table and its entry in
   "Notes for the wave-1 packages" of `docs/plans/clinic-cms.md`.
3. Test first, as `CLAUDE.md` "TDD" says, mobile and service tests included: wave 1 wrote several tests after their
   code. Write the failing test, run it, then the code; say in the report where you did not.
4. In shared files (`api/openapi.yaml`, `internal/api/app.go` and the router, `internal/worker`, `internal/config`,
   `cmd/salusctl`, `ownership.yaml`, `internal/retention/rules.go`, `docs/privacy/dpia.md`, the i18n catalogs,
   `sqlc.yaml`) add only your own block. Run `make gen-clients` and `make sqlc` for your own changes; the integration
   session regenerates them after the merges.
5. Your stack is your own: set `COMPOSE_PROJECT_NAME` to your worktree's folder name in lower case, and
   `SALUS_IMAGE_TAG`, in the git-ignored `deployments/.env`, with the override without published host ports, as
   `docs/plans/appointments.md` "Worktrees and stacks" describes. Other packages are running in sibling worktrees at
   the same time: never touch them, never `git stash`.
6. Where the plan conflicts with the code, stop and report instead of improvising.
7. Write your section in `docs/clinic-cms/PROGRESS.md` under your own heading only.
8. A package with a screen walks it in a browser at desktop width and 390 px against its own stack before it
   commits; if it cannot, the report says why under DEFERRED.
9. In the report, add after the usual fields: every new user-facing string for the owner, every place you departed
   from the contract, and anything the integration session must do for you (`INTEGRATION NOTES:`).

## Check sets

- `go`: `make lint`, `make test`, `make archcheck`, `make cover`, `make i18n-check`; `make fhir-validate` when a
  golden FHIR file changed; `make gen-clients` leaves no diff.
- `web`: `make web-lint`, `make web-typecheck`, `make web-test`.
- `mobile`: `make mobile-analyze`, `make mobile-test`.
- `e2e`: `make e2e`.

## Definition of done

The package's behaviour is built and tested as its notes in `docs/plans/clinic-cms.md` require; `go` passes, and `web`,
`mobile` and `e2e` pass where the package touched them; the package's part of the DPIA, the retention rules and
`ownership.yaml` is written; `PROGRESS.md` has its section; one commit `cms(<package>): <summary>`; the tree is clean;
the compose stack is stopped.

## Packages

| Package | Depends on | Check sets | Done when |
| --- | --- | --- | --- |
| A, B, C, D | step 0 | | Built in wave 1 |
| E | wave 1 on main | go, web, mobile, e2e | A patient picks a service when booking; the clinic sees it and the receipt starts from it; the admin sees online and in-clinic work per practitioner |
| H | wave 1 on main | go, web, mobile | Receipt lines keep their order; PDF labels are in the catalogs; a failed fiscal report can be retried; the patient sees their receipts in the app; the CSV suits an accountant |
| I | wave 1 on main | go, web | The clinic gets its membership invoice as a PDF by e-mail and on the screen; a suspended clinic is not in the public directory |
| L | nothing | landing | The landing page sells the clinic system with telemedicine and the monthly membership |
| Q1 | nothing | go | Publishing accepts the full branching model and refuses broken logic; a response holds only answers on the path taken |
| Q2 | nothing (FHIR `enableWhen` is the contract) | web | The builder sets branches on the question answered, shows the tree and the join, warns on broken dependencies, and previews live |
| Q3 | nothing (FHIR `enableWhen` is the contract) | mobile | The app shows and hides branches live, warns before discarding answers, submits only the path taken |
| F | C, the owner's Q1 | go, web | Hold |
| G | C, the owner's Q6 | go | Hold |

Check set `landing`: `make landing-lint`, `make landing-typecheck`, `make landing-test`, `make landing-build`.

## Wave 2 packages

Decisions applied (agent's defaults, 2026-11-22, owner to confirm): a payment returns a past-due or suspended clinic
to active; the payment term is 14 days; billing starts on the first 1st after the trial; membership invoice numbers
are gapless; prices are in BAM only; a company's invoice is headed "Račun" in Croatian; the accountant's CSV has
readable headers and decimal amounts.

| Package | Its own files | Touches, appending its own block |
| --- | --- | --- |
| E | `internal/appointment/service_service.go`, `internal/clinicstats/service_practice.go` and its handler; web `OnlinePractice.tsx`; mobile `service_picker.dart` | `appointment` core files (the `service_id` column, the booking request), web `AppointmentDetail.tsx` and `ReceiptDialog.tsx` (the pre-filled line), mobile booking screens, `api/openapi.yaml` |
| H | `internal/invoice/service_retry.go`, `internal/invoice/templates/pdf/<language>.json`; mobile `receipts_screen.dart` | `invoice` core files (the line position, the CSV), web `Receipts.tsx` (the retry action), `scripts/i18n-check.sh` |
| I | `internal/subscription/service_document.go`, `internal/notification` templates for the invoice e-mail | `subscription` core files, `organization` (the public listing's query), web `Subscription.tsx` (the download), `internal/config` (the seller's details) |
| L | everything under `landing/` | nothing else |
| Q1 | `internal/questionnaire/validate_logic.go`, `internal/questionnaireresponse/service_enable.go`, `test/testdata/enablewhen/` | `questionnaire` and `questionnaireresponse` core files, golden FHIR files |
| Q2 | web `builder-logic.ts`, `BuilderBranch.tsx`, `BuilderPreview.tsx`, `enableWhen.ts` | web `Builder.tsx`, `ItemCard.tsx`, `builder.ts`, the `questionnaire` catalogs |
| Q3 | mobile `enable_when.dart` and its use in the questionnaire form screens | the mobile ARB catalogs |

### E: service on a booking and the online practice report (epic SAL-47)

Package A already gives a practitioner video hours beside in-person ones, and B's price list already says which
services are offered by video: those two are the "online hours" and "which services may be done online" of the plan,
and E builds neither again. E builds what is missing:

- A booking may name a service: `appointment.service_id` (nullable, the same clinic's, offered in the appointment's
  mode; migration), an optional `service_id` on the booking request, kept on a reschedule. The patient picks from
  `GET /organizations/{id}/services` for the slot's mode before confirming, or skips. Nothing is charged.
- The clinic sees the service on the appointment, and "Issue receipt" starts with that service's line.
- The online practice report for the admin, `GET /clinic/reports/practice`: per practitioner and period, by visit
  mode: hours offered, visits held, and the receipted amount of those visits. A web screen beside the receipts
  report. Equality tests against the raw appointments and receipts, as the clinic statistics have.

### H: receipts follow-ups (epic SAL-45)

- `invoice_line.position` (migration): lines come back in the order they were entered, on screen, PDF and report.
- The PDF's labels move from the Go map into `internal/invoice/templates/pdf/<language>.json`, covered by
  `make i18n-check`; "Račun" for a company's invoice in Croatian.
- A receipt whose fiscal report failed after the job gave up can be retried by the admin; a fiscalized one never.
- The patient's receipts in the app: a list and the PDF, from the existing `/me/receipts` routes, ten languages.
- The accountant's CSV: readable headers in the clinic's language, amounts as decimals.

### I: membership invoice document and the suspended listing (epic SAL-44)

- The membership invoice as a PDF (`pkg/pdfdoc`), at `GET /clinic/subscription/invoices/{id}/pdf` and attached to an
  e-mail to the clinic's billing e-mail when it is issued, in the clinic's default language. The seller's details
  come from `internal/config` and stay `[SELLER]` placeholders in `.env.example`.
- A suspended or cancelled clinic is not in the public directory. Tests: its existing appointments, records,
  documents, exports and subject rights keep working.

### L: landing page (no epic)

Only `landing/`. Salus is the system a private clinic runs on (in-person and video visits, price list, receipts ready
for the FBiH fiscalization law, questionnaires, documents, reminders and waitlist, chat, analytics), with telemedicine
built in so its doctors can also work online after hours. Monthly membership per practitioner, two plans, a 30-day
trial, `[PRICE]` placeholders kept. A short section for patients from the diaspora and abroad. Nothing that says
yearly subscription or a cut of payments; no claim of fiscal certification; what is not built is marked as coming.
The page's structure, look and languages stay.

### Q1, Q2, Q3: questionnaire branching (ticket SAL-43, read it whole)

The contract between the three is FHIR R4 only: `enableWhen` with several conditions, `enableBehavior` (`all`, `any`),
the operators `exists`, `=`, `!=`, `>`, `<`, `>=`, `<=`; a branch is a `group` carrying the condition; joining back
is the next unconditional item; a condition names only an earlier question. No extension.

- **Q1** (Go): publishing accepts that model and refuses broken logic with an error on the item (a missing or later
  question, an option that no longer exists, an operator that does not fit the type); published questionnaires with
  today's single `is` condition keep working unchanged. On submit the server evaluates the conditions itself and
  refuses an answer to an item that is not enabled (`answer_not_enabled`), so a response holds only the path taken.
  Q1 writes shared test vectors (`test/testdata/enablewhen/*.json`: a questionnaire, answers, the enabled items)
  covering every operator and type, `all` and `any`, nesting and the ticket's gynaecology example.
- **Q2** (web): "If the answer is X, ask ..." set on the question being answered; the tree shown as indented
  branches with a marker where they join; the condition editor for every type in the ticket; the warning when a
  question others depend on is changed or deleted, naming the branches; the live preview. The evaluator is
  `enableWhen.ts`, pure and tested.
- **Q3** (mobile): the form shows and hides branches as answers change, warns before discarding answers already
  given in a branch that is about to hide, and submits only enabled items' answers. The evaluator is
  `enable_when.dart`, pure and tested. All ten languages.

The integration session runs Q1's vectors against `enableWhen.ts` and `enable_when.dart`, fixes any difference, and
adds the e2e of both paths of the gynaecology example and the changed-answer case.
