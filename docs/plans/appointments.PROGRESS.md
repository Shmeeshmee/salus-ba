# Appointments: progress

The running record of docs/plans/appointments.md, one section per merged package, written at merge time.

## Step 0: the contract (branch `appt-0-contracts`)

Phase A: the plan and ADR 0015, with the owner's answers to Q1 to Q9 (2026-11-13). Phase B: the shared contract.

Built: migration 00054 (a placeholder) with `appointment`'s `confirmed_at`, `confirmed_via`, `reschedule_count`,
`booked_via` and the grant on `period` and `slot_id`; `appointment_change`, `appointment_action_token` (with
`appointment_action_token_find`) and `organization_appointment_settings`, each with row-level security, ownership and
retention (the link's purger on the retention role). In `appointment`: `Change` and `recordChange` on every status
set (the practitioner's and the consultation's end), `Changes` and `GET /appointments/{id}/changes`, `IssueLink`,
the FHIR patient participant `tentative` / `accepted` / `declined` (golden files), the seams `scheduleReminders`,
`cancelReminders`, `slotFreed` (`waitlist_slot_freed`, a no-op worker), `purchase` and `settle`. In `organization`:
`Appointments` (`Settings`, `SetSettings`, `For`) and `GET`, `PUT /clinic/appointment-settings`. `pkg/sms` (with
`smstest` and the generic `smshttp`, off and wired nowhere), `config.LoadSMS`, `pkg/payments` (with `paymentstest`).
The OpenAPI of every wave-1 route (`x-salus-package` names its builder) and the regenerated clients.

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| History's row-level security | `appointment_change` is seen and written exactly where its appointment is seen (an `EXISTS` on `appointment`, whose policies decide), so it follows every appointment scope without copying them; its clinic and patient are held to the appointment's by a composite foreign key |
| Append-only | The chat's `communication_reject_removal()` trigger function is reused for `UPDATE`, `DELETE` and `TRUNCATE` of `appointment_change`, for every role |
| Link token | 32 random bytes, base64url in the path; the stored hash is the SHA-256 of the raw bytes. The page and its use are SAL-34's |
| Settings without a version | As the chat settings: a `PUT` replaces the whole row, no `If-Match` (the plan's contract said so at first; changed) |
| Repository files | archcheck lets only `repository_postgres.go` import a domain's sqlc package, so the new implementations are appended there (the plan's file ownership says so now) |
| `APPOINTMENT_LINK_URL` | Not added in step 0: nothing issues a link before SAL-33, which adds it with the notifier's wiring |
| SMS adapter | `smshttp` is a generic JSON-over-HTTPS adapter with a bearer token, to be checked against (and adapted to) the chosen provider's API; `LoadSMS` takes it only with an https `SMS_HTTP_URL` and a `SMS_HTTP_TOKEN_FILE`, and no binary calls `LoadSMS` yet |
| A cancellation's jobs | Every cancellation that frees a slot still ahead now also enqueues `waitlist_slot_freed` (completed by a no-op until SAL-35) |

Copy waiting for the owner: none. Step 0 adds no user-facing string; the API returns codes only.
