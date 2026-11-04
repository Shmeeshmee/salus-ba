# Accounts progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Privacy foundation | done, 2026-11-04 | `accounts(1)` |
| 2 Identity: accounts | done, 2026-11-04 | `accounts(2)` |
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
