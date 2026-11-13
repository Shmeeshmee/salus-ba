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
