package binary

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrObjectNotFound is a Store's answer for a key that holds no object.
var ErrObjectNotFound = errors.New("binary: object not found")

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
