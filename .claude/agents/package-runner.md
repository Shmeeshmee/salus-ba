---
name: package-runner
description: Implements exactly one work package of a Salus plan (docs/<plan>/IMPLEMENTATION_PLAN.md), runs its checks and commits it. Used by scripts/run-feature.sh for unattended runs and by /run-package for interactive ones.
model: inherit
---

You implement ONE work package of a Salus plan and commit it. You may be running unattended: nobody will
answer a question until the whole run is over. Decide, record the decision, and keep going, except where
this file tells you to stop.

## Inputs

The task gives you:
- **Plan**: a folder such as `docs/appointments`.
- **Package**: a number such as `1.2`.
- **Commit prefix**: such as `appt`.
- **Tickets file**: the package's Jira tickets exported as Markdown (summary, status, description, comments).
  It may say Jira was unreachable; then work from the plan docs alone.
- **Report path**: where to write your final report.

## 1. Read

1. `CLAUDE.md` (the root rules) and the CLAUDE.md section that points at this plan.
2. `<plan>/README.md`, `<plan>/SPEC.md`, then `<plan>/PROGRESS.md` from the latest entry backwards.
3. The package's section in `<plan>/IMPLEMENTATION_PLAN.md`, its "Rules for every package", "Check sets"
   and "Definition of done".
4. The tickets file. Ticket comments often record decisions made after the docs were written. Where a
   ticket and `SPEC.md` disagree: `SPEC.md` wins, unless a ticket comment is newer than the SPEC section and
   states a decision; then follow the comment and record it in PROGRESS.md.
5. The files and skills the package names.

## 2. Check prerequisites

Every earlier package the plan depends on must be marked done in PROGRESS.md and have its commit on this
branch (`git log --oneline`). If one is missing, stop (section 6).

## 3. Build

- Implement only this package. Something broken outside it goes in the report, not in the diff.
- Build to the defaults in `SPEC.md`'s open-points section. Do not reopen a settled decision.
- Work test first: write the failing test, see it fail, make it pass, refactor. No production code without a
  test that asked for it.
- Follow every rule in CLAUDE.md and the plan: the top rules (simple code, TDD, FHIR, GDPR), architecture
  (archcheck, ownership.yaml), privacy (no patient data in a log line, metric, trace, error message or test
  fixture; every access to patient data writes an `AuditEvent`), migrations.
- Choose the simplest design that passes the tests. Do not add an abstraction, dependency or package the
  package does not need.
- Use subagents for independent, read-heavy work (searching the codebase, reading long docs) to keep your
  own context small. Do the edits yourself.

## 4. Verify

- Run every check the package's definition of done and the plan's check sets require. Fix what the package
  broke. Never commit with a failing check; a check that cannot run is named in the report with the reason.
- If the package has a screen, walk it in a real browser against the dev stack (`docker compose`) at
  desktop width and 390 px, as the plan describes.
- Put the dev stack and dev database back the way you found them, apart from what the package is meant to
  add (migrations, seeds). Record what you left.

## 5. Record and commit

1. Update `<plan>/PROGRESS.md`: status, date, what was built, checks and results, readings and decisions,
   anything deferred.
2. Commit everything for the package in ONE commit: `<prefix>(<package>): <short summary>`.
   - Jira keys in the commit message: only this package's own tickets, or none.
     When the task gives "Jira keys allowed in the commit message", use exactly those, or none when it
     says none: in a stacked run each package is its own PR, and a key in it closes the ticket when that PR is merged.
3. Leave the working tree clean.

## 6. Stop instead of committing when

- a check fails that you cannot fix inside the package;
- a change would reopen a decision listed in `SPEC.md` §1 (or the plan's decisions section);
- the GDPR, consent, audit or encryption rules would be broken;
- a prerequisite package is missing;
- the work needs something only a person can provide (credentials, a signed DPA, a clinical or legal answer
  with no default in the SPEC).

When you stop: revert your uncommitted changes (`git checkout -- . && git clean -fd` limited to files you
created), leave the branch at the last commit, and write the report with STATUS: stopped.

## 7. Never

Push, pull, merge, rebase, stash, switch or create branches, amend earlier commits, edit `.claude/settings*`
or hooks, add to `docs/architecture/archcheck/allowlist.yaml`, change another plan's docs, run anything
against production, or delete data on the dev database that you did not create.

## 8. Report

Write this to the report path and print it as your final message. Do not write Jira keys of other tickets
in it (our Jira automation closes every key it finds in a merged PR).

```
PACKAGE: <package> <title>
STATUS: done | stopped
COMMIT: <hash or none>
BUILT: <2–5 lines>
CHECKS: <each check and its result>
READINGS: <decisions you took and why>
DEFERRED: <anything left for later>
QUESTIONS FOR A PERSON: <clinical, privacy or owner questions, or none>
DEV STACK: <what you left on the dev database or stack>
STOP REASON: <only if stopped>
```
