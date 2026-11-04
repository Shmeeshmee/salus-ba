# Accounts implementation plan

Read `SPEC.md` first. Packages run in order; each is one commit `accounts(<n>): <summary>` on
`feat/accounts`, made with `/run-package docs/accounts <n>`.

## Rules for every package

- Test first: write the failing test, run it and record the failure in `PROGRESS.md`, then the least code to
  pass, then refactor. Service tests with hand-written fakes and `clock.Fixed`; repository tests on
  `pgtest.NewDatabase`; Redis code on `redistest`; handlers with `httptest`, FHIR handlers against golden
  files.
- Every new domain is added to `docs/architecture/archcheck/config.yaml`, every new table to `ownership.yaml`
  (with `personal_data` from package 1 on) and a retention rule, and every domain with queries to
  `sqlc.yaml`, in the commit that adds it.
- Every migration is a new file; grants to `salus_app` sit in the migration that creates the table, and every
  clinic-scoped table gets `ENABLE ROW LEVEL SECURITY` and its policy there too.
- A change to `api/openapi.yaml` is followed by `make gen-clients` in the same commit (CI fails on stale
  clients).
- No personal data in a log line, error message, panic, golden file or fixture: `@example.test` addresses
  and invented names only. A test asserts the log output of each new handler holds no e-mail or token.
- Pin every version (Go modules, npm, pub).
- Update `PROGRESS.md` in the same commit.

## Check sets

Every check runs through `make`, in Docker. Never `go`, `npm` or `flutter` on the host.

- **Go**: `make lint`, `make test`, `make archcheck`, `make cover` (every package at 80% or more).
- **FHIR**: `make fhir-validate`.
- **Clients**: `make gen-clients` leaves no diff.
- **Web**: `make web-lint`, `make web-typecheck`, `make web-test`, `make web-build`.
- **Mobile**: `make mobile-analyze`, `make mobile-test`.
- **Stack**: `make up`, then the package's manual check, then `make down`.

## Packages

### 1. Privacy foundation

Migration: role `salus_app` (`LOGIN`, no password, no `BYPASSRLS`, created in a `DO` block that ignores
`duplicate_object`) and `audit_event` (SPEC §2 "AuditEvent shape"), with the triggers that reject `UPDATE`,
`DELETE` and `TRUNCATE`, an index on `(entity_reference, recorded)` and on `(agent_account_id, recorded)`,
and `GRANT INSERT, SELECT` to `salus_app` only. `cmd/migrate up` sets the role's password from
`APP_DATABASE_PASSWORD` when it is set (`config`). Compose: `api`, `worker`, `notifier`, `scheduler` get a
`DATABASE_URL` as `salus_app` (synthetic local password); `migrate` keeps the owner URL; the `tools` service
gains `TEST_APP_DATABASE_PASSWORD`. `pgtest` gains a helper that opens a pool on a test database as
`salus_app`. `postgres.SetLocal`. `internal/auditevent`: model (FHIR codes as `system` + `code` constants), `Service.Record`
inside the caller's transaction (it calls `WithTx`, which joins), Postgres repository. `internal/retention`:
`Rule` and `rules.go` (the `audit_event` rule). `ownership.yaml` gains `personal_data` on every
domain-owned table; archcheck gains the three retention checks (SPEC §2 "Retention hook"), test first with a
fixture repository per failure. `docs/privacy/processors.md` (no processor wired yet; SES, the hosting
provider and LiveKit's host listed as pending, with what each will touch), `breach.md` (detection,
assessment, the 72-hour notification to the supervisory authority, informing data subjects, the record of
breaches; contacts as labelled placeholders), `dpia.md` (the processing this feature adds: purposes, data,
necessity, risks and measures; updated by later packages).
Tests: as `salus_app`, `INSERT` and `SELECT` on `audit_event` work and `UPDATE`, `DELETE` and `TRUNCATE`
are denied by privilege; as the owner, the triggers reject all three; `salus_app` is not a superuser, has no
`BYPASSRLS`, owns no table, and with `SET row_security = off` a query on an RLS-guarded table fails (a
test-only table with a policy, created in the test database, until package 4 brings real ones; package 4
repeats the check on its tables); `cmd/migrate` sets the password and a pool can log in with it; `SetLocal`
is visible only inside its transaction.
Checks: Go.

### 2. Identity: accounts

Migration: `account` (SPEC §4), unique index on `email`, check that `organization_id` is null exactly when
`role = 'patient'`; grants. `internal/auth`: model, errors, password hashing (argon2id, PHC string, re-hash
on old parameters, semaphore), `Service.Register` (patient), `Service.CheckPassword` (used by login in 3),
`Service.Me`; each writes its `AuditEvent` in the same transaction. `BenchmarkHash` over a grid of `m`/`t`
at `p = 1`, run in the `tools` container and on CI; the chosen parameters and the numbers go in
`PROGRESS.md`. Handlers `POST /auth/register`, `GET /auth/me` (session check stubbed until 3: the handler
takes the account id from a context value the router will set), the JSON error shape, `openapi.yaml`,
`make gen-clients`. Retention rule for `account`.
Tests: register normalises the e-mail, rejects a short or over-long password (400), a duplicate e-mail
(409), stores no password; dummy hash on unknown e-mail (timing seam: the fake hasher counts calls); an
audit event per access, also on failure; a log line test.
Checks: Go, Clients.

### 3. Identity: sessions and MFA

Migration: `session` (index on `token_hash`, on `account_id`); grants. `internal/auth`: `Service.Login`,
`Logout` (sets `revoked_at`), `Authenticate` (one indexed read of the session row; no Redis), challenge store, TOTP (RFC 6238 test
vectors), `SecretBox` with an AES-GCM implementation keyed from config (`TOTP_KEY`), `Enrol`, `Confirm`,
`Verify`; login rate limiter (`repository_redis.go`, `RATE_LIMIT_KEY`, `TRUSTED_PROXY_CIDRS`). Session
middleware mounted in `internal/api/router` (cookie or bearer, CSRF header for cookies); `/auth/me` now uses
it. Handlers for login, MFA, logout; `openapi.yaml`; `make gen-clients`. Retention rule for `session`. A
benchmark of `Authenticate` with the numbers in `PROGRESS.md`.
Tests: staff without MFA get `next: enrol` and no session; a reused TOTP step is rejected; five wrong codes
end the challenge; 429 with `Retry-After` after the limit, counted per e-mail and per IP, keys expire;
revoked and expired sessions are rejected on the next request; logout writes no Redis key; audit events for each login outcome and logout; cookies carry `HttpOnly`, `Secure`,
`SameSite=Strict`.
Checks: Go, Clients.

### 4. Directory

Migrations: `organization`, `practitioner`, `practitioner_role` (each with `organization_id`, RLS policy,
grants; `organization`'s policy compares `id`). `internal/organization`, `internal/practitioner`,
`internal/practitionerrole`: models, services that set the clinic with `SetLocal` and check the caller's role
and clinic, Postgres repositories through their sqlc packages, `version_id` with `If-Match`-style updates on
`organization`. Services only; FHIR handlers come in 6. Audit events on every practitioner and role access.
Retention rules for `practitioner` and `practitioner_role`.
Tests: as `salus_app` (the package-1 pool helper), with `SET row_security = off` a query fails, and a query with clinic A set sees none of clinic B's rows and cannot insert a row for
clinic B; with no clinic set, nothing is visible; services reject a caller from another clinic (404) and a
patient (404); a stale version gives the conflict error.
Checks: Go.

### 5. Practitioner invites

Migration: `invite` (RLS, grants), role `salus_invite_lookup` (`NOLOGIN`, not a superuser, no `BYPASSRLS`;
`SELECT (organization_id, token_hash)` on `invite` and a `FOR SELECT TO salus_invite_lookup USING (true)`
policy), and `invite_organization(bytea)` (`SECURITY DEFINER`, `SET search_path = pg_catalog, public`,
schema-qualified body, `ALTER FUNCTION … OWNER TO salus_invite_lookup`, `EXECUTE` revoked from `PUBLIC` and
granted to `salus_app`). `internal/auth`: `Service.Invite` (admin only, own clinic; enqueues
`invite_email` with `InsertTx`), `ListInvites`, `IssueInviteToken`, `AcceptInvite` (password, TOTP
enrolment, account, and `Practitioner` plus `PractitionerRole` through their services, one transaction).
`internal/notification`: `Sender` port, SMTP implementation (`net/smtp`), the invite e-mail text.
`internal/worker`: the notifier's River client on the `email` queue with the `invite_email` worker;
`cmd/notifier` runs it. `cmd/salusctl` (owner connection) with `create-clinic` and `reset-mfa <account-id>`
(`auth.Service.ResetMFA`: clears TOTP, revokes the account's sessions, writes an `AuditEvent`, one
transaction; staff accounts only); `make salusctl`. Config `SMTP_ADDR`, `WEB_BASE_URL`.
Handlers `POST /invites`, `GET /invites`, `POST /invites/accept`; `openapi.yaml`; `make gen-clients`.
Retention rule for `invite`. `docs/privacy/` updated (e-mail now sent; Mailpit is local only).
Tests: the function's owner is `salus_invite_lookup`, not a superuser, and its `proconfig` holds the
`search_path`; `salus_app` cannot select from `invite` without a clinic but gets the clinic id from the
function; `reset-mfa` clears enrolment, revokes sessions, writes its audit event, and refuses a patient
account and an unknown id; the job is inserted in the same transaction (a rolled-back invite leaves no job); job arguments hold
only the two ids (asserted on the stored JSON); a retried job replaces the token, so the earlier link fails (recorded in `PROGRESS.md`); an expired or accepted
invite is rejected; a practitioner cannot invite; the SMTP sender against Mailpit in the compose network
(`TEST_SMTP_ADDR`, failing without it, like the database URLs); the e-mail body holds the link and the clinic
name and nothing else.
Checks: Go, Clients, Stack (`make salusctl args="create-clinic …"`, the e-mail arrives in Mailpit).

### 6. FHIR reads

`handler_fhir.go` in `organization` (read, update with `If-Match`, 412 on a stale version), `practitioner`
(read), `practitionerrole` (read, search by `organization` with `_include=PractitionerRole:practitioner`, a
`searchset` Bundle with `pkg/fhir` bundle primitives). `meta.versionId` and `meta.lastUpdated` on every
resource; `ETag` header. `api/fhir/CapabilityStatement.json` lists exactly these interactions and the search
parameter. Golden files for each resource, the Bundle, and the 404 / 412 `OperationOutcome`s; the metadata
golden updated.
Tests: golden JSON; another clinic's id and a patient caller both give 404; unknown `_include` or search
parameter gives 400; audit events for reads and searches.
Checks: Go, FHIR.

### 7. Web

`web/src`: sign in (password, then code; first staff login goes through enrolment with the QR code and the
secret as text), accept invite (from the link fragment), clinic settings (clinic name, saved with
`If-Match`), practitioners (list from the `PractitionerRole` search, open invites, invite form). A small
`fetch` wrapper for the FHIR calls with hand-written types for the elements used; the JSON calls through the
generated client. Tokens and colours from SPEC §2 "Web" in one `tokens.css`; inline SVG icons. Vite proxy
gains `/auth`, `/invites`, `/fhir`.
Tests: Vitest for each screen's states (loading, error codes, success), the MFA step order, the CSRF header
on every write, no secret left in the DOM after enrolment.
Checks: Web, Clients; the screens walked in a browser against the stack at desktop width and 390 px.

### 8. Mobile

`mobile/lib`: registration and sign-in screens (with the TOTP step when the account has MFA), token in
`flutter_secure_storage`, a signed-in screen that shows the account e-mail from `/auth/me` and a sign-out
button.
Tests: widget tests driving the generated client through `MockClient`: validation, 409 on a taken e-mail,
429, the MFA step, sign-out clears the stored token (storage behind a small fake).
Checks: Mobile, Clients.

### 9. End-to-end

`test/e2e/` (build tag `e2e`): a patient registers and logs in, `/auth/me` answers, logout makes the token
fail; `salusctl create-clinic` creates a clinic, the admin accepts the invite from Mailpit, enrols TOTP (the
test computes codes from the secret) and logs in; `salusctl reset-mfa` on the admin makes the next login ask
for enrolment again; the admin invites a practitioner; the practitioner accepts,
enrols and logs in with a code; the practitioner reads their `Organization` and the `PractitionerRole` search
returns them. `make e2e` runs `salusctl` and reads Mailpit's API on the compose network. Each journey uses
fresh synthetic e-mails so reruns do not collide.
Checks: all of the above, `make e2e` twice in a row.

## Definition of done

Every package committed, `PROGRESS.md` complete, all check sets pass locally and in CI on the PR into `main`.
