package notification_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

type riverJob struct {
	at                time.Time
	kind, queue, args string
	attempts          int
}

// The chat's notification jobs, each on its queue in the caller's transaction with ids only: a second unread check
// of a participant at the same read position is not enqueued (one after they read on is), nor a second delivery of
// an event to an endpoint.
func TestRiverJobs_EnqueueTheNotificationsOnTheirQueuesOnceEach(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	client, err := river.NewClient(riverpgxv5.New(app), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	jobs := notification.NewRiverJobs(client)
	tr := postgres.NewTransactor(app)
	ctx := t.Context()
	at := time.Date(2030, 1, 1, 10, 30, 0, 0, time.UTC)
	unread := notification.ChatUnreadEmailArgs{ParticipantID: patientPart, ConversationID: convID, OrganizationID: clinicID, Seq: 2}
	hook := notification.WebhookDeliveryArgs{EndpointID: msgID, EventID: convID, OrganizationID: clinicID, ConversationID: convID, MessageID: msgID}
	enqueue := func(ctx context.Context) error {
		if err := jobs.EnqueuePush(ctx, notification.ChatPushArgs{DeviceID: msgID, AccountID: patientAccount}); err != nil {
			return err
		}
		if err := jobs.EnqueueUnreadEmail(ctx, unread, at); err != nil {
			return err
		}
		later := unread // the next message of the burst: the participant has not read on
		later.Seq = 3
		if err := jobs.EnqueueUnreadEmail(ctx, later, at.Add(time.Minute)); err != nil {
			return err
		}
		readOn := unread // a message after the participant read up to 3: a check of its own
		readOn.ReadSeq, readOn.Seq = 3, 4
		if err := jobs.EnqueueUnreadEmail(ctx, readOn, at.Add(2*time.Minute)); err != nil {
			return err
		}
		if err := jobs.EnqueueWebhook(ctx, hook); err != nil {
			return err
		}
		return jobs.EnqueueWebhook(ctx, hook)
	}
	errRollBack := errors.New("the fan-out failed")
	err = tr.WithTx(ctx, func(ctx context.Context) error {
		if err := enqueue(ctx); err != nil {
			return err
		}
		return errRollBack
	})
	var n int
	if qerr := owner.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n); !errors.Is(err, errRollBack) || qerr != nil || n != 0 {
		t.Fatalf("after a rollback: %d jobs (%v, %v)", n, err, qerr)
	}
	if err := tr.WithTx(ctx, enqueue); err != nil {
		t.Fatal(err)
	}
	rows, err := owner.Query(ctx, "SELECT kind, queue, args::text, max_attempts, scheduled_at FROM river_job ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []riverJob
	for rows.Next() {
		var j riverJob
		if err := rows.Scan(&j.kind, &j.queue, &j.args, &j.attempts, &j.at); err != nil {
			t.Fatal(err)
		}
		got = append(got, j)
	}
	want := []riverJob{
		{kind: "chat_push", queue: "push", attempts: 5, args: `{"device_id": "` + msgID + `", "account_id": "` + patientAccount + `"}`},
		{kind: "chat_unread_email", queue: "email", attempts: 10, at: at,
			args: `{"seq": 2, "read_seq": 0, "participant_id": "` + patientPart + `", "conversation_id": "` + convID + `", "organization_id": "` + clinicID + `"}`},
		{kind: "chat_unread_email", queue: "email", attempts: 10, at: at.Add(2 * time.Minute),
			args: `{"seq": 4, "read_seq": 3, "participant_id": "` + patientPart + `", "conversation_id": "` + convID + `", "organization_id": "` + clinicID + `"}`},
		{kind: "webhook_delivery", queue: "webhook", attempts: 8,
			args: `{"event_id": "` + convID + `", "message_id": "` + msgID + `", "endpoint_id": "` + msgID + `", "conversation_id": "` + convID + `", "organization_id": "` + clinicID + `"}`},
	}
	if len(got) != len(want) {
		t.Fatalf("jobs %+v, want %+v", got, want)
	}
	for i := range want {
		g := got[i]
		if g.kind != want[i].kind || g.queue != want[i].queue || g.args != want[i].args || g.attempts != want[i].attempts || (!want[i].at.IsZero() && !g.at.Equal(want[i].at)) {
			t.Errorf("job %d = %+v,\nwant %+v", i, g, want[i])
		}
	}
	if err := jobs.EnqueuePush(ctx, notification.ChatPushArgs{DeviceID: msgID}); !errors.Is(err, postgres.ErrNoTx) {
		t.Errorf("outside a transaction = %v", err)
	}
}
