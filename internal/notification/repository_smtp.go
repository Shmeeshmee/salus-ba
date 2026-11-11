package notification

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// sendTimeout bounds one delivery when the context has no earlier deadline.
const sendTimeout = 30 * time.Second

type smtpSender struct {
	from               mail.Address
	addr, host         string
	username, password string
	security           SMTPSecurity
}

// NewSMTPSender returns the Sender on the SMTP server at addr (host:port, SMTP_ADDR): an EU relay with STARTTLS or
// implicit TLS and AUTH PLAIN as username and password (ADR 0011), or Mailpit in plaintext without them locally.
// from (SMTP_FROM) is the From header and the envelope sender. The certificate must name addr's host and chain to
// the system roots; credentials are never sent in plaintext.
func NewSMTPSender(addr string, security SMTPSecurity, username, password string, from mail.Address) Sender {
	host, _, _ := net.SplitHostPort(addr)
	return &smtpSender{from: from, addr: addr, host: host, username: username, password: password, security: security}
}

func (s *smtpSender) Send(ctx context.Context, m Message) error {
	if strings.ContainsAny(m.To, "\r\n") || strings.ContainsAny(m.Subject, "\r\n") {
		return errors.New("notification: a header holds a line break")
	}
	if s.security == SMTPPlaintext && (s.username != "" || s.password != "") {
		return errors.New("notification: smtp: credentials are never sent without TLS")
	}
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	conn, err := s.dial(ctx)
	if err != nil {
		return fmt.Errorf("notification: smtp: %w", err)
	}
	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		_ = conn.Close()
		return smtpError(err)
	}
	defer func() { _ = c.Close() }()
	if err := s.secure(c); err != nil {
		return smtpError(err)
	}
	if err := deliver(c, s.from, m); err != nil {
		return smtpError(err)
	}
	return smtpError(c.Quit())
}

func (s *smtpSender) tlsConfig() *tls.Config {
	return &tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}
}

// dial connects to the server, through TLS from the first byte for implicit TLS, with the context's deadline on
// every read and write.
func (s *smtpSender) dial(ctx context.Context) (net.Conn, error) {
	var conn net.Conn
	var err error
	if s.security == SMTPImplicitTLS {
		conn, err = (&tls.Dialer{Config: s.tlsConfig()}).DialContext(ctx, "tcp", s.addr)
	} else {
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", s.addr)
	}
	if err != nil {
		return nil, err
	}
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// secure upgrades the connection with STARTTLS when it is configured, refusing a server that does not offer it,
// then logs in when there are credentials.
func (s *smtpSender) secure(c *smtp.Client) error {
	if s.security == SMTPStartTLS {
		if err := c.Hello("localhost"); err != nil {
			return err
		}
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("the server does not offer STARTTLS")
		}
		if err := c.StartTLS(s.tlsConfig()); err != nil {
			return err
		}
	}
	if s.username == "" {
		return nil
	}
	if ok, _ := c.Extension("AUTH"); !ok {
		return errors.New("the server does not offer AUTH")
	}
	return c.Auth(smtp.PlainAuth("", s.username, s.password, s.host))
}

// smtpError keeps only the code of a server's reply: its text often echoes the recipient's address, and the
// error is stored with the job and logged.
func smtpError(err error) error {
	var reply *textproto.Error
	switch {
	case err == nil:
		return nil
	case errors.As(err, &reply):
		return fmt.Errorf("notification: smtp: the server replied %d", reply.Code)
	}
	return fmt.Errorf("notification: smtp: %w", err)
}

func deliver(c *smtp.Client, from mail.Address, m Message) error {
	if err := c.Mail(from.Address); err != nil {
		return err
	}
	if err := c.Rcpt(m.To); err != nil {
		return err
	}
	w, err := c.Data() // dot-stuffs the body
	if err != nil {
		return err
	}
	// A subject or a sender's name outside ASCII is encoded (RFC 2047); an ASCII one is left as it is.
	header := "From: " + from.String() + "\r\nTo: " + m.To + "\r\nSubject: " + mime.QEncoding.Encode("utf-8", m.Subject) +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n"
	if _, err := w.Write([]byte(header + strings.ReplaceAll(m.Body, "\n", "\r\n"))); err != nil {
		return err
	}
	return w.Close()
}
