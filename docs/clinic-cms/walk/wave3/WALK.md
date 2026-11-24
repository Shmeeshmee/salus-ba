# Wave 3 front desk walk

Walked 2026-11-23 on `main` at `5f3d860c` (PR #117 squash-merged), in a fresh worktree with its own compose project
(`salus-walk3`, no published host ports), with headless Chromium from the `playwright` service and the web app's Vite dev
server, as the web journeys of `make e2e` run.

**Seeded through the API** (synthetic data only): a clinic created with `salusctl create-clinic` and listed, its address
(Kanton Sarajevo, Centar, Synthetic street 1) and billing details, a price list with two in-person services (Placeholder
examination, BAM 50.00; Placeholder follow-up, BAM 30.00), one practitioner (Bo Syntheticson) with in-person hours
every day 00:00 to 23:30 in 30-minute slots, a second admin and a receptionist, each invited and enrolled with TOTP.

**Each width** ran the whole journey from its own fresh patient and visit: at 1280 px Vedrana Syntheticova (receipt
no. 1, reversed by no. 2), at 390 px Amra Syntheticova (receipt no. 3, reversed by no. 4). The receptionist and the admin
signed in through the sign-in page with a password and a TOTP code. The driver (a Playwright spec and its Go seeding
test) was temporary and is not committed; its screenshots and the two receipt PDFs it fetched are here.

## Steps

| # | Step | Expected | 1280 px | 390 px | Screenshots |
| --- | --- | --- | --- | --- | --- |
| 1 | Sign in as the receptionist | Only "Front desk" in the menu, and / lands on /desk | pass | pass | [1280](1280-01-reception-signin.png), [390](390-01-reception-signin.png) |
| 2 | Open the walk-in and search for the patient | No patient found, and the way to register one is offered | pass (findings 5, 6) | pass (finding 5) | [1280](1280-02-walkin-search.png), [390](390-02-walkin-search.png) |
| 3 | Fill in the new patient form | The form asks for given name, family name, birth date, phone and the in-person consent line | pass | pass | [1280](1280-03-walkin-register-form.png), [390](390-03-walkin-register-form.png) |
| 4 | Register the walk-in | The patient is registered and the booking step opens with them | pass | pass | [1280](1280-04-walkin-registered.png), [390](390-04-walkin-registered.png) |
| 5 | Choose today's in-person time and a service | Today's free in-person times and the price list are offered | pass | pass | [1280](1280-05-book-choose.png), [390](390-05-book-choose.png) |
| 6 | Book the visit | The visit is booked and on the desk's list with the service | pass | pass | [1280](1280-06-booked.png), [390](390-06-booked.png) |
| 7 | Mark "Arrived" | The row shows the arrival time | pass | pass | [1280](1280-07-arrived.png), [390](390-07-arrived.png) |
| 8 | Issue receipt from the row | The receipt dialog opens with the booked service as its first line | pass | pass | [1280](1280-08-receipt-dialog.png), [390](390-08-receipt-dialog.png) |
| 9 | Issue the receipt | The receipt is issued and its number shown | pass (finding 2) | pass (finding 2) | [1280](1280-09-receipt-issued.png), [390](390-09-receipt-issued.png) |
| 10 | Open the receipt's PDF | The PDF opens for the receptionist | fail (finding 1) | fail (finding 1) | [1280](1280-10-receipt-pdf.png), [390](390-10-receipt-pdf.png) |
| 11 | As the receptionist: The patient's visit page (questionnaire answers, documents) in the web app | Refused, with none of the patient's data | pass | pass | [1280](1280-11-refused-answers-web.png), [390](390-11-refused-answers-web.png) |
| 12 | As the receptionist: The patient's questionnaire answers by URL | Refused, with none of the patient's data | pass | pass | [1280](1280-12-refused-answers-api.png), [390](390-12-refused-answers-api.png) |
| 13 | As the receptionist: The visit's consultation and its documents by URL | Refused, with none of the patient's data | pass | pass | [1280](1280-13-refused-documents-api.png), [390](390-13-refused-documents-api.png) |
| 14 | As the receptionist: The patient's history by URL | Refused, with none of the patient's data | pass | pass | [1280](1280-14-refused-history-api.png), [390](390-14-refused-history-api.png) |
| 15 | As the receptionist: The chat in the web app | Refused, with none of the patient's data | pass | pass | [1280](1280-15-refused-chat-web.png), [390](390-15-refused-chat-web.png) |
| 16 | As the receptionist: The conversations by URL | Refused, with none of the patient's data | pass | pass | [1280](1280-16-refused-chat-api.png), [390](390-16-refused-chat-api.png) |
| 17 | Sign in as the admin and open the visit | The visit with the patient, the service and its arrival | pass (findings 3, 4) | pass (findings 3, 4) | [1280](1280-17-admin-visit.png), [390](390-17-admin-visit.png) |
| 18 | Open /receipts | The desk's receipt is listed | pass | pass | [1280](1280-18-admin-receipts.png), [390](390-18-admin-receipts.png) |
| 19 | Reverse it: the question | The reversal asks for the refund method | pass | pass (finding 7) | [1280](1280-19-admin-reverse-ask.png), [390](390-19-admin-reverse-ask.png) |
| 20 | Confirm the reversal | A reversal is issued and the receipt is marked reversed | pass | pass | [1280](1280-20-admin-reversed.png), [390](390-20-admin-reversed.png) |

## What happened

1. **Sign in as the receptionist.** 1280: / went to /desk; the menu holds only "Front desk"
   390: the same.
2. **Open the walk-in and search for the patient.** 1280: "No patient found." with "Register a new patient"
   390: the same.
3. **Fill in the new patient form.** 1280: filled in, the consent line checked
   390: the same.
4. **Register the walk-in.** 1280: registered (patient 8d064cf0); the booking step shows the patient
   390: registered (patient c6d165a9); the booking step shows the patient
5. **Choose today's in-person time and a service.** 1280: 5 free times offered; chose 21:00–21:30 and "Placeholder examination"
   390: 4 free times offered; chose 21:30–22:00 and "Placeholder examination"
6. **Book the visit.** 1280: the row reads: 21:00–21:30 Vedrana Syntheticova Booked 6 June 1966 · +38761000666 · At the clinic · Placeholder examination Arrived Issue receipt
   390: the row reads: 21:30–22:00 Amra Syntheticova Booked 14 April 1984 · +38761000444 · At the clinic · Placeholder examination Arrived Issue receipt
7. **Mark "Arrived".** 1280: the row's status reads "Arrived 20:58"
   390: the row's status reads "Arrived 20:59"
8. **Issue receipt from the row.** 1280: the dialog opens; line 1 is "Placeholder examination · BAM 50.00"
   390: the same.
9. **Issue the receipt.** 1280: issued (receipt no. 1); notice: Receipt no. 1 was issued.
   390: issued (receipt no. 3); notice: Receipt no. 3 was issued.
10. **Open the receipt's PDF.** 1280: the desk offers no link to the PDF; by its URL it opens (200, application/pdf)
   390: the same.
11. **As the receptionist: The patient's visit page (questionnaire answers, documents) in the web app.** 1280: the web app says "This page does not exist."
   390: the same.
12. **As the receptionist: The patient's questionnaire answers by URL.** 1280: 200 with an empty search (nothing shown)
   390: the same.
13. **As the receptionist: The visit's consultation and its documents by URL.** 1280: 200 with an empty search (nothing shown)
   390: the same.
14. **As the receptionist: The patient's history by URL.** 1280: 404 {"error":"not_found"} 
   390: the same.
15. **As the receptionist: The chat in the web app.** 1280: the web app says "This page does not exist."
   390: the same.
16. **As the receptionist: The conversations by URL.** 1280: 403 {"error":"forbidden"} 
   390: the same.
17. **Sign in as the admin and open the visit.** 1280: the visit page shows the patient and "Placeholder examination"; it reads: Your clinic's setup is not finished. Continue setup Calendar Vedrana Syntheticova Monday, 23 November 2026, 21:00–21:30 calendar.status.arrived At the clinic Se
   390: the visit page shows the patient and "Placeholder examination"; it reads: Your clinic's setup is not finished. Continue setup Calendar Amra Syntheticova Monday, 23 November 2026, 21:30–22:00 calendar.status.arrived At the clinic Servi
18. **Open /receipts.** 1280: listed: No. 1 · BAM 50.00 23 Nov 2026, 20:58 · Patient · Cash Fiscalized Open PDF Reverse
   390: listed: No. 3 · BAM 50.00 23 Nov 2026, 20:59 · Patient · Cash Fiscalized Open PDF Reverse
19. **Reverse it: the question.** 1280: Reverse receipt no. 1? A reversing receipt is issued and reported to the tax authority. This cannot be undone. Refund paid by Cash Card Bank transfer Reverse Keep
   390: Reverse receipt no. 3? A reversing receipt is issued and reported to the tax authority. This cannot be undone. Refund paid by Cash Card Bank transfer Reverse Keep
20. **Confirm the reversal.** 1280: the receipt reads: No. 1 · BAM 50.00 23 Nov 2026, 20:58 · Patient · Cash Fiscalized Reversed Open PDF; the list: No. 2 · -BAM 50.00 / No. 1 · BAM 50.00
   390: the receipt reads: No. 3 · BAM 50.00 23 Nov 2026, 20:59 · Patient · Cash Fiscalized Reversed Open PDF; the list: No. 4 · -BAM 50.00 / No. 3 · BAM 50.00 / No. 2 · -BAM 50.00 / No. 1 · BAM 50.00

The receipt PDFs fetched by URL as the receptionist: [1280-receipt.pdf](1280-receipt.pdf), [390-receipt.pdf](390-receipt.pdf).

On the refusals (steps 11 to 16): the web app's pages answer "This page does not exist."; the history answers
`404 not_found` and the conversations `403 forbidden`; the two FHIR searches (QuestionnaireResponse by patient,
Encounter by appointment) answer `200` with an empty searchset, which is how a FHIR search refuses a stranger (the same
answer as for a patient who does not exist). None of the answers holds any of the patient's data.

## Findings

Not fixed in this session.

1. **The desk offers no way to open a receipt's PDF** (step 10, both widths). After "Issue receipt" the desk shows only
   "Receipt no. N was issued."; the row keeps its "Issue receipt" button and has no "Open PDF". The PDF itself is the
   receptionist's to read: by its URL (`/clinic/receipts/{id}/pdf`) it answers 200 with the PDF.
2. **A receipted visit looks unreceipted on the desk** (steps 9 and 10, both widths). After the receipt is issued the row
   is unchanged and still offers "Issue receipt", with nothing to say it has one: a second receptionist, or the same one
   later, can issue a second receipt for the visit.
3. **The admin's visit page shows the raw key `calendar.status.arrived`** for an arrived visit (step 17, both widths). The
   calendar catalogs have no `status.arrived`; `AppointmentDetail.tsx` and `Calendar.tsx` both read `calendar.status.*`
   (the desk has its own `desk.status.arrived`).
4. **The visit page's actions wrap unevenly** (step 17, both widths, worse at 390 px): "Show changes" and "Issue receipt"
   each take a row of their own, then "Write a quote" and "Show quotes" share one; the buttons are of two styles and
   sizes.
5. **The walk-in's search is narrower than the form under it** (steps 2 and 3, both widths): the Name and Birth date
   fields and the Search button stop at about two thirds of the dialog, while the new patient form's fields and its
   Register button run its full width.
6. **The walk-in dialog at desktop width** (step 2, 1280 px): the search fields fill about half of the dialog, leaving its
   right half empty.
7. **At 390 px the reversal's question squeezes its select** (step 19): "Refund paid by" and the select share one line,
   the select narrower than the other controls on the page. It fits and works.

Nothing ran past the screen's edge and no page scrolled sideways at 390 px on any step. Every dialog fit the screen.
The date inputs show the browser's own format (`mm/dd/yyyy` in this en-US Chromium), not the UI language's; that is the
native control's and not new in wave 3.

## Rerun after fixes

Rerun 2026-11-23 on `fix/desk-walk` (the fixes of the seven findings, before their merge), on a fresh stack of its own
(compose project `salus-deskfix`, new volumes), seeded and walked as above: the whole journey at both widths from new
patients (Vedrana Syntheticova at 1280 px, receipt no. 1; Amra Syntheticova at 390 px, receipt no. 3). The screenshots
of steps 2, 9, 10, 17 and 19 above are the rerun's and replace the first walk's; the receipt PDFs are the rerun's too.

| # | Step | 1280 px | 390 px | What happened |
| --- | --- | --- | --- | --- |
| 2 | Open the walk-in and search for the patient | pass | pass | The Name and Birth date fields and the Search button take the dialog's full width, as the new patient form does (findings 5, 6) |
| 9 | Issue the receipt | pass | pass | The row shows "Receipt no. N" with "Open PDF" at once, and its button reads "Issue another receipt" (findings 1, 2) |
| 10 | Open the receipt's PDF | pass | pass | The row's "Open PDF" opens it (200, application/pdf); "Issue another receipt" first says "This visit already has receipt no. N. Issue another receipt?" and shows no lines until confirmed (findings 1, 2) |
| 17 | Sign in as the admin and open the visit | pass | pass | The status reads "Arrived" (finding 3); "Show changes", "Issue another receipt", "Write a quote" and "Show quotes" in one row of one style, two even columns at 390 px (finding 4); "Receipt no. N" with "Open PDF" under them |
| 19 | Reverse it: the question | pass | pass | "Refund paid by" stands above a select of the full width (finding 7) |

The other steps passed again at both widths. Nothing ran past the screen's edge and no page scrolled sideways at
390 px. All seven findings are fixed.
