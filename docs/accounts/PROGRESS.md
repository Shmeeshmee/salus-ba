# Accounts progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Privacy foundation | done, 2026-11-04 | `accounts(1)` |
| 2 Identity: accounts | done, 2026-11-04 | `accounts(2)` |
| 3 Identity: sessions and MFA | done, 2026-11-04 | `accounts(3)` |
| 4 Directory | done, 2026-11-04 | `accounts(4)` |
| 5 Practitioner invites | done, 2026-11-04 | `accounts(5)` |
| 6 FHIR reads | done, 2026-11-04 | `accounts(6)` |
| 7 Web | done, 2026-11-04 | `accounts(7)` |
| 8 Mobile | done, 2026-11-04 | `accounts(8)` |
| 9 End-to-end | done, 2026-11-04 | `accounts(9)` |

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
- 2026-11-04, owner (asked during package 1):
  - **Coverage of generated code.** `make cover` excludes the generated sqlc packages
    `internal/<domain>/<domain>db` as well as `cmd/*/main.go` (`scripts/cover.sh`, CLAUDE.md "Checks"). They
    are never edited by hand and are exercised by their domain's repository tests; measured on their own they
    are always 0%.
  - **River's reindexer is off** (`ReindexerIndexNames: []string{}` in `internal/worker`). It runs
    `REINDEX INDEX CONCURRENTLY`, which needs the table's owner, and the worker logs in as `salus_app`, which
    owns nothing. Index maintenance on River's tables is an owner-side ops task.

## Waiting on people

- Before production: a legal decision on the retention periods (`SPEC.md` §4) and a KMS decision for
  the TOTP key (`TOTP_KEY` is local and CI only).
- Before production (ops): the owner role that runs `migrate` needs `CREATEROLE` to create `salus_app`;
  River index maintenance runs as the owner; DDL statement logging must not capture `ALTER ROLE … PASSWORD`
  (or `APP_DATABASE_PASSWORD` is left unset and the role provisioned outside migrate).
- Before production (ops): `ALTER FUNCTION invite_organization … OWNER TO salus_invite_lookup` (migration
  `00013`) needs a superuser, or an owner role that can `SET ROLE salus_invite_lookup` and a
  `salus_invite_lookup` with `CREATE` on schema `public`; locally `salus` is a superuser. Like `CREATEROLE`
  above, this is decided with the hosting choice.
- Before production: the e-mail sender address (`no-reply@salus.invalid` now, a constant in
  `internal/notification`) becomes configuration with a verified domain when SES is wired (needs its DPA).
- Before production: `FHIR_BASE_URL`, the api's public FHIR base (`https://<host>/fhir/R4`), set with the
  hosting choice; compose sets `http://localhost:${SALUS_API_PORT}/fhir/R4`.
- Branch protection on `main` (carried over from `docs/bootstrap/PROGRESS.md`): needs GitHub Pro.

## Log

### 2026-11-04 — plan

`SPEC.md`, this plan and `RUN.tsv` written.

### 2026-11-04 — plan approved

Owner approved with the changes under Decisions; `SPEC.md` and the plan updated. Committed as
`accounts(0): spec and plan`.

### 2026-11-04 — package 1, privacy foundation

Built:
- Migration `00009_privacy_foundation.sql`: role `salus_app` (`LOGIN`, no password, `NOSUPERUSER NOBYPASSRLS
  NOCREATEROLE NOCREATEDB`), its grants on River's tables, and `audit_event` (SPEC §2 "AuditEvent shape")
  with indexes on `(entity_reference, recorded)` and `(agent_account_id, recorded)`, triggers that reject
  `UPDATE`, `DELETE` (row) and `TRUNCATE` (statement) for every role, and `INSERT, SELECT` for `salus_app` only.
- `pkg/postgres`: `SetLocal` (`set_config(…, true)` on the transaction in the context; `ErrNoTx` outside one)
  and `SetPassword` (server-side `format('%I', '%L')`, never spliced; errors never carry the password).
- `pkg/postgres/pgtest`: `AppRole` and `NewAppDatabase(t) (owner, app)`; `Main` sets the role's password
  from `TEST_APP_DATABASE_PASSWORD` once per test binary when the migrations created the role.
- `internal/config`: `APP_DATABASE_PASSWORD`; `cmd/migrate up` sets it on `salus_app` after the migrations.
- Compose: `api`, `worker`, `notifier`, `scheduler` log in as `salus_app`; `migrate` as the owner with
  `APP_DATABASE_PASSWORD`; `tools` has `TEST_APP_DATABASE_PASSWORD`. One synthetic value,
  `local-app-password`, everywhere: the role is cluster-wide, so tests and the stack must agree. It differs
  from the role name, so a "no password in the output" check cannot pass by accident.
- `internal/auditevent` (context `privacy`): model with FHIR systems and codes as constants, `Service.Record`
  (validates action, outcome and codings; stamps `recorded` from the clock and the request id from the
  context; joins the caller's `WithTx`), sqlc repository (`auditeventdb`, first entry in `sqlc.yaml`).
  `pkg/middleware.WithRequestID` added (used by `RequestID`), so the request id reaches the service without
  `net/http`.
- `internal/retention` (context `privacy`): `Rule{Table, Basis, Then, Keep}`, `Rules()` with `audit_event`
  (3653 days, then removed by range).
- archcheck: `personal_data` required on every domain-owned table (`personal-data-unmarked`), a rule for
  every `personal_data: true` table (`retention-rule-missing`), and no rule on a table that no migration
  creates or that was dropped (`retention-rule-unknown-table`). `ownership.yaml` gains `audit_event`.
- `internal/worker`: River's reindexer off (owner decision above).
- `docs/privacy/processors.md` (none wired; SES, the hosting provider and the LiveKit host pending),
  `breach.md`, `dpia.md`.

Test first, failures seen before the code: archcheck's four retention tests (no violations reported);
`pkg/postgres` and `pgtest` (undefined `SetLocal`, `SetPassword`, `NewAppDatabase`); migration tests
(`relation "audit_event" does not exist`); the worker as `salus_app` without the River grants
(`permission denied for table river_queue`); `migrate up` ("did not set the salus_app password"); config
(`AppDatabasePassword` undefined); `auditevent` (undefined `Service`, `NewPostgresRepository`); retention (no
package); the reindexer config (undefined `config`).

Checks: `make lint` 0 issues, check-migrations ok; `make archcheck` ok; `make test` ok; `make cover` ok
(total 91.6%; auditevent, retention, worker, config 100%, postgres 92.9%, pgtest 82.1%, middleware 97.8%);
`make sqlc` leaves no diff. Stack: `make migrate`-equivalent through compose applied 00009 and set the
password; api (`/readyz` 200), worker, notifier and scheduler connected as `salus_app`. `make up` itself
could not bind 8080 and 8090 on this host (held by another project), so the api ran on `SALUS_API_PORT=18080`
for the check and landing was not started; not caused by this package.

Readings:
- **Concurrent `CREATE ROLE` raises `unique_violation`, not `duplicate_object`** (measured on Postgres 18.6:
  six sessions at once, five got `duplicate key value violates unique constraint "pg_authid_rolname_index"`).
  The `DO` block catches both. In that path it also fails the migration when an existing `salus_app` is a
  superuser, has `BYPASSRLS`, `CREATEROLE` or `CREATEDB`, or cannot log in.
- **Concurrent `ALTER ROLE` on one role raises `tuple concurrently updated` (XX000)** (eight sessions, seven
  failed). `SetPassword` retries that error up to 10 times with jittered backoff (≤ 10 ms × attempt); eight
  concurrent calls succeed, stable over 10 runs.
- **For package 4: write RLS policies as**
  `organization_id = NULLIF(current_setting('app.organization_id', true), '')::uuid`. After a transaction
  that used `SET LOCAL` ends, a pooled connection reads the setting as `''`, not NULL, and `''::uuid` is an
  error rather than "no rows". The test-only `clinic_probe` table in `migrations/privacy_test.go` uses this
  form; with it, no clinic set means nothing visible.
- **River grants for `salus_app`:** `SELECT, INSERT, UPDATE, DELETE` on `river_job`, `river_leader`,
  `river_queue`; `SELECT, DELETE` on `river_notification` (the pgx driver only prunes it); `USAGE` on
  `river_job_id_seq`. Taken from the queries in River's pgx driver (v0.49.0).
- **The production owner role needs `CREATEROLE`** to create `salus_app` (locally `salus` is a superuser).
- **`ALTER ROLE … PASSWORD` carries the password in plain text** to the server, so it appears in the server
  log if DDL statements are logged (`log_statement = ddl` or `all`). Production either keeps DDL logging off
  for migrate or leaves `APP_DATABASE_PASSWORD` unset and provisions the role outside migrate.
- `audit_event` is not row-level secured: its clinic is optional (a patient login has none) and only the
  application writes it; there is no read API in this feature.
- Not tested: the `DO` block's refusal of a privileged existing role. Testing it would mean making the
  cluster-wide role a superuser while other packages' tests use it.

Dev stack left: postgres and redis running, as found. Dev database at migration 9 (`audit_event`), role
`salus_app` with the synthetic password `local-app-password`. The other containers were removed again.

### 2026-11-04 — package 2, identity: accounts

Built:
- Migration `00010_account.sql`: `account` with the SPEC §4 fields (the MFA columns stay empty until
  package 3, `practitioner_id` until 5), unique index on `email`, checks that `organization_id` is null
  exactly for `role = 'patient'`, that `role` is one of the three, and that `email` is stored trimmed and
  lower-cased; `SELECT, INSERT, UPDATE` to `salus_app` (no `DELETE`, `TRUNCATE`). No foreign keys to the
  directory yet: its tables arrive in package 4.
- `internal/auth` (context `identity`, in `config.yaml`; `account` in `ownership.yaml` with
  `personal_data: true`; `authdb` in `sqlc.yaml`): model (`Account`, `Role`, `WithAccountID` /
  `AccountIDFrom`), errors, `Argon2Hasher` (`password.go`: PHC string, 16-byte salt, 32-byte key,
  constant-time compare, `stale` for older parameters, a semaphore that waits on the context),
  `Service.Register` (patient), `Service.CheckPassword` (for login in 3: dummy hash on an unknown e-mail,
  re-hash on older parameters), `Service.Me`, each auditing in the same transaction, also on failure;
  Postgres repository; `Handler` with `POST /auth/register` and `GET /auth/me` (account id from the context
  value the session middleware will set in package 3; until then `/auth/me` answers 401), errors as
  `{"error": "<code>"}`.
- `internal/retention`: `ThenKeep` (kept while the subject exists, no period) and the `account` rule. The
  rules test now requires a zero period exactly for `ThenKeep` and a positive one otherwise.
- `router.Config.Routes` mounts each domain's registration function behind the middleware; `app.go` builds
  `auditevent` and `auth` on the `salus_app` pool.
- `api/openapi.yaml`: both endpoints, `Account`, `RegisterRequest`, `Error` (fixed code enum), the
  `sessionCookie` and `bearerAuth` security schemes; `make gen-clients` regenerated the TypeScript and Dart
  clients.
- `make bench-hash` and a step in the Go workflow's test job, so CI prints the same numbers.
- `golang.org/x/crypto v0.58.0` (argon2; the version `tools/go.mod` already pinned).
- `docs/privacy/dpia.md`: package 2 section.

Test first, failures seen before the code: migration tests (`relation "account" does not exist`); retention
(`undefined: retention.ThenKeep`); password and service tests (`no non-test Go files`, then
`undefined: auth.Account`); handler (`undefined: auth.NewHandler`); router (`unknown field Routes`). The
repository tests and the `app.go` wiring tests were written before their code but first run after it.

Checks: `make lint` 0 issues, check-migrations ok; `make archcheck` ok; `make test` ok; `make cover` ok
(total 94.5%; auth 99.0%, api, router, auditevent, retention 100%); `make sqlc` and a second
`make gen-clients` leave no diff; `make mobile-analyze` no issues and `make web-typecheck` ok on the
regenerated clients.

Readings:
- **argon2id parameters, `BenchmarkHash` (p = 1, 20 iterations × 3 runs, tools container, Intel i7-14700HX,
  28 threads), ms per hash:**

  | m \ t | 2 | 3 | 4 |
  | --- | --- | --- | --- |
  | 19 MiB | 19.8 | 26.2 | 33.6 |
  | 32 MiB | 30.9 | 44.7 | **58.5** |
  | 46 MiB | 44.8 | 65.5 | 86.2 |
  | 64 MiB | 63.8 | 93.5 | 123.1 |

  Chosen: **m = 32 MiB, t = 4, p = 1**, the cheapest setting at or above the floor inside 50–100 ms (58.5 ms;
  the next are 64 MiB/t=2 at 63.8 ms and 46 MiB/t=3 at 65.5 ms). The SPEC asks for the CI runner; these are
  the tools container's numbers. CI prints its own with `make bench-hash` in the Go workflow; if the runner
  lands outside 50–100 ms, `DefaultArgon2Params` changes and stored hashes are re-hashed on the next login.
- **Semaphore, `BenchmarkHashParallel` (m = 32 MiB, t = 4), ms per hash of throughput:** 1 at a time 56.9,
  2: 30.7, 4: 16.7, 8: 9.8, 16: 8.6. Throughput scales almost linearly to 4 and flattens past 8 (memory
  bandwidth). `MaxConcurrentHashes = 4`: about 60 hashes/s with at most 128 MiB in use; a GitHub runner has 4 vCPUs.
- **The password check holds a transaction while it hashes.** SPEC requires the read and its audit event in
  one transaction, so `CheckPassword` keeps a pooled connection for one hash (about 60 ms), and longer when
  it waits for a hash slot. `Register` hashes before its transaction. Package 3's `Authenticate` benchmark
  should look at the pool size against `MaxConcurrentHashes` under a login burst.
- **A taken e-mail is detected with `ON CONFLICT (email) DO NOTHING`**, not by catching the unique violation:
  a violation aborts the transaction, and the failure has to be audited in it.
- **What is audited.** Registration: rest/create, C (success: agent and entity the new account; taken
  e-mail: outcome 4, no agent, no entity). Password check: DCM 110114/110122, E (outcome 0, or 4 for a wrong
  password with the account as agent and entity, or 4 with neither for an unknown e-mail). `GET /auth/me`:
  rest/read, R (an unknown account: outcome 4). The clinic is set for staff. A request rejected by
  validation before any account is touched (bad JSON, short password, invalid e-mail) is not audited: no
  account was accessed.
- **Audit rows in the database are checked in `internal/api`'s tests**, not `internal/auth`'s: archcheck
  (`foreign-repository`) forbids a domain, its tests included, to build `auditevent`'s repository. auth's
  Postgres tests use the fake auditor.
- `GET /auth/me` maps a missing account to 401 `unauthenticated` (the session points at nothing), not 404.
- The JSON 500 body stays `{"error":"internal error"}` from `httpx.InternalError`, and the timeout body
  `{"error":"timeout"}`; both are in the `Error` enum.
- E-mail validity: `net/mail.ParseAddress` must give back exactly the normalised input (no display name or
  comment), at most 254 characters. Password length is counted in characters (runes), 12 to 128.
- `TestHasher_BoundsConcurrentHashes` is timing-based (one slot held by a ~115 ms hash, a 20 ms deadline on
  the second); stable in every run here.

Deferred: login, sessions, MFA and the session middleware that sets the account id (package 3).

Dev stack left: postgres and redis running, as found. Dev database at migration 10 (`account`, empty). The
api was not started (no stack check in this package), so no account or audit rows were written.

### 2026-11-04 — package 3, identity: sessions and MFA

Built:
- Migration `00011_session.sql`: `session` (SPEC §4: account, clinic for staff, SHA-256 of the token,
  `mfa_verified`, `created_at`, `expires_at`, `revoked_at`), unique index on `token_hash`, index on
  `account_id`, foreign key to `account`, check that a staff session (clinic set) is MFA-verified;
  `SELECT, INSERT, UPDATE` to `salus_app` (no `DELETE`, `TRUNCATE`). Not row-level secured, like `account`
  (a patient's session has no clinic, and the session is found by token before the clinic is known).
  `ownership.yaml` gains `session` (`personal_data: true`); `internal/retention` gains `ThenDelete` and the
  `session` rule (30 days after `revoked_at` or `expires_at`).
- `internal/auth`:
  - `Service.Login` (rate limit, password, then a session for a patient without MFA or a challenge:
    `next: verify` when enrolled, `next: enrol` for staff without MFA), `Authenticate` (one indexed read, no
    Redis, not audited), `Logout` (sets `revoked_at`, audited), `Enrol`, `Confirm`, `Verify`
    (`service_session.go`, `service_mfa.go`). `CheckPassword` is folded into `Login`, its only caller; its
    tests became `Login` tests. `NewService` takes a `Ports` struct.
  - TOTP (`totp.go`, RFC 6238 with the standard library: HMAC-SHA-1, 6 digits, 30 s, one step of drift);
    exported `TOTPCode` (package 9's e2e test computes codes with it).
  - `SecretBox` port and `AESSecretBox` (`secretbox.go`, AES-256-GCM, nonce || ciphertext, the account id as
    additional data so a sealed secret opens only for its account).
  - `repository_redis.go`: `NewRedisChallengeStore` (`mfa:challenge:<sha256 hex>`, a hash with the account
    id and the wrong-code count; a Lua script counts only on an existing key and deletes at the limit) and
    `NewRedisCounters` (`ratelimit:<key>`, `INCR` + `PEXPIRE … NX` + `PTTL` in one `MULTI`: a fixed window).
  - Postgres repository: session and TOTP queries (`authdb`).
  - `handler_http_login.go`: `POST /auth/login`, `/auth/mfa/enrol`, `/auth/mfa/confirm`,
    `/auth/mfa/verify`, `/auth/logout`; the `Session` middleware (bearer token, else the `salus_session`
    cookie; a cookie write without `X-Salus-CSRF: 1` is 403 `csrf` before the session is read; an invalid
    token goes on anonymously); the client IP (remote address; `X-Forwarded-For` read from the right only
    while the hop is a trusted proxy). Error codes: `wrong_credentials`, `rate_limited` (429,
    `Retry-After` rounded up to whole seconds), `invalid_challenge`, `wrong_code`, `already_enrolled`,
    `not_enrolled`, `csrf`.
- `router.Config.Session` mounted inside the timeout and the request id; `app.go` builds the Redis stores,
  the AES box and the handler, and refuses a `TOTP_KEY` that is not 32 bytes or a `RATE_LIMIT_KEY` under 32.
- `internal/config`: `TOTP_KEY`, `RATE_LIMIT_KEY` (both base64), `TRUSTED_PROXY_CIDRS`. Compose gives the api
  synthetic keys; `configs/.env.example` documents them.
- `api/openapi.yaml`: the five endpoints, `LoginRequest`, `LoginResponse`, `EnrolRequest`, `Enrolment`,
  `ConfirmRequest`, `VerifyRequest`, the `X-Salus-CSRF` parameter, the new error codes; `make gen-clients`.
- `make bench-auth` (`internal/auth/login_bench_test.go`).
- `docs/privacy/dpia.md`: package 3 section.

Test first, failures seen before the code: the hashing-outside-the-transaction tests (`1 of 1 hashes inside
the transaction`, `repository calls outside the transaction: [], want [ByEmail]`); migration tests
(`relation "session" does not exist`); retention (`undefined: retention.ThenDelete`) and archcheck
(`retention-rule-missing: table session`); TOTP (`undefined: auth.TOTPCode`); the secret box
(`undefined: ErrSealedSecret`); service, repository and handler tests (`*postgresRepository does not
implement Repository (missing method ConfirmMFA)`, then the missing `Login`, `Enrol`, …); config
(`cfg.TOTPKey undefined`); router (`unknown field Session`); the app (`not enough arguments in call to
auth.NewHandler`). The Redis tests were written before `repository_redis.go` but first run after it.

Checks: `make lint` 0 issues, check-migrations ok; `make test` ok; `make archcheck` ok; `make cover` ok
(total 96.5%; auth 98.9%, api, router, config, retention 100%); `make sqlc` and a second `make gen-clients`
leave no diff; `make web-typecheck` ok and `make mobile-analyze` no issues on the regenerated clients. Stack
smoke (not required by this package): the api from compose on `SALUS_API_PORT=18080` (as an environment
variable, `deployments/.env` untouched) answered `/readyz` 200 and the expected 401/400 to requests that
touch no data, so no audit row was written to the dev database.

Readings:
- **The password hash no longer holds a database connection (owner's question).** Package 2's
  `CheckPassword` hashed inside its transaction. `BenchmarkLoginBurst` (`make bench-auth`): 32 concurrent
  patient logins with real argon2id (m = 32 MiB, t = 4, `MaxConcurrentHashes` = 4) against a pool of 4
  connections (the pgxpool default on a 4-vCPU host) and of 16, while an indexed read (what every other
  request does) arrives every 5 ms; tools container, i7-14700HX, three runs each:

  | pool | hash | burst | other reads p50 | p99 | max |
  | --- | --- | --- | --- | --- | --- |
  | 4 | inside the transaction | 602–656 ms | 272–281 ms | 589–623 ms | 613–653 ms |
  | 16 | inside the transaction | 573–610 ms | 59–67 ms | 364–391 ms | 382–420 ms |
  | 4 | before the transaction | 560–565 ms | 0.24–0.26 ms | 0.46–0.54 ms | 2.8–8.1 ms |
  | 16 | before the transaction | 568–577 ms | 0.24 ms | 0.46–0.69 ms | 0.7–3.5 ms |

  With the hash inside, every connection was held by a login hashing or waiting for a hash slot, and every
  other request waited for the backlog to drain: a real bottleneck. The burst itself is bound by the hash
  semaphore either way (32 / 4 × ~60 ms). Restructured: `Login` reads the account by e-mail with no
  transaction, verifies (and, for stale parameters, re-hashes) outside it, then in one transaction reads the
  account again by id, decides on that read, writes the new hash, opens the session when there is no MFA
  step, and writes the AuditEvent. The decision rests on the read inside the audited transaction; when the
  hash that was verified was replaced in between (a concurrent re-hash) the attempt fails as a wrong
  password, and the user signs in again. An unknown e-mail still costs one hash.
- **`Authenticate`, `BenchmarkAuthenticate`** (as `salus_app`, 10 000 sessions, the default pool of 28):
  34–36 µs per call serially (one round trip on the unique `token_hash` index), 3.5 µs per call of
  throughput in parallel (about 280 000 checks/s). No change needed.
- **The web app is told apart by `X-Salus-CSRF: 1`.** SPEC gives the cookie to the web app and the token in
  the body to the mobile app but not how the server knows which; the web app already sends the header on
  every write, so a login step with it gets the cookie (and no token in the body), one without it the token.
  Documented in `openapi.yaml`.
- **An invalid or stale session token is ignored, not refused**, so a browser with an expired cookie can
  still reach `/auth/login`; handlers that need a session answer 401. The CSRF check needs only the cookie's
  presence, so it runs before the session read.
- **Not audited: a rate-limited attempt, and a challenge that is unknown, expired or malformed.** Neither
  touches an account (package 2's reading for requests rejected before any account is accessed), and
  auditing throttled attempts would let one IP write unbounded rows into a table kept ten years. Audited:
  each login attempt that checks a password (DCM 110114/110122, E, also unknown e-mail and wrong password),
  each verify (110122, also wrong or reused codes), enrol and confirm (rest/update, U, also refusals), logout
  (110123, E, a second logout of the same session as outcome 4).
- **Counting:** every attempt counts against the IP (successes too, 30 per 15 minutes); only failures count
  against the e-mail (5 per 15 minutes); a success does not reset the e-mail's count. Fixed windows: the
  first count starts the window (`PEXPIRE … NX`) and later ones do not extend it.
- **Pending TOTP secret on the account.** Enrol stores the sealed secret in `account.totp_secret` with
  `mfa_enrolled_at` still null; a repeated enrol replaces it; confirm sets `mfa_enrolled_at` and
  `totp_last_step` once (`WHERE mfa_enrolled_at IS NULL`). A verify uses a step only if it is newer than the
  last (`UPDATE … WHERE totp_last_step < step`), race-free.
- **A successful verify or confirm deletes the challenge after the commit.** If that delete fails the answer
  is 500 and the session row exists but its token never left the server, so it is unusable and expires.
- **The failed-code limit lives in the Redis script** (`HINCRBY` only on an existing key, so a late wrong
  code never leaves a key without a TTL). The challenge holds the account id and the count in one hash.
- **The session table is not row-level secured** and is not clinic-scoped, for the same reason as
  `account`; package 4's RLS checks do not apply to it.
- `TOTPCode` is exported for package 9's end-to-end test, which computes codes from the enrolment secret.
- `golang.org/x/crypto` was not needed beyond argon2: TOTP and AES-GCM are the standard library's.

Deferred: none from this package. `RATE_LIMIT_KEY` and `TOTP_KEY` need production values (the KMS decision
for the TOTP key is already under "Waiting on people"); `TRUSTED_PROXY_CIDRS` is set when the load balancer
is chosen.

Dev stack left: postgres and redis running, as found. Dev database at migration 11 (`session`, empty;
`account` and `audit_event` empty). The api and migrate containers were started for the smoke check and
removed; the `salus:dev` image was rebuilt.

### 2026-11-04 — package 4, directory

Built:
- Migration `00012_directory.sql`: `organization` (`name`, `active`), `practitioner` (`given_name`,
  `family_name`, `organization_id`, `active`) and `practitioner_role` (`practitioner_id`, `organization_id`,
  `active`), each with `updated_at` and `version_id` (`meta.lastUpdated`, `meta.versionId`). Every table has
  `ENABLE ROW LEVEL SECURITY` and one policy in package 1's form,
  `… = NULLIF(current_setting('app.organization_id', true), '')::uuid` (`organization`'s compares `id`).
  `practitioner_role` references `(practitioner_id, organization_id)` of `practitioner`, so a role is always
  in its practitioner's clinic, with a unique index on `practitioner_id` (one role per practitioner) and an
  index on `organization_id` (the search). Grants to `salus_app`: `SELECT` and `UPDATE (name, updated_at,
  version_id)` on `organization` (clinics are created by `salusctl` as the owner); `SELECT, INSERT` on
  `practitioner` and `practitioner_role` (created on invite acceptance, package 5). No `DELETE`, `TRUNCATE`.
- `internal/organization`, `internal/practitioner`, `internal/practitionerrole` (context `directory` in
  `config.yaml`; the tables in `ownership.yaml`, `organization` with `personal_data: false`, the other two
  `true`; three `sqlc.yaml` entries): model, errors, `Repository` port, service, Postgres repository through
  the domain's sqlc package.
  - Each service takes a `Caller` (account id and, for staff, the clinic; `organization.Caller` also says
    whether the caller is the clinic's admin), sets the caller's clinic on its transaction through the
    repository's `SetOrganization` (`postgres.SetLocal`), and checks the clinic itself as well, so the
    rule does not rest on RLS alone. A patient (no clinic) and a caller of another clinic get `ErrNotFound`.
  - `organization.Service`: `Read`; `Update` (rename at the version read: `ErrConflict` for a stale version,
    also when it changes between the read and the `UPDATE … WHERE version_id = …`; `ErrForbidden` for a
    practitioner of the clinic; `ErrInvalidName` for a blank name or one over 200 characters; the name is
    trimmed).
  - `practitioner.Service.Read`; `practitionerrole.Service.Read` and `Search` (`PractitionerRole?organization=`,
    oldest first).
- `internal/retention`: `ThenRestrict` and the rules for `practitioner` and `practitioner_role` (kept while
  active, restricted after leaving while records reference them; no period). The rules test now allows no
  period exactly for `ThenKeep` and `ThenRestrict`.
- `docs/privacy/dpia.md`: package 4 section.
- `make sqlc` also regenerated `models.go` in `auditeventdb` and `authdb` (every generated package lists all
  tables of the schema).

Test first, failures seen before the code: migration tests (`relation "organization" does not exist`);
archcheck (`table-without-owner` for the three tables, then `retention-rule-missing` for `practitioner` and
`practitioner_role`); retention (`undefined: retention.ThenRestrict`); the three services (`no non-test Go
files`); the repositories (`undefined: organization.NewPostgresRepository`, and build failures for the other
two).

Checks: `make lint` 0 issues, check-migrations ok; `make archcheck` ok; `make test` ok; `make cover` ok
(total 96.1%; organization 93.8%, practitioner 94.6%, practitionerrole 94.7%, retention 100%); `make sqlc`
leaves no diff. No stack check in this package.

Readings:
- **Package 1's "salus_app cannot bypass RLS" checks, repeated on the real tables** (`migrations/directory_test.go`,
  as `salus_app` through `pgtest.NewAppDatabase`): with no clinic set nothing is visible, also on a pooled
  connection after a transaction that set one (the `''` case); with clinic A set only A's rows are visible;
  an `INSERT` of a practitioner or a role for clinic B with A set is refused (42501, RLS); an `UPDATE` of
  clinic B with A set touches no row; `SET LOCAL row_security = off` makes a query on each table fail
  (42501); `DELETE`, `TRUNCATE`, `INSERT` into `organization` and updates of columns outside the grants are
  denied by privilege. The package-1 tests that `salus_app` is not a superuser, has no `BYPASSRLS` and owns
  nothing run over the whole schema, now including these tables. The repository tests also run as
  `salus_app`: another clinic's id, an id that is not a UUID and no clinic set all give "not found".
- **What is audited.** `Practitioner` read: rest/read, R. `PractitionerRole` read: rest/read, R; search:
  rest/search-type, E (FHIR `AuditEvent.action`: a query is an execute), **one event per role returned**, so
  "who saw this record" is answered by the `(entity_reference, recorded)` index; a search that returns
  nothing writes one event without an entity. `Organization` update: rest/update, U. `Organization` reads
  are not audited: a clinic's name is not personal data and SPEC §2 asks for reads of `Practitioner` and
  `PractitionerRole` and for every write. All with purpose `HOPERAT`, the caller's account as agent and the
  caller's clinic.
- **A refused or empty access records no entity.** A patient, another clinic's caller, an unknown id or a
  stale version that never read the row: the event has outcome 4 and no `entity_reference`. The entity is
  written only from a row the database returned (a stale-version update does read the row, so it names the
  clinic), so no id from a request (a URL in package 6) is ever stored unvalidated. The agent and the clinic
  still say who tried.
- **A refusal is audited and committed.** The service returns nil from `WithTx` after recording the refusal
  and gives the error after the commit, as `auth` does; returning the error inside would roll the event back.
- **Organization: refused before the database for anyone but the clinic's members**; a practitioner of the
  clinic gets `ErrForbidden` for an update (they may read it, so its existence is no secret), everyone else
  `ErrNotFound`.
- **Foreign keys from `account` to the directory are not added.** `account.organization_id` and
  `practitioner_id` stay without them: the package's plan does not list them, and the identity tests insert
  staff accounts with synthetic clinic ids that have no `organization` row. Adding them is a separate change
  (with those tests creating clinics), best taken with package 5, which creates the first linked rows.

Deferred:
- Creating directory rows (`organization` by `salusctl create-clinic`, `practitioner` and `practitioner_role`
  on invite acceptance) is package 5's: the services have no `Create` yet, because nothing in this package
  calls one. The grants for those writes are already in `00012`.
- `_include=PractitionerRole:practitioner` (package 6): the search returns roles; the handler or a
  `practitioner` read of the listed ids adds the practitioners, audited as reads.
- Wiring the three services in `internal/api/app.go` comes with their FHIR handlers in package 6; the audit
  rows they write are checked against the database there, as package 2 did for `auth`.
- The role search is not paginated (a clinic has few practitioners); `_count` is not supported.

Dev stack left: postgres and redis running, as found. Dev database at migration 12 (`organization`,
`practitioner`, `practitioner_role`, all empty); `make migrate` set the `salus_app` password again (the same
synthetic value). No other container was started.

### 2026-11-04 — package 5, practitioner invites

Built:
- Migration `00013_invite.sql`: `invite` (SPEC §4: e-mail, names for a practitioner only, role, inviter,
  `expires_at`, `accepted_at`, `token_hash`), a partial unique index on `(organization_id, email) WHERE
  accepted_at IS NULL` (one open invite per e-mail and clinic, replaced by `INSERT … ON CONFLICT … DO
  UPDATE`), a unique index on `token_hash`, RLS in package 1's form; grants to `salus_app`: `SELECT,
  INSERT` and `UPDATE` of every column but `email` and `organization_id` (no `DELETE`, `TRUNCATE`). Role
  `salus_invite_lookup` (`NOLOGIN`, no superuser, no `BYPASSRLS`, created in the `DO` block form of
  `salus_app`) with `SELECT (organization_id, token_hash)` and one `FOR SELECT TO salus_invite_lookup USING
  (true)` policy. `invite_organization(bytea) RETURNS uuid`: `SECURITY DEFINER`, `STABLE`, `SET search_path
  = pg_catalog, public`, schema-qualified body, owned by `salus_invite_lookup`, `EXECUTE` revoked from
  `PUBLIC` and granted to `salus_app`. `audit_event` gains `agent_name` (FHIR `AuditEvent.agent.name`) for
  the operator command. `ownership.yaml`: `invite` (auth, `personal_data: true`); retention rule: deleted 30
  days after `accepted_at` or `expires_at`.
- `internal/auditevent`: `AgentName`, `AgentOperator = "salusctl"`.
- Directory `Create` (deferred from package 4): `organization.Service.Create` (operator only, as the owner;
  audited with `salusctl` as the agent), `practitioner.Service.Create` and `practitionerrole.Service.Create`
  (in the caller's clinic, set with `SetLocal`; a caller without a clinic gets `ErrForbidden`; audited as
  rest/create, C, also when refused).
- `internal/auth`: `Service.Invite` (admin of the session's clinic only; the account is read in the
  transaction, so the role comes from the database, not the session), `InviteAdmin` (operator),
  `ListInvites`, `IssueInviteToken`, `AcceptInvite`, `ResetMFA` (`service_invite.go`, `service_mfa.go`);
  ports `Jobs` (`repository_river.go`: `InsertTx` on queue `email`), `Practitioners`, `PractitionerRoles`
  (the directory services, wired as they are); `InviteEmailArgs{invite_id, organization_id}`; handlers
  `POST /invites`, `GET /invites`, `POST /invites/accept` (`handler_http_invite.go`); error codes
  `forbidden`, `invalid_role`, `invalid_name`, `invalid_invite`.
- `internal/notification` (context `notification`): `Message`, the `Sender` port, the SMTP sender
  (`net/smtp` over a context dialer, no auth or TLS: Mailpit), `Service.SendInvite` (clinic name through
  `organization.Service.Read`, token through `auth.Service.IssueInviteToken`, then the send, outside any
  transaction).
- `internal/worker`: `NewInserter` (insert-only client for the api and salusctl) and `NewNotifier` (works
  only `email`, with the `invite_email` worker; an invite that is no longer open cancels the job;
  reindexer off). `cmd/notifier` runs it (`SMTP_ADDR`, `WEB_BASE_URL`); `cmd/worker` keeps the default queue.
- `cmd/salusctl`: `create-clinic -name … -admin-email …` (clinic and invite in one transaction, prints the
  two ids) and `reset-mfa <account-id>`; `make salusctl args="…"` (the tools container, as the owner).
- `pkg/postgres.Tx(ctx)`: the transaction of the `WithTx` in the context, for `InsertTx`.
- `internal/config`: `SMTP_ADDR`, `WEB_BASE_URL`. Compose: the notifier gets both and waits for Mailpit;
  the tools service gets `TEST_SMTP_ADDR`, `TEST_MAILPIT_URL` and waits for Mailpit; Postgres runs with
  `max_connections=200` (reading below).
- `api/openapi.yaml`: the three endpoints, `InviteRequest`, `Invite`, `InviteList`, `AcceptInviteRequest`,
  `Acceptance`, the new error codes; `make gen-clients`.
- `docs/privacy/dpia.md` (package 5 section), `processors.md` (e-mail is sent, to Mailpit only).

Test first, failures seen before the code: migration tests (`relation "invite" does not exist`; `column
"agent_name" … does not exist`; no `invite_organization` row in `pg_proc`); archcheck
(`table-without-owner: table invite`); retention (`no rule for invite`); auditevent (`e.AgentName undefined`);
the three directory `Create`s (`undefined: practitioner.ErrForbidden`, `… has no field or method Create`);
`pkg/postgres` (`undefined: postgres.Tx`); auth services (`undefined: auth.Invite`); River jobs
(`undefined: auth.NewRiverJobs`); invite handlers (404, no routes); notification (`no non-test Go files`);
the SMTP error test (the server's reply, with the address, was in the error); config (`cfg.SMTPAddr
undefined`); worker (`undefined: worker.NewNotifier`); salusctl (`undefined: run`); the api journey
(`POST /invites` failed: no jobs port wired). The auth Postgres invite tests were written after the
repository methods (with the service) and first run after them.

Checks: `make lint` 0 issues, check-migrations ok; `make archcheck` ok; `make test` ok (twice); `make cover`
ok (total 94.3%; auth 95.5%, notification 87.8%, worker 100%, api 88.9%, organization 94.0%, practitioner
95.2%, practitionerrole 94.9%, postgres 93.2%); a second `make sqlc` and `make gen-clients` leave no
further diff; `make web-typecheck` ok, `make mobile-analyze` no issues. Stack: `make up` (with
`SALUS_API_PORT=18080`, `SALUS_MAILPIT_PORT=18025`, `SALUS_LANDING_PORT=18090` in a temporary
`deployments/.env`), migrate applied `00013`, `make salusctl args="create-clinic -name 'Synthetic stack
clinic' -admin-email stack-check@example.test"` printed the two ids, the notifier completed the job on its
first attempt, the e-mail arrived in Mailpit with the clinic name and the link and nothing else, one token
hash was stored, both creates were audited with `agent_name = salusctl`, and no e-mail address appeared in
the api or notifier logs.

Readings:
- **A retried `invite_email` after a send that succeeded issues a new token** (owner request). The job
  issues the token in its own transaction before sending; a retry (the job marked failed after the SMTP
  server accepted the message, or a send that failed) stores a new hash, so the link in an earlier e-mail
  stops working and only the newest is valid. Tested: `TestIssueInviteToken_StoresOnlyTheHashAndReplacesAnEarlierToken`
  accepts with the first token and gets `ErrInvalidInvite`. Replacing an open invite (a new invite for the
  same e-mail) also clears its token.
- **Accepting enrols TOTP in the acceptance transaction and confirms it with the existing endpoint.**
  `POST /invites/accept` stores the account (with the invite's clinic and role), the sealed pending secret,
  the accepted mark and, for a practitioner, the `Practitioner`, `PractitionerRole` and
  `account.practitioner_id`, in one transaction, and returns the secret, its URI and a 5-minute challenge;
  `POST /auth/mfa/confirm` with the first code marks enrolment and opens the session. If the challenge
  lapses, a password login gives `next: enrol` (staff without MFA), so nothing is stuck.
- **Order inside the acceptance transaction:** the account is inserted first (`ON CONFLICT DO NOTHING`), so a
  taken e-mail is audited and committed with nothing else written; then the invite is marked accepted
  (`WHERE accepted_at IS NULL AND expires_at > now`), so a concurrent acceptance of the same invite fails
  there and rolls its account back.
- **What is audited.** Invite: rest/create, C, entity `Invite/<id>` (refused: outcome 4, no entity). List:
  rest/search-type, E, one event per invite returned, one without an entity for none (as package 4's role
  search). Acceptance: rest/create, C, agent and entity the new `Account/<id>`; an expired or used invite or a
  taken e-mail: outcome 4, no agent, entity `Invite/<id>`; an unknown token is not audited (nothing was
  accessed). The directory creates audit themselves with the new account as agent, so a practitioner's
  acceptance writes three events. `IssueInviteToken` (a system step of the job) is not audited: SPEC §2
  lists invite creation and acceptance. `Invite/<id>` is not a FHIR resource type; it is the entity text
  until an `AuditEvent` read API maps entities. `salusctl` actions: agent `agent_name = salusctl`, no
  account; `reset-mfa` of a patient is refused and audited, of an unknown id refused and not audited.
- **Invite creation does not check whether the e-mail has an account**, so an admin cannot learn that
  someone (perhaps a patient) has one; the invitee gets `email_taken` on acceptance.
- **The SMTP sender keeps only the reply code** of a refusal: servers echo the recipient in their reply,
  and River stores the job's errors and logs them.
- **`max_connections` 100 → 200 on the compose Postgres.** The first `make cover` of this package failed
  with 53300 (too many clients) in several packages; sampled every 0.5 s during a passing run, the peak was
  87 of 100. Every test package runs in parallel and each app-role test opens two pools, the notifier tests
  a River client too. With 200, `make cover` passed twice and `make test` twice.
- `invite_email` jobs for an invite that is replaced, accepted or expired are cancelled (`river.JobCancel`),
  not retried.
- The notifier builds `auth.Service` with only `Repo`, `Tx` and `Clock`: `IssueInviteToken` needs nothing
  else (documented on `auth.Ports`).
- The clinic name for the e-mail is read through `organization.Service.Read` with the invite's clinic as the
  caller's; reads of a clinic are not audited (package 4).
- `make test` now starts Mailpit (the tools service depends on it). On this host 8025 is held by another
  project, so local runs need `SALUS_MAILPIT_PORT` in `deployments/.env`.

Deferred:
- Foreign keys from `account.organization_id` and `account.practitioner_id` to the directory (package 4's
  note) are still not added: the plan does not list them and the identity tests insert staff with synthetic
  clinic ids.
- The sender address becomes configuration with SES (under "Waiting on people").
- Wiring the directory services' FHIR handlers in `app.go` is package 6's; their services are now built in
  `app.go` for invite acceptance.

Dev stack left: postgres and redis running. Postgres was recreated with `max_connections=200` (same data
volume). Dev database at migration 13; it holds the stack check's synthetic rows: organization "Synthetic
stack clinic", its admin invite to `stack-check@example.test` (token issued, not accepted), one completed
`invite_email` job and two `audit_event` rows (append-only, so they stay). The api, worker, scheduler,
notifier, agent, livekit, mailpit, landing and migrate containers were removed again (Mailpit's in-memory
messages with them); the temporary `deployments/.env` was deleted.

### 2026-11-04 — package 6, FHIR reads

Built:
- `internal/organization/handler_fhir.go`: `GET /fhir/R4/Organization/{id}` and `PUT /fhir/R4/Organization/{id}`
  (the body is the Organization; only `name` changes; `If-Match` names the version). `internal/practitioner/
  handler_fhir.go`: `GET /fhir/R4/Practitioner/{id}`, and `ToFHIR`/`Resource` for the include.
  `internal/practitionerrole/handler_fhir.go`: `GET /fhir/R4/PractitionerRole/{id}` and
  `GET /fhir/R4/PractitionerRole?organization=<id>|Organization/<id>[&_include=PractitionerRole:practitioner]`,
  a `searchset` Bundle. Every resource carries `meta.versionId` and `meta.lastUpdated` and is sent with
  `ETag: W/"<version>"`. Each handler takes its caller as a function from the composition root (the
  directory packages cannot import `auth`: `auth` imports `practitioner` and `practitionerrole`).
- Status mapping: no session 401 (`login`); not found or not allowed to see 404 (`not-found`); unsupported
  search 400 (`not-supported`); a body that is not this Organization 400 (`invalid`); missing, malformed or
  stale `If-Match` 412 (`conflict`); a practitioner renaming the clinic 403 (`forbidden`); a blank or
  over-long name 422 (`business-rule`); anything else 500 through `fhir.InternalError`. Fixed diagnostics only.
- `pkg/fhir`: `Meta`/`NewMeta`, `Reference`, `ETag`, `IfMatch`, `Bundle`/`BundleEntry`/`BundleLink`/
  `BundleSearch`/`NewSearchset`, `InternalError` (`httpx.InternalError` for FHIR endpoints: logs the error,
  sends a fixed `exception` OperationOutcome); `Write` no longer HTML-escapes (`&` in links stays `&`).
- `auth.Service.IsClinicAdmin` (whether the session's account is its clinic's admin, read from `account`).
- `internal/api/app.go`: builds the three directory services once (shared with `auth`), mounts the three
  handlers with their callers, and requires `FHIR_BASE_URL` (absolute http(s)); `internal/config`
  `FHIRBaseURL`; compose gives the api `FHIR_BASE_URL`; `configs/.env.example` documents it.
- `api/fhir/CapabilityStatement.json`: exactly Organization read and update (`versioned-update`),
  Practitioner read, PractitionerRole read and search-type with `organization` (reference) and
  `searchInclude` `PractitionerRole:practitioner`; the metadata golden updated on purpose.
- Golden files (all reviewed): organization (`organization`, `organization_updated`, `not_found`,
  `forbidden`, `if_match_required`, `version_conflict`, `invalid_resource`, `invalid_name`,
  `unauthenticated`), practitioner (`practitioner`, `not_found`, `unauthenticated`), practitionerrole
  (`practitioner_role`, `search`, `search_include`, `search_empty`, `unsupported_search`, `not_found`,
  `unauthenticated`).
- `docs/privacy/dpia.md`: package 6 section.

Test first, failures seen before the code: `pkg/fhir` (`undefined: fhir.ETag`, `fhir.IfMatch`,
`fhir.NewMeta`, `fhir.NewSearchset`, `fhir.InternalError`; then `unknown field FullURL`; then the escaped
`\u0026`); auth (`f.svc.IsClinicAdmin undefined`); the three handlers (`undefined: …NewHandler`); the
CapabilityStatement test (no resources listed); the metadata golden (differs); config (`cfg.FHIRBaseURL
undefined`); the api journey (every FHIR path 404 from the catch-all, 0 audit events, a missing or relative
`FHIR_BASE_URL` accepted); `cmd/api` (refused to start without `FHIR_BASE_URL`, before its test set it);
`make fhir-validate` on the first searchset goldens (errors: entries without `fullUrl`, relative references
inside them; warning: no self link).

Checks: `make lint` 0 issues, check-migrations ok; `make archcheck` ok; `make test` ok; `make cover` ok
(total 94.8%; organization 95.8%, practitioner 96.1%, practitionerrole 96.3%, auth 95.6%, api 91.8%,
config 100%, `pkg/fhir` 100%, `api/fhir` 100%); `make fhir-validate` 0 errors on all 21 goldens (the only
warnings are dom-6, no narrative, as on the existing goldens). No migration, query or `openapi.yaml`
change, so `make sqlc` and `make gen-clients` are not affected. No stack check in this package (plan: Go,
FHIR).

Readings:
- **Audit rows checked against the database** (`internal/api/app_directory_test.go`, deferred from package
  4): through the real app, an admin and a practitioner of one clinic (from invites) and a patient. Rename,
  stale rename (outcome 4, names the clinic, since the row was read) and the practitioner's refused rename
  (outcome 4, no entity) each write one `update`; the search writes one `search-type`/E per role returned and
  the include one `read`/R per practitioner; the admin's two reads one each; the patient's three refusals
  have outcome 4, no entity and no clinic; Organization reads and signed-out requests write nothing. Exactly
  10 events for the requests, agents checked against the accounts.
- **No session is 401, not 404.** SPEC §2's 404 is for signed-in callers who may not see the resource. A
  request without a session is answered before any service call and is not audited (package 2 and 3's
  reading: nothing was accessed, and auditing anonymous requests would let anyone write rows kept ten years).
  The 401 is the same for every id, so it discloses nothing.
- **The admin flag is read from `account` per Organization request** (`auth.Service.IsClinicAdmin`, one
  indexed read by id), as `Invite` does: the session row holds no role. Not audited, like the session check
  (SPEC §2 "What every access to an account means"); the Organization access it serves is audited.
- **`_include` is done in the handler, through `practitioner.Service.Read`**, one audited read per included
  practitioner (package 4's deferred note). Each read is its own transaction; the role search and the reads
  are not one snapshot, which only matters if a practitioner changed in between (they cannot be deleted). A
  practitioner the read refuses is left out of the Bundle. No measurement needed: a clinic has few
  practitioners (one role each, unique index), so at most a handful of reads.
- **`FHIR_BASE_URL`.** The HL7 validator rejects a searchset entry without an absolute `fullUrl` (and
  relative references inside it), so the api needs its public base. A setting, not the request's Host and
  scheme: behind a TLS-terminating load balancer the scheme would be wrong, and the Host header is the
  client's. The self link is the search as understood (`organization=<id>`, plus the include), so both
  accepted forms of the parameter give the same Bundle.
- **Search parameters are strict** (plan): only `organization` (exactly once, one id, `<id>` or
  `Organization/<id>`) and `_include=PractitionerRole:practitioner` (at most once); anything else, a
  modifier, a comma list or `_count` gives 400 without calling the service. A query that does not parse is
  400 too.
- **Organization update.** `If-Match` is required (R4 http: a server that requires version-aware updates
  answers 412 without it); `W/"n"` and `"n"` are accepted, a version that is not an int32 is 412. The body
  must be an Organization with the URL's id (400 otherwise), at most 64 KiB; only `name` is applied (the
  clinic settings screen changes only that; `active` and the rest are ignored, as documented in the
  CapabilityStatement). A blank or over-long name is 422 `business-rule`: valid FHIR, refused by our rule.
- **Element order in the JSON follows the structs**, which `fieldalignment` orders; FHIR JSON has no
  element order, so no custom marshalers.
- **`fhir.InternalError` instead of `httpx.InternalError` on FHIR endpoints**: the same log line, but the
  body is an OperationOutcome (CLAUDE.md "FHIR": errors on FHIR endpoints are OperationOutcomes).

Deferred:
- `make e2e` covers the practitioner reading their `Organization` and the role search (package 9).
- Patients reading the directory (scheduling feature, SPEC §5).
- `POST /fhir/R4/PractitionerRole/_search`, paging and `_count` are not supported (not in SPEC).

Dev stack left: postgres, redis and mailpit running, as found; no other container started, and the dev
database untouched (still at migration 13 with package 5's synthetic rows). The `salus-tools:dev` and
`fhir-validator` images were rebuilt by make. `deployments/.env` was not touched.

### 2026-11-04 — package 7, web

Built:
- `web/src`, four screens chosen from `location.pathname` in `App.tsx` and `route.ts` (`usePath` over
  `popstate`, `navigate`), no router library:
  - `/sign-in` (`SignIn.tsx`): e-mail and password, then the code (`next: verify`), or for staff without
    MFA the enrolment (`next: enrol`: `POST /auth/mfa/enrol` with the challenge, the QR code and the secret
    as text, the first code to `/auth/mfa/confirm`). The session arrives as the cookie.
  - `/invite` (`AcceptInvite.tsx`): the token from the fragment, a new password typed twice,
    `POST /invites/accept`, then the same enrolment (`Enrolment.tsx`) confirmed with the acceptance's
    challenge.
  - `/clinic` (`ClinicSettings.tsx`, admins only): the clinic name from `GET /fhir/R4/Organization/{id}`,
    saved with `PUT` and `If-Match: W/"<versionId>"`; 412 offers a reload.
  - `/` (`Practitioners.tsx`): the practitioners from `PractitionerRole?organization=<id>&_include=
    PractitionerRole:practitioner`; for an admin also the open invites (`GET /invites`) and the invite form
    (practitioner with given and family name, or admin with the e-mail only).
  - A header with the two links (clinic settings for admins only) and sign out; `/auth/me` decides the
    account, its role and clinic, and a 401 sends the visitor to `/sign-in`.
- `api/client.ts`: an openapi-fetch middleware sets `X-Salus-CSRF: 1` on every request that is not a read.
  `api/fhir.ts`: a small `fetch` wrapper for `/fhir/R4` (`get`, `put` with `If-Match`, the FHIR media type
  and the CSRF header) with hand-written types for the elements used, and `FhirError` (status and the first
  `OperationOutcome` issue code). `api/result.ts`: `attempt` turns a generated-client call into data or an
  error code (none when the API is unreachable).
- `QRCode.tsx`: the `otpauth://` URI drawn as an SVG path with `qrcode-generator` 2.0.4 (pinned exact, no
  dependencies, MIT), quiet zone of four modules. `icons.tsx`: the logo and three line icons. `ui.tsx`
  (card, field, alert, notice, submit button, link), `useSubmit.ts`, `messages.ts` (fixed sentences).
- `tokens.css` (SPEC §2 "Web": page `#F6F7F5`, white surfaces, ink `#0E1512`, brand `#0A6E4F`, mint
  `#34D399`, the Helvetica stack; plus an error red) and `app.css`; one breakpoint at 520 px.
- `vite.config.ts`: the dev proxy gains `/auth`, `/invites`, `/fhir`.
- `HealthStatus` and its test removed: `App` was its only user and the screens no longer show the API
  status (CLAUDE.md: delete code that is not used).
- `docs/privacy/dpia.md`: package 7 section.

Test first, failures seen before the code: `api/client.test.ts` (`expected null to be '1'`: no CSRF header
on the logout write); `api/fhir.test.ts`, `SignIn`, `AcceptInvite`, `ClinicSettings`, `Practitioners`
(`Failed to resolve import`); `App.test.tsx` (10 of 10 failing against the old health page). Vitest: 56
tests in 7 files, covering each screen's loading, error codes and success, the order login → enrol →
confirm and login → verify, the CSRF header on every write (`expectCsrfOnWrites` in each screen's write
tests), and that the secret is gone from the DOM after enrolment.

Checks: `make web-lint` 0 warnings, 0 errors; `make web-typecheck` ok; `make web-test` 56 passed;
`make web-build` ok (264.77 kB JS, 83.56 kB gzip; 4.76 kB CSS); `make gen-clients` no diff (no
`openapi.yaml` change); `make lint` 0 issues, check-migrations ok; `make archcheck` ok; `make test` ok;
`make cover` ok (total 94.8%, unchanged: no Go change).

Browser walk: `make up` (ports from `deployments/.env`), the Vite dev server (`--profile web up -d node`),
`make salusctl args="create-clinic -name 'Synthetic walk clinic' -admin-email walk-admin@example.test"`, then
a Playwright script (scratch only, not in the repo; `mcr.microsoft.com/playwright:v1.56.1-noble`, Chromium,
host network) that reads the invite links from Mailpit's API and computes codes from the shown secret.
At 1280 × 800: accept the admin invite, enrolment; first sign-in with `next: enrol` and enrolment; invite a
practitioner; rename the clinic; sign out; sign in with a code. At 390 × 844: the practitioner accepts their
invite and enrols and sees the list; the admin signs in with a code and opens both screens; sign out. Every
screen screenshotted and reviewed. Checked on each: no horizontal overflow, the invite token gone from the
address bar, the secret gone from the DOM after confirming. Console: only the expected 401s of `/auth/me`
while signed out. No e-mail, invite link or password in the api, notifier or dev server logs.

Readings:
- **Screens by role.** The clinic settings link and screen are for admins (a practitioner's `PUT` is 403);
  a practitioner on `/clinic` sees the practitioners. A patient who signs in is told the web app is for
  clinic staff and sees no clinic data (SPEC §5: no patient web, patients use the mobile app). Unknown paths
  say the page does not exist.
- **The password is typed twice on acceptance.** Password reset is out of scope (SPEC §5), so a typo would
  lock the new account out; the sign-in form asks once. Length is checked as 12 to 128 characters (code
  points, as the API counts) before sending.
- **The invite token is removed from the address bar in an effect**, not in the state initialiser: React
  StrictMode calls initialisers twice, and the second call would find no fragment.
- **A lapsed challenge.** During sign-in it returns to the password step with a notice; after acceptance it
  says the account exists and links to sign in, where `next: enrol` resumes (package 5's reading).
- **Sign-out that fails keeps the user signed in** with a message, since the cookie may still be valid; a 401
  counts as signed out.
- **An invite for an e-mail already listed replaces that row**, as the API replaces the open invite.
- **The secret is shown unformatted** (base32 in a monospace block) so it can be copied as it is; the
  `otpauth://` URI is never put in the DOM, only drawn.
- **Fixed messages.** Every API error code maps to a sentence in the screen; anything else, and an
  unreachable API, gives "Something went wrong. Try again." No response text is shown.
- No browser storage is used; the session is only the `HttpOnly` cookie.

Deferred:
- Serving the built web app (a production image and its host) is not in this feature; the dev server proxies
  the API. `WEB_BASE_URL` points at it locally.
- No browser test in CI: the walk is a manual check (plan), and `make e2e` (package 9) drives the API.

Dev stack left: postgres, redis and mailpit running, as found; the api, worker, scheduler, notifier, agent,
livekit, landing, migrate and node containers were started for the walk and removed again; `salus:dev` was
rebuilt by `make up`. Dev database still at migration 13, now also holding the walk's synthetic rows: the
clinic "Synthetic walk clinic, renamed" with its admin `walk-admin@example.test` and practitioner
`walk-practitioner@example.test` (accounts enrolled, `Practitioner` and `PractitionerRole`), both invites
accepted, their sessions (one admin session still open, it expires after 12 h), two completed `invite_email`
jobs and the audit rows (append-only). Mailpit holds the two invite e-mails (in memory). The Playwright image
was pulled. `deployments/.env` was not touched.

### 2026-11-04 — package 8, mobile

Built:
- `mobile/lib`, patients only, all calls through the generated Dart client (`packages/salus_api`, untouched):
  - `session.dart`: `Session` over `AuthApi` and the client's `HttpBearerAuth`: `restore` (stored token →
    `GET /auth/me`; a 401 deletes the token), `register`, `login` (`done` opens the session, `verify` asks
    for the code, `enrol` is a staff account), `verify`, `signOut`. Requests carry
    `Authorization: Bearer` and never `X-Salus-CSRF`, so the API answers with the token in the body
    (package 3). `AuthFailure` keeps only the status, the fixed error code and `Retry-After`; an
    unreachable API is status 0.
  - `token_store.dart`: the `TokenStore` port (faked in tests) and `SecureTokenStore` over
    `flutter_secure_storage` 11.2.0 (pinned exact; Keychain on iOS, Keystore-backed on Android), key
    `session_token`. Never shared preferences.
  - Screens: `sign_in_screen.dart` (e-mail and password, then the 6-digit code; "Use another account" goes
    back), `register_screen.dart` (e-mail, password of 12 to 128 characters counted in code points, then
    straight into the account), `home_screen.dart` ("Signed in as <e-mail>" from `/auth/me`, sign out),
    `app.dart` (restores the session, switches the screen; no router), `messages.dart` (fixed sentences and
    the validators), `ui.dart` (theme with the web app's colours, page column, message, submit button).
  - `main.dart` wires the client, the bearer auth and the secure store.
- `health_screen.dart` and its test removed: the signed-in screen replaces it and nothing else used it
  (CLAUDE.md: delete code that is not used).
- `docs/privacy/dpia.md`: package 8 section.

Test first, failures seen before the code: every new test file failed to load (`Method not found:
'SecureTokenStore'`, missing `package:salus/app.dart`, `session.dart`, `token_store.dart`). After the code,
the tests that read the account failed on the fixture (`Required key "Account[mfaEnrolled]" is missing from
JSON`), fixed in the fixture; the log test first failed because it restored `debugPrint` in tear-down,
after the binding checks it. Widget tests, 31 in 5 files, drive the generated client through `MockClient`
with the token store faked: validation before any request (e-mail, password length, code format), sign-in
without MFA, wrong password, 429 with and without `Retry-After`, an unreachable API, the MFA step (order
login → verify → me, request bodies, wrong code, an ended challenge back to the password, going back),
staff refused (`next: enrol` without enrolling; a staff session from a code ended with `POST /auth/logout`
and not stored), registration (success into the account, 409, the API's 400 codes, a sign-in after
registering that fails lands on sign-in with a notice), restoring a stored token (bearer header, 401 deletes
it, unreachable or 500 keeps it with "Try again"), sign-out (204 and 401 clear the token and the next
request has no `Authorization`; unreachable keeps it), no `X-Salus-CSRF` on any request, nothing printed
during a full sign-in, and `SecureTokenStore` against the plugin's mock storage.

Checks: `make mobile-analyze` no issues; `make mobile-test` 31 passed; `make gen-clients` no diff (no
`openapi.yaml` change); `make lint` 0 issues, check-migrations ok; `make archcheck` ok; `make test` ok;
`make cover` ok (total 94.8%, unchanged: no Go change). Running on a device or emulator is manual by design
(README.md, "Mobile on a device or emulator") and was not done; the API calls are covered by the widget
tests against the documented shapes, and package 9's e2e test drives the same endpoints on the real stack.

Readings:
- **Login uses `loginWithHttpInfo`.** The plain `login()` of the generated client drops the response
  headers, and a 429 needs `Retry-After` for "Try again in N minutes" (the web app's sentence). The body is
  decoded with the generated `LoginResponse.fromJson`.
- **Staff are not signed in on mobile** (SPEC §5: no practitioner or admin use of the mobile app). A login
  answering `next: enrol` stops there with "This app is for patients. Clinic staff use the Salus web app.",
  so staff never enrol from the app. A staff account with MFA is only recognisable after the code: the app
  reads `/auth/me`, ends that session at once with `POST /auth/logout` (best effort) and stores nothing. The
  token is written to secure storage only after `/auth/me` says `patient`.
- **Registration signs in straight away** with the same credentials (a new patient has no MFA, so login
  answers `done`). If that sign-in fails (for example the IP limit), the account exists, so the app goes to
  sign-in with "Your account is ready. Sign in to continue." rather than an error that would invite a second
  registration (409).
- **Sign-out that does not reach the API keeps the user signed in** with "Could not sign out. Check your
  connection and try again.", as the web app does: deleting only the local copy would leave a valid session
  the user believes is ended. A 401 counts as signed out.
- **Start-up with a stored token:** a 401 deletes the token and shows sign-in; an unreachable API or a 5xx
  keeps the token and offers "Try again", so a patient offline is not signed out.
- **Errors are fixed sentences** per API error code; a response body is never shown. An unreachable API is
  recognised by the generated client's `ApiException` carrying an inner exception.
- `flutter_secure_storage` 11.2.0 (published 2026-09-16, BSD-3-Clause) brings its platform packages and
  `path_provider` transitively (`pubspec.lock`). Its Android side needs no manifest change for this use.
- Dart code formatted with `dart format -l 120` (the app had no formatter setting; the generated client keeps
  its own).

Deferred:
- Self-service MFA enrolment for patients (`POST /auth/mfa/enrol` with a session) has no screen: the plan
  asks only for the TOTP step of a patient who enrolled elsewhere. The generated client already has the call.
- No device or emulator run (manual by design).

Dev stack left: postgres, redis and mailpit running, as found; only `flutter` and `tools` run containers were
started (and removed). No API was started, so nothing was written to the dev database. The `pub` volume now
also caches `flutter_secure_storage` and its dependencies. `deployments/.env` was not touched.

### 2026-11-04 — package 9, end-to-end

Built:
- `test/e2e/` (build tag `e2e`, the real stack through `make e2e`):
  - `accounts_test.go`, two journeys. `TestPatient_RegistersLogsInAndLogsOut`: register, login answers
    `next: done` with a token, `GET /auth/me` returns the registered patient (no clinic), logout 204, then
    `/auth/me` with the same token is 401. `TestClinic_AdminAndPractitionerSignInAndReadTheDirectory`:
    `salusctl create-clinic` (the clinic id read from its output), the admin's invite link read from
    Mailpit, `POST /invites/accept`, the first code to `/auth/mfa/confirm`, a fresh login answering
    `next: verify` and `/auth/mfa/verify` with the next code; `/auth/me` is an enrolled admin of the clinic.
    `salusctl reset-mfa <account-id>`: the earlier session is 401 and the next login answers `next: enrol`;
    the admin enrols again from that challenge (`/auth/mfa/enrol`, `/auth/mfa/confirm`) and invites a
    practitioner, who accepts, enrols and logs in with a code the same way. The practitioner reads
    `Organization/<clinic>` and `PractitionerRole?organization=<clinic>&_include=PractitionerRole:practitioner`
    returns a `searchset` with the clinic's one role and the invited `Practitioner` it references.
  - `stack_test.go`, the helpers: `waitReady` (the API's `/readyz`, as before), a bearer client of the JSON
    and FHIR APIs, `authenticator` (codes from the enrolment's base32 secret with `auth.TOTPCode`),
    `salusctl` (runs the binary, returns its output), `inviteToken` (polls Mailpit's API for the e-mail to
    the address, takes the token from the `/invite#` link, deletes the message).
  - `readyz_test.go` now calls `waitReady`.
- `Makefile`: `make e2e` depends on `up` and `tools-image`, builds `cmd/salusctl` to `/tmp/salusctl` in the
  `tools` container and passes `E2E_BASE_URL=http://api:8080`, `E2E_MAILPIT_URL=http://mailpit:8025` and
  `E2E_SALUSCTL`. salusctl uses the `tools` service's owner `DATABASE_URL`. Everything is addressed on the
  compose network, so nothing depends on the host ports or `deployments/.env`, and the CI job
  (`.github/workflows/go.yml`, `make e2e`) runs it unchanged.

Test first, failure seen before the Makefile change: the patient journey passed and the clinic journey failed
with "E2E_SALUSCTL is not set; run the end-to-end tests with `make e2e`". With the Makefile change both
pass. Lint then flagged `time.Now` in the helpers (forbidigo, `go vet -tags e2e` and golangci-lint cover
`test/`); they use `clock.System{}` as the API's integration tests do.

Checks: `make e2e` passed twice in a row (1.1 s and 1.2 s of tests; earlier runs while building it also
passed); `make lint` 0 issues, check-migrations ok; `make archcheck` ok; `make test` ok; `make cover` ok
(total 94.8%, unchanged: no production code); `make fhir-validate` ok (warnings only, as before);
`make gen-clients` no diff; `make web-lint` 0 warnings, 0 errors; `make web-typecheck` ok; `make web-test`
56 passed; `make web-build` ok; `make mobile-analyze` no issues; `make mobile-test` 31 passed. After the runs
the api, notifier and worker logs hold no `example.test` address, invite link or bearer token.

Readings:
- **Fresh synthetic e-mails per run**: `e2e-<who>-<12 hex>@example.test`, so reruns against the same dev
  database never collide (no 409, no replaced invite). The clinic is new per run, so the directory search
  has exactly one role.
- **TOTP steps.** `account.totp_last_step` rejects a step already used, so the login after enrolment uses the
  next step's code (one step of drift is accepted); the helper waits if that would be more than one step
  ahead. No sleep is needed in a normal run.
- **Login rate limit.** Every e2e login comes from the `tools` container's address; a run makes four login
  attempts (patient, admin, admin after `reset-mfa`, practitioner), all successful, against the limit of 30
  per IP in 15 minutes. About seven runs fit in a window; more fail with 429 until it passes. Not worked
  around: the limit is the API's real behaviour.
- **salusctl is built from source in the `tools` container** rather than taken from the runtime image: the
  test calls it mid-journey and reads its output, and the `tools` container already has the owner's
  connection; the code is the same commit the `salus:dev` image is built from.
- **Mailpit is left as found**: each invite e-mail the test reads is deleted. The journeys' rows stay in the
  dev database (append-only audit rows cannot be removed, and the e2e suite runs against the real stack).
- The admin's role and clinic come from `/auth/me`; the `Account` schema has no practitioner id, so the
  practitioner is checked through the role's reference and the invited names.

Deferred: none.

Definition of done (plan, SPEC §7): all nine packages committed on `feat/accounts`; `PROGRESS.md` records
every package with its readings; locally every check set passes (Go, FHIR, Clients, Web, Mobile, `make e2e`
twice). Passing in CI is confirmed on the PR into `main`, which is opened after this commit.

Dev stack left: postgres, redis and mailpit running, as found; the api, worker, scheduler, notifier, agent,
livekit, landing and migrate containers started by `make e2e` were removed again; `salus:dev` and
`salus-tools:dev` were rebuilt. Dev database still at migration 13, now also holding the e2e runs' synthetic
rows: 5 clinics "E2E synthetic clinic", 16 `e2e-*@example.test` accounts (patients, admins, practitioners)
with their sessions, accepted invites, `Practitioner` and `PractitionerRole` rows, completed `invite_email`
jobs and the audit rows. Mailpit holds only what it held before (the package 7 walk's two e-mails).
`deployments/.env` was not touched.

### 2026-11-04 — rebase on main and a flaky start-up test

Rebased `feat/accounts` on `main` at `86299fe` (the Dependabot merges, including River 0.51 and pgx 5.12).
The one conflict was `go.mod`: River 0.51 from `main` kept, `golang.org/x/crypto` from package 2 added;
`go mod tidy` made `rivertype` a direct requirement, folded into the package 5 commit that first imports it.

After the rebase, `make test -count=1` failed twice in `cmd/worker` and `cmd/notifier`
(`postgres: ping: context deadline exceeded`). Their bootstrap-era run tests gave all of `run()` a 500 ms
context, including the first Postgres connection; with this feature's test packages running in parallel on
one server, connecting can take longer. The tests now cancel `run()` once it logs "started" (up to 15 s for
start-up), through a mutex-guarded log buffer, and still assert a clean start and stop. Checked three times
with `-race`, then `make lint`, `make test -count=1` and `make cover` (94.8% total).
