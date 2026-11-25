# Wave 5 walk

Walked 2026-11-24 in the worktree `salus-cms-w5` (detached at `feat/cms` 34f230ff, "cms(wave 5): the review's fixes",
with the walk's own fixes copied in and then committed with the integration). Own compose project `salus-cms-w5`
(ports 20400–20407) on fresh volumes, started as `make e2e` starts it, with headless Chromium from the `playwright`
service and the web app's Vite dev server.

**The driver is committed**: `test/e2e/web/privacy.spec.ts`, run by `TestPrivacy_TheAdminHandlesARestrictionInTheWebApp`
with `E2E_WALK_DIR` set, which keeps the screenshots below. **Seeded through the API** (synthetic data only): a clinic made
with `salusctl create-clinic` and listed, one practitioner with hours every day, a second admin invited and enrolled with
TOTP (who walks), and a patient the clinic registered at the desk (Rada Ogranicena, 9 August 1975, in-person consent),
whose restriction request the desk filed for them (S10). Both widths walk the same seed; the journey lifts the
restriction at the end so the second width starts from the same state.

## Steps

| # | Step | Expected | 1280 px | 390 px | Screenshots |
| --- | --- | --- | --- | --- | --- |
| 1 | Front desk, "Patients" search (FX) for the family name, open the card | The patient is found; the card opens | pass | pass | [1280](1280-01-desk-card-before.png), [390](390-01-desk-card-before.png) |
| 2 | The admin's queue (`/data-requests`) | The request with its kind, due date in the clinic's zone, what it needs, its state, "Filed at the desk" | pass (finding 3) | pass (finding 3) | [1280](1280-02-queue.png), [390](390-02-queue.png) |
| 3 | Open the request | Kind, state, due and received dates, what it needs, answer, extend, hand-over, restrict, lift, history | pass after the fix (findings 1, 3, 4) | pass after the fix (findings 1, 3, 4) | [1280](1280-03-detail.png), [390](390-03-detail.png) |
| 4 | "Restrict the record" | "The record is restricted." | pass | pass | [1280](1280-04-restricted.png), [390](390-04-restricted.png) |
| 5 | Front desk, "Patients" search again | "No patient found.": the restricted patient is left out (S7) | pass (findings 2, 5) | pass (findings 2, 5) | [1280](1280-05-desk-search-after.png), [390](390-05-desk-search-after.png) |
| 6 | Back on the request, "Lift the restriction" | "The restriction is lifted." | pass | pass | [1280](1280-06-lifted.png), [390](390-06-lifted.png) |

Nothing is wider than the screen at either width (the journey checks `scrollWidth` at each step). The "Your clinic's
setup is not finished" banner comes from the seed (the clinic skipped the wizard), not from wave 5.

## Findings

| # | Finding | Status |
| --- | --- | --- |
| 1 | The detail's actions ("Send the answer", "Extend by two months", "Restrict the record", "Lift the restriction") were bare text with no button style, easy to miss as labels | **Fixed**: the rows take the shared `actions` style (bordered buttons) and "Send the answer" is the primary action |
| 2 | The 390 px "after" screenshot was taken while the search was still running, so the journey's "patient left out" check passed before the answer came | **Fixed** in the journey: it waits for "No patient found." |
| 3 | The queue and the detail never name the patient: the admin sees "Restriction · Due …" and a patient id behind the hand-over link, and cannot tell whose request it is without opening the hand-over (a registered patient's only) | **Open**: needs the patient's name on the clinic's request (a contract change, the read audited); for the owner |
| 4 | "Restrict the record" and "Lift the restriction" both show whatever the record's state; the detail does not say whether the record is restricted now | **Open**: the contract has no read of the restriction; small follow-up |
| 5 | Once restricted, the patient leaves the desk's search for everyone, the admin included, with no hint; the admin reaches the record only through the request's hand-over | **Open, by design** (S7: hidden from everyday screens); the owner to confirm, with finding 3 |
| 6 | An app patient's name at the desk is text, not a card (R2), so a restriction of an app patient's record cannot be walked from the card; it is covered by `TestPrivacy_ARestrictionHidesThePatientFromThePractitionerAndTheDesk` through the API | Noted |
