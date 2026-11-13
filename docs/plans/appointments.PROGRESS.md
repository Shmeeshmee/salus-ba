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
`smstest`; no adapter), `config.LoadSMS` (the fake only), `pkg/payments` (with `paymentstest`).
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
| SMS adapter | None (review): a generic adapter for an unknown API was speculative; the adapter comes with the chosen EU/EEA provider. `LoadSMS` refuses any engine but the fake, and the clinic's SMS option is refused (`sms_unavailable`) while it is the fake |
| A cancellation's jobs | Every cancellation that frees a slot still ahead now also enqueues `waitlist_slot_freed` (completed by a no-op until SAL-35), and every cancellation writes a `status_set` change to cancelled |
| Direct booking's idempotency key | `booking:<patient>:<slot>` until SAL-23 gives a booking a request id: a retry of the same booking is charged once |
| Revocation in FHIR | A cancellation by the revocation (reason `pat`, no account) leaves the patient participant as it was; only the patient's own cancellation is `declined` |

Copy waiting for the owner: none. Step 0 adds no user-facing string; the API returns codes only.

Independent review of PR #86 (2026-11-13), all fixed: the reschedule columns guarded by `appointment_guard_change` under
every UPDATE policy (`TestAppointment_OnlyThePatientMovesOrConfirmsTheirBookedAppointment`); `start_at` on the link
(`TestAppointmentActionToken_IssuedByTheJobUsedByThePatient`, `TestIssueLink_StoresTheHashOfAOneTimeLinkForABookedAppointment`);
a change row for every cancellation (`TestHistory_KeepsEveryCancellationAndWho`); the purchase before the booking's
transaction with a refund on failure (`TestPurchase_*`); a used link never changes (migration test); a used link's
page shows its outcome, and the cancel's ticked box against scanners that click (ADR 0015, OpenAPI); the revocation's
FHIR participant (`TestHandlerRead_ThePatientParticipantFollowsTheConfirmation`); the payments fake refuses a reused
key with other parameters, no SMS adapter; the SMS option refused with the fake engine
(`TestAppointments_RefusesTheSMSOptionWhileNoProviderSends`); `For`'s scope documented and the history's insert tied to
the scope (`TestAppointmentChange_IsAppendOnly`).
