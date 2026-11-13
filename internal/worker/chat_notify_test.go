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
		{`INSERT INTO conversation_participant (conversation_id, role, practitioner_role_id, organization_id, added_by) VALUES ($1, 'practitioner', $2, $3, $4)`, []any{r.conv, role, clinicID, patientAccount}},
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
	pusher, mails := startChatJobs(t, db, log)
	chat := sendingChat(t, db, log)
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

// startChatJobs starts the notifier and the worker as they run, with a captured push sender and mailbox.
func startChatJobs(t *testing.T, db logins, log *slog.Logger) (*pushtest.Sender, *captureSender) {
	t.Helper()
	ctx := t.Context()
	pusher, mails := &pushtest.Sender{}, &captureSender{}
	notifier, err := worker.NewNotifier(db.app, log, mails, pusher, "https://app.example.test", "https://patient.example.test")
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := worker.New(db.app, db.retention, db.stats, liveKit(t), testFiles(t), webhookKey, worker.Trends{}, log)
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
	return pusher, mails
}

// sendingChat is the chat's service as the api builds it for sending.
func sendingChat(t *testing.T, db logins, log *slog.Logger) *communication.Service {
	t.Helper()
	inserter, err := worker.NewInserter(db.app)
	if err != nil {
		t.Fatal(err)
	}
	tx := postgres.NewTransactor(db.app)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(db.app), tx, clock.System{})
	consents := consent.NewService(consent.NewPostgresRepository(db.app), audit, tx, clock.System{}, nil)
	return communication.NewService(communication.Ports{
		Repo: communication.NewPostgresRepository(db.app), Publisher: nopPublisher{}, Consents: consents, ChatConsents: consents,
		Roles: practitionerrole.NewService(practitionerrole.Ports{Repo: practitionerrole.NewPostgresRepository(db.app), Audit: audit, Tx: tx, Clock: clock.System{}}),
		Audit: audit, Tx: tx, Clock: clock.System{}, Jobs: communication.NewRiverJobs(inserter), Log: log,
	})
}

// guestClinicID is the second clinic of seedChatGuest.
const guestClinicID = "0a000000-0000-4000-8000-0000000000b2"

// seedChatGuest adds, as the owner, a doctor of a second clinic to r's conversation, with their account, as an active
// participant the patient granted a chat consent for (SAL-28).
func seedChatGuest(t *testing.T, owner *pgxpool.Pool, b booking, r chatRows) {
	t.Helper()
	ctx := t.Context()
	var practitioner, role string
	if err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (id, name, listed) VALUES ($1, 'Synthetic guest clinic', true) RETURNING id),
		p AS (INSERT INTO practitioner (organization_id, given_name, family_name) SELECT id, 'Gus', 'Example' FROM o RETURNING id, organization_id)
		INSERT INTO practitioner_role (organization_id, practitioner_id) SELECT organization_id, id FROM p RETURNING practitioner_id::text, id::text`,
		guestClinicID).Scan(&practitioner, &role); err != nil {
		t.Fatal(err)
	}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO account (email, password_hash, role, organization_id, practitioner_id, language) VALUES ($1, 'h', 'practitioner', $2, $3, 'en')`,
			[]any{"guest-" + strings.ToLower(rand.Text()) + "@example.test", guestClinicID, practitioner}},
		{`INSERT INTO conversation_participant (conversation_id, role, practitioner_role_id, organization_id, status, added_by)
			VALUES ($1, 'practitioner', $2, $3, 'active', $4)`, []any{r.conv, role, guestClinicID, r.practAccount}},
		{`INSERT INTO chat_consent (patient_id, organization_id, conversation_id, practitioner_role_id, practitioner_organization_id, status, date_time)
			VALUES ($1, $2, $3, $4, $5, 'active', now() - interval '1 hour')`, []any{b.patient, clinicID, r.conv, role, guestClinicID}},
	} {
		if _, err := owner.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
}

// failedChatJobs is every chat notification job that recorded an error, but the webhook's attempts at a host that
// never resolves, which fail by design; those must fail on the host and nothing else.
func failedChatJobs(t *testing.T, owner *pgxpool.Pool) []string {
	t.Helper()
	rows, err := owner.Query(t.Context(), `SELECT kind, array_to_string(errors, ' ') FROM river_job WHERE cardinality(errors) > 0`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var failed []string
	for rows.Next() {
		var kind, errs string
		if err := rows.Scan(&kind, &errs); err != nil {
			t.Fatal(err)
		}
		if kind != "webhook_delivery" || strings.Contains(errs, "panic") {
			failed = append(failed, kind+" "+errs)
		}
	}
	return failed
}

// A conversation with a doctor of another clinic the patient granted (SAL-28, SAL-29): the notifier and the worker
// check the guest's chat consent, and the patient still gets the push and the unread e-mail, the guest their e-mail,
// with no job failing on the way.
func TestChatNotifications_ReachAConversationWithAGrantedGuest(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	r := seedChatNotifications(t, db.owner, b)
	seedChatGuest(t, db.owner, b, r)
	ctx := t.Context()
	var logs syncBuffer
	log := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	pusher, mails := startChatJobs(t, db, log)
	pract := appointment.Caller{AccountID: r.practAccount, OrganizationID: clinicID, PractitionerID: r.practitioner}
	if _, _, err := sendingChat(t, db, log).Send(ctx, pract, r.conv, "c2000000-0000-4000-8000-0000000000c2", secretChatText); err != nil {
		t.Fatalf("Send() = %v", err)
	}

	eventually(t, "the fan-out's first attempt", func() bool {
		var done bool
		_ = db.owner.QueryRow(ctx, `SELECT state = 'completed' OR cardinality(errors) > 0 FROM river_job WHERE kind = $1`,
			communication.MessageSentArgs{}.Kind()).Scan(&done)
		return done
	})
	if failed := failedChatJobs(t, db.owner); failed != nil {
		t.Fatalf("jobs failed:\n%s", strings.Join(failed, "\n"))
	}
	eventually(t, "the push", func() bool { return len(pusher.Sent()) == 1 })
	if _, err := db.owner.Exec(ctx, `UPDATE river_job SET scheduled_at = now(), state = 'available' WHERE kind = 'chat_unread_email'`); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the patient's and the guest's unread e-mails", func() bool { return len(mails.messages()) == 2 })
	eventually(t, "the webhook's failed attempt", func() bool {
		var n int
		_ = db.owner.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind = 'webhook_delivery' AND cardinality(errors) > 0`).Scan(&n)
		return n == 1
	})
	var toPatient bool
	for _, m := range mails.messages() {
		toPatient = toPatient || m.To == b.email
	}
	if !toPatient {
		t.Errorf("mailed %+v, want the patient among them", mails.messages())
	}
	if failed := failedChatJobs(t, db.owner); failed != nil {
		t.Errorf("jobs failed:\n%s", strings.Join(failed, "\n"))
	}
}

// The patient revokes the care consent with the treating clinic after the message (SAL-29): its fan-out tells nobody,
// the granted guest included.
func TestChatNotifications_TellNobodyOnceTheCareConsentIsRevoked(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	b := seedBooking(t, db.owner)
	r := seedChatNotifications(t, db.owner, b)
	seedChatGuest(t, db.owner, b, r)
	ctx := t.Context()
	log := slog.New(slog.DiscardHandler)
	pract := appointment.Caller{AccountID: r.practAccount, OrganizationID: clinicID, PractitionerID: r.practitioner}
	if _, _, err := sendingChat(t, db, log).Send(ctx, pract, r.conv, "c3000000-0000-4000-8000-0000000000c3", secretChatText); err != nil {
		t.Fatalf("Send() = %v", err)
	}
	if _, err := db.owner.Exec(ctx, `UPDATE consent SET status = 'inactive', revoked_at = now() WHERE patient_id = $1 AND organization_id = $2`, b.patient, clinicID); err != nil {
		t.Fatal(err)
	}
	pusher, mails := startChatJobs(t, db, log)

	eventually(t, "the fan-out", func() bool {
		var state string
		_ = db.owner.QueryRow(ctx, `SELECT state FROM river_job WHERE kind = $1`, communication.MessageSentArgs{}.Kind()).Scan(&state)
		return state == "completed"
	})
	var n int
	if err := db.owner.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind <> $1`, communication.MessageSentArgs{}.Kind()).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 || len(pusher.Sent()) != 0 || len(mails.messages()) != 0 {
		t.Errorf("%d jobs enqueued, %d pushes, %d e-mails after the revocation; want none", n, len(pusher.Sent()), len(mails.messages()))
	}
}
