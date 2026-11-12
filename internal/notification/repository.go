package notification

import (
	"context"
	"time"
)

// Sender delivers e-mail: SMTP, to Mailpit locally and through an EU relay in production (ADR 0011).
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// Repository stores the accounts' push devices and the clinics' webhook endpoints and dead letters (migration
// 00043). The devices are not row-level secured (like the accounts): every device method names the account or the
// device. The endpoints and dead letters are seen only in the clinic set by SetClinic.
type Repository interface {
	// RegisterDevice stores the token as the account's on platform, moving it from any other account.
	RegisterDevice(ctx context.Context, d Device, at time.Time) error
	// UnregisterDevice deletes the account's device of the token, if there is one.
	UnregisterDevice(ctx context.Context, accountID, token string) error
	// Devices returns the account's devices.
	Devices(ctx context.Context, accountID string) ([]Device, error)
	// Device returns the device, or ErrNotFound.
	Device(ctx context.Context, id string) (Device, error)
	// DeleteDevice deletes the device if it still holds token (the platform said it is gone).
	DeleteDevice(ctx context.Context, id, token string) error

	// SetClinic scopes the rest of the transaction to the clinic id (app.organization_id).
	SetClinic(ctx context.Context, id string) error
	// CreateEndpoint stores e (with its id) and returns it with its creation time.
	CreateEndpoint(ctx context.Context, e Endpoint) (Endpoint, error)
	// Endpoints returns the clinic's endpoints, oldest first.
	Endpoints(ctx context.Context, org string) ([]Endpoint, error)
	// Endpoint returns the endpoint, or ErrNotFound.
	Endpoint(ctx context.Context, id string) (Endpoint, error)
	// DeleteEndpoint deletes the endpoint and its dead letters, or returns ErrNotFound.
	DeleteEndpoint(ctx context.Context, id string) error
	// AddDeadLetter records a delivery that failed its last attempt, once per endpoint and event.
	AddDeadLetter(ctx context.Context, d DeadLetter) error
	// DeleteDeadLettersBefore deletes the dead letters of every clinic recorded before before, and counts them.
	// Only the retention role may: it bypasses row-level security.
	DeleteDeadLettersBefore(ctx context.Context, before time.Time) (int64, error)
}

// Jobs enqueues the chat's notifications in the transaction in the context (River's InsertTx).
type Jobs interface {
	EnqueuePush(ctx context.Context, a ChatPushArgs) error
	// EnqueueUnreadEmail schedules the check at at; one of the same participant and read position, waiting or run
	// within a day, is kept instead.
	EnqueueUnreadEmail(ctx context.Context, a ChatUnreadEmailArgs, at time.Time) error
	EnqueueWebhook(ctx context.Context, a WebhookDeliveryArgs) error
}

// Poster posts a webhook's body to its URL with the headers, and returns the answer's status. It refuses to connect
// to an address that is not public (NewWebhookPoster).
type Poster interface {
	Post(ctx context.Context, url string, body []byte, header map[string]string) (int, error)
}

// SecretBox seals the webhook secrets at rest, bound to their endpoint's id (auth.AESSecretBox on WEBHOOK_KEY, a key
// of its own: ADR 0011's root-only key file in production).
type SecretBox interface {
	Seal(plaintext []byte, id string) ([]byte, error)
	Open(sealed []byte, id string) ([]byte, error)
}
