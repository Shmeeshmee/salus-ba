package transcript

import (
	"context"
	"time"
)

// Repository stores the final segments (transcript_segment). Row-level security shows a transaction the
// segments of the encounters its clinic's scope shows; the encounter's service sets the scope.
type Repository interface {
	// Insert stores the final segment e under its SegmentID unless a segment with that id exists already, and
	// reports whether this call stored it: a segment delivered twice is stored once.
	Insert(ctx context.Context, e Event) (bool, error)
	// ByEncounter returns the encounter's stored segments in the order they were said, as final events.
	ByEncounter(ctx context.Context, encounterID string) ([]Event, error)
}

// Stream reads an encounter's Redis Stream (StreamKey).
type Stream interface {
	// Read returns the entries strictly after the entry id after ("0" for every entry), oldest first and at
	// most a page of them, waiting up to wait for one when there is none yet; none when wait passes or the
	// stream does not exist.
	Read(ctx context.Context, encounterID, after string, wait time.Duration) ([]Entry, error)
}

// Publisher puts live events on an encounter's Redis Stream (repository_redis.go). Errors carry no event.
type Publisher interface {
	// Publish appends the envelope to the encounter's stream (StreamKey) under StreamField, capped at about
	// StreamMaxLen entries and expiring StreamTTL after its last entry.
	Publish(ctx context.Context, encounterID string, envelope []byte) error
}
