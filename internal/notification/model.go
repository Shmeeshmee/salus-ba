// Package notification sends what the platform tells people (docs/accounts/SPEC.md "Notifier"): the invite
// e-mail, an appointment's confirmation, cancellation and reminder (docs/booking/SPEC.md "E-mail"), and that a
// signed document of an appointment is ready (docs/consultation/SPEC.md "Patient documents").
// E-mails are plain text in the reader's language (templates/<language>/, docs/i18n/SPEC.md §3), carry no
// clinical content, and their body is never logged.
package notification

// Message is one plain-text e-mail.
type Message struct {
	To      string
	Subject string
	Body    string
}
