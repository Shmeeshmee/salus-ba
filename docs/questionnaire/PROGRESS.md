# Questionnaires progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 questionnaire | not started | |
| 2 questionnaire API | not started | |
| 3 questionnaireresponse and the access rule | not started | |
| 4 Answers | not started | |
| 5 Booking and cancellation | not started | |
| 6 Responses API | not started | |
| 7 E-mail mention | not started | |
| 8 Web: library and builder | held: W41 and W42 not approved | |
| 9 Web: appointment and working hours | not started | |
| 10 Mobile | not started | |
| 11 End-to-end | not started | |

## Decisions

- 2026-11-06, owner (recorded in `SPEC.md` §2 and §3): D1 uploads in a separate feature, "questionnaire
  attachments", right after this one, with `attachment` reserved in the model; D7 `@dnd-kit`, pinned, plus
  Move up and Move down; D2 to D6 and D8 as recommended; D9 the appointment's practitioner keeps reading the
  submitted answers of past appointments after a revocation, nobody else. `COPY.md` approved with W11, W35 and
  M13 changed.
- 2026-11-06, plan: `@dnd-kit` speaks to screen readers during a drag, and the approved text has nothing for
  it; W41 and W42 are proposed in `COPY.md`, and package 8 is held until the owner approves them.
