# Appointments: reminders, one-tap confirm and cancel, rescheduling, waitlist, no-shows

Epic SAL-32. Status: approved by the owner, 2026-11-13, with the answers under "Decisions". Design in ADR 0015. Step 0 lands the
shared contract (migration, Go types and seams, the clinic's settings, the token table, `pkg/sms`, `pkg/payments`);
SAL-33 to SAL-36 then run in parallel. Purchase protection (SAL-37) needs payments (SAL-23) and is not in this epic:
only its seam is placed.

## Tickets

| Ticket | Branch | What |
| --- | --- | --- |
| Step 0 | `appt-0-contracts` | This plan and ADR 0015 (phase A, documents); then the shared contract (phase B): migration, `appointment` fields, `appointment_change`, `appointment_action_token` and issuing a link, the clinic's appointment settings (`organization`), the FHIR participant status of a confirmation, the seams the wave fills, `pkg/sms` and `pkg/payments` (ports and fakes), the OpenAPI of every wave-1 route and the generated clients |
| SAL-33 | `appt-33-reminders` | Reminders per the clinic's settings (default 24 h and 1 h before) through push, e-mail and the paid SMS option through step 0's `pkg/sms`; SMS usage per clinic; the patient's channels, quiet hours and zone; scheduled through River, sent once across restarts, cancelled with the appointment |
| SAL-34 | `appt-34-confirm-reschedule` | Confirm or cancel with one tap (the link page, in-app actions); confirmation status in the web calendar; self-service rescheduling within the clinic's notice and limit, picking free slots of the same practitioner role; the freed slot handed to the waitlist; misuse tests of the links |
| SAL-35 | `appt-35-waitlist` | Join for a practitioner role, a specialty or a time window; freed slots offered one patient at a time with the clinic's hold, then the next; acceptance books (and charges through step 0's `pkg/payments` once prices exist); the clinic sees and orders the waitlist; concurrency tests |
| SAL-36 | `appt-36-noshows` | No-show marked by the practitioner or the system, auditable and correctable by the clinic; reports of no-show rate over time, by practitioner and specialty, and slots refilled from the waitlist, in SAL-20's clinic statistics; the repeat no-show flag |

## Waves

```
Step 0 (one PR): phase A (this plan, ADR 0015), the owner's answers, then phase B (the contract below)
Wave 1, in parallel, each in its own worktree and compose project:  SAL-33   SAL-34   SAL-35   SAL-36
```

- **Worktrees and stacks.** `git worktree add ../salus-appt33 -b appt-33-reminders origin/main` (and `salus-appt34`,
  `salus-appt35`, `salus-appt36`), each with `COMPOSE_PROJECT_NAME` (`salus-appt33` … `salus-appt36`) and its own
  `SALUS_IMAGE_TAG` in its git-ignored `deployments/.env`, an override without published host ports, as the chat's
  wave did. Never `git stash`; one worktree never touches another.
- **Merge order** in wave 1: SAL-33, SAL-34, SAL-35, SAL-36, one at a time, main merged into the next branch first
  (no force-push). SAL-33 first: it fills the reminder seam, and its reminder e-mail carries the link SAL-34's e2e
  starts from. SAL-35 after SAL-33: its offers go through SAL-33's deliveries, SMS and quiet hours (built against the
  plain e-mail and push until then). SAL-36 last: it reads `booked_via` that SAL-35 writes, extends SAL-20's
  statistics (`origin/analytics-clinic`, on main before step 0's phase B merges), and its reports cover every change
  the others make.
- **Migrations.** Main is at `00050` (2026-11-13), and `00051` to `00053` are taken by branches merging before this
  one. Step 0 takes `00054` as a placeholder; the wave's placeholders follow: SAL-33 `00055`, SAL-35 `00056`, SAL-36
  `00057` (SAL-34 needs none; `00058` if it finds it must). Each branch renumbers to the next free number when it
  merges main before its own merge, and its migration test file follows.
- **Shared files.** `api/openapi.yaml`, the generated clients (regenerated after each merge with `make gen-clients`,
  never hand-merged), `internal/api/app.go` and the router, `internal/worker` (registration lines), `internal/config`,
  `docs/architecture/archcheck/ownership.yaml`, `internal/retention/rules.go`, `docs/privacy/dpia.md`, the i18n
  catalogs and the sqlc output: the later merge resolves.
- **File ownership inside `internal/appointment`** (one domain, as `communication` was for the chat): each package
  adds its own files and touches the core files (`model.go`, `errors.go`, `repository.go`, `repository_postgres.go`,
  `query.sql`, `service.go`, `service_change.go`, `handler_http.go`, `handler_fhir.go`) only where it must, appending
  its own block (a port interface at the end of `repository.go`, a section headed by its ticket in `query.sql` and
  its implementation at the end of `repository_postgres.go`: archcheck lets only that file import the sqlc package).
  Test fakes go in `fakes_<area>_test.go`.

| Package | Its own files in `internal/appointment` | Elsewhere |
| --- | --- | --- |
| SAL-33 | `service_reminder.go` (fills step 0's seam), its block of `repository_postgres.go`, `service_reminder_test.go` | `internal/notification` `service_reminder.go`, `service_preferences.go`, `service_sms.go`, `repository_sms.go`, templates; the SMS provider's adapter under `pkg/sms/` once one is chosen; `internal/worker/reminders.go`; mobile `notification_settings_screen.dart`; web `ClinicSettingsReminders.tsx` |
| SAL-34 | `service_confirm.go`, `service_reschedule.go`, `service_link_use.go`, `handler_link.go` (the page), `handler_actions.go` (`ActionsHandler`), `templates/link/` | mobile `appointment_actions.dart`, `reschedule_screen.dart`; web `Calendar.tsx` (the badge), `AppointmentChanges.tsx`, `ClinicSettingsReschedule.tsx` |
| SAL-35 | `service_waitlist.go`, `service_offer.go`, `handler_waitlist.go` (`WaitlistHandler`), its block of `repository_postgres.go`, `service_purchase.go` (fills step 0's stub with the price lookup) | `internal/slot` `service_offer.go` (tentative hold and release); `internal/notification/service_waitlist.go`; `internal/worker/waitlist.go`; mobile `waitlist_screen.dart`; web `Waitlist.tsx`, `ClinicSettingsWaitlist.tsx` |
| SAL-36 | `service_noshow.go`, `handler_noshow.go` | `internal/clinicstats` (its files, the added dimension); web `AppointmentChanges.tsx` is SAL-34's: SAL-36 adds its rows' rendering after SAL-34's merge; web `ClinicSettingsNoShows.tsx`, the analytics charts' added series |

## Decisions

### Owner, 2026-11-13

The answers to the open questions of phase A, each the recommended option.

| # | Topic | Decision |
| --- | --- | --- |
| Q1 | Phone for SMS | The profile's phone may carry appointment reminders and waitlist offers by SMS, opt-in per patient and per channel, never marketing. docs/booking/SPEC.md §4 and docs/privacy/dpia.md are amended |
| Q2 | SMS provider | An EU/EEA-established provider with a DPA, delivering to BiH networks; not chosen yet. `pkg/sms` runs its fake everywhere until then; docs/privacy/processors.md lists it as planned, not wired in |
| Q3 | Confirmation in FHIR | The patient participant is `tentative` until confirmed, `accepted` after, `declined` once they cancel. This supersedes docs/booking/SPEC.md "each `status: accepted`" for the patient participant (built in step 0) |
| Q4 | Where a link lands | A page served by the api on its own origin, without JavaScript; App Links and Universal Links later, once the link domain is chosen |
| Q5 | Cancel from a link | Without signing in, on a second tap (the page's `POST`) |
| Q6 | Correcting a no-show | The appointment's practitioner and the clinic's admin, within 30 days of the start, each correction audited with an `appointment_change` row. Supersedes docs/booking/SPEC.md "a no-show appointment does not change again" and "admin: not change other statuses" for this correction only |
| Q7 | Repeat no-shows | A flag for staff only |
| Q8 | Charging on a waitlist acceptance | Through the payments seam, charging nothing until SAL-23 adds prices |
| Q9 | Retention | As in "Personal data", marked in the DPIA as placeholders that need a legal decision before production |

### Agent, for the owner to review (2026-11-13)

| Topic | Decision |
| --- | --- |
| Domain | Everything stays in the `scheduling` context's domains: `appointment` owns the reminders' schedule, the change history, the action tokens and the waitlist (FHIR's waitlist is an `Appointment` status); `slot` the tentative hold; `organization` the clinic's settings (beside its chat settings); `notification` the patient's channels, the deliveries and SMS usage. No new domain, so CLAUDE.md's lists do not change |
| Statuses | ADR 0015's table: `booked`, `cancelled`, `fulfilled`, `noshow` as today; confirmation is the patient participant's status (Q3), reschedule a new version of the same `Appointment`, the waitlist a `waitlist_entry` (FHIR-shaped `Appointment` with `waitlist` and `requestedPeriod`, JSON API only), an offer the slot's `busy-tentative` |
| Reschedule | The same Appointment with a new `period` and `slot_id`, same practitioner role only, by the patient only, within the clinic's notice and limit, never once the join window has opened; history in `appointment_change`. Not cancel and rebook: that tears down the questionnaire responses and the consultation's access and re-sends the cancellation (ADR 0015) |
| Choosing the new time confirms it | A reschedule sets `confirmed_at` (`via` `app`) |
| Reminder schedule | `appointment_reminder` rows per appointment and offset, taken from the clinic's settings at booking, acceptance or reschedule; a River job per row at `due_at`, unique by the row; cancelled in the cancellation's and reschedule's transaction; an hourly sweep repairs lost jobs. A settings change applies to appointments booked after it |
| Sent once | Dispatch and the per-channel deliveries in one transaction; each send re-checks the reminder and the appointment; at most a repeat on a crash between the provider's acceptance and the commit (ADR 0015) |
| Time zones | `due_at` is absolute (start − offset). Times are shown in the clinic's zone with its name and, when the patient's zone differs, in theirs too. Quiet hours in the patient's zone, from their settings (the device's IANA zone, sent by the app), else the clinic's |
| Quiet hours | Default 22:00 to 08:00, the patient may change or turn them off. Push and SMS inside them wait until they end if that is still 30 minutes before the start, else are dropped; e-mail is never held; nothing is moved earlier |
| Channels | The patient's settings: push and e-mail on, SMS off by default. SMS only with the clinic's paid option on, a phone in the profile and the patient's opt-in (Q1) |
| Push text | Generic and without a data payload, as the chat's (SAL-29): Google and Apple are outside the EU/EEA, and an appointment with a named clinic is health data. The app opens the upcoming appointments |
| Legacy reminder | `appointment_reminder_email` stops being enqueued; its worker stays as a no-op for jobs enqueued before the deploy and is removed in a later release. No backfill of rows: appointments booked before keep their one 24-hour e-mail |
| Waitlist order | `priority` descending, then `created_at`: first come, first served, and the clinic's staff may move an entry up (`priority`). Default off per clinic: the admin turns it on |
| Waitlist matching | Same clinic; the entry's role, or any active role of its specialty; the slot's start inside the entry's dates and, when given, its time of day (clinic zone); at least the hold plus 60 minutes ahead; the patient's care consent active; not the patient who freed it; no other appointment of theirs overlapping. An entry whose patient is in quiet hours is passed over for that offer. One open offer per slot and per entry |
| Hold | In Postgres: `waitlist_offer` with `expires_at`, the slot `busy-tentative`, an expiry job at `expires_at`. Acceptance and expiry serialise on the offer's row lock; acceptance moves the slot to `busy` and inserts the appointment (the exclusion constraint guards last). Not Redis: it would be the only copy (CLAUDE.md, the booking SPEC's "No Redis slot hold") |
| Entry lifetime | Active until booked through an offer, withdrawn by the patient or removed by staff, past its latest date, or the care consent revoked (the `consent_revoked` job withdraws them and expires open offers). A decline or an expired offer leaves the entry active. At most 5 active entries per patient and clinic; dates at most 90 days ahead |
| Accepting | In the app only, signed in (it books and, once prices exist, charges). The offer's e-mail and SMS say to open the app |
| Payments | `pkg/payments` (port and fake, step 0). Today's booking charges nothing and keeps doing so: no price exists. Both booking paths go through `service_purchase.go`, which charges only when a price exists (never until SAL-23); the fake tests the order (charge before the transaction, refund when it fails) |
| Purchase protection seam | `settle` in `service_purchase.go`, called by the patient's cancellation and reschedule, a no-op until SAL-37 |
| No-show | Marked by the practitioner (as today) or the system (the consultation's end without the patient, `RecordConsultationOutcome`, as today); both write an `appointment_change` (`status_set`, `via` `staff` or `system`). Corrected `noshow` ↔ `fulfilled` within 30 days of the start (`status_corrected`) by the practitioner or the admin (Q6), enqueuing the clinic-day refresh |
| Repeat no-shows | A flag only: with the clinic's threshold on, staff see "N no-shows in the last D days" beside the patient's appointments, computed on read, never stored. Requiring prepayment or protection waits for SAL-23 and SAL-37; no blocking of self-booking (Q7) |
| Reports | SAL-20's `clinic_appointment_stat` gains `booked_via`; `GET /analytics/clinic/appointments` gains additive fields (below). The no-show rate is computed from the counts, by the client |
| JSON naming | Fields added to the existing `Appointment` schema follow its camelCase; new schemas use snake_case, as the chat and analytics routes |

## Contract (step 0, phase B, migration 00054 as a placeholder)

### Tables and columns

| Table | Columns |
| --- | --- |
| `appointment` (added) | `confirmed_at`, `confirmed_via` (`app` \| `link`; both null or both set), `reschedule_count` (0 default), `booked_via` (`direct` \| `waitlist`, `direct` default). Grants `UPDATE (period, slot_id, confirmed_at, confirmed_via, reschedule_count)` beside the status columns; the 00019 comment's "when never changes" is amended in 00054's |
| `appointment_change` | `id`, `appointment_id`, `organization_id`, `patient_id` (copied for row-level security), `at`, `kind` (`confirmed` \| `rescheduled` \| `status_set` \| `status_corrected`), `via` (`app` \| `link` \| `staff` \| `system`), `by_account_id` (null for the system), `agent` (a job's kind, for the system), `from_status`, `to_status`, `from_period`, `to_period`, `from_slot_id`, `to_slot_id`. Append-only: `SELECT, INSERT` only, triggers refuse `UPDATE`, `DELETE` and `TRUNCATE` for every role, as `communication`. Row-level security as `appointment`'s (the patient's own; the clinic's with the consent, and the past ones after a revocation) |
| `appointment_action_token` | `id`, `token_hash` (SHA-256, unique), `appointment_id`, `organization_id`, `patient_id`, `account_id`, `reminder_id` (SAL-33 adds its foreign key), `channel` (`email` \| `sms`), `language`, `created_at`, `expires_at`, `used_at`, `used_action` (`confirm` \| `cancel`). Found by hash only through `appointment_action_token_find(hash)`, `SECURITY DEFINER`, returning the ids; inserted in a notifier job's clinic scope with `app.appointment_id` set |
| `organization_appointment_settings` | `organization_id` (key), `reminder_offsets_minutes` (`integer[]`, 0 to 3 distinct values, 15 to 10080; default `{1440, 60}`), `sms_reminders` (false), `reschedule_notice_hours` (0 to 168, 24), `reschedule_max` (0 to 5, 2), `waitlist_enabled` (false), `waitlist_hold_minutes` (5 to 120, 15), `noshow_flag_threshold` (0 to 10, 0 for off), `noshow_flag_window_days` (30 to 730, 180), `created_at`, `updated_at`. Not personal data. No version: like the chat settings, a `PUT` replaces the whole row |

### Go

- `appointment`: the new fields on `Appointment`; `Change`, `ChangeKind`, `Via`, `BookedVia`; errors
  `ErrRescheduleWindowClosed`, `ErrRescheduleLimit`, `ErrNotReschedulable`, `ErrLinkInvalid` (unknown, used and
  expired alike); `Service.Changes(ctx, c, id)` (the appointment's history, audited, at `Full` access) and the
  unexported `recordChange` every later write uses; `Service.IssueLink(ctx, agent, org, appointmentID, reminderID,
  channel) (url string, err error)` (a token for a notifier job, audited with the job as agent).
- Seams the wave fills, each called where it belongs from step 0 on:

| Seam | Step 0 | Filled by |
| --- | --- | --- |
| `scheduleReminders(ctx, a, now)` in `service_reminder.go`, called by `book` | today's `EnqueueReminder` (24 h), moved here | SAL-33: the rows and jobs; SAL-34 calls it from the reschedule, SAL-35 from the acceptance |
| `cancelReminders(ctx, id)`, called by `cancelled` | nothing | SAL-33 |
| `slotFreed(ctx, a, slotID)`, called by `cancelled` | enqueues `waitlist_slot_freed {slot_id, organization_id}` on the default queue; a no-op worker completes it | SAL-35: the real worker; SAL-34 calls it from the reschedule |
| `purchase(ctx, …)` and `settle(ctx, a, …)` in `service_purchase.go` | nothing: no price exists | SAL-35 (`purchase` with `pkg/payments`), SAL-37 (`settle`) |
| `recordChange` in `SetStatus` and `RecordConsultationOutcome` | writes `status_set` | SAL-36 adds the correction |

- `organization`: `AppointmentSettings` (the columns above, JSON snake_case), `organization.Appointments` (a second
  service of the domain like `Chat`): `Settings` and `SetSettings` for the clinic's admin (audited, refusals too,
  every range validated here so the wave never edits it), and `For(ctx, org)` for the other domains in their
  transaction, unaudited (not personal data).
- `Ports.Tokens` and `Ports.LinkURL` of `appointment.Service` for `IssueLink`; SAL-33 adds `APPOINTMENT_LINK_URL` (the
  link page's base, the api's public origin) to `internal/config` and wires both into the notifier, its only issuer.
- New public routes (`/a/`, `/waitlist`, `/me/notification-settings` and the others) get their Vite proxy and
  Caddyfile entries (`scripts/check-caddy-routes.sh`) in the package that mounts them.

### HTTP (`api/openapi.yaml`)

| Route | What |
| --- | --- |
| `GET`, `PUT /clinic/appointment-settings` | The admin's: the settings above, every field required on `PUT` |
| `GET /appointments/{id}/changes` | `{changes: [AppointmentChange]}`, oldest first, for whoever sees the appointment at `Full`: `kind`, `via`, `at`, `from_status`, `to_status`, `from_start`, `from_end`, `to_start`, `to_end`; never the account or agent to the patient |
| `Appointment` schema | gains `confirmedAt`, `confirmedVia`, `rescheduleCount`, `bookedVia` |

FHIR: the patient participant's status per Q3, built in step 0 (golden files).

`pkg/sms` (`Sender`, `Message`, `Segments`, `smstest`) and `pkg/payments` (`Provider`, `Charge`, `Refund`, `Receipt`,
`paymentstest`) are built in step 0, as ADR 0015 describes; `config.LoadSMS` gives the fake unless `SMS_ENGINE=http`
comes with its endpoint and token file; nothing loads it before SAL-33, and the generic adapter stays off until the
provider is chosen. Every wave-1 route below is in `api/openapi.yaml` from step 0 (`x-salus-package` names its
builder), so the generated clients and the screens can be built in parallel; a route answers 404 until its package
mounts it.

## Contract of wave 1 (settled in step 0)

The shapes each package builds and the others call. A package may add a field, never rename or drop one; a change
it needs is reported, not made. Their paths and schemas are in `api/openapi.yaml` from step 0.

**SAL-33 (reminders).**

| Route, job or table | What |
| --- | --- |
| `GET`, `PUT /me/notification-settings` | The patient's: `channels {push, email, sms}`, `quiet_hours {start, end}` (HH:MM, may wrap midnight) or null, `time_zone` (IANA). `sms: true` needs a phone in the profile (409 `phone_required`). Staff get 403 |
| `GET /clinic/sms-usage?from=YYYY-MM&to=YYYY-MM` | The admin's: `{months: [{month, messages, segments}]}` |
| `appointment_reminder` (00055) | `id`, `appointment_id`, `organization_id`, `offset_minutes`, `start_at` (the start it was scheduled for), `due_at`, `state` (`scheduled` \| `dispatched` \| `cancelled` \| `skipped`), `created_at`, `dispatched_at`; unique `(appointment_id, offset_minutes, start_at)`. Owned by `appointment` |
| `notification_delivery` (00055) | `id`, `kind` (`reminder` \| `waitlist_offer`), `source_id`, `organization_id`, `account_id`, `channel` (`push` \| `email` \| `sms`), `device_id`, `state` (`pending` \| `sent` \| `cancelled` \| `failed`), `segments`, `created_at`, `sent_at`; unique `(kind, source_id, channel, device_id)`. No address, number, token or text. Owned by `notification` |
| `notification_preference` (00055) | `account_id` (key, cascades with the account), `push`, `email`, `sms`, `quiet_start`, `quiet_end`, `time_zone`, `updated_at`. Owned by `notification` |
| `sms_usage` (00055) | `organization_id`, `month`, `messages`, `segments`; updated in the SMS delivery's transaction. Not personal data |
| Jobs | `appointment_reminder {reminder_id, organization_id}` (notifier, `push` queue: the dispatch), `reminder_email`, `reminder_push`, `reminder_sms {delivery_id}` (notifier; `email`, `push`, and a new `sms` queue), `appointment_reminders_sweep` (scheduler, hourly, default queue). Opaque ids only |
| Ports SAL-35 reuses | `notification.Service.Deliver(ctx, kind, sourceID, org, accountID, content)` in the caller's transaction: the channel choice, quiet hours, the delivery rows and their jobs |

**SAL-34 (confirm, cancel, reschedule).**

| Route | What |
| --- | --- |
| `POST /appointments/{id}/confirm` | The patient's own, while `booked` and ahead; 200 `Appointment`; repeating it is 200 with the same |
| `POST /appointments/{id}/reschedule` | The patient's own: `{slotId}` (a free slot of the same role, ahead); 200 `Appointment` at its new version; 409 `reschedule_window_closed`, `reschedule_limit`, `not_reschedulable` (another role, the join window open, self-service off), `slot_unavailable`, `appointment_closed` (the existing code) |
| `GET /appointments/{id}/actions` | For the patient's app: `{confirm, cancel, cancel_until, reschedule, reschedule_until, reschedules_left}` |
| `GET /a/{token}` | The link page (`text/html`, no session): the clinic, date and time, the state, and the possible actions as buttons |
| `POST /a/{token}` | Form `action=confirm\|cancel`; 303 to `GET /a/{token}`, which then shows the result. A used, expired or unknown token: the same page saying the link no longer works |

**SAL-35 (waitlist).**

| Route, job or table | What |
| --- | --- |
| `POST /waitlist` | The patient: `organization_id`, `practitioner_role_id` or `specialty` (one of them), `earliest`, `latest` (dates in the clinic's zone), optional `times {start, end}`; 201 `WaitlistEntry` (`id`, those fields, `status` `active` \| `booked` \| `withdrawn` \| `expired`, `created_at`, `offer` when one is open: `{id, slot_start, slot_end, expires_at}`). 409 `consent_required`, `waitlist_off`, `too_many_entries` |
| `GET /me/waitlist` | The patient's entries |
| `DELETE /waitlist/{id}` | The patient withdraws, or the clinic's staff remove; 204 |
| `POST /waitlist/offers/{id}/accept` | The patient; 201 `Appointment` (`bookedVia: waitlist`); 409 `offer_expired`, `slot_unavailable`, `consent_required` |
| `POST /waitlist/offers/{id}/decline` | The patient; 204; the next entry gets an offer |
| `GET /clinic/waitlist?practitioner_role=&status=` | The clinic's staff: entries in offer order with `priority`, the patient's name at the level `access.Profile` gives, open offers |
| `PUT /clinic/waitlist/{id}/priority` | Staff: `{priority}` (0 to 100) |
| `waitlist_entry`, `waitlist_offer` (00056) | Owned by `appointment`; the offer: `id`, `entry_id`, `slot_id`, `organization_id`, `patient_id`, `created_at`, `expires_at`, `state` (`open` \| `accepted` \| `declined` \| `expired` \| `withdrawn`), `appointment_id`; partial unique indexes on `slot_id` and `entry_id` where `open` |
| `slot.status` | gains `busy-tentative`; `slot.Service.Offer(org, id)` free → tentative, `TakeOffered` tentative → busy, `Unoffer` tentative → free while ahead |
| Jobs | `waitlist_slot_freed` (worker, default queue; replaces step 0's no-op), `waitlist_offer_expire {offer_id, organization_id}` (worker, at `expires_at`), `waitlist_offer_notify {offer_id, organization_id}` (notifier) |

**SAL-36 (no-shows).**

| Route | What |
| --- | --- |
| `POST /appointments/{id}/status` | Unchanged for `booked`; also takes `fulfilled` for a `noshow` and `noshow` for a `fulfilled` within 30 days of the start, by the practitioner or the admin (Q6); 409 `correction_window_closed` |
| `GET /clinic/noshow-flags?from=&to=` | Staff, with the clinic's threshold on: `{flags: [{appointment_id, noshows, window_days}]}` for the appointments in range the caller sees at `Full`; empty when off |
| `GET /analytics/clinic/appointments` | SAL-20's route (on main since #82); SAL-36 adds to its schema, additive: `series[].refilled` (booked through the waitlist), `practitioners[].noshow` and `.booked`, `specialties: [{specialty, booked, fulfilled, noshow, cancelled}]` (admins; as SAL-20's rule C2) |
| `clinic_appointment_stat` (00057) | gains `booked_via` in its key; the refresh and the nightly pass count it |

## Notes for the wave-1 packages

- **SAL-33.** Read `docs/plans/chat.PROGRESS.md` SAL-29 first: the fan-out job, per-device pushes, the notifier's
  queues, `patient.NoticeAccount`, the templates in ten languages; reuse them. The phone comes from a new audited
  `patient.NoticePhone` (job as agent), only for an SMS delivery. The SMS text fits one GSM-7 segment where the
  language allows (copy below). `TestReminder_SentOnceAtTheRightInstant` covers a patient in Asia/Dubai with a
  clinic in Europe/Sarajevo across the October DST change, quiet hours on both sides, a restart (the job run twice,
  the dispatch rolled back once), a cancellation and a reschedule between scheduling and dispatch. Measure: the
  sweep over 100 000 scheduled reminders, and dispatch plus sends at 1 000 reminders due in the same minute; numbers
  in PROGRESS.
- **SAL-34.** The page in `html/template`, its strings from `internal/appointment/templates/link/<language>.json` in the ten
  patient languages, added to `scripts/i18n-check.sh`; the token's language picks it. Misuse tests: a `GET` by a
  prefetcher (any number) changes nothing; a second `POST` is refused; an expired, a revoked-consent, a cancelled and
  a rescheduled appointment's token; another appointment's id with a valid token; a `POST` without or with a foreign
  `Origin`; a token in the log or the audit (never); the rate limit. The reschedule: concurrent reschedules of one
  appointment, a reschedule racing a booking of the target slot and a cancellation, the exclusion constraint's
  refusal audited in its own transaction (as `Book`). e2e from a reminder in Mailpit: confirm through the page,
  cancel through the page, reschedule through the API as the app does after "open the app". Caddy's log for the
  api host must not carry the URI of `/a/`.
- **SAL-35.** Concurrency tests on Postgres: an acceptance racing the expiry, two acceptances (two devices), an
  acceptance racing a direct booking (which never sees the slot), a cancellation racing an offer, two freed-slot jobs
  for one slot, and after every round the invariant that a slot is `busy` exactly when a non-cancelled appointment
  holds it and `busy-tentative` exactly when an open offer does. Measure the directory's free-slot search with
  tentative slots (main's 82 s to 4 ms fix must hold) and the matching query with 10 000 entries in a clinic.
  `pkg/payments`' fake proves the order only; nothing is charged.
- **SAL-36.** Equality tests as SAL-20's: the reports match the raw appointments and their changes for a generated
  clinic (every status, corrections, waitlist bookings, a DST change). Check that the window-close path records the
  outcome; add the system's marking there if it does not. The flag's count is the patient's `noshow` appointments at
  the clinic in the window, read in the clinic's scope.

## Checks per PR

`make lint`, `make test`, `make archcheck`, `make cover`, `make e2e`, `make i18n-check`; `make fhir-validate` where
the golden files changed (SAL-34); `make web-typecheck`, `make web-test`, `make mobile-analyze`, `make mobile-test`
where they changed; `make gen-clients` leaves no diff; CI green. A fresh subagent that has not seen the work reviews
each PR's diff (Copilot is out of quota until 2026-12-01); valid findings are fixed before the merge, declined ones
listed in the PR. Each merge writes its section in `docs/plans/appointments.PROGRESS.md`.

## Copy waiting for the owner

Every new user-facing string goes to the owner first, listed per PR and in PROGRESS when merged. Expected:

- SAL-33: the reminder e-mail with its two links (ten languages), the generic reminder push, the SMS (whether to drop
  diacritics so a message fits one 160-character GSM-7 segment instead of 70 characters of UCS-2: the owner's
  call), the second time in the patient's zone, the mobile notification settings (`legal`: the SMS opt-in), the web
  reminders and SMS section with its price (`[PRICE]` placeholder until the owner sets it).
- SAL-34: the link page in ten languages (`legal`: the cancellation's effect, "inside 24 hours, contact the clinic"),
  the reschedule e-mail, the mobile actions and picker, the web badge ("Confirmed", "Not confirmed") and history.
- SAL-35: the offer push, e-mail and SMS, the mobile waitlist screens (`legal`: that accepting books the time, and
  pays for it once payments exist), the web waitlist and its settings.
- SAL-36: the web correction dialog, the flag's tooltip, the report series' names.

## Personal data

| Field | Why |
| --- | --- |
| `appointment.confirmed_at`, `confirmed_via` | Whether the patient said they will come: the clinic plans its day and calls the unconfirmed |
| `appointment.reschedule_count`, `booked_via` | The clinic's reschedule limit; the report of slots refilled from the waitlist |
| `appointment_change` (all columns) | Who changed an appointment's time or status, from what to what, and how: the no-show's audit and its correction (SAL-36 AC), a reschedule's history; kept with the appointment |
| `appointment_action_token` | A one-time link in one message, bound to its appointment and patient; the hash only, never the token; deleted 30 days after it expires |
| `appointment_reminder` (SAL-33) | When each reminder is due and whether it went; deleted 90 days after its due time |
| `notification_delivery` (SAL-33) | Which channel a reminder or an offer went through and when, for "sent once" and SMS usage; no address, number or text; deleted 90 days after it was sent |
| `notification_preference` (SAL-33) | The patient's channels and quiet hours, and their device's time zone, only to respect quiet hours and show the time in their zone (it can show that they are abroad: kept to the zone name, replaced on each update, deleted with the account) |
| `patient.phone`, newly read for SMS (SAL-33) | Only for a reminder or offer by SMS the patient opted into, at a clinic that offers it (Q1: the phone's purpose amended) |
| `waitlist_entry` (SAL-35) | `patient_id`, `organization_id`, the role or specialty, dates and times, `priority`, `status`, timestamps: to offer a freed slot to someone who wants one; that a patient waits for a specialty is health data (Art. 9). Deleted 30 days after it closes |
| `waitlist_offer` (SAL-35) | Who was offered which slot and what they did, for the order and the hold; deleted with its entry |
| River job arguments | Opaque ids and an instant only, as today |
| SMS and e-mail content | The clinic's name, the date and time, a link; never a practitioner, specialty or anything clinical |

Processors: the SMS provider (Q2, planned) and any payment provider (SAL-23) get an entry in `docs/privacy/processors.md` and
a signed DPA before they are switched on; until then the fakes run in every environment. Each package adds its part
to `docs/privacy/dpia.md` (section "Appointments (`docs/plans/appointments.md`)") and its retention rules to
`internal/retention/rules.go`; every period above needs a legal decision before production, as the booking's.

## As built in step 0 (phase B)

See `docs/plans/appointments.PROGRESS.md`.

## Open questions for the owner

None open: the owner answered Q1 to Q9 on 2026-11-13 ("Decisions", Owner).
