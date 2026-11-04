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
No role can remove an audit row yet; the daily retention job counts rows past the period and fails, with the
count only, if there are any (see "Booking, package 1" below).
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
`delete`; applied daily by the retention job, booking package 1). Redis keys expire on their own. The TOTP
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

### Package 4: directory

**Processing added.** Tables `organization` (the clinic: name, whether active; not personal data),
`practitioner` (one name: given and family; the clinic; whether active) and `practitioner_role` (the
practitioner's membership of the clinic). Services only: reading a clinic, renaming it, reading a
practitioner or a role, and listing a clinic's roles. No endpoint serves them yet (FHIR reads come in
package 6), and no rows are created yet (invites, package 5).

**Purpose and lawful basis.** Practitioners' names and roles: the clinic's employment relationship and the
contract with the clinic (Art. 6(1)(b), (f)), to identify the practitioner to colleagues now and to patients
when booking arrives.

**Necessity and minimisation.** One name per practitioner and no contact details, qualifications, specialty
or photo. The directory knows nothing about accounts: `account.practitioner_id` links the two from the
identity side.

**Retention.** `practitioner` and `practitioner_role`: kept while active; restricted, not deleted, after the
practitioner leaves, for as long as clinical records reference them (`internal/retention`, `restrict`).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A clinic sees another clinic's staff | The service scopes every query to the caller's clinic, and Postgres row-level security shows only that clinic's rows to `salus_app`; no clinic set means no rows, and the service checks each row's clinic again |
| The application turns row-level security off | `salus_app` has no `BYPASSRLS` and owns no table; with `row_security = off` a query on these tables fails (tested) |
| A probe learns that a record exists | Patients and other clinics get "not found", not "forbidden" |
| A practitioner's details are read unnoticed | Every read and search of `Practitioner` and `PractitionerRole`, and every change of a clinic, writes an `AuditEvent` in the same transaction, also when refused; a refused or empty access records no entity, so no id from the request is stored |
| Lost updates on the clinic settings | Renaming names the version it changes; a stale version is refused |

### Package 5: practitioner invites

**Processing added.** Table `invite`: the invitee's e-mail, for a practitioner their given and family name,
the role, the inviting admin (none for the first admin, invited by the operator command), when it expires
and when it was accepted, and the SHA-256 of the link's token. The notifier sends one plain-text e-mail per
invite with the clinic's name and the link `<web>/invite#<token>`; nothing else. Accepting creates the
account (and, for a practitioner, the `Practitioner` and `PractitionerRole`) and links the account to the
practitioner (`account.practitioner_id`). `salusctl` creates a clinic with its first admin's invite and
resets a staff member's MFA. `audit_event` gains `agent_name` for the operator command, which is no account.

**Purpose and lawful basis.** The clinic's employment relationship and the contract with the clinic
(Art. 6(1)(b), (f)): to bring the clinic's staff into the platform.

**Necessity and minimisation.** Only the fields of `docs/accounts/SPEC.md` section 4. The e-mail holds no
name, role or inviter. The job carries only the invite and clinic ids. Inviting does not tell the admin
whether the e-mail already has an account (it might be a patient's); only the invitee learns it, on
acceptance.

**Retention.** `invite`: deleted 30 days after `accepted_at` or `expires_at` (`internal/retention`, `delete`;
applied daily by the retention job, booking package 1).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A link is intercepted or leaks from a log | The token is 32 random bytes in the URL fragment (never sent to a server); only its SHA-256 is stored; it works once and for at most 72 hours; a newer link replaces it |
| A clinic sees another clinic's invites | `invite` is row-level secured like the directory; the one lookup before the clinic is known is a `SECURITY DEFINER` function owned by `salus_invite_lookup` (no login, no superuser, no `BYPASSRLS`, two columns readable), with a fixed `search_path`, returning only the clinic id |
| An admin learns that someone has a (patient) account | Invite creation never checks or reports it |
| Personal data in job rows, logs or errors | Job arguments are two opaque ids; the SMTP sender keeps only the server's reply code (replies often echo the address); handler tests assert that no e-mail, name, token or password is logged |
| Header injection through the e-mail | The subject is fixed, the recipient is a validated plain address, and the sender refuses a line break in either |
| An operator action goes unrecorded | `salusctl create-clinic` and `reset-mfa` audit with `agent_name = salusctl`; `reset-mfa` also revokes the account's sessions, in the same transaction |
| E-mail to a third party without a DPA | Only Mailpit, locally; SES stays unwired until its DPA (`processors.md`) |

### Package 6: FHIR reads

**Processing added.** The directory is served over the FHIR API: `GET /fhir/R4/Organization/{id}`,
`PUT /fhir/R4/Organization/{id}` (rename, with `If-Match`), `GET /fhir/R4/Practitioner/{id}`,
`GET /fhir/R4/PractitionerRole/{id}` and `GET /fhir/R4/PractitionerRole?organization=…`, optionally with
`_include=PractitionerRole:practitioner`. A practitioner's name now leaves the database, to the staff of
their own clinic only. No new field or table.

**Purpose and lawful basis.** As package 4: the clinic's employment relationship and the contract with the
clinic (Art. 6(1)(b), (f)).

**Necessity and minimisation.** The resources carry only the elements of `docs/accounts/SPEC.md` "Directory
model" (name, active, the references, `meta`); nothing about accounts, e-mail or roles in the identity
sense. Patients do not read the directory yet.

**Retention.** Nothing new is stored but `AuditEvent` rows (kept 10 years, package 1).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Someone outside the clinic reads its staff | Every request needs a session (401 otherwise, nothing read or audited); the services refuse a patient or another clinic's staff with 404, so even existence is not disclosed; row-level security is the second guard |
| A read goes unrecorded | Each `Practitioner` read, each role read, each role a search returns, and each practitioner a search includes (read through the practitioner service) writes an `AuditEvent`; refusals too. Checked against the database in `internal/api` |
| A non-admin changes the clinic | Whether the caller is the clinic's admin is read from `account` on each request, not taken from the session; a practitioner gets 403 and the refusal is audited |
| Lost updates on the clinic settings | `If-Match` is required; a missing or stale version gets 412 |
| Names in logs or errors | OperationOutcomes carry fixed text; a 500 logs the error only; handler tests assert no name is logged |

### Package 7: web app

**Processing added.** The web app (`web/`) for clinic staff: sign in with the TOTP step or first-time
enrolment, accepting an invite, renaming the clinic, the clinic's practitioners, its open invites and the
invite form. It shows what the API already returns to that user (practitioners' names, invitees' e-mails and
names, the user's own TOTP secret while enrolling) and stores nothing new on the server.

**Purpose and lawful basis.** As packages 3 to 6.

**Necessity and minimisation.** Each screen shows only what its task needs. Patients who sign in are told
to use the mobile app and see no clinic data.

**Retention.** Nothing is kept in the browser beyond the session cookie: no `localStorage`,
`sessionStorage` or IndexedDB.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| The session token is stolen by script or a cross-site request | It is only in the `salus_session` cookie (`HttpOnly; Secure; SameSite=Strict`), never readable by the page; every write sends `X-Salus-CSRF: 1` (tested for every write) |
| The invite token stays in the browser history | It is read from the fragment once and removed from the address bar and the history entry (tested and checked in the browser) |
| The TOTP secret lingers on screen | It is held in component state only while enrolment is pending and leaves the DOM when the first code is confirmed or the set-up times out (tested and checked in the browser); the QR code is drawn locally (`qrcode-generator`), so the secret is never sent to a third party |
| Error text reveals internals | The screens map API error codes to fixed sentences and never show a response body |

### Package 8: mobile app

**Processing added.** The Flutter app (`mobile/`) for patients: registration, sign-in with the TOTP step for
a patient who has MFA, a signed-in screen with the account's e-mail, and sign-out. It calls only the
endpoints of packages 2 and 3 and stores nothing new on the server.

**Purpose and lawful basis.** As packages 2 and 3.

**Necessity and minimisation.** The app keeps one item on the device: the session token. The e-mail and
password are held in memory only while the form is open; the account is read from `/auth/me` and not
stored.

**Retention.** The token stays on the device until sign-out (or until the API refuses it, when the app
deletes it); the session itself expires after 30 days (package 3).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| The token is read from the device by another app or a backup | It is kept with `flutter_secure_storage` (Keychain on iOS, Keystore-backed on Android), never in shared preferences or a file; storage sits behind a small port so tests use a fake |
| A staff account's session lingers on a patient device | Staff are not signed in: a `next: enrol` login stops before enrolment, and a staff session opened with a code is ended at once (`POST /auth/logout`) and never stored (tested) |
| The token outlives sign-out | Sign-out revokes the session on the API and then deletes the stored token; when the API cannot be reached the app stays signed in and says so, so the session is never left valid without the user knowing (tested) |
| Credentials, codes or tokens reach a log | The app logs nothing (`avoid_print`; a test asserts nothing is printed during sign-in); error text is a fixed sentence per API error code, never a response body |

## Patients, consent and booking (`docs/booking`)

### Package 1: retention job

**What.** The retention rules are applied, not only recorded. `retention_apply` (River, default queue,
`cmd/worker`) runs daily at 03:00 UTC, enqueued by `cmd/scheduler`. It deletes sessions and invites 30 days
after they ended, does nothing for `keep` and `restrict` rules (closure, erasure and deactivation are later
features), and counts audit events past their ten years: nothing can remove them yet, so any overdue row
fails the job, with the count only. It refuses to start when a rule that removes rows has no applier.

**Who can delete.** A dedicated role, `salus_retention`: `BYPASSRLS` (invites are row-level secured by
clinic), not a superuser, owner of nothing, `SELECT` and `DELETE` on exactly the tables whose rule is
`delete` (`session`, `invite`), and no privilege on `audit_event`. Its own login, `RETENTION_DATABASE_URL`, is
given only to the worker service and loaded only by `cmd/worker`, for this job's purgers alone; the api,
notifier and scheduler never read it, and every other job runs as `salus_app` (tests in `internal/config`,
`internal/worker` and `migrations/`).

**What is processed and logged.** The job reads no personal field: each purger is one `DELETE … WHERE` on
the end timestamps. It logs the table and the counts per rule, nothing else; job arguments are empty.

**Risks and measures.** A role that bypasses row-level security could read every clinic's invites: it holds
only `SELECT`/`DELETE` on two tables, its credentials reach one process, and the SQL lives with the owning
domain (`auth`). Deletions are not audited row by row: the role cannot write `audit_event` by design, and the
deleted rows are expired credentials and invitations; the log keeps the daily counts.

**Open before production.** The retention periods (legal decision, unchanged); provisioning
`salus_retention` where `migrate` does not run as a superuser: creating a `BYPASSRLS` role needs a superuser
or a role that has `BYPASSRLS` itself (Postgres 16+), as well as `CREATEROLE`.

### Package 2: patient profile

**Processing added.** Table `patient` (FHIR `Patient`): one profile per patient account, holding the given
and family name, the birth date and an optional mobile phone (E.164), with `account_id`, `version_id` and
timestamps. Nothing else: no national ID, address or gender. The patient creates it with every required
field after registration and changes it later (`GET`/`PUT /me/patient`, patients only); there is no
incomplete row. No clinic can read it yet: staff access, through the patient's consent, comes with
package 3.

**Purpose and lawful basis.** So the clinic knows who is coming and can tell apart patients with the same
name, and the practitioner can reach the patient about a booked appointment (SPEC.md section 4). Seeking care
is health data (Art. 9): provision of health care (Art. 9(2)(h)) with the contract with the patient
(Art. 6(1)(b)); sharing with a clinic before care rests on consent (package 3). **Needs legal confirmation
before production** (SPEC.md section 3).

**Necessity and minimisation.** Four fields, each with its reason in SPEC.md section 4. The phone is
optional and never used for SMS or marketing. The response is never cached (`Cache-Control: no-store`).

**Retention.** `keep` while the account exists (`internal/retention`); erasure comes with the subject-rights
feature. **Needs a legal decision before production.**

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A patient reads or changes another patient's profile | The service works only on the profile of the signed-in account; Postgres row-level security shows `salus_app` only the row of the account (`app.account_id`) or patient (`app.patient_id`) set on the transaction, and nothing when neither is set (tested); inserting or updating another account's row is refused or touches nothing (tested) |
| Staff read a profile without a basis | `/me/patient` refuses staff (403); `Service.Read` gives staff "not found" until the consent rule of package 3 |
| A profile is read or changed unnoticed | Every read, create and update writes an `AuditEvent` in the same transaction (purpose `PATRQT` for the patient, `TREAT` for staff), also when refused; a refused access records no entity |
| Profile fields leak into logs or errors | Handlers log only through the fixed internal-error path, with the route and the error, which never holds a field; domain errors carry no value; a test sends a synthetic profile through every path and checks the log holds none of its fields |
| Lost updates | An update names the version it changes; a stale version, or a second create, is 409 |
| The application deletes a profile | `salus_app` has `SELECT`, `INSERT` and column-level `UPDATE` only; no `DELETE` or `TRUNCATE`, and the account of a profile cannot change (tested) |
