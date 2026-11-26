# Readiness walk: the staff header's menu on a phone

Taken 2026-11-26 in headless Chromium (the e2e suite's Playwright image) by `test/e2e/web/a11y.spec.ts` with
`E2E_WALK_DIR` set, on this worktree's own e2e project (`salus-rd-menu-e2e`, removed afterwards), with the rows
`TestAccessibility_EveryStaffScreenAndDialog` makes. Synthetic data only. 1280 × 900 with a mouse; 390 × 844 on a touch
screen (`(pointer: coarse)`), reduced motion. Each role's first screen (the admin's and the receptionist's desk, the
practitioner's calendar).

| Screenshot | What it shows |
| --- | --- |
| [1280, admin](1280-admin.png), [practitioner](1280-practitioner.png), [receptionist](1280-reception.png) | The header as before: the links, the medical terms, the language and sign out in the bar; no menu button |
| [390, admin, closed](390-admin-menu-closed.png), [open](390-admin-menu-open.png) | The brand and the menu button; open, the links one per row, then the medical terms, the language and sign out |
| [390, practitioner, closed](390-practitioner-menu-closed.png), [open](390-practitioner-menu-open.png) | The same for the practitioner's links |
| [390, receptionist, closed](390-reception-menu-closed.png), [open](390-reception-menu-open.png) | The front desk link, the language and sign out |

The closed header on a phone is one row (the brand and "Menu"), where the admin's was about 260 px of wrapped links.

The 390 px screenshots show the menu button with the focus ring: the journey had just closed the menu with Escape (the
focus goes back to the button) and opened it again with a click. The journey itself passed for the three roles
(`TestAccessibility_EveryStaffScreenAndDialog`, 105 s on its own). Above 520 px the header was also compared with the one
before this change in a throwaway Playwright page (the same header markup, the old and the new `app.css`, at 1280, 1024,
800, 600 and 521 px, mouse and touch): the screenshots were byte for byte the same.
