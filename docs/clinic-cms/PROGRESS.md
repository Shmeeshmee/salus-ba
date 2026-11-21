# Clinic system: progress

Step 0's record is in `docs/plans/clinic-cms.PROGRESS.md` (merged, PR #107, `b18974f`). From wave 1 on, each package
writes under its own heading here.

| Package | Status | Commit |
| --- | --- | --- |
| A | not started | |
| B | not started | |
| C | built | this commit |
| D | not started | |
| E | hold (after wave 1) | |
| F | hold (Q1) | |
| G | hold (Q6) | |

## A: visit modes

## B: price list

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
