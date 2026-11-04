# Data protection impact assessment notes

Notes for the DPIA (GDPR Art. 35). Salus will process health data at scale, so a DPIA is required before
production. This file grows with every feature that touches personal data; each section names the feature
and package that added it.

## Accounts and the clinic directory (`docs/accounts`)

### Package 1: privacy foundation

**Processing added.** One table, `audit_event`: who accessed which account or directory record, when, with
what outcome and for what purpose (FHIR `AuditEvent`). It holds opaque ids (agent account, entity reference,
clinic), codes, the time and the request id. It holds no e-mail, name, IP address or user agent. No
personal data is collected from people yet; this package prepares the controls the later packages rely on.

**Purpose and lawful basis.** Accountability and security of processing (Art. 5(2), 32): a legal obligation
and a legitimate interest (Art. 6(1)(c), (f)). Not used for analytics, profiling or marketing.

**Necessity and minimisation.** Each column is listed in `docs/accounts/SPEC.md` section 4 with its reason.
Ids are opaque UUIDs, so an audit row says nothing about a person without the account table.

**Retention.** Ten years, then removed by whole time ranges as the schema owner
(`internal/retention/rules.go`). The period is a default that needs a legal decision before production.
archcheck fails when a table marked `personal_data: true` in `ownership.yaml` has no retention rule.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| An application bug or attacker rewrites or deletes the audit trail | `audit_event` is append-only: triggers reject `UPDATE`, `DELETE` and `TRUNCATE` for every role, the owner included, and the application role `salus_app` has only `INSERT` and `SELECT`, so it is refused by privilege before the triggers run |
| One clinic's staff see another clinic's data | The application connects as `salus_app`: no superuser, no `BYPASSRLS`, owner of nothing, so row-level security binds it and cannot be switched off (`row_security = off` makes a guarded query fail). The clinic is set per transaction (`postgres.SetLocal`); when it is unset nothing is visible. Proven by tests on a test-only table now, on the directory tables in package 4 |
| An access goes unrecorded | `auditevent.Service.Record` joins the caller's transaction, so the access and its record commit or roll back together |
| Personal data leaks into logs or errors | Audit rows and error messages carry ids and codes only; tests check that errors do not echo values |
| The database credentials of the application are over-privileged | Only `migrate` (and later `salusctl`) connect as the owner; `api`, `worker`, `notifier` and `scheduler` connect as `salus_app` with grants per table |

**Open before production.** The retention periods (legal decision), the hosting provider and its DPA, and
the KMS (`processors.md`).

### Package 2: accounts

**Processing added.** Table `account`: the e-mail (the login), the argon2id hash of the password, the role,
and for staff the clinic. Columns for MFA (`totp_secret`, `totp_last_step`, `mfa_enrolled_at`) and the
practitioner link exist but stay empty until packages 3 to 5. Patients register themselves
(`POST /auth/register`); `GET /auth/me` returns one's own account. No health data. A patient account on a
telemedicine service can still hint at a health matter, so patient accounts are treated as sensitive.

**Purpose and lawful basis.** Performance of the contract the user signs up for (Art. 6(1)(b)).

**Necessity and minimisation.** Only the fields of `docs/accounts/SPEC.md` section 4. No name, phone, birth
date, IP address or device data. The API returns the account's id, e-mail, role, clinic and whether MFA is
set up, never the hash.

**Retention.** Kept while the account exists (`internal/retention/rules.go`, `keep`); closure and erasure
come with the subject-rights feature.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Stolen database reveals passwords | Only an argon2id hash is stored (m = 32 MiB, t = 4, p = 1, above the OWASP floor; 16-byte random salt), re-hashed on login when the parameters change |
| A burst of logins exhausts memory | At most four hashes run at once (128 MiB); the rest wait, bounded by the request timeout |
| Login timing reveals which e-mails have accounts | An unknown e-mail costs one hash, like a known one, and gets the same error |
| Registration reveals that an e-mail has an account (409) | Accepted in `SPEC.md`; the login rate limit (package 3) bounds probing; hiding it needs e-mail verification (out of scope) |
| An account access goes unrecorded | Registration (also a taken e-mail), each password check (success, wrong password, unknown e-mail) and reading one's own account each write an `AuditEvent` in the same transaction |
| E-mails or passwords reach logs or error text | Logs carry the route, status and request id; errors carry fixed codes; a handler test asserts the log holds no e-mail, password or hash |
| The application deletes accounts or audit rows | `salus_app` has `SELECT, INSERT, UPDATE` on `account`, no `DELETE` or `TRUNCATE` |

### Package 3: sessions and MFA

**Processing added.** Table `session`: the account, for staff the clinic, the SHA-256 of the session token
(the token itself is sent to the client once and never stored), whether the sign-in passed a TOTP code, and
when it was created, expires and was revoked. On `account`, the TOTP secret (sealed with AES-256-GCM, bound
to the account id), the last used TOTP step and the enrolment time. In Redis, with a TTL: the MFA challenge
(`mfa:challenge:<sha256>` → account id and a count of wrong codes, 5 minutes) and the login rate-limit
counters (`ratelimit:login:email:<hmac>`, `ratelimit:login:ip:<hmac>`, 15 minutes). Endpoints: login, MFA
enrol, confirm and verify, logout; a session middleware reads the session row on each request.

**Purpose and lawful basis.** Sessions and MFA: performance of the contract (Art. 6(1)(b)). Rate limiting:
security of processing (Art. 32), legitimate interest (Art. 6(1)(f)).

**Necessity and minimisation.** No IP address, user agent or device data is stored with a session. The
client IP is used only to name a rate-limit counter, by HMAC (`RATE_LIMIT_KEY`), so neither it nor the
e-mail is readable in Redis; it is never logged. The `otpauth://` URI carries the e-mail as its label, to
the account holder only, and is not cached (`Cache-Control: no-store`).

**Retention.** Sessions are deleted 30 days after they were revoked or expired (`internal/retention`,
`delete`; applied by the retention job, the next privacy feature). Redis keys expire on their own. The TOTP
columns follow the account.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A stolen database gives working sessions | Only the SHA-256 of each 32-byte random token is stored |
| A stolen database gives working TOTP secrets | Secrets are sealed with AES-256-GCM, the account id as additional data; the key is outside the database (`TOTP_KEY` locally, a KMS in production, not decided yet) |
| A signed-out or revoked session keeps working | Every request reads the session row (`revoked_at IS NULL AND expires_at > now`), so logout takes effect on the next request; there is no cached copy in Redis |
| Password guessing | 5 failures per e-mail and 30 attempts per IP in 15 minutes; over the limit the password is not checked (429, `Retry-After`) |
| TOTP guessing or replay | Five wrong codes end the challenge; a code is accepted one step either way and only once (`totp_last_step`) |
| Staff without a second factor | Staff get no session before a TOTP code: login answers `next: enrol` or `verify`, and a check constraint refuses a staff session that is not MFA-verified |
| Cross-site requests with the web app's cookie | The cookie is `HttpOnly; Secure; SameSite=Strict`, and a write with it needs `X-Salus-CSRF: 1` (403 otherwise) |
| A login burst starves other requests | The password hash runs outside the database transaction, so a waiting login holds no connection (`docs/accounts/PROGRESS.md`, package 3) |
| A spoofed `X-Forwarded-For` evades the IP limit | The header is read only from proxies in `TRUSTED_PROXY_CIDRS` |
| An access goes unrecorded | Each login attempt that checks a password, each enrolment step, each code verification and each logout writes an `AuditEvent` in the same transaction; a rate-limited attempt or an unknown challenge touches no account and is not audited |
| Tokens, codes or e-mails reach logs | Handler tests assert the log holds no e-mail, password, token, challenge or code |

**Open before production.** The KMS for the TOTP key, and `TRUSTED_PROXY_CIDRS` for the production load
balancer.
