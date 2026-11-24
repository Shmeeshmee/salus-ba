# Clinic system: wave 5, data subject rights and the desk fixes

Part of `docs/clinic-cms/IMPLEMENTATION_PLAN.md`: its "How this plan runs", "Rules for every package", "Check sets"
and "Definition of done" apply unchanged. Planned 2026-11-24, after wave 4 (PR #120). CLAUDE.md "Privacy" names the
feature ("Subject rights are features, built and tested like any other", tracked in `datarequest` with their legal
deadline) and the retention rules say account closure and erasure "come with the subject-rights feature": this wave
is that feature. Nothing of it exists today: there is no `internal/datarequest`, and `account`, `patient` and
`consent` are `ThenKeep` in `internal/retention/rules.go`.

Run: first `wave docs/clinic-cms 5.0 FX --max 2` (the contract and the desk fixes share no file), then, with both
merged into `feat/cms`, `wave docs/clinic-cms D1 D2 D3 D4 D5 D6 --max 3`. Integration order D1, D2, D3, D4, D5, D6.

## Decisions applied (agent's defaults, 2026-11-24; the owner confirms, and the lawyer before production)

| # | Point | Default |
| --- | --- | --- |
| S1 | Who answers a request | A request names its target: the patient's Salus account (Salus answers, through the operator) or the patient's record at one clinic (that clinic's admin answers; the clinic is the controller of its record, Salus its processor) |
| S2 | Kinds | access, portability, rectification, restriction, objection, erasure. Access and portability are served by the same export |
| S3 | Deadline | One month from receipt, extendable once by two months with a reason told to the patient (GDPR Art. 12(3)); the request shows its due date; the answerer is warned 7 days and 1 day before |
| S4 | A patient's own export | Everything the patient can already read in the app, from every clinic, plus their account data, as FHIR Bundles in one encrypted archive, downloadable in the app for 7 days. Built by a job with every read audited as the patient's request |
| S5 | Closing the account | At once: sign-in ends, sessions and device tokens are removed, future visits are cancelled with their notices, waitlist entries withdrawn, consents revoked. Not reversible by the patient; the operator may reopen within 30 days |
| S6 | Erasure | 30 days after closure a job erases what has no legal basis to stay (the account's e-mail, password and MFA, language choices, device tokens, notification logs, unsubmitted questionnaire answers, chat uploads never attached) and records what it erased and what it kept. The clinical record stays with each clinic for the medical-record retention period (placeholder, the lawyer's L1 to L3), marked restricted |
| S7 | Restricted record | Hidden from the clinic's everyday screens and searches; the clinic's admin may open it with a stated reason, audited; never used for analytics, reminders or messages |
| S8 | Rectification | A patient corrects their own profile themselves. For a clinical entry or a signed document they ask the clinic: the clinic answers and may add an amendment that refers to the original; a signed document is never changed |
| S9 | Objection | To the use of their data in the anonymised analytics: from the request on, the patient's events are left out of new aggregates. Existing k-anonymous aggregates are not recomputed |
| S10 | Patients the clinic registered | The clinic files the request on the patient's behalf at the desk (target: the record at that clinic), after checking identity; the same handling follows |
| S11 | Withdrawn consent and the record (walk finding 9) | The hand-over and the export include the visits cancelled by a withdrawal: the builder reads them with the job's own scope, audited, because the data is the patient's |

## Packages

Migration numbers (main's next free is `00071`): 5.0 `00071`; FX `00072`, D1 `00073`, D2 `00074`, D3 `00075`, D4
`00076`, D5 `00077`, each only if needed.

| Package | Depends on | Check sets | Its own files | Touches, appending its own block |
| --- | --- | --- | --- | --- |
| 5.0 | wave 4 on main | go, web, mobile | the migration, `internal/datarequest/*` skeleton, the OpenAPI of every route below, generated clients | `ownership.yaml`, retention rules, DPIA, `archcheck/config.yaml` (the domain in `privacy`) |
| FX | wave 4 on main | go, web, e2e | web `DeskPatients.tsx`, `ClinicSettingsNav.tsx`; `internal/clinicexport` record builder (S11) | web `Desk.tsx`, `DeskPatient.tsx`, `DeskMove.tsx`, `ClinicSettings.tsx` |
| D1 | 5.0 | go | `internal/datarequest/service.go`, `service_deadline.go`, handlers for the patient's and the clinic's request routes | `internal/worker`, `internal/api/app.go` |
| D2 | 5.0 | go | `internal/datarequest/service_export.go`, its job | `internal/clinicexport` (the builder gains the patient's own scope), `internal/worker` |
| D3 | 5.0 | go | `internal/datarequest/service_closure.go`, `service_erasure.go`, their jobs; `internal/auth/service_close.go` | `auth`, `appointment`, `consent`, `retention` core files |
| D4 | 5.0 | go, web | `internal/access/restricted.go`; web `DataRequests.tsx`, `DataRequestDetail.tsx` | every staff read's scope (restricted records), `clinicstats` (S9), web `App.tsx` (the admin's route) |
| D5 | 5.0 | go | `cmd/salusctl` datarequest commands; `internal/notification` templates for the deadline warnings | `internal/worker` |
| D6 | 5.0 | mobile | mobile `privacy_screen.dart`, `data_request_screen.dart`, `close_account_screen.dart` | mobile home (the entry), the ARB catalogs |

### 5.0: the contract

As step 0 and 3.0: one migration, the `datarequest` domain in the `privacy` context with its model, errors,
repository and refusing service methods, the OpenAPI of every route with `x-salus-package`, regenerated clients, a
fresh subagent's review before the commit. Add PROGRESS.md headings and table rows for 5.0, FX and D1 to D6.

- `data_request`: id, the patient (account or clinic-registered profile), target (`account` or a clinic), kind,
  status (`received`, `in_progress`, `fulfilled`, `refused`, `withdrawn`), received_at, due_at, extended_until and
  its reason, filed_by (the patient, or the staff account for S10), answered_at, the answer's code (no free text
  about health), version. Row-level security: the patient sees their own; a clinic's admin sees those targeting
  the clinic; the operator sees those targeting the account. Append-only history of status changes.
- `account`: `closed_at`, `erase_after`; `patient` at a clinic: a restriction (`restricted_at`, per clinic);
  `analytics objection` per patient.
- Routes: the patient's `/me/data-requests` (list, file, withdraw), `/me/data-requests/{id}/export` (status and
  download), `/me/account/close`; the clinic's `/clinic/data-requests` (list, read, answer, extend),
  `/clinic/patients/{id}/data-requests` (file on behalf, S10), `/clinic/patients/{id}/restriction`; nothing for
  the operator (salusctl).

### FX: desk fixes from the wave 4 walk

- A "Patients" search on the desk for the admin and the receptionist (name and birth date) that opens the patient's
  card whether or not a consent is active, so a patient who withdrew can still be handed their record.
- The record includes visits cancelled by a withdrawal (S11); the export's manifest and the screen hints follow.
- The card shows the phone grouped as people write it; the move dialog groups its times by day; the clinic settings
  page gets an in-page navigation so each section is one click away.
- Walk the changed screens at 1280 and 390 px and add a "Rerun" section to `docs/clinic-cms/walk/wave4/WALK.md`.

### D1: requests and deadlines

A patient files, lists and withdraws requests; a clinic's admin lists, reads, extends once and answers those
targeting the clinic; staff file on behalf of a clinic-registered patient (S10). Due dates per S3, computed in the
answerer's time zone. A scheduled job queues the 7-day and 1-day warnings (their sending is D5's). Every request,
answer and read is audited. A request is never deleted; a withdrawn one keeps its history.

### D2: the patient's export

Fulfils access and portability for the account target and for a clinic target (S4): a job builds the archive
(reusing the record builder, extended to the patient's own scope and clinical content they can read in the app),
stores it encrypted, marks the request fulfilled, and the download expires after 7 days and is audited. One export
in progress per patient. A clinic-registered patient's export is the hand-over the admin downloads (wave 4).

### D3: closing the account and erasure

Closing per S5 in one transaction with its notices; erasure per S6 as a retention applier driven by `erase_after`,
writing its record on the erasure request (what was erased, what was kept and under which rule). A patient with no
clinical record anywhere is erased whole. The operator may reopen within 30 days (salusctl is D5's: expose the
service method). Tests: after erasure no query as any role returns the erased fields; the retained record is
restricted; a second run erases nothing more.

### D4: the clinic's handling and restricted records

The admin's queue of requests with due dates and what each needs; the detail with its actions (answer with a code,
extend with a reason, open the hand-over, restrict or lift a restriction). `access` hides a restricted patient's
record at that clinic from practitioners, the desk, search, reminders and chat (S7); the admin opens it only with a
reason, audited. An objection leaves the patient out of new analytics aggregates (S9), with an equality test.

### D5: the operator and the warnings

`salusctl datarequest list | show | answer | extend | reopen-account`, connected as the owner, for requests
targeting the account. The deadline warnings by e-mail to the clinic's admin or the operator, in the recipient's
language, naming the request's kind and due date and never the patient.

### D6: the patient app

A "Privacy" entry: my requests with their state and due date; file a request (kind, target: my account or one of my
clinics, with a plain explanation of each kind); download my export when ready; close my account, with what it does
and what stays (S5, S6), behind a confirmation that asks for the password. All ten languages; the explanations and
the closure text are tagged legal.
