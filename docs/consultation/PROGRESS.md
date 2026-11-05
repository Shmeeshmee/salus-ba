# Video consultations progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 videosession | not started | |
| 2 encounter | not started | |
| 3 Webhooks | not started | |
| 4 API | not started | |
| 5 Revocation and cancellation during a call | not started | |
| 6 Web | held: copy review | |
| 7 Mobile | held: copy review | |
| 8 End-to-end | not started | |

## Decisions

- 2026-11-05, owner (on approval, recorded in `SPEC.md` §2 "Changed on approval"):
  - A patient "the practitioner did not join" ended state; `video_session.patient_joined_at` kept and shown to
    the clinic through `GET /appointments/{id}/consultation`.
  - A hard cap 2 hours after the scheduled end (`time_limit`), tested.
  - A separate e2e LiveKit config is acceptable if one config cannot serve both; stop only if that fails too.
  - Decisions 2 (consent for the practitioner too) and 3 (removal through River jobs) and everything else
    approved as written.

## Waiting on people

- The owner, before package 6: review of every new web and mobile string, sent on its own. Packages 6 and 7
  are marked `hold` in `RUN.tsv` until the answer is recorded here.
- Before production: `SPEC.md` §8 (LiveKit domain, TLS, TURN, keys, node IP; retention and lawful basis;
  localisation).

## Log

### 2026-11-05 — plan

`SPEC.md`, this plan and `RUN.tsv` written from the owner's brief, ADR 0006 and the booking feature. 

### 2026-11-05 — plan approved

Owner approved with the changes under Decisions; `SPEC.md` and the plan updated. Committed as
`consultation(0): spec and plan`.
