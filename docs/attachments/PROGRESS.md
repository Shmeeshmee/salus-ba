# Questionnaire attachments progress

| Package | Status | Commit |
| --- | --- | --- |
| 0.1 Baseline | not started | |
| 1.1 pkg/envelope | not started | |
| 1.2 pkg/filetype | not started | |
| 1.3 pkg/clamd | not started | |
| 1.4 internal/binary: the object store | not started | |
| 1.5 internal/questionnaire: attachment items | not started | |
| 1.6 Web: builder options and the file viewer | not started | |
| 1.7 Mobile: pickers, conversion and upload | not started | |
| 2.1 internal/binary: service, Postgres, handlers and wiring | not started | |
| 3.1 Scan, discard and purge | not started | |
| 3.2 internal/questionnaireresponse: attachment answers | not started | |
| 4.1 End-to-end and privacy | not started | |

In a wave, packages write only under their own heading below (package 0.1 adds one per package); this table is
updated in the wave's integration step (`IMPLEMENTATION_PLAN.md` "Running a wave").

## Decisions

- 2026-11-07, owner (recorded in `SPEC.md` §2): photos, documents and short videos attached to questionnaire
  answers in the mobile app and viewed in the web app, the reserved `attachment` type and `valueAttachment`
  enabled; bytes in an S3-compatible object store (a pinned compose container locally, the hosting provider's EU
  store in production, not chosen); no MinIO image, ADR 0010 picks the local store (SeaweedFS and Garage compared
  at least) and the Go S3 client; a new domain `binary` (FHIR `Binary`) in `clinical`, owning the store, the
  `binary` table and the files' lifecycle.
- 2026-11-07, plan: the open points O1 to O12 (`SPEC.md` §3) are built to their defaults until the owner
  answers them. O8 (compute the R4 SHA-1 `hash`), O9 (submit refused while a file is pending), O10 (Garage and
  `aws-sdk-go-v2`, checked in 0.1), O11 (keep the file name as `title`) and O12 (no viewer of one's own files on
  mobile) were added by the plan to the owner's list.
- 2026-11-07, plan: `questionnaireresponse` → `binary`, not the reverse; `binary.Holder` is implemented by
  `questionnaireresponse`, as `retention.Purger` is by the domains, and `questionnaireresponse` reaches files only
  through the `binary_discard` job (`SPEC.md` §2 "Dependency direction"). Attachment answers live in their own
  column, outside `version_id`, so an upload never makes the patient's next save stale.
- 2026-11-07, plan: `COPY.md` is a draft. `RUN.tsv` holds 1.6 until the owner approves it; 1.7 needs the same
  approval (its permission text replaces the two iOS strings of the consultation feature) and does not start
  before it either.
- 2026-11-07, owner: the plan's changes to the brief (the `Holder` port, the separate `attachments` column,
  `DELETE /binaries/{id}`, the `keep` → `delete` retention rule, the route deadlines) approved; O1 to O12 approved
  as defaulted; `COPY.md` approved as written, the hold on 1.6 removed and 1.7 cleared to run. EXIF: the app
  strips location and all other metadata from photos and videos before upload and keeps the orientation, tested
  in 1.7 (SPEC §2 "Image and video metadata", O13). Production placement of ClamAV stays in SPEC §8. Folded in:
  1.6 the duplicate-key warning in `Availability.tsx` and the spacing under "Edit as new version" and "Retire";
  1.7 `compileSdk 37`, the questionnaire "Saved" state in the footer (background, top border, always visible) and
  no slot offered that starts within 5 minutes; 4.1 a CI job building the debug APK. The run goes on unattended:
  the session decides, records and continues (below).

## Decided while the owner was away

Each decision the session took during the unattended run, with its reason; the ones that touch the SPEC are also
rows of SPEC §3 marked "decided by the session, owner to review".

- 2026-11-07, session: the main checkout holds the owner's uncommitted `CLAUDE.md` edit, which would block
  merging package 0.1 (it edits `CLAUDE.md`). The main checkout was switched back to `main` (the edit travels with
  it, untouched; it was made on `main`), and every integration step runs in a worktree of `feat/attachments` at
  `../salus-attachments`, which also starts the shared stack. Wave packages run `docker compose … run --rm
  --no-deps tools …`, so that a run from one worktree never recreates a shared service whose bind-mounted config
  path differs between worktrees.
