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
