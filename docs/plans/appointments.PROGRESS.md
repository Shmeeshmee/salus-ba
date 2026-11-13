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
added to the 409s). `notification.Waitlist` sends the offer's e-mail and push (ten languages) for the
`waitlist_offer_notify` job. Web: `Waitlist.tsx` (the clinic's list, priority, removal) at `/waitlist` and
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
| Offers until SAL-33 | `appointment.OfferNotices` is the port: today `NewRiverOfferNotices` enqueues `waitlist_offer_notify`, which the notifier sends as a plain e-mail and a push per device (`notification.Waitlist`); a retry may repeat a message and there is no SMS. At SAL-33's merge, implement `OfferNotices.NotifyOffer` with `notification.Service.Deliver(ctx, "waitlist_offer", offerID, org, accountID, content)` in the offer's transaction (the account from `patient.NoticeAccount`), and drop the job and `notification/service_waitlist.go`'s sends |
| Quiet hours | `Ports.Quiet` (`QuietHours`) is a seam, nil in every binary until SAL-33: nobody is passed over. SAL-33 wires its notification settings there |
| Handing a slot on | One path, `handOn`: the job locks the slot (`ForWaitlist`), leaves it when booked or held by an open offer, gives back a held slot whose offer closed, then offers it to the first entry the matching query and the consent check give, or leaves it free. Decline, withdrawal and revocation close the offer in the patient's scope and enqueue `waitlist_slot_freed`; the expiry closes it and hands on in its own transaction |
| Offer visibility | The clinic's jobs see every offer of the clinic whatever the patient's consent (they must see an open offer of a slot to leave it held); entries stay consent-gated, and an offer is inserted only for an entry the clinic sees |
| Offer's role | `waitlist_offer` gains `practitioner_role_id` (the slot's), so an acceptance books without reading the slot first |
| `expired` | Not stored: an active entry past `ends_at` (the end of its last day in the clinic's zone) shows as `expired`, matches nothing and no longer counts towards the five; retention deletes it 30 days after `ends_at` |
| Overlap check | "No other appointment of theirs overlapping" is checked in the clinic's scope, so only the same clinic's appointments count (row-level security hides the others) |
| Matching times | An entry's times are minutes after the clinic's midnight; a slot matches when it lies wholly inside them |
| Staff | Any staff of the clinic list the waitlist, move an entry and remove one; names follow `access.Profile` |
| Started slots | `Unoffer` frees a held slot even after its start (no offer can hold it, and retention deletes free slots only); the contract said "while ahead" |
| Audit of the entry | An entry is audited as `Appointment/<entry id>` (FHIR's waitlist is an Appointment status); the job's read of candidates as a search without an entity |
| The five-entry limit | Counted in the join's transaction without a lock: two joins at the same instant could make six (harmless; no lock added without a measurement) |
| Retry key | The acceptance's key `waitlist:<offer>` replays the first receipt on a retry after a refund, as the direct booking's key does: SAL-23 settles both |
| Join entry point (mobile) | From a practitioner's free times, for that practitioner, over 7, 14, 30 or 90 days, any time, mornings (08:00 to 12:00) or afternoons (12:00 to 17:00); joining for a specialty is in the API only |

Copy waiting for the owner: the offer e-mail and push in ten languages
(`internal/notification/templates/*/waitlist_offer*.txt`); the mobile `waitlist*` strings in ten languages (`legal`:
`waitlistExplained`, `waitlistAcceptEffect`, `waitlistAccept`); the web `waitlist` area in five languages.
