package worker_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
)

// Package I of docs/plans/clinic-cms.md end to end in the workers: the monthly job issues a clinic's invoice and
// enqueues subscription_invoice_email in the same transaction; the notifier sends it to the clinic's billing e-mail
// through the compose Mailpit, in the clinic's language, with the invoice's PDF attached. Synthetic seller and clinic.
func TestNotifier_SendsAnIssuedMembershipInvoiceWithItsPDF(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	const clinic = "0a000000-0000-4000-8000-0000000000e1"
	billing := "billing-" + strings.ToLower(rand.Text()) + "@example.test"
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.owner.Exec(t.Context(), q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO organization (id, name, legal_name, tax_id, billing_email, default_language)
		VALUES ($1, 'Synthetic clinic E1', 'Synthetic clinic E1 d.o.o.', '4200000000001', $2, 'en')`, clinic, billing)
	exec("INSERT INTO subscription_plan (code, name, price_per_seat_minor) VALUES ('basic', 'Basic', 10000)")
	exec(`INSERT INTO clinic_subscription (organization_id, plan_code, status, trial_ends_at) VALUES ($1, 'basic', 'trial', '2026-11-20T00:00:00Z')`, clinic)
	var practitioner string
	if err := db.owner.QueryRow(t.Context(), "INSERT INTO practitioner (organization_id, given_name, family_name) VALUES ($1, 'Synthetic', 'Doctor') RETURNING id", clinic).Scan(&practitioner); err != nil {
		t.Fatal(err)
	}
	exec("INSERT INTO practitioner_role (organization_id, practitioner_id, active) VALUES ($1, $2, true)", clinic, practitioner)

	month := time.Date(2026, 12, 1, 6, 0, 0, 0, time.UTC)
	if kind, logs := runJobOn(t, db, liveKit(t), testFiles(t), subscription.MonthlyArgs{Month: month, OrganizationID: clinic}, nil); kind != river.EventKindJobCompleted {
		t.Fatalf("the monthly job ended %s: %s", kind, logs)
	}
	var number int64
	if err := db.owner.QueryRow(t.Context(), "SELECT number FROM subscription_invoice WHERE organization_id = $1", clinic).Scan(&number); err != nil {
		t.Fatal(err)
	}

	n := testNotifier(mailpitSender(t), &pushtest.Sender{})
	n.Seller = subscription.Seller{Name: "[SELLER]", TaxID: "[SELLER TAX ID]", Address: "[SELLER ADDRESS]", Account: "[SELLER ACCOUNT]"}
	client, err := worker.NewNotifier(db.app, slog.New(slog.DiscardHandler), n)
	if err != nil {
		t.Fatalf("NewNotifier() error = %v", err)
	}
	events, cancel := client.Subscribe(river.EventKindJobCompleted, river.EventKindJobCancelled, river.EventKindJobFailed)
	defer cancel()
	if err := client.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Stop(context.WithoutCancel(t.Context())) }()
	timeout := time.After(15 * time.Second)
	for done := false; !done; {
		select {
		case ev := <-events:
			if ev.Job.Kind != (subscription.InvoiceEmailArgs{}).Kind() {
				continue
			}
			if ev.Kind != river.EventKindJobCompleted {
				t.Fatalf("the subscription_invoice_email job ended %s: %v", ev.Kind, ev.Job.Errors)
			}
			done = true
		case <-timeout:
			t.Fatal("the subscription_invoice_email job did not finish within 15s")
		}
	}

	m := mailpitMessage(t, billing)
	if want := "Salus membership invoice no. " + strconv.FormatInt(number, 10); m.Subject != want {
		t.Errorf("the subject is %q, want %q", m.Subject, want)
	}
	if len(m.Attachments) != 1 || m.Attachments[0].ContentType != "application/pdf" || !strings.HasPrefix(string(m.attachment(t, 0)), "%PDF-") {
		t.Fatalf("the attachments are %+v, want one PDF", m.Attachments)
	}
}

// mailpitMail is one message in the compose Mailpit, as its API describes it.
type mailpitMail struct {
	ID          string
	Subject     string
	Text        string
	Attachments []struct{ PartID, FileName, ContentType string }
}

// mailpitMessage returns the one message to address in the compose Mailpit and deletes it when the test ends.
func mailpitMessage(t *testing.T, address string) mailpitMail {
	t.Helper()
	api := os.Getenv("TEST_MAILPIT_URL")
	var found struct{ Messages []struct{ ID string } }
	mailpitGet(t, api+"/api/v1/search?query="+url.QueryEscape(`to:"`+address+`"`), &found)
	if len(found.Messages) != 1 {
		t.Fatalf("%d messages to the address, want 1", len(found.Messages))
	}
	var m mailpitMail
	mailpitGet(t, api+"/api/v1/message/"+found.Messages[0].ID, &m)
	t.Cleanup(func() {
		body, _ := json.Marshal(map[string][]string{"IDs": {m.ID}})
		req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, api+"/api/v1/messages", bytes.NewReader(body))
		if err == nil {
			if resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req); err == nil {
				_ = resp.Body.Close()
			}
		}
	})
	return m
}

// attachment downloads the message's i-th attachment.
func (m mailpitMail) attachment(t *testing.T, i int) []byte {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, os.Getenv("TEST_MAILPIT_URL")+"/api/v1/message/"+m.ID+"/part/"+m.Attachments[i].PartID, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mailpitGet(t *testing.T, u string, v any) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}
