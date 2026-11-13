# 15. Appointments: one Appointment through every change, reminders and waitlist holds in Postgres, one-tap links

Status: accepted, 2026-11-13 (docs/plans/appointments.md, epic SAL-32), with the owner's answers to Q1 to Q9 (the plan's
"Decisions", Owner). They supersede, in docs/booking/SPEC.md, the phone's "never used for SMS" (now: reminders and
waitlist offers, opt-in), "each participant `accepted`" for the patient participant, and "a no-show does not change
again" for the clinic's correction within 30 days.

## Context

SAL-32 asks for fewer no-shows and fuller schedules: reminders (push, e-mail, optional paid SMS), one-tap confirm or
cancel from a reminder, self-service rescheduling within the clinic's rules, a waitlist that offers freed slots one
patient at a time with a short hold, and no-show tracking with reports. Purchase protection (SAL-37) needs payments
(SAL-23) and is out of scope; only its seam is designed here.

What exists (docs/booking/SPEC.md, settled): an `appointment` is booked at once from a free `slot` in one transaction
(`slot` free to busy, then the insert), guarded twice against a double booking (the exclusion constraint
`appointment_no_double_booking` and the slot's `UPDATE … WHERE status = 'free'`), with "No Redis slot hold". Statuses
`booked`, `cancelled`, `fulfilled`, `noshow`; the patient cancels until 24 hours before, staff at any time; a status
other than `booked` never changes again. Every FHIR participant is `accepted`. Times are stored in UTC and shown in
the clinic's zone. Migration 00019 grants `UPDATE` only on the status columns: "who, with whom, where and when never
change". One reminder e-mail, `appointment_reminder_email`, is a River job scheduled 24 hours before the start that
sends only while the appointment is `booked`. Push (`pkg/push`, a fake by default) and the notifier's River queues
exist since SAL-29; push alerts carry a fixed text and no data payload.

## Decision

### Status and FHIR R4

| What happens | `Appointment.status` | Patient `participant.status` | Stored |
| --- | --- | --- | --- |
| Booked, not confirmed | `booked` | `tentative` (Q3) | `confirmed_at` null |
| Confirmed by the patient (link or app) | `booked` | `accepted` | `confirmed_at`, `confirmed_via` (`link` \| `app`), an `appointment_change` row |
| Cancelled by the patient (their app or a link) | `cancelled`, `cancelationReason` `pat` | `declined` | as today (`cancelled_by_account_id` their account), plus a `status_set` change row |
| Cancelled by staff or a revocation | `cancelled`, `prov` or `pat` | unchanged: the patient did not decline (a revocation cancels with `pat` and no account, which is how it is told apart) | as today, plus a `status_set` change row (`via` `staff` or `system`) |
| Rescheduled by the patient | `booked`, new `start`, `end`, `slot`; `meta.versionId` + 1 | `accepted` (choosing the time confirms it) | `period` and `slot_id` updated, `reschedule_count` + 1, a change row with the old and new period and slot |
| Attended | `fulfilled` | unchanged | a change row (`status_set`, by the practitioner or the system) |
| Not attended | `noshow` | unchanged | a change row (`status_set`), corrected later by a `status_corrected` row |
| On the waitlist | not an Appointment row: a `waitlist_entry`, FHIR-shaped as an `Appointment` with status `waitlist` and `requestedPeriod`; JSON API only, no FHIR interaction added | | `waitlist_entry` |
| A freed slot offered to a waitlisted patient | the slot's `status` `busy-tentative` (FHIR `SlotStatus`) | | `waitlist_offer` |

`booked` with a `tentative` patient is a deliberate reading of R4: the patient booked the slot themselves, so the
appointment is confirmed to go ahead (`booked`, not `pending`, which R4 keeps for participants who have not yet
finalised an appointment request), while the participant's status says whether they have reconfirmed that they will
come. Practitioner and role participants stay `accepted`. The FHIR `Appointment` read and search keep their interactions;
the CapabilityStatement does not change.

### A reschedule is the same Appointment with a new time

A patient's reschedule moves the appointment to another free slot **of the same practitioner role**, in one
transaction: the new slot is held (free to busy), `period` and `slot_id` are updated (the exclusion constraint
checks the new period), the old slot is released while ahead, the reminders are cancelled and scheduled again, an
`appointment_change` row keeps the old and new period and slot, the change is audited, and the freed slot goes to
the waitlist (a job). Migration 00054 (placeholder number) grants `UPDATE (period, slot_id)`; who and with whom still
never change.

Why not cancel and book again with a link: everything else of the visit hangs on the appointment's id, and a
cancellation tears it down. A cancellation discards the questionnaire responses (`questionnaire_responses_discard`),
ends the consultation's access (`consultation_access_end`), sends the cancellation e-mail and asks for the default
questionnaire again on the new booking. Each would need an exception for "cancelled because rescheduled", plus moving
the patient's answers and files across appointments owned by another domain. Keeping the id keeps the
questionnaire, the chat's link and the encounter that follows. FHIR R4 has no reschedule element (R5 adds
`previousAppointment`, for a rebooking); updating the resource with a new version is ordinary FHIR. The
history the FHIR resource does not carry is the `appointment_change` table, append-only, kept with the appointment.

Limits: a reschedule is refused once the consultation's join window has opened, whatever the clinic's notice, so a
video session never changes time under its participants. Staff rescheduling is not part of this epic (staff cancel).

### Reminders: rows in Postgres, River jobs at their due time

- **Per appointment.** Booking, a waitlist acceptance and a reschedule insert one `appointment_reminder` row per
  offset of the clinic's settings at that moment (default 24 hours and 1 hour before the start), each with its
  `due_at` (start − offset, an absolute instant) and a River job `appointment_reminder` scheduled at `due_at`, unique
  by the reminder's id, in the same transaction. A reminder whose `due_at` has passed is not created.
- **Cancelled with the appointment.** A cancellation or a reschedule sets the appointment's scheduled reminders to
  `cancelled` in its transaction. A job that finds its reminder not `scheduled`, or its appointment not `booked` with
  the start it was scheduled for, does nothing.
- **Sent once.** The dispatch job locks its reminder, decides the channels, inserts one `notification_delivery` row
  per channel (and per device for push; unique by source, channel and device) with a send job each, and sets the
  reminder `dispatched`, all in one transaction: a restart before the commit redoes all of it, after the commit none
  of it. A send job locks its delivery, re-checks that the reminder and the appointment still hold, sends, and marks
  it `sent`. The one window left is a crash between the provider's acceptance and that commit: the retry sends again.
  SMS passes the delivery's id as the provider's client reference where the provider deduplicates on it. An hourly
  `appointment_reminders_sweep` re-enqueues the job of any `scheduled` reminder past its due time (River's unique key
  makes that a no-op when the job is still there), so a lost job is repaired.
- **Time zones.** Instants are UTC; `due_at` does not depend on any zone, so a tourist abroad gets the reminder at the
  same instant as at home. What the zone changes: the text gives the time in the clinic's zone with the zone's name
  (the booking rule) and, when the patient's zone differs, in the patient's zone too; quiet hours are the patient's,
  in the patient's zone (the IANA zone the app reports from the device, stored with their notification settings; the
  clinic's zone when unknown).
- **Quiet hours.** A push or SMS due inside them is held until they end if that is still at least 30 minutes before
  the start, and dropped otherwise (the earlier reminder stands). E-mail is silent and never held. A reminder is never
  moved earlier.
- **Channels.** The patient chooses push, e-mail and SMS in their settings (push and e-mail on by default, SMS off).
  SMS is sent only when the clinic has the paid option on and the patient has a phone and opted in (Q1: the
  phone's purpose is amended to reminders and waitlist offers).
- **Content.** The clinic's name, the date and time, and a link; nothing clinical, no practitioner or specialty. Push
  goes through Google and Apple outside the EU/EEA, so its alert stays generic like the chat's ("You have an
  appointment reminder"), with no data payload: the app opens the upcoming appointments. The e-mail and the SMS carry
  the one-tap link.
- The existing `appointment_reminder_email` job is replaced: it is no longer enqueued, and its worker keeps sending the
  jobs enqueued before the deploy, so an appointment booked before keeps its one 24-hour e-mail (the plan's "Legacy
  reminder"; amended in SAL-33); the worker goes in a later release, once those jobs have drained.

### The waitlist hold lives in Postgres

- A `waitlist_entry` is a patient's wish for a practitioner role or a specialty of a clinic, a date range and,
  optionally, a time of day in the clinic's zone, under an active care consent. Entries are offered in the order
  (`priority` descending, `created_at`): first come, first served, with the clinic able to move an entry up.
- When a slot is freed (a cancellation or a reschedule) the freeing transaction enqueues `waitlist_slot_freed`. The
  job, in one transaction, marks the slot `busy-tentative` (`UPDATE … WHERE status = 'free'`: a slot booked meanwhile
  is simply not offered), picks the first matching entry and inserts a `waitlist_offer` (`expires_at` = now + the
  clinic's hold, default 15 minutes) with its notification job and its expiry job at `expires_at`. A partial unique
  index allows one open offer per slot and one per entry.
- **Why Postgres and not Redis.** The offer is the only record of who may book that slot for the next 15 minutes and
  must survive a Redis restart; CLAUDE.md forbids Redis being the only copy, and the booking SPEC settled "No Redis
  slot hold". The slot's own status hides it from the public directory (`slot_directory` shows `free` only) without a
  second store.
- **Double booking stays impossible.** Accepting locks the offer (`FOR UPDATE`), checks it is open and unexpired,
  moves the slot from `busy-tentative` to `busy` (no row: refused) and inserts the appointment, where the exclusion
  constraint is the last word. The expiry job takes the same lock, so an acceptance and an expiry are serialised; the
  next entry gets the next offer, and with none left the slot goes back to `free` while ahead.
- Accepting happens in the app only, signed in: it books (and, once prices exist, charges), which a bearer link in a
  message must not do.

### One-tap links

- A link is `<APPOINTMENT_LINK_URL>/a/<token>`: 32 random bytes, base64url, stored only as its SHA-256 in
  `appointment_action_token`, bound to one appointment, its patient and account, the reminder and the channel it was
  sent in, the language to answer in, and the start it was issued for (`start_at`): a link whose `start_at` is not the
  appointment's start now is refused (SAL-34), so a reschedule leaves the old reminders' links useless; it expires at
  that start.
- **GET never changes anything.** Mail scanners and link previewers fetch links; a `GET` renders a page with what
  the message itself already said (clinic, date and time) and the actions still possible as buttons. The action is
  a `POST` of a form from that page (Post/Redirect/Get). No JavaScript, no cookies, no third-party requests; `Cache-
  Control: no-store`, `Referrer-Policy: strict-origin`, `X-Robots-Tag: noindex`, a strict CSP, and the `POST` checks
  that `Origin` is the page's own. Not `no-referrer`: under it browsers send `Origin: null` with the form's `POST`, which
  that check must refuse; `strict-origin` lets a Referer carry the api's origin only, never the token's path (SAL-34).
- **CSRF.** No session or cookie is involved, so no ambient authority exists to ride; the token in the form is the
  credential, and a cross-site attacker does not have it.
- **Single use and expiry.** The first `POST` uses the token with a conditional update (`SET used_at, used_action
  WHERE used_at IS NULL`) in the transaction that confirms or cancels, and a trigger refuses any change to a used token
  for every role, so a second action is impossible in the database too. A link is single use for one action: after a
  confirmation, cancelling needs a later reminder's link or the app. A token expires at the start it was issued for and
  is useless once the appointment is no longer `booked`. A used link's `GET` shows its holder its own outcome ("You
  confirmed this appointment", "You cancelled this appointment", with the clinic, date and time only) and offers
  nothing more; unknown and expired tokens get the same "this link no longer works" answer; token lookups are
  rate-limited per address in Redis (with a TTL).
- **Scanners that click.** Some mail security products open links in a sandboxed browser and may submit what they
  find; such a browser loads the page from its own origin, so its `POST` carries a same-origin `Origin` and passes that
  check, and the second tap alone does not stop it. Mitigation (cheap, in SAL-34): the cancel form needs a box ticked
  (`confirm_cancel=yes`, checked by the server), which a scanner that only presses buttons does not do; confirming
  needs nothing more, as a wrongly recorded confirmation harms no one. Accepted risk: a sandbox that also ticks boxes
  could cancel. What is then true: the appointment is cancelled and its slot freed (and offered to the waitlist), the
  patient gets the cancellation e-mail at once, and they can book again only if the time is still free; it is not
  undone automatically. The link is single use and lets nothing else happen, and the same scanner already sees the
  reminder's contents. If it is seen in practice, the cancel moves to the app only.
- **Binding.** The action is done as the token's patient on the token's appointment only: the transaction is scoped
  to that patient as their app would be, after a `SECURITY DEFINER` lookup by hash; the audit event names the
  patient's account with the agent `action_link`.
- **What it reveals.** Only the clinic, the date and time and the appointment's state; never a name, practitioner or
  specialty. A forwarded link lets its holder confirm or cancel that one appointment, as the reminder itself would.
- **What it offers.** Confirm while unconfirmed; cancel while the patient's 24-hour window is open (inside it, the
  page says to contact the clinic); reschedule only as "open the app", where the patient is signed in. The page is
  server-rendered by the api on its own origin (Q4); App Links and Universal Links come later, once the link domain is
  chosen. Cancelling needs no sign-in, only the page's second tap (Q5).
- The route pattern is logged, never the path (`pkg/middleware.AccessLog`); Caddy's access log for that host must not
  carry the URI (checked in SAL-34).

### Clinic rules

One row per clinic, `organization_appointment_settings`, owned by `organization` beside the chat settings, defaults
when none is stored, read and changed by the clinic's admin only (`GET`, `PUT /clinic/appointment-settings`, audited):
reminder offsets (0 to 3, each 15 minutes to 7 days, default 24 h and 1 h), the paid SMS option (off), the reschedule
notice (0 to 168 hours, default 24) and limit (0 to 5 per appointment, default 2; 0 turns self-service rescheduling
off), the waitlist (off) and its hold (5 to 120 minutes, default 15), and the repeat no-show flag (a threshold, 0 for
off, over a window of 30 to 730 days, default 180). The patient's 24-hour cancellation rule stays fixed (settled).

### `pkg/sms`, `pkg/payments` and the purchase seam

- **`pkg/sms`** (step 0): `Sender.Send(ctx, Message{To (E.164), Body, Reference}) (providerID string, err error)` and
  `Segments(body)` (GSM-7 or UCS-2 parts, for usage), `smstest` the in-memory fake and the default
  (`SMS_ENGINE=fake`, the only engine `config.LoadSMS` takes). No adapter exists: the provider's comes with the chosen
  EU/EEA provider (Q2), once its DPA and processor entry are recorded, with the engine and credentials that turn it
  on. The clinic's paid SMS option cannot be turned on while the engine is the fake (`sms_unavailable`), so no usage
  is counted for messages never sent. Nothing here logs;
  errors never carry a number or a body.
- **`pkg/payments`** (step 0): `Provider.Charge(ctx, Charge{Key, AmountMinor, Currency, Customer}) (Receipt, error)`
  and `Refund(ctx, Refund{Key, ReceiptID, AmountMinor}) error`, `Key` the idempotency key, and `paymentstest` the fake
  (a key reused with another amount, currency or customer is refused, as providers do). No adapter in this epic.
- **Today's booking does not charge, and stays that way.** No price exists anywhere in the model. Both booking paths
  (a direct booking and a waitlist acceptance) go through one step, `appointment` `service_purchase.go`, that asks for
  the booking's price (a `Prices` port, nil in every binary) and charges nothing when there is none, which is always
  until SAL-23 adds prices. With a price it charges before the booking's transaction and refunds in full when the
  booking then fails (built in step 0 and tested with the fake). The idempotency key is the offer's id for a waitlist
  acceptance and, for a direct booking until SAL-23 gives a booking a request id, the patient's and the slot's.
- **SAL-37's seam.** The patient's cancellation and reschedule call `settle` in `service_purchase.go`, a no-op today:
  purchase protection decides there whether a refund is due. No interface is added for it until it has a body.

## Consequences

- `period` and `slot_id` of an appointment can change. Code that keyed something on the start at booking time must
  re-read it: the reminders check the start they were scheduled for; the clinic statistics refresh both days.
- Every status change and confirmation writes an `appointment_change` row: the history the clinic sees and the audit
  of no-shows. AuditEvents stay as they are.
- `slot.status` gains `busy-tentative`; slot generation and retention touch `free` slots only, so they leave an
  offered slot alone; the directory's free-slot search is measured again (SAL-35).
- Each reminder costs one row, one dispatch job and one send job per channel. Measured in SAL-33 against the
  scheduler's sweep and the notifier's queues.
- A link page is the first HTML the api serves. Its strings go through catalogs in the ten patient languages.
- The FHIR golden files of `Appointment` change: the patient participant is `tentative` until confirmed (Q3).

## Alternatives

- **Cancel and rebook with a link** (`supportingInformation`, or R5's `previousAppointment` as a cross-version
  extension): simpler grants, but every cancellation side effect needs an exception and the visit's data would have
  to move between appointments (above).
- **Reminders as River jobs only**, as today: no row to cancel, to count SMS against, or to prove a reminder went
  once; a reschedule could not tell an old job from a new one by status alone.
- **The waitlist hold in Redis** with a TTL: fast expiry for free, but it would be the only copy of the offer, and a
  Redis restart would drop holds while the slot stays hidden or, worse, double-offer it.
- **A slot-level lock without a status** (an offer row only): the public search would have to read offers on every
  query of the directory; the `busy-tentative` status keeps the search as it is.
- **A separate `waitlist` domain**: fewer edits to `appointment`, but CLAUDE.md names domains after the FHIR resource
  they own, and FHIR's waitlist is an `Appointment` status; a new domain would also need CLAUDE.md's lists changed.
- **Signed (HMAC) links without a table**: no storage, but no single use, and no way to expire one when the
  appointment changes.
