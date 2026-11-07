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
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/notification"
)

// mailpit returns the SMTP address and the API URL of the compose Mailpit; the test fails without them, as
// the database tests do without theirs.
func mailpit(t *testing.T) (smtpAddr, api string) {
	t.Helper()
	smtpAddr, api = os.Getenv("TEST_SMTP_ADDR"), os.Getenv("TEST_MAILPIT_URL")
	if smtpAddr == "" || api == "" {
		t.Fatal("TEST_SMTP_ADDR and TEST_MAILPIT_URL are not set; run the tests with `make test`, which points them at the compose Mailpit")
	}
	return smtpAddr, api
}

type mailpitMessage struct {
	ID      string
	Subject string
	Text    string
	From    struct{ Address string }
	To      []struct{ Address string }
}

func mailpitGet(t *testing.T, client *http.Client, u string, v any) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}

func TestSMTPSender_DeliversThePlainTextMessage(t *testing.T) {
	smtpAddr, api := mailpit(t)
	to := "invite-" + rand.Text() + "@example.test"
	msg := notification.Message{To: to, Subject: "Your invitation to Salus", Body: "Line one, with ünïcode.\n.\nhttps://app.example.test/invite#abc\n"}

	if err := notification.NewSMTPSender(smtpAddr).Send(t.Context(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	var found struct{ Messages []mailpitMessage }
	mailpitGet(t, client, api+"/api/v1/search?query="+url.QueryEscape(`to:"`+to+`"`), &found)
	if len(found.Messages) != 1 {
		t.Fatalf("%d messages to the address in Mailpit, want 1", len(found.Messages))
	}
	id := found.Messages[0].ID
	t.Cleanup(func() { // the dev Mailpit is left as it was
		body, _ := json.Marshal(map[string][]string{"IDs": {id}})
		req, err := http.NewRequestWithContext(context.WithoutCancel(t.Context()), http.MethodDelete, api+"/api/v1/messages", bytes.NewReader(body))
		if err == nil {
			if resp, err := client.Do(req); err == nil {
				_ = resp.Body.Close()
			}
		}
	})
	var got mailpitMessage
	mailpitGet(t, client, fmt.Sprintf("%s/api/v1/message/%s", api, id), &got)
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

	if err := notification.NewSMTPSender(smtpAddr).Send(t.Context(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	var found struct{ Messages []mailpitMessage }
	mailpitGet(t, client, api+"/api/v1/search?query="+url.QueryEscape(`to:"`+to+`"`), &found)
	if len(found.Messages) != 1 {
		t.Fatalf("%d messages to the address in Mailpit, want 1", len(found.Messages))
	}
	id := found.Messages[0].ID
	t.Cleanup(func() { // the dev Mailpit is left as it was
		body, _ := json.Marshal(map[string][]string{"IDs": {id}})
		req, err := http.NewRequestWithContext(context.WithoutCancel(t.Context()), http.MethodDelete, api+"/api/v1/messages", bytes.NewReader(body))
		if err == nil {
			if resp, err := client.Do(req); err == nil {
				_ = resp.Body.Close()
			}
		}
	})
	var got mailpitMessage
	mailpitGet(t, client, fmt.Sprintf("%s/api/v1/message/%s", api, id), &got)
	if got.Subject != msg.Subject {
		t.Errorf("delivered subject %q, want %q", got.Subject, msg.Subject)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("%s/api/v1/message/%s/raw", api, id), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	header, _, _ := strings.Cut(string(raw), "\r\n\r\n")
	for _, line := range strings.Split(header, "\r\n") {
		if strings.HasPrefix(line, "Subject:") && !isASCII(line) {
			t.Errorf("the subject header %q is not ASCII", line)
		}
	}
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
		if err := notification.NewSMTPSender(smtpAddr).Send(t.Context(), m); err == nil {
			t.Errorf("Send(%q, %q): error = nil, want a refusal", m.To, m.Subject)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := notification.NewSMTPSender("127.0.0.1:1").Send(ctx, notification.Message{To: "cy@example.test", Subject: "s", Body: "b"}); err == nil {
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
		err := notification.NewSMTPSender(refusingServer(t, greeting)).Send(t.Context(), m)
		if err == nil || strings.Contains(err.Error(), "example.test") || strings.Contains(err.Error(), "the body") {
			t.Errorf("Send() to a server greeting %q = %v, want an error without the address or the body", greeting, err)
		}
	}
}
