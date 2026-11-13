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

## SAL-34: one-tap confirm or cancel, self-service rescheduling (branch `appt-34-confirm-reschedule`)

No migration. Built: in `appointment`, `Confirm`, `Actions` (`service_confirm.go`), `Reschedule` (`service_reschedule.go`),
`Link` and `UseLink` (`service_link_use.go`) on a new `SelfServiceRepository` port (the lock, the confirmation, the
move, the link's lookup by hash, read and single use; Postgres in its SAL-34 block) and a `LinkLimiter` (Redis,
`repository_redis.go`); `ActionsHandler` (`POST /appointments/{id}/confirm`, `POST /appointments/{id}/reschedule`,
`GET /appointments/{id}/actions`) and `LinkHandler` (`GET`, `POST /a/{token}`: `html/template`, its catalogs in
`templates/link/<language>.json` in the ten patient languages, checked by `make i18n-check`). The link's cancel
passes `ViaLink` through `Caller.Via`, a field of the call; its audit events name the patient's account with the agent
`action_link`. The reschedule frees the old slot through the `slotFreed` seam (`waitlist_slot_freed`), calls
`cancelReminders` and `scheduleReminders`, `settle`, refreshes both days' statistics and enqueues
`appointment_rescheduled_email` (notifier, ten languages). Web: the confirmation badge in `Calendar.tsx`,
`AppointmentChanges.tsx`, `ClinicSettingsReschedule.tsx`; mobile: `appointment_actions.dart`, `reschedule_screen.dart`.
The Vite proxy and the Caddyfile route `/appointments/{id}/changes` and `/clinic/appointment-settings`;
`scripts/check-caddy-routes.sh` also fails unless Caddy's log filter deletes the URI.

Misuse tests: any number of `GET`s change nothing (`TestLink_ShowsTheClinicTheTimeAndTheActionsAndChangesNothing`); a
second `POST` is refused (`TestUseLink_ConfirmsOnceAsThePatientThroughTheLink`, and on Postgres
`TestUseToken_TwoAtOnceUseTheLinkOnce`); expired, revoked-consent, cancelled, rescheduled, unknown and another
appointment's links (`TestLink_LinksThatNoLongerWork`); a cancel without the box (`TestUseLink_CancelsWithTheTickedBox…`);
no and a foreign `Origin` (`TestLinkUse_Refusals`); the token never in the log or the audit
(`TestLinkPage_TheAccessLogNamesTheRouteNotTheToken`, `TestLink_ATokenIsNeverInAnAuditEvent`); the rate limit
(`TestLink_RateLimitedPerAddressUnderAnHMAC`, `TestRedisLinkLimiter_…`). Reschedule on Postgres: three at once of one
appointment respect the limit, racing a booking of the target slot and a cancellation keep slots in step, the
exclusion constraint's refusal audited in its own transaction (`internal/api/reschedule_test.go`). e2e:
`TestLinks_ThePatientConfirmsAndCancelsThroughTheLinkAndReschedulesInTheApp`. Nothing to measure was asked of SAL-34.

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| e2e's link | Issued through `IssueLink` as the notifier would, until SAL-33's reminder e-mail carries it; then the journey starts from that e-mail in Mailpit |
| The page's own origin | The `POST`'s `Origin` host must be the request's `Host` (no `APPOINTMENT_LINK_URL` before SAL-33); a missing or `null` `Origin` is refused |
| Consent | A reschedule needs the care consent active (`409 consent_required`, added to the OpenAPI), as a booking; a link no longer works under a consent no longer active; the in-app confirmation needs none (the revocation cancels the appointments it applies to) |
| Confirming a confirmed appointment through a link | Refused without using the link, so it can still cancel; the page (303) shows the confirmed state |
| A used link's page | Its outcome with the start it was issued for (`start_at`), whatever the appointment became since |
| Rate limit | 30 link lookups (`GET` or `POST`) per address in 10 minutes, Redis `ratelimit:link:<HMAC of the address>` (cleared by `make e2e` with the others) |
| Unknown link's language | The browser's `Accept-Language` among the ten, else Bosnian; a known link's own language otherwise |
| Join window | Opens 10 minutes before the start, as `videosession`'s: the constant is repeated in `appointment` (`joinOpensBefore`) |
| Refusal order of a reschedule | Self-service off, another role or clinic, the join window open (`not_reschedulable`); then the notice (`reschedule_window_closed`); then the limit; then its own slot (`slot_unavailable`); then the consent |
| The reschedule e-mail | A new notifier job, `appointment_rescheduled_email`, in the reschedule's transaction; it sends the time current when it runs |
| The legacy reminder until SAL-33 | The booking's 24-hour `appointment_reminder_email` stays scheduled for the old time and sends then with the current time; the reschedule schedules one for the new time. SAL-33 replaces both with its rows |
| Client address | `httpx.ClientIP` (moved from `auth`, which now uses it), for the rate limit only |
| Web history | Behind "Show changes" on the appointment's page, read (and audited) only when asked |

Copy waiting for the owner (all of it new): the link page in ten languages (`internal/appointment/templates/link/`; `legal`:
`cancel_effect`, `cancel_tick`, `cancel_closed`), the reschedule e-mail (`rescheduled.txt`, ten languages), the
mobile actions and picker (ten languages; `legal`: `rescheduleConfirmBody`), the web badge, history and reschedule
settings (five languages).
