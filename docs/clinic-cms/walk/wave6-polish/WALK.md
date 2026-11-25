# Wave 6 web polish walk (BG2)

Walked 2026-11-25 in headless Chromium (the `salus-playwright:1.65.1` image) against the web app's Vite dev server on
this worktree, at 1280 × 900 and 390 × 844, reduced motion. **The API was a fake** (Playwright `page.route`): BG1's
invite withdrawal route is built in parallel and not on this branch, and the run needed no seeded stack. Synthetic data
only: the admin `admin@example.test`, practitioners Cy Example and Dee Sample, patients Ana Example (+38761000001) and
Ida-Marija Importović-Syntheticova (+38761000004, a long name on purpose), invites `dana.a.very.long.address@example.test`
and `reception@example.test`, three membership invoices. The walk script stayed in the session's scratch folder; it is
not a check of the repository.

Every screenshot was also measured: the page's `scrollWidth` against the viewport, and every visible button, input,
select, link, heading, label, cell and paragraph for running past the right edge or being cut by `overflow: hidden`.

## The listed items

| # | Item (WAVE6.md, BG2) | Before | After | 1280 px | 390 px |
| --- | --- | --- | --- | --- | --- |
| 1 | Nested dialogs where Escape closes both | Escape on the patient card opened from the desk's patient search closed the card and the search | `Dialog` stops the key: one Escape closes the card only, a second the search | pass | pass |
| 2 | The patient search's name button at 390 px | The name sat centred above the birth date (also at 1280 px); the search's fields stopped at two thirds of the dialog | The name on the left as on the desk's rows; the search fills the dialog as the walk-in's does | [1280](1280-patient-search.png) | [390](390-patient-search.png) |
| 3 | The membership invoices table at 390 px | Already one card per invoice since package D's walk fix: no sideways scroll, each value under its column's name | unchanged | [1280](1280-membership.png) | [390](390-membership.png) |
| 4 | The walk-in dialog's width at desktop | The walk-in's search already fills its 520 px dialog (wave 3 rerun); the patient search, a dialog of the same classes, did not | Both desk dialogs share one rule (item 2) | [1280](1280-walk-in.png) | [390](390-walk-in.png) |
| 5 | The phone in the grouped form on every screen | The desk's rows, patient search and card grouped it; the visit page's "Patient details" showed it raw | `+387 60 000 001` there too. No other staff screen shows a patient's phone | (test) | (test) |
| 6 | The lint warning in `analytics/ClinicView.tsx` | `react(only-export-components)` for `diagnosisBar` | Moved to `analytics/drill.ts` (pure, beside the drill-down it serves); oxlint 0 warnings | n/a | n/a |
| 7 | "Withdraw" on an open invite | none | Each open invite has "Withdraw" (secondary style, named "Withdraw the invite for …"); it asks first, then `DELETE /invites/{id}`; the invite leaves the list with "The invite was withdrawn." A 404 also takes it off with "This invite is no longer open …"; any other failure keeps it with an alert | [list](1280-practitioners.png), [ask](1280-withdraw-ask.png), [done](1280-withdrawn.png) | [list](390-practitioners.png), [ask](390-withdraw-ask.png), [done](390-withdrawn.png) |

Desk at both widths after the fixes: [1280](1280-desk.png), [390](390-desk.png); the card over the search:
[1280](1280-patient-card-over-search.png), [390](390-patient-card-over-search.png).

## The pass over every staff screen

Walked as the admin at both widths: the desk, calendar, working hours, practitioners, questionnaires, waitlist, analytics,
receipts, the receipts report and online practice, quotes, data requests, clinic settings, membership, price list, setup
and term review. Nothing scrolled sideways and no control ran off screen or was clipped at either width (the only hit
was the language select's visually hidden label, which is hidden on purpose).

Found and fixed:

- **"Import patients" in clinic settings looked like plain text** (both widths): a bare `<button>` outside `.actions`,
  as wave 4's finding 4 was. It now has the page's secondary button style.
- **A row of red "Withdraw" buttons** (first version of item 7): one filled danger button per invite was the loudest thing
  on the page; the row's button is now the secondary style and only the question's confirm is red, as elsewhere.

Limits of this pass: the fake answered only the routes the listed items need, so most other screens showed their
empty or "could not be loaded" state, not a full list. Their layout with real rows (long receipt lists, the analytics
charts with data) was not seen here; the integration's walk on a seeded stack should look at them.
