package notification

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// from is the envelope and header sender. SES wiring will make it configuration (a verified domain); .invalid
// cannot be delivered to, so nothing replies to it meanwhile.
const from = "no-reply@salus.invalid"

// sendTimeout bounds one delivery when the context has no earlier deadline.
const sendTimeout = 30 * time.Second

type smtpSender struct {
	addr string
}

// NewSMTPSender returns the Sender on the SMTP server at addr (host:port), without authentication or TLS:
// Mailpit on the compose network. Production sends through SES once its DPA is signed.
func NewSMTPSender(addr string) Sender {
	return &smtpSender{addr: addr}
}

func (s *smtpSender) Send(ctx context.Context, m Message) error {
	if strings.ContainsAny(m.To, "\r\n") || strings.ContainsAny(m.Subject, "\r\n") {
		return errors.New("notification: a header holds a line break")
	}
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("notification: smtp: %w", err)
	}
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return fmt.Errorf("notification: smtp: %w", err)
	}
	host, _, _ := net.SplitHostPort(s.addr)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return smtpError(err)
	}
	defer func() { _ = c.Close() }()
	if err := deliver(c, m); err != nil {
		return smtpError(err)
	}
	return smtpError(c.Quit())
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

func deliver(c *smtp.Client, m Message) error {
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(m.To); err != nil {
		return err
	}
	w, err := c.Data() // dot-stuffs the body
	if err != nil {
		return err
	}
	header := "From: Salus <" + from + ">\r\nTo: " + m.To + "\r\nSubject: " + m.Subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n"
	if _, err := w.Write([]byte(header + strings.ReplaceAll(m.Body, "\n", "\r\n"))); err != nil {
		return err
	}
	return w.Close()
}
