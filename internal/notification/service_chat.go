package notification

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/push"
)

// Transactor runs fn in a transaction carried by the context, joining one already there (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Chats reads a conversation for the notifications, in the clinic's scope, audited with the job as agent
// (communication.Service).
type Chats interface {
	NoticeConversation(ctx context.Context, agent, org, id string) (communication.Conversation, error)
	NoticeMessage(ctx context.Context, agent, org, id, messageID string) (communication.Message, error)
}

// Patients gives a patient's account in the clinic's scope, audited with the job as agent (patient.Service).
type Patients interface {
	NoticeAccount(ctx context.Context, agent, org, id string) (string, error)
}

// Roles gives a practitioner role of the clinic (practitionerrole.Service).
type Roles interface {
	Lookup(ctx context.Context, org, id string) (practitionerrole.PractitionerRole, error)
}

// ChatAccounts gives the accounts' e-mails and languages, audited with the job as agent (auth.Service).
type ChatAccounts interface {
	AccountEmail(ctx context.Context, agent, id string) (email, language string, err error)
	AccountLanguage(ctx context.Context, agent, id string) (string, error)
	PractitionerAccount(ctx context.Context, agent, org, practitionerID string) (id, email, language string, err error)
}

// ChatClinics gives the clinic's delay of the unread e-mail (organization.Chat).
type ChatClinics interface {
	EmailAfter(ctx context.Context, org string) (time.Duration, error)
}

// ChatPorts are what Chat runs on, and the bases of its links. A binary that calls only some methods leaves the
// others' ports nil: the api registers devices and manages webhooks, the notifier fans out, pushes and mails, the
// worker delivers webhooks.
type ChatPorts struct {
	Repo     Repository
	Jobs     Jobs
	Chats    Chats
	Patients Patients
	Roles    Roles
	Accounts ChatAccounts
	Clinics  ChatClinics
	Pusher   push.Sender
	Sender   Sender
	Poster   Poster
	Box      SecretBox
	Audit    Auditor
	Tx       Transactor
	Clock    clock.Clock
	// WebBaseURL is WEB_BASE_URL: a practitioner's e-mail links to <WebBaseURL>/chat.
	WebBaseURL string
	// PatientAppURL is PATIENT_APP_URL: a patient's e-mail links to <PatientAppURL>/chat.
	PatientAppURL string
}

// Chat holds the chat's notifications (docs/plans/chat.md, SAL-29): who is told of a new message and how, the
// devices that receive pushes, and the clinic's webhooks. Nothing it sends or enqueues holds a message's text, a
// sender or a name; a webhook holds the text only when the clinic's admin turned clinical content on. Every read
// of a person's data is audited, with the job as agent for a job's.
type Chat struct {
	letters map[string]letters
	ChatPorts
}

// NewChat returns Chat on its ports.
func NewChat(p ChatPorts) *Chat {
	p.WebBaseURL, p.PatientAppURL = strings.TrimRight(p.WebBaseURL, "/"), strings.TrimRight(p.PatientAppURL, "/")
	return &Chat{ChatPorts: p, letters: loadLetters()}
}

// The agents of the jobs' reads: their kinds.
var (
	agentSent   = communication.MessageSentArgs{}.Kind()
	agentPush   = ChatPushArgs{}.Kind()
	agentUnread = ChatUnreadEmailArgs{}.Kind()
)

// MessageSent fans a new message out (the chat_message_sent job): for every other active participant an unread
// check at the clinic's e-mail delay and, for the patient, a push per device of their account; for every webhook
// of the clinic that takes the event, a delivery with the event's stable id. In one transaction, so a retry
// enqueues all or nothing again. A conversation the clinic no longer sees (the patient revoked the care consent)
// tells nobody.
func (s *Chat) MessageSent(ctx context.Context, a communication.MessageSentArgs) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		conv, err := s.Chats.NoticeConversation(ctx, agentSent, a.OrganizationID, a.ConversationID)
		if err != nil {
			return err
		}
		delay, err := s.Clinics.EmailAfter(ctx, a.OrganizationID)
		if err != nil {
			return err
		}
		for _, p := range conv.Participants {
			if p.ID == a.SenderParticipantID {
				continue
			}
			if err := s.tell(ctx, a, p, s.Clock.Now().Add(delay)); err != nil {
				return err
			}
		}
		return s.enqueueWebhooks(ctx, a)
	})
	if errors.Is(err, communication.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("notification: message sent: %w", err)
	}
	return nil
}

// tell enqueues the participant p's notifications of a: the unread check at at and, for the patient, a push per
// device. Staff register no devices (the app is the patients').
func (s *Chat) tell(ctx context.Context, a communication.MessageSentArgs, p communication.Participant, at time.Time) error {
	err := s.Jobs.EnqueueUnreadEmail(ctx, ChatUnreadEmailArgs{
		ParticipantID: p.ID, ConversationID: a.ConversationID, OrganizationID: a.OrganizationID, ReadSeq: p.ReadSeq, Seq: a.Seq,
	}, at)
	if err != nil || p.Role != communication.RolePatient {
		return err
	}
	account, err := s.Patients.NoticeAccount(ctx, agentSent, a.OrganizationID, p.PatientID)
	if err != nil {
		return err
	}
	devices, err := s.Repo.Devices(ctx, account)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return nil
	}
	if err := s.recordDevices(ctx, agentSent, a.OrganizationID, account, len(devices)); err != nil {
		return err
	}
	for _, d := range devices {
		if err := s.Jobs.EnqueuePush(ctx, ChatPushArgs{DeviceID: d.ID, AccountID: account}); err != nil {
			return err
		}
	}
	return nil
}

// enqueueWebhooks enqueues a delivery of a's event to each webhook of its clinic that takes it.
func (s *Chat) enqueueWebhooks(ctx context.Context, a communication.MessageSentArgs) error {
	if err := s.Repo.SetClinic(ctx, a.OrganizationID); err != nil {
		return err
	}
	endpoints, err := s.Repo.Endpoints(ctx, a.OrganizationID)
	if err != nil {
		return err
	}
	for _, e := range endpoints {
		if !takes(e, EventMessageCreated) {
			continue
		}
		err := s.Jobs.EnqueueWebhook(ctx, WebhookDeliveryArgs{
			EndpointID: e.ID, EventID: eventID(EventMessageCreated, a.MessageID), OrganizationID: a.OrganizationID,
			ConversationID: a.ConversationID, MessageID: a.MessageID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func takes(e Endpoint, event string) bool {
	for _, ev := range e.Events {
		if ev == event {
			return true
		}
	}
	return false
}

// eventID is the stable id of the event of the resource id: a UUID (version 8) of their SHA-256, the same on
// every fan-out and delivery attempt, the receiver's idempotency key. It reveals nothing of the resource's id.
func eventID(event, id string) string {
	sum := sha256.Sum256([]byte("salus webhook event\x00" + event + "\x00" + id))
	b := sum[:16]
	b[6] = b[6]&0x0f | 0x80 // version 8
	b[8] = b[8]&0x3f | 0x80 // RFC 9562 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Push sends the device's notification (the chat_push job): the fixed "new message from your clinic" in its
// account's language, nothing else. A token the platform says is gone is forgotten; a device unregistered
// meanwhile, or whose token moved to another account, sends nothing. Errors carry no token.
func (s *Chat) Push(ctx context.Context, a ChatPushArgs) error {
	var d Device
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if d, err = s.Repo.Device(ctx, a.DeviceID); err != nil {
			return err
		}
		if d.AccountID != a.AccountID {
			return ErrNotFound // the token moved to another account: it gets none of this one's notifications
		}
		return s.recordDevices(ctx, agentPush, "", d.AccountID, 1)
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("notification: push: %w", err)
	}
	language, err := s.Accounts.AccountLanguage(ctx, agentPush, d.AccountID)
	if err != nil {
		return fmt.Errorf("notification: push: %w", err)
	}
	alert, err := pick(s.letters, language).message("", "chat_push", letter{})
	if err != nil {
		return fmt.Errorf("notification: push: %w", err)
	}
	err = s.Pusher.Send(ctx, push.Message{Token: d.Token, Platform: d.Platform, Title: alert.Subject, Body: strings.TrimSuffix(alert.Body, "\n")})
	switch {
	case errors.Is(err, push.ErrTokenGone):
		if err := s.Repo.DeleteDevice(ctx, d.ID, d.Token); err != nil {
			return fmt.Errorf("notification: push: forget the device: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("notification: push: %w", err)
	}
	return nil
}

// UnreadEmail e-mails the participant of a who has not read up to its message by now (the chat_unread_email job):
// the fixed "new message" in their language with the link to the app (a patient) or the web app (a practitioner),
// no text, no name. A participant who has read to the end or left, or a conversation the clinic no longer sees,
// gets nothing; one who has read a's message but not a later one is checked again (checkAgain). Errors carry no
// address.
func (s *Chat) UnreadEmail(ctx context.Context, a ChatUnreadEmailArgs) error {
	conv, err := s.Chats.NoticeConversation(ctx, agentUnread, a.OrganizationID, a.ConversationID)
	if errors.Is(err, communication.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("notification: unread e-mail: %w", err)
	}
	var p communication.Participant
	for _, q := range conv.Participants {
		if q.ID == a.ParticipantID {
			p = q
		}
	}
	switch {
	case p.ID == "" || p.ReadSeq >= conv.LastSeq:
		return nil
	case p.ReadSeq >= a.Seq:
		return s.checkAgain(ctx, a, p.ReadSeq, conv.LastSeq)
	}
	to, language, kind, link, err := s.recipient(ctx, a.OrganizationID, p)
	if gone(err) {
		return nil // no account to write to (a role or account gone): nothing a retry would change
	}
	if err != nil {
		return fmt.Errorf("notification: unread e-mail: %w", err)
	}
	msg, err := pick(s.letters, language).message(to, kind, letter{Link: link})
	if err != nil {
		return fmt.Errorf("notification: unread e-mail: %w", err)
	}
	if err := s.Sender.Send(ctx, msg); err != nil {
		return fmt.Errorf("notification: unread e-mail: %w", err)
	}
	return nil
}

// checkAgain schedules a's participant's unread check from their new read position up to the conversation's last
// message, at the clinic's delay: they read a's message, but a later one that came while they had not (and so was
// kept as a's check by its unique key) is still unread. A check already there for that read position (the later
// message's own) is kept instead, so they get one e-mail.
func (s *Chat) checkAgain(ctx context.Context, a ChatUnreadEmailArgs, readSeq, lastSeq int64) error {
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		delay, err := s.Clinics.EmailAfter(ctx, a.OrganizationID)
		if err != nil {
			return err
		}
		next := ChatUnreadEmailArgs{ParticipantID: a.ParticipantID, ConversationID: a.ConversationID, OrganizationID: a.OrganizationID, ReadSeq: readSeq, Seq: lastSeq}
		return s.Jobs.EnqueueUnreadEmail(ctx, next, s.Clock.Now().Add(delay))
	})
	if err != nil {
		return fmt.Errorf("notification: unread e-mail: check again: %w", err)
	}
	return nil
}

// recipient is the participant's address and language, and the template and link of their e-mail.
func (s *Chat) recipient(ctx context.Context, org string, p communication.Participant) (to, language, kind, link string, err error) {
	if p.Role == communication.RolePatient {
		account, err := s.Patients.NoticeAccount(ctx, agentUnread, org, p.PatientID)
		if err != nil {
			return "", "", "", "", err
		}
		to, language, err = s.Accounts.AccountEmail(ctx, agentUnread, account)
		return to, language, "chat_unread", s.PatientAppURL + "/chat", err
	}
	role, err := s.Roles.Lookup(ctx, org, p.PractitionerRoleID)
	if err != nil {
		return "", "", "", "", err
	}
	_, to, language, err = s.Accounts.PractitionerAccount(ctx, agentUnread, org, role.PractitionerID)
	return to, language, "chat_unread_staff", s.WebBaseURL + "/chat", err
}

// gone tells whether err is a lookup's "not found": the participant has no account to write to.
func gone(err error) bool {
	return errors.Is(err, patient.ErrNotFound) || errors.Is(err, practitionerrole.ErrNotFound) || errors.Is(err, auth.ErrNotFound)
}

// recordDevices audits the job agent's read of n of the account's devices.
func (s *Chat) recordDevices(ctx context.Context, agent, org, account string, n int) error {
	return s.Audit.Record(ctx, auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeTREAT},
		Action:          auditevent.ActionRead,
		Outcome:         auditevent.OutcomeSuccess,
		AgentName:       agent,
		OrganizationID:  org,
		EntityReference: "Account/" + account,
		EntityName:      "push_device",
		EntityCount:     int64(n),
	})
}
