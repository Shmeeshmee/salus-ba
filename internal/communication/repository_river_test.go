package communication_test

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// chat_message_sent goes on the notifier's push queue, in the caller's transaction, with the ids only.
func TestRiverJobs_EnqueueTheMessageSentJobOnThePushQueueInTheTransaction(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := communication.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()
	a := communication.MessageSentArgs{ConversationID: clinicA, OrganizationID: clinicA, MessageID: patientA, SenderParticipantID: roleA1, Seq: 7}

	errRollBack := errors.New("the send failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := jobs.EnqueueMessageSent(ctx, a); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rolled-back send: %d jobs (%v, %v), want none", n, err, qerr)
	}
	if err := tr.WithTx(ctx, func(ctx context.Context) error { return jobs.EnqueueMessageSent(ctx, a) }); err != nil {
		t.Fatal(err)
	}
	var kind, queue, args string
	var max int
	if err := owner.QueryRow(ctx, "SELECT kind, queue, args::text, max_attempts FROM river_job").Scan(&kind, &queue, &args, &max); err != nil {
		t.Fatal(err)
	}
	want := `{"seq": 7, "message_id": "` + patientA + `", "conversation_id": "` + clinicA + `", "organization_id": "` + clinicA + `", "sender_participant_id": "` + roleA1 + `"}`
	if kind != "chat_message_sent" || queue != "push" || args != want || max != 10 {
		t.Errorf("job %s on %s with %s (%d attempts); want chat_message_sent on push with %s", kind, queue, args, max, want)
	}
	if err := jobs.EnqueueMessageSent(ctx, a); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("outside a transaction = %v, want postgres.ErrNoTx", err)
	}
}
