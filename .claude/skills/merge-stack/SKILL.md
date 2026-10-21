---
name: merge-stack
description: Merge a Salus stacked PR series (stack/prefix/* branches) bottom-up with squash, restack after each merge, and archive the plan's PROGRESS log when the last PR lands.
---

# Merge a stacked PR series

Use when the owner says "merge the stack", "merge appointments", "merge up to 2.2" or similar, for a plan run with
`scripts/run-feature.sh --stacked` (branches `stack/<prefix>/plan`, `stack/<prefix>/<package>`).

The owner relies on CI and Copilot instead of reading every diff, so this skill is the review gate: it never
merges a PR with failing checks or unresolved review comments without the owner saying so for that PR.

## Inputs

- The plan folder, e.g. `docs/appointments`.
- Optional stop point: "up to <package>". Default: every open PR in the stack.

## 0. Prepare

1. Repo root, clean working tree (`git status --porcelain --untracked-files=no` empty); otherwise stop.
2. `gh auth status`, `git fetch origin`.
3. Read `<plan>/RUN.tsv`: `# prefix:`, `# base:` (default `main`), the rows in order. The chain is `plan` followed by
   every package whose fourth column is not `hold`. A package with no `stack/<prefix>/<package>` branch is not
   started: the chain ends before it.
4. For each branch in the chain: `gh pr view <branch> --json number,state,title,url,baseRefName,additions,deletions,changedFiles`.
   Skip merged ones. The lowest open PR must target the base; if not, run `bash scripts/run-feature.sh <plan> --restack`.
5. Show the list to merge (number, title, +/- lines) and get one confirmation, unless the owner's request already
   names the PRs to merge. That covers the run; stop again only for the gates below.

## 1. For each open PR, bottom-up

1. Checks: `gh pr checks <n> --watch --fail-fast`. A failure: stop, show the check and its log link, ask whether to
   fix it on that branch or stop.
2. Review comments: list unresolved threads with `gh api graphql` on
   `pullRequest.reviewThreads { isResolved, comments { author { login } body path line } }`, and whether Copilot
   reviewed (`gh pr view <n> --json reviews`). For each unresolved thread show file, line and a one-line summary and
   ask: fix now on the PR's branch, merge anyway, or stop. A fix is a commit on that branch
   (`<prefix>(<package>): fix review comment`), pushed, checks green again, then the thread resolved. Never resolve or
   dismiss a thread without a fix or the owner's "merge anyway".
3. Digest, always, a few lines: what the PR does (from its body), files and lines changed, and anything risky it
   touches (migrations, auth, consent, audit, patient data, encryption, retention, deletes).
4. Merge: `gh pr merge <n> --squash --delete-branch --subject "<PR title> (#<n>)"`. Keep the title: the runner put a
   Jira key only on the PR that closes that ticket.
5. Restack: `bash scripts/run-feature.sh <plan> --restack`. On a conflict it aborts and prints the `git rebase --onto`
   command: stop and show it; do not resolve conflicts in code the owner has not seen.
6. Next PR, from step 1.

Stop at the requested package, at the first `hold` row, at a package that is not started, or when nothing is open.

## 2. Archive, only once the whole chain is merged

Only when every non-hold package of RUN.tsv has a merged PR. Never after a partial merge: later packages read PROGRESS.

1. `git switch <base> && git pull --ff-only && git switch -c chore/<prefix>-archive`.
2. `git mv <plan>/PROGRESS.md <plan>/archive/PROGRESS_LOG.md`, then a new short `<plan>/PROGRESS.md` with only the
   status table, Decisions, "Waiting on people", any "Carried into" notes, and `Full log: archive/PROGRESS_LOG.md`.
3. `<plan>/README.md` status line: "Done <date>; log archived" (or "Done up to <package>; <rows> on hold until <reason>").
4. Root `CLAUDE.md`: the plan's section becomes one or two lines (what it built, pointer to `<plan>/README.md`); with
   hold rows, one more line on how to resume (`bash scripts/run-feature.sh <plan> --unhold --from <package>`).
5. `make check-docs` passes. Commit `docs(<prefix>): archive progress log`, push, PR to the base, checks green,
   squash-merge. No Jira keys.
6. Delete merged local stack branches.

## 3. Report

Each PR merged (number, title, link), fixes made for review comments, PRs left open and why, the archive PR link if
any, and the Jira keys the merged titles will close.

## Never

- Merge with failing or running checks, or unresolved threads, without the owner's answer for that PR.
- Merge a PR in the middle of the stack into the branch below it; use merge commits or rebase merges.
- Force-push except through `--restack` (it uses a lease). Run `git stash` in this repository.
- Archive while any non-hold package is unmerged or not started.
