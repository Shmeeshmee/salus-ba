package worker_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
)

// keptMail keeps what the notifier sends.
type keptMail struct {
	sent []notification.Message
	mu   sync.Mutex
}

func (m *keptMail) Send(_ context.Context, msg notification.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

// warningRequests seeds, as the owner, clinic A with two admins (one writing Croatian) and three requests due in a
// week: one to the clinic, one to the account, and one to the account answered already.
func warningRequests(t *testing.T, owner *pgxpool.Pool) (toClinic, toAccount, answered string) {
	t.Helper()
	ctx := t.Context()
	var account, profile string
	if err := owner.QueryRow(ctx, `WITH o AS (INSERT INTO organization (id, name, time_zone) VALUES ($1, 'Synthetic clinic', 'Europe/Zagreb')),
		a AS (INSERT INTO account (email, password_hash, role, organization_id, language) VALUES
		        ('warn-admin-a@example.test', 'h', 'admin', $1, 'hr'), ('warn-admin-b@example.test', 'h', 'admin', $1, NULL))
		INSERT INTO account (email, password_hash, role) VALUES ('warn-patient@example.test', 'h', 'patient') RETURNING id`, clinicID).Scan(&account); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO patient (account_id, given_name, family_name, birth_date) VALUES ($1, 'Ana', 'Synthetic', '1980-01-01') RETURNING id`, account).Scan(&profile); err != nil {
		t.Fatal(err)
	}
	insert := func(org any, kind string) string {
		var id string
		if err := owner.QueryRow(ctx, `INSERT INTO data_request (account_id, patient_id, organization_id, filed_by_account_id, kind, received_at, due_at)
			VALUES ($1, $2, $3, $1, $4, now() - interval '3 weeks', now() + interval '7 days') RETURNING id`, account, profile, org, kind).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	toClinic, toAccount, answered = insert(clinicID, "erasure"), insert(nil, "access"), insert(nil, "objection")
	if _, err := owner.Exec(ctx, `UPDATE data_request SET status = 'fulfilled', answered_at = now(), answer_code = 'done', version = 2 WHERE id = $1`, answered); err != nil {
		t.Fatal(err)
	}
	return toClinic, toAccount, answered
}

// runWarnings starts the notifier with operator as the operator's address, enqueues the warnings of ids as the deadline
// job does, and returns each job's final state and the log.
func runWarnings(t *testing.T, app *pgxpool.Pool, mail *keptMail, operator string, ids ...string) ([]rivertype.JobState, string) {
	t.Helper()
	var logs bytes.Buffer
	n := testNotifier(mail, &pushtest.Sender{})
	n.OperatorEmail, n.OperatorLanguage = operator, "en"
	client, err := worker.NewNotifier(app, slog.New(slog.NewJSONHandler(&logs, nil)), n)
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
		t.Fatal(err)
	}
	var states []rivertype.JobState
	for _, id := range ids {
		err := postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			return datarequest.NewRiverJobs(inserter).EnqueueWarning(ctx, datarequest.WarningArgs{RequestID: id, Days: datarequest.WarnDays})
		})
		if err != nil {
			t.Fatal(err)
		}
		select {
		case ev := <-events:
			states = append(states, ev.Job.State)
		case <-time.After(15 * time.Second):
			t.Fatal("the data_request_deadline_warning job did not finish within 15s")
		}
	}
	return states, logs.String()
}

// The deadline warnings (docs/clinic-cms/WAVE5.md D5): a clinic target's to each of its admins in their language, an
// account target's to the operator; an answered request completes without a send. Every read is audited, and nothing
// names the patient.
func TestNotifier_SendsTheDataRequestDeadlineWarnings(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	toClinic, toAccount, answered := warningRequests(t, owner)
	mail := &keptMail{}

	states, _ := runWarnings(t, app, mail, "operator@example.test", toClinic, toAccount, answered)
	for i, s := range states {
		if s != rivertype.JobStateCompleted {
			t.Errorf("job %d ended %s, want completed", i, s)
		}
	}
	to := map[string]notification.Message{}
	for _, m := range mail.sent {
		to[m.To] = m
		if strings.Contains(m.Subject+m.Body, "Ana") || strings.Contains(m.Body, "warn-patient") {
			t.Errorf("the warning to %s names the patient:\n%s", m.To, m.Body)
		}
	}
	if len(mail.sent) != 3 {
		t.Fatalf("%d e-mails, want 3 (two admins, the operator)", len(mail.sent))
	}
	if m := to["warn-admin-a@example.test"]; !strings.Contains(m.Body, "Brisanje") || !strings.Contains(m.Body, "Europe/Zagreb") || !strings.Contains(m.Body, "/data-requests/"+toClinic) {
		t.Errorf("the Croatian admin's warning:\n%s", m.Body)
	}
	if m := to["operator@example.test"]; !strings.Contains(m.Body, "Kind: Access") || !strings.Contains(m.Body, "salusctl datarequest show "+toAccount) {
		t.Errorf("the operator's warning:\n%s", m.Body)
	}
	var n int
	if err := owner.QueryRow(t.Context(), `SELECT count(*) FROM audit_event WHERE agent_name = 'data_request_deadline_warning'`).Scan(&n); err != nil || n != 3 {
		t.Errorf("%d audit events of the warnings, %v; want 3 (two requests read, the admins read)", n, err)
	}
}

// In dev without an operator address the account target's warning is logged by its opaque ids and completes.
func TestNotifier_LogsTheOperatorsWarningWithoutAnAddress(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	_, toAccount, _ := warningRequests(t, owner)
	mail := &keptMail{}

	states, logs := runWarnings(t, app, mail, "", toAccount)
	if len(states) != 1 || states[0] != rivertype.JobStateCompleted || len(mail.sent) != 0 {
		t.Fatalf("states %v, %d e-mails; want completed with none", states, len(mail.sent))
	}
	if !strings.Contains(logs, `"request_id":"`+toAccount+`"`) || !strings.Contains(logs, `"days":7`) || strings.Contains(logs, "example.test") {
		t.Errorf("log %s: want the request's id and the days, no address", logs)
	}
}
