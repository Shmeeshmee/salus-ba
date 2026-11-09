# Consultation intelligence: progress

The running record of epic SAL-1 (`docs/plans/consult-intelligence.md`), one section per ticket. Each ticket's
branch writes only its own section.

## SAL-4

Recorded by its own branch.

## SAL-6

Recorded by its own branch.

## SAL-9: the practitioner's call page (captions, transcript, suggestions, documents)

### 2026-11-09: built against the mocked event stream

What was built (web only; no Go, API or migration change):

- `web/src/consult.ts`: the two seams and the state. `LiveSource.subscribe(appointmentId, onMessage)` delivers the
  P0 envelopes (`ConsultMessage`: `transcript`, `suggestion`, `draft`, version 1, the generated types of
  `api/openapi.yaml`); `ConsultActions` holds `setSuggestionStatus`, `templates`, `generate` and `sign`.
  `decodeMessage` reads one frame and drops anything that is not a known type at version 1. `apply` folds a message
  into the state idempotently (segments by `segment_id`, ordered by `ts_start`; suggestions by `id`; drafts by
  `draft_id`), so a replay changes nothing. `useConsult` subscribes while the call is on (joined or reconnecting).
- `web/src/test/fakeConsult.ts`: the in-memory mock of both seams, and builders of the three envelopes with
  placeholder text.
- `web/src/ConsultPanels.tsx`: the captions, and the panel beside the video with three tabs (Transcript,
  Suggestions with the number not dismissed, Documents). `web/src/Documents.tsx`: template → Generate → review
  (sources, missing-field flags) → edit → Sign, and Regenerate.
- `web/src/CallScreen.tsx`: an optional `consult` prop (`{ live, actions }`); the captions go into the stage, the
  panel beside it.
- Copy: `docs/consult-intelligence/COPY.md` (W1 to W23), the English in `web/src/i18n/consultation/en.json` under
  `consultation.live.*`, drafted in bs, hr, sr-Latn and sr-Cyrl (`docs/i18n/TRANSLATIONS.md`).
- Tests: `consult.test.ts` (14), `ConsultPanels.test.tsx` (19), `Documents.test.tsx` (14), and 3 new in
  `CallScreen.test.tsx`. The layout was also checked in a browser (Playwright, 1280 and 820 px wide, a throwaway
  harness on the fakes, not committed): at 1280 px the panel sits to the right of the video, at 820 px below it,
  and the captions keep to the video's lower band, left of the own camera.

No performance claim is made, so nothing was measured.

### Decisions (2026-11-09, agent decision, for the owner to review)

- **"Nothing covers the video".** The panel is beside the video (below it under 960 px), never over it. The
  captions are the one live element on the video: a band along its lower edge, at most two lines (the latest two
  of a long sentence), 12 px from the edges and clear of the practitioner's own camera tile in the corner, so the
  patient's face stays clear. The tooltip with the original appears just above the band only while the
  practitioner hovers or focuses the caption.
- **What is captioned.** Only the patient's latest segment, in the practitioner's language (the translation; the
  original until a translation arrives), and only while the patient is in the room. The practitioner's own words
  are not captioned on their side. A final caption goes after 6 seconds without a new one; a partial one stays
  until it is revised.
- **Keyboard and screen readers.** The caption is focusable; its original is the tooltip, shown on focus as on
  hover, and is the caption's accessible description. The caption band is not a live region (a revised partial
  would be read out again and again); the transcript is the full text, in reading order.
- **One panel, three tabs**, rather than three columns: the practitioner sees one thing at a time (CLAUDE.md
  "UI direction"). A source link (W12) switches to the transcript, highlights the cited lines and brings the first
  into view (smoothly, or at once under `prefers-reduced-motion`). Every tab stays mounted, so the transcript keeps
  its scroll.
- **The transcript** shows partial lines greyed and revised in place; each line's text in the practitioner's
  language comes first (the translation of a patient line, the original of a practitioner line), the other below.
  Times are the practitioner's browser time zone, hours and minutes. It follows the latest line while the
  practitioner is within 24 px of the bottom; otherwise it does not move and offers "Show the latest".
- **Suggestions**: pinned first, then the newest first; a dismissed one leaves the list, with no undo; the
  confidence is not shown. A pin or a dismissal shows at once and is put back, with W14, if the action fails. The
  line that a suggestion is not a diagnosis (W9) is always at the top and is tagged `legal`.
- **Documents**: edits stay on the page until Sign, which sends every field's value; a field the practitioner has
  typed in loses its source link and its flag. Sign always asks to confirm (and says how many fields are empty);
  Regenerate asks only when it would replace edits. A draft `entered-in-error` is not shown.
- **No live source wired (the real app today)**: no panel and no captions at all; the call page is as before.
  `web/src/App.tsx` does not pass `consult` yet.
- **The strings** live in the existing `consultation` catalog area (keys `live.*`), not a new area, so the i18n
  plan's list of areas is unchanged.

### What the real backend must provide (for SAL-6, SAL-8 and SAL-10)

Wiring is `App.tsx` passing `consult={{ live, actions }}` to `CallScreen`, with `live` and `actions` built on the
endpoints below, each built once (not per render: the page resubscribes when `live` changes and reloads the
templates when `actions` changes). Nothing in `api/openapi.yaml` was added for them; SAL-6 and SAL-8 own those contracts.

- **Live source (SAL-6).** Keyed by the appointment id (the page knows the appointment, not the encounter); the
  practitioner of that appointment only. Each frame is one envelope as text, the stream entry's bytes unchanged;
  the page decodes it with `decodeMessage`. On every (re)connect it should send the consultation's messages from
  the start (or at least every final segment, the latest status of every suggestion and the latest version of
  every draft) before the live ones: the transcript, the source links and the drafts are only as complete as what
  the page has received. Duplicates are harmless. A suggestion's status change and every draft change are pushed
  as events too (the page also applies its own changes at once).
- **`setSuggestionStatus(id, status)`** (SAL-7/SAL-6): sets `shown`, `pinned` or `dismissed`; any 2xx is enough,
  a failure is shown as "could not be updated".
- **`templates()`** (SAL-8): the clinic's templates, each `id`, `title` and `fields` (`key`, `label`) in order.
- **`generate(appointmentId, templateId)`** (SAL-8): starts drafting (or redrafting) that template from the
  transcript and answers at once; the draft arrives on the live source as `draft` events. A redraft may reuse the
  draft's id or create a new one; the page shows the latest draft of the template.
- **`sign(draftId, fields)`** (SAL-8): takes `[{key, value}]` for every field as the practitioner left them, saves
  them (a value that differs from the drafted one has no sources any more), sets the status `final`, and answers
  with the signed `DraftEvent`. A draft that is not `preliminary` is refused.
- Every one of these reads or writes patient data, so each writes its `AuditEvent` server side. The page keeps
  the transcript, suggestions and drafts in memory only (no browser storage) and logs nothing.

### Left for later

- **Follow-up ticket to create:** the patient's consent notice for transcription at the start of the call (web and
  mobile), out of SAL-9's scope by the owner's decision of 2026-11-09.
- Native-speaker review of the drafted bs, hr and sr strings, and the lawyer's review of W9 in each language.
- An undo for a dismissed suggestion, and showing a suggestion's confidence, if the owner wants either.
