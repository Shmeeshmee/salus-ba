package clinicexport

import (
	"context"
	"io"
	"time"
)

// Repository stores the exports. Row-level security shows a transaction the exports of the clinic set by
// SetOrganization.
type Repository interface {
	// SetOrganization scopes the rest of the transaction in ctx to the clinic id.
	SetOrganization(ctx context.Context, id string) error
	// Insert stores e queued and returns it, or reports false when the clinic already has a queued export.
	Insert(ctx context.Context, e Export) (Export, bool, error)
	// Queued returns the clinic's queued export, or ErrNotFound.
	Queued(ctx context.Context, org string) (Export, error)
	// ByID returns the export, or ErrNotFound (also outside the scope, or for an id that is not a UUID).
	ByID(ctx context.Context, id string) (Export, error)
	// MarkReady sets the queued export e.ID ready with e's ReadyAt, Size, KEKID and WrappedKey, and reports whether it did.
	MarkReady(ctx context.Context, e Export) (bool, error)
	// MarkFailed sets the queued export id failed and reports whether it did.
	MarkFailed(ctx context.Context, id string) (bool, error)
	// RequestedBefore returns up to max ids of the exports asked for before before, in every clinic (salus_retention).
	RequestedBefore(ctx context.Context, before time.Time, max int32) ([]string, error)
	// Delete deletes the export id and reports whether it did (salus_retention).
	Delete(ctx context.Context, id string) (bool, error)
}

// Store holds the archives' encrypted bytes in the object store (binary.Store, ADR 0010), under ObjectKey. It never
// sees a plaintext byte. Calls to it stay outside every transaction.
type Store interface {
	// Put stores exactly size bytes read from r under key.
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	// GetRange returns the bytes from offset from to offset to of the object at key, both inclusive.
	GetRange(ctx context.Context, key string, from, to int64) (io.ReadCloser, error)
	// Delete removes the object at key; a key that holds no object is not an error.
	Delete(ctx context.Context, key string) error
}

// Jobs enqueues the export's build in the transaction in ctx (River's InsertTx).
type Jobs interface {
	EnqueueBuild(ctx context.Context, args BuildArgs) error
}
