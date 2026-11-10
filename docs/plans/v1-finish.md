# Finishing v1: SAL-10, SAL-12, SAL-13, SAL-14, SAL-17

Epic SAL-1. Status: planned, 2026-11-10. Builds on docs/plans/consult-intelligence.md (its decisions and gates hold
unless this file says otherwise) and docs/terminology/SPEC.md.

## Tickets

| Ticket | Branch | What |
| --- | --- | --- |
| SAL-10 | `sal-10-e2e` | End-to-end tests of the consult pipeline on the fake engines: the call starts, the agent joins, captions and translations appear, the transcript is stored, suggestions are shown, pinned and dismissed, a document is drafted, edited and signed, a consent withdrawal removes the agent mid-call. Scripted fixtures for a German, an Arabic and a Turkish patient with a Bosnian doctor. Latency against the plan's targets (caption under 1 s, translation under 2 s, draft under 15 s), reported per run |
| SAL-12 | `sal-12-consent` | The patient's transcription notice on web and mobile before the call, with decline: the call then runs without the agent. Replaces "The consultation is not recorded" |
| SAL-14 | `sal-14-term-review` | The terminology review panel in `web/`: queue (unreviewed first), filters by language and status, inline edit, approve or needs changes with a note, the reviewer permission, an AuditEvent for every approval and edit, progress per language, keyboard first |
| SAL-17 | `sal-17-patient-docs` | The patient's signed documents per appointment in `mobile/`: open, download and share the PDF; a transcript copy behind a per-clinic setting; a notification when a document is signed. Never a draft; only the encounter's patient |
| SAL-13 | `sal-13-term-hover` | Plain-language explanations of the most common terms in every app language, drafted and stored as unreviewed (so they land in SAL-14's queue); terms tagged in transcript lines, captions and document fields with the glossary's matching; a hover tooltip on web, a tap in SAL-17's documents screen; the viewer's language with English as the fallback; labelled informational, never a diagnosis; a signed document is never altered |

## Waves and dependencies

```
Step 0 (one PR): lint in worktrees, this plan, the review contract (00037)
Wave 1, in parallel:  SAL-10   SAL-12   SAL-14   SAL-17
Wave 2, after SAL-14 and SAL-17 are merged:  SAL-13
```

- **SAL-13 needs SAL-14** (the review queue its drafts land in, and the reviewer's writes) **and SAL-17** (the
  patient's documents screen it adds the tap to). Both need the contract of step 0.
- **SAL-10 and SAL-12 overlap**: SAL-12 changes when the agent joins. SAL-12 merges first; SAL-10 then merges main
  and its journey gives the transcription consent before the call (and a declining patient gets no agent).
- **SAL-12 and SAL-17** both touch the patient's appointment screen in `mobile/`: the second to merge resolves it.
- **Merge order** in wave 1: SAL-14, SAL-17, SAL-12, SAL-10, one at a time, main merged into the next branch first
  (no force-push).

Each package runs in its own git worktree and compose project (`-p salus-sal10`, `salus-sal12`, `salus-sal14`,
`salus-sal17`, then `salus-sal13`) with an override file without published host ports and with its own app image
tag. Migrations: each branch takes the next free number when it merges main before its own merge (a migration not
yet on main may be renumbered; `scripts/check-migrations.sh` guards only main's). Reserved while in flight: SAL-12
`00038`, SAL-14 `00039`, SAL-17 `00040`, SAL-10 `00041` (if it needs one), SAL-13 `00042`.

## Decisions (owner, 2026-11-10)

| Topic | Decision |
| --- | --- |
| Reviewer identity | `reviewed_by` is the reviewer's opaque account id on every term, translation and explanation it reviews, and every edit and status change is also an AuditEvent. This replaces docs/terminology/SPEC.md §3 "No personal data in the file or the tables: no reviewer name, only the date": the terminology tables are now personal data of staff (`term`, `term_translation`, `term_explanation`; ownership.yaml), kept with the term while the account exists (retention `keep`), the reference cleared with the account's erasure (subject-rights feature). The curator's CSV still names no one |
| Reviewer permission | A platform-level permission on a staff account, granted and revoked only by the operator (`salusctl`), never by a clinic admin: the term list is shared by every clinic, so an edit reaches all of them. Only a reviewer edits, approves or marks needs changes; every staff member reads. This replaces SPEC §3 "Read-only for the app ... no curator role in the app (yet)": the app gets writes on the review's tables for reviewers only (SAL-14); the curator's CSV round trip through `salusctl` stays |
| Transcription consent (SAL-12) | A separate, granular, revocable FHIR `Consent` for transcription, translation and AI assistance, per patient and clinic, asked before the call (web and mobile). The agent joins only while both the care Consent and this one are active; declining or withdrawing keeps the call running without the agent (removed at once, as SAL-3's revocation path). This extends the interim decision of 2026-11-09 (plan, "Gates"), still pending the lawyer's Q16 (docs/legal/BRIEF.md). The wording is drafted in every app language and waits for the owner's and the doctor partner's approval |
| Explanation drafts (SAL-13) | Drafted by Claude in the SAL-13 session into a committed seed file, imported with `salusctl` as `unreviewed`. No language model is called at runtime, no processor is added: the input is the term list (reference data), nothing about a patient |

## Contract (step 0, migration 00037)

- **One review status**, the Postgres domain `review_status`: `unreviewed`, `approved`, `needs_changes`. It replaces
  `term.review_status`'s `draft` and `reviewed` (`draft` became `unreviewed`, `reviewed` became `approved`), so the
  cheatsheet's "Not reviewed yet" mark (now on anything not approved), the curator's CSV, `salusctl terminology
  export -status` and the review panel read one status. Go: `terminology.ReviewUnreviewed`, `ReviewApproved`,
  `ReviewNeedsChanges`; OpenAPI: `ReviewStatus` (a contract test keeps the two equal).
- **Review metadata** on `term`, `term_translation` and `term_explanation`: `review_status`, `reviewed_by` (an
  account), `reviewed_at` (set with every status but `unreviewed`, kept when an entry goes back to unreviewed) and
  `note`. Go: `terminology.Review`; OpenAPI: `Review`.
- **`term_translation`**: the review of a term's labels in one language, primary key (`term_id`, `language`),
  `version_id` for `If-Match`. A term and language with labels and no row is unreviewed. Filled by 00037 from each
  term's status for every language it has labels in. Go: `terminology.Translation`; OpenAPI: `TermTranslation`.
- **`term_explanation`**: a plain-language explanation per term and app language (never Latin), trimmed, 1 to 500
  characters, `version_id`. Go: `terminology.Explanation`; OpenAPI: `TermExplanation`.
- **Rules for SAL-14's service** (with its unit tests): editing an approved entry (a label of a translation, an
  explanation's text) sets it back to `unreviewed`; approving or marking needs changes sets `reviewed_by` and
  `reviewed_at`; needs changes takes a note. The curator's import that changes a term's status clears
  `reviewed_by` (the file names no one) and sets `reviewed_at` for `approved` and `needs_changes`.
- **Grants**: `salus_app` reads `term_translation` and `term_explanation`; SAL-14 adds the reviewer's writes.

## Checks per PR

`make lint`, `make test`, `make archcheck`, `make cover`, `make web-typecheck`, `make mobile-analyze`, `make e2e`, plus
the web and mobile tests of what changed. GitHub Copilot reviews through GraphQL; CI and Copilot clean before the
merge. `make lint` now works in a worktree: compose mounts the git common directory at its host path (make exports
`GIT_COMMON_DIR`).

## Copy waiting for the owner

Every new user-facing string goes to the owner first (CLAUDE.md "Copy"). Listed per PR in its description and in
PROGRESS when merged: SAL-12's notice (all ten languages, `legal`), SAL-14's panel (five staff languages), SAL-17's
documents screen and e-mail (ten), SAL-13's tooltip label (ten) and the explanations themselves (the doctor partner
reviews them in SAL-14's panel).
