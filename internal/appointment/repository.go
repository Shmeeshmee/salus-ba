package appointment

import (
	"context"
	"time"
)

// Repository stores appointments. Row-level security shows a transaction the appointments of the patient set
// by SetPatient; of the clinic set by SetClinic, those with an active consent and, after a revocation, those
// that started before it; and, with SetNotice beside SetClinic, the one appointment of a job.
type Repository interface {
	// SetPatient scopes the rest of the transaction in ctx to the patient id.
	SetPatient(ctx context.Context, id string) error
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// SetNotice adds the appointment id of an e-mail job to the clinic scope.
	SetNotice(ctx context.Context, id string) error
	// Create stores a booked appointment, created at a.Created, and returns it; ErrSlotUnavailable when the
	// role has an appointment at an overlapping time, or another is being booked at one at the same moment
	// (either aborts the transaction).
	Create(ctx context.Context, a Appointment) (Appointment, error)
	// ByID returns the appointment, or ErrNotFound (also outside the scope, or for an id that is not a UUID).
	ByID(ctx context.Context, id string) (Appointment, error)
	// ByIDForShare is ByID, locking the appointment against a change until the transaction ends (FOR SHARE: a
	// concurrent cancellation waits, and one that committed first is seen). The scope must also let it change
	// the appointment: the patient's own, or the clinic's with the patient's active consent.
	ByIDForShare(ctx context.Context, id string) (Appointment, error)
	// Search returns the appointments the filter and the scope give, by start.
	Search(ctx context.Context, f Filter) ([]Appointment, error)
	// Cancel cancels the booked appointment id at at for reason, by the account by (empty for a job), and
	// returns it at its next version; ErrClosed when it is not booked.
	Cancel(ctx context.Context, id string, reason Reason, by string, at time.Time) (Appointment, error)
	// SetStatus moves the booked appointment id to status at at and returns it at its next version;
	// ErrClosed when it is not booked.
	SetStatus(ctx context.Context, id string, status Status, at time.Time) (Appointment, error)
	// AddChange appends c to the history of its appointment, with the appointment's clinic and patient, and returns
	// its id; ErrNotFound when the scope does not show the appointment.
	AddChange(ctx context.Context, c Change) (string, error)
	// Changes returns the history of the appointment id, oldest first, as the scope shows it.
	Changes(ctx context.Context, id string) ([]Change, error)
}

// TokenRepository stores the one-time links of reminders (appointment_action_token, ADR 0015): only their hash.
// InsertToken runs in a notifier job's scope (SetClinic and SetNotice of Repository, on the same transaction); the
// retention role's purges every clinic's.
type TokenRepository interface {
	// InsertToken stores the link t.
	InsertToken(ctx context.Context, t Token) error
	// DeleteTokensExpiredBefore deletes the links that expired before before, and counts them.
	DeleteTokensExpiredBefore(ctx context.Context, before time.Time) (int64, error)
}

// StatsRepository reads the appointments for the aggregate builds, on salus_stats.
type StatsRepository interface {
	// Booked returns a clinic's appointments starting in [from, to), every status, oldest first.
	Booked(ctx context.Context, orgID string, from, to time.Time) ([]Booked, error)
	// Fulfilled returns a clinic's fulfilled appointments starting in [from, to), oldest first.
	Fulfilled(ctx context.Context, orgID string, from, to time.Time) ([]Booked, error)
}

// SelfServiceRepository confirms and moves a patient's appointments and uses the one-time links of their reminders
// (SAL-34), in the patient's scope set by Repository's SetPatient on the same transaction; only FindToken needs none.
type SelfServiceRepository interface {
	// ByIDForUpdate is ByID, locking the appointment until the transaction ends (FOR UPDATE: a concurrent reschedule
	// or cancellation waits, and one that committed first is seen).
	ByIDForUpdate(ctx context.Context, id string) (Appointment, error)
	// Confirm records the patient's confirmation of the booked, unconfirmed appointment id at at, made via, and
	// returns it at its next version; ErrClosed when it is not booked or is confirmed already.
	Confirm(ctx context.Context, id string, via Via, at time.Time) (Appointment, error)
	// Reschedule moves the booked appointment id to m's slot and period at at, counting the reschedule and confirming
	// it in the app when it was not (choosing the time confirms it), and returns it at its next version; ErrClosed
	// when it is no longer booked or no longer at m.From reschedules; ErrSlotUnavailable when the role has an
	// appointment at an overlapping time (which aborts the transaction).
	Reschedule(ctx context.Context, id string, m Move, at time.Time) (Appointment, error)
	// FindToken returns the ids of the link whose token hashes to hash (ID, AppointmentID, OrganizationID, PatientID,
	// AccountID), whatever the scope; ErrLinkInvalid for none.
	FindToken(ctx context.Context, hash [32]byte) (Token, error)
	// TokenByID returns the link id as the patient's scope shows it; ErrLinkInvalid when it does not.
	TokenByID(ctx context.Context, id string) (Token, error)
	// UseToken marks the unused link id used for action at at; ErrLinkInvalid when it was used already (by a
	// concurrent use that committed first too).
	UseToken(ctx context.Context, id string, action LinkAction, at time.Time) error
}

// LinkLimiter counts the link lookups of an address (Redis, every key with a TTL).
type LinkLimiter interface {
	// Allow counts one lookup under key and reports whether it is within limit lookups in the window that the first
	// one started.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// ReminderRepository stores the reminders of appointments (appointment_reminder, SAL-33), in the scope Repository's
// SetPatient or SetClinic set on the same transaction: the patient's own, or their clinic's.
type ReminderRepository interface {
	// AddReminder stores r, scheduled, and returns it with its id; ok is false when the appointment already has a live
	// reminder (scheduled or dispatched) of that offset for that start (nothing is stored).
	AddReminder(ctx context.Context, r Reminder) (out Reminder, ok bool, err error)
	// CancelReminders cancels the live reminders (scheduled or dispatched) of the appointment id and counts them.
	CancelReminders(ctx context.Context, appointmentID string) (int64, error)
	// LockReminder returns the reminder id, locked against a change until the transaction ends, or ErrNotFound.
	LockReminder(ctx context.Context, id string) (Reminder, error)
	// Reminder returns the reminder id, or ErrNotFound.
	Reminder(ctx context.Context, id string) (Reminder, error)
	// SetReminderState moves the scheduled reminder id to state at at (its dispatch time, for dispatched).
	SetReminderState(ctx context.Context, id string, state ReminderState, at time.Time) error
	// OverdueReminders returns, of every clinic and whatever the scope, up to max reminders still scheduled at at
	// although due, whose start is ahead of at, by due time and id after (afterDue, afterID): ids, clinic and due time.
	OverdueReminders(ctx context.Context, at, afterDue time.Time, afterID string, max int) ([]Reminder, error)
	// DeleteRemindersDueBefore deletes the reminders of every clinic due before before, and counts them; only the
	// retention role may.
	DeleteRemindersDueBefore(ctx context.Context, before time.Time) (int64, error)
}

// WaitlistRepository stores the waitlist (SAL-35): waitlist_entry and waitlist_offer. Row-level security shows the
// patient's own (SetPatient), and the clinic's (SetClinic) entries with the patient's active care consent and every
// offer of the clinic.
type WaitlistRepository interface {
	// InsertEntry stores the active entry e, created at e.Created, and returns it.
	InsertEntry(ctx context.Context, e WaitlistEntry) (WaitlistEntry, error)
	// CountActiveEntries counts the patient's active entries with the clinic that end after now.
	CountActiveEntries(ctx context.Context, patientID, org string, now time.Time) (int, error)
	// PatientEntries returns the patient's entries, newest first, each with its open offer.
	PatientEntries(ctx context.Context, patientID string) ([]WaitlistEntry, error)
	// ClinicEntries returns a page of the clinic's entries in offer order (priority descending, then created_at), each
	// with its open offer; with q's role, those of the role or of its specialty; with q's status, those in it.
	ClinicEntries(ctx context.Context, q ClinicQuery) ([]WaitlistEntry, error)
	// LockEntries holds the patient's entries with the clinic until the transaction ends: joins run one at a time.
	LockEntries(ctx context.Context, patientID, org string) error
	// EntryForUpdate returns the entry id locked until the transaction ends, or ErrNotFound.
	EntryForUpdate(ctx context.Context, id string) (WaitlistEntry, error)
	// CloseEntry closes the active entry id with status at at and returns it; ErrNotFound when it is not active.
	CloseEntry(ctx context.Context, id string, status WaitlistStatus, at time.Time) (WaitlistEntry, error)
	// SetPriority sets the priority of the active entry id at at and returns it; ErrNotFound when it is not active.
	SetPriority(ctx context.Context, id string, priority int, at time.Time) (WaitlistEntry, error)
	// ActiveEntries returns the patient's active entries with the clinic.
	ActiveEntries(ctx context.Context, patientID, org string) ([]WaitlistEntry, error)
	// Candidates returns, locked, at most n active entries that m matches, in offer order (those another transaction
	// holds are passed over): of m's clinic, role or specialty,
	// day and times, never offered m's slot, without an open offer, of a patient who never held the slot (its
	// cancelled or rescheduled appointment) and has no booked appointment overlapping it that the scope shows.
	Candidates(ctx context.Context, m Match, n int) ([]WaitlistEntry, error)
	// InsertOffer stores the open offer o and returns it.
	InsertOffer(ctx context.Context, o Offer) (Offer, error)
	// Offer returns the offer id, or ErrNotFound.
	Offer(ctx context.Context, id string) (Offer, error)
	// OfferForUpdate returns the offer id locked until the transaction ends, or ErrNotFound: acceptance and expiry
	// serialise on it.
	OfferForUpdate(ctx context.Context, id string) (Offer, error)
	// OpenOffer returns the open offer of the slot (or, with entry set, of the entry) and whether there is one.
	OpenOffer(ctx context.Context, slotID, entryID string) (Offer, bool, error)
	// CloseOffer closes the open offer id with state at at (with the appointment an acceptance booked) and returns it;
	// ErrNotFound when it is not open.
	CloseOffer(ctx context.Context, id string, state OfferState, appointmentID string, at time.Time) (Offer, error)
	// DeleteEntriesClosedBefore deletes the entries that closed, or ended while active, before before, with their
	// offers (the retention role's), and counts them.
	DeleteEntriesClosedBefore(ctx context.Context, before time.Time) (int64, error)
	// DeleteOffersClosedBefore deletes the offers of those entries, and counts them.
	DeleteOffersClosedBefore(ctx context.Context, before time.Time) (int64, error)
}

// NoShowRepository corrects how an appointment went (SAL-36), in the transaction and scope in ctx.
type NoShowRepository interface {
	// CorrectStatus moves the appointment id from the status from to to (noshow and fulfilled into each other) at at
	// and returns it at its next version; ErrClosed when it is not in from (a concurrent change committed first).
	CorrectStatus(ctx context.Context, id string, from, to Status, at time.Time) (Appointment, error)
	// NoShowsSince counts, as the scope shows them, each patient's noshow appointments at the clinic org that
	// started in [since, until]; a patient without any is absent, an id that is not a UUID matches nothing.
	NoShowsSince(ctx context.Context, org string, patientIDs []string, since, until time.Time) (map[string]int, error)
}
