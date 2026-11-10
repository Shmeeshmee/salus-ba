# Accounts and the clinic directory

Status: approved, 2026-11-04, with the owner's changes (section 2, "Changed on approval"). Branch `feat/accounts` from `main`, one commit per
package (`accounts(<n>): <summary>`), one PR at the end.

## 1. Goal

People can sign in. A patient registers once and uses one account with any clinic. A clinic (FHIR
`Organization`) has admins and practitioners. An admin invites a practitioner by e-mail, and the practitioner
signs in with a password and TOTP. The directory (`Organization`, `Practitioner`, `PractitionerRole`) is
readable over FHIR. Every access to an account and to directory data is audited, and the privacy foundation
(append-only `AuditEvent`, retention rules, processor, breach and DPIA notes) exists before any personal data
is stored.

## 2. Settled decisions

Given by the owner. Do not reopen them.

| Topic | Decision |
| --- | --- |
| Identity provider | Auth is built in the identity context (`internal/auth`). No Keycloak, Zitadel or other IdP |
| Account scope | A patient has one account across every clinic. Admins and practitioners belong to exactly one clinic (`Organization`) |
| Tenancy | Shared tables, `organization_id` on every clinic-scoped table. Postgres row-level security is the second guard: the service sets the clinic on the transaction and RLS enforces it |
| Roles | `admin`, `practitioner`, `patient` |
| Passwords | argon2id. Parameters chosen by a benchmark, numbers recorded in `PROGRESS.md` |
| Sessions | Stored in Postgres; revocation is `session.revoked_at` in Postgres only (changed on approval, below) |
| MFA | TOTP. Required for practitioners and admins, optional for patients |
| Personal data | Every field listed in section 4 with the reason it is needed |

Decisions taken in this plan (reopen only with the owner):

- **Package names follow CLAUDE.md.** A domain is one package named after what it owns; contexts are lists.
  This feature adds `auditevent` and `retention` (context `privacy`), `auth` (context `identity`),
  `organization`, `practitioner` and `practitionerrole` (context `directory`), and `notification` (context
  `notification`). `config.yaml`, `ownership.yaml` and `sqlc.yaml` gain each one in the commit that adds it.
- **Database roles.** `salus_app` is its own `LOGIN` role with its own credentials: not a superuser, no
  `BYPASSRLS`, `CREATEROLE` or `CREATEDB`, owner of nothing. `api`, `worker`, `notifier` and `scheduler`
  connect as `salus_app` (their `DATABASE_URL` names it). Only `migrate` and `salusctl` connect as the owner
  (`salus` locally). The first migration of this feature creates `salus_app` (`LOGIN`, no password) when it
  is missing (a `DO` block that ignores `duplicate_object`, since roles are cluster-wide and test packages
  migrate in parallel); each later migration grants it only what the application needs on each table. The
  password is never in a migration: `cmd/migrate up` sets it with `ALTER ROLE` when `APP_DATABASE_PASSWORD`
  is set (compose and CI: a synthetic value; production: from the secret store, or provisioned outside
  migrate and the variable left unset). `pgtest` gains a helper that connects to a test database as
  `salus_app`. Because `salus_app` owns no table, RLS applies to it without `FORCE`. A test proves
  `salus_app` cannot bypass RLS (no `BYPASSRLS` in `pg_roles`; `SET row_security = off` makes an RLS-guarded
  query fail; another clinic's rows stay invisible) and can only `INSERT` and `SELECT` on `audit_event`
  (`UPDATE`, `DELETE`, `TRUNCATE` denied by privilege, before the triggers are reached).
- **Setting the clinic.** `postgres.SetLocal(ctx, "app.organization_id", id)` runs `set_config(..., true)`
  inside the current `WithTx`. Policies compare `organization_id` with
  `current_setting('app.organization_id', true)::uuid`; when it is unset the comparison is NULL and no row
  matches, so a forgotten setting fails closed. Clinic-scoped queries always run inside `WithTx`.
- **Before the clinic is known.** Two lookups have to find a clinic-scoped row first: accepting an invite by
  its token, and a notifier job. The job carries the organization id with the invite id (both opaque). The
  token lookup goes through one `SECURITY DEFINER` function,
  `invite_organization(token_hash bytea) RETURNS uuid`, which returns only the clinic id, with
  `SET search_path = pg_catalog, public` on the function and schema-qualified names in its body. It is owned
  by `salus_invite_lookup`, a `NOLOGIN` role that is not a superuser and has no `BYPASSRLS`: it holds
  `SELECT (organization_id, token_hash)` on `invite` and one `FOR SELECT TO salus_invite_lookup USING (true)`
  policy, nothing else. `EXECUTE` is revoked from `PUBLIC` and granted to `salus_app`. The service then sets
  the clinic and reads the invite through RLS. `account` is not clinic-scoped (a patient has no clinic,
  and login must find the account by e-mail first); its `organization_id` is checked in the service.
- **Append-only audit.** `audit_event` has `BEFORE UPDATE OR DELETE` (row) and `BEFORE TRUNCATE` (statement)
  triggers that raise, and `salus_app` is granted `INSERT, SELECT` only. Triggers fire for every role,
  including the owner. Retention will remove old audit rows by dropping whole time ranges as the owner (see
  section 5); the application can never remove one.
- **AuditEvent shape.** FHIR R4 `AuditEvent` elements, stored as columns: `recorded`, `type` and `subtype`
  (`http://dicom.nema.org/resources/ontology/DCM` 110114 User Authentication with 110122 Login / 110123
  Logout; `http://terminology.hl7.org/CodeSystem/audit-event-type` `rest` with
  `http://hl7.org/fhir/restful-interaction` read, create, update, search-type), `action` (C, R, U, D, E),
  `outcome` (0, 4, 8, 12), the agent (`Account/<id>`, or none for a failed login with an unknown e-mail),
  the entity (`<ResourceType>/<id>`), `purposeOfEvent` (`http://terminology.hl7.org/CodeSystem/v3-ActReason`
  `HOPERAT` for operations, `TREAT` later for care), the clinic when there is one, and the request id. No
  e-mail, name, IP address or user agent. No FHIR read endpoint for `AuditEvent` in this feature.
- **What "every access to an account" means.** Registration, each login attempt (success or failure), MFA
  enrolment and verification, logout, reading one's own account (`GET /auth/me`), invite creation and
  acceptance each write one `AuditEvent` in the same transaction as the access. The per-request session check
  that authenticates a call is not audited on its own: the login that created the session is, and so is the
  domain access the request makes.
- **Retention hook.** `internal/retention` holds the rules as data (`rules.go`: one `Rule{Table, Basis,
  Keep, Then}` per table). Every domain-owned table in `ownership.yaml` gains `personal_data: true|false`.
  archcheck fails when the field is missing, when a `personal_data: true` table has no rule, or when a rule
  names a table that does not exist. Applying the rules (the scheduled job) is the next privacy feature; see
  section 5.
- **Passwords.** `golang.org/x/crypto/argon2` (the standard library has no argon2), salt 16 bytes from
  `crypto/rand`, key 32 bytes, stored as a PHC string (`$argon2id$v=19$m=…,t=…,p=…$salt$hash`) so the
  parameters can change; a hash with older parameters is re-hashed on the next successful login. Floor: the
  OWASP minimum (m = 19 MiB, t = 2, p = 1). Target: the cheapest setting at or above the floor that takes
  50–100 ms on the CI runner, measured by `BenchmarkHash`; concurrent hashes are bounded by a semaphore sized
  from the same benchmark so a burst of logins cannot exhaust memory. Length 12 to 128 characters, no
  composition rules (NIST SP 800-63B). Login compares in constant time and hashes a dummy password for an
  unknown e-mail so timing does not reveal which e-mails exist.
- **E-mail as login.** Trimmed and lower-cased before storing; unique. Registration with an e-mail that
  exists returns 409. This reveals that the e-mail has an account; the login rate limit bounds probing, and
  hiding it needs e-mail verification (section 5).
- **Session tokens.** 32 random bytes, base64url, sent to the client once; Postgres keeps only the SHA-256
  of the token. Table `session`: account, organization (staff), `mfa_verified`, `created_at`,
  `expires_at`, `revoked_at`. Lifetimes are absolute, no idle timeout (it would need a write per request):
  12 h for staff, 30 days for patients. Logout sets `revoked_at`. Each request reads the session row by
  token hash (`revoked_at IS NULL AND expires_at > now`); `revoked_at` in Postgres is the only revocation,
  and it takes effect on the next request. Redis is used only for the login rate limit and the MFA
  challenge.
- **Transport.** The web app gets the token in a cookie `salus_session` (`HttpOnly; Secure; SameSite=Strict;
  Path=/`); every state-changing request from it must also send `X-Salus-CSRF: 1`, which a cross-site form
  cannot set. The mobile app gets it in the response body and sends `Authorization: Bearer`. One session
  table serves both.
- **Login with MFA.** `POST /auth/login` checks the password. Without MFA (a patient who has not enrolled) it
  returns the session. Otherwise it returns a challenge: 32 random bytes, kept in Redis 5 minutes as
  `mfa:challenge:<sha256>` → account id, with `next: verify` or, for staff not yet enrolled,
  `next: enrol`. `POST /auth/mfa/enrol` (with the challenge, or with a patient's session) returns the
  secret and the `otpauth://` URI; `POST /auth/mfa/confirm` with the first code marks enrolment and, from a
  challenge, opens the session. `POST /auth/mfa/verify` with a code opens the session. Five wrong codes delete
  the challenge.
- **TOTP.** RFC 6238 with the standard library (`crypto/hmac`, SHA-1, 6 digits, 30 s step, one step of
  drift either way): about 30 lines, no dependency. `account.totp_last_step` rejects a code already used.
  The secret (20 random bytes) is stored encrypted with AES-256-GCM through a `SecretBox` port in `auth`.
  The key comes from the KMS in production; locally and in CI from `TOTP_KEY` (synthetic, base64, in
  `configs/.env.example`). **Needs a KMS decision before production.**
- **Login rate limit.** Redis fixed windows of 15 minutes, keyed by an HMAC-SHA256 (key `RATE_LIMIT_KEY`) of
  the normalised e-mail and, separately, of the client IP: 5 failed attempts per e-mail, 30 attempts per IP.
  Over the limit the API answers 429 with `Retry-After` and does not check the password. The IP is never
  stored elsewhere or logged. The client IP is the connection's remote address; a proxy header is trusted
  only when `TRUSTED_PROXY_CIDRS` lists the sender.
- **Invites.** An admin invites a practitioner (e-mail, given and family name) or another admin (e-mail).
  `invite` rows are clinic-scoped and expire after 72 h. Creating one enqueues `invite_email` with `InsertTx`
  in the same transaction; its arguments are the invite id and organization id. The job issues the token
  (32 random bytes, only its SHA-256 stored, replacing any earlier one), so no raw token is ever at rest, and
  sends the e-mail with the link `<WEB_BASE_URL>/invite#<token>` (fragment, so it never reaches a server
  log). A retried job issues a new token and the old link stops working. Accepting sets the password,
  enrols TOTP, creates the account and, for a practitioner, the `Practitioner` and its `PractitionerRole`
  through their services, all in one transaction. An open invite for the same e-mail in the same clinic is
  replaced, not duplicated.
- **Retried invite e-mails.** A retry after a send that did succeed (for example, the job is marked failed
  after the SMTP server accepted the message) issues a new token, so the link in the earlier e-mail stops
  working and only the newest one is valid. Accepted for now; recorded in `PROGRESS.md`.
- **Notifier.** `cmd/notifier` runs a River client that works only the `email` queue (ADR 0007: e-mail is sent
  by the notifier); `cmd/worker` keeps the default queue. Both clients are built in `internal/worker`. The
  `notification` domain owns the `Sender` port with one implementation now, SMTP (Mailpit locally). SES is
  not wired in this feature: it needs its DPA first. E-mails are plain text, carry no clinical content, and
  their body is never logged.
- **Operator commands.** `cmd/salusctl` is an operator command (a different lifecycle from the servers,
  CLAUDE.md "Layout"), run with `make salusctl args="…"` against the compose stack, connected as the owner.
  `create-clinic -name <name> -admin-email <e-mail>` creates the `Organization` and an admin invite through
  the same services; there is no HTTP endpoint that creates a clinic. `reset-mfa <account-id>` recovers a
  staff member who lost their device: it clears the TOTP secret, `totp_last_step` and `mfa_enrolled_at`,
  revokes the account's open sessions, and writes an `AuditEvent` (agent: the operator command, entity
  `Account/<id>`, action U, purpose `HOPERAT`), all in one transaction; the next login asks for enrolment.
  The id is opaque, so no e-mail appears in shell history. Self-service recovery is out of scope.
  `term-reviewer grant|revoke <account-id>` (SAL-14) gives a staff account the reviewer permission of the shared term
  list, or takes it away (`term_reviewer`, which `salus_app` only reads), audited like `reset-mfa`; a patient is
  refused (docs/terminology/SPEC.md §3 "Review").
- **Directory model.** `Organization`: `name`, `active`. `Practitioner`: `name` (one `HumanName` with
  `given` and `family`), `active`. `PractitionerRole`: `practitioner`, `organization`, `active`; no `code`
  or `specialty` yet. A practitioner has one role, in their clinic. An admin is an account, not a
  `Practitioner`. `account.practitioner_id` links a practitioner's account to the resource; the directory
  knows nothing about accounts.
- **FHIR interactions.** `Organization`: read, and update with `If-Match` (the admin's clinic settings
  screen). `Practitioner`: read. `PractitionerRole`: read and search by `organization` with
  `_include=PractitionerRole:practitioner`. R4 has no `organization` search parameter on `Practitioner`, so
  "practitioners of a clinic" is the standard `PractitionerRole?organization=…` search rather than an invented
  parameter. Only an admin or practitioner of that clinic may read; anyone else gets 404 (the resource's
  existence is not disclosed). Patients read the directory from the scheduling feature on. Every read and
  search of `Practitioner` or `PractitionerRole` and every write is audited.
- **JSON API.** Under `api/openapi.yaml`: `POST /auth/register`, `/auth/login`, `/auth/mfa/enrol`,
  `/auth/mfa/confirm`, `/auth/mfa/verify`, `/auth/logout`, `GET /auth/me`, `POST /invites`, `GET /invites`
  (open invites of the admin's clinic), `POST /invites/accept`. Errors use one JSON shape `{"error": "<code>"}`
  with a fixed code per case; never internal text.
- **Web.** The existing visual direction: light page `#F6F7F5`, white surfaces, ink `#0E1512`, brand
  `#0A6E4F`, mint `#34D399` for accents, flat backgrounds, Helvetica-like system stack
  (`'Helvetica Neue', Helvetica, Arial, sans-serif`), a few hand-drawn inline SVG icons, only what the task
  needs on screen. No UI library or router library: four screens (sign in with MFA, accept invite, clinic
  settings, practitioners and invites) switch on `location.pathname` in about 30 lines. One dependency is
  justified: `qrcode-generator` (no dependencies, pinned) to draw the `otpauth://` QR code, because a QR
  encoder is several hundred lines; the secret is also shown as text for manual entry.
- **Mobile.** Patient registration and login (including the TOTP step for a patient who enrolled elsewhere)
  through the generated Dart client. The token is kept with `flutter_secure_storage` (Keychain / Keystore),
  pinned; never in shared preferences.

Changed on approval (2026-11-04, owner):

- `salus_app` is a separate `LOGIN` role with its own credentials instead of `SET ROLE` from the owner's
  connection; only `migrate` and `salusctl` connect as the owner.
- The Redis revocation key is dropped: session revocation is `session.revoked_at` in Postgres only. This
  replaces "their revocation held in Redis with a TTL" in the settled decisions above.
- `salusctl reset-mfa` is added (package 5).
- The `SECURITY DEFINER` lookup sets `search_path` and is owned by a non-superuser role.

## 3. Lawful basis

- Accounts, sessions and MFA: performance of the contract the user signs up for (Art. 6(1)(b)).
- Audit events and login rate limiting: security of processing and accountability (Art. 5(2), 32), a legal
  obligation and legitimate interest (Art. 6(1)(c), (f)).
- Practitioners' names and invites: the clinic's employment relationship and the contract with the clinic
  (Art. 6(1)(b), (f)).
- No health data is processed in this feature. A patient account on a telemedicine service can still hint at
  a health matter, so the DPIA treats patient accounts as sensitive and no patient e-mail leaves the system
  except to the patient.
- No consent is collected here: nothing in this feature rests on consent. `Consent` arrives with the first
  feature that processes health data.

## 4. Personal data

| Where | Field | Why it is needed |
| --- | --- | --- |
| `account` | `email` | The login identifier, and where account and invite e-mails go |
| `account` | `password_hash` | To check the password; argon2id output only, the password is never stored |
| `account` | `totp_secret` (encrypted) | To check TOTP codes |
| `account` | `totp_last_step` | To reject a TOTP code that was already used |
| `account` | `role`, `organization_id`, `practitioner_id` | To decide what the account may do and in which clinic |
| `account` | `mfa_enrolled_at` | To know whether MFA is set up (required for staff) |
| `session` | `account_id`, `organization_id`, `token_hash`, `mfa_verified`, `created_at`, `expires_at`, `revoked_at` | To keep a user signed in and to end the session. No IP address, no user agent, no device data |
| Redis | `mfa:challenge:<hash>` → account id | The step between password and code; TTL 5 min |
| Redis | `ratelimit:login:email:<hmac>`, `ratelimit:login:ip:<hmac>` (counters) | Brute-force protection; keyed by HMAC so the e-mail and IP are not readable in Redis; TTL 15 min |
| `audit_event` | `agent_account_id`, `entity_reference`, `organization_id`, `recorded`, `action`, `outcome`, codes, `request_id` | Accountability: who accessed which account or directory record, when and with what result |
| `practitioner` | `given_name`, `family_name` | To identify the practitioner to colleagues now and to patients when booking arrives |
| `practitioner` | `organization_id`, `active` | Tenancy; whether the practitioner currently works there |
| `practitioner_role` | `practitioner_id`, `organization_id`, `active` | The practitioner's membership of the clinic |
| `invite` | `email` | Where to send the invite, and the login the new account gets |
| `invite` | `given_name`, `family_name` | To create the `Practitioner` on acceptance; empty for an admin invite |
| `invite` | `role`, `invited_by`, `token_hash`, `expires_at`, `accepted_at` | Who invited whom, as what; to accept the link once, before it expires |
| E-mail (Mailpit locally) | Recipient address, clinic name, link | To deliver the invite; no other content |

`organization.name` is not personal data. Logs carry request id, route, status, duration, opaque ids and
error codes only (bootstrap rules); no e-mail, name, token, code, IP or user agent. Test data is synthetic
(`@example.test` addresses, invented names).

Retention rules. These are defaults: **needs a legal decision before production.**

| Table | Rule |
| --- | --- |
| `account` | Kept while the account exists; account closure and erasure come with the subject-rights feature |
| `session` | Deleted 30 days after `expires_at` or `revoked_at` |
| `invite` | Deleted 30 days after `accepted_at` or `expires_at` |
| `audit_event` | Kept 10 years, then removed by range |
| `practitioner`, `practitioner_role` | Kept while active; restricted, not deleted, after leaving, while clinical records reference them |

## 5. Out of scope

E-mail verification, password reset and change, self-service MFA recovery (recovery codes, reset by an admin
in the web app), account closure and erasure (subject-rights feature), the scheduled job that applies the
retention rules (see "Up next"),
deactivating a practitioner, `Patient` resources and patient access to the directory (scheduling feature),
`Consent`, SES wiring (needs the DPA), the production KMS (needs the hosting choice), SMS and push, an
`AuditEvent` read API, admin UI in the mobile app, practitioner use of the mobile app.

## 6. Up next

- **Retention job.** A `scheduler` job that applies every rule in `internal/retention`: deletes expired
  sessions and invites, restricts practitioners who left, and removes audit events past their period by
  range as the owner. Its own feature, right after this one, before any production data.

## 7. Done when

All nine packages are committed on `feat/accounts`; `make lint`, `make test`, `make archcheck`, `make cover`
(every package at 80% or more), `make fhir-validate` and `make e2e` pass locally and in CI; `make gen-clients`
leaves no diff; the web and mobile checks pass; `docs/privacy/` describes what exists; `PROGRESS.md` records
every package with its readings.
