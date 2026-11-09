package questionnaireresponse

import (
	"context"
	"time"

	"github.com/Shmeeshmee/salus/internal/binary"
)

// Repository stores the responses. Row-level security shows a transaction the responses of the patient set
// by SetPatient, and of the clinic set by SetClinic those of its patients with an active consent and, after a
// revocation, the completed ones of appointments that started before it; with SetNotice beside SetClinic, it
// deletes those of that one appointment once it is cancelled.
type Repository interface {
	// SetPatient scopes the rest of the transaction in ctx to the patient id.
	SetPatient(ctx context.Context, id string) error
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// SetNotice adds the appointment id of a job to the clinic scope: the discard job deletes its responses.
	SetNotice(ctx context.Context, id string) error
	// Insert stores r in progress at at, with no answers, unless its appointment has a live request of its
	// series, and returns it and whether this call created it.
	Insert(ctx context.Context, r Response, at time.Time) (Response, bool, error)
	// ByID returns the response, or ErrNotFound (also outside the scope, or for an id that is not a UUID).
	ByID(ctx context.Context, id string) (Response, error)
	// Lock is ByID, locking the response until the transaction ends.
	Lock(ctx context.Context, id string) (Response, error)
	// Stop withdraws the response at at: stopped, its answers cleared.
	Stop(ctx context.Context, id string, at time.Time) (Response, error)
	// SetLanguage sets the language the patient answers the in-progress response id in, at at, moving it to its
	// next version; or returns ErrNotFound when there is no such response in progress with nothing answered.
	SetLanguage(ctx context.Context, id, language string, at time.Time) (Response, error)
	// SaveAnswers replaces the answers of the in-progress response id at at, moving it to its next version,
	// or returns ErrNotFound when there is no such response in progress.
	SaveAnswers(ctx context.Context, id string, answers []Answer, at time.Time) (Response, error)
	// Complete submits the in-progress response id with answers at at: completed, authored at at, its next
	// version; or returns ErrNotFound when there is no such response in progress.
	Complete(ctx context.Context, id string, answers []Answer, at time.Time) (Response, error)
	// Search returns the responses of f that the scope shows, by due time.
	Search(ctx context.Context, f Filter) ([]Response, error)
	// DeleteOfAppointment deletes the responses of the appointment id that the scope lets it (the clinic's with
	// the appointment's notice, once it is cancelled), submitted or not, and returns their ids; none for an id
	// that is not a UUID.
	DeleteOfAppointment(ctx context.Context, id string) ([]string, error)
	// DeleteUnsubmittedBefore deletes, in every clinic, the responses not completed whose due time is
	// before before, and returns how many (the retention purger, on salus_retention's pool).
	DeleteUnsubmittedBefore(ctx context.Context, before time.Time) (int64, error)
	// AddAttachment appends a to the attachments of the response id at at (updated_at, not version_id), or
	// returns ErrNotFound when the scope shows no such response.
	AddAttachment(ctx context.Context, id string, a binary.Attachment, at time.Time) error
	// RemoveAttachment removes the file binaryID from the attachments of the response id at at; none to remove
	// changes nothing.
	RemoveAttachment(ctx context.Context, id, binaryID string, at time.Time) error
	// ClearAttachment marks the file binaryID of the response id no longer pending at at; a file not pending
	// changes nothing.
	ClearAttachment(ctx context.Context, id, binaryID string, at time.Time) error
	// Attachments returns the attachments of the response id, or ErrNotFound.
	Attachments(ctx context.Context, id string) ([]binary.Attachment, error)
}

// Jobs enqueues the background work a change of the answers causes, in the caller's transaction (River's
// InsertTx): binary_discard, once the response no longer lists some of its files.
type Jobs interface {
	EnqueueDiscard(ctx context.Context, args binary.DiscardArgs) error
}
