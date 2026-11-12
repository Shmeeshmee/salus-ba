package communication_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// stubChats answers every call with what the test set, and keeps what it was asked.
type stubChats struct {
	err          error
	afterErr     error
	next         *communication.ListCursor
	sub          communication.Subscription
	page         communication.Page
	list         []communication.Conversation
	revs         []communication.Revision
	after        [][]communication.Message // successive answers of After; then afterErr, then nothing
	asked        []string
	sent         communication.Message
	summary      communication.Summary
	conv         communication.Conversation
	mu           sync.Mutex
	sessionEnded bool
	created      bool
}

func (s *stubChats) ask(format string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.asked = append(s.asked, format)
}

func (s *stubChats) Open(_ context.Context, _ communication.Caller, in communication.Open) (communication.Conversation, bool, error) {
	s.ask("open " + in.PatientID)
	return s.conv, s.created, s.err
}

func (s *stubChats) List(_ context.Context, _ communication.Caller, before *communication.ListCursor) ([]communication.Conversation, *communication.ListCursor, error) {
	if before != nil {
		s.ask("list before " + before.ID)
	}
	return s.list, s.next, s.err
}

func (s *stubChats) Get(context.Context, communication.Caller, string) (communication.Conversation, error) {
	return s.conv, s.err
}

func (s *stubChats) Messages(_ context.Context, _ communication.Caller, _ string, before *communication.Cursor, limit int) (communication.Page, error) {
	if before != nil {
		s.ask("before")
	}
	if limit != 0 {
		s.ask("limit")
	}
	return s.page, s.err
}

func (s *stubChats) Send(_ context.Context, _ communication.Caller, _, client, body string) (communication.Message, bool, error) {
	s.ask("send " + client + " " + body)
	return s.sent, s.created, s.err
}

func (s *stubChats) Edit(_ context.Context, _ communication.Caller, _, _, body string, version int32) (communication.Message, error) {
	if version != 1 {
		return communication.Message{}, communication.ErrVersionConflict
	}
	s.ask("edit " + body)
	return s.sent, s.err
}

func (s *stubChats) Revisions(context.Context, communication.Caller, string, string) ([]communication.Revision, error) {
	return s.revs, s.err
}

func (s *stubChats) MarkRead(_ context.Context, _ communication.Caller, _ string, seq int64) error {
	if seq == 3 {
		s.ask("read 3")
	}
	return s.err
}

func (s *stubChats) Subscribe(context.Context, communication.Caller, string, string) (communication.Subscription, error) {
	return s.sub, s.err
}

func (s *stubChats) After(context.Context, communication.Caller, string, communication.Cursor) ([]communication.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.after) > 0 {
		next := s.after[0]
		s.after = s.after[1:]
		return next, nil
	}
	return nil, s.afterErr
}

func (s *stubChats) Summarize(_ context.Context, _ communication.Caller, _, lang string) (communication.Summary, error) {
	s.ask("summary " + lang)
	return s.summary, s.err
}

// sessionOpen is the socket's session check: open until the test ends it.
func (s *stubChats) sessionOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.sessionEnded
}

func (s *stubChats) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.asked...)
}

// fakeLive is the hub in one process: the test publishes, every subscriber of the channel receives.
type fakeLive struct {
	subs  map[string][]chan []byte
	ready chan struct{}
	mu    sync.Mutex
}

func newFakeLive() *fakeLive {
	return &fakeLive{subs: map[string][]chan []byte{}, ready: make(chan struct{}, 8)}
}

func (l *fakeLive) Subscribe(_ context.Context, channel string) (<-chan []byte, func(), error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan []byte, 16)
	l.subs[channel] = append(l.subs[channel], ch)
	l.ready <- struct{}{}
	return ch, func() {}, nil
}

func (l *fakeLive) publish(channel string, frame []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, ch := range l.subs[channel] {
		ch <- frame
	}
}

func newServer(t *testing.T, svc *stubChats, live *fakeLive, signedIn bool) *httptest.Server {
	t.Helper()
	return newServerEvery(t, svc, live, signedIn, time.Hour)
}

// newServerEvery is newServer with the socket's periodic check every check.
func newServerEvery(t *testing.T, svc *stubChats, live *fakeLive, signedIn bool, check time.Duration) *httptest.Server {
	t.Helper()
	caller := func(context.Context) (communication.Caller, bool, error) { return me, signedIn, nil }
	h := communication.NewHTTPHandler(svc, live, caller, func(*http.Request) (bool, error) { return svc.sessionOpen(), nil }, clock.System{}, check,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	h.StreamRoutes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, srv *httptest.Server, method, path, body string, header map[string]string) (int, string, http.Header) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, strings.TrimSpace(string(b)), res.Header
}

var sample = communication.Message{
	SentAt: start, ID: "f0000001-0000-4000-8000-000000000000", ConversationID: "c0000001-0000-4000-8000-000000000000",
	SenderParticipantID: "d0000001-0000-4000-8000-000000000000", SenderAccountID: me.AccountID, ClientMessageID: clientID1,
	Body: "Synthetic placeholder", SenderRole: communication.RolePatient, Kind: communication.KindText, Seq: 1, VersionID: 1,
}

const convPath = "/conversations/c0000001-0000-4000-8000-000000000000"

// The ChatMessage schema: the body, the cursor, never the sender's account or the patient's id.
func TestMessageJSON_IsTheChatMessageSchema(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"body":"Synthetic placeholder","cursor":"` + sample.Cursor().String() + `","sent_at":"2026-11-12T10:00:00Z",` +
		`"id":"f0000001-0000-4000-8000-000000000000","conversation_id":"c0000001-0000-4000-8000-000000000000",` +
		`"sender_participant_id":"d0000001-0000-4000-8000-000000000000","client_message_id":"` + clientID1 + `",` +
		`"sender_role":"patient","kind":"text","seq":1,"version_id":1}`
	if string(b) != want {
		t.Errorf("JSON =\n%s\nwant\n%s", b, want)
	}
	sys := communication.Message{SentAt: start, Kind: communication.KindSystem, SenderRole: communication.RoleSystem, Seq: 2,
		SystemCode: "participant_added", SystemArgs: map[string]string{"practitioner_role_id": roleA2}, EditedAt: start}
	b, _ = json.Marshal(sys)
	if !strings.Contains(string(b), `"system":{"args":{"practitioner_role_id":"`+roleA2+`"},"code":"participant_added"}`) ||
		strings.Contains(string(b), `"body"`) || !strings.Contains(string(b), `"edited_at"`) {
		t.Errorf("system message JSON = %s", b)
	}
}

func TestHTTP_OpenAndSendAnswer201WhenNewAnd200ForTheExistingOne(t *testing.T) {
	t.Parallel()
	svc := &stubChats{conv: communication.Conversation{ID: "c0000001-0000-4000-8000-000000000000", Created: start}, created: true, sent: sample}
	srv := newServer(t, svc, newFakeLive(), true)
	if code, body, _ := do(t, srv, http.MethodPost, "/conversations", `{"patient_id":"`+patientA+`"}`, nil); code != http.StatusCreated || !strings.Contains(body, `"id":"c0000001`) {
		t.Errorf("open = %d %s", code, body)
	}
	code, body, h := do(t, srv, http.MethodPost, convPath+"/messages", `{"client_message_id":"`+clientID1+`","body":"Synthetic placeholder"}`, nil)
	if code != http.StatusCreated || h.Get("ETag") != `W/"1"` || h.Get("Cache-Control") != "no-store" || !strings.Contains(body, `"seq":1`) {
		t.Errorf("send = %d %s %v", code, body, h)
	}
	svc.created = false
	if code, _, _ := do(t, srv, http.MethodPost, convPath+"/messages", `{"client_message_id":"`+clientID1+`","body":"x"}`, nil); code != http.StatusOK {
		t.Errorf("repeat = %d, want 200", code)
	}
	if code, _, _ := do(t, srv, http.MethodPost, convPath+"/messages", `{"client_message_id":"x","body":"x","extra":1}`, nil); code != http.StatusBadRequest {
		t.Errorf("unknown field = %d, want 400", code)
	}
}

func TestHTTP_MapsEachDomainErrorToItsStatusAndCode(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		err    error
		code   string
		status int
	}{
		{communication.ErrInvalid, "invalid_request", http.StatusBadRequest},
		{communication.ErrInvalidCursor, "invalid_cursor", http.StatusBadRequest},
		{communication.ErrNotFound, "not_found", http.StatusNotFound},
		{communication.ErrForbidden, "forbidden", http.StatusForbidden},
		{communication.ErrConsentRequired, "consent_required", http.StatusForbidden},
		{communication.ErrVersionConflict, "version_conflict", http.StatusPreconditionFailed},
		{errors.New("communication: messages: secret detail"), "", http.StatusInternalServerError},
	} {
		srv := newServer(t, &stubChats{err: tt.err}, newFakeLive(), true)
		code, body, _ := do(t, srv, http.MethodGet, convPath+"/messages", "", nil)
		if code != tt.status || (tt.code != "" && body != `{"error":"`+tt.code+`"}`) || strings.Contains(body, "secret") {
			t.Errorf("%v: %d %s, want %d %s", tt.err, code, body, tt.status, tt.code)
		}
	}
	srv := newServer(t, &stubChats{}, newFakeLive(), false)
	if code, _, _ := do(t, srv, http.MethodGet, "/conversations", "", nil); code != http.StatusUnauthorized {
		t.Errorf("signed out = %d, want 401", code)
	}
}

func TestHTTP_PagesListsEditsRevisionsAndTheReadPosition(t *testing.T) {
	t.Parallel()
	next := sample.Cursor()
	svc := &stubChats{
		page: communication.Page{Messages: []communication.Message{sample}, Next: &next}, sent: sample,
		list: []communication.Conversation{{ID: "c0000001-0000-4000-8000-000000000000"}},
		next: &communication.ListCursor{At: start, ID: "c0000001-0000-4000-8000-000000000000"},
		revs: []communication.Revision{{EditedAt: start, Body: "Synthetic placeholder", VersionID: 1}},
	}
	srv := newServer(t, svc, newFakeLive(), true)
	code, body, _ := do(t, srv, http.MethodGet, convPath+"/messages?limit=10&before="+next.String(), "", nil)
	if code != http.StatusOK || !strings.Contains(body, `"next":"`+next.String()+`"`) {
		t.Errorf("messages = %d %s", code, body)
	}
	if code, _, _ := do(t, srv, http.MethodGet, convPath+"/messages?before=bad", "", nil); code != http.StatusBadRequest {
		t.Errorf("bad cursor = %d", code)
	}
	if code, _, _ := do(t, srv, http.MethodGet, convPath+"/messages?limit=zero", "", nil); code != http.StatusBadRequest {
		t.Errorf("bad limit = %d", code)
	}
	code, body, _ = do(t, srv, http.MethodGet, "/conversations", "", nil)
	var list struct {
		Next          string `json:"next"`
		Conversations []json.RawMessage
	}
	if code != http.StatusOK || json.Unmarshal([]byte(body), &list) != nil || list.Next == "" || len(list.Conversations) != 1 {
		t.Fatalf("list = %d %s", code, body)
	}
	if code, _, _ := do(t, srv, http.MethodGet, "/conversations?before="+list.Next, "", nil); code != http.StatusOK {
		t.Errorf("list with its cursor = %d", code)
	}
	if code, _, _ := do(t, srv, http.MethodPatch, convPath+"/messages/"+sample.ID, `{"body":"Edited placeholder"}`, nil); code != http.StatusPreconditionRequired {
		t.Errorf("edit without If-Match = %d", code)
	}
	if code, _, _ := do(t, srv, http.MethodPatch, convPath+"/messages/"+sample.ID, `{"body":"Edited placeholder"}`, map[string]string{"If-Match": `W/"1"`}); code != http.StatusOK {
		t.Errorf("edit = %d", code)
	}
	if code, body, _ := do(t, srv, http.MethodGet, convPath+"/messages/"+sample.ID+"/revisions", "", nil); code != http.StatusOK || !strings.Contains(body, `"revisions":[{"edited_at"`) {
		t.Errorf("revisions = %d %s", code, body)
	}
	if code, _, _ := do(t, srv, http.MethodPut, convPath+"/read", `{"seq":3}`, nil); code != http.StatusNoContent {
		t.Errorf("read = %d", code)
	}
	if code, _, _ := do(t, srv, http.MethodPut, convPath+"/read", `{}`, nil); code != http.StatusBadRequest {
		t.Errorf("read without seq = %d", code)
	}
	want := []string{"before", "limit", "list before c0000001-0000-4000-8000-000000000000", "edit Edited placeholder", "read 3"}
	if got := svc.calls(); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("calls = %q, want %q", got, want)
	}
}

func TestHTTP_EmptyListsAreArrays(t *testing.T) {
	t.Parallel()
	srv := newServer(t, &stubChats{}, newFakeLive(), true)
	if _, body, _ := do(t, srv, http.MethodGet, "/conversations", "", nil); body != `{"conversations":[]}` {
		t.Errorf("list = %s", body)
	}
	if _, body, _ := do(t, srv, http.MethodGet, convPath+"/messages", "", nil); body != `{"messages":[]}` {
		t.Errorf("messages = %s", body)
	}
	if _, body, _ := do(t, srv, http.MethodGet, convPath+"/messages/"+sample.ID+"/revisions", "", nil); !bytes.Equal([]byte(body), []byte(`{"revisions":[]}`)) {
		t.Errorf("revisions = %s", body)
	}
}
