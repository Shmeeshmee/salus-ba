package notification

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/communication"
)

// The limits of a clinic's webhooks.
const (
	maxEndpoints = 10
	maxURL       = 2048
	secretPrefix = "whsec_"
)

// events are the webhook events there are.
var events = []string{EventMessageCreated}

var agentWebhook = WebhookDeliveryArgs{}.Kind()

// Webhooks returns the admin's clinic's endpoints, without their secrets, audited as a read. ErrForbidden for its
// other staff, ErrNotFound for anyone else; refusals are audited.
func (s *Chat) Webhooks(ctx context.Context, c Caller) ([]Endpoint, error) {
	if err := s.admin(ctx, c, auditevent.ActionRead, auditevent.CodeRead); err != nil {
		return nil, err
	}
	var out []Endpoint
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		var err error
		if out, err = s.Repo.Endpoints(ctx, c.OrganizationID); err != nil {
			return err
		}
		return s.recordEndpoint(ctx, c, auditevent.ActionRead, auditevent.CodeRead, int64(len(out)), auditevent.OutcomeSuccess)
	})
	if err != nil {
		return nil, fmt.Errorf("notification: webhooks: %w", err)
	}
	for i := range out {
		out[i].SecretSealed = nil
	}
	return out, nil
}

// CreateWebhook adds an endpoint to the admin's clinic and returns it with its secret, shown this once and stored
// sealed to the endpoint. ErrInvalid for a URL that is not https to a public host name, no or unknown events, or an
// eleventh endpoint; refusals as Webhooks'.
func (s *Chat) CreateWebhook(ctx context.Context, c Caller, in NewEndpoint) (Endpoint, string, error) {
	if err := s.admin(ctx, c, auditevent.ActionCreate, auditevent.CodeCreate); err != nil {
		return Endpoint{}, "", err
	}
	evs, err := validEndpoint(in)
	if err != nil {
		return Endpoint{}, "", err
	}
	raw := make([]byte, 32)
	_, _ = rand.Read(raw) // crypto/rand.Read never returns an error
	secret := secretPrefix + base64.RawURLEncoding.EncodeToString(raw)
	e := Endpoint{ID: newID(), OrganizationID: c.OrganizationID, CreatedBy: c.AccountID, URL: in.URL, Events: evs, IncludeClinical: in.IncludeClinical}
	if e.SecretSealed, err = s.Box.Seal([]byte(secret), e.ID); err != nil {
		return Endpoint{}, "", fmt.Errorf("notification: create webhook: %w", err)
	}
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		existing, err := s.Repo.Endpoints(ctx, c.OrganizationID)
		switch {
		case err != nil:
			return err
		case len(existing) >= maxEndpoints:
			return ErrInvalid
		}
		if e, err = s.Repo.CreateEndpoint(ctx, e); err != nil {
			return err
		}
		return s.recordEndpoint(ctx, c, auditevent.ActionCreate, auditevent.CodeCreate, 1, auditevent.OutcomeSuccess)
	})
	if errors.Is(err, ErrInvalid) {
		return Endpoint{}, "", ErrInvalid
	}
	if err != nil {
		return Endpoint{}, "", fmt.Errorf("notification: create webhook: %w", err)
	}
	e.SecretSealed = nil
	return e, secret, nil
}

// DeleteWebhook deletes the endpoint id of the admin's clinic with its dead letters; ErrNotFound for one it does not
// have; refusals as Webhooks'.
func (s *Chat) DeleteWebhook(ctx context.Context, c Caller, id string) error {
	if err := s.admin(ctx, c, auditevent.ActionDelete, auditevent.CodeDelete); err != nil {
		return err
	}
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, c.OrganizationID); err != nil {
			return err
		}
		if e, err := s.Repo.Endpoint(ctx, id); err != nil || e.OrganizationID != c.OrganizationID {
			return errors.Join(ErrNotFound, err)
		}
		if err := s.Repo.DeleteEndpoint(ctx, id); err != nil {
			return err
		}
		return s.recordEndpoint(ctx, c, auditevent.ActionDelete, auditevent.CodeDelete, 1, auditevent.OutcomeSuccess)
	})
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("notification: delete webhook: %w", err)
	}
	return nil
}

// admin is nil for the admin of a clinic; otherwise it audits the refused action and returns ErrForbidden for the
// clinic's other staff, ErrNotFound for anyone else.
func (s *Chat) admin(ctx context.Context, c Caller, action auditevent.Action, interaction string) error {
	var refusal error
	switch {
	case c.OrganizationID == "":
		refusal = ErrNotFound
	case !c.Admin:
		refusal = ErrForbidden
	default:
		return nil
	}
	if err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		return s.recordEndpoint(ctx, c, action, interaction, 0, auditevent.OutcomeMinorFailure)
	}); err != nil {
		return fmt.Errorf("notification: audit refusal: %w", err)
	}
	return refusal
}

func (s *Chat) recordEndpoint(ctx context.Context, c Caller, action auditevent.Action, interaction string, n int64, outcome auditevent.Outcome) error {
	return s.Audit.Record(ctx, auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
		EntityName:     "webhook_endpoint",
		EntityCount:    n,
	})
}

// validEndpoint returns in's events, each once, or ErrInvalid for a URL that is not https to a public host name
// (validURL) or events that are none or unknown.
func validEndpoint(in NewEndpoint) ([]string, error) {
	if !validURL(in.URL) || len(in.Events) == 0 {
		return nil, ErrInvalid
	}
	var out []string
	for _, ev := range in.Events {
		if !slices.Contains(events, ev) {
			return nil, ErrInvalid
		}
		if !slices.Contains(out, ev) {
			out = append(out, ev)
		}
	}
	return out, nil
}

// validURL tells whether raw is an https URL of at most 2048 bytes to a host name with a dot, or a public address,
// without credentials or a fragment. A host name's addresses are checked again at every connection (the poster's
// dialer): a name may resolve elsewhere later.
func validURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > maxURL || u.Scheme != "https" || u.User != nil || u.Fragment != "" || strings.Contains(raw, "#") || u.Opaque != "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if addr, err := netip.ParseAddr(host); err == nil {
		return publicAddr(addr)
	}
	return strings.Contains(host, ".") && host != "localhost" && !strings.HasSuffix(host, ".localhost") && !strings.HasSuffix(host, ".")
}

// nonPublic are the ranges publicAddr refuses beside what netip names: the shared address space of carrier-grade
// NAT, this network, IETF protocol assignments, and the benchmarking range.
var nonPublic = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("198.18.0.0/15"),
}

// publicAddr tells whether a webhook may connect to addr: not loopback, private (RFC 1918, unique-local), link-local,
// multicast, unspecified, or one of nonPublic; an IPv4-mapped IPv6 address is judged as its IPv4 address.
func publicAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() || addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return false
	}
	for _, p := range nonPublic {
		if p.Contains(addr) {
			return false
		}
	}
	return true
}

// newID is a random UUID (version 4).
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// eventJSON is a delivery's body: the event's id and type, the clinic, and the ids of what it is about; the
// message itself only when the endpoint includes clinical content.
type eventJSON struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	OrganizationID string    `json:"organization_id"`
	Data           eventData `json:"data"`
}

type eventData struct {
	Message        *eventMessage `json:"message,omitempty"`
	ConversationID string        `json:"conversation_id"`
	MessageID      string        `json:"message_id"`
}

type eventMessage struct {
	SentAt     time.Time `json:"sent_at"`
	PatientID  string    `json:"patient_id"`
	SenderRole string    `json:"sender_role"`
	Body       string    `json:"body"`
	Seq        int64     `json:"seq"`
}

// DeliverWebhook posts a's event to its endpoint (the webhook_delivery job's attempt of maxAttempts), signed: the
// header Salus-Signature is t=<unix seconds>,v1=<hex HMAC-SHA256 of "<t>.<body>" with the endpoint's secret>, and
// Salus-Event-Id the event's stable id. A status other than 2xx, or no answer, is an error for River to retry with
// backoff; the last attempt's failure records a dead letter first. A deleted endpoint, or a message the clinic no
// longer sees (with clinical content), is ErrNotFound: nothing to retry. Errors name a status, never the URL.
func (s *Chat) DeliverWebhook(ctx context.Context, a WebhookDeliveryArgs, attempt, maxAttempts int) error {
	var e Endpoint
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, a.OrganizationID); err != nil {
			return err
		}
		var err error
		e, err = s.Repo.Endpoint(ctx, a.EndpointID)
		return err
	})
	if err != nil {
		return fmt.Errorf("notification: webhook: %w", err)
	}
	body, err := s.eventBody(ctx, a, e.IncludeClinical)
	if err != nil {
		return fmt.Errorf("notification: webhook: %w", err)
	}
	secret, err := s.Box.Open(e.SecretSealed, e.ID)
	if err != nil {
		return fmt.Errorf("notification: webhook: the secret: %w", err)
	}
	ts := strconv.FormatInt(s.Clock.Now().Unix(), 10)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(ts + "." + string(body)))
	status, err := s.Poster.Post(ctx, e.URL, body, map[string]string{
		"Content-Type": "application/json", "Salus-Event-Id": a.EventID, "Salus-Signature": "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil)),
	})
	if err == nil && status >= 200 && status < 300 {
		return nil
	}
	failure := fmt.Errorf("notification: webhook: status %d", status)
	if err != nil {
		failure, status = fmt.Errorf("notification: webhook: no answer: %w", err), 0
	}
	if attempt >= maxAttempts {
		if derr := s.deadLetter(ctx, a, attempt, status); derr != nil {
			return errors.Join(failure, derr)
		}
	}
	return failure
}

// eventBody is a's event as JSON, with the message when clinical is set (read and audited as the job's).
func (s *Chat) eventBody(ctx context.Context, a WebhookDeliveryArgs, clinical bool) ([]byte, error) {
	ev := eventJSON{ID: a.EventID, Type: EventMessageCreated, OrganizationID: a.OrganizationID,
		Data: eventData{ConversationID: a.ConversationID, MessageID: a.MessageID}}
	if clinical {
		m, err := s.Chats.NoticeMessage(ctx, agentWebhook, a.OrganizationID, a.ConversationID, a.MessageID)
		if errors.Is(err, communication.ErrNotFound) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		ev.Data.Message = &eventMessage{SentAt: m.SentAt, PatientID: m.PatientID, SenderRole: string(m.SenderRole), Body: m.Body, Seq: m.Seq}
	}
	return json.Marshal(ev)
}

// deadLetter records a's delivery as given up after attempts, with the last status (0 for none).
func (s *Chat) deadLetter(ctx context.Context, a WebhookDeliveryArgs, attempts, status int) error {
	return s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetClinic(ctx, a.OrganizationID); err != nil {
			return err
		}
		return s.Repo.AddDeadLetter(ctx, DeadLetter{
			EndpointID: a.EndpointID, OrganizationID: a.OrganizationID, EventID: a.EventID, EventType: EventMessageCreated,
			ConversationID: a.ConversationID, MessageID: a.MessageID, Attempts: attempts, LastStatus: status,
		})
	})
}
