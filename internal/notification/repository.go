package notification

import "context"

// Sender delivers e-mail: SMTP, to Mailpit locally and through an EU relay in production (ADR 0011).
type Sender interface {
	Send(ctx context.Context, m Message) error
}
