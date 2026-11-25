# Wave 6 locations walk (LC2)

Walked 2026-11-25 in headless Chromium (the `salus-playwright:1.65.1` image) against the web app's Vite dev server on
this worktree, at 1280 × 900 and 390 × 844, reduced motion. **The API was a fake** (Playwright `page.route` on every
fetch): the routes LC2 builds against are LC1's and RM1's, built in parallel and not on this branch. Synthetic data
only: the admin `admin@example.test`, practitioners Cy Example and Dee Sample, patients Ana Example, Bo Sample and Cleo
Test, the sites "Placeholder main site", "Placeholder annex with a rather long name for the narrow screen" (long on
purpose) and "Placeholder old site" (not in use), rooms and a device of the main site. The walk script stayed in the
session's scratch folder; it is not a check of the repository.

Every screenshot was measured: the page's `scrollWidth` against the viewport, and every visible button, input, select,
link, heading, label and paragraph for running past the right edge.

| Screen | 1280 px | 390 px |
| --- | --- | --- |
| Clinic settings: the links to the price list and the locations | [1280](1280-clinic-links.png) | [390](390-clinic-links.png) |
| Locations: three sites, one not in use | [1280](1280-locations.png) | [390](390-locations.png) |
| A site's rooms and devices opened (a room not in use) | [1280](1280-locations-rooms.png) | [390](390-locations-rooms.png) |
| Adding a room | [1280](1280-locations-add-room.png) | [390](390-locations-add-room.png) |
| Editing a site (phone shown in groups) | [1280](1280-locations-edit.png) | [390](390-locations-edit.png) |
| Working hours: two in-person rows (one with a room) and a video row | [1280](1280-availability.png) | [390](390-availability.png) |
| Front desk, every location | [1280](1280-desk.png) | [390](390-desk.png) |
| Front desk, the annex picked: its visit and the video visits | [1280](1280-desk-annex.png) | [390](390-desk-annex.png) |

## Found and fixed during the walk

- **The desk scrolled sideways at 390 px** (scrollWidth 556): the location select took the width of its longest
  option. It now fills the toolbar's line on a phone and is cut with an ellipsis.
- **The working hours row at 1280 px broke in odd places**: the location select, sized to the longest site name, pushed
  the times onto a line of their own. The location and room now sit on a line of their own under the row's day, type
  and times, each select at most 320 px wide; on a phone they stack like the row's other selects.
- **"Not in use" stretched to the buttons' height** beside them: the tag is centred now.

Nothing else ran off screen or was clipped at either width.
