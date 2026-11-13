package auditevent

import (
	"context"
	"time"
)

// Repository stores audit events. It only ever inserts: audit_event is append-only.
type Repository interface {
	Insert(ctx context.Context, e AuditEvent) error
	// InsertAll stores the events in one statement.
	InsertAll(ctx context.Context, es []AuditEvent) error
	// CountRecordedBefore counts the events recorded before before.
	CountRecordedBefore(ctx context.Context, before time.Time) (int64, error)
}
