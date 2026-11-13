// Package notification sends what the platform tells people (docs/accounts/SPEC.md "Notifier"): the invite
// e-mail, an appointment's confirmation, cancellation and reminder (docs/booking/SPEC.md "E-mail"), and that a
// signed document of an appointment is ready (docs/consultation/SPEC.md "Patient documents"); and the chat's
// notifications (docs/plans/chat.md, SAL-29): a push to the patient's devices, an e-mail when a message stays
// unread, and the clinic's own webhooks, none of which carries a message's text, a sender or a patient's name
// (a webhook carries the text only when the clinic's admin turns clinical content on).
// E-mails are plain text in the reader's language (templates/<language>/, docs/i18n/SPEC.md §3), carry no
// clinical content, and their body is never logged.
package notification

import (
	"time"

	"github.com/Shmeeshmee/salus/pkg/push"
)

// Message is one plain-text e-mail.
type Message struct {
	To      string
	Subject string
	Body    string
}

// SMTPSecurity is how the SMTP sender secures its connection (SMTP_TLS, ADR 0011).
type SMTPSecurity string

// The modes: a relay takes STARTTLS on 587 (required, never opportunistic) or implicit TLS on 465, each with the
// certificate verified against the system roots and the address's host. Plaintext is for the compose Mailpit and
// never carries credentials.
const (
	SMTPStartTLS    SMTPSecurity = "starttls"
	SMTPImplicitTLS SMTPSecurity = "tls"
	SMTPPlaintext   SMTPSecurity = "none"
)

// Caller is who asks: the signed-in account, its clinic for staff (with whether it is the clinic's admin), or none
// for a patient.
type Caller struct {
	AccountID      string
	OrganizationID string
	Admin          bool
}

// Device is an app's push token, registered by the account signed in on the device: personal data, deleted when
// the app unregisters, when the platform says it is gone, and with the account.
type Device struct {
	ID        string
	AccountID string
	Token     string
	Platform  push.Platform
}

// EventMessageCreated is the one webhook event: a new chat message in the clinic.
const EventMessageCreated = "message.created"

// Endpoint is a clinic's webhook: its own system's https URL, the events it takes, and whether a delivery carries
// the message's text (off by default). Its secret is stored sealed and shown once, at creation.
type Endpoint struct {
	CreatedAt       time.Time `json:"created_at"`
	ID              string    `json:"id"`
	OrganizationID  string    `json:"-"`
	CreatedBy       string    `json:"-"`
	URL             string    `json:"url"`
	Events          []string  `json:"events"`
	SecretSealed    []byte    `json:"-"`
	IncludeClinical bool      `json:"include_clinical"`
}

// NewEndpoint is what an admin asks for: the URL, the events and whether to include clinical content.
type NewEndpoint struct {
	URL             string   `json:"url"`
	Events          []string `json:"events"`
	IncludeClinical bool     `json:"include_clinical"`
}

// DeadLetter is a webhook delivery that failed its last attempt: ids, how often it was tried, and the last status
// (0 when the endpoint never answered).
type DeadLetter struct {
	EndpointID     string
	OrganizationID string
	EventID        string
	EventType      string
	ConversationID string
	MessageID      string
	Attempts       int
	LastStatus     int
}

// ChatPushArgs is the chat_push job: one notification to one device of the account. Opaque ids only; the account's
// so that a token moved to another account meanwhile receives nothing.
type ChatPushArgs struct {
	DeviceID  string `json:"device_id"`
	AccountID string `json:"account_id"`
}

// Kind names the job in the river_job table.
func (ChatPushArgs) Kind() string { return "chat_push" }

// ChatUnreadEmailArgs is the chat_unread_email job, scheduled the clinic's e-mail delay after a message: it e-mails
// the participant if they have not read up to Seq by then, and checks again if they have but not to the
// conversation's end (Chat.UnreadEmail). Its unique key is the participant and their read position
// when the message came (ReadSeq): a burst of messages to someone who has not read on gives one check and so one
// e-mail, and River keeps a run check for a day, so one e-mail a day at most while they read nothing; a message
// after they read on gets a check of its own. Ids and numbers only.
type ChatUnreadEmailArgs struct {
	ParticipantID  string `json:"participant_id" river:"unique"`
	ConversationID string `json:"conversation_id"`
	OrganizationID string `json:"organization_id"`
	ReadSeq        int64  `json:"read_seq" river:"unique"`
	Seq            int64  `json:"seq"`
}

// Kind names the job in the river_job table.
func (ChatUnreadEmailArgs) Kind() string { return "chat_unread_email" }

// WebhookDeliveryArgs is the webhook_delivery job: one event to one endpoint, retried with backoff and dead-lettered
// after its last attempt. Ids only; the event's id is the same on every attempt (the receiver's idempotency key).
type WebhookDeliveryArgs struct {
	EndpointID     string `json:"endpoint_id" river:"unique"`
	EventID        string `json:"event_id" river:"unique"`
	OrganizationID string `json:"organization_id"`
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
}

// Kind names the job in the river_job table.
func (WebhookDeliveryArgs) Kind() string { return "webhook_delivery" }

// SAL-33: the deliveries of reminders and waitlist offers (docs/plans/appointments.md, ADR 0015 "Reminders").

// Kind is what a delivery tells: a reminder (appointment_reminder) or a waitlist offer (SAL-35).
type Kind string

// The kinds of delivery (notification_delivery.kind).
const (
	KindReminder      Kind = "reminder"
	KindWaitlistOffer Kind = "waitlist_offer"
)

// Channel is how a delivery goes out.
type Channel string

// The channels (notification_delivery.channel).
const (
	ChannelPush  Channel = "push"
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
)

// DeliveryState is where a delivery is: pending until its send job sends it (sent), finds it no longer applies
// (cancelled) or fails for good (failed). It leaves pending once.
type DeliveryState string

// The delivery states (notification_delivery.state).
const (
	DeliveryPending   DeliveryState = "pending"
	DeliverySent      DeliveryState = "sent"
	DeliveryCancelled DeliveryState = "cancelled"
	DeliveryFailed    DeliveryState = "failed"
)

// Delivery is one message of a reminder or an offer through one channel (and, for a push, to one device): ids only, no
// address, number, token or text.
type Delivery struct {
	CreatedAt      time.Time
	SentAt         time.Time
	ID             string
	SourceID       string
	OrganizationID string
	AccountID      string
	DeviceID       string // a push's device; "" otherwise
	Kind           Kind
	Channel        Channel
	State          DeliveryState
	Segments       int // an SMS's parts, once sent
}

// Content is what Deliver needs to know of the message: Deadline, the instant it is about (an appointment's start, an
// offer's expiry), before which a push or SMS held by quiet hours must still go out by at least 30 minutes.
type Content struct {
	Deadline time.Time
}

// QuietHours is a stretch of the day in the patient's zone, HH:MM to HH:MM, that may wrap midnight (22:00 to 08:00).
type QuietHours struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Channels are the patient's choice of how reminders and offers reach them.
type Channels struct {
	Push  bool `json:"push"`
	Email bool `json:"email"`
	SMS   bool `json:"sms"`
}

// Preferences are the patient's notification settings (NotificationSettings): their channels, their quiet hours (nil
// for none) and their device's IANA zone ("": the clinic's is used). Personal data: the zone can show they are abroad.
type Preferences struct {
	Quiet    *QuietHours `json:"quiet_hours,omitempty"`
	TimeZone string      `json:"time_zone,omitempty"`
	Channels Channels    `json:"channels"`
}

// DefaultPreferences are a patient's settings until they change them: push and e-mail on, SMS off, quiet hours 22:00
// to 08:00, no zone.
func DefaultPreferences() Preferences {
	return Preferences{Channels: Channels{Push: true, Email: true}, Quiet: &QuietHours{Start: "22:00", End: "08:00"}}
}

// UsageMonth is a clinic's SMS usage in one calendar month (UTC): messages sent and their parts.
type UsageMonth struct {
	Month    string `json:"month"` // YYYY-MM
	Messages int    `json:"messages"`
	Segments int    `json:"segments"`
}

// DeliveryArgs is a delivery's send job: reminder_email, reminder_push or reminder_sms by its channel (on the email,
// push and sms queues), for a reminder's or an offer's delivery alike. The delivery's id only.
type DeliveryArgs struct {
	DeliveryID string `json:"delivery_id" river:"unique"`
}

// EmailDeliveryArgs, PushDeliveryArgs and SMSDeliveryArgs are DeliveryArgs under each channel's job kind.
type (
	EmailDeliveryArgs DeliveryArgs
	PushDeliveryArgs  DeliveryArgs
	SMSDeliveryArgs   DeliveryArgs
)

// Kind names the job in river_job.
func (EmailDeliveryArgs) Kind() string { return "reminder_email" }

// Kind names the job in river_job.
func (PushDeliveryArgs) Kind() string { return "reminder_push" }

// Kind names the job in river_job.
func (SMSDeliveryArgs) Kind() string { return "reminder_sms" }
