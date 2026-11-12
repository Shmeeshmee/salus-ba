package notification_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/notification"
)

var clinicAdminCaller = notification.Caller{AccountID: clinicAdmin, OrganizationID: clinicID, Admin: true}

// The admin creates a webhook and gets its secret once; the stored secret is sealed to the endpoint; the list never
// shows it; the admin deletes it. Every step is audited.
func TestWebhooks_TheAdminCreatesListsAndDeletesTheClinicsEndpoints(t *testing.T) {
	f := newChatFixture(t)
	e, secret, err := f.svc.CreateWebhook(t.Context(), clinicAdminCaller, notification.NewEndpoint{
		URL: "https://hooks.example.test/salus?source=chat", Events: []string{notification.EventMessageCreated, notification.EventMessageCreated},
	})
	if err != nil {
		t.Fatalf("CreateWebhook() = %v", err)
	}
	if !strings.HasPrefix(secret, "whsec_") || len(secret) != len("whsec_")+43 || e.IncludeClinical || len(e.Events) != 1 || len(e.ID) != 36 {
		t.Errorf("created %+v with secret %q", e, secret)
	}
	stored := f.repo.endpoints[e.ID]
	if opened, err := (fakeBox{}).Open(stored.SecretSealed, e.ID); err != nil || string(opened) != secret || stored.CreatedBy != clinicAdmin || stored.OrganizationID != clinicID {
		t.Errorf("stored %+v: the secret opens to %q (%v)", stored, opened, err)
	}
	list, err := f.svc.Webhooks(t.Context(), clinicAdminCaller)
	if err != nil || len(list) != 1 || list[0].ID != e.ID {
		t.Fatalf("Webhooks() = %+v, %v", list, err)
	}
	b, _ := json.Marshal(list[0])
	if strings.Contains(string(b), "whsec_") || strings.Contains(string(b), "secret") {
		t.Errorf("the listed endpoint %s shows its secret", b)
	}
	if err := f.svc.DeleteWebhook(t.Context(), clinicAdminCaller, e.ID); err != nil || len(f.repo.endpoints) != 0 {
		t.Errorf("DeleteWebhook() = %v, left %+v", err, f.repo.endpoints)
	}
	if err := f.svc.DeleteWebhook(t.Context(), clinicAdminCaller, e.ID); !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("DeleteWebhook(again) = %v, want ErrNotFound", err)
	}
	var actions []auditevent.Action
	for _, ev := range f.audit.events {
		actions = append(actions, ev.Action)
		if ev.AgentAccountID != clinicAdmin || ev.OrganizationID != clinicID || ev.EntityName != "webhook_endpoint" {
			t.Errorf("audit %+v", ev)
		}
	}
	if fmt.Sprint(actions) != fmt.Sprint([]auditevent.Action{auditevent.ActionCreate, auditevent.ActionRead, auditevent.ActionDelete}) {
		t.Errorf("audited %v", actions)
	}
	for _, ev := range f.audit.events {
		if strings.Contains(fmt.Sprintf("%+v", ev), "whsec_") || strings.Contains(fmt.Sprintf("%+v", ev), "hooks.example.test") {
			t.Errorf("an audit event holds the secret or the URL: %+v", ev)
		}
	}
}

// Only the clinic's admin manages its webhooks (refusals audited), at most ten, https to a public host name, for
// the events there are.
func TestWebhooks_RefuseAnyoneButTheAdminAndInvalidEndpoints(t *testing.T) {
	f := newChatFixture(t)
	staff := notification.Caller{AccountID: practAccount, OrganizationID: clinicID}
	if _, _, err := f.svc.CreateWebhook(t.Context(), staff, notification.NewEndpoint{URL: "https://hooks.example.test", Events: []string{notification.EventMessageCreated}}); !errors.Is(err, notification.ErrForbidden) {
		t.Errorf("staff create: %v", err)
	}
	if _, err := f.svc.Webhooks(t.Context(), notification.Caller{AccountID: patientAccount}); !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("a patient lists: %v", err)
	}
	if err := f.svc.DeleteWebhook(t.Context(), staff, "e1000000-0000-4000-8000-0000000000e1"); !errors.Is(err, notification.ErrForbidden) {
		t.Errorf("staff delete: %v", err)
	}
	if len(f.audit.events) != 3 || f.audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("refusals audited %+v", f.audit.events)
	}
	for _, url := range []string{
		"http://hooks.example.test", "https://", "https://user:pass@hooks.example.test", "https://hooks.example.test/#x",
		"https://localhost/hook", "https://api.localhost", "https://127.0.0.1/", "https://10.0.0.5/", "https://[::1]/",
		"https://169.254.169.254/latest", "https://[fd00::1]/", "https://192.168.1.10:8443/", "https://100.64.0.1/", "https://intranet/",
		"https://0.0.0.0/", "https://[::ffff:127.0.0.1]/", "https://[64:ff9b::a00:1]/", "https://[2002:a00:1::1]/", "https://240.0.0.1/", "ftp://hooks.example.test", "https://hooks.example.test/" + strings.Repeat("a", 2048),
	} {
		if _, _, err := f.svc.CreateWebhook(t.Context(), clinicAdminCaller, notification.NewEndpoint{URL: url, Events: []string{notification.EventMessageCreated}}); !errors.Is(err, notification.ErrInvalid) {
			t.Errorf("CreateWebhook(%q) = %v, want ErrInvalid", url, err)
		}
	}
	for _, events := range [][]string{nil, {"patient.read"}, {notification.EventMessageCreated, ""}} {
		if _, _, err := f.svc.CreateWebhook(t.Context(), clinicAdminCaller, notification.NewEndpoint{URL: "https://hooks.example.test", Events: events}); !errors.Is(err, notification.ErrInvalid) {
			t.Errorf("events %v: %v, want ErrInvalid", events, err)
		}
	}
	for i := range 10 {
		if _, _, err := f.svc.CreateWebhook(t.Context(), clinicAdminCaller, notification.NewEndpoint{URL: fmt.Sprintf("https://hooks%d.example.test", i), Events: []string{notification.EventMessageCreated}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := f.svc.CreateWebhook(t.Context(), clinicAdminCaller, notification.NewEndpoint{URL: "https://hooks.example.test", Events: []string{notification.EventMessageCreated}}); !errors.Is(err, notification.ErrInvalid) {
		t.Errorf("the eleventh endpoint: %v, want ErrInvalid", err)
	}
}

// A creation counts the clinic's endpoints under the clinic's lock, so two at once cannot both pass the tenth.
func TestCreateWebhook_CountsUnderTheClinicsLock(t *testing.T) {
	f := newChatFixture(t)
	if _, _, err := f.svc.CreateWebhook(t.Context(), clinicAdminCaller, notification.NewEndpoint{URL: "https://hooks.example.test", Events: []string{notification.EventMessageCreated}}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"lock " + clinicID, "endpoints", "create"}; fmt.Sprint(f.repo.calls) != fmt.Sprint(want) {
		t.Errorf("calls %v, want %v", f.repo.calls, want)
	}
}

// endpoint stores a webhook of the clinic with the secret, sealed.
func (f *chatFixture) endpoint(t *testing.T, id, secret string, clinical bool) {
	t.Helper()
	sealed, _ := fakeBox{}.Seal([]byte(secret), id)
	f.repo.endpoints[id] = notification.Endpoint{ID: id, OrganizationID: clinicID, URL: "https://hooks.example.test/salus", Events: []string{notification.EventMessageCreated},
		SecretSealed: sealed, IncludeClinical: clinical}
}

const hookID = "e1000000-0000-4000-8000-0000000000e1"

var delivery = notification.WebhookDeliveryArgs{EndpointID: hookID, EventID: "8e000000-0000-8000-8000-00000000008e", OrganizationID: clinicID, ConversationID: convID, MessageID: msgID}

// A delivery posts the event with the ids and type only, signed with HMAC-SHA256 over "<timestamp>.<body>" with the
// endpoint's secret, the timestamp in the signature against replay, and the event's id in a header.
func TestDeliverWebhook_PostsTheSignedEventWithIDsOnly(t *testing.T) {
	f := newChatFixture(t)
	f.endpoint(t, hookID, "whsec_synthetic", false)
	if err := f.svc.DeliverWebhook(t.Context(), delivery, 1, 8); err != nil {
		t.Fatalf("DeliverWebhook() = %v", err)
	}
	if len(f.poster.bodies) != 1 || f.poster.urls[0] != "https://hooks.example.test/salus" {
		t.Fatalf("posted %v to %v", f.poster.bodies, f.poster.urls)
	}
	body, h := f.poster.bodies[0], f.poster.headers[0]
	want := `{"id":"` + delivery.EventID + `","type":"message.created","organization_id":"` + clinicID + `","data":{"conversation_id":"` + convID + `","message_id":"` + msgID + `"}}`
	if body != want {
		t.Errorf("body %s,\nwant %s", body, want)
	}
	ts := fmt.Sprint(chatNow.Unix())
	mac := hmac.New(sha256.New, []byte("whsec_synthetic"))
	mac.Write([]byte(ts + "." + body))
	if h["Salus-Signature"] != "t="+ts+",v1="+hex.EncodeToString(mac.Sum(nil)) || h["Salus-Event-Id"] != delivery.EventID || h["Content-Type"] != "application/json" {
		t.Errorf("headers %v", h)
	}
	if strings.Contains(body, "7f3a") || len(f.chats.asked) != 0 {
		t.Errorf("without clinical content the message was read (%v) or sent", f.chats.asked)
	}
}

// With clinical content on, the delivery also carries the message (its text), read and audited as the job's.
func TestDeliverWebhook_CarriesTheMessageWhenTheClinicTurnedClinicalContentOn(t *testing.T) {
	f := newChatFixture(t)
	f.endpoint(t, hookID, "whsec_synthetic", true)
	if err := f.svc.DeliverWebhook(t.Context(), delivery, 1, 8); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Data struct {
			Message struct {
				Body       string `json:"body"`
				PatientID  string `json:"patient_id"`
				SenderRole string `json:"sender_role"`
				Seq        int64  `json:"seq"`
			} `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(f.poster.bodies[0]), &got); err != nil || got.Data.Message.Body != secretText || got.Data.Message.Seq != 2 ||
		got.Data.Message.SenderRole != "patient" || got.Data.Message.PatientID != patientID {
		t.Errorf("body %s (%v)", f.poster.bodies[0], err)
	}
	if len(f.chats.asked) != 1 || !strings.HasPrefix(f.chats.asked[0], "webhook_delivery/") {
		t.Errorf("read as %v, want by the job", f.chats.asked)
	}
}

// A failed attempt is the job's error, for River to retry with backoff (naming the status, never the URL); the last
// one records a dead letter. A deleted endpoint, or a conversation the clinic no longer sees, is not retried.
func TestDeliverWebhook_RetriesThenDeadLettersAndGivesUpOnWhatIsGone(t *testing.T) {
	f := newChatFixture(t)
	f.endpoint(t, hookID, "whsec_synthetic", true)
	f.poster.status = 503
	err := f.svc.DeliverWebhook(t.Context(), delivery, 3, 8)
	if err == nil || !strings.Contains(err.Error(), "503") || strings.Contains(err.Error(), "hooks.example.test") || len(f.repo.dead) != 0 {
		t.Errorf("attempt 3 of 8 = %v, dead %+v; want a retry", err, f.repo.dead)
	}
	f.poster.status, f.poster.err = 0, errors.New("unreachable")
	if err := f.svc.DeliverWebhook(t.Context(), delivery, 8, 8); err == nil {
		t.Error("the last attempt succeeded")
	}
	want := notification.DeadLetter{EndpointID: hookID, OrganizationID: clinicID, EventID: delivery.EventID, EventType: notification.EventMessageCreated,
		ConversationID: convID, MessageID: msgID, Attempts: 8}
	if len(f.repo.dead) != 1 || f.repo.dead[0] != want {
		t.Errorf("dead letters %+v, want %+v", f.repo.dead, want)
	}

	f = newChatFixture(t)
	if err := f.svc.DeliverWebhook(t.Context(), delivery, 1, 8); !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("a deleted endpoint: %v, want ErrNotFound", err)
	}
	f.endpoint(t, hookID, "whsec_synthetic", true)
	f.chats.err = errors.Join(errors.New("revoked"), errNotFoundChat())
	if err := f.svc.DeliverWebhook(t.Context(), delivery, 1, 8); !errors.Is(err, notification.ErrNotFound) || len(f.poster.bodies) != 0 {
		t.Errorf("after a revocation: %v, posted %d; want nothing", err, len(f.poster.bodies))
	}
}

func errNotFoundChat() error { return joinNotFound(nil) }
