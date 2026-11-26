# RD3 walk: accessibility and keyboard

Walked 2026-11-26 in headless Chromium (the e2e suite's Playwright image) against the real stack of this worktree's
e2e project (`salus-cms-rd3-e2e`, removed afterwards), with the rows `TestAccessibility_EveryStaffScreenAndDialog` makes:
a clinic with in-person hours, a second location with a room and a device, a service, a receptionist, the desk's day
with an arrived walk-in and her receipt and a booked visit, a patient of the app with a visit, a lab result, a draft
treatment plan and a conversation, and a data request. Synthetic data only. At 1280 × 900 (mouse) and 390 × 844 (touch:
`hasTouch`, `isMobile`, so `(pointer: coarse)` matches), reduced motion.

The journey (`test/e2e/web/a11y.spec.ts`) is the check; these screenshots show what its fixes look like:

| Screenshot | What it shows |
| --- | --- |
| [1280, the desk, focus on the fourth stop](1280-admin-desk-focus.png) | The focus ring in the brand green (was mint, 1.9:1) |
| [1280, term review, focus](1280-admin-term-review-focus.png) | The same ring on a page-coloured background |
| [1280, the visit page, focus](1280-practitioner-the-visit-focus.png) | The ring in the header |
| [390, the desk](390-admin-desk.png), [as the receptionist](390-reception-desk.png) | Header links, the language select, the row's buttons and the patient's name at 44 px |
| [390, clinic settings](390-admin-clinic-settings.png) | The settings links and every checkbox label at 44 px |
| [390, practitioners](390-admin-practitioners.png) | The role radios' labels at 44 px |
| [390, receipts](390-admin-receipts.png) | The toolbar's links at 44 px |
| [390, a conversation](390-practitioner-messages.png) | The attach button at 44 px |

Nothing ran past the right edge at 390 px in the screenshots; the header is taller on a phone (each link 44 px) and
the rows of the desk are a little looser.
