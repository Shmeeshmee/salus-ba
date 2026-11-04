# Personal data breach procedure

What we do when personal data is lost, disclosed, altered or made unavailable without authorisation
(GDPR Art. 4(12), 33, 34). Health data is special-category data, so assume a breach is likely to be a risk
to people until the assessment says otherwise.

Contacts are placeholders until they are named:

| Role | Who |
| --- | --- |
| Incident lead | [INCIDENT LEAD NAME, PHONE] |
| Data protection officer | [DPO NAME, EMAIL] |
| Controller representative (each clinic's contact for its own data) | [CLINIC CONTACT, kept per clinic] |
| Supervisory authority | [LEAD SUPERVISORY AUTHORITY, notification portal URL] |

## 1. Detect and record (hour 0)

Sources: an alert, an `AuditEvent` pattern that should not occur (one account reading many clinics' records,
bursts of failed logins beyond the rate limit), a report from a clinic, patient, processor or researcher.

- Whoever notices it tells the incident lead at once. The 72-hour clock starts when we become **aware**:
  when we have a reasonable degree of certainty that a breach happened, not when the investigation ends.
- Open an entry in the breach register (section 6) with the time of awareness.
- Contain: revoke sessions (`session.revoked_at`), rotate credentials and keys, disable the affected route
  or account, preserve logs and audit events. Never delete evidence; `audit_event` is append-only.

## 2. Assess (within 24 hours)

Record for the register:

- What happened, when, and how it was found.
- Which data: categories (account e-mail, password hash, TOTP secret, practitioner names, audit events,
  health data in later features) and the approximate number of people and records.
- Whose data, and which clinics (controllers) are affected.
- Likely consequences for the people: discrimination, identity fraud, disclosure of a health matter
  (even a patient account on a telemedicine service can hint at one), loss of access to care.
- Whether the data was encrypted with keys that were not compromised (this can remove the need to inform
  people, Art. 34(3)(a)).

Classify: no risk (record only), risk (notify the authority), high risk (notify the authority and inform
the people).

## 3. Notify the supervisory authority (within 72 hours of awareness)

Unless the breach is unlikely to result in a risk to people (Art. 33(1)). Where Salus acts as a processor
for a clinic, we notify that clinic without undue delay and it notifies its authority (Art. 33(2)); we give it
everything below.

The notification holds (Art. 33(3)): the nature of the breach, categories and approximate numbers of people
and records; the DPO's contact; the likely consequences; the measures taken or proposed. If not everything
is known by hour 72, notify with what is known and add the rest in phases, saying why it is late if it is.

## 4. Inform the people affected (without undue delay, when the risk is high)

In plain language (Art. 34(2)): what happened, the likely consequences, what we have done, what they can do
(for example, change a reused password), and whom to contact. By e-mail to the account address, or a public
notice when contacting each person would need disproportionate effort. The message carries no more personal
or clinical data than the person already has.

## 5. Follow up

Fix the cause, add a test that reproduces it where it was a code fault, and review this document and the
DPIA (`dpia.md`).

## 6. Breach register

Every breach is recorded, notified or not (Art. 33(5)): facts, effects, remedial action, the classification
and its reasoning, and the times of awareness and of each notification. The register lives
[REGISTER LOCATION, access-restricted] and holds no more personal data than the assessment needs.
