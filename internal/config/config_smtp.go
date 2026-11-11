package config

import (
	"errors"
	"net"
	"net/mail"
	"os"
)

// SMTPTLS is how the notifier secures its connection to the SMTP server (SMTP_TLS).
type SMTPTLS string

// The modes. A relay (Brevo, Mailjet: ADR 0011) takes STARTTLS on 587 or implicit TLS on 465; plaintext is for
// the compose Mailpit, and never carries credentials.
const (
	SMTPStartTLS    SMTPTLS = "starttls"
	SMTPImplicitTLS SMTPTLS = "tls"
	SMTPPlaintext   SMTPTLS = "none"
)

// SMTP holds the notifier's SMTP server settings (ADR 0011 "E-mail through an EU SMTP relay"). Load never reads
// them; only cmd/notifier calls LoadSMTP. The password comes from the secret store, never from a committed file.
type SMTP struct {
	From     mail.Address // SMTP_FROM, one RFC 5322 address: the From header and the envelope sender
	Addr     string       // SMTP_ADDR, host:port; the host is also the name the certificate must carry
	TLS      SMTPTLS      // SMTP_TLS: starttls, tls or none
	Username string       // SMTP_USERNAME, with SMTP_PASSWORD or not at all (AUTH PLAIN)
	Password string       // SMTP_PASSWORD
}

// LoadSMTP reads the SMTP settings from the environment. Errors name the variable, never a value.
func LoadSMTP() (SMTP, error) {
	s := SMTP{
		Addr:     os.Getenv("SMTP_ADDR"),
		TLS:      SMTPTLS(os.Getenv("SMTP_TLS")),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
	}
	if host, port, err := net.SplitHostPort(s.Addr); err != nil || host == "" || port == "" {
		return SMTP{}, errors.New("config: SMTP_ADDR is not host:port")
	}
	switch s.TLS {
	case SMTPStartTLS, SMTPImplicitTLS, SMTPPlaintext:
	default:
		return SMTP{}, errors.New("config: SMTP_TLS is none of starttls, tls and none")
	}
	switch {
	case (s.Username != "" || s.Password != "") && s.TLS == SMTPPlaintext:
		return SMTP{}, errors.New("config: SMTP_TLS=none sends no credentials; use starttls or tls")
	case s.Username != "" && s.Password == "":
		return SMTP{}, errors.New("config: SMTP_PASSWORD is required with SMTP_USERNAME")
	case s.Username == "" && s.Password != "":
		return SMTP{}, errors.New("config: SMTP_USERNAME is required with SMTP_PASSWORD")
	}
	from, err := mail.ParseAddress(os.Getenv("SMTP_FROM"))
	if err != nil {
		return SMTP{}, errors.New("config: SMTP_FROM is not one e-mail address (Name <address> or address)")
	}
	s.From = *from
	return s, nil
}
