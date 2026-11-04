# Accounts progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Privacy foundation | todo | |
| 2 Identity: accounts | todo | |
| 3 Identity: sessions and MFA | todo | |
| 4 Directory | todo | |
| 5 Practitioner invites | todo | |
| 6 FHIR reads | todo | |
| 7 Web | todo | |
| 8 Mobile | todo | |
| 9 End-to-end | todo | |

## Decisions

- 2026-11-04, owner (changes on approval, recorded in `SPEC.md` §2 "Changed on approval"):
  - `salus_app` is a separate `LOGIN` role with its own credentials; `api`, `worker`, `notifier` and
    `scheduler` connect as it, only `migrate` and `salusctl` as the owner. Replaces `SET ROLE` on the
    owner's connection.
  - **Changed settled decision:** the Redis session revocation key is dropped. `session.revoked_at` in
    Postgres is the only revocation; Redis keeps the login rate limit and the MFA challenge.
  - `salusctl reset-mfa <account-id>` (audited) added to package 5; self-service recovery stays out of scope.
  - The `SECURITY DEFINER` invite lookup sets `search_path` and is owned by `salus_invite_lookup`, a
    non-superuser role.
  - A retry of `invite_email` after a send that succeeded issues a new token, so the earlier e-mail's link
    stops working; only the newest link is valid. Accepted.
  - The retention job is listed under "Up next" in `SPEC.md`.
  - Retention periods and `TOTP_KEY` accepted for now, marked "needs legal / KMS decision before
    production" in `SPEC.md`.

## Waiting on people

- Before production: a legal decision on the retention periods (`SPEC.md` §4) and a KMS decision for
  the TOTP key (`TOTP_KEY` is local and CI only).
- Branch protection on `main` (carried over from `docs/bootstrap/PROGRESS.md`): needs GitHub Pro.

## Log

### 2026-11-04 — plan

`SPEC.md`, this plan and `RUN.tsv` written.

### 2026-11-04 — plan approved

Owner approved with the changes under Decisions; `SPEC.md` and the plan updated. Committed as
`accounts(0): spec and plan`.
