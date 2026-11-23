package worker_test

import (
	"context"
	"crypto/rand"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/Shmeeshmee/salus/internal/signup"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
)

// The signup_email job (docs/clinic-cms, O1) issues a new verification token and sends the link; a signup verified
// meanwhile has nothing to send, and its job is cancelled.
func TestNotifier_SendsTheSignupVerificationEmail(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	email := "signup-" + strings.ToLower(rand.Text()) + "@example.test"
	var id, verifiedID string
	err := owner.QueryRow(t.Context(), `INSERT INTO clinic_signup (email, clinic_name, language, token_hash)
		VALUES ($1, 'Synthetic signup clinic', 'en', sha256('one')) RETURNING id`, email).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	err = owner.QueryRow(t.Context(), `WITH o AS (INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id)
		INSERT INTO clinic_signup (email, clinic_name, language, token_hash, verified_at, organization_id)
		SELECT 'verified@example.test', 'Synthetic clinic', 'en', sha256('two'), now(), id FROM o RETURNING id`).Scan(&verifiedID)
	if err != nil {
		t.Fatal(err)
	}
	client, err := worker.NewNotifier(app, slog.New(slog.DiscardHandler), testNotifier(mailpitSender(t), &pushtest.Sender{}))
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
	states := map[string]rivertype.JobState{}
	for _, sid := range []string{id, verifiedID} {
		err = postgres.NewTransactor(app).WithTx(t.Context(), func(ctx context.Context) error {
			return signup.NewRiverJobs(inserter).EnqueueEmail(ctx, sid)
		})
		if err != nil {
			t.Fatal(err)
		}
		select {
		case ev := <-events:
			states[sid] = ev.Job.State
		case <-time.After(15 * time.Second):
			t.Fatal("the signup_email job did not finish within 15s")
		}
	}
	if states[id] != rivertype.JobStateCompleted || states[verifiedID] != rivertype.JobStateCancelled {
		t.Fatalf("job states %v, want the open signup's completed and the verified one's cancelled", states)
	}
	var replaced bool
	if err := owner.QueryRow(t.Context(), "SELECT token_hash <> sha256('one') FROM clinic_signup WHERE id = $1", id).Scan(&replaced); err != nil || !replaced {
		t.Errorf("token replaced: %v (%v), want a new hash", replaced, err)
	}
	text := mailpitText(t, email)
	if !strings.Contains(text, "Synthetic signup clinic") || !strings.Contains(text, "https://app.example.test/signup/verify#") {
		t.Errorf("the e-mail lacks the clinic name or the link:\n%s", text)
	}
}
