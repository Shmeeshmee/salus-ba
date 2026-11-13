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

**Open before production.** The retention periods (legal decision) and the hosting provider's DPA
(`processors.md`). The keys: a root-only key file on the encrypted volume, no KMS (ADR 0011 point 3).

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
| A stolen database gives working TOTP secrets | Secrets are sealed with AES-256-GCM, the account id as additional data; the key is outside the database (`TOTP_KEY` locally; in production a root-only key file on the encrypted volume, ADR 0011 point 3) |
| A signed-out or revoked session keeps working | Every request reads the session row (`revoked_at IS NULL AND expires_at > now`), so logout takes effect on the next request; there is no cached copy in Redis |
| Password guessing | 5 failures per e-mail and 30 attempts per IP in 15 minutes; over the limit the password is not checked (429, `Retry-After`) |
| TOTP guessing or replay | Five wrong codes end the challenge; a code is accepted one step either way and only once (`totp_last_step`) |
| Staff without a second factor | Staff get no session before a TOTP code: login answers `next: enrol` or `verify`, and a check constraint refuses a staff session that is not MFA-verified |
| Cross-site requests with the web app's cookie | The cookie is `HttpOnly; Secure; SameSite=Strict`, and a write with it needs `X-Salus-CSRF: 1` (403 otherwise) |
| A login burst starves other requests | The password hash runs outside the database transaction, so a waiting login holds no connection (`docs/accounts/PROGRESS.md`, package 3) |
| A spoofed `X-Forwarded-For` evades the IP limit | The header is read only from proxies in `TRUSTED_PROXY_CIDRS` |
| An access goes unrecorded | Each login attempt that checks a password, each enrolment step, each code verification and each logout writes an `AuditEvent` in the same transaction; a rate-limited attempt or an unknown challenge touches no account and is not audited |
| Tokens, codes or e-mails reach logs | Handler tests assert the log holds no e-mail, password, token, challenge or code |

**Open before production.** `TRUSTED_PROXY_CIDRS` for the production load
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
deleted rows are expired credentials and invitations. Each run writes one `AuditEvent` per table it deletes
from, as `salus_app`, with the job as agent, the table name and the row count (owner's decision on review,
2026-11-05); the log keeps the same counts.

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

### Package 3: consent and access

**Processing added.** Table `consent` (FHIR `Consent`): one row per patient and clinic each time the patient
consents to share their profile and appointments with that clinic for care, with `patient_id`,
`organization_id`, `status` (`active` | `inactive`), `date_time` (granted), `revoked_at`, `version_id` and
timestamps. The patient grants (`POST /consents`), revokes (`POST /consents/{id}/revoke`) and lists
(`GET /me/consents`) their own; staff cannot. At most one is active per clinic; granting again after a
revocation adds a row, so the history is kept. The pure `access` package decides what a viewer may see of a
profile or an appointment (SPEC.md "Access rule"); the patient service now lets staff read a profile only at
`Full` access, which needs an appointment with the practitioner, so no staff read succeeds before
appointments exist (package 6).

**Purpose and lawful basis.** The consent is the basis for sharing the profile before care and for new
bookings (Art. 9(2)(a)); the row is the proof of the decision, when it was given and withdrawn (Art. 7(1)).
Care records rest on Art. 9(2)(h): after a revocation the clinic keeps the name, times and status of past
appointments only (package 6). **Needs legal confirmation before production** (SPEC.md section 3).

**Necessity and minimisation.** Only who, with which clinic, when granted and when revoked (SPEC.md section
4); no free text. Responses are never cached (`Cache-Control: no-store`): a consent shows whom the patient
consults.

**Retention.** `keep`, with the clinical record, as proof (`internal/retention`). **Needs a legal decision
before production.**

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A clinic sees a patient without consent | Row-level security shows a clinic's transaction (`app.organization_id`) a `patient` row only while an active consent for that clinic exists, and `consent` rows only of that clinic, read-only (tested); the service also checks `access` before it reads the profile |
| Access outlives the revocation | Revocation sets `inactive` and `revoked_at` in one transaction; row-level security and `access` read the consent's state, so the clinic loses the row when that transaction commits (tested at the database and the repository) |
| A patient's transaction sees another patient | The clinic branches of `patient` and `consent` apply only while neither `app.patient_id` nor `app.account_id` is set, so a patient's transaction that also sets a clinic sees only that patient (tested) |
| A patient changes another patient's consent, or a clinic changes any | The patient branch covers only `patient_id = app.patient_id`; the clinic branch is `SELECT` only; `salus_app` may update only `status`, `revoked_at`, `updated_at` and `version_id`, and has no `DELETE` or `TRUNCATE` (tested) |
| A consent is granted, revoked or read unnoticed | Every grant, revocation, list and read writes an `AuditEvent` in its transaction (`PATRQT` for the patient, `TREAT` for staff), refusals included; a grant to an unknown clinic, which aborts its transaction, is audited in a transaction of its own; the profile read that finds the calling patient is audited by the patient service |
| Consent details leak into logs or errors | Handlers log only through the fixed internal-error path; domain errors carry no value; a test runs the whole flow and checks the log holds no profile field or e-mail |

### Package 4: directory additions

**Processing added.** A clinic is listed in a public directory or not (`organization.listed`, off by
default) and has a time zone (`organization.time_zone`, IANA, default `Europe/Sarajevo`); a practitioner role
has a specialty (`practitioner_role.specialty_code`, one of twelve SNOMED CT codes from the R4
`c80-practice-codes` value set), set by the clinic's admin (`PUT /practitioner-roles/{id}/specialty`). The
services can now read, for anyone, the listed clinics, the active roles of a listed clinic with their
specialty, and the names of those roles' practitioners. No endpoint serves the public reads yet (FHIR in
package 7).

**Purpose and lawful basis.** So patients can find a clinic and the right practitioner: the clinic's
employment relationship with the practitioner and its legitimate interest in being found (Art. 6(1)(b), (f));
the clinic decides when it is listed (SPEC.md section 3).

**Necessity and minimisation.** The public reads return a clinic's name, time zone and listing, a role's
clinic, practitioner and specialty, and a practitioner's name: nothing else (no clinic, contact details or
qualifications for the practitioner; tested on the public queries' results). An unlisted clinic and inactive
practitioners and roles are not shown.

**Retention.** No new table: the specialty follows `practitioner_role` (`restrict`).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| An unlisted clinic or its practitioners are published | The directory scope (`app.directory = 'public'`) has `SELECT`-only policies that show only listed clinics and the active practitioners and roles of listed clinics; unset or any other value shows nothing (tested at the database and in each repository); the services check again what the rows carry (`listed` on a clinic, `active` on a practitioner or role) |
| The public directory changes data | Its policies are `SELECT` only; an update in the directory scope changes no row (tested) |
| A non-admin changes a specialty or the listing | Whether the caller is the clinic's admin is read from `account` on each request; a practitioner gets 403, anyone outside the clinic 404, and the refusal is audited; `salus_app` may update only `listed` and `time_zone` (with the name) on `organization` and `specialty_code`, `updated_at` and `version_id` on `practitioner_role` (tested) |
| Public reads go unrecorded | Deliberate: they hold only what a clinic chose to publish, and auditing anonymous traffic would turn each request into a write (SPEC.md "Public browse"). Staff reads and every change stay audited |

### Package 5: schedules and slots

**Processing added.** A practitioner role has a schedule (`schedule`): a slot length and weekly availability
rules (day, start, end in the clinic's time zone), edited by the practitioner for their own role or by the
clinic's admin (`GET`/`PUT /practitioner-roles/{id}/availability`). The scheduler's `slots_generate` job
materialises the schedule as bookable slots (`slot`: schedule, clinic, period, `free` or `busy`) 28 days ahead,
hourly and after each edit. The services can read, for anyone, the schedules and free future slots of listed
clinics; no endpoint serves those public reads yet (FHIR in package 7). Appointments that hold a slot arrive
in package 6.

**Purpose and lawful basis.** To offer bookable times: the clinic's employment relationship with the
practitioner and its legitimate interest in being booked (Art. 6(1)(b), (f); SPEC.md section 3). A schedule
reveals a practitioner's working hours; a slot's status reveals whether a time is taken, not by whom.

**Necessity and minimisation.** A schedule holds the role, the clinic, a length and the rules; a slot its
schedule, clinic, period and status (SPEC.md section 4). No patient data: who holds a busy slot is the
appointment's (package 6). The public directory sees the schedules of active roles of listed clinics and only
their free slots; a busy slot is not shown outside the clinic. The `slots_generate` job's arguments are the
schedule and clinic ids, or nothing for the hourly run; its log holds counts only (tested).

**Retention.** `schedule`: `restrict`, like `practitioner_role` (no applier until practitioners can leave).
`slot`: `delete` free slots 30 days after their end, by the `retention_apply` job as `salus_retention`, which
is granted `SELECT, DELETE` on `slot` in the same migration; a busy slot is kept with its appointment.
**Needs a legal decision before production**, like every period here.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A practitioner sees or changes another practitioner's working hours | The service allows the role's own practitioner (the account's `practitioner_id`, read per request) and the clinic's admin only; another member gets 403, anyone outside the clinic 404; reads and edits are audited, refusals included (tested) |
| Another clinic's schedules or slots are read or changed | Both tables are row-level secured by clinic, fail closed when no scope is set, and the services check the clinic again (tested) |
| Booked times leak through the public directory | The directory policy on `slot` shows `free` slots only, of schedules its own policy shows (active roles of listed clinics); directory reads change nothing (tested) |
| Generation removes a booking | Generation deletes only future free slots; a `RESTRICTIVE` policy lets `salus_app` delete free slots only, and an overlapping new slot is skipped (`ON CONFLICT DO NOTHING` on the exclusion constraint), so a busy slot and its appointment stay (tested) |
| The retention role reaches more than slots | It gains `SELECT, DELETE` on `slot` only, which the grant test ties to the `delete` rules; it still cannot read `schedule` or touch `audit_event` (tested) |

### Package 6: appointments

**Processing added.** A patient books a free slot of a listed clinic (`appointment`: patient, practitioner
role, clinic, slot, period, status, and on cancellation when, why as a code (`pat`, `prov`) and by which
account). The patient cancels their own up to 24 hours before the start; the appointment's practitioner and
the clinic's admin cancel at any time; the practitioner marks it fulfilled or no-show once it has started
(`POST /appointments`, `/appointments/{id}/cancel`, `/appointments/{id}/status`). The notifier e-mails the
patient a confirmation, a cancellation, and a reminder 24 hours before. Revoking a consent now enqueues
`consent_revoked`, whose worker cancels the patient's future appointments with that clinic, frees their slots
and e-mails each cancellation. Reads (`Read`, `Search`) are built in the service; their FHIR endpoints arrive
in package 7.

**Purpose and lawful basis.** The booking is the care relationship: Art. 9(2)(h) for the care record, with
Art. 6(1)(b); booking itself requires the patient's active consent with the clinic (Art. 9(2)(a), SPEC.md
section 3). E-mails about the appointment: the contract with the patient (Art. 6(1)(b)). **Needs legal
confirmation before production.**

**Necessity and minimisation.** The table holds the fields of SPEC.md section 4 and nothing else: no reason
for the visit, no free text; the cancellation reason is who cancelled. The JSON answers carry neither the
patient nor who cancelled. Staff see an appointment through `access`: with an active consent its practitioner
and the clinic's admin see all of it; after a revocation they see a past appointment's times, status and the
patient's name only, and nothing future. The patient's name is given with an appointment only at a level
`access.Profile` allows, and that read is audited as a `Patient` read. Job arguments are the appointment and
clinic ids (e-mails) or the patient and clinic ids (`consent_revoked`); logs hold counts and opaque ids only
(tested). E-mails hold the clinic's name, the date and time in its zone with the zone's name, and the link
`<PATIENT_APP_URL>/appointments` (golden texts, tested); never the practitioner, the specialty or anything
clinical.

**Access.** Row-level security on `appointment`: the patient's own scope; a clinic's scope with an active
consent, or, after a revocation, for an appointment that started before the latest one; changes by a clinic
with an active consent, and after a revocation only marking such a past appointment `fulfilled` or `noshow`
(the care record, Art. 9(2)(h); cancelling it is refused by `WITH CHECK`, tested); no booking in a clinic's
scope. The `patient` table gains the clinic's
past-appointment branch (the name is the service's to give). The notifier's e-mail jobs read in the clinic's
scope narrowed to the one appointment the job names (`app.appointment_id`), and that appointment's patient
row: a cancellation after a revocation must still reach the patient, though the clinic itself no longer sees
it. Staff transactions never set that setting. All of it fails closed when no scope is set (tested).

**Retention.** `appointment`: `keep` with the clinical record. **Needs a legal decision before production.**
No `DELETE` grant to any role; the slot of a cancelled appointment may be deleted later as a free slot, so
`slot_id` has no foreign key.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| The clinic keeps seeing a patient after a revocation | Access ends when the revocation commits: row-level security and `access` read the consent's state, not the appointments' (tested before the job runs: no profile, no future appointment; the past one at the name only) |
| Future appointments survive a revocation | `consent_revoked` is enqueued in the revocation's transaction (a rollback leaves no job, tested); its worker cancels them in one transaction with their slots and e-mails, and a retry changes nothing (tested) |
| A double booking | The slot is held (free to busy) in the booking's transaction, and the database's exclusion constraint refuses overlapping appointments of one role; concurrent bookings give one booking (tested); book and cancel run concurrently for many rounds and the slots never disagree with their appointments (tested) |
| A patient's appointments are seen by another patient or clinic | Row-level security per scope, checked again by the service, `access` per viewer; refusals are audited (tested) |
| An e-mail says more than it should | Fixed texts with the clinic, time and link only (golden files); the recipient is read through `auth` with the job as agent, audited; SMTP errors keep the reply code only |
| A reminder or confirmation of an appointment that changed | The job reads the appointment when it runs and sends nothing unless it is still in the status the e-mail is about (tested) |

### Package 7: FHIR reads

**Processing added.** The booking resources are served over the FHIR API: `Patient` and `Consent` read,
`Appointment` read and search (`patient`, `practitioner`, `date`, `status`), and the public directory without a
session: `Organization?listed=true`, a listed clinic's `PractitionerRole` search with its practitioners' names
and specialty, `Schedule` read and search by `actor`, and the free future `Slot`s of a schedule. The clinic's
admin lists the clinic and sets its time zone through the `Organization` update (two extensions). No new
field or table.

**Purpose and lawful basis.** As packages 2, 3, 4 and 6: the care relationship and the patient's consent for
patient data; the clinic's legitimate interest in being found for the public directory.

**Necessity and minimisation.** Each resource carries the elements of SPEC.md section 2 only. `Patient` goes to
the patient and to a practitioner whom `access` gives all of the profile; the clinic's admin gets 404. An
`Appointment` carries the patient's name only at a level `access.Profile` allows; after a revocation a past
appointment shows the name, times and status, without its slot, creation time or cancellation details. The
public reads show what a listed clinic publishes: its name and zone, its active roles with specialty, their
practitioners' names, schedules and free slots.

**Retention.** Nothing new is stored but `AuditEvent` rows (kept 10 years).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Patient data reaches someone outside the care relationship | The services decide through `access`, row-level security is the second guard; the access matrix (patient, their practitioner, another practitioner, the admin, another clinic, nobody, and the clinic after a revocation) is tested through HTTP on the application login |
| A read goes unrecorded | Every `Patient`, `Consent` and `Appointment` read audits, refusals included; a search audits each appointment returned and each name given, or one event when it returns none (tested against the database) |
| The public directory shows more than the clinic chose | Public reads run in the directory scope (listed clinics, active roles, free slots) and are not audited (tested: no event); a clinic that is not listed has no roles in the search |
| Patient data is cached by a client or proxy | `Cache-Control: no-store` on `Patient`, `Consent` and `Appointment` answers |
| Names in logs or errors | OperationOutcomes carry fixed text; a 500 logs the error only; handler and app tests assert the log holds no profile field |

### Package 8: web app

**What it is.** The staff web app gains a practitioner's week calendar (their own; the clinic's admin picks a
practitioner), the appointment detail with the outcome and cancel actions, the working-hours editor, the
specialty editor and, in clinic settings, the listing and the time zone. It reads through the FHIR reads and
searches of package 7 and writes through the JSON actions. Two small additions to the API: `GET /specialties`
(the fixed public list of specialties, no personal data, not audited) and `practitionerId` on `GET /auth/me` for
a practitioner account (an opaque id, so the calendar can search by practitioner). No new field or table.

**Purpose and lawful basis.** As packages 6 and 7: care records and the patient's consent.

**Necessity and minimisation.** The calendar shows times, status and the patient's name only where the
appointment carries it. The detail shows the profile only when the `Patient` read answers (a practitioner with
full access); otherwise the name the appointment gives, or none. The birth date and phone sit behind a
"Patient details" disclosure, not on the screen by default. The admin's detail never asks for the `Patient`
(the API answers it 404 to an admin, SPEC "FHIR reads and searches"), so no refused read is audited for nothing.

**Retention.** Nothing is stored in the browser: no local or session storage, the session is the `HttpOnly`
cookie; the API's `Cache-Control: no-store` keeps patient answers out of the browser cache.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A profile shown to someone the access rules exclude | The screen shows only what the API returns; the API decides through `access` (package 7). Vitest covers the profile hidden on a 404 and the admin's detail without a `Patient` read; the browser walk confirmed no `Patient` request from the admin |
| Patient data on a shared screen | Secondary fields behind a disclosure; the calendar shows the name and times only |
| Personal data in logs | No logging in the web app; the walk's api, notifier, worker and dev-server logs held no name, address or phone |

### Package 9: mobile app

**What it is.** The patient app gains the profile (completed after registration and before the first booking,
editable after), the listed clinics, a clinic's practitioners with specialty, free times by day in the clinic's
time zone, the consent screen on the first booking with a clinic, booking, the patient's appointments with
cancellation until 24 hours before the start, and the patient's consents with revocation. It reads through the
public FHIR searches and the patient's own `Appointment` search (package 7) and writes through the JSON actions
of packages 2, 3 and 6. No change to the API, no new field or table.

**Purpose and lawful basis.** As packages 2, 3 and 6: the profile and new bookings rest on the patient's consent
per clinic, given on the consent screen; past appointments are the clinic's care records (`SPEC.md` §3).

**Consent as given.** The screen names the clinic and says, before the patient agrees: what is shared (the profile:
name, birth date, the phone if given; the appointments with that clinic), who sees it (the practitioner booked
sees the profile, the clinic's administrators the name and times), what for (care at that clinic), and how to
revoke (under "My consents"; the profile is no longer visible and future appointments there are cancelled, past
ones stay as care records). Nothing is pre-ticked: the consent is granted only on "Agree and book", and "Not now"
books nothing. Revoking asks for a confirmation that repeats these consequences.

**Necessity and minimisation.** The device keeps only the session token, in the platform's secure storage
(package 8 of accounts). The profile, appointments and consents are held in memory while a screen shows them
and are not written to the device. The directory reads (clinics, practitioners, slots) are public data the
clinics chose to publish.

**Retention.** Nothing new on the device beyond the token; on the server as packages 2, 3 and 6.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Consent that is not informed or not explicit | A separate screen before "Agree and book": what is shared, who sees it, what for, who is responsible (the clinic, with Salus processing on its behalf) with a link to the privacy notice (`PRIVACY_NOTICE_URL`, a placeholder until published), and how to revoke; widget tests check the text, the link, that the screen comes before any grant and that declining grants and books nothing. The text needs legal review before production |
| A revocation whose effect surprises the patient | The consent screen and the confirmation dialog both say that the clinic keeps the patient's name and the details of appointments that have already taken place, as its care records, and that future appointments are cancelled; tested |
| Personal data in logs or on screen from an error | The app prints nothing (a widget test captures `debugPrint` through a whole booking); API errors map to fixed sentences, never the response body |
| Patient data left on the device | Nothing but the session token is stored, in secure storage |

## Video consultations (`docs/consultation`)

### Package 1: videosession

**Processing added.** The first token request for an appointment's consultation stores its `video_session`:
the appointment and clinic ids, a random room name and two random participant identities (`r_…`, `p_…`, 128
bits from `crypto/rand`, derived from no id of ours), the join window's close (the appointment's end + 30
minutes), and later who joined when and how it ended. `videosession.Service.Join` issues a 5-minute LiveKit
token for that one room to the appointment's patient or its practitioner only; its HTTP endpoint arrives in
package 4. The room is created on our own LiveKit (`deployments/livekit.yaml`) with no metadata and at most
two participants; the token carries the identity and the grant only, no name, metadata or attributes (tested
against the compose LiveKit). The other person's name comes from our API: the practitioner's public name for
the patient, the patient's name at the access level for the practitioner (audited as a `Patient` read).

**Purpose and lawful basis.** Running the consultation, part of the care: Art. 9(2)(h), with the active
consent for the sharing before care (SPEC.md section 4), checked for both participants. **Needs legal
confirmation before production.**

**Necessity and minimisation.** The fields of SPEC.md section 5; `closes_at` is there so the retention role,
which may read only `video_session`, can delete a session that never ended. LiveKit holds only the random
names and the live media, in memory; nothing is recorded (ADR 0006). Errors of the provider keep LiveKit's
error code only, never a room, identity or token (tested).

**Access.** Row-level security on `video_session`: the patient's own; a clinic's with an active consent or,
after a revocation, for an appointment that started before it; the webhook's `app.video_room` (package 3)
sees and updates the one session of its room and creates none. `salus_app` cannot delete or change what a
session is for (tested). Every join, issued or refused, writes an `AuditEvent` (`operation`, action `E`).

**Retention.** `video_session`: `delete` 30 days after `ended_at`, or after the window's close for a session
that never ended, by the retention job (`salus_retention`: `SELECT, DELETE` on this table only). **Needs a
legal decision before production.**

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| LiveKit (or its logs) learns who is in a call | Random room and identities, no name, metadata or attributes in the token or the room (tested); JSON logs at info |
| Someone else joins the call | The token grants one room, for 5 minutes, to the patient's or the practitioner's fixed identity; `max_participants` 2; a repeated identity replaces the earlier connection (tested); `room.auto_create` off, so a token cannot create a room (tested) |
| A join without consent or outside the window | Checked in the service before any session or token; refusals audited (tested) |

### Package 2: encounter

**Processing added.** The FHIR `Encounter` of a consultation (`encounter` table): the appointment, the clinic,
the patient and the practitioner role (references copied once from the appointment when it starts, never
changed), the status (`in-progress`, `finished`) and the period. It holds no clinical content, only that a
consultation took place and when. It is started when the practitioner first joins (the webhook, package 3)
and finished when the consultation ends; a finished encounter never changes. Finishing it records the
appointment's outcome: a `booked` appointment becomes `fulfilled` when the patient joined, else `noshow`; a
status set by hand or a cancellation is kept. The `consultation_window_close` job (session and clinic ids
only) ends a session at the window's close when the room is empty, when the practitioner never joined
(`practitioner_absent`: no encounter, the appointment stays `booked`, `patient_joined_at` is kept), or at the
hard cap 2 hours after the scheduled end whoever is in the room (`time_limit`), and then deletes the room.

**Purpose and lawful basis.** The care record that the consultation happened (Art. 9(2)(h)), as for the
appointment. **Needs legal confirmation before production.**

**Access.** Row-level security on `encounter` as on `video_session`, without a webhook scope: the patient
reads their own and never writes; a clinic reads and writes those of its appointments with an active consent
or, after a revocation, of those that started before it. `salus_app` cannot delete an encounter or change
whose it is or when it started (tested). Reads follow `access.Appointment` (name level or more); a
practitioner searches only their own role's. Every read, every returned resource and every lifecycle write
(the start, the finish, the session's end, the appointment's outcome) writes an `AuditEvent`; the job's
writes have the agent `consultation_window_close`, purpose `HOPERAT`. The job's check of an occupied room
(once a minute) reads only the session's own row and writes nothing, so it is not audited.

**Retention.** `encounter`: `keep`, with the clinical record. **Needs a legal decision before production.**

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A call left open indefinitely | The hard cap ends and deletes the room 2 hours after the scheduled end, whoever is in it (tested at the cap ± 1 s) |
| The job's log reveals who was in a call | It logs the session id and the snooze only; tested to hold no name, e-mail, room or identity |
| A no-show recorded against a patient because the practitioner never came | No encounter and no outcome when the practitioner never joined; the appointment stays `booked` for the clinic (tested) |

### Package 3: webhooks

**Processing added.** LiveKit's webhooks reach the api at `POST /webhooks/livekit`, on the compose network
locally (`deployments/livekit.yaml`). An event carries only what LiveKit knows: the random room, the random
identity, its kind and LiveKit's times. The api records each participant's first join
(`video_session.patient_joined_at`, `practitioner_joined_at`), starts the `Encounter` when the practitioner
first joins, and, after the join window has closed, ends the consultation when the last one leaves. Nothing
new is stored: these are the columns of packages 1 and 2.

**Access.** The webhook carries no user. Its signature (a JWT of the API key pair carrying the body's SHA-256,
unexpired) is verified before the body is read; anything else is 401 and handles nothing (tested). Its
transaction sees only the one session of its room (`app.video_room`) and then the session's clinic, as a
clinic's scope does. Its writes are audited with the agent `livekit_webhook`, purpose `HOPERAT`: each join it
records, the `Encounter`'s create, the session's end, the `Encounter`'s finish and the appointment's outcome.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A forged or replayed webhook changes a consultation | Signature, body hash and expiry checked (tested for a wrong key, wrong secret, changed body, missing header and expired token); every effect is a change guarded in SQL, so a repeat or a late event changes nothing (every order of the five events, each delivered twice, tested on the real database) |
| A patient joins with a token issued just before a revocation or a cancellation committed | Every patient join checks the consent and the appointment's status again and removes the patient from the room when either is gone (tested, also on the real database after a revocation) |
| The webhook's log reveals who was in a call | It logs the request id and the event kind only; an event for a room we do not know is logged as `unknown room` with nothing from the body (tested) |

### Package 4: API

**Processing added.** No new table or column. Three JSON endpoints and the FHIR `Encounter` read and search
expose what packages 1 to 3 store. `POST /appointments/{id}/consultation/join` hands the appointment's patient or
its practitioner a LiveKit token and the other person's display name (the practitioner's public directory name
for the patient; the patient's name at the appointment's access level for the practitioner, audited as a
`Patient` read). `GET /appointments/{id}/consultation` shows the patient, the practitioner and the clinic's admin
whether and when each side joined and when and why the consultation ended (the owner's addition: the clinic
sees whether the patient came when the practitioner did not). `POST /appointments/{id}/consultation/end` lets the
practitioner end it. `Encounter` is read and searched as its appointment is seen (FHIR R4, class `VR`).

**Purpose and lawful basis.** As packages 1 to 3: running the consultation and its care record (Art. 9(2)(h),
with the active consent for joining). The status and the Encounter carry no clinical content, only that a
consultation took place, who took part (as references) and when.

**Access.** In the services: the join to the appointment's patient and its practitioner only (the admin
included is 404); the status to whoever `access.Appointment` gives the name level or more; the end to the
appointment's practitioner only, also after a revocation, so they can close the call they are in. Every call is
audited in its own transaction, refusals included: the join and the end as an `operation` on the `Appointment`,
the status as a `read` of it, each `Encounter` read and returned resource as a `read` or `search-type`. The writes
an end causes are audited with the agent `consultation_end`, purpose `HOPERAT`.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A token, a room or a name leaks through the logs | The handlers log nothing on success and only the error on a 500; the access log carries the route pattern, not the path; tested to hold no token or name (handler and `api.New` tests), and the api's log after 3,000 joins in the stack held none |
| Anyone other than the participants learns that the patient attended | The status is 404 for anyone `access.Appointment` gives nothing, audited without the entity; the join and the end are 404 for anyone but the participants (tested per caller) |
| A response is cached by a shared cache | Every consultation response is `Cache-Control: no-store` (tested) |

### Package 5: Revocation and cancellation during a call

**Processing added.** No new table or column. Every cancellation (the patient's, staff's, or the revocation
job's) also enqueues `consultation_access_end` in its transaction, with the appointment's and the clinic's ids
only; its worker removes the patient from the appointment's video room and ends the session (`cancelled`),
which deletes the room. After a revocation, the `consent_revoked` job, once it has cancelled the appointments
that had not started, reads the patient's sessions with that clinic whose room may still be open (in the
patient's own scope) and removes the patient from each; the practitioner stays in the room of a consultation
that started before the revocation and ends it as usual. The session of an appointment the revocation cancelled
is ended there in its room's scope (`app.video_room`), because the clinic's scope no longer shows it.

**Purpose and lawful basis.** Giving effect to a revoked consent and to a cancellation at once (GDPR Art. 7(3)
for the consent): no further token is issued from the commit, and the patient is taken out of a call already
running as soon as the worker picks up the job.

**Access.** The jobs read only the sessions and appointments they act on, in the narrowest scope that shows
them; their writes (the session's end, the `Encounter`'s finish) are audited with the job's kind as agent,
purpose `HOPERAT`. Changed in package 5.1: the removal is audited too (below).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A patient stays in a call after revoking or after a cancellation | The token is refused from the commit (tested through `api.New` with the job still queued); the patient is removed by the job (tested on the real LiveKit: the patient disconnected, the practitioner still in the room after a revocation, both gone after a cancellation); in the stack the patient was disconnected 5 to 27 ms after the revoke's response (`make bench-removal`) |
| A failed removal leaves the patient in the room | The job fails and River retries it; a failure in one room does not stop the removal from the others; a join on a token issued before the commit is removed by the webhook (package 3) |
| The jobs or their logs carry personal data | The job arguments are opaque ids (tested); the `consultation_access_end` log line carries the appointment's id only, the revocation's the count (tested to hold no name, e-mail, room or identity; the stack's logs after the measurement held none) |
| An `Encounter` started before the start of an appointment the revocation cancels stays `in-progress` | Closed in package 5.1: the room's scope finishes it (below) |

### Package 5.1: the owner's changes after package 5

**Processing added.** No new table or column. A new row-level security policy on `encounter`
(`migrations/00023_encounter_room_scope.sql`): a transaction with a room set (`app.video_room`, no patient or
account set) sees and updates the one encounter of the appointment whose session has that room, as the webhook
scope on `video_session` does; it never creates one. The end of a session, which already runs in its room's
scope, therefore finishes the `Encounter` of an appointment a revocation cancelled after the practitioner
joined, with `period.end` set, although the clinic's scope no longer shows it. The appointment stays cancelled.

**Access.** The room is a random name known only to the api, the worker and LiveKit; the scope shows nothing
more than the session of that room already does (tested as `salus_app`: one encounter per room, nothing for an
unknown room or with a patient or an account set, and the clinic's scope without the room still hides the
encounter after the revocation). Every removal of a patient from a room (the webhook's re-check,
`consultation_access_end`, `consent_revoked`) now writes an `AuditEvent` (operation, action `E`,
`Appointment/<id>`, the removing agent, `HOPERAT`) before LiveKit is asked to remove them. It is written when
LiveKit lists the patient in the room, so a repeat, which finds them gone, writes none.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A removal goes unrecorded | Audited before the removal; a failed audit removes nobody and the job or webhook is retried (tested per path, on the real database and the real LiveKit) |
| The room's scope shows more than its consultation | It needs the session of that room; tested as `salus_app` against every other scope |

### Package 6: web app

**Processing added.** No new table, column or endpoint. The practitioner's web app joins the consultation with
`livekit-client` (pinned, `web/src/call.ts` the only module that uses it): it asks our API for a token on every
join and rejoin and connects to our own LiveKit; LiveKit gets the token only, never a name. The patient's name
shown in the call comes from the join response (`peerName`, already audited as a `Patient` read in package 1).
The appointment detail shows the consultation's state and when each side joined
(`GET /appointments/{id}/consultation`, audited as a read of the appointment since package 4), to the
practitioner and the admin. Camera and microphone stay in the browser until the practitioner joins; the device
check's preview and level meter are local and are stopped when the check is left.

**Access.** The join button is offered to the appointment's practitioner only, inside the window; the server
decides every token. When the patient leaves the room the app asks the join endpoint once more to tell a patient
who left from one removed after withdrawing consent (`consent_required`); that request is audited like any join.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| The browser keeps the camera or microphone on after the check or the call | Previews are stopped when another device is picked and when the check is left (tested with a fake); the room is left on leave, end and when the screen closes |
| Patient data in what the web app sends to LiveKit | Nothing but our token; no name, metadata or attributes are set by the client (`call.ts`); the walk's api, worker and LiveKit logs held no name, e-mail or token |
| Text that misstates what happens to the appointment | Every string is the owner's approved text (`docs/consultation/COPY.md`), checked word for word |

### Package 7: mobile app

**Processing added.** No new table, column or endpoint. The patient's app joins the consultation with
`livekit_client` (pinned, `mobile/lib/call.dart` the only file that uses it, with `permission_handler` for the
camera and microphone permissions): it asks our API for a token on every join and rejoin and connects to our own
LiveKit; LiveKit gets the token only, never a name. The practitioner's name shown in the call comes from the join
response (`peerName`, the practitioner's public name). When LiveKit removes the patient or deletes the room, the
app asks the join endpoint or the consultation's status (both audited since packages 1 and 4) to say why. The
token is held in memory for the connection only: never stored, printed or logged. LiveKit's own client log is
turned off.

**Device permissions.** The camera and the microphone are asked for on the device check, not at install or
launch. iOS shows the approved `NSCameraUsageDescription` and `NSMicrophoneUsageDescription`; Android shows the
approved explanation (M35) before the system prompt. A refusal is respected: the patient can still join with
what remains, and the app offers the device settings. "The consultation is not recorded" is true while
ADR 0006 holds and must change if recording is ever added. Changed by SAL-12 (draft): "not recorded as audio or
video", and, with the patient's agreement, what is said is transcribed and translated.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| The phone keeps the camera or microphone on after the check or the call | The previews are stopped on Join and when the check is left; the room is left on Leave, on back and when the screen closes (tested with a fake) |
| Patient data in what the app sends to LiveKit | Nothing but our token; the app sets no name, metadata or attributes (`call.dart`) |
| Patient data in the device log | The app prints nothing through a consultation (tested); LiveKit's client log is off |
| Text that misstates what happens | Every string and both iOS permission texts are the owner's approved text (`docs/consultation/COPY.md`), checked word for word by `mobile/test/copy_test.dart` |

### Package 7.1: the owner's changes after package 6

**Processing added.** One column, `video_session.removal_reason` (`consent_withdrawn` | `cancelled`,
`migrations/00024_video_session_reason.sql`): why the patient was removed from the room, written once, in the
removal's own audited transaction and only when LiveKit lists the patient in the room. It reveals that the
patient withdrew consent or that the appointment was cancelled, which the clinic already knows from `consent`
and `appointment`; it is deleted with the session (retention unchanged, 30 days after the end).
`GET /appointments/{id}/consultation` gains `reason`, derived from it and from the end. Neither client asks the
join endpoint any more to learn why someone left, so the extra token request and its audit event of package 6
are gone.

**Access.** The practitioner who was in the room keeps reading the status of that consultation when a revocation
before the start hid the appointment from the clinic: a new read-only policy (`video_session_attended`) shows a
clinic's scope narrowed to one appointment (`app.appointment_id`, the setting of the notice scope) that
appointment's session once its practitioner joined, and the service then checks that the caller is the
practitioner of the appointment's role. The read returns only the status (state, times, reasons), never the
patient's name, and is audited as a read of the appointment. Another practitioner, the clinic's admin and
another clinic get 404, audited (tested as `salus_app` on the real database).

**Client logs.** `livekit-client` (web) logs nothing (`setLogLevel(silent)`); `livekit_client` (mobile) has its
logger off, which until now threw before any call could start. Chrome itself still prints a failed signalling
connection's URL, the token in its query, to the user's own console (tested in the walk); it is the browser's
network log, not ours or LiveKit's, and the token is valid 5 minutes for that one room.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| The room rule shows a consultation to someone else | It needs the clinic, the one appointment, the practitioner's join and the caller being the role's practitioner; tested in SQL (RLS), with fakes and on the real database |
| The removal's reason outlives its purpose | It lives on `video_session` and is deleted with it |
| A token in the console | livekit-client and livekit_client logs are off (tested); the browser's own network error lines remain (above) |

### Package 8: the end-to-end journeys

**Processing added.** None in the product. `make e2e` runs two consultation journeys on synthetic data made
fresh through the API (the web app in Chromium with fake devices for the practitioner, a Go SDK participant
publishing a generated tone for the patient), and then checks LiveKit's whole log: it must hold none of the run's
synthetic names, e-mails, appointment, patient, practitioner or clinic ids, and must hold the run's opaque room
names, so the check is known to read the right log (`scripts/e2e-livekit-log.sh`; it prints labels, never
values). This is SPEC §2 "LiveKit logs", now checked on every run and in CI.

**Seen in LiveKit's log, not ours to decide here.** At `info` LiveKit logs each client connection's
`remoteAddress` and `localAddress` (IP and port) and its SDK and browser; in production a client's IP is personal
data. Recorded for the owner with the production LiveKit setup (SPEC §8).

### SAL-3: the transcription agent's grant and TURN (packages 9 to 11)

**Processing added.** None of patient data yet. TURN (LiveKit's built-in, TLS on 443 and UDP on 3478) relays a
call's media when a network blocks UDP; the media stays DTLS-SRTP encrypted end to end between each client and
the SFU, the relay sees only ciphertext, and the TURN server is LiveKit itself, so no new processor. TURN
credentials are LiveKit's short-lived ones, per connection. `videosession.Service.AgentJoin` can give the
transcription agent (`cmd/agent`, self-hosted) a 5-minute subscribe-only token for a consultation's room; nothing
calls it yet, and `cmd/agent` is still a stub, so no audio reaches any agent today.

**Transparency.** The agent is not hidden: both apps show "Transcription is on" while it is in the room, and it
is never mistaken for the other person (tested on web and mobile). A hidden agent was considered and refused by
the owner: the patient could not have known.

**Access.** The agent's token needs the same as a person's: the session open and inside its window, the
appointment booked and the patient's consent with the clinic active; refused otherwise, and every request is
audited (`transcription_agent`, `HOPERAT`). It grants no publishing, no data channel, no name or metadata; its
identity is derived from the room's random name, so LiveKit learns nothing new. When the patient's access ends
(revocation, cancellation), the agent is removed from the room with them.

**Open before anything calls the agent (SAL-1).** The lawful basis for transcribing a consultation and whether it
needs its own granular `Consent` (SPEC §4); the speech-to-text engine, and its DPA and `processors.md` entry if
hosted; retention of any transcript; and the permission text "The consultation is not recorded" (COPY.md), which
must change once a transcript is kept.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A listener in the call the patient does not know about | The agent is a visible participant of kind agent; both apps show W41 / M36 while it is in the room (tested) |
| The agent stays after the patient's access ends | Removed with the patient on every removal path (tested in unit, api and worker tests) |
| The agent publishes or injects data into the call | Its grant has no publish and no data permission (tested against the compose LiveKit's token claims) |
| TURN as a relay into the private network | LiveKit relays UDP only and to private addresses only where `allow_restricted_peer_cidrs` allows; locally the compose network only, in production nothing |

### SAL-4: the transcription agent hears the consultation

**Processing added.** While the patient's care consent with the clinic is active (owner's interim decision,
2026-11-09, pending `docs/legal/BRIEF.md` Q16), the self-hosted agent (`cmd/agent`) joins each consultation once
the practitioner is in, decodes each person's microphone in memory and streams it to a speech-to-text engine; the
partial and final text, with the speaker's role and language, goes to the encounter's Redis Stream (transport
only, capped, expiring 30 minutes after its last entry). No audio is stored. The engine is the fake, which
sends audio nowhere; Azure AI Speech is built and not enabled (`processors.md`). The patient's language is read
from their account (audited, the agent as agent).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Transcribing after the consent is withdrawn | The revocation's job removes the agent (SAL-3); its recognitions end at once and pending results are dropped; it asks again once connected, so a revocation during its connection is seen; the next join is refused (tested in unit, worker and e2e tests) |
| Audio or text reaching a third party | Fake by default; Azure only in an EU region (the adapter refuses others) and only once its DPA and zero data retention are in writing |
| What was said in a log | Logs carry the session id, the speaker's role and error codes; the SDK's logger is silenced (tested) |
| The stream outliving the consultation | `MAXLEN ~ 10000` and a 30-minute TTL refreshed on each entry |

### SAL-9: the practitioner's call page (captions, transcript, suggestions, documents)

**Processing added.** None yet: the page is built and tested against in-memory fakes, and the real app passes no
live source, so it shows no panel. Once wired (SAL-6, SAL-8), it shows the practitioner what the plan already
processes (`docs/plans/consult-intelligence.md` "Personal data"): the bilingual transcript, the suggestions and
the drafts of their own appointment's consultation. No new table, column, endpoint or processor.

**Access.** The page asks for one appointment's messages; who may receive them is the gateway's and the drafting
endpoints' decision, server side, each audited there (`docs/plans/consult-intelligence.PROGRESS.md`, SAL-9). Only
the practitioner's web app has these panels; the patient app has none.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Transcript or suggestion text left on the device | Kept in the page's memory only: no browser storage, no log line; gone when the page closes |
| A suggestion read as a diagnosis | Each one is labelled "Suggestion: …" by kind, and the panel always says suggestions are not a diagnosis (`docs/consult-intelligence/COPY.md` W9, tagged `legal`) |
| A document signed with what the conversation did not cover | Fields the draft could not fill are flagged; signing always asks to confirm and says how many fields are empty |

**Open (follow-up ticket).** The patient's notice at the start of the call that it is transcribed (web and
mobile), out of SAL-9's scope by the owner's decision of 2026-11-09. Built by SAL-12 (below).

### SAL-12: the patient's transcription consent

**Processing changed.** Transcription, translation and AI assistance (the agent in the call, the stored
transcript, the suggestions, the drafting) now rest on a separate, granular, revocable FHIR `Consent` per
patient and clinic as well as the care consent (owner, 2026-11-10, `docs/plans/v1-finish.md` "Decisions",
pending `docs/legal/BRIEF.md` Q16). The patient app asks before every call while it is not active: what is
transcribed and translated, the AI assistance for the practitioner (suggestions and document drafts, checked by
the practitioner), where the transcript is kept, that nothing is recorded as audio or video, what declining does,
and how to withdraw (`docs/consultation/COPY.md` M37 to M48, a draft waiting for the owner, the doctor partner and
the lawyer). No processor is named: the engines are fakes and the real ones are not enabled.

**Personal data added.** `transcription_consent` (migration 00040): the patient, the clinic, when it was given,
when it was withdrawn, its status and version. Needed as the proof of the patient's decision (Art. 7(1)) and for
the agent's, the suggestions' and the drafting's check; kept with the clinical record like `consent` (retention
`keep`, needs a legal decision before production). Row-level security as `consent`: the patient's own scope
reads and writes it, the clinic's reads the ones given to it. Every grant, withdrawal and read is audited
(`Consent/<id>`), as for the care consent.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| The agent joins a patient who did not agree | The webhook enqueues no `transcription_join` without both consents; `AgentJoin` refuses; the agent asks again once connected (unit, worker, api and e2e tests) |
| The agent stays after a withdrawal | The withdrawal's `transcription_consent_revoked` job removes it at once from every open room of the patient with the clinic; the agent's own check refuses from the commit (unit, worker and e2e tests) |
| Lines, suggestions or drafts made after a withdrawal | `transcript.Store`, the suggestions' record and run, and the drafting check both consents before each write or model call (unit tests); the stored transcript stays, as said in the notice |
| A withdrawal that costs the patient their consultation or appointments | It removes only the agent: the call, the care consent and the appointments stay (e2e test) |
| A care consent re-granted bringing transcription back without asking | A care revocation ends the transcription consent in the same transaction; it is asked again before the next call |
| The patient not knowing transcription is on | The badge M36 stays while the agent is in the room, and is the way to stop it |

## Questionnaires (`docs/questionnaire`)

### Package 3: questionnaire responses and the access rule

**Processing added.** The FHIR `QuestionnaireResponse` (`questionnaire_response` table): a request to the
patient to fill in one questionnaire version before one appointment, and later their answers (package 4). The
appointment, the clinic, the patient, the practitioner role and the due time (the appointment's start) are
copied once from the appointment; the questionnaire version and its series; how it was requested (`default` or
`added`); the status (`in-progress`, `completed`, `stopped`), the submit time and the answers (a flat list of
`linkId` and values). That a patient was asked a given questionnaire is itself health data (its title can name a
condition). In this package the appointment's practitioner can request a questionnaire for a booked appointment
that has not started, with the patient's active consent, and withdraw it while it is not submitted; a
withdrawal clears the answers at once and keeps the row, with no answers, as the record of the request. Reads
and searches exist in the service; the endpoints come in package 6.

**Purpose and lawful basis.** Preparing the appointment and, once submitted, the care record (Art. 9(2)(h)),
with the consent for sharing before care. **Needs legal confirmation before production** (SPEC §4, §8).

**Access.** `access.QuestionnaireResponse`, pure and table-tested over role, consent, appointment, clinic and
submission: the patient their own; with an active consent, the appointment's practitioner the answers once
submitted and only that it exists and its status before, the clinic's admin that it exists and its status; after
a revocation, for an appointment that started before it, the practitioner the submitted answers (D9) and the
admin that it exists; nobody else, and nobody but the patient and the appointment's practitioner ever sees
answers (asserted for every combination). Row-level security on `questionnaire_response` matches it: the patient
their own (read and update); a clinic the rows of its patients with an active consent (read, request, withdraw)
and after a revocation only the `completed` rows of appointments that started before it (read only); the
notifier's notice scope the rows of its one appointment; deletion only by the discard scope, for a cancelled
appointment, and by `salus_retention`'s purger. `salus_app` cannot change whose a response is, which appointment,
version or due time (tested). The patient scope now also reads the questionnaire versions its own responses
reference, and nothing else of the library. Every request, withdrawal, read and returned search result writes an
`AuditEvent` in its transaction, refusals included (`TREAT` for staff, `PATRQT` for the patient); a request also
records the practitioner's read of the version.

**Retention.** `questionnaire_response`: `delete` an unsubmitted response (in progress or withdrawn) once its due
time has passed, by the daily retention job (one `AuditEvent` per run with the count); a `completed` one is kept
with the clinical record. **Needs a legal decision before production** (SPEC §5, §8).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A practitioner reads answers the patient has not submitted | The rule gives them the status only, and a `Name` view carries no answers (tested); after a revocation row-level security hides unsubmitted rows |
| Answers left after a withdrawal | The withdrawal clears them in the same statement; a check constraint refuses a withdrawn row with answers (tested) |
| Unsubmitted answers kept after the appointment | The purger deletes them once due, in every clinic (tested at the due time ± 1 s) |
| The fact of a request reaching another clinic | Row-level security and the service both scope to the clinic and the patient's consent; a request's version must belong to its clinic (foreign key, tested) |

### Package 4: answers

**Processing added.** The patient's answers to a requested questionnaire (`questionnaire_response.answers`, a
flat list of `linkId` and values) and their submission (`status` `completed`, `authored_at`). The patient saves
answers as they go, each save replacing the saved list, and submits once; a submitted response is final. Each
answer is checked against its item's type and limits; on submit every enabled required question must be
answered, and the answers of items hidden by a condition are removed, so only what the clinic asked and the
patient was shown is kept. The endpoints come in package 6.

**Purpose and lawful basis.** As package 3: preparing the appointment and, once submitted, the care record
(Art. 9(2)(h)), with the consent for sharing before care. **Needs legal confirmation before production.**

**Access.** Only the patient saves and submits their own response, while it is in progress, before the
appointment starts and with their active consent with the clinic: the consent is read in the same transaction,
so a revocation refuses the very next save (service test). Staff who see the response are refused (`ErrForbidden`),
anyone else is told it does not exist. To check the answers the service reads the questionnaire version in the
patient's scope, where row-level security shows only the versions their own responses reference; that read is
audited as the patient's (`PATRQT`). Every save and submit writes an `AuditEvent` in its transaction, refusals
included. A save or submit is made on the version last read (`If-Match`), so a stale device cannot overwrite
newer answers.

**Retention.** Unchanged from package 3: unsubmitted answers are deleted once due; submitted ones are kept with
the clinical record.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Answers kept for questions the patient never saw | Submit removes the answers of hidden items (conditions and hidden sections, chains, tested) |
| Answers saved after a revocation | The consent is read in the save's transaction, not cached (service test) |
| Answers or linkIds in logs or errors | Error text names no linkId and no value; the offending linkIds travel only in `AnswersError.LinkIDs` for the 422 (tested) |
| Oversized or malformed answers stored | Every answer is checked against its item's type and limits on every save and again on submit (tested at each limit) |

### Package 5: booking and cancellation

**Processing added.** Two jobs on the worker's default queue, each enqueued in the transaction that causes it,
with the appointment's id and the clinic's id only. `questionnaire_default_request`, with every booking: the
active version of the default questionnaire of the appointment's role is requested from the patient (a
`questionnaire_response` row, in progress, no answers, source `default`), once. `questionnaire_responses_discard`,
with every cancellation (by the patient, by staff, and by the revocation job): every response of the cancelled
appointment is deleted, submitted or not (D8).

**Purpose and lawful basis.** As package 3. The request needs the patient's active consent with the clinic:
the job reads the appointment in the clinic's scope, which shows a future appointment only while the consent is
active, and the insert policy requires it too.

**Access.** The jobs act on one appointment of one clinic. The request runs in the clinic's scope; it does
nothing for an appointment it cannot see, one no longer booked or already started, a role without a default, or
a default with no active version. The discard runs in the clinic's scope with the appointment's notice and first
checks that the appointment is cancelled; the delete policy checks it again. Both are audited with the job as
agent (`HOPERAT`): one create per request and one delete per response deleted; the questionnaire's service
audits the job's read of the role's default and the version. The jobs log the appointment's id only.

**Retention.** Responses of a cancelled appointment are deleted at once by the discard, including after a
revocation; responses of appointments that started before a revocation are kept (D9). Otherwise unchanged from
package 3.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Answers kept for an appointment that will not take place | Every cancellation path enqueues the discard in its own transaction (service tests); the discard deletes submitted and unsubmitted responses (tested on the real database, also after a revocation) |
| A past appointment's care record lost to a revocation | The discard deletes only for a cancelled appointment; the revocation cancels only future ones; the past one's completed response stays (tested end to end in the worker) |
| A questionnaire requested without consent | The request reads the appointment in the clinic's scope, which hides it after a revocation, and the insert policy requires an active consent (tested) |
| Personal data in job arguments or logs | The arguments are the two ids (tested on River's table); the logs hold the appointment's id only, never a title, question or answer (tested) |
| A retried job duplicating or deleting more | The request inserts once per series (`ON CONFLICT DO NOTHING`); a repeated discard finds nothing left (tested) |

### Package 6: responses API

**Processing added.** The JSON API: the appointment's practitioner requests a questionnaire for an appointment
(`POST /appointments/{id}/questionnaires`) and withdraws a request
(`POST /questionnaire-responses/{id}/withdraw`); the patient saves (`PUT /questionnaire-responses/{id}/answers`)
and submits (`POST /questionnaire-responses/{id}/submit`) their answers. The FHIR API: `QuestionnaireResponse`
read and search (by appointment, patient and status, with `_include=QuestionnaireResponse:questionnaire`), the
answers nested under the questionnaire's sections; and a patient's read and search of the `Questionnaire`
versions their own responses answer. No new field, table or processor.

**Purpose and lawful basis.** As package 3.

**Access.** Through the services of packages 1 to 4 (`access.QuestionnaireResponse`, the consent check, row-level
security as `salus_app`): the patient their own; the appointment's practitioner the answers once submitted and
the status before; the clinic's admin the status only; after a revocation, the appointment's practitioner the
submitted answers of an appointment that started before it (D9), nobody else; anyone else 404. A `Name` view
carries no answers. Each response is written against the version it answers, read once per request as the
caller's own audited read of it. Bodies are read up to 256 KiB, unknown fields refused. Answers are sent with
`Cache-Control: no-store`.

**Audit.** Every request, withdrawal, save, submit and read writes an `AuditEvent` in its transaction, refusals
included; a search one per response returned (one without an entity when none), and one read per version it
answers. The patient's caller reads their own profile first (audited, as for appointments).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Answers shown to staff before the submit, to the admin, or to another practitioner | The handler writes `item` only for a `Full` view; the access matrix is tested through the API on the real database with row-level security on, before and after a revocation |
| Answers in logs, errors or caches | Error bodies are fixed codes with `linkIds` only; the handlers' logs hold no answer, title or question (tested); `no-store` on every answer |
| A patient reading another clinic's questionnaire library | A patient reads a version only in their scope, where row-level security shows the versions their own responses reference (tested) |

### Package 7: e-mail mention

**Processing added.** The booking confirmation and the 24-hour reminder add one fixed sentence (COPY E1) when
the appointment has a questionnaire outstanding: a response in progress, or the active default of the
appointment's role not yet requested. The notifier asks `questionnaireresponse.Service.Outstanding` with the
appointment's id and the clinic's id; only a yes or no comes back. No new field, table, job or processor; SES
(not wired in) would carry the sentence, recorded in `processors.md`.

**Purpose and lawful basis.** As package 3, and as the appointment e-mails (`docs/booking`): so the patient knows
to fill the questionnaire in before the appointment.

**Access.** The check runs in the clinic's scope with the appointment's notice, as `salus_app`: row-level
security shows the responses of that one appointment; the service drops any of another clinic. It reads their
status and series, not their answers. It is audited with the e-mail job as agent (`HOPERAT`): a search event per
response read, or one without an entity when there is none; the questionnaire's service audits the read of the
role's default.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| The questionnaire's title, which can name a condition, in an e-mail | The sentence is a constant; the service returns a boolean only; the e-mail is tested on the real database not to hold the title, a question or the version's id |
| A mention for a questionnaire already submitted | A submitted or withdrawn response, or a submitted default, is not outstanding (service tests and the real database) |
| A failed check sending the e-mail without the sentence | A failed check sends nothing and the job retries (tested) |

### Package 10: mobile

**Processing added.** The patient app shows, on each appointment's card in My appointments, the questionnaires
asked for it, read with one FHIR search of the patient's own responses with the versions they answer
(`QuestionnaireResponse?patient=…&_include=QuestionnaireResponse:questionnaire`); a withdrawn one is not shown,
nor anything on a cancelled appointment. The questionnaire screen saves the answers through the JSON API when the
patient leaves a field, 2 seconds after they stop typing, and at once for a choice, yes or no, or a date; it submits
them after a confirmation and shows them read only afterwards. No new field, table, endpoint or processor; no new
dependency.

**Purpose and lawful basis.** As package 3.

**Access.** Only the signed-in patient's own responses, through the API's access rule and consent check (packages
3, 4 and 6); the app decides nothing the server does not decide again. Conditions are evaluated on the device the
same way as on the server, only to show and hide questions; the server removes the answers of hidden items on
submit.

**Storage on the device.** None: answers live in the screen's memory while it is open and on the server once
saved. Nothing is written to disk, shared preferences or a cache; the session token stays in the platform's
secure storage as before.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Answers or the questionnaire's title in the device's log | The app prints nothing while filling in, saving, a refused save and submitting (tested) |
| An answer lost without the patient knowing | The saved state is always in view (M6); a failed save is shown and tried again every 5 seconds; leaving with an answer not saved asks first (M15) (tested) |
| A save after the consent was revoked or the appointment started | The server refuses it (409); the app shows M16 and stops saving (tested) |
| Answers shown to someone else on a shared device | The app's session rules apply as for appointments; the read-only view shows only the patient's own submitted answers |

## Questionnaire attachments (`docs/attachments`)

### Package 0.1: the `binary` table, the object store and the scanner

**Processing added.** Files the patient attaches to questionnaire answers (photos, PDF documents, short videos)
are FHIR `Binary` resources of the new domain `binary`. Their bytes are stored encrypted in an S3-compatible
object store (Garage in compose, the hosting provider's EU store in production, not chosen); the `binary` table
holds, per file, the response, item, clinic, patient and appointment it belongs to, the detected content type,
the size, the title (the file's name, which can say anything), the SHA-1, the scan status and time, and the
wrapped data key with its key id. `questionnaire_response.attachments` lists the files of each answer. The SPEC's
§5 says why each field is needed. ClamAV (clamd) runs in our own stack to scan the files. No processor is added.

**Purpose and lawful basis.** As the questionnaire's answers: preparing the appointment and, once submitted, the
care record (Art. 9(2)(h)), with the consent for sharing before care; malware scanning is security of
processing (Art. 32). Files are used for the patient's care only: no analysis, thumbnails, AI processing or
training on them. **Needs legal confirmation before production** (SPEC §4, §8).

**Access.** Row-level security on `"binary"`: the patient sees and deletes their own files and inserts files only
for themselves on their own response, with that response's clinic and appointment; a clinic sees a file only
when it can see its response (an `EXISTS` on `questionnaire_response`, so consent and the after-revocation rule
D9 apply unchanged); the scan and discard jobs see the files of one response of one clinic. `salus_app` may update
only the status, scan time, wrapped key and `updated_at`. A response's deletion, by any path, deletes its files'
rows through the foreign key, the wrapped keys with them (tested for `salus_retention`, which holds no privilege
on `binary` for that).

**Retention.** At 0.1 `keep` with the response; made `delete` in package 3.1 (below).

### Package 1.7: the mobile app's pickers, conversion and upload

**Processing added.** The patient picks or records a file (camera, video camera, photo library, document
picker), the app converts HEIC/HEIF to JPEG and other videos to H.264/AAC MP4 on the device, strips the metadata
and uploads the file to the API, one at a time, then polls its status until it is clean or rejected.

**Data minimisation (owner, SPEC §2 "Image and video metadata", O13 and O20).** Before upload the app removes
from photos EXIF (keeping only the orientation), XMP, IPTC, comments, JFIF and anything after the end of the
image (appended previews and depth maps); from PNG every text, time and Exif chunk; from MP4/MOV the `udta`,
`meta` and `uuid` boxes and the creation and modification times. It keeps the JPEG ICC profile and Adobe colour
transform and the PNG colour chunks and `pHYs`, which say how to show the colours and hold no location, device,
time or person. Tested on generated files with location, device and time planted in them. iOS's photo picker
runs without full metadata (`requestFullMetadata: false`). The server does not alter the bytes it receives.

**Storage on the device.** The picker's and the conversion's copies sit in the app's temporary directory and are
deleted once the file is uploaded, removed or left; a file outside that directory (the patient's own) is never
touched. Nothing is written to shared preferences or a cache. The app prints nothing about files.

**Third parties.** `image_picker` and `file_picker` (pinned) and Android's Media3 run on the device and send
nothing anywhere; no processor.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A photo or video reveals where or when it was taken, or the device | Metadata stripped on the device before upload, the orientation alone kept (tested on generated files) |
| A file left in the app's storage | Copies deleted after upload, removal or leaving the screen (tested) |
| A file's name in the device's log | Nothing printed while picking, preparing, uploading or polling (tested) |
| A file uploaded after the consent was revoked or the appointment started | The server refuses it; the app ends filling in as for the answers (tested) |

### Package 2.1: upload, download, status and deletion

**Processing added.** `POST /questionnaire-responses/{id}/items/{linkId}/attachments`, `GET /binaries/{id}`,
`GET /binaries/{id}/content` (with `Range`) and `DELETE /binaries/{id}`. The upload checks the type from the
bytes (never from the declared type or the name), streams the file through SHA-1, the MP4 codec check and the
envelope encryption into one object, and only then records the row, the pending attachment, the scan job and the
audit in one transaction. A file is quarantined until its scan says it is clean. The title comes from the
request's `Content-Disposition`, never from the URL, which access logs may hold.

**Encryption.** Each file has its own random 256-bit data key; the bytes are sealed in 64 KiB AES-256-GCM
segments bound to the file's id, the segment's index and the final flag, so segments cannot be moved, swapped or
truncated undetected. The data key is wrapped by the key-encryption key (`BINARY_KEK` from configuration locally
and in CI, read only in `internal/config`; in production from a root-only key file on the encrypted volume, ADR 0011 point 3). Deleting the row deletes the
wrapped key, after which the stored bytes can no longer be read (crypto-shredding). The store's own encryption at
rest comes on top in production.

**Access.** Through the questionnaire response's rule and consent (`binary.Holder`, implemented by
`questionnaireresponse`): only the patient uploads and deletes, on their own response while it is in progress, the
appointment booked and not started, with the active consent; reads at `Full` only (the patient on their own
response, the appointment's practitioner on a completed one, D9 included); anyone else gets 404 without the file
being disclosed. The download is refused until `clean` (409 `scanning`) and is 404 once rejected. Download
headers: `Cache-Control: private, no-store`, `X-Content-Type-Options: nosniff`, `Cross-Origin-Resource-Policy:
same-origin`, `Content-Disposition: inline`, and `Content-Security-Policy: default-src 'none'` (with `sandbox`
for images and video, O23).

**Audit.** Every upload (`C`, `PATRQT`), status read and download (`R`; `PATRQT` for the patient, `TREAT` for
staff), each range request included (a seek in a video is a read), and every deletion (`D`) writes an
`AuditEvent` in the same transaction, refusals included: one event per entity, `Binary/<id>` and
`QuestionnaireResponse/<id>` (O22). The download's audit commits before a byte is sent.

**Logs.** The request id, route, status, duration and opaque ids only; never the title, a byte of the file, or the
content type next to a patient id (tested on the handler's log).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A file read before it was scanned | Download refused while `quarantined`; nobody, the patient included, opens it before `clean` (tested) |
| A practitioner reads a file the patient has not submitted | `Holder.Readable` gives staff only completed responses (tested); a completed response holds only clean files (3.2) |
| The stored object read outside the API | Encrypted per file; the key lives only wrapped in the row; the store has no host port |
| A malicious file served to a practitioner's browser | Type from the bytes, MP4 codecs checked, `nosniff`, the CSP per type, same-origin only |
| A file twice from a retried upload | An identical retry (same item, SHA-1 and size) returns the first file and deletes the new object (tested) |
| A file's name or bytes in a log | Never logged (tested) |

### Package 3.1: scan, discard and purge

**Processing added.** The `binary_scan` job (queue `scan`, 2 at a time) decrypts the stored file and streams it to
clamd's `INSTREAM`: clean marks the file clean and the answer no longer pending; a signature found rejects it,
clears its wrapped key at once, detaches it from the answer and deletes the object after the commit. The
`binary_discard` job deletes the files a response no longer lists (after a withdrawal, or a submit that hid an
item). The daily retention purger deletes rejected rows 24 h after their scan and every object older than 24 h
with no row. The jobs' arguments are opaque ids only (the binary, the response, the clinic; O21).

**Scanning on our own infrastructure.** clamd runs in our stack beside the worker; the file never leaves it. Its
signature updates (freshclam) download from ClamAV's CDN and send no personal data. A scan that fails or cannot
reach clamd leaves the file quarantined and River retries; nobody opens it meanwhile.

**Audit.** A clean result is an `update` and a rejection a `delete`, each by `binary_scan` as agent for `HOPERAT`;
a discard a `delete` by `binary_discard`; one event per entity. **The ClamAV signature name is not audited** (O25):
`audit_event` has no field for it, and the rejection is recorded without it. The signature appears only on the
job's log line, next to the binary id and nothing else, never next to a patient id; it is a ClamAV name, not
personal data.

**Retention.** `"binary"`: `delete`, 24 h: a rejected row once 24 h have passed since its scan; every other row
lives exactly as long as its response or its answer. Objects: deleted after the commit that removes their row,
and by the purger after 24 h when no row names them. `salus_retention` gains `SELECT, DELETE` on `"binary"` (and
nothing else on it). **Needs a legal decision before production** (SPEC §5, §8).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Malware reaches a practitioner | Every file scanned before anyone can open it; a rejected file's key cleared at once and its object deleted (tested with the EICAR string in a generated PDF) |
| A rejected file's name kept | The row (with its title) goes after 24 h; its key and bytes at once |
| A deleted file's bytes left in the store | Unreadable once the row goes (the key goes with it); the purger deletes ciphertext without a row after 24 h (tested) |
| The signature linked to a patient in logs | Logged with the binary id only (tested) |
| A job reaching another clinic's files | The job scope is one response of one clinic (row-level security, tested) |

### Package 3.2: attachment answers in questionnaire responses

**Processing added.** A `QuestionnaireResponse` carries each file as a `valueAttachment` (`contentType`, `size`,
`title`, `hash` the base64 SHA-1, and `url`, the API's download endpoint, never the bucket). A required file
upload item needs a file; a submit is refused while any file is still being checked (409 `attachments_pending`), so
a completed response holds only clean files. On submit the files of items hidden by a condition are detached and
discarded; a withdrawal clears the files with the answers and enqueues the discard. No new table or field.

**Access.** As the response: the patient's own view lists their pending files too; the practitioner sees files
only on a completed response; a `Name` view (the admin's, or the practitioner's before the submit) carries no
file (tested).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A practitioner sees a file still being checked or rejected | The submit is refused while a file is pending; a rejected one is detached (tested) |
| Files kept for an answer the patient no longer gives | Hidden items' files detached on submit and discarded by the job (tested) |
| Files kept after a withdrawal or a cancellation | The withdrawal clears them and discards; a cancellation deletes the response and its files' rows with it (tested, and end to end in 4.1) |

### Package 4.1: the end-to-end journeys

No new processing. The journeys (`test/e2e/attachments_test.go`, `test/e2e/web/attachments.spec.ts`) use only
generated files (a JPEG, an H.264 MP4 made in the test, a PDF with the EICAR string assembled in the test) and
synthetic accounts, and check end to end that a rejected file is detached and unreadable, that the practitioner
reads only the clean files of the submitted response, that every upload, status read, download and range request
is audited, and that a cancellation takes the file's row with the response.

## Languages (`docs/i18n`)

### Package 3.1: the three language fields, end to end

**Processing added.** Three personal-data fields (docs/i18n/SPEC.md §5), added by packages 0.1, 1.1 and 2.1 and
walked end to end in 3.1:

| Where | Field | Why it is needed |
| --- | --- | --- |
| `account` | `language` | The user's chosen language, so every device and every e-mail uses it |
| `account` | `detected_language` | The language for e-mails when the user hasn't chosen one |
| `questionnaire_response` | `language` | Which translation the patient answered, so the practitioner reads the questions the patient read |

`account.language` is set only by the user (the picker, or a choice made on the sign-in or registration screen and
sent with the next sign-in) and cleared by "Use the browser's language" / "Use the device's language".
`account.detected_language` is the language the client matched from the browser's or the phone's language list at
the last registration or sign-in; only the matched tag is sent, never the device's list. `questionnaire_response.
language` is set when the patient starts a questionnaire and cannot change after the first saved answer.
`organization.default_language` and `questionnaire_translation` hold no personal data. No processor is added: the
translations are drafted in the repository, and machine translation of questionnaires is out of the feature
(it would need an AI processor, a DPA and its own SPEC).

**Purpose and lawful basis.** The account's two fields provide the service the user signed up for, in a language
they read (Art. 6(1)(b)); they are not special-category data, although a language can hint at origin, which is
why they are minimised (one tag each, no history) and never used for anything but the UI and the e-mails: no
analytics, marketing or profiling. The response's language is part of the answers, under the questionnaire's
basis (docs/questionnaire/SPEC.md §4, Art. 9(2)(h)), and is shown only with the answers: the FHIR read without
the answers does not carry it.

**Access and audit.** The account's fields are read and written only by `auth` (its own account, `GET` and
`PATCH /auth/me`, sign-in and registration), and every change is audited like any account write; the notification
job reads the e-mail language through `auth.Service.AccountEmail`, one audited read with the job as agent. The
response's language follows the response's row-level security and audit (the start is audited, refused or not).

**Logs.** No language appears in a log line, metric or trace next to an account or patient id (checked in 3.1:
no log call in `internal/` names a language).

**Retention.** Covered by the existing rules: the account's fields go with the account, the response's with the
response (`retention`); no new table with personal data.

**Tested.** `test/e2e/language_test.go`: a German phone's registration and its German confirmation e-mail; a staff
choice that holds on the next sign-in from another browser, a patient language refused for staff, and the choice
cleared; a translated questionnaire answered in English by a German patient, the language locked after the first
answer and read by the practitioner; an Italian patient on a Bosnian-only questionnaire. All with synthetic
accounts and placeholder questionnaire wording.

## Consultation intelligence (`docs/plans/consult-intelligence.md`)

### P0 (SAL-2): contracts and tables

**Processing added.** None runs yet: P0 adds the event types, the engine ports with in-memory fakes, and four
tables nothing writes to so far. `transcript_segment` will hold what each participant said in a consultation
(the original, its translation, the speaker, the language and the times); `suggestion` what was suggested to the
practitioner from it and what they did with it; `document_draft` the document the practitioner writes from a
clinic's `document_template` (no personal data). Each row references its FHIR `Encounter`. The plan's "Personal
data" section says why each field is needed.

**Purpose and lawful basis.** The care of the patient in the consultation (Art. 9(2)(h)). Whether the care consent
covers translation, suggestions and drafting, or each needs its own `Consent`, is open: a decision for the owner and
legal before SAL-4 (the plan's "Gates"). **Needs legal confirmation before production.**

**Access.** Row-level security, the clinic's scope only: a row follows its encounter (the active consent, or after a
revocation the appointments that started before it); a draft also needs its template in the same clinic. A video
room's scope, which shows an encounter after a revocation, sees and writes nothing here; neither does the patient's
until the export feature (tested). `salus_app` never changes a segment, a suggestion's content or which encounter or
template a row belongs to, and deletes nothing (tested).

**Retention.** `keep` with the clinical record, like `encounter`; the periods need the legal decision.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Patient speech or text sent to a hosted engine without a DPA | Only the fakes implement the ports; a real STT, translation, suggestion or TTS engine needs its DPA, zero data retention in writing and an entry in `processors.md` first |
| A transcript written after a revocation | No room scope on these tables; the service checks the active consent before each write (from SAL-4) |
| Transcript text in logs or errors | Validation errors name the field, never its value (tested) |
| A suggestion read as a diagnosis | Shown to the practitioner only, labelled as a suggestion, citing its segments; what they did with it is kept |

### C (SAL-6): stream storage, the WebSocket gateway and the transcript export

**Processing added.** Three things, all on the encounter's Redis Stream (`consult:<encounter_id>:events`, entries
`{envelope: <wire envelope>}`):

- The stream consumer (`worker.ConsultStream`, in `cmd/worker`) stores what the stream carries: final segments into
  `transcript_segment` (partial ones stay on the stream only), suggestions into `suggestion` with their latest
  status. Drafts are SAL-8's and left alone. It has only the encounter's id: `encounter_organization(uuid)`
  (migration 00033, a `SECURITY DEFINER` function owned by `salus_encounter_lookup`, which reads two columns of
  `encounter`) gives it the clinic, and nothing else; the write then runs in that clinic's scope under the tables'
  row-level security.
- `GET /encounters/{id}/events`, a WebSocket that forwards the stream's envelopes unchanged, each after a cursor
  frame with its stream id, to the encounter's practitioner.
- `GET /encounters/{id}/transcript`, the stored segments, to the encounter's practitioner.

No new personal field; the tables are P0's.

**Purpose and lawful basis.** As P0: the care of the patient in the consultation (Art. 9(2)(h)); the consent
question for translation, suggestions and drafting stays open (P0's note), and nothing here sends data anywhere new:
the stream, the api, the worker and Postgres are ours.

**Access.** `access.ConsultationRecord`: the appointment's practitioner with the patient's active consent, nobody
else (not the clinic's admin, not the patient, whose copy comes with the export feature). After a revocation the
practitioner keeps the encounter at the name level, but neither the stream nor the transcript, also for a
consultation that started before it. The gateway checks access again after every read of the stream: a revocation
closes the socket (`4403 consent_required`) and nothing more leaves, within a second on an idle stream (tested).
The consumer stores nothing once the consent is not active. The upgrade must come from the API's own origin: the
session cookie is `SameSite=Strict` and a WebSocket cannot send `X-Salus-CSRF`, so the origin is the cross-site
check.

**Audit.** Every stored segment and suggestion (agent `consult_stream`, `HOPERAT`, `Encounter/<id>`, the table as
`entity.name`), every refused write, every subscription to the stream (operation, action `E`) and every export
(read, with the number of segments), refused ones too.

**Retention.** Unchanged: the tables' `keep` rules. The stream is transport (ADR 0005): capped and expired by its
producers; losing it loses live captions, never the stored transcript.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A transcript or suggestion written or shown after a revocation | Consent checked before every write and after every stream read (tested against the database and through api.New) |
| A cross-site page opens the stream with the practitioner's cookie | The handshake's Origin must be the API's own (tested, a sibling origin included) |
| An entry written twice, or a crashed worker's entry lost | Consumer group, at-least-once; segments by their event id, suggestions by theirs, so a repeat adds no row (tested); an unacknowledged entry is claimed again after 30 s |
| What was said in a log line | The consumer logs the stream key, the entry id, the type and the reason only; the gateway the encounter id (tested for the synthetic text) |
| The lookup function discloses more than the clinic | Owned by a role with two columns of `encounter` and no other privilege; `EXECUTE` for `salus_app` only (tested) |

### B (SAL-5): translation of the transcript

**Processing added.** Inside the agent (`cmd/agent`), while it is in the consultation's room, each final segment
is translated into the listener's language before it goes on the encounter's stream: a patient's into the
practitioner's (the clinic's default), a practitioner's into the patient's (their account's, else the clinic's).
Partial segments and segments already in the listener's language are not translated. For each line the
terminology (reference data, no personal data) is searched for the medical terms said in it, by the line's words,
in memory; the line is never stored by that search, logged or put into an error. The translation, or the flag
`translation_failed` when it could not be made within 1.5 s and one retry, is stored with the segment
(`transcript_segment.translation`, `translation_failed`, migration 00034): the flag tells a reader of the record
that the line's translation is missing, rather than not needed. No new table.

The translation is done by a language model behind `llm.Translator`. The default is the fake, which sends text
nowhere. The Anthropic Claude API adapter (`pkg/llm/anthropic`) is built and not enabled: it would send each line,
its two languages and the terms found in it (no name, account or id of a person) to a processor with no EU region
today (`processors.md`).

**Purpose and lawful basis.** As P0: the care of the patient in the consultation (Art. 9(2)(h)), so that the
practitioner and the patient understand each other. The consent question for translation stays open (P0's note,
`docs/legal/BRIEF.md` Q16); the owner's interim decision applies: translation runs only while the agent is in the
room, that is while the patient's care consent with the clinic is active.

**Audit.** No AuditEvent of its own: like the speech-to-text, the translation is processing in the agent's memory
during the session; the stored segment's write is audited by SAL-6's consumer.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Patient speech sent to a hosted model without a DPA, outside the EU, or retained | Fake by default; the Anthropic adapter needs `LLM_ENGINE=anthropic` and may not be enabled before the DPA, zero data retention (an organisation arrangement with Anthropic) and an EU region, which the API does not offer today (`processors.md`) |
| A line translated or published after the consent is withdrawn | The removal ends the call: a pending translation is cancelled and nothing is published after it (tested) |
| A mistranslated line read as what was said | The original is always kept and shown with the translation; a failed translation is flagged, never replaced by a guess; the prompt forbids adding, explaining or correcting |
| A line injected as instructions to the model | The prompt marks the line and the glossary as data, never instructions; the answer is constrained to a fixed JSON schema |
| What was said in a log line or error | Logs carry the session id, the speaker's role and the error; the adapter's errors name a status, an API error type or a stop reason only (tested with marker text) |

### D (SAL-7): the live suggestions worker, and dismissing and pinning

**Processing added.** The suggestions worker (`worker.Suggestions` in `cmd/worker`, `suggestion.Assistant`) reads
each consultation's stream, keeps the latest 30 final lines in memory, and sends them with the texts of the
suggestions already shown to the model behind `llm.Suggester`, in the clinic's default language. What comes back is
checked (a known type, confidence 0 to 1, sources among the lines sent, a text of 1 to 300 characters), stored in
`suggestion` and put on the stream for the practitioner. The practitioner dismisses, pins or unpins a suggestion
over the event stream's WebSocket; the status is stored. No new table or field: `suggestion` already holds what was
suggested, from which lines, and what the practitioner did with it (P0). Redis holds one lease key per encounter
(`suggest:<encounter_id>:lease`, the worker member's random name, 30 s TTL): no personal data.

**Purpose and lawful basis.** As P0: AI assistance for the practitioner's care of the patient in the consultation
(Art. 9(2)(h)), under the owner's interim consent decision (plan, "Gates", pending `docs/legal/BRIEF.md` Q16): the
worker suggests only while the patient's care consent with the clinic is active, and stops at once on its
withdrawal (checked before every model call and again before every store). A suggestion is labelled as AI
assistance in the event (`origin: "ai"`) and never presented as a diagnosis.

**Processors.** None new: `LLM_ENGINE=fake` (the default) suggests nothing and sends nothing. The Anthropic adapter
(SAL-5's entry in `processors.md`) is merged and not enabled; enabled, it would receive the window and the texts of
the suggestions already shown, no name or id of a person.

**Access.** The worker is a system path (the encounter's clinic scope, row-level security). A status change has the
stream's rule (`access.ConsultationRecord`: the encounter's practitioner with an active consent) and only a
suggestion of that encounter.

**Audit.** The worker's read of the shown suggestions before each run and each suggestion it stores (agent
`suggestions_worker`, `HOPERAT`), refusals too; every status change by the practitioner (update, `TREAT`), refused
ones too.

**Retention.** Unchanged: `suggestion` is `keep` with the record; the lease expires on its own.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A suggestion made or shown after a revocation | Consent checked before the model call and before each store; nothing is published before it is stored (tested) |
| A dismissed suggestion comes back | The stored suggestions decide, any status, by their words (a rewording sharing three words in four is the same); the model is also told them (tested, the real database included) |
| A stale event sets an older status back | The stream consumer never changes a stored status; only the practitioner's path does, before publishing (tested) |
| The model's output shows something wrong or unsourced | Validated against the contract; dropped items are counted, not shown (tested) |
| What was said or suggested in a log line | The worker logs encounter ids, counts and reasons; a failed model call is logged without its error, which may echo the input (tested for the synthetic text) |
| A status change for another encounter's suggestion | The update is keyed by the encounter and the id together; another's is `not_found` (tested on the database and through api.New) |

### E (SAL-8): document templates, drafting and signing

**Processing added.** The encounter's practitioner drafts a document from a clinic template: the stored transcript
(final segments, as said) and the template's fields go to the drafting port (`llm.Drafter`); today only the
in-memory fake implements it, so nothing leaves the stack. The draft (`document_draft`, P0's table) keeps each
field's value with the segments it came from; signing adds the final Composition, a PDF of it in the object store
(encrypted per file with `pkg/envelope`, as attachments are) and the signature's Provenance. New personal fields:
`document_draft.signed_at`, `signer_role_id` (who signed, when: the practitioner's accountability for the record)
and the PDF's size, SHA-1 and wrapped key (to serve and verify the PDF). `document_template_translation` holds no
personal data.

**Purpose and lawful basis.** The care of the patient (Art. 9(2)(h)): the practitioner's documentation of the
consultation. Drafting follows the owner's interim consent decision: only while the patient's care consent with the
clinic is active.

**Access.** `access.ConsultationRecord`: the encounter's practitioner with the patient's active consent, in the
clinic's scope; checked again after the model answers and before a signature is committed, so a revocation stores
nothing more. Templates: staff of the clinic.

**Audit.** Every draft written (`create`/`update`, `Composition/<id>`), every signature (the Composition's update,
the DocumentReference's and the Provenance's creation), every read of a draft list, a Composition, a
DocumentReference, a Provenance or the PDF, and every refusal. The transcript read for a draft is audited by
`transcript`. Template writes are audited with `entity.name` `document_template`.

**Retention.** `document_draft` is `keep` (P0); the PDF lives as long as its row's wrapped key (deleting the row makes
the object unreadable). An object whose commit failed is unreadable and not yet deleted (PROGRESS, "Left").

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A model's value presented as said when it was not | A value is kept only when it cites segments of the transcript and is of its field's type; otherwise the field is missing; the practitioner edits and signs, an edited value loses its sources (tested) |
| A required field guessed | Never filled without sources; a sign with a required field empty is refused (tested) |
| A document written or read after a revocation | Access checked before and after the model's call and the PDF's put, and on every read (tested) |
| Field values or transcript text in logs | Logs carry ids and the error only; errors name field keys, never values (tested for a publish failure) |
| The PDF readable from the object store | Encrypted with a per-file key wrapped by the KEK; the store sees ciphertext only |
| Patient text to a hosted model without a DPA | Only the fake is wired; SAL-5's adapter is off by default and gated on its DPA |

### SAL-17: the patient's signed documents and transcript copy

**Processing added.** No new personal data is collected. The patient reads what is already kept about their own
consultations: the signed documents (Composition, DocumentReference, Provenance, the PDF) and, while their clinic
shares it, the stored transcript. A signature enqueues an e-mail job (appointment and clinic ids) and the notifier
tells the patient a document is ready. The clinic's setting `organization.patient_transcript_copy` is not personal
data.

**Purpose and lawful basis.** The patient's right of access to their own record (Art. 15) and the care itself
(Art. 9(2)(h)). The e-mail is part of the care service, as the appointment e-mails are.

**Access.** `access.SignedDocument`: the encounter's patient, whatever the consent, signed documents only; the
practitioner as before. `access.TranscriptCopy`: the encounter's patient while the clinic shares transcripts. Both
in the patient's own scope: row-level security (`document_draft_own`, `transcript_segment_own`, migration 00038)
shows them only their own encounters' rows, final documents only.

**Audit.** Every search, read and PDF download by the patient, and every refusal, with purpose `PATRQT`; the
e-mail job's reads with the job as agent, as the appointment e-mails.

**Retention.** Unchanged. A PDF the patient shares leaves the app on their own decision; the app's temporary copy is
deleted when the share sheet closes and swept at the next start.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A draft shown to the patient as if final | Row-level security shows the patient's scope final rows only; the service refuses a non-final one as not found (tested in the repository, the service, the API and the migration) |
| Another patient's document or transcript | The patient's scope matches the encounter's patient (tested) |
| Clinical content in the e-mail | The template names the clinic, the date and the link only (tested in every language) |
| A copy of the PDF left on the device | Written under `salus-document-*` in the temporary directory, deleted after the share, swept at start with share_plus's own copy |
| The transcript shown when the clinic does not share it | Checked in the service on every read (tested); off by default |

## Medical terminology (`docs/terminology`)

### SAL-14: the review panel

**Personal data.** Staff data only, no patient data. `term_reviewer` (00039) holds the account id of each staff
member the operator made a reviewer of the term list, and when. `term.reviewed_by`, `term_translation.reviewed_by` and
`term_explanation.reviewed_by` (00037) hold the account id of the reviewer who last approved an entry or marked it as
needing changes, with the time. The panel's labels, explanations and notes are reference data about medical terms;
the notes are the reviewer's, about a term, and the panel says nothing of a patient.

**Purpose and lawful basis.** Accountability for the medical term list every clinic works with (Art. 5(2), Art.
6(1)(f)): who approved or flagged an entry. The permission itself is part of the contract with the staff member's
clinic (Art. 6(1)(b)).

**Access.** Every staff member reads the queue and the progress; only a reviewer writes. The permission is granted
and revoked only by the operator (`salusctl term-reviewer grant|revoke`), never by a clinic admin; `salus_app` can read
`term_reviewer` but not write it (tested), so no request to the API can grant it.

**Audit.** Every edit, approval and request for changes is an `AuditEvent` with the reviewer as the agent and
`entity.name` `term_translation` or `term_explanation` (as the document templates are audited, the term list being no
FHIR resource), in the write's transaction; a write refused to a non-reviewer is audited as a minor failure. A grant
or revocation is audited with salusctl as the agent and `Account/<id>` as the entity.

**Retention.** `term_reviewer` is `keep` while granted: revoking deletes the row, and so does deleting the account
(foreign key). The `reviewed_by` columns are `keep` with the term, the reference cleared with the account's erasure
(subject-rights feature).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A clinic admin or a bug grants the permission | `salus_app` has no write on `term_reviewer`; only salusctl, as the owner, writes it (tested) |
| A reviewer's edit reaches every clinic unseen | An edit sets the entry back to unreviewed (the cheatsheet marks it "Not reviewed yet"), and every write is audited |
| Patient data typed into a label, an explanation or a note | The panel asks for terms and plain-language text about them; no log line carries a label, an explanation or a note |

### SAL-13: plain-language explanations of terms

**Personal data.** No new field and nothing new stored. Tagging reads patient data already read: a transcript line (said
in a consultation) or a document's line (signed by the practitioner). The spans it returns (which term is where in a
text) are derived from that text and so are health data too: they go only to whoever reads the text, in the same
answer or event, and are never stored (not in `transcript_segment`, not in `document_draft`). The explanations
themselves are reference data about medical terms, the same for every patient.

**Where the texts are processed.** Only where they are already read, under that read's rule, on our own servers: the
agent tags a final line before it goes on the encounter's stream (the practitioner's, as before); the transcript copy is
tagged in `GET /encounters/{id}/transcript` (the patient's while the clinic shares it, `access.TranscriptCopy`); a
document is tagged in `GET /compositions/{id}/terms`, which reads it through the Composition's rule
(`access.SignedDocument` for the patient: signed documents only; the practitioner with the active consent) and is
audited as that read. No endpoint takes a text from a client, so no text travels anywhere it did not before, and no
access rule is bypassed. The explanations are fetched as one list per language (`GET /terminology/explanations`), with
no patient data in the request: which terms a patient's text holds never leaves the server for them.

**Processors.** None added: the explanations were drafted once, in the development session, from the term list
(reference data); no language model is called at runtime (docs/plans/v1-finish.md "Decisions").

**Logs.** A tagging failure is logged with the session id, the speaker's role and the error, never the line; the texts
and spans are in no log line, error, metric or trace. `GET /compositions/{id}/terms` answers `Cache-Control: no-store`.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| An explanation reads as a diagnosis or advice | Written as information only, every one labelled "Informational, not a diagnosis" on screen, "Not reviewed yet" until the doctor partner approves it in the review panel; one marked as needing changes is not shown |
| A signed document appears altered | The spans are an overlay: the document's text, its Composition and its PDF are unchanged (tested); the explanation sits in a tooltip or dialog apart from the text |
| A term list read gives away a patient's conditions | The page asks for all explanations of a language, not for the terms of a text |
| The spans outlive the text | Never stored: they exist only in the answer or event that carries the text |

## Deployment (`docs/plans/deploy.md`, ADR 0011)

Notes for the first production deployment; nothing is deployed and no real patient data is processed until the
points below are settled.

- **Where the data is.** One Hetzner Cloud server in Helsinki (FI) holds every table, Redis and the attachments
  (Garage); a Hetzner Storage Box holds the nightly Postgres dumps and archives of Garage's objects (each object
  already envelope-encrypted with `BINARY_KEK`), encrypted on the server with `age` before they leave it. The
  backups' failure alerts go to the operator through the SMTP relay and carry the host, the backup kind, the time
  and the exit status only, no personal data. Hetzner is a processor (docs/privacy/processors.md, pending its DPA).
- **Backups and erasure.** Dumps and object archives are kept 14 days, so data erased in the database or the store
  stays in them for up to 14 days, unreadable without the `age` private key held offline. To be confirmed by the
  owner as the backups' retention rule. A restore must redo, by hand, the erasures completed after the dump: their
  record is lost with the database, so the owner keeps a list of them outside the server (docs/deploy/backup-restore.md
  "Erasure"; no tool re-applies them yet).
- **Logs.** Caddy writes no access log with client IP addresses; LiveKit logs at `warn`; container logs rotate.
- **Decided (ADR 0011 "Before real patient data", points 2 to 6, 2026-11-11).** Postgres's and Garage's data sit
  on a LUKS-encrypted Hetzner Volume unlocked by hand after a reboot (ClamAV's scan spool on a `tmpfs`, no swap);
  `BINARY_KEK` and `TOTP_KEY` sit in a root-only file on that volume, not in the env file; traffic between
  containers on the one host is plaintext, everything leaving it is encrypted in transit; e-mail goes through an EU
  SMTP relay (Brevo or Mailjet), a processor once its DPA is signed; live speech-to-text and the language model stay
  on the fakes until their DPAs are signed. The attachments are backed up nightly like the dumps (point 7).
- **Open (ADR 0011 points 1 and 7).** The DPAs (Hetzner, the SMTP relay) and the backups' retention.

## Chat (`docs/plans/chat.md`, ADR 0012)

### SAL-27: core messaging

**Processing added.** Four tables: `conversation` (a chat about one patient in one clinic, with its treating
practitioner and optionally an encounter), `conversation_participant` (the patient and the practitioners who take
part, by profile and role, with each one's read position), `communication` (each message: its text, its time, who
wrote it) and `communication_revision` (the earlier text of an edited message). Messages travel live through Redis
pub/sub (`chat:<conversation_id>`), never stored there. No new processor.

**Purpose and lawful basis.** Care between visits (Art. 9(2)(h)), shared with the clinic under the patient's care
consent (docs/booking/SPEC.md section 3). Not used for analytics, training or marketing. The AI summary of SAL-31
is a further purpose and needs its own basis (plan, "Notes for the wave-1 packages").

**Necessity and minimisation.** Each column is listed in the plan's "Personal data" with its reason. The API never
returns the sender's account or the patient's id with a message; system messages carry codes and opaque ids, never a
name. The read position is the reader's own (owner, 2026-11-12); read receipts and typing are not stored (SAL-30).

**Retention.** Kept with the clinical record (`ThenKeep`), like the encounter and the transcript; needs a legal
decision before production. Never deleted by a user or the app: no role has `DELETE`, and triggers refuse `DELETE`
and `TRUNCATE` for every role, the owner included. Erasure under Art. 17 is limited by medical-record retention law
and comes with the subject-rights feature (export through `Patient/$everything` there too).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A clinic, or another patient, reads a patient's messages | Row-level security on all four tables: the patient sees their own conversations, never a doctor-only one; a clinic its conversations only while the care consent is active; messages only with the conversation set in a scope that sees it (migration tests over every scope). The service checks `access.ReadsChat` and `WritesChat` as well; no admin reads a chat |
| A revocation does not stop a doctor who has the chat open | The socket checks access again at least every 5 seconds and closes (4404); every read after the revocation is refused (app test) |
| A doctor added later reads the history before their point (SAL-28) | `history_from_seq` bounds every page, replay, revision and live edit the service and the socket give |
| A message is changed without trace | Only the sender edits, only the text; the earlier body is copied by a trigger in the same statement, so no edit skips the history |
| Message text leaks into logs or notifications | No log line holds a message, a body or a name (app test checks the log); the live frames go only to sockets that passed the access check; notifications (SAL-29) never carry the text |
| A lost message or a duplicate | Postgres is the record; pub/sub only transport. Numbers without gaps under the conversation's lock; a socket fills any gap from Postgres; a retried send with the same client id returns the first message (concurrency test on Postgres) |

**Open before production.** The retention period of the chat with the clinical record (legal decision).

### SAL-28: group consults

**Processing added.** The treating practitioner adds a doctor of their clinic to a conversation, or invites a doctor of
another clinic (listed in the public directory), with a reason code and how much of the history they read; doctors
open doctor-only side threads of a conversation. New: `conversation_participant.organization_id`, `status` and
`reason`, and the table `chat_consent` (consent domain): the patient's consent for one conversation and one invited
practitioner role, with its grant and revocation times. System messages in the conversation say who joined, left,
was invited, granted or declined, as codes and opaque ids. No new processor: the other clinic is a controller of its
own for the care it gives, and receives the conversation only on the patient's consent.

**Purpose and lawful basis.** Care between visits within the treating clinic (Art. 9(2)(h)), as SAL-27. Sharing with
another clinic rests on the patient's explicit, granular consent (Art. 9(2)(a), Art. 7): that conversation and that
doctor, revocable at any time, with effect at once. The care consent with the treating clinic still rules: its
revocation ends every doctor's access, the invited ones' too.

**Necessity and minimisation.** Before the patient consents, the invited doctor sees nothing: not the conversation,
its participants, its messages or side threads, nor that the invitation exists. A doctor added from a chosen point
reads only what follows it. A side thread keeps the doctors it was opened with; a doctor added later is not in it.
The system messages carry no name.

**Retention.** `chat_consent` is kept with the clinical record (`ThenKeep`) as proof of the decision, like `consent`;
the participant columns with the conversation. Needs a legal decision before production.

### SAL-30: attachments, read receipts, typing, search and terms

**Processing added.** Files a participant sends in a conversation (photos and PDFs, lab results among them) go through
the questionnaire attachments' pipeline (`binary`, ADR 0010): the type from the bytes, encrypted with a key per file
in the self-hosted object store, scanned by the self-hosted ClamAV before anyone opens them. A `binary` row now has
one of two parents, a questionnaire answer or a conversation (`conversation_id`, the uploading participant, the
message once sent). A clean image gets a thumbnail (a JPEG of at most 320 pixels a side), made on our worker from the
clean file and stored encrypted like it. The message counts its files (`communication.attachment_count`). Read
receipts (how far each participant has read) and typing go through the Redis pub/sub channel of the conversation; the
receipts are also kept in Redis for 7 days after each participant's last read, each under its own key, so a socket
opening later shows those of the conversation's current participants (a removed participant's is never sent), and
typing leaves a key of one second (its rate). Neither is stored in Postgres. Search is Postgres full-text search over the message texts, with an
index per partition; the query is never logged or stored. Terms are tagged on read against the reference term list
(SAL-13) and never stored. No new processor: the object store, ClamAV and Redis are self-hosted.

**Purpose and lawful basis.** Care between visits (Art. 9(2)(h)) under the care consent, as the messages. Thumbnails
are a preview of the file for the same care, made by us, never by a third party. The questionnaire SPEC's "no
thumbnails without a separate basis and SPEC" (docs/attachments/SPEC.md section 4) concerns questionnaire files; the
chat's thumbnails are asked for by the chat's plan (SAL-30), not made for questionnaire files.

**Necessity and minimisation.** A file keeps its name (so the reader can tell files apart; never logged), its type,
size and SHA-1; nothing else of the file is read but by the scan and the thumbnail. Receipts carry a participant id
and a number, typing a participant id; both opaque. A search returns only messages the caller may read, at most 50,
and is audited once for each patient whose messages it returns.

**Retention.** A file attached to a message is part of the record with it: kept with the clinical record, never
deleted while attached (a trigger refuses it for every role, the owner included, unless the file was rejected). A
file never sent is deleted 24 hours after its upload, a rejected one 24 hours after its scan, an object without a row
(a file's or a thumbnail's) 24 hours after it was written. A participant's receipt expires 7 days after their last
read, typing after one second.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Malware reaches a doctor's or a patient's device | Nobody opens a file before the scan says it is clean (409 `scanning`); an infected file is rejected, its key cleared and its object deleted (worker test with the EICAR string); a file that cannot be scanned is rejected after the last attempt, never left opening later |
| A file is read by someone who may not read its message | Row-level security on chat files mirrors the messages' (migration tests over every scope: the patient never in a doctor-only thread, a clinic only with the consent active and the conversation set, the scan job only its conversation); the service opens a file only for whoever reads its message, after their `history_from_seq`, or before it is sent for its uploader only (service and app tests) |
| A search shows another patient's or a revoked conversation's messages | The index search (`communication_search`, `SECURITY DEFINER` because row-level security cannot use the text index) returns keys only, for what the scope may read by the same rules as the policies (migration test over every scope); the rows are then read through the policies, in a read-only role scope that checks active participation (never a pending invitation), `history_from_seq` and the consent, and for a doctor of another clinic the patient's chat consent for them and only the side threads they take part in (SAL-28); the service checks every result's conversation as it checks a page; tests on Postgres and through the API (after a revocation the practitioner finds nothing) |
| Receipts become a stored record of who read what | Only the reader's own position is stored (owner, 2026-11-12); receipts and typing live in Redis with a TTL and on the socket |
| A file name, a query or a message leaks into logs | No log line holds a file name, a search query or a message (app and worker tests check the logs); job arguments carry opaque ids only |

**Open before production.** The production object store and its storage classes (the "cheaper storage" for older
attachments is deferred until it is chosen); the retention of chat files with the clinical record (legal decision,
with the messages').

### SAL-29: notifications, chat hours and webhooks

**Processing added.** Three tables with personal data: `push_device` (the push token of a patient's app, by account
and platform), `webhook_endpoint` (a clinic's webhook: its URL, the events, whether it includes clinical content, the
sealed secret, and the admin's account that created it) and `webhook_dead_letter` (the ids of a delivery that failed
its last attempt). `organization_chat_settings` holds the clinic's chat hours, stated response time and e-mail delay:
no personal data. A new message enqueues a job with opaque ids only; the notifier fans it out to a push per device of
the patient's account, an e-mail check per other participant after the clinic's delay, and a delivery per clinic
webhook. The e-mail check reads the participant's stored read position.

**Purpose and lawful basis.** Telling a participant that a message waits for them: for the patient the contract
with them (Art. 6(1)(b)), for staff their employment and the contract with the clinic; the content stays in the app.
The webhooks are the clinic's own integration: the endpoint is the clinic's system and the clinic is the controller
of what it receives (the platform processes on its behalf); with clinical content turned on, the message's text goes
to the clinic's system under the same care basis and consent that let the clinic read the chat, and only while that
consent is active.

**Necessity and minimisation.** A push says "New message from your clinic" and nothing else: no text, sender, name,
clinic or id (no data payload). The unread e-mail says the same with the link to the app (a patient) or the web app
(a practitioner). A webhook carries the event's type, its stable id, the clinic's and the conversation's and the
message's ids; the message (text, time, sender role, patient id) only when the clinic's admin turns clinical content
on. Job arguments, error texts and log lines hold ids, codes and statuses only, proven by a test that sends a
recognisable synthetic text through the chat, the notifier and the worker and finds it nowhere outside the record.

**Retention.** `push_device`: kept while the app is registered; deleted on unregister (sign-out), when FCM or APNs
says the token is gone, and with the account. `webhook_endpoint`: kept while the clinic keeps it; deleted by its
admin, with its dead letters. `webhook_dead_letter`: 30 days.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A notification discloses the conversation to whoever sees the phone's lock screen or the inbox | Fixed text with no content, sender or clinic, in the recipient's language; no id in the push |
| Google or Apple (outside the EU/EEA) learn about the patient | They receive the device token and the fixed text only. They are processors pending their DPA and transfer basis (docs/privacy/processors.md); the adapters are off and the fake is the default until then |
| A device signed into another account receives the first account's notifications | A token is one row: registering it moves it to the account signed in last; sign-out deletes it |
| A webhook URL reaches the platform's own network (SSRF) | https only, a host name with a dot or a public address at creation; the delivery's dialer refuses loopback, private, link-local, unique-local, multicast, unspecified and carrier-grade NAT addresses at connect time, on the resolved address, so a name rebound later is refused too; no redirect is followed; no proxy (tests) |
| A forged or replayed delivery is believed by the clinic's system | HMAC-SHA256 over the timestamp and the exact body with the endpoint's own secret; the timestamp is in the signature; the event id is stable for deduplication |
| The webhook secret leaks | Shown once at creation; stored sealed with `WEBHOOK_KEY` (AES-256-GCM bound to the endpoint), a key of its own in the root-only key file (ADR 0011); never listed, logged or audited |
| Clinical content leaves the platform after a revocation | A delivery with clinical content reads the message in the clinic's scope, which shows nothing after the revocation: the delivery is cancelled |
| The admin's changes go unrecorded | Every change of the chat settings and every list, creation and deletion of a webhook is audited, refusals too |

**Open before production.** The DPAs and transfer basis of Google (FCM) and Apple (APNs) before either is switched
on (docs/deploy/push.md).

### SAL-31: the chat's screens and the AI summary

**Processing added.** No new table or stored field. The web app (doctors) and the patient app show the conversations
of SAL-27 to SAL-30; nothing of a message is kept on the device beyond the screen's memory (no local database, no
browser storage of message text). `POST /conversations/{id}/summary` gives a practitioner who may read a long
conversation (at least 50 messages they read) a summary of its latest 500 messages, generated on demand, never stored,
labelled AI generated, shown above the messages, which stay as written; its read of the rows is audited as a read
(`AuditEvent`, the patient, the number of rows).

**Lawful basis.** The summary runs on the fake engine only (`llmtest.Summarizer`: a placeholder naming how many
messages it covers; no text leaves the api process). `config.LoadChatSummary` refuses any other engine
(`CHAT_SUMMARY_ENGINE`), whatever `LLM_ENGINE` says, because the patient's messages may reach a third-party model only
under a basis the owner has not decided yet: SAL-12's transcription consent covers the consultation, not the chat.
Waiting for the owner: the basis (a granular AI consent, or another), then the processor's DPA and an entry in
`processors.md`, before a real engine is wired.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A patient's messages sent to a hosted model without a basis | The summary's engine is the fake and the config refuses another; the Summarizer port is the only path |
| The summary taken for the record, or for a diagnosis | Not stored; always labelled AI generated; the messages stay below it unchanged; the copy states what was written, never a diagnosis |
| The patient sees the doctors' summary or side threads | The summary is refused to the patient (403); the patient app has no summary or thread screen; side threads stay hidden by row-level security (SAL-28) |
| A cross-clinic doctor joins without the patient knowing, or keeps access the patient cannot find to withdraw | The patient app's consent prompt (strings tagged `legal`) names the doctor and their clinic (grant stays disabled until they are shown), why, and what they would read; grant or decline; the chat view lists every granted doctor of the conversation from the API (`status=active`), so each can be revoked at any time, also after a restart (SAL-28's consent) |

## Analytics (`docs/plans/analytics.md`, ADR 0013)

### SAL-19: coded diagnoses and diagnosis groups

**Processing added.** `condition`: one FHIR Condition per filled coded field of a signed document (an ICD-10 code,
the patient, the encounter, the clinic, the signing practitioner role, the signing time): health data (Art. 9)
already in the signed document, now kept as a coded record. The documents signed before this feature got theirs
by a backfill, marked unconfirmed and not counted. The diagnosis groups and the places are reference data with no
personal data. A new database role, `salus_stats`, may read the source columns of the future aggregates (codes,
times, statuses, clinic, practitioner role, and the patient id of `condition` and `appointment`); no binary logs in
as it yet.

**Purpose and lawful basis.** The clinical record (Art. 9(2)(h)), as the signed document it comes from; it is the
source of the clinic's statistics (SAL-20) on the clinic's behalf. The platform-wide figures (SAL-21) are behind a
feature flag that stays off until the owner confirms the legal answers (plan, L1 to L8).

**Necessity and minimisation.** The columns are listed in the plan, §3, each with its reason. No free text, no
transcript, no questionnaire answer is read. `salus_stats` reads no name, contact detail, document field or
transcript (tested in the migration); a patient id leaves `condition` and `appointment` only as HMAC-SHA256 under a
key that lives for one build (`pkg/pseudonym`).

**Access.** Row-level security: the clinic's scope sees its own Conditions and inserts them, the patient's scope
sees their own, a video room's scope and an account's see none; the application never updates or deletes one
(tested, migration 00042). Signing refuses a filled diagnosis code the practitioner did not confirm.

**Audit.** Each Condition's creation is audited with the signature, in its transaction. The group-name reviews are
audited on `diagnosis_group_name`, also when refused.

**Retention.** `condition`: kept with the clinical record, as `document_draft` (a legal decision before production,
as the other clinical tables).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| An invited doctor reads the conversation before the patient consents, or after a revocation, a decline or a removal | Row-level security: a doctor of another clinic (their clinic and role set) sees a conversation, and those of its side threads they take part in, only while they are an active participant, the patient's chat consent for them is active and the care consent with the treating clinic is active (never another doctor's pending invitation); one `SECURITY DEFINER` function, owned by a role that reads only the columns it needs, decides it (migration tests over every state and scope). The service checks `access.GuestChat` as well (service tests on fakes and on Postgres, app test). An open socket checks again every 5 s |
| The patient sees a doctor-only side thread | The patient's policies exclude every `staff_only` conversation and its messages; the service refuses the side threads' list and their creation to the patient (service tests on every path, app test) |
| A doctor removed from a conversation keeps its side threads | A trigger removes them from the conversation's side threads in the same statement, whoever removes them (migration test) |
| A doctor is added who should not be | Only the treating practitioner adds and removes (`access.ManagesChat`); a doctor of the clinic must be an active role of it, one of another clinic in the public directory |
| No proof of what the patient decided | Every invitation, add, removal, grant, decline and revocation is audited in its transaction, naming the participant by its opaque id; a refused revocation too; a grant is a stored Consent with its time, a revocation keeps the row |
| A consent left granted to a doctor who was removed, which the patient can no longer see or withdraw | A removal by the treating practitioner ends the patient's chat consent for that doctor in the same transaction, as the clinic, audited (the clinic may end a chat consent, never grant one: migration test); the patient lists their granted invitations (`GET /me/chat-invitations?status=active`) and revokes each (Art. 7(3)) |

**Open before production.** The retention of `chat_consent` with the clinical record (legal decision).

| A model-suggested code recorded as a diagnosis nobody made | The practitioner confirms each code before the signature (tested in the service, the API and the browser journey); the backfilled codes are unconfirmed and not counted |
| Another clinic's diagnoses | Row-level security by clinic (tested) |
| The aggregate builds reading identities | `salus_stats` has column grants on the source columns only (tested); patients are pseudonymised per build |
| WHO or national-edition text used without a licence | Codes only, as identifiers; group names are our own (ADR 0013) |

### SAL-20: the clinic's own statistics

**Processing added.** `clinic_appointment_stat` and `clinic_diagnosis_stat`: counts per clinic, local day and hour,
practitioner role and its specialty, and appointment status or diagnosis group, subgroup and ICD-10 code. No patient
identifier is stored or pseudonymised: the builds ask the sources without a key. The source queries
(`ListBookedForStats`, condition's `ListDiagnoses`) are shared with SAL-21, which pseudonymises the patient with its
key, so they do select `patient_id` as `salus_stats`; without a key the source services drop it in Go before the
build sees a row, and it is never written, logged or counted. The counts are personal data
of the practitioners (their workload and the diagnoses they recorded). Computed by the `clinicstats_refresh` job (a
clinic-day, enqueued in the transaction of every appointment write and every signature with a diagnosis, with the
clinic's id and an instant only) and the nightly `clinicstats_nightly` job, as `salus_stats`, which `cmd/worker` now
logs in as (`STATS_DATABASE_URL`).

**Purpose and lawful basis.** The management of the clinic's health-care services, on the clinic's behalf as
controller, Salus as processor (Art. 9(2)(h)). Not used for the platform figures (those are built from the sources,
SAL-21), for marketing, profiling or model training.

**Necessity and minimisation.** Only counts; no age, sex, residence or free text. Only confirmed diagnoses count.

**Access.** Only the clinic's staff, in the service (rules C1 to C3, plan §4): the admin sees everything of their
clinic, the breakdown per practitioner included; a practitioner who is not an admin sees the clinic's appointment
figures (or their own role's, never a specialty's, which can be one colleague's) and only their own role's diagnoses; anyone else is refused. Row-level security by
clinic on both tables (one clinic's scope reads none of another's rows, tested); only `salus_stats` writes them
(insert and delete; tested in migration 00048).

**Audit.** Every read and every refusal writes an AuditEvent on `Organization/<clinic>`, purpose `HOPERAT`
(v3-ActReason has no code for statistics), in the read's transaction. The builds are not audited: they read and write
counts only, on behalf of no one.

**Retention.** Kept as long as their sources (appointment and condition, with the clinical record) and recomputed
from them; a refresh replaces its days. When erasure and restriction are built (`datarequest`), they enqueue the
refresh of the affected days.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A practitioner learning about patients outside their care through the clinic's diagnosis totals | C2: their own role's diagnoses only, another role's refused (tested per role) |
| Small counts singling out a patient within the clinic | The figures stay inside the clinic that holds the records; no count leaves it (the platform view is SAL-21's, with its own suppression) |
| Per-practitioner workload as employee monitoring | Admins only; open question L5 for the owner's lawyer before production |
| An admin learning one patient's diagnosis: a one-day range, one practitioner role and `by=code` (J45: 1) next to the calendar showing that practitioner saw one patient that day | **Open, for the owner** (review finding 2): the figures stay inside the clinic that already holds the record, and every read is audited on the clinic; options are a minimum range, or month granularity, when an admin combines a role with a code breakdown |
| Stale or lost counts | The nightly pass recomputes the last 35 days of every clinic; the refresh is idempotent (tested) |

### SAL-21: platform-wide anonymized trends

**Processing added.** The clinic's location (a city and its leaf region, places of `terminology`) and whether it
contributes to the platform figures, each with the time it last changed, on `organization`: no personal data. The
builds (`internal/trends`, in `cmd/worker` as `salus_stats`, or `salusctl trends build`) read every contributing
clinic's confirmed Conditions and fulfilled appointments of a closed month, quarter or week, with each patient only
as HMAC-SHA256 under 32 random bytes made for the build and dropped with it; inside the build a patient and a clinic
are an int32 that never leaves memory. They write `trends_cell` (distinct-patient counts per period and table cell,
rounded to 5, or suppressed) and `trends_rise` (region, group and week of an unusual rise, no count): no clinic,
practitioner or patient column (A1, a test walks every response's JSON against an allow-list).

**Purpose and lawful basis.** Platform-wide figures of diagnoses and appointments by region, city, specialty and
month, for the staff of the clinics that contribute. This is a new purpose for which Salus would act as controller
of the anonymisation step: **L1 (role) and L2 (lawful basis) are open** with the owner's lawyer, so the whole view
and its builds are behind `ANALYTICS_PLATFORM_ENABLED`, off by default (§2a): off, the scheduler enqueues no build,
`salusctl trends build` refuses, the routes are not mounted (404) and no figure is computed. L3 (whether the output
is anonymous, and may stay after an erasure) is open too. Not used for marketing, profiling or model training.

**Necessity and minimisation.** Only confirmed diagnosis codes and fulfilled appointments' times, through the
owning services; no age, sex, residence or free text. Geography is the clinic's location, never the patient's (A2).
A clinic counts in a period only when its location and its contribution did not change during it; the rises only
with the same location through all ten weeks (§5, §6). A clinic may opt out (O6): it is left out of every build from
the next one on, and its staff lose the view (`403 analytics_not_contributing`).

**Anonymisation (plan §5, rules A1 to A13, as built).** Each closed period is built once and frozen (A3). A cell is
published only with at least k patients (`ANALYTICS_MIN_COUNT`, at least 10) from at least 3 clinics and at least
max(k, 0.1·x1) left once its two largest clinics are taken out (A4); every relation of every table (parents and
children along each dimension, cells shared between tables) has none or at least two suppressed cells whose distinct
union passes A4 (A6); every suppressed cell is audited as an LP over its neighbourhood, with the published values as
[v-2, v+2], the inequality relations of distinct counts and the two largest clinics' own patients known, failing
closed (more suppression, then no code level, then the period refused, A7); a code is a key only when published at
geography = all (A8); a breakdown returns every key with null for 0 and suppressed alike (A8); sums over months are
patient-months marked partial (A9); counts are rounded to 5 (A10). An exact integer check over 60 random synthetic
months (`internal/trends/exact_test.go`) confirms the LP never passes a month whose integer interval is narrower
than k.

**Access.** Staff (admins and practitioners) of a contributing clinic only (O5); a patient gets 403. salus_app only
reads the figures; salus_stats writes each period once; nobody but the owner changes or deletes them (migration
00052, tested).

**Audit.** Every read of the view, refused ones too, is an AuditEvent (entity `trends_cell` or `trends_rise`,
purpose HRESCH). Every build is audited before it writes: its agent (`trends_build` or `salusctl`) and the number
of Conditions or appointments read. Build logs carry the period and counts of cells, suppressed cells and rises
only (A13).

**Retention.** `trends_cell` and `trends_rise`: deleted five years after their build (retention rule, purged by
`salus_retention`); the period row stays, so a period is never built again, and is no longer listed once its cells
are gone. `organization`'s new columns: with the clinic. While L3 is open, `trends_cell` and `trends_rise` are marked
`personal_data: true` in `docs/architecture/archcheck/ownership.yaml`: the safe choice, so they keep a retention rule
and count as personal data until the lawyer says the output is anonymous.

**Pending L3.** The Clinic settings copy calls the figures "anonymized" (`admin.contributes`, `admin.contributesHint`,
tagged `legal`): the wording waits for L3 and the owner's approval. Until the view is on, the opt-out and the
location hint are not shown.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| Re-identification through small counts | A4 with k >= 10, 3 clinics and the two-largest rule; rounding to 5 |
| Differencing a parent and its other children, or two tables, or several queries | Suppression decided once per period over every relation of every table (A5, A6); frozen periods (A3); only the fixed table combinations are queryable |
| Two large clinics recovering a small third | Their own patients treated as known in the A7 audit |
| Telling 0 from 1 | Every key returned, null for 0 and suppressed alike (A8) |
| A clinic moving or opting out | Left out of every period it changed during, and of the rises for ten weeks |
| A rise flag read as an official alert | No count; copy "Unusual rise" with a tooltip that it is not a public-health alert (ANA-4); L6 open |
| Processing before the legal answers | `ANALYTICS_PLATFORM_ENABLED` off by default; tests run both ways |

### SAL-22: the analytics charts (web)

**Processing added.** None on the server. The staff web app gets an Analytics page that shows what the clinic and
platform endpoints of SAL-20 and SAL-21 return: counts per month, weekday and hour, diagnosis group, region, city,
specialty and, for a clinic admin, practitioner, through the generated client (`web/src/analytics/data.ts`); every
read is audited by those endpoints. The tests use synthetic placeholder figures (`web/src/analytics/fixtures.ts`).

**Minimisation.** The page asks only for counts and keys (codes, place ids, role ids); names come from the reference
endpoints. Nothing is stored in the browser and nothing is logged. A practitioner who is not an admin is offered no
practitioner or specialty filter (the API refuses both) and no practitioner breakdown, and is told the diagnoses shown are the
ones they recorded (rule C2).

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| The page asks for a platform combination that would undo the suppression | The All of Salus filters follow one table at a time (plan §5: geography × diagnosis, geography × specialty, specialty × diagnosis); a dimension outside the table is not offered (tested against fixtures that refuse it) |
| A hidden cell read as a small number or as zero | It reads "Hidden" with a hatched mark and the tooltip that small or dominated groups are hidden; a sum over months is labelled patient-months and "Partial" when a cell is hidden |
| A rise flag taken as an official alert | "Unusual rise", never "outbreak", with the tooltip that it compares with the previous weeks and is not a public-health alert (legal-tagged string) |
| The platform view before the legal answers | The tab is shown only with the `analytics_platform` feature of `GET /auth/me` and for a contributing clinic |

## Patient history (`docs/plans/patient-history.md`, ADR 0014)

### Step 0: the contract (migration 00043)

**Processing added.** The patient's allergies and current medications in their profile (FHIR AllergyIntolerance,
MedicationStatement), and a third consent, the history consent, by which the patient shares chosen visits of theirs
at other clinics with one clinic. Lawful basis: care (Art. 9(2)(h)) for the profile, as for the rest of the record;
the patient's explicit consent for sharing across clinics, granular per visit, off by default, revocable at once. No
processor is added; nothing of the history goes to a language model.

**Personal data added.** The four tables of the plan's "Personal data" section, all health data, all kept with the
clinical record (retention `keep`, needs a legal decision before production). A patient's own unconfirmed entry is
deleted when they remove it; anything confirmed or recorded by a clinic is kept as entered-in-error.

**Who sees what.** ADR 0014 "Access model": the patient their whole history; a practitioner only the visits they
held themselves at their clinic and the visits the patient shared with that clinic; the profile where the care
consent and an appointment allow it (as the profile today); a clinic's entry only where its visit is seen; which
clinic recorded or confirmed an entry only to the patient and that clinic. Row-level security says the same in the
clinic's scope (tested per scope and table); the service rule (`access.HistoryVisit`) narrows it to the
practitioner. Every read, grant, change and revocation is audited.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A visit the patient did not tick reaches the doctor | It is never selected: `history_shared` and the `*_shared` policies open only listed visits, `access.HistoryVisit` refuses the rest before anything is listed, counted, searched or summarised (migration and access tests; service tests in SAL-40) |
| A removal or revocation that does not take effect at once | Both end access when their transaction commits, in row-level security and in the rule, which read the current state (tests); revoking the care consent revokes the history consent too |
| A doctor's entry in the profile tells another clinic where the patient was treated | Doctor entries belong to their visit (ADR 0014 decision 4); the patient adds one to the profile only on purpose, and then it shows no clinic, no practitioner and no visit (`access.Attributed`, FHIR golden files) |
| The patient shares another patient's visit, or one of the receiving clinic's own, or one still in progress | Refused by the insert check of `history_consent_encounter` as well as the service |
| A shared visit exposes more than its record | Only the encounter, signed documents, completed answers and their clean files, and the entries recorded in it; never the transcript, suggestions, drafts or appointment (policy tests) |
| History reaches the language model | Not in this epic: suggestions and drafting read only the consultation's transcript (SAL-42 adds the test) |

### SAL-39: the patient's timeline

**Processing added.** The patient reads their own history across every clinic (`GET /me/history`, the mobile app's
"My history"): their visits, signed documents, completed questionnaire answers and the clean files of those, newest
first, one page at a time. Their own record (Art. 15); no new personal data is collected and no table is added.

**Who sees what.** Only the patient, of their own data, in their own row-level security scope; staff are refused.
Never a draft, a response in progress or withdrawn, or a file not yet found clean. Clinic names are read for the
clinics of the patient's own items only (no personal data in a clinic). Chats are not listed yet.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A draft or another patient's item in the timeline | Each owning domain reads in the patient's scope, filters by the patient and applies its own rule (`access.HistoryVisit`, `access.SignedDocument`, `access.QuestionnaireResponse`, binary's own check); service and repository tests per domain, the e2e journey checks the open draft is absent |
| A read that is not audited | Every item a source returns is audited by its owning domain in its transaction (an empty source is one event without an entity). Each source is asked for one item more than the page and the page is cut from their merge, so what a source returned past the cut is audited without being shown (the next page shows it, audited again): the record errs towards more, never less |
| History details in logs or the cursor | The cursor is an opaque date and reference (no personal data beyond the patient's own resource id); the handler logs nothing of the query or the items |

### SAL-40: the practitioner's view and the pre-visit summary (migration 00051)

**Processing added.** No new personal data and no new table: the practitioner's view of the patient's history
(`GET /patients/{id}/history`) and the pre-visit summary of one of their appointments (`.../history/summary`) read
what the owning domains already hold, in the web app on the appointment (opened on request) and the call page.
Migration 00051 adds two `SECURITY DEFINER` lookups. `history_questionnaire` lets a practitioner read the questions of a
completed response that was shared with their clinic: it returns that questionnaire version only (clinic content, no
personal data), never another of the other clinic's versions, and nothing of any response.
`history_confirmed_conditions` says which coded fields of a shared signed document have a confirmed Condition (the
document and field key only), so a code that was never confirmed (00042's backfill) is not shown as a diagnosis. Both
answer only a practitioner with an appointment with the patient at the clinic.

**Who sees what.** Only a practitioner of the clinic with an appointment with the patient under an active care
consent has a view (`access.Profile` at Full, then `access.HistoryVisit` per visit, in `encounter.Service.History`);
the clinic's admin, the patient and anyone else get 404. The view lists only those visits and what belongs to them;
a visit that is not shared is in no item, count, summary or empty state. A shared visit's signed documents, completed
answers, their questions and their clean files open for that practitioner through the same rule; never its drafts.

**Audit.** Every view writes an AuditEvent of the view (the patient as entity, purpose TREAT) and one per item shown,
by its owning domain, in the same transaction; reading another clinic's shared document or file is audited under the
reader's clinic, so the visit's clinic is not told whom the patient shared it with.

| Risk | Measure |
| --- | --- |
| A colleague's visit or an unshared visit shows in the view, its count or the summary | Filtered by `access.HistoryVisit` before anything is listed (service tests over every viewer; API test on the real database: the unshared visit's ids and diagnosis are nowhere in the answer) |
| The view outlives a removal or a revocation | It is read anew on every page and every open (API test: after the removal the visit and its document are gone at once) |
| The questions of another clinic leak beyond the shared response | `history_questionnaire` returns only a version a completed, shared response answers, and only to the receiving clinic's scope and a practitioner there with an appointment with the patient (migration test) |
| An unconfirmed code reads as a diagnosis | Each coded field carries its label and its Condition's confirmation; an unconfirmed one is marked "Not confirmed" (service, API and migration tests) |
| An unread allergy list reads as "no allergies" | A failed read marks the section unavailable, never empty; the empty text says "shown to you", since the visit-scoped rule may hide a colleague's entry (service, handler and web tests) |

### SAL-41 follow-up: the Provenance of the clinical writes (migration 00053)

**Processing added.** Who recorded, changed, confirmed or removed an allergy or a medication, and when: one FHIR
Provenance per practitioner's write and per patient's write on a clinic's entry (`allergy_intolerance_provenance`,
`medication_statement_provenance`), written in the write's transaction, append-only. Lawful basis: care
(Art. 9(2)(h)), as the entry it attests. No processor is added. The patient's writes on their own entries have none
(the entry names the patient as its source, the AuditEvent records the write).

**Personal data added.** The entry, the version written, when, the activity, and the agent (the practitioner role and
its clinic, or the patient). Kept with the entry (retention `keep`) and never deleted with it: an entry with a
Provenance is removed as entered-in-error, not deleted.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A Provenance tells another clinic where the patient was treated (an entry in the profile, or one of a shared visit) | It is read only by the patient and by the writing clinic's staff who see the entry (`access.Attributed`); row-level security shows a clinic only its own Provenances of entries it sees (migration and service tests) |
| A clinic's Provenance is lost with the entry | The foreign key has no action, so deleting an entry with a Provenance fails; the service deletes only an entry no clinic confirmed or wrote; the application can neither change nor delete a Provenance (no grant, tested) |
| A forged Provenance (another role, another write) | Each insert is tied to the write it attests: the entry at that version and in the state it left (confirmed for a confirmation, entered-in-error for a removal, in the profile for the patient's addition), the recorder for a recording, the confirmer for a confirmation, the entry's clinic for a change or removal, the entry's patient for theirs, and one Provenance per version (migration tests). Which role of a clinic changed or removed its clinic's entry is bound by the service only (the caller's role); the database checks the clinic |
| A removed entry's record is withheld from the patient | A removed entry and its Provenances are shown to no one in the app, but they are the patient's data: the Art. 15 access and portability export (datarequest, `Patient/$everything`, not built yet) must include them; recorded as a requirement in docs/plans/patient-history.md |

## Appointments (`docs/plans/appointments.md`, ADR 0015)

### Step 0: the contract (migration 00054, renumbered at merge)

**Processing added.** Whether a patient confirmed an appointment and how (a one-time link or the app); the history of
an appointment's changes (confirmation, reschedule, a status set or corrected, by whom and how); one-time action
links sent in reminders; the clinic's appointment settings (reminder times, the SMS option, rescheduling, the
waitlist, the repeat no-show flag: not personal data). Lawful basis as for the appointment: the contract with the
patient (Art. 6(1)(b)) and the clinic's care record (Art. 9(2)(h)). The patient's phone may carry SMS reminders and
waitlist offers, opt-in per patient and channel, never marketing (owner, 2026-11-13, Q1; docs/booking/SPEC.md §4). No
processor is wired in: `pkg/sms` and `pkg/payments` run their fakes; the SMS provider is planned
(docs/privacy/processors.md) and nothing is charged.

**Personal data added.** `appointment.confirmed_at`, `confirmed_via`, `reschedule_count`, `booked_via`;
`appointment_change` (kept with the appointment); `appointment_action_token` (the SHA-256 of a link's token, bound to
one appointment, its patient and account, the channel and the language; deleted 30 days after it expires). The
periods are placeholders that need a legal decision before production (Q9), as the booking's.

**Who sees what.** The history of an appointment: the patient their own (without who or which job made a change), the
clinic's staff with full access to the appointment (`access.Appointment`), as row-level security says (the patient's
own; the clinic's while the care consent is active, and the past ones after a revocation). A link's token is found
only through a `SECURITY DEFINER` function by its hash; its row is read in the issuing job's scope or the patient's.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A link's token leaks from the database or a log | Only its SHA-256 is stored; the route pattern is logged, never the path; the audit names the appointment, never the token |
| A link prefetcher or a forwarded message changes an appointment | A `GET` never changes anything (SAL-34's misuse tests); an action is single use, expires at the start, and touches only its appointment |
| A history row is altered to hide a correction | `appointment_change` is append-only: no `UPDATE` or `DELETE` grant, triggers refuse both and `TRUNCATE` for every role |
| A patient learns who at the clinic changed their appointment | The patient's view of the history carries no account or agent |
| A history row claims a staff member or a patient's app it was not | The insert policy ties `via` to the scope: the patient's scope writes `app` or `link` by the patient's own account (or a job), the clinic's scope `staff` by an account of that clinic (or a job) |

### SAL-33: reminders (migration 00055, renumbered at merge)

**Processing added.** Appointment reminders at the clinic's offsets (default 24 hours and 1 hour before) by push,
e-mail and, at clinics with the paid option and for patients who opted in, SMS; the patient's notification settings
(channels, quiet hours, the device's time zone); per clinic, how many SMS messages and parts went each month. Lawful
basis: the contract with the patient (Art. 6(1)(b)) for the reminders and the settings; the clinic's legitimate
interest in billing the SMS option (Art. 6(1)(f)) for the usage, which is not personal data. The phone is read only
for an SMS the patient opted into (Q1), through `patient.NoticePhone`, audited with the job as agent.

**Personal data added.** `appointment_reminder` (when each reminder of an appointment is due and whether it went;
deleted 90 days after its due time); `notification_delivery` (which channel and device a reminder went through and
when, no address, number, token or text; deleted 90 days after it was sent, or created when it never was, and with the
account); `notification_preference` (the patient's channels, quiet hours and the IANA zone of their device, which can
show that they are abroad: kept to the zone name, replaced on each change, deleted with the account). The periods are
placeholders that need a legal decision before production (Q9).

**What a message says.** E-mail: the clinic's name, the date and time in the clinic's zone with the zone's name and,
when the patient's zone differs, in theirs too, whether a questionnaire is outstanding (never which), a one-time link
to confirm or cancel, and the link to the app. SMS: the clinic's name, the date and time in the clinic's zone and the
one-time link, with the zone's name, and the patient's day and time when their zone differs. Push: a fixed "You have an appointment reminder" with no data payload, as the chat's (Google and Apple
are outside the EU/EEA). Never a practitioner, specialty, reason for the visit or anything clinical; job arguments carry
ids only; nothing of a message is logged.

**Who sees what.** The patient reads and changes their own settings (staff get 403), audited on their account. A
reminder is seen in its patient's scope and its clinic's (ids and times only); the hourly sweep reads the scheduled
ones of every clinic through a view of ids and times. The SMS usage is the clinic admin's, in the clinic's scope.
### SAL-35: the waitlist (migration 00056, renumbered at merge)

**Processing added.** A patient joins a clinic's waitlist for one of its practitioners or a specialty, between two
dates and, optionally, a time of day (`waitlist_entry`: patient, clinic, role or specialty, dates, times, priority,
status, timestamps). When a slot is freed (a cancellation, a reschedule, a declined or withdrawn offer), the worker
holds it (`slot.status` `busy-tentative`) for the first matching entry for the clinic's hold (15 minutes by default) and
records the offer (`waitlist_offer`: the entry, slot and its period, the role, clinic, patient, expiry, state and the
appointment its acceptance booked). The patient is told by e-mail (clinic, date and time, the hold's end, a link to the
app), by a push with a fixed text and no data payload, and by SMS when the patient opted in and the clinic offers it (SAL-33's deliveries, the same channels, quiet hours and records as reminders); they accept in the app, signed in, which books the
appointment (`booked_via` `waitlist`) through the purchase step (nothing is charged before SAL-23), or decline. The
clinic's staff see the waitlist in offer order and move an entry up or remove it. The clinic turns the waitlist on
(off by default). Lawful basis: the contract with the patient (Art. 6(1)(b)); that a patient waits for a practitioner
or a specialty is health data, processed for care (Art. 9(2)(h)) under the patient's active care consent, checked when
they join and again before every offer and acceptance.

**Personal data added.** `waitlist_entry` and `waitlist_offer` as above, both deleted 30 days after the entry was
booked or withdrawn, or after its last day (the offers with their entry). No new field of the patient is read: the
e-mail address and the push devices are the existing ones. The periods are placeholders that need a legal decision
before production (Q9).

**Who sees what.** The patient their own entries and offers. The clinic sees its entries while the patient's care
consent there is active (row-level security, as the appointment's), and the patient's name only where `access.Profile`
gives it (the admin with the consent; a practitioner only for patients with an appointment in their role); the clinic's
jobs see every offer of the clinic, to hand a slot on, whatever the consent. Every read and write is audited, the
jobs' with the job as agent (the entry as `Appointment/<entry id>`, FHIR's waitlist being an Appointment status).
The public directory never sees a held slot. Job arguments carry opaque ids only.

**Risks and measures.**

| Risk | Measure |
| --- | --- |
| A reminder reveals health data to Google or Apple | The push is generic, without clinic, time or payload |
| A reminder goes twice, or after a cancellation or reschedule | Dispatch and its deliveries in one transaction; a delivery leaves pending once and a reminder scheduled once (a dispatched one is only retired, with its appointment's cancellation or move), by triggers for every role; each send re-reads the reminder and the appointment and sends nothing once it no longer holds at the start it was scheduled for. Left: a crash between the provider's acceptance and the commit repeats one message, and an e-mail or SMS repeated carries a second valid one-time link (ADR 0015's accepted window; only a link's hash is stored, so the first cannot be reused) |
| A reminder on a channel the patient turned off | The patient's channels, and the clinic's SMS option, are checked at dispatch and again at each send: a send held by quiet hours is cancelled once its channel is off |
| A reminder at night | Quiet hours in the patient's zone (else the clinic's) hold push and SMS until they end, or drop them when that is under 30 minutes before the start; e-mail is silent and never held |
| The zone shows the patient is abroad | Only the zone name is stored, replaced on each change, read by the notifier's jobs (audited) and the patient; deleted with the account |
| SMS to someone who did not ask | SMS needs the patient's opt-in with a phone in their profile (409 `phone_required` otherwise) and the clinic's paid option, which cannot be turned on while `pkg/sms` is the fake; the provider is planned, not wired in (docs/privacy/processors.md) |
| A freed slot is booked twice, or held while it is booked | The offer's row lock serialises acceptance and expiry; the slot moves only from the status before it (free to held, held to busy or free), its row locked by the job that hands it on; one open offer per slot and per entry (partial unique indexes); the exclusion constraint guards last. Concurrency tests on Postgres check after every round that a slot is busy exactly when an appointment holds it and held exactly when an open offer does |
| An offer reveals the patient's health to others | The e-mail names the clinic, the date and time only, never a practitioner or specialty; the push is generic; the offer is accepted only in the app, signed in |
| A patient keeps being offered after revoking the consent | The `consent_revoked` job withdraws their entries and open offers there and hands the slots on; every offer and acceptance checks the consent again |
| A waitlist that a patient forgot keeps their data | Entries end after their last day (at most 90 days ahead) and are deleted 30 days after they close or end, with their offers |
| A held slot stays hidden after its offer ended | Every decline, withdrawal and expiry hands the slot on in a job that locks it; with nobody matching it is free again |
| An offer disturbs a patient at night or by an unwanted channel | Offers go through SAL-33's deliveries: the patient's channels and the SMS opt-in apply, a push or SMS inside quiet hours is held or dropped, and a patient inside their quiet hours is passed over for an offer |
