# RD6 walk: wave 6 leftovers on the web

Walked 2026-11-26 in headless Chromium (the `salus-playwright:1.65.1` image) against the web app's Vite dev server of
this worktree's compose project (`salus-cms-rd6`), at 1280 × 900 and 390 × 844, reduced motion. **The API was a fake**
(Playwright `page.route` on every fetch), as LC2's walk did: the screens' server side is covered by the Go tests of the
package (`internal/schedule`), and a working hours row with a busy room needs a role brought back into use, which no
route makes in one step. Synthetic data only: the admin `admin@example.test`, practitioners Cy Example and Dee Sample,
patients Ana Example, Bo Sample, Cleo Test and Ada Example, the sites "Placeholder main site", "Placeholder annex" (Room
1) and "Placeholder old site" (not in use), the draft "Placeholder knee plan". The walk script stayed in the session's
scratch folder; it is not a check of the repository.

Every screenshot was measured: the page's `scrollWidth` against the viewport, and every visible button, input, select,
link, heading, label, paragraph and list item for running past the right edge. Nothing did at either width.

| Screen | 1280 px | 390 px | Item |
| --- | --- | --- | --- |
| Working hours: Monday's room held by another practitioner then, Tuesday's free | [1280](1280-hours-room-busy.png) | [390](390-hours-room-busy.png) | 1 |
| Front desk, every location: Ana's visit at the old site, "(not in use)" | [1280](1280-desk-location-not-in-use.png) | [390](390-desk-location-not-in-use.png) | 2 |
| Laying out a plan: "Add step" outlined, "Save draft" disabled | [1280](1280-care-plan-add-step.png) | [390](390-care-plan-add-step.png) | 3, 4 |
| Booking a plan: the first day named under the browser's date field, "Book every step" disabled | [1280](1280-care-plan-book-first-day.png) | [390](390-care-plan-book-first-day.png) | 4, 5 |
| A visit before its start: "Mark fulfilled" and "Mark no-show" disabled | [1280](1280-visit-disabled-button.png) | [390](390-visit-disabled-button.png) | 4 |

No finding. The headless browser's date field shows `11/26/2026` (en-US), the hint under it "Thursday, 26 November
2026", as the steps below it name their days.
