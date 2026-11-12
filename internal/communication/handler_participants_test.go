package communication_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// stubGroup answers the group consult's calls with what the test set, and keeps what it was asked.
type stubGroup struct {
	err         error
	invitations []communication.Invitation
	threads     []communication.Conversation
	asked       []string
	part        communication.Participant
	thread      communication.Conversation
}

func (s *stubGroup) AddParticipant(_ context.Context, _ communication.Caller, id string, in communication.AddParticipant) (communication.Participant, error) {
	from := "none"
	if in.FromSeq != nil {
		from = "set"
	}
	s.asked = append(s.asked, "add "+id+" "+in.PractitionerRoleID+" "+string(in.Reason)+" "+in.History+" "+from)
	return s.part, s.err
}

func (s *stubGroup) RemoveParticipant(_ context.Context, _ communication.Caller, id, participantID string) error {
	s.asked = append(s.asked, "remove "+id+" "+participantID)
	return s.err
}

func (s *stubGroup) Invitations(context.Context, communication.Caller) ([]communication.Invitation, error) {
	return s.invitations, s.err
}

func (s *stubGroup) DecideInvitation(_ context.Context, _ communication.Caller, id string, d communication.Decision) error {
	s.asked = append(s.asked, "decide "+id+" "+string(d))
	return s.err
}

func (s *stubGroup) RevokeInvitation(_ context.Context, _ communication.Caller, id string) error {
	s.asked = append(s.asked, "revoke "+id)
	return s.err
}

func (s *stubGroup) OpenThread(_ context.Context, _ communication.Caller, id string) (communication.Conversation, error) {
	s.asked = append(s.asked, "thread "+id)
	return s.thread, s.err
}

func (s *stubGroup) Threads(context.Context, communication.Caller, string) ([]communication.Conversation, error) {
	return s.threads, s.err
}

func newGroupServer(t *testing.T, svc *stubGroup, signedIn bool) *httptest.Server {
	t.Helper()
	caller := func(context.Context) (communication.Caller, bool, error) { return me, signedIn, nil }
	base := communication.NewHTTPHandler(&stubChats{}, newFakeLive(), caller, func(*http.Request) (bool, error) { return true, nil }, clock.System{},
		time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	communication.NewGroupHandler(svc, base).Routes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

const partID = "d0000009-0000-4000-8000-000000000000"

func TestGroupHTTP_AddsRemovesAndOpensThreads(t *testing.T) {
	t.Parallel()
	svc := &stubGroup{
		part:   communication.Participant{ID: partID, AddedAt: start, PractitionerRoleID: roleB1, OrganizationID: clinicB, Role: communication.RolePractitioner, Status: communication.StatusPendingConsent, Reason: communication.ReasonSecondOpinion},
		thread: communication.Conversation{ID: "c0000002-0000-4000-8000-000000000000", ParentID: "c0000001-0000-4000-8000-000000000000", StaffOnly: true, Created: start},
	}
	srv := newGroupServer(t, svc, true)
	code, body, h := do(t, srv, http.MethodPost, convPath+"/participants",
		`{"practitioner_role_id":"`+roleB1+`","organization_id":"`+clinicB+`","reason":"second_opinion","history":"from","from_seq":2}`, nil)
	want := `{"added_at":"2026-11-12T10:00:00Z","id":"` + partID + `","practitioner_role_id":"` + roleB1 + `","organization_id":"` + clinicB +
		`","role":"practitioner","status":"pending_consent","reason":"second_opinion","history_from_seq":0}`
	if code != http.StatusCreated || body != want || h.Get("Cache-Control") != "no-store" {
		t.Errorf("add = %d %s\nwant %s", code, body, want)
	}
	if code, _, _ := do(t, srv, http.MethodPost, convPath+"/participants", `{"practitioner_role_id":"x","extra":1}`, nil); code != http.StatusBadRequest {
		t.Errorf("an unknown field = %d, want 400", code)
	}
	if code, _, _ := do(t, srv, http.MethodDelete, convPath+"/participants/"+partID, "", nil); code != http.StatusNoContent {
		t.Errorf("remove = %d, want 204", code)
	}
	if code, body, _ := do(t, srv, http.MethodPost, convPath+"/threads", "", nil); code != http.StatusCreated || !strings.Contains(body, `"staff_only":true`) {
		t.Errorf("open thread = %d %s", code, body)
	}
	if code, body, _ := do(t, srv, http.MethodGet, convPath+"/threads", "", nil); code != http.StatusOK || body != `{"conversations":[]}` {
		t.Errorf("threads = %d %s", code, body)
	}
	svc.threads = []communication.Conversation{svc.thread}
	if code, body, _ := do(t, srv, http.MethodGet, convPath+"/threads", "", nil); code != http.StatusOK || !strings.Contains(body, `"parent_id":"c0000001`) {
		t.Errorf("threads = %d %s", code, body)
	}
	wantCalls := "add c0000001-0000-4000-8000-000000000000 " + roleB1 + " second_opinion from set|remove c0000001-0000-4000-8000-000000000000 " + partID +
		"|thread c0000001-0000-4000-8000-000000000000"
	if got := strings.Join(svc.asked, "|"); got != wantCalls {
		t.Errorf("calls = %q, want %q", got, wantCalls)
	}
}

func TestGroupHTTP_ThePatientListsDecidesAndRevokesInvitations(t *testing.T) {
	t.Parallel()
	svc := &stubGroup{}
	srv := newGroupServer(t, svc, true)
	if code, body, _ := do(t, srv, http.MethodGet, "/me/chat-invitations", "", nil); code != http.StatusOK || body != `{"invitations":[]}` {
		t.Errorf("no invitations = %d %s", code, body)
	}
	svc.invitations = []communication.Invitation{{InvitedAt: start, ID: partID, ConversationID: "c0000001-0000-4000-8000-000000000000", OrganizationID: clinicB,
		PractitionerRoleID: roleB1, Reason: communication.ReasonSpecialistReview}}
	want := `{"invitations":[{"invited_at":"2026-11-12T10:00:00Z","id":"` + partID + `","conversation_id":"c0000001-0000-4000-8000-000000000000",` +
		`"organization_id":"` + clinicB + `","practitioner_role_id":"` + roleB1 + `","reason":"specialist_review"}]}`
	if code, body, h := do(t, srv, http.MethodGet, "/me/chat-invitations", "", nil); code != http.StatusOK || body != want || h.Get("Cache-Control") != "no-store" {
		t.Errorf("invitations = %d %s\nwant %s", code, body, want)
	}
	if code, _, _ := do(t, srv, http.MethodPost, "/me/chat-invitations/"+partID+"/decision", `{"decision":"grant"}`, nil); code != http.StatusNoContent {
		t.Errorf("grant = %d, want 204", code)
	}
	if code, _, _ := do(t, srv, http.MethodPost, "/me/chat-invitations/"+partID+"/decision", `{}`, nil); code != http.StatusBadRequest {
		t.Errorf("no decision = %d, want 400", code)
	}
	if code, _, _ := do(t, srv, http.MethodPost, "/me/chat-invitations/"+partID+"/revoke", "", nil); code != http.StatusNoContent {
		t.Errorf("revoke = %d, want 204", code)
	}
	if got, want := strings.Join(svc.asked, "|"), "decide "+partID+" grant|revoke "+partID; got != want {
		t.Errorf("calls = %q, want %q", got, want)
	}
}

func TestGroupHTTP_MapsTheDomainErrorsAndRefusesTheSignedOut(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		err    error
		code   string
		status int
	}{
		{communication.ErrAlreadyParticipant, "already_participant", http.StatusConflict},
		{communication.ErrConsentRequired, "consent_required", http.StatusForbidden},
		{communication.ErrNotFound, "not_found", http.StatusNotFound},
		{communication.ErrInvalid, "invalid_request", http.StatusBadRequest},
		{errors.New("communication: add participant: secret detail"), "", http.StatusInternalServerError},
	} {
		srv := newGroupServer(t, &stubGroup{err: tt.err}, true)
		code, body, _ := do(t, srv, http.MethodPost, convPath+"/participants", `{"practitioner_role_id":"`+roleA2+`"}`, nil)
		if code != tt.status || (tt.code != "" && body != `{"error":"`+tt.code+`"}`) || strings.Contains(body, "secret") {
			t.Errorf("%v: %d %s, want %d %s", tt.err, code, body, tt.status, tt.code)
		}
	}
	srv := newGroupServer(t, &stubGroup{}, false)
	for _, r := range [][2]string{
		{http.MethodPost, convPath + "/participants"}, {http.MethodDelete, convPath + "/participants/" + partID}, {http.MethodGet, "/me/chat-invitations"},
		{http.MethodPost, "/me/chat-invitations/" + partID + "/decision"}, {http.MethodPost, "/me/chat-invitations/" + partID + "/revoke"},
		{http.MethodPost, convPath + "/threads"}, {http.MethodGet, convPath + "/threads"},
	} {
		if code, _, _ := do(t, srv, r[0], r[1], `{}`, nil); code != http.StatusUnauthorized {
			t.Errorf("signed out %s %s = %d, want 401", r[0], r[1], code)
		}
	}
}
