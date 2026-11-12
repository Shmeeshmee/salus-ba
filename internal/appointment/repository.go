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
}

// StatsRepository reads the appointments for the aggregate builds, on salus_stats.
type StatsRepository interface {
	// Booked returns a clinic's appointments starting in [from, to), every status, oldest first.
	Booked(ctx context.Context, orgID string, from, to time.Time) ([]Booked, error)
}
