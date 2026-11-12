package communication_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/communication"
)

// fakeJobs records the jobs enqueued, and whether each was enqueued inside a transaction; the fake transaction
// drops those of a transaction that rolls back.
type fakeJobs struct {
	err  error
	sent []communication.MessageSentArgs
	inTx []bool
	mu   sync.Mutex
}

func (j *fakeJobs) EnqueueMessageSent(ctx context.Context, a communication.MessageSentArgs) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.err != nil {
		return j.err
	}
	j.sent = append(j.sent, a)
	j.inTx = append(j.inTx, ctx.Value(txKey{}) != nil)
	return nil
}

func (j *fakeJobs) count() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return len(j.sent)
}

func (j *fakeJobs) truncate(n int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.sent, j.inTx = j.sent[:n], j.inTx[:n]
}

// secretText is a recognisable synthetic message: it must appear in no job argument (CLAUDE.md "Background work").
const secretText = "Synthetic secret text 7f3a: never in a notification"

// A new message enqueues one chat_message_sent job in the send's transaction, with opaque ids only (SAL-29); a
// repeat of the client's id enqueues nothing.
func TestSend_EnqueuesOneNotificationJobWithIDsOnly(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	m, _, err := f.svc.Send(t.Context(), me, c.ID, clientID1, secretText)
	if err != nil {
		t.Fatal(err)
	}
	if _, created, err := f.svc.Send(t.Context(), me, c.ID, clientID1, secretText); err != nil || created {
		t.Fatalf("a repeat: %v, created %v", err, created)
	}
	if f.jobs.count() != 1 || !f.jobs.inTx[0] {
		t.Fatalf("jobs %+v in a transaction %v, want one, in the send's", f.jobs.sent, f.jobs.inTx)
	}
	want := communication.MessageSentArgs{ConversationID: c.ID, OrganizationID: clinicA, MessageID: m.ID, SenderParticipantID: m.SenderParticipantID, Seq: 1}
	if f.jobs.sent[0] != want {
		t.Errorf("job %+v, want %+v", f.jobs.sent[0], want)
	}
	b, err := json.Marshal(f.jobs.sent[0])
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	_ = json.Unmarshal(b, &keys)
	if strings.Contains(string(b), "7f3a") || len(keys) != 5 || keys["seq"] != float64(1) {
		t.Errorf("the job's arguments %s, want the four ids and the number only", b)
	}
	if f.jobs.sent[0].Kind() != "chat_message_sent" {
		t.Errorf("kind %s", f.jobs.sent[0].Kind())
	}
}

// The job is part of the send: a failed enqueue rolls the send back, unpublished and unaudited.
func TestSend_TheJobCommitsWithTheMessageOrNotAtAll(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	events := f.audit.count()
	f.jobs.err = errBackend
	if _, _, err := f.svc.Send(t.Context(), me, c.ID, clientID1, "Synthetic placeholder"); !errors.Is(err, errBackend) {
		t.Fatalf("Send with the queue down = %v, want its error", err)
	}
	if len(f.pub.published()) != 0 || f.repo.convs[c.ID].LastSeq != 0 || f.audit.count() != events {
		t.Error("a send whose job failed was stored, published or audited")
	}
}

// The notifier reads a conversation's active participants and their read positions in the clinic's scope, audited
// with the job as agent; the clinic sees nothing after the patient's revocation.
func TestNoticeParticipants_ReadsTheParticipantsInTheClinicsScopeAuditedWithTheJob(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	f.send(t, pract, c.ID, "Synthetic placeholder")
	if err := f.svc.MarkRead(t.Context(), me, c.ID, 1); err != nil {
		t.Fatal(err)
	}
	parts, err := f.svc.NoticeParticipants(t.Context(), "chat_message_sent", clinicA, c.ID)
	if err != nil || len(parts) != 2 {
		t.Fatalf("NoticeParticipants() = %+v, %v", parts, err)
	}
	reads := map[communication.Role]int64{}
	for _, p := range parts {
		reads[p.Role] = p.ReadSeq
	}
	if reads[communication.RolePatient] != 1 || reads[communication.RolePractitioner] != 1 {
		t.Errorf("read positions %v, want both at 1", reads)
	}
	e := f.audit.last(t)
	if e.AgentName != "chat_message_sent" || e.AgentAccountID != "" || e.EntityReference != "Patient/"+patientA ||
		e.EntityName != "conversation_participant" || e.EntityCount != 2 || e.Action != auditevent.ActionRead || e.OrganizationID != clinicA {
		t.Errorf("audit %+v", e)
	}
	if _, err := f.svc.NoticeParticipants(t.Context(), "chat_message_sent", clinicB, c.ID); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("another clinic: %v, want ErrNotFound", err)
	}
	f.revoke()
	if _, err := f.svc.NoticeParticipants(t.Context(), "chat_message_sent", clinicA, c.ID); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("after the revocation: %v, want ErrNotFound", err)
	}
}

// A clinic's webhook with clinical content reads the message in the clinic's scope, audited with the job as agent.
func TestNoticeMessage_ReadsTheMessageInTheClinicsScopeAuditedWithTheJob(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	m := f.send(t, me, c.ID, secretText)
	got, err := f.svc.NoticeMessage(t.Context(), "webhook_delivery", clinicA, c.ID, m.ID)
	if err != nil || got.Body != secretText || got.ID != m.ID {
		t.Fatalf("NoticeMessage() = %+v, %v", got, err)
	}
	if e := f.audit.last(t); e.AgentName != "webhook_delivery" || e.EntityReference != "Communication/"+m.ID || e.Action != auditevent.ActionRead {
		t.Errorf("audit %+v", e)
	}
	for name, ids := range map[string][3]string{
		"another clinic":  {clinicB, c.ID, m.ID},
		"no such message": {clinicA, c.ID, unknownID},
		"not a UUID":      {clinicA, "x", m.ID},
	} {
		if _, err := f.svc.NoticeMessage(t.Context(), "webhook_delivery", ids[0], ids[1], ids[2]); !errors.Is(err, communication.ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
	f.revoke()
	if _, err := f.svc.NoticeMessage(t.Context(), "webhook_delivery", clinicA, c.ID, m.ID); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("after the revocation: %v, want ErrNotFound", err)
	}
}
