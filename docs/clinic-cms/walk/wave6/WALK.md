# Wave 6 walk, second half

Walked 2026-11-25 in headless Chromium (the e2e suite's Playwright image) against **the real stack**: a fresh e2e compose
project (`salus-w6probe-e2e`, removed afterwards), the web app's Vite dev server, the API, the worker with its jobs, and
ClamAV. At 1280 × 900 and 390 × 844, reduced motion. A Go driver in the e2e package made the rows through the API, as
the journeys do, then ran the walk's spec once as the clinic's admin and once as its practitioner; both stayed in the
session's scratch folder and are not checks of the repository. Synthetic data only: a clinic with its first location and
the "Synthetic annex" (Synthetic annex 2, 88000) with Room 1 and an ultrasound; a service that needs it and a follow-up
service; Bo Syntheticson's in-person hours at the annex in Room 1, Di Syntheticova's at the annex without a room; two
patients' in-person visits; a lab result with three values (a long test name and a text value among them) and a PDF; a
draft and a booked three-step treatment plan.

Every screenshot was measured as BG2's walk did: the page's `scrollWidth` against the viewport, and every visible button,
input, select, link, heading, label, cell and paragraph for running past the right edge or being cut by
`overflow: hidden`. The language select's visually hidden label is reported on every page and is hidden on purpose.

## The wave's screens

| Screen | 1280 px | 390 px | Result |
| --- | --- | --- | --- |
| Locations | [list](1280-locations.png), [rooms and devices](1280-locations-rooms-devices.png) | [list](390-locations.png), [rooms and devices](390-locations-rooms-devices.png) | pass |
| Working hours with location and room (the practitioner) | [1280](1280-hours-location-room.png) | [390](390-hours-location-room.png) | pass |
| Working hours (the admin) | [1280](1280-hours-admin.png) | [390](390-hours-admin.png) | pass |
| The desk, every location (each in-person visit names its location) | [1280](1280-desk-all-locations.png) | [390](390-desk-all-locations.png) | pass |
| The desk, the annex picked | [1280](1280-desk-annex.png) | [390](390-desk-annex.png) | pass |
| The price list and its device picker | [list](1280-price-list.png), [picker](1280-price-list-device-picker.png) | [list](390-price-list.png), [picker](390-price-list-device-picker.png) | pass |
| Lab results on the visit page | [1280](1280-visit-lab-results.png) | [390](390-visit-lab-results.png) | fixed (finding 3) |
| Treatment plans on the visit page | [1280](1280-visit-care-plans.png) | [390](390-visit-care-plans.png) | fixed (finding 2) |
| Laying out a plan | [1280](1280-care-plan-lay-out.png) | [390](390-care-plan-lay-out.png) | pass |
| Booking a plan | [1280](1280-care-plan-book.png) | [390](390-care-plan-book.png) | pass |
| Cancelling a plan (the question) | [1280](1280-care-plan-cancel-ask.png) | [390](390-care-plan-cancel-ask.png) | pass |

## BG2's deferred pass over every staff screen, with real rows

As the admin at both widths: the calendar, working hours, practitioners, questionnaires, waitlist, analytics, receipts,
the receipts report and online practice, quotes, data requests, clinic settings, membership, setup, term review, the
desk, the price list and locations (screenshots `<width>-<screen>.png` in this folder). Nothing scrolled sideways after
the fix below, and no control ran off screen or was clipped. The analytics tables shown with "Show as table" are wider
than a phone and scroll inside their own box, as designed.

## Findings

Fixed:

1. **Analytics at 1280 px scrolled sideways** (to 1322 px): with real terminology the "Diagnosis group" select took the
   width of its longest option and ran out of its 260 px field. It now fills its field and ends in an ellipsis
   (`analytics/analytics.css`). The fake API of earlier walks had short names.
2. **A booked plan's steps gave a date only**: three visits on one day read the same. A booked step now gives its
   visit's day and time ("Thursday, 26 November 2026, 00:30"), a draft its day as before (`carePlan.ts`, test first).
3. **A text value's lab result row at 390 px** printed "Reference range" and "Flag" over nothing. An empty cell of the
   stacked table now prints no label (`app.css`).

Listed, not changed:

4. "Add step" in the plan dialog uses the app's `quiet` button style and reads as plain text at a glance. It is the
   style the app uses for such secondary in-form actions; left to the owner.
5. "Mark fulfilled" before the visit has started is a disabled primary button and still looks filled (existing, not
   wave 6).
6. The booking dialog's date field shows the browser's own date format (an `<input type="date">`), here en-US in the
   headless browser.
7. The desk names a visit's location only for locations in use; a visit at a location taken out of use shows none.
8. The patient app dates a booked step by its visit's day without the time (the web's finding 2 is not mirrored there).
