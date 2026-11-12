package notification_test

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/pkg/push"
)

// sentArgs is the job of a message the patient sent (sender patientPart): seq 2.
var sentArgs = communication.MessageSentArgs{ConversationID: convID, OrganizationID: clinicID, MessageID: msgID, SenderParticipantID: patientPart, Seq: 2}

// The fan-out tells every other active participant: a push job per device of the patient's account, an unread
// check per participant at the clinic's delay, a delivery per clinic webhook that takes the event. All in one
// transaction; nothing for the sender.
func TestMessageSent_FansOutToTheOtherParticipantsDevicesEmailsAndWebhooks(t *testing.T) {
	f := newChatFixture(t)
	dev1 := f.device(t, patientAccount, "synthetic-token-1", push.Android)
	dev2 := f.device(t, patientAccount, "synthetic-token-2", push.IOS)
	f.device(t, practAccount, "synthetic-staff-token", push.Android)
	f.repo.endpoints["e1000000-0000-4000-8000-0000000000e1"] = notification.Endpoint{ID: "e1000000-0000-4000-8000-0000000000e1", OrganizationID: clinicID, Events: []string{notification.EventMessageCreated}}
	f.repo.endpoints["e2000000-0000-4000-8000-0000000000e2"] = notification.Endpoint{ID: "e2000000-0000-4000-8000-0000000000e2", OrganizationID: "other", Events: []string{notification.EventMessageCreated}}

	practSent := sentArgs
	practSent.SenderParticipantID = practPart
	if err := f.svc.MessageSent(t.Context(), practSent); err != nil {
		t.Fatalf("MessageSent() = %v", err)
	}
	wantPushes := []notification.ChatPushArgs{{DeviceID: dev1, AccountID: patientAccount}, {DeviceID: dev2, AccountID: patientAccount}}
	if !slices.Equal(f.jobs.pushes, wantPushes) {
		t.Errorf("pushes %+v, want the patient's two devices, each with the account", f.jobs.pushes)
	}
	want := notification.ChatUnreadEmailArgs{ParticipantID: patientPart, ConversationID: convID, OrganizationID: clinicID, ReadSeq: 0, Seq: 2}
	if len(f.jobs.emails) != 1 || f.jobs.emails[0] != want || !f.jobs.emailAt[0].Equal(chatNow.Add(30*time.Minute)) {
		t.Errorf("e-mail checks %+v at %v, want %+v in 30 minutes", f.jobs.emails, f.jobs.emailAt, want)
	}
	if len(f.jobs.webhooks) != 1 || f.jobs.webhooks[0].EndpointID != "e1000000-0000-4000-8000-0000000000e1" || f.jobs.webhooks[0].MessageID != msgID ||
		f.jobs.webhooks[0].ConversationID != convID || f.jobs.webhooks[0].OrganizationID != clinicID {
		t.Errorf("webhooks %+v, want clinic A's one", f.jobs.webhooks)
	}
	if f.jobs.outside != 0 {
		t.Errorf("%d jobs enqueued outside the transaction", f.jobs.outside)
	}

	// The patient's own message: only the practitioner's unread check (staff register no devices).
	*f.jobs = fakeChatJobs{}
	if err := f.svc.MessageSent(t.Context(), sentArgs); err != nil {
		t.Fatal(err)
	}
	if len(f.jobs.pushes) != 0 || len(f.jobs.emails) != 1 || f.jobs.emails[0].ParticipantID != practPart || f.jobs.emails[0].ReadSeq != 1 {
		t.Errorf("the patient's message: pushes %+v, checks %+v; want the practitioner's check only", f.jobs.pushes, f.jobs.emails)
	}
}

// The event id is the same however often the fan-out runs (a retried job), and differs per message.
func TestMessageSent_TheWebhookEventIDIsStablePerMessage(t *testing.T) {
	f := newChatFixture(t)
	f.repo.endpoints["e1000000-0000-4000-8000-0000000000e1"] = notification.Endpoint{ID: "e1000000-0000-4000-8000-0000000000e1", OrganizationID: clinicID, Events: []string{notification.EventMessageCreated}}
	other := sentArgs
	other.MessageID = "e9000000-0000-4000-8000-0000000000e9"
	for _, a := range []communication.MessageSentArgs{sentArgs, sentArgs, other} {
		if err := f.svc.MessageSent(t.Context(), a); err != nil {
			t.Fatal(err)
		}
	}
	w := f.jobs.webhooks
	if len(w) != 3 || w[0].EventID != w[1].EventID || w[0].EventID == w[2].EventID || len(w[0].EventID) != 36 || w[0].EventID[14] != '8' {
		t.Errorf("event ids %q %q %q, want one stable UUID (version 8) per message", w[0].EventID, w[1].EventID, w[2].EventID)
	}
}

// A conversation the clinic no longer sees (the patient revoked the care consent) tells nobody; a failure of a read
// or an enqueue is the job's error, for River to retry.
func TestMessageSent_NothingAfterARevocationAndFailuresAreRetried(t *testing.T) {
	f := newChatFixture(t)
	f.chats.err = communication.ErrNotFound
	if err := f.svc.MessageSent(t.Context(), sentArgs); err != nil || len(f.jobs.emails)+len(f.jobs.pushes) != 0 {
		t.Errorf("after a revocation: %v, %d jobs; want nothing", err, len(f.jobs.emails)+len(f.jobs.pushes))
	}
	down := errors.New("the database is down")
	f = newChatFixture(t)
	f.jobs.err = down
	if err := f.svc.MessageSent(t.Context(), sentArgs); !errors.Is(err, down) {
		t.Errorf("enqueue fails: %v", err)
	}
}

// A push says only that there is a new message from the clinic, in the account's language: no text, no name, no id.
// A token the platform says is gone is forgotten; a device gone meanwhile sends nothing.
func TestPush_SendsTheFixedAlertInTheAccountsLanguageAndForgetsAGoneToken(t *testing.T) {
	f := newChatFixture(t)
	dev := f.device(t, patientAccount, "synthetic-token-1", push.IOS)
	if err := f.svc.Push(t.Context(), notification.ChatPushArgs{DeviceID: dev, AccountID: patientAccount}); err != nil {
		t.Fatalf("Push() = %v", err)
	}
	sent := f.pusher.Sent()
	want := push.Message{Token: "synthetic-token-1", Platform: push.IOS, Title: "Salus", Body: "Neue Nachricht von Ihrer Praxis"}
	if len(sent) != 1 || sent[0] != want {
		t.Errorf("pushed %+v, want %+v", sent, want)
	}
	if e := f.audit.events[len(f.audit.events)-1]; e.AgentName != "chat_push" || e.EntityReference != "Account/"+patientAccount || e.EntityName != "push_device" {
		t.Errorf("audit %+v, want the device's read by the job", e)
	}

	f.pusher.Gone("synthetic-token-1")
	if err := f.svc.Push(t.Context(), notification.ChatPushArgs{DeviceID: dev, AccountID: patientAccount}); err != nil {
		t.Fatalf("Push(gone) = %v", err)
	}
	if _, ok := f.repo.devices[dev]; ok {
		t.Error("a gone token was kept")
	}
	if err := f.svc.Push(t.Context(), notification.ChatPushArgs{DeviceID: dev, AccountID: patientAccount}); err != nil {
		t.Errorf("Push(unregistered) = %v, want nothing to do", err)
	}

	down := errors.New("push service down")
	f.pusher.Err = down
	dev = f.device(t, patientAccount, "synthetic-token-2", push.Android)
	if err := f.svc.Push(t.Context(), notification.ChatPushArgs{DeviceID: dev, AccountID: patientAccount}); !errors.Is(err, down) || strings.Contains(err.Error(), "synthetic-token") {
		t.Errorf("Push(down) = %v, want its error without the token", err)
	}
}

// A push queued for one account sends nothing once its device's token moved to another account (signed out of the
// first, into the second): the second must never learn of the first one's messages.
func TestPush_SendsNothingOnceTheTokenMovedToAnotherAccount(t *testing.T) {
	f := newChatFixture(t)
	dev := f.device(t, patientAccount, "synthetic-token-1", push.IOS)
	if moved := f.device(t, practAccount, "synthetic-token-1", push.IOS); moved != dev {
		t.Fatalf("the fake moved the token to a new row %s, want %s", moved, dev)
	}
	if err := f.svc.Push(t.Context(), notification.ChatPushArgs{DeviceID: dev, AccountID: patientAccount}); err != nil {
		t.Fatalf("Push() = %v", err)
	}
	if sent := f.pusher.Sent(); len(sent) != 0 {
		t.Errorf("pushed %+v to the token's new account", sent)
	}
}

// The e-mail of an unread message goes when the participant has not read up to it: to the patient in their
// language with the app's link, to the practitioner in theirs with the web app's; it names nobody and holds no
// text. A participant who has read it, or left, gets nothing.
func TestUnreadEmail_MailsAParticipantWhoHasNotReadTheMessage(t *testing.T) {
	f := newChatFixture(t)
	if err := f.svc.UnreadEmail(t.Context(), notification.ChatUnreadEmailArgs{ParticipantID: patientPart, ConversationID: convID, OrganizationID: clinicID, Seq: 2}); err != nil {
		t.Fatalf("UnreadEmail(patient) = %v", err)
	}
	if err := f.svc.UnreadEmail(t.Context(), notification.ChatUnreadEmailArgs{ParticipantID: practPart, ConversationID: convID, OrganizationID: clinicID, Seq: 2}); err != nil {
		t.Fatalf("UnreadEmail(practitioner) = %v", err)
	}
	if len(f.mail.sent) != 2 {
		t.Fatalf("sent %+v, want two e-mails", f.mail.sent)
	}
	patientMail, practMail := f.mail.sent[0], f.mail.sent[1]
	if patientMail.To != "ana@example.test" || patientMail.Subject != "Neue Nachricht von Ihrer Praxis" ||
		patientMail.Body != "Sie haben eine neue Nachricht von Ihrer Praxis.\n\nUm sie zu lesen, öffnen Sie die Salus-App:\nhttps://patient.example.test/chat\n" {
		t.Errorf("the patient's e-mail %+v", patientMail)
	}
	if practMail.To != "dr.example@example.test" || practMail.Subject != "Nova poruka u chatu u aplikaciji Salus" ||
		!strings.HasSuffix(practMail.Body, "https://app.example.test/chat\n") {
		t.Errorf("the practitioner's e-mail %+v", practMail)
	}
	for _, m := range f.mail.sent {
		for _, never := range []string{"7f3a", patientName, practitionerNam, convID, msgID} {
			if strings.Contains(m.Subject+m.Body, never) {
				t.Errorf("an e-mail holds %q: %+v", never, m)
			}
		}
	}

	f = newChatFixture(t)
	f.chats.parts[1].ReadSeq = 2 // the practitioner read to the end
	for name, a := range map[string]notification.ChatUnreadEmailArgs{
		"read":            {ParticipantID: practPart, ConversationID: convID, OrganizationID: clinicID, ReadSeq: 1, Seq: 2},
		"left":            {ParticipantID: "f9000000-0000-4000-8000-0000000000f9", ConversationID: convID, OrganizationID: clinicID, Seq: 2},
		"revoked or gone": {ParticipantID: patientPart, ConversationID: "c9000000-0000-4000-8000-0000000000c9", OrganizationID: clinicID, Seq: 2},
	} {
		if err := f.svc.UnreadEmail(t.Context(), a); err != nil || len(f.mail.sent)+len(f.jobs.emails) != 0 {
			t.Errorf("%s: %v, sent %d, checks %d; want nothing", name, err, len(f.mail.sent), len(f.jobs.emails))
		}
	}
}

// A check that finds its message read, while a later one of the same burst is not (read on part of the way, the
// later message's check kept as this one by the unique key), mails nothing yet but checks again at the clinic's
// delay from the new read position: one later message is never left without its e-mail. Read to the end, nothing.
func TestUnreadEmail_ChecksAgainWhenALaterMessageIsStillUnread(t *testing.T) {
	f := newChatFixture(t)
	f.chats.lastSeq, f.chats.parts[1].ReadSeq = 3, 2
	a := notification.ChatUnreadEmailArgs{ParticipantID: practPart, ConversationID: convID, OrganizationID: clinicID, ReadSeq: 1, Seq: 2}
	if err := f.svc.UnreadEmail(t.Context(), a); err != nil {
		t.Fatalf("UnreadEmail() = %v", err)
	}
	want := notification.ChatUnreadEmailArgs{ParticipantID: practPart, ConversationID: convID, OrganizationID: clinicID, ReadSeq: 2, Seq: 3}
	if len(f.mail.sent) != 0 || len(f.jobs.emails) != 1 || f.jobs.emails[0] != want || !f.jobs.emailAt[0].Equal(chatNow.Add(30*time.Minute)) || f.jobs.outside != 0 {
		t.Errorf("sent %d, checks %+v at %v (%d outside a transaction); want none sent and %+v in 30 minutes",
			len(f.mail.sent), f.jobs.emails, f.jobs.emailAt, f.jobs.outside, want)
	}

	f = newChatFixture(t)
	f.chats.lastSeq, f.chats.parts[1].ReadSeq = 3, 3
	if err := f.svc.UnreadEmail(t.Context(), a); err != nil || len(f.mail.sent)+len(f.jobs.emails) != 0 {
		t.Errorf("read to the end: %v, sent %d, checks %+v; want nothing", err, len(f.mail.sent), f.jobs.emails)
	}
}

// A recipient whose account cannot be found (a practitioner's role or account gone) gets nothing, and the job completes instead of retrying what cannot change.
func TestUnreadEmail_CompletesWhenTheRecipientHasNoAccount(t *testing.T) {
	f := newChatFixture(t)
	f.chats.parts[1].PractitionerRoleID = "b9000000-0000-4000-8000-0000000000b9" // no such role in the clinic
	if err := f.svc.UnreadEmail(t.Context(), notification.ChatUnreadEmailArgs{ParticipantID: practPart, ConversationID: convID, OrganizationID: clinicID, Seq: 2}); err != nil {
		t.Errorf("UnreadEmail(no role) = %v, want nil", err)
	}
	f.chats.parts[0].PatientID = "d9000000-0000-4000-8000-0000000000d9" // not seen in the clinic's scope
	if err := f.svc.UnreadEmail(t.Context(), notification.ChatUnreadEmailArgs{ParticipantID: patientPart, ConversationID: convID, OrganizationID: clinicID, Seq: 2}); err != nil {
		t.Errorf("UnreadEmail(no patient) = %v, want nil", err)
	}
	if len(f.mail.sent) != 0 {
		t.Errorf("sent %+v", f.mail.sent)
	}
}

// A doctor of another clinic the patient granted into the conversation (SAL-28; NoticeConversation gives them only
// while their chat consent and the care consent are active) is e-mailed like the clinic's own: their role and
// account are looked up in their own clinic, not in the conversation's.
func TestUnreadEmail_MailsAGrantedGuestThroughTheirOwnClinic(t *testing.T) {
	f := newChatFixture(t)
	f.chats.parts = append(f.chats.parts, communication.Participant{ID: guestPart, ConversationID: convID, PractitionerRoleID: guestRole,
		OrganizationID: guestClinic, Role: communication.RolePractitioner, Status: communication.StatusActive, ReadSeq: 1})
	if err := f.svc.UnreadEmail(t.Context(), notification.ChatUnreadEmailArgs{ParticipantID: guestPart, ConversationID: convID, OrganizationID: clinicID, ReadSeq: 1, Seq: 2}); err != nil {
		t.Fatalf("UnreadEmail(guest) = %v", err)
	}
	if len(f.mail.sent) != 1 || f.mail.sent[0].To != "guest.example@example.test" || !strings.HasSuffix(f.mail.sent[0].Body, "https://app.example.test/chat\n") {
		t.Errorf("sent %+v, want the guest's e-mail with the web app's link", f.mail.sent)
	}
}

// What every job of a message carries, serialised as River stores it, holds no text and no name.
func TestJobArguments_HoldIDsOnly(t *testing.T) {
	for _, a := range []any{
		notification.ChatPushArgs{DeviceID: "9d000000-0000-4000-8000-000000000001", AccountID: patientAccount},
		notification.ChatUnreadEmailArgs{ParticipantID: patientPart, ConversationID: convID, OrganizationID: clinicID, Seq: 2},
		notification.WebhookDeliveryArgs{EndpointID: msgID, EventID: msgID, OrganizationID: clinicID, ConversationID: convID, MessageID: msgID},
	} {
		b, _ := json.Marshal(a)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		for k, v := range m {
			if s, ok := v.(string); ok && len(s) != 36 {
				t.Errorf("%T.%s = %q, want an id", a, k, s)
			}
		}
	}
}
