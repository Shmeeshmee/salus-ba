// Package notification sends what the platform tells people (docs/accounts/SPEC.md "Notifier"): for now the
// invite e-mail. E-mails are plain text, carry no clinical content, and their body is never logged.
package notification

// Message is one plain-text e-mail.
type Message struct {
	To      string
	Subject string
	Body    string
}
