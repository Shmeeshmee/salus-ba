package worker_test

import (
	"context"
	"crypto/rand"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
)

// secretChatText is a recognisable synthetic message (SAL-29): it must appear in nothing sent, enqueued or logged.
const secretChatText = "Synthetic secret chat text 5c1d: never outside the record"

// captureSender keeps the e-mails it is given.
type captureSender struct {
	sent []notification.Message
	mu   sync.Mutex
}

func (c *captureSender) Send(_ context.Context, m notification.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, m)
	return nil
}

func (c *captureSender) messages() []notification.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]notification.Message(nil), c.sent...)
}

type nopPublisher struct{}

func (nopPublisher) Publish(context.Context, string, []byte) error { return nil }

// chatRows are seedChatNotifications's: the conversation of seedBooking's patient and practitioner, both accounts.
type chatRows struct {
	conv, practitioner, practAccount string
}

// seedChatNotifications adds, as the owner, to seedBooking: the practitioner's account, their conversation with the
// patient, a device of the patient's app, the clinic's chat settings and a webhook to a host that never resolves.
func seedChatNotifications(t *testing.T, owner *pgxpool.Pool, b booking) chatRows {
	t.Helper()
	ctx := t.Context()
	var r chatRows
	var role, patientAccount string
	if err := owner.QueryRow(ctx, `SELECT r.practitioner_id::text, r.id::text, (SELECT account_id::text FROM patient WHERE id = $2)
		FROM practitioner_role r WHERE r.organization_id = $1`, clinicID, b.patient).Scan(&r.practitioner, &role, &patientAccount); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO account (email, password_hash, role, organization_id, practitioner_id, language)
		VALUES ($1, 'h', 'practitioner', $2, $3, 'en') RETURNING id`, "dr-"+strings.ToLower(rand.Text())+"@example.test", clinicID, r.practitioner).Scan(&r.practAccount); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO conversation (organization_id, patient_id, practitioner_role_id, created_by)
		VALUES ($1, $2, $3, $4) RETURNING id`, clinicID, b.patient, role, patientAccount).Scan(&r.conv); err != nil {
		t.Fatal(err)
	}
	box, err := auth.NewAESSecretBox(webhookKey)
	if err != nil {
		t.Fatal(err)
	}
	hook := "e1000000-0000-4000-8000-0000000000e1"
	sealed, _ := box.Seal([]byte("whsec_synthetic"), hook)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO conversation_participant (conversation_id, role, patient_id, added_by) VALUES ($1, 'patient', $2, $3)`, []any{r.conv, b.patient, patientAccount}},
		{`INSERT INTO conversation_participant (conversation_id, role, practitioner_role_id, added_by) VALUES ($1, 'practitioner', $2, $3)`, []any{r.conv, role, patientAccount}},
		{`INSERT INTO push_device (account_id, platform, token) VALUES ($1, 'ios', 'synthetic-device-token')`, []any{patientAccount}},
		{`INSERT INTO organization_chat_settings (organization_id, response_time_hours, email_after_minutes) VALUES ($1, 24, 5)`, []any{clinicID}},
		{`INSERT INTO webhook_endpoint (id, organization_id, created_by, url, events, secret_sealed)
			VALUES ($1, $2, $3, 'https://hooks.salus.invalid/chat', '{message.created}', $4)`, []any{hook, clinicID, r.practAccount, sealed}},
	} {
		if _, err := owner.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func eventually(t *testing.T, what string, done func() bool) {
	t.Helper()
	wall := clock.System{}
	deadline := wall.Now().Add(20 * time.Second)
	for !done() {
		if wall.Now().After(deadline) {
			t.Fatalf("%s: not within 20 s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// storedJobs is every job's kind, arguments and errors, as River stores them.
func storedJobs(t *testing.T, owner *pgxpool.Pool) []string {
	t.Helper()
	rows, err := owner.Query(t.Context(), `SELECT kind || ' ' || args::text || ' ' || coalesce(array_to_string(errors, ' '), '') FROM river_job`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var stored []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		stored = append(stored, s)
	}
	if len(stored) < 4 {
		t.Errorf("jobs %v, want the fan-out, the push, the e-mail check and the webhook", stored)
	}
	return stored
}

// A message through the chat as the api sends it, then the notifier and the worker as they run (SAL-29): the
// patient's device gets the fixed push, the patient the unread e-mail, the clinic's webhook an attempt; the message's
// text is in none of them, in no job's arguments or errors, and in no log line, and neither is a name.
func TestChatNotifications_NeverCarryTheMessageText(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	r := seedChatNotifications(t, db.owner, b)
	ctx := t.Context()
	var logs syncBuffer
	log := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	pusher, mails := &pushtest.Sender{}, &captureSender{}
	notifier, err := worker.NewNotifier(db.app, log, mails, pusher, "https://app.example.test", "https://patient.example.test")
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := worker.New(db.app, db.retention, liveKit(t), testFiles(t), webhookKey, log)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []interface {
		Start(context.Context) error
		Stop(context.Context) error
	}{notifier, jobs} {
		if err := c.Start(ctx); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = c.Stop(context.WithoutCancel(ctx)) })
	}

	inserter, err := worker.NewInserter(db.app)
	if err != nil {
		t.Fatal(err)
	}
	tx := postgres.NewTransactor(db.app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(db.app), tx, clock.System{})
	chat := communication.NewService(communication.Ports{
		Repo: communication.NewPostgresRepository(db.app), Publisher: nopPublisher{},
		Consents: consent.NewService(consent.NewPostgresRepository(db.app), audit, tx, clock.System{}, nil),
		Roles:    practitionerrole.NewService(practitionerrole.Ports{Repo: practitionerrole.NewPostgresRepository(db.app), Audit: audit, Tx: tx, Clock: clock.System{}}),
		Audit:    audit, Tx: tx, Clock: clock.System{}, Jobs: communication.NewRiverJobs(inserter), Log: log,
	})
	pract := appointment.Caller{AccountID: r.practAccount, OrganizationID: clinicID, PractitionerID: r.practitioner}
	if _, _, err := chat.Send(ctx, pract, r.conv, "c1000000-0000-4000-8000-0000000000c1", secretChatText); err != nil {
		t.Fatalf("Send() = %v", err)
	}

	eventually(t, "the push", func() bool { return len(pusher.Sent()) == 1 })
	// The unread check waits the clinic's five minutes: it is made due now.
	if _, err := db.owner.Exec(ctx, `UPDATE river_job SET scheduled_at = now(), state = 'available' WHERE kind = 'chat_unread_email'`); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the unread e-mail", func() bool { return len(mails.messages()) == 1 })
	eventually(t, "the webhook's failed attempt", func() bool {
		var n int
		_ = db.owner.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind = 'webhook_delivery' AND cardinality(errors) > 0`).Scan(&n)
		return n == 1
	})

	if p := pusher.Sent()[0]; p.Title != "Salus" || p.Body != "New message from your clinic" || p.Token != "synthetic-device-token" {
		t.Errorf("pushed %+v", p)
	}
	m := mails.messages()[0]
	if m.To != b.email || m.Subject != "New message from your clinic" || !strings.Contains(m.Body, "https://patient.example.test/chat") {
		t.Errorf("mailed %+v", m)
	}
	stored := storedJobs(t, db.owner)
	everything := strings.Join(stored, "\n") + "\n" + logs.String() + "\n" + m.Subject + m.Body + "\n" + pusher.Sent()[0].Title + pusher.Sent()[0].Body
	for _, never := range []string{"5c1d", secretChatText, "Ana", "Cy Example", "Example", "whsec_"} {
		if strings.Contains(everything, never) {
			t.Errorf("%q is in what was sent, enqueued or logged", never)
		}
	}
}
