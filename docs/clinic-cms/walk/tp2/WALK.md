# TP2 walk: treatment plans on the visit page

Walked 2026-11-25 in headless Chromium (`mcr.microsoft.com/playwright:v1.65.0-noble`) against the web app's Vite dev
server on this worktree, at 1280 × 900 and 390 × 844, reduced motion. **The API was a fake** (Playwright
`page.route`): TP1 builds the plan routes in parallel and they are not on this branch. Synthetic data only: the
practitioner Cy Example (`cy@example.test`), the patient Ada Example, a booked plan and a draft of three steps each
(one service with a long name on purpose), one quote, four free slots. The walk script stayed in the session's scratch
folder; it is not a check of the repository.

Every screenshot was also measured: the page's `scrollWidth` against the viewport, and every visible button, input,
select, link, heading, label, list item and paragraph for running past the right edge or being cut by
`overflow: hidden`. The only hit on every screen is the shell's visually hidden "Language" label, which is meant to be.

| Screen | 1280 px | 390 px |
| --- | --- | --- |
| The visit's "Show treatment plans": a booked plan with its days, a draft with its day offsets | [1280](1280-plans.png) | [390](390-plans.png) |
| New treatment plan, two steps filled in | [1280](1280-new-plan.png) | [390](390-new-plan.png) |
| Book the draft: practitioner, first day, a time per step | [1280](1280-book.png) | [390](390-book.png) |
| Cancel a plan: the question and its effect | [1280](1280-cancel-ask.png) | [390](390-cancel-ask.png) |

Found and fixed during the walk: at 390 px the plan dialog ran past the screen's right edge (a fieldset's and a grid
item's minimum width is their content's, and a long service name in a select set it); the dialogs' fields now shrink
(`.care-plan-dialog` in `app.css`). The state tag sat in the row's buttons and was stretched to a button's size at 390
px: it now follows the plan's title. "Cancel plan" was a red button in every row and wrapped at 1280 px: it is an
ordinary button in the row, red only in the question that confirms it.

Left: the cancel question opens under the list, not under the plan it asks about (as the page's other `Confirm`s, at
the end of their section).
