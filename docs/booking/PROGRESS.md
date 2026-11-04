# Booking progress

| Package | Status | Commit |
| --- | --- | --- |
| 1 Retention job | to do | |
| 2 Patient | to do | |
| 3 Consent and access | to do | |
| 4 Directory additions | to do | |
| 5 Schedule and slot | to do | |
| 6 Appointment | to do | |
| 7 FHIR | to do | |
| 8 Web | to do | |
| 9 Mobile | to do | |
| 10 End-to-end | to do | |

## Decisions

- 2026-11-04, owner (changes on approval, recorded in `SPEC.md` §2 "Changed on approval" and §3):
  - Revocation enqueues `consent_revoked` (patient id, clinic id) in its transaction; the worker cancels the
    future appointments and sends the e-mails. Access ends at the revocation's commit. Dependency
    appointment → consent only; no function value in `app.go`.
  - Lawful basis: care records under Art. 9(2)(h), consent for pre-care sharing and new bookings. After a
    revocation the clinic keeps minimal read access to past appointments (name, times, status) and loses the
    profile and anything future. **Needs legal confirmation before production.**
  - `salus_retention` has its own credentials, `RETENTION_DATABASE_URL`, loaded only for the retention job,
    with tests that the api and the other jobs cannot use them.
  - Slot status changes in the appointment's transaction; a concurrent book-and-cancel test proves status
    and the exclusion constraint agree.
  - Specialties: general practice, internal medicine, cardiology, dermatology, gynaecology, paediatrics,
    psychiatry, neurology, ENT, ophthalmology, orthopaedics, endocrinology (c80 practice codes).
  - Everything else in `SPEC.md` approved as written.

## Waiting on people

- Before production: a legal decision on the retention of `appointment` and `consent`, and legal
  confirmation of the lawful basis split (`SPEC.md` §3, §4).
- Before production: `PATIENT_APP_URL`, the link in appointment e-mails, once the app has a link domain.
- Before production (ops): the owner role that runs `migrate` needs `CREATEROLE` for `salus_retention`, as for
  `salus_app` (accounts).
- Before 2036-11: removal of audit ranges (`SPEC.md` §6).

## Log

### 2026-11-04 — plan

`SPEC.md`, this plan and `RUN.tsv` written.

### 2026-11-04 — plan approved

Owner approved with the changes under Decisions; `SPEC.md` and the plan updated. Committed as
`booking(0): spec and plan`.
