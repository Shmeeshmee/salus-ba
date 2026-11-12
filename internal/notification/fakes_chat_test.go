package notification_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
)

// Synthetic ids of the chat's notifications.
const (
	convID          = "c0000000-0000-4000-8000-0000000000c0"
	msgID           = "e0000000-0000-4000-8000-0000000000e0"
	patientPart     = "f1000000-0000-4000-8000-0000000000f1"
	practPart       = "f2000000-0000-4000-8000-0000000000f2"
	patientID       = "d1000000-0000-4000-8000-0000000000d1"
	patientAccount  = "a1000000-0000-4000-8000-0000000000a1"
	roleID          = "b1000000-0000-4000-8000-0000000000b1"
	practitionerID  = "b2000000-0000-4000-8000-0000000000b2"
	practAccount    = "a2000000-0000-4000-8000-0000000000a2"
	clinicAdmin     = "a3000000-0000-4000-8000-0000000000a3"
	secretText      = "Synthetic secret text 7f3a: never in a notification"
	patientName     = "Ada Example"
	practitionerNam = "Dr Synthetic Example"
)

var chatNow = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

type chatTxKey struct{}

type fakeChatTx struct{}

func (fakeChatTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, chatTxKey{}, true))
}

// fakeChatRepo holds devices, endpoints and dead letters; endpoints and dead letters only in the clinic set.
type fakeChatRepo struct {
	err       error
	devices   map[string]notification.Device // by id
	endpoints map[string]notification.Endpoint
	dead      []notification.DeadLetter
	clinic    string
	calls     []string // the endpoint calls, in order: "lock <clinic>", "endpoints", "create"
	registers []time.Time
	mu        sync.Mutex
}

func newFakeChatRepo() *fakeChatRepo {
	return &fakeChatRepo{devices: map[string]notification.Device{}, endpoints: map[string]notification.Endpoint{}}
}

func (r *fakeChatRepo) RegisterDevice(_ context.Context, d notification.Device, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	d.ID = fmt.Sprintf("9d000000-0000-4000-8000-%012d", len(r.registers)+1)
	for id, old := range r.devices {
		if old.Token == d.Token {
			d.ID = id // the token's row moves to the account, as in Postgres
		}
	}
	r.devices[d.ID] = d
	r.registers = append(r.registers, at)
	return nil
}

func (r *fakeChatRepo) UnregisterDevice(_ context.Context, accountID, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, d := range r.devices {
		if d.AccountID == accountID && d.Token == token {
			delete(r.devices, id)
		}
	}
	return r.err
}

func (r *fakeChatRepo) Devices(_ context.Context, accountID string) ([]notification.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []notification.Device
	for _, d := range r.devices {
		if d.AccountID == accountID {
			out = append(out, d)
		}
	}
	slices.SortFunc(out, func(a, b notification.Device) int { return compare(a.ID, b.ID) })
	return out, r.err
}

func (r *fakeChatRepo) Device(_ context.Context, id string) (notification.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.devices[id]
	if !ok {
		return notification.Device{}, notification.ErrNotFound
	}
	return d, r.err
}

func (r *fakeChatRepo) DeleteDevice(_ context.Context, id, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if d, ok := r.devices[id]; ok && d.Token == token {
		delete(r.devices, id)
	}
	return r.err
}

func (r *fakeChatRepo) SetClinic(_ context.Context, id string) error {
	r.clinic = id
	return nil
}

func (r *fakeChatRepo) LockEndpoints(_ context.Context, org string) error {
	r.calls = append(r.calls, "lock "+org)
	return r.err
}

func (r *fakeChatRepo) CreateEndpoint(_ context.Context, e notification.Endpoint) (notification.Endpoint, error) {
	r.calls = append(r.calls, "create")
	if r.err != nil {
		return notification.Endpoint{}, r.err
	}
	if e.OrganizationID != r.clinic {
		return notification.Endpoint{}, notification.ErrNotFound
	}
	e.CreatedAt = chatNow
	r.endpoints[e.ID] = e
	return e, nil
}

func (r *fakeChatRepo) Endpoints(_ context.Context, org string) ([]notification.Endpoint, error) {
	r.calls = append(r.calls, "endpoints")
	var out []notification.Endpoint
	for _, e := range r.endpoints {
		if e.OrganizationID == org && org == r.clinic {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b notification.Endpoint) int { return compare(a.ID, b.ID) })
	return out, r.err
}

func (r *fakeChatRepo) Endpoint(_ context.Context, id string) (notification.Endpoint, error) {
	e, ok := r.endpoints[id]
	if !ok || e.OrganizationID != r.clinic {
		return notification.Endpoint{}, notification.ErrNotFound
	}
	return e, r.err
}

func (r *fakeChatRepo) DeleteEndpoint(_ context.Context, id string) error {
	e, ok := r.endpoints[id]
	if !ok || e.OrganizationID != r.clinic {
		return notification.ErrNotFound
	}
	delete(r.endpoints, id)
	return r.err
}

func (r *fakeChatRepo) AddDeadLetter(_ context.Context, d notification.DeadLetter) error {
	if d.OrganizationID != r.clinic {
		return notification.ErrNotFound
	}
	r.dead = append(r.dead, d)
	return r.err
}

func (r *fakeChatRepo) DeleteDeadLettersBefore(context.Context, time.Time) (int64, error) {
	n := int64(len(r.dead))
	r.dead = nil
	return n, r.err
}

func compare(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// fakeChatJobs records what is enqueued, and whether in a transaction.
type fakeChatJobs struct {
	err      error
	pushes   []notification.ChatPushArgs
	emails   []notification.ChatUnreadEmailArgs
	emailAt  []time.Time
	webhooks []notification.WebhookDeliveryArgs
	outside  int
}

func (j *fakeChatJobs) inTx(ctx context.Context) {
	if ctx.Value(chatTxKey{}) == nil {
		j.outside++
	}
}

func (j *fakeChatJobs) EnqueuePush(ctx context.Context, a notification.ChatPushArgs) error {
	j.inTx(ctx)
	j.pushes = append(j.pushes, a)
	return j.err
}

func (j *fakeChatJobs) EnqueueUnreadEmail(ctx context.Context, a notification.ChatUnreadEmailArgs, at time.Time) error {
	j.inTx(ctx)
	j.emails, j.emailAt = append(j.emails, a), append(j.emailAt, at)
	return j.err
}

func (j *fakeChatJobs) EnqueueWebhook(ctx context.Context, a notification.WebhookDeliveryArgs) error {
	j.inTx(ctx)
	j.webhooks = append(j.webhooks, a)
	return j.err
}

// fakeChats is the conversation as the clinic's scope shows it: the patient and the practitioner, and one message.
type fakeChats struct {
	err   error
	parts []communication.Participant
	asked []string // agent/org/conversation
}

func newFakeChats() *fakeChats {
	return &fakeChats{parts: []communication.Participant{
		{ID: patientPart, ConversationID: convID, PatientID: patientID, Role: communication.RolePatient},
		{ID: practPart, ConversationID: convID, PractitionerRoleID: roleID, Role: communication.RolePractitioner, ReadSeq: 1},
	}}
}

func (c *fakeChats) NoticeParticipants(_ context.Context, agent, org, id string) ([]communication.Participant, error) {
	c.asked = append(c.asked, agent+"/"+org+"/"+id)
	if c.err != nil || id != convID || org != clinicID {
		return nil, joinNotFound(c.err)
	}
	return slices.Clone(c.parts), nil
}

func (c *fakeChats) NoticeMessage(_ context.Context, agent, org, id, messageID string) (communication.Message, error) {
	c.asked = append(c.asked, agent+"/"+org+"/"+id+"/"+messageID)
	if c.err != nil || id != convID || org != clinicID || messageID != msgID {
		return communication.Message{}, joinNotFound(c.err)
	}
	return communication.Message{ID: msgID, ConversationID: convID, OrganizationID: clinicID, PatientID: patientID, Seq: 2,
		SentAt: chatNow, SenderRole: communication.RolePatient, SenderParticipantID: patientPart, Kind: communication.KindText, Body: secretText}, nil
}

func joinNotFound(err error) error {
	if err != nil {
		return err
	}
	return communication.ErrNotFound
}

type fakePatients struct{ asked []string }

func (p *fakePatients) NoticeAccount(_ context.Context, agent, org, id string) (string, error) {
	p.asked = append(p.asked, agent+"/"+org+"/"+id)
	if id != patientID || org != clinicID {
		return "", patient.ErrNotFound
	}
	return patientAccount, nil
}

type fakeRoles struct{}

func (fakeRoles) Lookup(_ context.Context, org, id string) (practitionerrole.PractitionerRole, error) {
	if org != clinicID || id != roleID {
		return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
	}
	return practitionerrole.PractitionerRole{ID: roleID, OrganizationID: clinicID, PractitionerID: practitionerID}, nil
}

// fakeChatAccounts gives the two accounts' e-mails and languages.
type fakeChatAccounts struct {
	language map[string]string
	asked    []string
}

func (a *fakeChatAccounts) AccountEmail(_ context.Context, agent, id string) (string, string, error) {
	a.asked = append(a.asked, agent+"/email/"+id)
	if id != patientAccount {
		return "", "", communication.ErrNotFound
	}
	return "ana@example.test", a.language[id], nil
}

func (a *fakeChatAccounts) AccountLanguage(_ context.Context, agent, id string) (string, error) {
	a.asked = append(a.asked, agent+"/language/"+id)
	return a.language[id], nil
}

func (a *fakeChatAccounts) PractitionerAccount(_ context.Context, agent, org, practitioner string) (string, string, string, error) {
	a.asked = append(a.asked, agent+"/practitioner/"+practitioner)
	if org != clinicID || practitioner != practitionerID {
		return "", "", "", auth.ErrNotFound
	}
	return practAccount, "dr.example@example.test", a.language[practAccount], nil
}

type fakeChatClinics struct{ delay time.Duration }

func (c fakeChatClinics) EmailAfter(_ context.Context, org string) (time.Duration, error) {
	if org != clinicID {
		return 0, communication.ErrNotFound
	}
	return c.delay, nil
}

// fakeChatAudit keeps the events.
type fakeChatAudit struct{ events []auditevent.AuditEvent }

func (a *fakeChatAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	a.events = append(a.events, e)
	return nil
}

// fakePoster records the posts and answers status (200 when zero), or err.
type fakePoster struct {
	err     error
	urls    []string
	bodies  []string
	headers []map[string]string
	status  int
}

func (p *fakePoster) Post(_ context.Context, url string, body []byte, header map[string]string) (int, error) {
	p.urls, p.bodies, p.headers = append(p.urls, url), append(p.bodies, string(body)), append(p.headers, header)
	if p.status == 0 {
		return 200, p.err
	}
	return p.status, p.err
}

// fakeBox "seals" by prefixing the id: it opens only for the id it sealed for.
type fakeBox struct{}

func (fakeBox) Seal(plaintext []byte, id string) ([]byte, error) {
	return append([]byte(id+"|"), plaintext...), nil
}

func (fakeBox) Open(sealed []byte, id string) ([]byte, error) {
	if len(sealed) < len(id)+1 || string(sealed[:len(id)+1]) != id+"|" {
		return nil, notification.ErrInvalid
	}
	return sealed[len(id)+1:], nil
}

// chatFixture is the chat's notifications on fakes.
type chatFixture struct {
	svc      *notification.Chat
	repo     *fakeChatRepo
	jobs     *fakeChatJobs
	chats    *fakeChats
	patients *fakePatients
	accounts *fakeChatAccounts
	pusher   *pushtest.Sender
	mail     *fakeSender
	poster   *fakePoster
	audit    *fakeChatAudit
}

func newChatFixture(t *testing.T) *chatFixture {
	t.Helper()
	f := &chatFixture{
		repo: newFakeChatRepo(), jobs: &fakeChatJobs{}, chats: newFakeChats(), patients: &fakePatients{},
		accounts: &fakeChatAccounts{language: map[string]string{patientAccount: "de", practAccount: "hr"}},
		pusher:   &pushtest.Sender{}, mail: &fakeSender{}, poster: &fakePoster{}, audit: &fakeChatAudit{},
	}
	f.svc = notification.NewChat(notification.ChatPorts{
		Repo: f.repo, Jobs: f.jobs, Chats: f.chats, Patients: f.patients, Roles: fakeRoles{}, Accounts: f.accounts,
		Clinics: fakeChatClinics{delay: 30 * time.Minute}, Pusher: f.pusher, Sender: f.mail, Poster: f.poster, Box: fakeBox{},
		Audit: f.audit, Tx: fakeChatTx{}, Clock: clock.Fixed{T: chatNow},
		WebBaseURL: "https://app.example.test/", PatientAppURL: "https://patient.example.test",
	})
	return f
}

// device registers a device of the account directly in the fake.
func (f *chatFixture) device(t *testing.T, account, token string, p push.Platform) string {
	t.Helper()
	if err := f.repo.RegisterDevice(t.Context(), notification.Device{AccountID: account, Token: token, Platform: p}, chatNow); err != nil {
		t.Fatal(err)
	}
	for id, d := range f.repo.devices {
		if d.Token == token {
			return id
		}
	}
	t.Fatal("no device")
	return ""
}
