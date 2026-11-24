# Clinic system: wave 4

Part of `docs/clinic-cms/IMPLEMENTATION_PLAN.md`: its "How this plan runs", "Rules for every package", "Check sets"
and "Definition of done" apply unchanged. This file is the packages' specification. Planned 2026-11-24, after waves
1 to 3 and the front desk walk were merged (main at `a88e4fb2`). No contract package: the seven own disjoint files
and each adds its own block to `api/openapi.yaml` (M1's route is already in the wave 3 contract).

Run: `wave docs/clinic-cms R1 R2 R3 M1 M2 M3 M4 --max 3` from a fresh `feat/cms`. Integration order R1, R2, R3, M1,
M2, M3, M4.

## Why these

R1 to R3 are needed before a real clinic goes live: `docs/clinic-cms/PROGRESS.md` "Deferred to wave 4" and the wave 3
integration's declined review findings. M1 to M4 are what the desk and a clinic switching from other software still
miss.

## Decisions applied (agent's defaults, 2026-11-24, owner to confirm; record them in PROGRESS.md)

| # | Point | Default |
| --- | --- | --- |
| V1 | A clinic not yet approved or listed | Its slots are generated like any clinic's; only the public directory hides it |
| V2 | Rights of a patient the clinic registered | Exercised through the clinic: staff correct the profile, record the withdrawal of the in-person consent, and hand over the record. Withdrawal acts as an app patient's revocation does. Erasure follows the same retention rules as an app patient's record. Needs the lawyer's word before production |
| V3 | Reminders without an account | SMS only, to the profile's phone, where the clinic's SMS option is on; `pkg/sms` stays the fake until a provider is chosen; no SMS option or no phone means no reminder, and the desk shows that |
| V4 | Staff moving a visit | Admin and reception, any free slot of the same visit type at the clinic (another practitioner allowed); it does not count toward the patient's own reschedule limit and does not confirm the visit; the patient is told by their channels |
| V5 | Imported patients | Imported as clinic-registered profiles without a consent; the desk records the consent at the first booking, as for a walk-in. Appointments are not imported |
| V6 | Invites | At most 50 open invites per clinic |

## Packages

Migration numbers (main's next free is `00067`): R2 `00067`, R3 `00068`, M1 `00069`, M2 `00070`, M3 `00071`, M4
`00072`, each only if needed. R1 should need none.

| Package | Check sets | Its own files | Touches, appending its own block |
| --- | --- | --- | --- |
| R1 | go | `internal/slot/service_generate_unlisted_test.go` and the change in `slot`, `schedule` and the scheduler's job | nothing else |
| R2 | go, web, e2e | `internal/patient/service_clinic_rights.go`, `internal/consent/service_inperson_withdraw.go`, their handlers; web `DeskPatient.tsx` | `patient`, `consent` core files; web `Desk.tsx` (opening the patient's card); `api/openapi.yaml`; DPIA |
| R3 | go | `internal/notification/service_patient_sms.go`, `internal/appointment/service_reminder_patient.go` | `notification`, `appointment` core files; `internal/worker`; retention rules |
| M1 | go, web, e2e | `internal/appointment/service_staff_move.go`; web `DeskMove.tsx` | `appointment` core files (the unserved reschedule route becomes served); web `Desk.tsx` (the action) |
| M2 | go, web | `internal/organization/service_export.go` (or where the wave 3 contract put the skeleton), its job and handler; web `ClinicExport.tsx` | `internal/worker`; web `ClinicSettings.tsx` (a link); ownership, retention, DPIA |
| M3 | go, web | `internal/patient/service_import.go` and handler; web `PatientImport.tsx` | `patient` core files; `api/openapi.yaml`; web `ClinicSettings.tsx` (a link) |
| M4 | go, web | web `Quotes.tsx`; `internal/auth/service_invite_limit.go` | web `App.tsx` (the route, admin only); `auth` core files |

### R1: free times for every clinic

The hourly slot generation covers listed clinics only, so an unapproved clinic's free times end 28 days after its
last change of hours. Generate for every active clinic with a schedule; the directory keeps hiding the unlisted.
Tests: an unlisted clinic's slots roll forward on the hourly run; a suspended clinic's are still generated (its
booking stop is elsewhere); the directory's results do not change. Measure the run over 1 000 clinics and record it.

### R2: rights of patients the clinic registered

From the desk, the admin and the receptionist open a registered patient's card: correct the name, birth date and
phone (audited, versioned); record that the patient withdrew the consent they gave in person. A withdrawal does what
an app patient's revocation does (future visits cancelled with their notices, access ended, waitlist entries
withdrawn), in one transaction, audited with the staff account. The admin downloads the patient's record as a FHIR
Bundle to hand over (audited). Erasure: find how an app patient's erasure and retention work today and apply the same
rules to a registered profile; if there is no erasure path for app patients either, stop and report rather than
invent one. A profile registered by the clinic is never shown or changed by another clinic: test it.

### R3: SMS reminders for patients without an account

Deliveries are per account today. Give a reminder a patient-addressed SMS delivery for a profile without an account:
scheduled with the visit as other reminders, sent through `pkg/sms` once, cancelled with the visit, counted in the
clinic's SMS usage, never sent without the clinic's SMS option and a phone. The phone is read through an audited
patient read with the job as agent, only for the send. `TestReminder_SentOnceAtTheRightInstant`'s cases apply to
this path too. The desk row's data says whether a reminder will go (a field on the desk visit; the screen is not
yours).

### M1: staff moving a visit

Serve `/clinic/appointments/{id}/reschedule` per V4. The history records `via staff` with the account. Concurrency
tests as the patient's reschedule. Reminders are rescheduled; for a video visit the join window follows. On the desk,
"Move" opens today's and later free times of the same visit type. An arrived, fulfilled, cancelled or no-show visit
cannot be moved.

### M2: clinic data export

What O2 left: the admin asks for an export, a job builds one encrypted archive in the object store (price list,
staff, appointments, receipts and quotes as CSV; each patient's record as a FHIR Bundle, every read audited with the
job as agent), the admin downloads it within 7 days (audited), then it is deleted. One export in progress per clinic.
The routes are in the wave 3 contract. Ownership, retention and DPIA entries for the export's table.

### M3: patient import

The admin uploads a CSV or XLSX of patients (given name, family name, birth date, phone), reusing O2's reader
(UTF-8 or Windows-1250, headers in English and bs/hr/sr), with a preview of every row's errors and likely duplicates
(same name and birth date already at the clinic) before anything is saved; all valid rows or none. Imported profiles
are clinic-registered, without a consent (V5). At most 5 000 rows per file. Every import is audited with the count,
never the rows.

### M4: quotes page and invite limit

`/quotes` for the admin: the clinic's quotes with their state (sent, expired), filter by state, open the PDF; linked
from the receipts area. At most 50 open invites per clinic (V6): the next is refused with a code the screens
translate.
