package notification

import "context"

// Sender delivers e-mail: SMTP (Mailpit locally) now, SES once its DPA is signed (docs/privacy/processors.md).
type Sender interface {
	Send(ctx context.Context, m Message) error
}
