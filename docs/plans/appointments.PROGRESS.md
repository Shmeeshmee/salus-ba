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

Follow-ups: the flag in the calendar's week view after SAL-34's `Calendar.tsx`; the rendering of `status_corrected` rows in SAL-34's
`AppointmentChanges.tsx` after its merge.

Copy waiting for the owner (web, five staff languages, `calendar` area, keys `noshow.*`): the correction buttons
("Correct to fulfilled", "Correct to no-show"), its dialog (two questions, the 30-day line, "Correct", "Keep it"), its
errors, the flag's tag ("{count} no-shows") and tooltip, and the settings section (title, intro, the two fields and
their hints, the validation message, "Save the flag", "Saved."); the charts' names (`analytics` area: "No-show rate per
month", "No-show rate (%)", "Slots refilled from the waitlist", "Refilled slots", "No-show rate by practitioner", "No-show
rate by specialty", the percent unit, "No specialty", "No closed appointments").
