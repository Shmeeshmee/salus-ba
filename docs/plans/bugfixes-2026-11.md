# Bug fixes, November 2026

Status: planned, 2026-11-10. Three independent fixes, one PR each. SAL-16 merges before SAL-15.

## 0. The stack survives a host reboot (infra, branch `infra-restart-policy`)

After a reboot the app services (`restart: unless-stopped`) came back and crash-looped, while Postgres, Redis,
LiveKit and Mailpit, which had no restart policy, stayed down.

- `deployments/docker-compose.yml`: `restart: unless-stopped` on `postgres`, `redis`, `livekit` and `mailpit`.
  `objectstore`, `clamd`, `landing` and the app services already had it. `migrate` stays `"no"` (one-shot); the
  profile toolchains (`tools`, `node`, `flutter`, `playwright`, ...) are `run --rm` and get none.
- Docker ignores `depends_on` when it restarts containers after a reboot, so the app services can fail a few times
  until Postgres is healthy; with a policy on every dependency they now settle instead of staying down.
- Check: `docker compose config` resolves the policy on every long-running service.

## 1. SAL-16: `salusctl document-templates seed` seeds only the first clinic (branch `sal-16-seed`)

`salusctl` connects as the owner, which bypasses row-level security, so the "already seeded" check sees another
clinic's template of the same title and skips every clinic after the first.

- Test first (`cmd/salusctl`, against `pgtest`): seeding two clinics gives both their templates; re-running is a
  no-op per clinic.
- Fix: scope the existence check (and any other read) to the target clinic explicitly, or run the seed under the
  clinic's role so RLS applies.
- Review every other `salusctl` command that runs as the owner for the same cross-clinic read or write; fix each
  one found, test first, and list them in the PR.

## 2. SAL-15: Bosnian dates in Chromium (branch `sal-15-bs-dates`)

Chromium ships no Bosnian locale data, so `Intl` with `bs` renders "2026 M11 10, Tue".

- One shared formatter in `web/src` that every date and time display goes through (calendar, appointments,
  transcript timestamps, documents, and any other found). For `bs` it uses locale data that every browser has
  (Croatian or Serbian Latin) or our own Bosnian month and day names, whichever is simpler and correct.
  `hr`, `sr-Latn`, `sr-Cyrl` and `en` stay as they are; 24-hour times and Latin digits (CLAUDE.md "Languages").
- Unit tests (Vitest) for the formatter in all five staff languages; an e2e check in Chromium with Bosnian
  selected (`test/e2e/web`).
- The Flutter app (`intl`) is checked for the same gap: `intl` ships its own locale data rather than the
  browser's, so it is fixed only if a
  test shows the same problem.

## Running in parallel

SAL-15 and SAL-16 run at the same time, each in its own git worktree and compose project (`-p salus-sal15`,
`-p salus-sal16`), with an override file that removes the published host ports (LiveKit's 7881 and 7882/udp are
fixed) and gives the app image a per-project tag, so the two stacks neither collide nor run each other's build.
