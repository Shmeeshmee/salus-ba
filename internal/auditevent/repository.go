package auditevent

import "context"

// Repository stores audit events. It only ever inserts: audit_event is append-only.
type Repository interface {
	Insert(ctx context.Context, e AuditEvent) error
}
