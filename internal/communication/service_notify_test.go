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

// The notifier reads a conversation's active participants, their read positions and its last message's number in
// the clinic's scope, audited with the job as agent; the clinic sees nothing after the patient's revocation.
func TestNoticeConversation_ReadsTheParticipantsInTheClinicsScopeAuditedWithTheJob(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	f.send(t, pract, c.ID, "Synthetic placeholder")
	f.send(t, pract, c.ID, "Synthetic placeholder")
	if err := f.svc.MarkRead(t.Context(), me, c.ID, 1); err != nil {
		t.Fatal(err)
	}
	conv, err := f.svc.NoticeConversation(t.Context(), "chat_message_sent", clinicA, c.ID)
	if err != nil || len(conv.Participants) != 2 || conv.LastSeq != 2 || conv.ID != c.ID {
		t.Fatalf("NoticeConversation() = %+v, %v; want two participants and the last number 2", conv, err)
	}
	reads := map[communication.Role]int64{}
	for _, p := range conv.Participants {
		reads[p.Role] = p.ReadSeq
	}
	if reads[communication.RolePatient] != 1 || reads[communication.RolePractitioner] != 2 {
		t.Errorf("read positions %v, want the patient at 1 and the sender at 2", reads)
	}
	e := f.audit.last(t)
	if e.AgentName != "chat_message_sent" || e.AgentAccountID != "" || e.EntityReference != "Patient/"+patientA ||
		e.EntityName != "conversation_participant" || e.EntityCount != 2 || e.Action != auditevent.ActionRead || e.OrganizationID != clinicA {
		t.Errorf("audit %+v", e)
	}
	if _, err := f.svc.NoticeConversation(t.Context(), "chat_message_sent", clinicB, c.ID); !errors.Is(err, communication.ErrNotFound) {
		t.Errorf("another clinic: %v, want ErrNotFound", err)
	}
	f.revoke()
	if _, err := f.svc.NoticeConversation(t.Context(), "chat_message_sent", clinicA, c.ID); !errors.Is(err, communication.ErrNotFound) {
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

// noticed is the participants, by role id ("patient" for the patient), that NoticeConversation gives the notifier for the conversation id of clinic A.
func noticed(t *testing.T, f *fixture, id string) map[string]communication.Participant {
	t.Helper()
	conv, err := f.svc.NoticeConversation(t.Context(), "chat_message_sent", clinicA, id)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]communication.Participant{}
	for _, p := range conv.Participants {
		key := p.PractitionerRoleID
		if p.Role == communication.RolePatient {
			key = "patient"
		}
		out[key] = p
	}
	return out
}

// Only those who may read the conversation now are told of a message (SAL-28's rules): a doctor of another clinic
// waiting for the patient's consent is not, nor one whose chat consent is no longer active; a granted one is.
func TestNoticeConversation_GivesOnlyActiveParticipantsAndGrantedGuests(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	invited := f.add(t, c.ID, roleB1, clinicB) // pending the patient's consent
	if got := noticed(t, f, c.ID); len(got) != 2 || got[roleB1].ID != "" {
		t.Errorf("with a pending invitation: %+v, want the patient and the treating practitioner only", got)
	}
	if err := f.svc.DecideInvitation(t.Context(), me, invited.ID, communication.DecisionGrant); err != nil {
		t.Fatal(err)
	}
	if got := noticed(t, f, c.ID); len(got) != 3 || got[roleB1].Status != communication.StatusActive {
		t.Errorf("after the grant: %+v, want the guest too", got)
	}
	f.mu.Lock()
	f.chat[c.ID+roleB1] = false // the chat consent ended while the participant row is still active
	f.mu.Unlock()
	if got := noticed(t, f, c.ID); len(got) != 2 || got[roleB1].ID != "" {
		t.Errorf("without the chat consent: %+v, want the guest left out", got)
	}
}

// A side thread is the doctors' only: its notices never name the patient, whatever its rows say.
func TestNoticeConversation_ASideThreadNeverNotifiesThePatient(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.open(t)
	f.add(t, c.ID, roleA2, clinicA)
	thread, err := f.svc.OpenThread(t.Context(), pract, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.repo.mu.Lock()
	f.repo.parts[thread.ID] = append(f.repo.parts[thread.ID], communication.Participant{
		ID: "f1000000-0000-4000-8000-0000000000f1", ConversationID: thread.ID, PatientID: patientA, Role: communication.RolePatient,
		Status: communication.StatusActive,
	})
	f.repo.mu.Unlock()
	if got := noticed(t, f, thread.ID); len(got) != 2 || got["patient"].ID != "" {
		t.Errorf("the side thread's notices: %+v, want its two doctors only", got)
	}
}
