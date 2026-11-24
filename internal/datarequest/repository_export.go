package datarequest

import (
	"context"
	"io"
	"time"
)

// ExportRepository stores the patients' exports (data_request_export, package D2). Row-level security shows a
// transaction the exports of the account set by SetAccount; the retention job (salus_retention) sees them all.
type ExportRepository interface {
	// SetAccount sets the patient's scope (their account) on the transaction in ctx.
	SetAccount(ctx context.Context, accountID string) error
	// InsertExport stores x queued, or reports false when the account already has a queued export.
	InsertExport(ctx context.Context, x StoredExport) (bool, error)
	// ExportOf returns the export of the request id with its request's kind and target, or ErrNotFound.
	ExportOf(ctx context.Context, requestID string) (StoredExport, error)
	// MarkExportReady sets the queued export x.RequestID ready with x's ReadyAt, ExpiresAt, Size, KEKID and WrappedKey
	// while its request is still open, and reports whether it did.
	MarkExportReady(ctx context.Context, x StoredExport) (bool, error)
	// MarkExportFailed sets the queued export of the request id failed and reports whether it did.
	MarkExportFailed(ctx context.Context, requestID string) (bool, error)
	// Fulfil answers the scope's open request id done once its export is ready, with its history entry by the job, and
	// reports whether it did.
	Fulfil(ctx context.Context, requestID string) (bool, error)
	// ExportsDoneBefore returns up to max request ids of the exports ready before before, or asked for before it and
	// never ready (salus_retention).
	ExportsDoneBefore(ctx context.Context, before time.Time, max int32) ([]string, error)
	// DeleteExport deletes the export of the request id and reports whether it did (salus_retention).
	DeleteExport(ctx context.Context, requestID string) (bool, error)
}

// Store holds the archives' encrypted bytes in the object store (binary.Store, ADR 0010), under ExportObjectKey. It
// never sees a plaintext byte. Calls to it stay outside every transaction.
type Store interface {
	// Put stores exactly size bytes read from r under key.
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	// GetRange returns the bytes from offset from to offset to of the object at key, both inclusive.
	GetRange(ctx context.Context, key string, from, to int64) (io.ReadCloser, error)
	// Delete removes the object at key; a key that holds no object is not an error.
	Delete(ctx context.Context, key string) error
}

// ExportJobs enqueues the export's build in the transaction in ctx (River's InsertTx).
type ExportJobs interface {
	EnqueueExport(ctx context.Context, args ExportArgs) error
}
