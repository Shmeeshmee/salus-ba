package notification

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// sendTimeout bounds one delivery when the context has no earlier deadline.
const sendTimeout = 30 * time.Second

type smtpSender struct {
	clock              clock.Clock
	from               mail.Address
	addr, host         string
	username, password string
	security           SMTPSecurity
}

// NewSMTPSender returns the Sender on the SMTP server at addr (host:port, SMTP_ADDR): an EU relay with STARTTLS or
// implicit TLS and AUTH PLAIN as username and password (ADR 0011), or Mailpit in plaintext without them locally.
// from (SMTP_FROM) is the From header and the envelope sender. The certificate must name addr's host and chain to
// the system roots; credentials are never sent in plaintext. clk dates each message.
func NewSMTPSender(addr string, security SMTPSecurity, username, password string, from mail.Address, clk clock.Clock) Sender {
	host, _, _ := net.SplitHostPort(addr)
	return &smtpSender{clock: clk, from: from, addr: addr, host: host, username: username, password: password, security: security}
}

func (s *smtpSender) Send(ctx context.Context, m Message) error {
	if strings.ContainsAny(m.To, "\r\n") || strings.ContainsAny(m.Subject, "\r\n") {
		return errors.New("notification: a header holds a line break")
	}
	if !isASCII(m.To) {
		return ErrRecipientNotASCII
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
	if err := deliver(c, s.from, s.header(m), m.To, m.Body); err != nil {
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

// header is the message's header: the sender, the recipient, the subject, the date from the clock and a random
// Message-ID on the sender's domain (RFC 5322), and plain UTF-8 text.
func (s *smtpSender) header(m Message) string {
	_, domain, _ := strings.Cut(s.from.Address, "@")
	from := "From: " + s.from.String()
	if !isASCII(s.from.Name) {
		from = encodedWords("From", s.from.Name) + "\r\n <" + s.from.Address + ">"
	}
	subject := "Subject: " + m.Subject
	if !isASCII(m.Subject) {
		subject = encodedWords("Subject", m.Subject)
	}
	return from + "\r\nTo: " + m.To + "\r\n" + subject +
		"\r\nDate: " + s.clock.Now().Format(time.RFC1123Z) + "\r\nMessage-ID: <" + rand.Text() + "@" + domain + ">" +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n"
}

// maxLine is RFC 2047's limit on a header line that holds an encoded word.
const maxLine = 76

// encodedWords returns the header field name: value with value as UTF-8 Q encoded words (RFC 2047), folded between
// words so that no line passes maxLine. Every word holds whole characters.
func encodedWords(name, value string) string {
	const prefix, suffix = "=?utf-8?q?", "?="
	limit := maxLine - len(name) - len(": ") // the first line is the longest prefix; later ones start with a space
	var b strings.Builder
	b.WriteString(name + ": ")
	word := ""
	for _, r := range value {
		enc := qEncode(string(r))
		if word != "" && len(prefix)+len(word)+len(enc)+len(suffix) > limit {
			b.WriteString(prefix + word + suffix + "\r\n ")
			word = ""
		}
		word += enc
	}
	b.WriteString(prefix + word + suffix)
	return b.String()
}

// qEncode is RFC 2047's Q encoding: letters and digits as they are, a space as _, every other byte as =XX.
func qEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			b.WriteByte(c)
		case c == ' ':
			b.WriteByte('_')
		default:
			fmt.Fprintf(&b, "=%02X", c)
		}
	}
	return b.String()
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func deliver(c *smtp.Client, from mail.Address, header, to, body string) error {
	if err := c.Mail(from.Address); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data() // dot-stuffs the body
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(header + strings.ReplaceAll(body, "\n", "\r\n"))); err != nil {
		return err
	}
	return w.Close()
}
