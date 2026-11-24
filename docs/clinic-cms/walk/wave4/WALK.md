# Wave 4 walk

Walked 2026-11-24 in the worktree `salus-walk4`: detached `HEAD` at `39c54fc3` (`feat/cms`, "Merge branch 'wt/cms/M4'")
with the wave 4 changes staged on top and not committed (71 files, +2102 −181). Own compose project `salus-walk4`
(published ports 22000–22007), started as `make e2e` starts it (`make e2e-up web-install e2e-install`), with headless
Chromium from the `playwright` service and the web app's Vite dev server, as the web journeys of `make e2e` run.

**Seeded through the API** (synthetic data only), once per width, each width on its own clinic: a clinic created with
`salusctl create-clinic` and listed, its address (Kanton Sarajevo, Centar, Synthetic street 1) and billing details, a
price list with one in-person service (Placeholder examination, BAM 50.00), one practitioner (Bo Syntheticson) with
in-person hours every day 00:00 to 23:30 in 30-minute slots, a second admin and a receptionist, each invited and enrolled
with TOTP. Before the walk, at the desk through the API: a patient the import would duplicate (1280: Lejla Primjerova,
3 March 1971; 390: Merima Primjerović, 7 July 1968), and a second patient with a visit today at 12:30 and a treatment
quote from the admin (2 × Placeholder examination, BAM 100.00; 1280: Nermina Uzorkova; 390: Dženana Uzorković).

**The import file** had a header and five rows: the patient walked in below (1280: Ajla Importovana, 12.04.1979,
`061 000 311`; 390: Hana Importović, 21.01.1993, `061 000 411`), one clean row with a phone, one without, the likely
duplicate, and one bad row (1280: birth date `31.02.1985`; 390: no family name). The corrected file is the same without
the bad row (4 rows). Phones +38761000xxx are placeholders.

The admin and the receptionist signed in through the sign-in page with a password and a TOTP code, each in their own
browser context. The driver (a Playwright spec and its Go seeding test) was temporary and is not committed; its
screenshots and the files it fetched are here: the export archives (`1280-export.zip`, `390-export.zip`), the walked-in
patient's record (`1280-record.json`, `390-record.json`) and the quotes' PDFs (`1280-quote.pdf`, `390-quote.pdf`).

## Steps

| # | Step | Expected | 1280 px | 390 px | Screenshots |
| --- | --- | --- | --- | --- | --- |
| 1 | The admin imports a patient file in clinic settings | The preview shows the bad row and the duplicate; saving imports the rows | pass (findings 1, 8) | pass (findings 1, 8) | [1280 settings](1280-01-admin-clinic-settings.png), [errors](1280-02-import-preview-errors.png), [clean](1280-03-import-preview-clean.png), [saved](1280-04-import-saved.png); [390 settings](390-01-admin-clinic-settings.png), [errors](390-02-import-preview-errors.png), [clean](390-03-import-preview-clean.png), [saved](390-04-import-saved.png) |
| 2 | The receptionist walks in the imported patient | The picker finds them; picking asks for the in-person consent; recording it opens the booking; a time today is booked | pass | pass | [1280 picker](1280-05-walkin-picker.png), [consent](1280-06-walkin-consent-asked.png), [booking](1280-07-walkin-consent-recorded-booking.png), [booked](1280-08-booked.png); [390 picker](390-05-walkin-picker.png), [consent](390-06-walkin-consent-asked.png), [booking](390-07-walkin-consent-recorded-booking.png), [booked](390-08-booked.png) |
| 3 | "Move" the visit to another free time | The row shows the new time | pass (finding 7) | pass (finding 7) | [1280 dialog](1280-09-move-dialog.png), [moved](1280-10-moved.png); [390 dialog](390-09-move-dialog.png), [moved](390-10-moved.png) |
| 4 | The row's reminder state | "No reminder: SMS is off for the clinic" | pass | pass | [1280](1280-11-reminder-row.png), [390](390-11-reminder-row.png) |
| 5 | Open the patient card from the row's name, correct the phone, save | The phone is saved | pass, only in the +387 form (findings 2, 3, 6) | pass, only in the +387 form (findings 2, 3, 6) | [1280 refused](1280-12-card-phone-local-refused.png), [saved](1280-13-card-phone-saved.png); [390 refused](390-12-card-phone-local-refused.png), [saved](390-13-card-phone-saved.png) |
| 6 | Record the withdrawal of consent; the future visit is cancelled | The visit leaves the desk's day; the card then offers "Record consent" | pass (finding 4) | pass (finding 4) | [1280 confirm](1280-14-withdraw-confirm.png), [withdrawn](1280-15-withdrawn.png), [desk](1280-16-desk-after-withdrawal.png); [390 confirm](390-14-withdraw-confirm.png), [withdrawn](390-15-withdrawn.png), [desk](390-16-desk-after-withdrawal.png) |
| 7 | The admin downloads that patient's record | A FHIR Bundle with the patient's data | pass by URL only; the record lacks the cancelled visit (findings 5, 9, 10) | the same (findings 5, 9, 10) | [1280](1280-17-admin-card-download.png), [390](390-17-admin-card-download.png) |
| 8 | The admin asks for the clinic export, waits, downloads it | The archive with prices, staff, appointments, receipts, quotes, patients/, manifest | pass (findings 9, 11) | pass (findings 9, 11) | [1280 building](1280-18-export-building.png), [ready](1280-19-export-ready.png); [390 building](390-18-export-building.png), [ready](390-19-export-ready.png) |
| 9 | The admin opens /quotes and a quote's PDF | The quote with its state; the PDF 200 application/pdf | pass | pass | [1280](1280-20-admin-quotes.png), [390](390-20-admin-quotes.png) |
| 10 | As the receptionist: /quotes in the web app, and the export, quotes and import APIs | Not found in the app; 403 or 404 with no data | pass | pass | [1280 web](1280-21-reception-quotes-web.png), [api](1280-22-reception-quotes-api.png); [390 web](390-21-reception-quotes-web.png), [api](390-22-reception-quotes-api.png) |

## What happened

1. **Import.** 1280: the file with the bad row reads "1 of 5 rows has errors. Correct them in the file and choose it
   again.", then the duplicate's notice, then "Line 4: Lejla Primjerova, born 3 March 1971, may already be at the clinic"
   and "Line 5: the birth date is not a valid date in the past"; no Import button is offered (all or none, by design), only
   Cancel. The corrected file reads "4 patients read. Nothing is saved until you import.", the duplicate's line again, and
   "Import 4 patients"; saved: "4 patients imported."
   390: the same, with "Line 5: the family name is missing or longer than 100 characters" for the bad row.
2. **Walk-in.** 1280: searching "Importovana" lists "Ajla Importovana 12 April 1979" (and "Register a new patient");
   picking her shows "There is no active consent to care from this patient. Record the consent they give in person before
   booking.", the checkbox "The patient gave the clinic their consent to care, in person." and "Record consent". Once
   recorded, the booking step opens with her; 22 free times today (12:00–12:30 onwards, 12:30 taken by the seeded visit);
   chose 13:00–13:30 and Placeholder examination. The row: "13:00–13:30 Ajla Importovana Booked · 12 April 1979 ·
   +38761000311 · At the clinic · Placeholder examination · No reminder: SMS is off for the clinic · Arrived Move Issue
   receipt".
   390: the same for Hana Importović (13:00–13:30).
3. **Move.** 1280: the dialog shows the visit (Tuesday, 24 November 2026, 13:00–13:30), the practitioner and "New time"
   with 679 free times over 15 days, grouped by day; chose 14:30–15:00. "The visit was moved."; the row reads
   14:30–15:00.
   390: the same.
4. **Reminder.** 1280: "No reminder: SMS is off for the clinic" on the row.
   390: the same.
5. **Patient card.** 1280: the name on the row opens "Patient card" (profile, "Consent to care" with "Record withdrawal of
   consent"; no record section for the receptionist). Typing the phone as `061 000 399` (the form the import accepts) and
   saving is refused: "Enter the phone number with its country code, such as +387 61 123 456." `+387 61 000 399` saves:
   "The profile is saved." (stored +38761000399, as the admin's search showed later). The row behind the card still shows
   the old +38761000311.
   390: the same with `061 000 499`, then `+387 61 000 499` saved.
6. **Withdrawal.** 1280: "Record withdrawal of consent" shows the effect text with "Confirm withdrawal" and "Keep the
   consent"; confirmed: "The withdrawal is recorded.", and the card at once offers the consent checkbox and "Record
   consent" again. The row was still on the desk right after; on the first reload (2 s later) the visit was gone and the
   desk listed only the seeded visit.
   390: the same (gone after 1 s).
7. **Record (admin).** 1280: the admin's desk no longer shows the patient (her only visit was cancelled), so her card
   cannot be opened; the card of the other registered patient shows "Patient's record" with "Download the record", which
   downloads `patient-<id>.json`. Her record by URL as the admin (`GET /clinic/patients/{id}/record`): 200
   `application/fhir+json`, `attachment; filename="patient-<id>.json"`: a Bundle of type `collection` with entries
   **Patient, Consent** (status `inactive`); no Appointment.
   390: the same.
8. **Export.** 1280: "Export the clinic’s data" → "The archive is being built…"; ready after about 5 s with "Download the
   archive" and "Available until 1 Dec 2026, 11:53."; downloaded as `salus-export.zip`. Files: `prices.csv`, `staff.csv`,
   `appointments.csv`, `receipts.csv` (header only, no receipts), `quotes.csv` (1 quote), `patients/<id>.json` × 6 (the
   seeded two and the four imported), `manifest.txt`. `appointments.csv` holds one row (the seeded visit); the walked-in
   patient's cancelled visit is missing. Entries per patient file: seeded duplicate Patient, Consent; quote patient
   Patient, Consent, Appointment; walked-in patient Patient, Consent; the three other imported patients Patient only.
   390: the same shape (6 patient files, one appointment row).
9. **Quotes.** 1280: /quotes lists "Tuesday, 24 November 2026: BAM 100.00, valid until Thursday, 24 December 2026 · Sent ·
   Open PDF"; the PDF answers 200 `application/pdf` (one page).
   390: the same.
10. **Refusals (receptionist).** 1280: /quotes in the web app: "This page does not exist." `POST /clinic/export` 403
    `{"error":"forbidden"}`; `GET /clinic/quotes` 403 `{"error":"forbidden"}`; `POST /clinic/patients/import?save=true`
    and the preview both 403 `{"error":"forbidden"}`. The browser on `/clinic/quotes` shows `{"error":"forbidden"}`.
    390: the same.

Manifest (both widths, `Patients' records in this archive: 6`):

```
Salus clinic data export

prices.csv        the price list
staff.csv         the practitioners and their roles
appointments.csv  every appointment, with the patient's id and name
receipts.csv      every receipt and invoice issued
quotes.csv        every treatment quote sent
patients/<id>.json  the record of each patient the clinic registered, as a FHIR R4 Bundle (collection):
                  the profile, the consents given to the clinic, the appointments and the receipts at the clinic

Patients' records in this archive: 6

Not in this archive: clinical documents, questionnaire answers and the patient's history. Who may hand
them over is not decided yet. A patient with their own Salus account has no file in patients/; their
appointments, receipts and quotes at the clinic are in the CSV files.
```

Nothing ran past the screen's edge and no page scrolled sideways at either width on any step (`scrollWidth` equal to
the viewport on every screenshot: /clinic, /desk, /quotes).

## 5 000-patient import

Same stack, each run as the admin of a fresh clinic made with `salusctl create-clinic`, a CSV of 5 000 clean synthetic
rows (`given name,family name,birth date,phone`, 222 KB), `POST /clinic/patients/import`, wall clock from the Go client
in the `playwright` container (request sent to the whole JSON answer read). Another project's test run (`salus-cms-w4`)
was using the host's CPU at the same time.

| Run | Time | Answer |
| --- | --- | --- |
| Preview (`save=false`), empty clinic | 15 ms | 200, 5 000 rows, 0 errors, 0 duplicates |
| Save 1 | 1.50 s | 200, saved, 5 000 rows |
| Save 2 | 1.69 s | 200, saved, 5 000 rows |
| Save 3 | 1.27 s | 200, saved, 5 000 rows |

## Findings

Not fixed in this session.

1. **The preview with an error offers no way on but a new file** (step 1, both widths). With one bad row the screen says
   to correct the file and choose it again and offers only Cancel; the 4 clean rows cannot be imported without editing
   the file. This is the documented all-or-none rule (`api/openapi.yaml` `/clinic/patients/import`, `PatientImport.tsx`
   shows "Import N" only when `wrong.length === 0`), so it works as specified; noted in case "import the clean rows,
   skip the bad ones" is wanted. Not a bug.
2. **The card refuses the phone in the form the import accepts** (step 5, both widths). The import turns `061 000 311`
   into +38761000311 (`internal/patient/service_import.go:157` `phoneOf`: leading `0` → `+387`, `00` → `+`), but the
   card's save only strips spaces (`web/src/DeskPatient.tsx:81`) and the API refuses anything not E.164, with "Enter the
   phone number with its country code". A receptionist who types a local number as the patient says it is refused. The
   walk-in register form (`web/src/PatientPicker.tsx`, `register`) has the same rule. Small: apply the import's
   normalisation on the server for the desk's create and update too.
3. **The desk row keeps the old phone after the card saves a new one** (step 5, both widths). The card updates only the
   name button's state (`DeskPatient.tsx:18,27`, `setShown`); the row's details line reads `visit.patient.phone`
   (`web/src/Desk.tsx:135`) and is not reloaded, so it shows +38761000311 until the page is reloaded. Small: let the row
   take the saved profile (or reload the day on save).
4. **"Record withdrawal of consent" looks like plain text, not a button** (steps 5 and 6, both widths, visible in
   `*-13-card-phone-saved.png` and `1280-17-admin-card-download.png`). It is a bare `<button>` outside `.actions`
   (`web/src/DeskPatient.tsx:137`), and the base style (`web/src/app.css:110`) removes border and background. Its
   confirmation then uses the green primary style for "Confirm withdrawal" (`DeskPatient.tsx:146`) where `button.danger`
   exists (`app.css:165`). Small.
5. **After the withdrawal the patient's card cannot be reached, so the record cannot be downloaded from the app** (step
   7, both widths). The card opens only from a desk row (`web/src/Desk.tsx:143`, `DeskPatient`); the withdrawal cancels
   the patient's visits, so the row leaves the day and there is no other list of the clinic's patients in which to find
   them. The record was fetched by URL. Medium: a search on the desk (the walk-in's picker already finds patients) that
   opens the card would do.
6. **The card shows the phone raw while the user typed it spaced** (step 5): the field shows +38761000302 for a stored
   number and the row `+38761000311`, while the hint asks for `+387 61 123 456`. Cosmetic.
7. **The move's time list is one select of 679 options** (step 3, both widths): 15 days of 30-minute slots in optgroups
   (`web/src/DeskMove.tsx`, `daysAhead = 14`). It works, but at the desk a day picker, then the times of that day, would
   be quicker to use. Larger change; usability only.
8. **Clinic settings has become one very long page** (step 1, 390 px: about 5 200 px tall). The import button sits
   below Reminders and the export, and Waitlist and the abroad profile follow after "Membership and billing"
   (`web/src/ClinicSettings.tsx:88-118`, `web/src/App.tsx` `/clinic`). No overflow; navigation only.
9. **A withdrawn patient's cancelled visit is missing from the record and from the export** (steps 7 and 8, both
   widths). `appointments.csv` ("every appointment") has no row for the walked-in patient's visit, and her record has
   only Patient and Consent though the manifest says the record holds "the appointments and the receipts at the clinic".
   Both read through `appointment.Service.Search` as the admin (`internal/clinicexport/service_build.go:249`,
   `internal/clinicexport/service_record.go:49`), which drops what `access.Appointment` rates `None`
   (`internal/appointment/service_read.go:151`); after a revocation that rule gives only past visits
   (`internal/access/access.go:127-133`), so a future visit cancelled by the revocation disappears from the clinic's own
   export. Medium: the export and the hand-over need a read of the clinic's appointments not gated on the consent (the
   clinic holds the cancelled booking), or the manifest must say what is left out.
10. **The record's hint undersells it** (step 7): "The profile and the consents as a FHIR file" (`desk.patient.recordHint`,
    `web/src/i18n/desk/*.json`, shown at `DeskPatient.tsx:49`), while the record carries the appointments too (the quote
    patient's record: Patient, Consent, Appointment). Small copy change, with the `legal` tag, in the five staff languages.
11. **The export's hint is out of date** (step 8): "the price list, the staff, the appointments and the receipts as CSV
    files" (`admin.exportHint`, `web/src/i18n/admin/en.json:65`), while the archive also holds `quotes.csv` and a FHIR
    record per registered patient. Small copy change in the five staff languages.

## After the walk

Fixed in the integration commit (not walked again; covered by tests):

- Finding 2: the desk's registration and the card's correction read a phone as the import does (`061 …` and `00387 …`
  are `+387 …`); `TestRegisterAndCorrect_ReadALocalPhoneAsTheImportDoes`.
- Finding 3: closing the card after a change reads the desk's day again (the row's phone, a cancelled visit).
- Finding 4: "Record withdrawal of consent" is a bordered button in the card's actions, and "Confirm withdrawal" is the
  danger button.
- Findings 10 and 11: the record's and the export's hints name what they hold (visits, receipts, quotes, the patients'
  records) and that clinical documents are not in them, in the five staff languages.
- Finding 9 is stated, not fixed: the export's `manifest.txt` says that visits a withdrawal cancelled are in neither the
  CSV files nor the record (the clinic's scope shows a withdrawn patient's visits from before the withdrawal only). Showing
  them needs a decision on what the clinic may read after a withdrawal; listed in `docs/clinic-cms/PROGRESS.md`.

Left open (`docs/clinic-cms/PROGRESS.md`): finding 5 (the card is reached only from a desk row, so after a withdrawal
the admin downloads the record by its URL; a patient search on the desk that opens the card), 6 (the phone shown
unspaced), 7 (the move's one long select of free times), 8 (the long clinic settings page). Finding 1 is the specified
all-or-none import.
