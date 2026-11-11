package notification_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	netmail "net/mail"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/notification"
)

// from is the sender the tests configure (SMTP_FROM); .invalid cannot be delivered to.
var from = netmail.Address{Name: "Salus", Address: "no-reply@salus.invalid"}

// env returns the values of the test variables names, failing without any of them, as the database tests do
// without theirs.
func env(t *testing.T, names ...string) []string {
	t.Helper()
	values := make([]string, len(names))
	for i, name := range names {
		if values[i] = os.Getenv(name); values[i] == "" {
			t.Fatalf("%s is not set; run the tests with `make test`, which points the TEST_SMTP_* and TEST_*MAILPIT_URL variables at the compose Mailpits", name)
		}
	}
	return values
}

// mailpit returns the SMTP address and the API URL of the compose Mailpit (plaintext, no login).
func mailpit(t *testing.T) (smtpAddr, api string) {
	t.Helper()
	v := env(t, "TEST_SMTP_ADDR", "TEST_MAILPIT_URL")
	return v[0], v[1]
}

// plaintext is the Sender on the compose Mailpit, as the dev stack configures it.
func plaintext(addr string) notification.Sender {
	return notification.NewSMTPSender(addr, notification.SMTPPlaintext, "", "", from)
}

// relay is one of the compose Mailpits that stand in for a relay: TLS and a login required.
type relay struct {
	security           notification.SMTPSecurity
	addr, api          string
	username, password string
}

// relays returns the STARTTLS Mailpit and the implicit-TLS one.
func relays(t *testing.T) []relay {
	t.Helper()
	v := env(t, "TEST_SMTP_STARTTLS_ADDR", "TEST_SMTP_STARTTLS_MAILPIT_URL", "TEST_SMTP_TLS_ADDR", "TEST_SMTP_TLS_MAILPIT_URL",
		"TEST_SMTP_USERNAME", "TEST_SMTP_PASSWORD")
	return []relay{
		{security: notification.SMTPStartTLS, addr: v[0], api: v[1], username: v[4], password: v[5]},
		{security: notification.SMTPImplicitTLS, addr: v[2], api: v[3], username: v[4], password: v[5]},
	}
}

type mailpitMessage struct {
	ID      string
	Subject string
	Text    string
	From    struct{ Name, Address string }
	To      []struct{ Address string }
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func mailpitGet(t *testing.T, u string, v any) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}

// received returns the messages to the address in the Mailpit at api, and deletes them when the test ends, so
// the Mailpit is left as it was.
func received(t *testing.T, api, to string) []mailpitMessage {
	t.Helper()
	var found struct{ Messages []mailpitMessage }
	mailpitGet(t, api+"/api/v1/search?query="+url.QueryEscape(`to:"`+to+`"`), &found)
	ids := make([]string, 0, len(found.Messages))
	for _, m := range found.Messages {
		ids = append(ids, m.ID)
	}
	if len(ids) > 0 {
		t.Cleanup(func() {
			body, _ := json.Marshal(map[string][]string{"IDs": ids})
			req, err := http.NewRequestWithContext(context.WithoutCancel(t.Context()), http.MethodDelete, api+"/api/v1/messages", bytes.NewReader(body))
			if err == nil {
				if resp, err := httpClient.Do(req); err == nil {
					_ = resp.Body.Close()
				}
			}
		})
	}
	return found.Messages
}

// receivedOne returns the one message to the address in the Mailpit at api, in full.
func receivedOne(t *testing.T, api, to string) mailpitMessage {
	t.Helper()
	found := received(t, api, to)
	if len(found) != 1 {
		t.Fatalf("%d messages to the address in Mailpit, want 1", len(found))
	}
	var got mailpitMessage
	mailpitGet(t, fmt.Sprintf("%s/api/v1/message/%s", api, found[0].ID), &got)
	return got
}

func TestSMTPSender_DeliversThePlainTextMessage(t *testing.T) {
	smtpAddr, api := mailpit(t)
	to := "invite-" + rand.Text() + "@example.test"
	msg := notification.Message{To: to, Subject: "Your invitation to Salus", Body: "Line one, with ünïcode.\n.\nhttps://app.example.test/invite#abc\n"}

	if err := plaintext(smtpAddr).Send(t.Context(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	got := receivedOne(t, api, to)
	if got.Subject != msg.Subject || strings.ReplaceAll(got.Text, "\r\n", "\n") != msg.Body || got.From.Address != "no-reply@salus.invalid" || len(got.To) != 1 || got.To[0].Address != to {
		t.Errorf("delivered %+v,\nwant subject %q, body %q, from no-reply@salus.invalid, to %s", got, msg.Subject, msg.Body, to)
	}
}

// A subject outside ASCII (Cyrillic, Arabic, the accents of the other languages) is encoded in the header
// (RFC 2047), so every server and client reads it; the delivered message gives it back as written.
func TestSMTPSender_EncodesASubjectOutsideASCII(t *testing.T) {
	smtpAddr, api := mailpit(t)
	to := "subject-" + rand.Text() + "@example.test"
	msg := notification.Message{To: to, Subject: "Подсјетник на ваш термин", Body: "Тијело.\n"}

	if err := plaintext(smtpAddr).Send(t.Context(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	got := receivedOne(t, api, to)
	if got.Subject != msg.Subject {
		t.Errorf("delivered subject %q, want %q", got.Subject, msg.Subject)
	}
	for _, line := range strings.Split(rawHeader(t, api, got.ID), "\r\n") {
		if strings.HasPrefix(line, "Subject:") && !isASCII(line) {
			t.Errorf("the subject header %q is not ASCII", line)
		}
	}
}

// rawHeader returns the header of the message id in the Mailpit at api, as it was delivered.
func rawHeader(t *testing.T, api, id string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("%s/api/v1/message/%s/raw", api, id), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	header, _, _ := strings.Cut(string(raw), "\r\n\r\n")
	return header
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

func TestSMTPSender_RefusesAHeaderInjectionAndAnUnreachableServer(t *testing.T) {
	smtpAddr, _ := mailpit(t)
	for _, m := range []notification.Message{
		{To: "cy@example.test\r\nBcc: dee@example.test", Subject: "s", Body: "b"},
		{To: "cy@example.test", Subject: "s\nBcc: dee@example.test", Body: "b"},
	} {
		if err := plaintext(smtpAddr).Send(t.Context(), m); err == nil {
			t.Errorf("Send(%q, %q): error = nil, want a refusal", m.To, m.Subject)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := plaintext("127.0.0.1:1").Send(ctx, notification.Message{To: "cy@example.test", Subject: "s", Body: "b"}); err == nil {
		t.Error("Send() to a closed port: error = nil")
	}
}

// refusingServer greets with greeting and answers every command with 550, on a local port.
func refusingServer(t *testing.T, greeting string) string {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				_, _ = conn.Write([]byte(greeting + "\r\n"))
				r := bufio.NewReader(conn)
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if strings.HasPrefix(line, "EHLO") {
						_, _ = conn.Write([]byte("250 synthetic\r\n"))
						continue
					}
					_, _ = conn.Write([]byte("550 <cy@example.test> the body refused\r\n")) // servers echo the address
				}
			}()
		}
	}()
	return ln.Addr().String()
}

func TestSMTPSender_ReportsARefusalWithoutTheMessage(t *testing.T) {
	m := notification.Message{To: "cy@example.test", Subject: "s", Body: "the body"}
	for _, greeting := range []string{"554 go away", "220 synthetic ESMTP"} {
		err := plaintext(refusingServer(t, greeting)).Send(t.Context(), m)
		if err == nil || strings.Contains(err.Error(), "example.test") || strings.Contains(err.Error(), "the body") {
			t.Errorf("Send() to a server greeting %q = %v, want an error without the address or the body", greeting, err)
		}
	}
}

// A relay (ADR 0011): STARTTLS on 587 or implicit TLS on 465, then AUTH; the configured from address is the
// From header, its name encoded when it is not ASCII, and the envelope sender.
func TestSMTPSender_DeliversThroughARelayOverTLSWithTheLogin(t *testing.T) {
	sender := netmail.Address{Name: "Salus Klinika Čapljina", Address: "noreply@clinic.example.test"}
	for _, r := range relays(t) {
		t.Run(string(r.security), func(t *testing.T) {
			to := "relay-" + rand.Text() + "@example.test"
			msg := notification.Message{To: to, Subject: "Your invitation to Salus", Body: "Line one.\n"}

			err := notification.NewSMTPSender(r.addr, r.security, r.username, r.password, sender).Send(t.Context(), msg)

			if err != nil {
				t.Fatalf("Send() error = %v", err)
			}
			got := receivedOne(t, r.api, to)
			if got.From.Address != sender.Address || got.From.Name != sender.Name || got.Subject != msg.Subject || strings.ReplaceAll(got.Text, "\r\n", "\n") != msg.Body {
				t.Errorf("delivered %+v, want from %s, subject %q, body %q", got, sender.String(), msg.Subject, msg.Body)
			}
			header := rawHeader(t, r.api, got.ID)
			if !isASCII(header) || !strings.Contains(header, "Return-Path: <"+sender.Address+">") {
				t.Errorf("header %q: want ASCII only and the envelope sender %s", header, sender.Address)
			}
		})
	}
}

// A wrong login is refused, and the error carries the reply's code: neither the password nor the address.
func TestSMTPSender_ReportsAWrongLoginWithoutThePassword(t *testing.T) {
	for _, r := range relays(t) {
		t.Run(string(r.security), func(t *testing.T) {
			to := "login-" + rand.Text() + "@example.test"

			err := notification.NewSMTPSender(r.addr, r.security, r.username, "wrong-password-marker", from).Send(t.Context(),
				notification.Message{To: to, Subject: "s", Body: "b"})

			if err == nil || strings.Contains(err.Error(), "marker") || strings.Contains(err.Error(), "example.test") {
				t.Errorf("Send() with a wrong password = %v, want an error without the password or the address", err)
			}
			if n := len(received(t, r.api, to)); n != 0 {
				t.Errorf("%d messages delivered, want none", n)
			}
		})
	}
}

// STARTTLS is required, not opportunistic: a server that does not offer it gets nothing, the login least of all.
func TestSMTPSender_RefusesAServerWithoutSTARTTLS(t *testing.T) {
	smtpAddr, api := mailpit(t)
	r := relays(t)[0]
	to := "starttls-" + rand.Text() + "@example.test"

	err := notification.NewSMTPSender(smtpAddr, notification.SMTPStartTLS, r.username, r.password, from).Send(t.Context(),
		notification.Message{To: to, Subject: "s", Body: "b"})

	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Errorf("Send() to a server without STARTTLS = %v, want an error naming STARTTLS", err)
	}
	if n := len(received(t, api, to)); n != 0 {
		t.Errorf("%d messages delivered in plaintext, want none", n)
	}
}

// The certificate is verified against the system roots and the host of the address: by IP address, which the
// relay's certificate does not name, nothing is sent.
func TestSMTPSender_RefusesACertificateThatDoesNotNameTheHost(t *testing.T) {
	for _, r := range relays(t) {
		t.Run(string(r.security), func(t *testing.T) {
			host, port, err := net.SplitHostPort(r.addr)
			if err != nil {
				t.Fatal(err)
			}
			ips, err := net.DefaultResolver.LookupHost(t.Context(), host)
			if err != nil || len(ips) == 0 {
				t.Fatalf("LookupHost(%s) = %v, %v", host, ips, err)
			}
			to := "cert-" + rand.Text() + "@example.test"

			err = notification.NewSMTPSender(net.JoinHostPort(ips[0], port), r.security, r.username, r.password, from).Send(t.Context(),
				notification.Message{To: to, Subject: "s", Body: "b"})

			if err == nil || !strings.Contains(err.Error(), "certificate") {
				t.Errorf("Send() by IP address = %v, want a certificate error", err)
			}
			if n := len(received(t, r.api, to)); n != 0 {
				t.Errorf("%d messages delivered, want none", n)
			}
		})
	}
}

// Credentials never travel in plaintext, whatever the caller passes (config.LoadSMTP refuses them first).
func TestSMTPSender_SendsNoCredentialsInPlaintext(t *testing.T) {
	smtpAddr, api := mailpit(t)
	to := "plain-" + rand.Text() + "@example.test"

	err := notification.NewSMTPSender(smtpAddr, notification.SMTPPlaintext, "user", "password-marker", from).Send(t.Context(),
		notification.Message{To: to, Subject: "s", Body: "b"})

	if err == nil || strings.Contains(err.Error(), "marker") {
		t.Errorf("Send() with credentials in plaintext = %v, want a refusal without the password", err)
	}
	if n := len(received(t, api, to)); n != 0 {
		t.Errorf("%d messages delivered, want none", n)
	}
}
