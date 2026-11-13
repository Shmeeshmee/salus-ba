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

// ReminderRepository stores the reminders of appointments (appointment_reminder, SAL-33), in the scope Repository's
// SetPatient or SetClinic set on the same transaction: the patient's own, or their clinic's.
type ReminderRepository interface {
	// AddReminder stores r, scheduled, and returns it with its id; ok is false when the appointment already has a
	// reminder of that offset for that start (nothing is stored).
	AddReminder(ctx context.Context, r Reminder) (out Reminder, ok bool, err error)
	// CancelReminders cancels the scheduled reminders of the appointment id and counts them.
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
