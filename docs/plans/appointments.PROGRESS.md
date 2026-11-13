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
worked by the worker. The legacy `appointment_reminder_email` is no longer enqueued; its worker still sends the jobs
enqueued before (an appointment booked before keeps its one 24-hour e-mail; ADR 0015 amended to match). Web: `ClinicSettingsReminders.tsx` (offsets, the SMS option with `[PRICE]`, 12 months of usage) in the
clinic settings; the Vite proxy and the Caddyfile route `/clinic/` to the api (`/clinic` itself stays the screen).
Mobile: `notification_settings_screen.dart` from the home screen. e2e: `TestReminders_TheReminderEmailArrivesAtItsTimeWithAOneTimeLink`.

Tests named in the plan: `TestReminder_SentOnceAtTheRightInstant` (`internal/worker`): a patient in Asia/Dubai of a
clinic in Europe/Sarajevo across the 25 October 2026 change, quiet hours in either zone, a dispatch rolled back once and
every job run twice, a cancellation and a reschedule between scheduling and dispatch; against Postgres with the
real repositories and River's inserts, fake senders.

### Measured (`make bench-reminders`, 2026-11-13)

The host was shared with the other wave-1 stacks (load average 10 to 25 during the runs), so the numbers are upper
bounds; the queries are those of the code, on salus_app with its row-level security.

| Measure | Result |
| --- | --- |
| The hourly sweep over 100 000 scheduled reminders, 1 000 of them overdue: find them (two pages of the partial index on `(due_at, id)` where scheduled) and enqueue their dispatch again (unique by the reminder, `InsertManyTx`) | 30.0 ms and 33.8 ms per sweep (two counts of five) |
| 1 000 reminders dispatched at once by 10 workers (the notifier's), each storing an e-mail and a push delivery with their jobs | 1.37 s for the 1 000 (mean of three runs) |
| Their 2 000 sends by 10 workers on the fake senders (lock, re-read the reminder and the appointment, issue the e-mail's one-time link, send, mark sent) | 4.6 s for the 2 000 (mean of three runs) |

- The 1 000 dispatches share one practitioner role, so their appointments cannot share a start: the benchmark runs
  them all at one instant, the work a minute with 1 000 due reminders asks. About 1.4 ms of database work per
  dispatch and 2.3 ms per send at ten at a time: a burst of 1 000 is sent within seconds, far inside the reminders'
  lead of an hour or a day.
- The cost is in the sends' audited reads (the appointment, the patient's account and settings, each audited in the
  same transaction). Not tuned: nothing here is near a limit.
## SAL-35: the waitlist (branch `appt-35-waitlist`)

Built: migration 00056 (a placeholder) with `slot.status` `busy-tentative`, `waitlist_entry` and `waitlist_offer`
(row-level security, ownership, retention rules and purgers, migration tests). `slot`: `ForWaitlist` (the slot locked,
with its schedule's role, through the new `schedule.Service.Lookup`), `Offer`, `TakeOffered`, `Unoffer`. `appointment`:
`Join`, `MyWaitlist`, `ClinicWaitlist`, `Withdraw`, `SetPriority`, `WithdrawForRevokedConsent` (called by the
`consent_revoked` worker after the cancellations); `HandOn` (the real `waitlist_slot_freed` worker), `ExpireOffer`
(`waitlist_offer_expire` at the hold's end), `Accept` (through step 0's purchase step, key `waitlist:<offer>`: charged
before the transaction, refunded when the booking fails; nothing has a price, so nothing is charged), `Decline`,
`OfferNotice`; `booked` (the jobs after a booking) shared by both booking paths. The `WaitlistHandler` mounts every
SAL-35 route of `api/openapi.yaml` (their "404 until then" notes dropped; `profile_required` and `payment_declined`
added to the 409s). The offer's notice goes through SAL-33's deliveries (below, "At SAL-33's merge"). Web: `Waitlist.tsx` (the clinic's list, priority, removal) at `/waitlist` and
`ClinicSettingsWaitlist.tsx` under the clinic settings, with the proxy and Caddy route
`^/(waitlist/.+|clinic/(waitlist(/.*)?|appointment-settings))$`. Mobile: `waitlist_screen.dart` (join from a
practitioner's free times, the patient's waitlist with the offer's accept and decline), a button on the home screen.
e2e: `TestWaitlist_AFreedSlotIsOfferedAndAcceptedThroughTheApp`.

Concurrency tests on Postgres (`internal/api/waitlist_race_test.go`, real services, row-level security on), each round
followed by the invariant that a slot is `busy` exactly when a non-cancelled appointment holds it and `busy-tentative`
exactly when an open offer does, and no slot has two of either: an acceptance racing the expiry (the expiry delayed from
0 to 250 ms over eight rounds, both outcomes seen; typically 7 accepted, 1 expired first), two acceptances on two
devices (one booking, one `offer_expired`), an acceptance racing a direct booking (the direct one always
`slot_unavailable`), a cancellation racing a freed-slot job (the slot always ends offered to a waiting patient other than
the one who cancelled), and two freed-slot jobs of one slot (one offer).

Measured (compose Postgres, the tests' throwaway databases, salus_app with row-level security):

| What | Result |
| --- | --- |
| Free-slot search with a third of the searched schedule held (`TestWaitlist_TheFreeSlotSearchStaysFastWithOfferedSlots`, `seedSlotSearch`'s directory of ~22 000 slots) | cost 18 409 (main's 18 979 without held slots), 877 rows in 6 to 20 ms, both directory scopes; under a fifth of `jit_optimize_above_cost` |
| The matching query, 10 candidates of 10 010 entries in one clinic (`TestCandidates_TenThousandEntriesInAClinic`, 2 000 consenting patients) | median 3.6 to 4.0 ms over five runs (3.4 to 9.8 ms) |

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| `Deliver`'s `content` | `notification.Content{Deadline}`: the instant a held push or SMS must still be 30 minutes before (a reminder's start, an offer's expiry). The text is composed by the send job from the delivery's kind, so a send re-reads what still holds; SAL-35 adds the `waitlist_offer` composer in `compose` |
| Sends outside the transaction | A send locks and checks its delivery, composes (issuing a new one-time link for an e-mail or SMS) and commits; then sends; then marks it sent (an SMS counted in its clinic's month in that transaction). "Sent once" holds per dispatch: a crash between the provider's acceptance and the mark (the window ADR 0015 accepts) resends the e-mail, push or SMS, and an e-mail or SMS resent carries a second valid link (both single use, both expiring at the start). The first link cannot be reused: only its hash is stored |
| A send's channel | Checked at dispatch and again at the send: a send held by quiet hours whose channel the patient turned off meanwhile, or whose clinic turned SMS off, is cancelled |
| A reminder's generations | A cancellation or reschedule cancels the appointment's live reminders, the dispatched ones too (a dispatched reminder may be retired once, by the trigger), so their held sends are cancelled when they run; a live reminder (scheduled or dispatched) is unique per appointment, offset and start (a partial index), so a move from A to B and back to A schedules A's again |
| SMS text and length | The clinic's time with its zone's name, then the patient's day and time with theirs when it differs. With a 16-letter clinic name and a 71-character link: en 1 part at home, 2 abroad; bs, hr, sr-Latn, de, tr, it, fr 2 at home and abroad (GSM-7); sr-Cyrl and ar 3 at home and abroad (UCS-2). Longer clinic names add parts (at most `sms.MaxSegments`, 3) |
| Deliver's audit agent and scope | The agent is the kind's, `<kind>_delivery` (`reminder_delivery`, `waitlist_offer_delivery`); Deliver reads the clinic's SMS option in the clinic's scope and sets the caller's clinic scope back after. A kind without a composer (a waitlist offer before SAL-35) fails its delivery at the send instead of retrying |
| A Service without the delivery ports | The api builds `notification.Service` for the settings only; `Deliver`, `DispatchReminder` and `SendDelivery` return an error when their ports are missing, never a nil dereference |
| Sweep's cross-clinic read | A view with its owner's rights (`appointment_reminder_scheduled`, ids and times only) instead of a `SECURITY DEFINER` function: sqlc cannot read a function's `RETURNS TABLE`. The sweep takes reminders due and not started, in pages of 1 000 |
| Patient's zone from the app | The app sends `DateTime.now().timeZoneName` only when it is an IANA name; most phones report an abbreviation, so the zone is rarely sent and the clinic's is used. A real zone needs the `flutter_timezone` package: the owner's call |
| SMS text | Latin-script languages without diacritics (GSM-7); Cyrillic and Arabic shortened. The owner's call (plan's "Copy waiting") |
| `repository_sms.go` | Not created: archcheck lets only `repository_postgres.go` import the sqlc package, so the SMS usage's queries are there, beside the deliveries' |
| `NewNotifier` | Takes a `worker.Notifier` struct (sender, pusher, SMS, the three URLs) instead of positional arguments |
| SMS usage route | 403 for anyone but the clinic's admin, patients included |

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
`TestLinks_ThePatientConfirmsAndCancelsThroughTheLinkAndReschedulesInTheApp`, from the reminder e-mail in Mailpit. Nothing to measure was asked of SAL-34.

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| e2e's link | Taken from SAL-33's reminder e-mail in Mailpit (the clinic's offset just under the time to the start, so the reminder is due within a minute) |
| The page's own origin | The `POST`'s `Origin` host must be the request's `Host` (the api's own host, which serves `APPOINTMENT_LINK_URL`'s links); a missing or `null` `Origin` is refused |
| Consent | A reschedule needs the care consent active (`409 consent_required`, added to the OpenAPI), as a booking; a link no longer works under a consent no longer active; the in-app confirmation needs none (the revocation cancels the appointments it applies to) |
| Confirming a confirmed appointment through a link | Refused without using the link, so it can still cancel; the page (303) shows the confirmed state |
| A used link's page | Its outcome with the start it was issued for (`start_at`), whatever the appointment became since |
| Rate limit | 30 link lookups (`GET` or `POST`) per IPv4 address or IPv6 /64 in 10 minutes, Redis `ratelimit:link:<HMAC of the address>` (cleared by `make e2e` with the others) |
| Unknown link's language | The browser's `Accept-Language` among the ten, else Bosnian; a known link's own language otherwise |
| Join window | Opens 10 minutes before the start, as `videosession`'s: the constant is repeated in `appointment` (`joinOpensBefore`) |
| Refusal order of a reschedule | Self-service off, another role or clinic, the join window open (`not_reschedulable`); then the notice (`reschedule_window_closed`); then the limit; then its own slot (`slot_unavailable`); then the consent |
| The reschedule e-mail | A new notifier job, `appointment_rescheduled_email`, in the reschedule's transaction; it sends the time current when it runs |
| Client address | `httpx.ClientIP` (moved from `auth`, which now uses it), for the rate limit only |
| Referrer policy | `strict-origin`, not ADR 0015's `no-referrer` (review of #92): under `no-referrer` browsers send `Origin: null` with the form's `POST` (Fetch, "append a request Origin header"), which the `Origin` check refuses. A Referer then carries the api's origin only, never the token's path. ADR 0015 "GET never changes anything" is amended to match |
| A link's use | Reads the appointment under its row lock (`ByIDForUpdate`) and decides validity on that row, so a reschedule or cancellation committed after the page was read leaves an old-time link useless (review of #92) |
| Settings in the patient's transaction | `organization.Appointments.For` sets the clinic's scope; the reschedule and the actions drop it and set the patient's again right after (review of #92) |
| Web history | Behind "Show changes" on the appointment's page, read (and audited) only when asked |

Copy waiting for the owner (all of it new): the link page in ten languages (`internal/appointment/templates/link/`; `legal`:
`cancel_effect`, `cancel_tick`, `cancel_closed`), the reschedule e-mail (`rescheduled.txt`, ten languages), the
mobile actions and picker (ten languages; `legal`: `rescheduleConfirmBody`), the web badge, history and reschedule
settings (five languages).
| Handing a slot on | One path, `handOn`: the job locks the slot (`ForWaitlist`), leaves it when booked or held by an open offer, gives back a held slot whose offer closed, then offers it to the first entry the matching query and the consent check give, or leaves it free. Decline, withdrawal and revocation close the offer in the patient's scope and enqueue `waitlist_slot_freed`; the expiry closes it and hands on in its own transaction |
| Offer visibility | The clinic's jobs see every offer of the clinic whatever the patient's consent (they must see an open offer of a slot to leave it held); entries stay consent-gated, and an offer is inserted only for an entry the clinic sees |
| Offer's role | `waitlist_offer` gains `practitioner_role_id` (the slot's), so an acceptance books without reading the slot first |
| `expired` | Not stored: an active entry past `ends_at` (the end of its last day in the clinic's zone) shows as `expired`, matches nothing and no longer counts towards the five; retention deletes it 30 days after `ends_at` |
| Overlap check | "No other appointment of theirs overlapping" is checked in the clinic's scope, so only the same clinic's appointments count (row-level security hides the others), as a direct booking checks none; noted in the plan |
| Matching times | An entry's times are minutes after the clinic's midnight; a slot matches when it lies wholly inside them |
| Staff | Any staff of the clinic list the waitlist, move an entry and remove one; names follow `access.Profile` |
| Started slots | `Unoffer` frees a held slot even after its start (no offer can hold it, and retention deletes free slots only); the plan's contract is amended |
| Audit of the entry | An entry is audited as `Appointment/<entry id>` (FHIR's waitlist is an Appointment status); the job's read of candidates as a search without an entity |
| Retry key | The acceptance's key `waitlist:<offer>` replays the first receipt on a retry after a refund, as the direct booking's key does: SAL-23 settles both. A concurrent acceptance shares the receipt: an offer accepted already gives its appointment again, and a failed acceptance does not refund when the offer is accepted |
| Join entry point (mobile) | From a practitioner's free times, for that practitioner, over 7, 14, 30 or 90 days, any time, mornings (08:00 to 12:00) or afternoons (12:00 to 17:00); joining for a specialty is in the API only |

Checks (compose project salus-appt35): `make lint`, `make test`, `make archcheck`, `make cover` (91.3%), `make e2e`, `make i18n-check`, `make web-lint`, `make web-typecheck`, `make web-test`, `make mobile-analyze`, `make mobile-test` pass; `make gen-clients` leaves no diff.

Independent review of PR #93 (2026-11-13), all fixed test-first:

| Finding | Fix | Test |
| --- | --- | --- |
| 1. Two acceptances share the receipt; the loser's refusal refunded it | An offer accepted already gives its appointment again (audited as a read, no purchase); a failed acceptance re-reads the offer and refunds only when it is not accepted | `TestAccept_AnOfferAcceptedOnAnotherDeviceGivesItsAppointment`, `TestWaitlist_TwoAcceptancesBookAndChargeOnce` (priced: one receipt per slot, no refund) |
| 2. Accept locked offer then entry, withdraw entry then offer | Accept locks the entry first | `TestWaitlist_AnAcceptanceRacingAWithdrawalNeverDeadlocks` (deadlocked without the fix) |
| 3. The five-entry limit had no lock | `pg_advisory_xact_lock` on (patient, clinic) before the count | `TestWaitlist_ConcurrentJoinsKeepTheLimitOfFive` (six of six joined without it) |
| 4. The chosen entry was not locked | The matching query takes `FOR UPDATE OF e SKIP LOCKED`; the invariant now also checks that no open offer has a closed entry | `TestCandidates_PassOverAnEntryAnotherTransactionHolds`, `TestWaitlist_TwoSlotsFreedAtOnceOfferTheOneEntryOnce` |
| 5. The clinic's list read every entry and named each one by one | Status filtered in SQL (`active` by default), pages (`limit` 100, at most 500, `offset`, `next_offset`), the practitioner's visits read once, each patient's consent and name once, every audit event in one insert (`RecordAll`) | `TestClinicEntries_FiltersTheStatusAndPages`, `TestClinicWaitlist_GivesAPageOfTheActiveEntriesByDefault`, `TestWaitlist_APageOfTheClinicsListAtTenThousandEntries`: a page of 100 at offset 5 000 of 10 000 entries, median 53 ms as the admin (100 names), 41 ms as a practitioner (48 to 63 ms, 40 to 61 ms over five runs) |
| 6. An offer of a revoked patient in the staff's list | None was possible; now tested | `TestClinicEntries_NeverShowAnOfferOfARevokedPatient` |
| 7. Copy | The second device gets the booking (finding 1), so "You stay on the waitlist" is no longer shown after a booking; the hold's end carries the zone's name, and its date when it ends on a later day | `TestNotifyOffer_AHoldPastMidnightGivesItsDate`, golden e-mails |
| 8. The contract | The plan amends `Unoffer` (also after the start) and notes the same-clinic overlap check, the offer's `practitioner_role_id`, the list's pages and the repeated acceptance | |

At SAL-33's merge (2026-11-13): an offer's notice is `notification.Service.NotifyOffer`, which calls `Deliver(ctx,
"waitlist_offer", offerID, org, accountID, Content{Deadline: the offer's end})` in the offer's transaction, so the
patient's channels, quiet hours (a push or SMS held past the hold's end is dropped; the e-mail goes at once) and the
clinic's SMS option apply, each sent once; the `waitlist_offer_notify` job and the plain sends are gone. The send's
composer (`composeOffer`) re-reads the offer and sends nothing once it closed or its hold ended. `Ports.Quiet` is
`notification.Service.Quiet` (the patient's quiet hours in their zone, else the clinic's): an entry whose patient is
inside them is passed over. The account comes from `patient.Service.Lookup` in the clinic's scope (audited as a read
with the job as agent). The waitlist reads the clinic's settings through SAL-33's `Ports.Rules` (the same `For`), so
its own `Settings` port is gone. `TestWaitlistPorts_GiveTheOffersTheirDeliveriesAndQuietHours` (internal/api) and
`TestNewRevocations_GiveTheOffersTheirDeliveriesAndQuietHours` (internal/worker) check the wiring. New copy: the offer
SMS in ten languages (`waitlist_offer_sms.txt`, GSM-7 without diacritics where the language allows, as SAL-33's
reminder; one part for a short clinic name).

At SAL-34's merge (2026-11-13): a reschedule's and a link cancel's freed slot reaches the waitlist through the same
`waitlist_slot_freed` (`TestWaitlist_ARescheduleHandsTheSlotItLeavesToTheWaitlist`: the job's arguments name the slot
left, and the job offers it to a waiting patient, never to the one who moved off it). The acceptance-against-expiry
race now delays either side (the acceptance got faster), so both outcomes are seen every run (4 and 4).

Delta review of 7afdb7e..60b090f (2026-11-13), all fixed test-first:

| Finding | Fix | Test |
| --- | --- | --- |
| 1. Tests failed at night (the defaults' quiet hours) | The real-stack tests and the e2e journey give the waiting patients no quiet hours; one explicit night case with a fixed clock | `TestWaitlist_ANightTimeOfferWaitsForTheMorning` |
| 2. Passed-over patients in quiet hours lost the slot; the search stopped at 10 | `firstReachable` reads past passed-over entries a page at a time (100 at most); when all are passed over and some for quiet hours, `waitlist_slot_freed` is scheduled at the earliest end of those, unique by its slot (`EnqueueHandOn`). Recorded in the plan's "Waitlist matching" | `TestHandOn_TriesAgainAfterTheQuietHoursAndLooksPastTheFirstPage`, `TestWaitlist_ANightTimeOfferWaitsForTheMorning` |
| 3. A patient without a profile stopped the offer for everyone | Passed over | `TestHandOn_PassesOverWhomNoOfferReaches` |
| 4. Offer sends audited as reminders | `waitlist_offer_email`, `_push`, `_sms`; the reach read carries the job's agent | `TestSendDelivery_TheOffersSendsAreAuditedAsTheirOwn`, `TestReach_TellsTheQuietHoursAndWhetherAChannelCarriesTheOffer` |
| 5. A slot held for a patient nothing reaches | `Ports.Quiet` becomes `Ports.Reach` (`notification.Service.Reach`): reachable when e-mail is on, push on with a device, or SMS opted into at a clinic that offers it; otherwise passed over | as finding 4, `TestHandOn_PassesOverWhomNoOfferReaches` |
| 6. Compose and send race | Benign and the same as the reminders': a send that composed just before the offer closed still goes; the patient then finds the offer closed in the app | (noted) |

Copy waiting for the owner: the offer e-mail, push and SMS in ten languages
(`internal/notification/templates/*/waitlist_offer*.txt`); the mobile `waitlist*` strings in ten languages (`legal`:
`waitlistExplained`, `waitlistAcceptEffect`, `waitlistAccept`); the web `waitlist` area in five languages (after the
review: `showMore` added, `statusAll` dropped, the status filter uses `status.expired`, `.booked`, `.withdrawn`); the
offer e-mail's hold end now reads "until 09:10 (Europe/Sarajevo)", or with its date past midnight (Italian "fino alle"
before a date reads awkwardly in that rare case: the owner's call).

## SAL-36: no-shows, corrections, the repeat no-show flag and the reports (branch `appt-36-noshows`)

Built: migration 00057 (a placeholder: renumber at merge) adds `booked_via` to `clinic_appointment_stat`'s key (rows
counted before it are direct bookings). `appointment`: the correction in `POST /appointments/{id}/status` (noshow and
fulfilled into each other, within 30 days of the start, by the appointment's practitioner or the clinic's admin,
`access.CorrectsOutcome`; `status_corrected` by staff in the history; the clinic-day refresh enqueued; 409
`correction_window_closed`), `NoShowRepository.CorrectStatus` (moves only from the status it names, and only between
those two), `Service.NoShowFlags` and `GET /clinic/noshow-flags` (`handler_noshow.go`, `NoShowHandler`), the stats
source reads `booked_via`. `clinicstats`: the builds count by `booked_via`; `series[].refilled`,
`practitioners[].noshow` and `.booked`, `specialties[]` (admins only, C2). Web: the correction dialog and the flag on
the appointment's page (`NoShows.tsx`), the admin's threshold and window below the clinic settings
(`ClinicSettingsNoShows.tsx`). The `/clinic/` API prefix is proxied (Vite and Caddy).

Checked: the window's close records the outcome (practitioner joined, patient absent: `noshow` with one `status_set`
by the system; `TestConsultationWindowClose_MarksTheNoShowInTheHistory`); a practitioner who never joined leaves the
appointment booked, as before. Equality: `TestNoShowReports_EqualTheAppointmentsAndTheirHistory` (a synthetic clinic
over October and November 2026 with every status, corrections both ways, waitlist bookings and the 25 October clock
change: the series, per practitioner and per specialty equal the raw appointments whose latest history status is
noshow, and the waitlist bookings). Through the stack: `TestNew_NoShowsAreCorrectedWithin30DaysAndFlaggedForStaffOnly`.

Measured (independent review, `internal/worker` `BenchmarkNoShowFlags`, i7-14700HX, compose Postgres): the flags of 62 days of a busy
synthetic clinic (1 550 appointments of 600 patients, each with 6 past appointments, every appointment audited) took
660 to 680 ms with two queries per patient (the consent and the patient's whole history); 130 to 155 ms with the
patients' no-shows counted in one grouped query (`CountNoShowsSince`). The consent is still read once per patient
(`consent.Service.State`, the other domain's port, as the staff search does); a batch read there is the next step if
a range ever needs it.

As built (agent decisions, for the owner to review):

| Topic | Decision |
| --- | --- |
| `refilled` | Every appointment booked through the waitlist that starts in the month, whatever its status now (as `booked` counts) |
| The flag's window | The patient's `noshow` appointments at the clinic that started in the last `noshow_flag_window_days` days up to now, counted on read in the clinic's scope; flagged at `noshows >= threshold` |
| Who sees a flag | Staff, for each appointment of the range they see in full; a practitioner their own role's only; every appointment of the range seen in full is audited, flagged or not (search, `TREAT`; review), one event without an entity when none is; a patient gets 404, audited |
| A rate without closed appointments | No value, never 0 %: a gap in the monthly line (named "No closed appointments", also in the table view) and no bar per practitioner or specialty (review) |
| Saving the flag's settings | `PUT /clinic/appointment-settings` replaces the row and has no version: the section reads the settings again just before its `PUT` and changes only its two fields (review), so another section's or a concurrent admin's save is kept, but for a write landing in the milliseconds between the two |
| An admin's refusals | An admin asking to set a status on an appointment they may not correct (booked, cancelled, another clinic's, unknown) gets 403 `forbidden`, as before for every admin request |
| Superseded expectations | `internal/api` `checkMarking`'s "the practitioner again" now repeats `fulfilled` (409); a `noshow` there is a correction (Q6). Web `AppointmentDetail.test` "no actions on an appointment no longer booked" now checks a cancelled one, and a fulfilled one offering only the correction |
| Where the flag shows | On the appointment's page beside its status (`AppointmentDetail.tsx`), not yet in the week view: `Calendar.tsx` is SAL-34's (follow-up) |
| Strings | In the `calendar` catalog area under `noshow.` (a new area would edit `catalog.test.ts`, which PR #75 changes) |
| Settings section | Rendered inside the clinic settings panel once the clinic is read, with its own save button ("Save the flag"); it saves the whole appointment settings row with only its two fields changed |
| Ports | `appointment.Ports.NoShows` and `.Rules` (`ClinicRules`, `organization.Appointments.For`); the other wave packages may add a settings port of the same shape: merge them into one at merge time |

Charts (PR #75 reached main during this package, merged in): My clinic gains the no-show rate per month, the slots
refilled from the waitlist, and for admins the no-show rate per practitioner and per specialty
(`web/src/analytics/NoShowCharts.tsx`). The rate is no-shows over fulfilled plus no-shows (booked ones are still
ahead, cancelled ones were never due), in whole percent: an agent decision for the owner to review.



Copy waiting for the owner (web, five staff languages, `calendar` area, keys `noshow.*`): the correction buttons
("Correct to fulfilled", "Correct to no-show"), its dialog (two questions, the 30-day line, "Correct", "Keep it"), its
errors, the flag's tag ("{count} no-shows") and tooltip, and the settings section (title, intro, the two fields and
their hints, the validation message, "Save the flag", "Saved."); the charts' names (`analytics` area: "No-show rate per
month", "No-show rate (%)", "Slots refilled from the waitlist", "Refilled slots", "No-show rate by practitioner", "No-show
rate by specialty", the percent unit, "No specialty", "No closed appointments").

After SAL-33, SAL-34 and SAL-35 merged: the no-show port uses main's `Ports.Rules` (the same `organization.Appointments.For`);
`AppointmentChanges.tsx` describes `status_corrected` ("Corrected to fulfilled", "Corrected to no-show", five languages,
waiting for the owner); the waitlist e2e journey now checks that the real acceptance's appointment is counted as a slot
refilled in the clinic's statistics. Still a follow-up: the flag in the calendar's week view (`Calendar.tsx` has no API
client; it needs one passed from `App.tsx` and one flags request per week).
