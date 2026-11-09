package transcript

import "context"

// Publisher puts live events on an encounter's Redis Stream (repository_redis.go). Errors carry no event.
type Publisher interface {
	// Publish appends the envelope to the encounter's stream (StreamKey), capped at StreamMaxLen entries and
	// expiring StreamTTL after its last entry.
	Publish(ctx context.Context, encounterID string, envelope []byte) error
}
