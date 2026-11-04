# Accounts progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Privacy foundation | done, 2026-11-04 | `accounts(1)` |
| 2 Identity: accounts | done, 2026-11-04 | `accounts(2)` |
| 3 Identity: sessions and MFA | done, 2026-11-04 | `accounts(3)` |
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
