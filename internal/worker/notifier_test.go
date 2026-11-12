package worker_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
)

const clinicID = "c0000000-0000-4000-8000-00000000000c"

// mailpitSender is the Sender on the compose Mailpit, as the dev stack configures it (SMTP_TLS=none, no login).
func mailpitSender(t *testing.T) notification.Sender {
	t.Helper()
	addr := os.Getenv("TEST_SMTP_ADDR")
	if addr == "" || os.Getenv("TEST_MAILPIT_URL") == "" {
		t.Fatal("TEST_SMTP_ADDR and TEST_MAILPIT_URL are not set; run the tests with `make test`")
	}
	return notification.NewSMTPSender(addr, notification.SMTPPlaintext, "", "", mail.Address{Name: "Salus", Address: "no-reply@salus.invalid"}, clock.System{})
}

// seedInvite creates, as the owner, the clinic and an invite to a fresh synthetic address, and returns the
// invite's id and address.
func seedInvite(t *testing.T, owner *pgxpool.Pool, accepted bool) (string, string) {
	t.Helper()
	return seedInviteTo(t, owner, "invite-"+strings.ToLower(rand.Text())+"@example.test", accepted)
}

// seedInviteTo creates the clinic and an invite to email, and returns the invite's id and address.
func seedInviteTo(t *testing.T, owner *pgxpool.Pool, email string, accepted bool) (string, string) {
	t.Helper()
	var id string
	err := owner.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic') ON CONFLICT DO NOTHING)
		INSERT INTO invite (organization_id, expires_at, role, email, accepted_at) VALUES ($1, now() + interval '72 hours', 'admin', $2, CASE WHEN $3 THEN now() END)
		RETURNING id`, clinicID, email, accepted).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id, email
}

// runJob starts the notifier, enqueues invite_email for inviteID the way the api does, and returns the
// job's final state.
func runJob(t *testing.T, app *pgxpool.Pool, inviteID string) rivertype.JobState {
	t.Helper()
	client, err := worker.NewNotifier(app, slog.New(slog.DiscardHandler), mailpitSender(t), &pushtest.Sender{}, "https://app.example.test", "https://patient.example.test")
	if err != nil {
		t.Fatalf("NewNotifier() error = %v", err)
	}
	events, cancel := client.Subscribe(river.EventKindJobCompleted, river.EventKindJobCancelled, river.EventKindJobFailed)
	defer cancel()
	if err := client.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Stop(context.WithoutCancel(t.Context())) }()

	inserter, err := worker.NewInserter(app)
	if err != nil {
		t.Fatalf("NewInserter() error = %v", err)
	}
	err = postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
		return auth.NewRiverJobs(inserter).EnqueueInviteEmail(ctx, inviteID, clinicID)
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-events:
		return ev.Job.State
	case <-time.After(15 * time.Second):
		t.Fatal("the invite_email job did not finish within 15s")
	}
	return ""
}

func TestNotifier_SendsTheInviteEmailWithANewToken(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	id, email := seedInvite(t, owner, false)

	if state := runJob(t, app, id); state != rivertype.JobStateCompleted {
		t.Fatalf("job state %s, want completed", state)
	}
	var hashed bool
	if err := owner.QueryRow(t.Context(), "SELECT token_hash IS NOT NULL FROM invite WHERE id = $1", id).Scan(&hashed); err != nil || !hashed {
		t.Errorf("token issued: %v (%v), want a stored hash", hashed, err)
	}
	text := mailpitText(t, email)
	if !strings.Contains(text, "Synthetic clinic") || !strings.Contains(text, "https://app.example.test/invite#") {
		t.Errorf("the e-mail lacks the clinic name or the link:\n%s", text)
	}
}

func TestNotifier_CancelsTheJobOfAnInviteThatIsNoLongerOpen(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	id, _ := seedInvite(t, owner, true)

	if state := runJob(t, app, id); state != rivertype.JobStateCancelled {
		t.Errorf("job state %s, want cancelled: there is nothing to send", state)
	}
}

// An address SMTP cannot carry (a UTF-8 local part, which registration accepts) fails the same way on every retry:
// the job is cancelled, not retried for days.
func TestNotifier_CancelsTheJobOfAnInviteToAnAddressSMTPCannotCarry(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	id, _ := seedInviteTo(t, owner, "pozivnica-"+strings.ToLower(rand.Text())+"-ć@example.test", false)

	if state := runJob(t, app, id); state != rivertype.JobStateCancelled {
		t.Errorf("job state %s, want cancelled: a retry cannot send it either", state)
	}
}

func TestNewNotifier_NeedsTheSMTPServerAndTheWebAndPatientAppAddresses(t *testing.T) {
	_, app := pgtest.NewAppDatabase(t)
	sender := mailpitSender(t)
	for _, cfg := range []struct {
		sender          notification.Sender
		web, patientApp string
	}{
		{nil, "https://app.example.test", "https://patient.example.test"},
		{sender, "", "https://patient.example.test"},
		{sender, "https://app.example.test", ""},
	} {
		if _, err := worker.NewNotifier(app, slog.New(slog.DiscardHandler), cfg.sender, &pushtest.Sender{}, cfg.web, cfg.patientApp); err == nil {
			t.Errorf("NewNotifier(sender %v, %q, %q): error = nil", cfg.sender != nil, cfg.web, cfg.patientApp)
		}
	}
}

// mailpitText returns the text of the one message to address in the compose Mailpit, and deletes it.
func mailpitText(t *testing.T, address string) string {
	t.Helper()
	api := os.Getenv("TEST_MAILPIT_URL")
	client := &http.Client{Timeout: 5 * time.Second}
	get := func(u string, v any) {
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
	var found struct{ Messages []struct{ ID string } }
	get(api+"/api/v1/search?query="+url.QueryEscape(`to:"`+address+`"`), &found)
	if len(found.Messages) != 1 {
		t.Fatalf("%d messages to the address, want 1", len(found.Messages))
	}
	var msg struct{ Text string }
	get(api+"/api/v1/message/"+found.Messages[0].ID, &msg)
	body, _ := json.Marshal(map[string][]string{"IDs": {found.Messages[0].ID}})
	if req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, api+"/api/v1/messages", bytes.NewReader(body)); err == nil {
		if resp, err := client.Do(req); err == nil {
			_ = resp.Body.Close()
		}
	}
	return msg.Text
}
