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
