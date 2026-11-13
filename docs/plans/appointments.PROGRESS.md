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

## SAL-33: reminders (branch `appt-33-reminders`, migration 00055 as a placeholder)

Built: migration 00055 with `appointment_reminder` (owned by `appointment`; row-level security for the patient's own
and the clinic's scope; a reminder leaves `scheduled` once, by trigger, for every role; the view
`appointment_reminder_scheduled` of ids and times for the sweep), `notification_delivery`, `notification_preference`
and `sms_usage` (owned by `notification`; a delivery leaves `pending` once, by trigger), the foreign key of
`appointment_action_token.reminder_id`, ownership and retention (reminders and deliveries 90 days, the settings with
the account). In `appointment` (`service_reminder.go`): `scheduleReminders` (one row and one `appointment_reminder`
job per offset of the clinic's settings still ahead, unique by the reminder, in the booking's transaction),
`cancelReminders` (every cancellation), `TakeReminder`, `ReminderDispatched`, `ReminderNotice`, `SweepReminders` and
the reminder purger. In `notification`: `Service.Deliver` (the plan's contract, in the caller's transaction:
channels, quiet hours, the delivery rows and their jobs), `DispatchReminder`, `SendDelivery` (the `reminder_email`,
`reminder_push`, `reminder_sms` jobs on the `email`, `push` and new `sms` queues), the patient's settings
(`GET`, `PUT /me/notification-settings`) and the admin's SMS usage (`GET /clinic/sms-usage`), the delivery purger.
`patient.NoticePhone` (audited, job as agent). `APPOINTMENT_LINK_URL` in `internal/config`, the notifier and both
composes; the notifier loads `config.LoadSMS` (the fake). `appointment_reminders_sweep` hourly from the scheduler,
worked by the worker. The legacy `appointment_reminder_email` is no longer enqueued; its worker completes old jobs
without sending. Web: `ClinicSettingsReminders.tsx` (offsets, the SMS option with `[PRICE]`, 12 months of usage) in the
clinic settings; the Vite proxy and the Caddyfile route `/clinic/` to the api (`/clinic` itself stays the screen).
Mobile: `notification_settings_screen.dart` from the home screen. e2e: `TestReminders_TheReminderEmailArrivesAtItsTimeWithAOneTimeLink`.

Tests named in the plan: `TestReminder_SentOnceAtTheRightInstant` (`internal/worker`): a patient in Asia/Dubai of a
clinic in Europe/Sarajevo across the 25 October 2026 change, quiet hours in either zone, a dispatch rolled back once and
every job run twice, a cancellation and a reschedule between scheduling and dispatch; against Postgres with the
real repositories and River's inserts, fake senders.

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| `Deliver`'s `content` | `notification.Content{Deadline}`: the instant a held push or SMS must still be 30 minutes before (a reminder's start, an offer's expiry). The text is composed by the send job from the delivery's kind, so a send re-reads what still holds; SAL-35 adds the `waitlist_offer` composer in `compose` |
| Sends outside the transaction | A send locks and checks its delivery, composes (issuing a new one-time link for an e-mail or SMS) and commits; then sends; then marks it sent (an SMS counted in its clinic's month in that transaction). A retry after a failed send issues a new link; the older unused one expires at the start |
| Sweep's cross-clinic read | A view with its owner's rights (`appointment_reminder_scheduled`, ids and times only) instead of a `SECURITY DEFINER` function: sqlc cannot read a function's `RETURNS TABLE`. The sweep takes reminders due and not started, in pages of 1 000 |
| Patient's zone from the app | The app sends `DateTime.now().timeZoneName` only when it is an IANA name; most phones report an abbreviation, so the zone is rarely sent and the clinic's is used. A real zone needs the `flutter_timezone` package: the owner's call |
| SMS text | Latin-script languages without diacritics (GSM-7: one 160-character part with a short clinic name); Cyrillic and Arabic shortened to fit two UCS-2 parts. The owner's call (plan's "Copy waiting") |
| `repository_sms.go` | Not created: archcheck lets only `repository_postgres.go` import the sqlc package, so the SMS usage's queries are there, beside the deliveries' |
| `NewNotifier` | Takes a `worker.Notifier` struct (sender, pusher, SMS, the three URLs) instead of positional arguments |
| SMS usage route | 403 for anyone but the clinic's admin, patients included |
