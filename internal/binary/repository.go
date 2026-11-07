package binary

import (
	"context"
	"io"
	"time"
)

// Store holds the files' encrypted bytes in the object store, one object per file, keyed by the binary id
// (docs/attachments/SPEC.md §2 "Upload pipeline", "Download"). It never sees a plaintext byte, a title or a
// content type. Calls to it stay outside every transaction.
type Store interface {
	// Put stores exactly size bytes read from r under key, streamed in constant memory: r need not be
	// seekable. A body that ends before size bytes fails and leaves no object.
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	// GetRange returns the bytes from offset from to offset to of the object at key, both inclusive (one
	// HTTP range), or ErrObjectNotFound. The caller closes the reader.
	GetRange(ctx context.Context, key string, from, to int64) (io.ReadCloser, error)
	// Delete removes the object at key; a key that holds no object is not an error.
	Delete(ctx context.Context, key string) error
	// ListOlderThan calls fn once per page of the listing with the keys of that page's objects last modified
	// before before (a page with none is skipped), and stops at fn's first error, which it returns.
	ListOlderThan(ctx context.Context, before time.Time, fn func(keys []string) error) error
}

// Repository stores the files' rows. Row-level security shows a transaction the files of the patient set by
// SetPatient; of the clinic set by SetClinic those whose questionnaire response the clinic sees; and of the
// job scope set by SetJob the files of that one response.
type Repository interface {
	// SetPatient scopes the rest of the transaction in ctx to the patient id.
	SetPatient(ctx context.Context, id string) error
	// SetClinic scopes the rest of the transaction in ctx to the clinic id.
	SetClinic(ctx context.Context, id string) error
	// SetJob scopes the rest of the transaction in ctx to the files of the response responseID of the clinic
	// org, for binary's jobs.
	SetJob(ctx context.Context, responseID, org string) error
	// LockResponse serialises the uploads to the response id until the transaction ends (an advisory lock),
	// so two uploads cannot together pass a limit.
	LockResponse(ctx context.Context, id string) error
	// Insert stores b, quarantined, at at, and returns it.
	Insert(ctx context.Context, b Binary, at time.Time) (Binary, error)
	// ByID returns the file, or ErrNotFound (also outside the scope, or for an id that is not a UUID).
	ByID(ctx context.Context, id string) (Binary, error)
	// OfItem returns the files of the response's item linkID that are not rejected, oldest first.
	OfItem(ctx context.Context, responseID, linkID string) ([]Binary, error)
	// ResponseSize is the plaintext bytes of the response's files that are not rejected.
	ResponseSize(ctx context.Context, responseID string) (int64, error)
	// Delete deletes the file id that the scope lets it, and reports whether it did.
	Delete(ctx context.Context, id string) (bool, error)
	// MarkScanned sets the scan's result on the file id while it is quarantined, at at: clean, or rejected with
	// its wrapped key cleared; it reports whether it did (the job scope only).
	MarkScanned(ctx context.Context, id string, status Status, at time.Time) (bool, error)
	// OfResponse returns every file of the response, whatever its status, oldest first.
	OfResponse(ctx context.Context, responseID string) ([]Binary, error)
	// DeleteRejectedBefore deletes the rejected files scanned before before, in every clinic, and counts them
	// (the retention role's purger).
	DeleteRejectedBefore(ctx context.Context, before time.Time) (int64, error)
	// Existing returns the ids among ids that have a row (the retention role's purger); an id that is no UUID
	// has none.
	Existing(ctx context.Context, ids []string) ([]string, error)
}

// Jobs enqueues binary's jobs in the transaction in the context (River's InsertTx).
type Jobs interface {
	EnqueueScan(ctx context.Context, args ScanArgs) error
}
