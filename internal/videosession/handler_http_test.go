package videosession_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/middleware"
)

// stubSessions answers the webhook with what it is given and keeps what it was asked.
type stubSessions struct {
	err           error
	kind          videosession.EventKind
	body          string
	authorization string
	calls         int
}

func (s *stubSessions) HandleWebhook(_ context.Context, body []byte, authorization string) (videosession.EventKind, error) {
	s.calls++
	s.body, s.authorization = string(body), authorization
	return s.kind, s.err
}

// A synthetic webhook body naming a room and an identity, which no log line may repeat.
const webhookJSON = `{"event":"participant_joined","room":{"name":"r_secretroom"},"participant":{"identity":"p_secretidentity"}}`

// postWebhook sends body with the authorization to the handler, logging through the request-id handler to
// log, with the request id req-1.
func postWebhook(t *testing.T, svc *stubSessions, log *bytes.Buffer, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	videosession.NewHandler(svc, slog.New(middleware.NewLogHandler(slog.NewJSONHandler(log, nil)))).Routes(mux)
	req := httptest.NewRequestWithContext(middleware.WithRequestID(t.Context(), "req-1"), http.MethodPost, "/webhooks/livekit", strings.NewReader(body))
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// checkLog fails unless the log is exactly the one line, with the request id, and holds nothing of the body.
func checkLog(t *testing.T, log *bytes.Buffer, want string) {
	t.Helper()
	got := strings.TrimSpace(log.String())
	if i := strings.Index(got, `"msg"`); i >= 0 {
		got = "{" + got[i:] // drop the time and level
	}
	if got != want {
		t.Errorf("log = %s\nwant  %s", got, want)
	}
	for _, secret := range []string{"r_secretroom", "p_secretidentity", "signed-token"} {
		if strings.Contains(log.String(), secret) {
			t.Errorf("log holds %q: %s", secret, log.String())
		}
	}
}

// A handled event is 200 with an empty object, also when it has no effect; the service gets the body and the
// Authorization header as sent, and the log holds the request id and the event kind only.
func TestWebhook_AnswersAHandledEvent(t *testing.T) {
	for _, kind := range []videosession.EventKind{videosession.EventParticipantJoined, videosession.EventOther} {
		var log bytes.Buffer
		svc := &stubSessions{kind: kind}
		rec := postWebhook(t, svc, &log, webhookJSON, "signed-token")
		if rec.Code != http.StatusOK || rec.Body.String() != "{}\n" {
			t.Errorf("%s: %d %s, want 200 {}", kind, rec.Code, rec.Body)
		}
		if svc.body != webhookJSON || svc.authorization != "signed-token" {
			t.Errorf("the service got %q with %q", svc.body, svc.authorization)
		}
		checkLog(t, &log, `{"msg":"livekit webhook","kind":"`+string(kind)+`","request_id":"req-1"}`)
	}
}

// An event for a room we do not know is 200, logged as unknown room with its kind and nothing from the body.
func TestWebhook_AnswersAnUnknownRoom(t *testing.T) {
	var log bytes.Buffer
	rec := postWebhook(t, &stubSessions{kind: videosession.EventRoomFinished, err: videosession.ErrUnknownRoom}, &log, webhookJSON, "signed-token")
	if rec.Code != http.StatusOK {
		t.Errorf("status %d, want 200", rec.Code)
	}
	checkLog(t, &log, `{"msg":"unknown room","kind":"room_finished","request_id":"req-1"}`)
}

// A bad or missing signature is 401, logged with the request id only.
func TestWebhook_RefusesABadSignature(t *testing.T) {
	for _, authorization := range []string{"forged", ""} {
		var log bytes.Buffer
		rec := postWebhook(t, &stubSessions{err: videosession.ErrInvalidSignature}, &log, webhookJSON, authorization)
		if rec.Code != http.StatusUnauthorized || rec.Body.String() != `{"error":"unauthenticated"}`+"\n" {
			t.Errorf("%q: %d %s, want 401 unauthenticated", authorization, rec.Code, rec.Body)
		}
		checkLog(t, &log, `{"msg":"livekit webhook refused","request_id":"req-1"}`)
	}
}

// A handling error is a 500 through httpx.InternalError, so LiveKit delivers the webhook again.
func TestWebhook_FailsOnAHandlingError(t *testing.T) {
	var log bytes.Buffer
	rec := postWebhook(t, &stubSessions{kind: videosession.EventParticipantJoined, err: errors.New("videosession: webhook: the backend is down")}, &log, webhookJSON, "signed-token")
	if rec.Code != http.StatusInternalServerError || rec.Body.String() != `{"error":"internal error"}`+"\n" {
		t.Errorf("%d %s, want the fixed 500", rec.Code, rec.Body)
	}
	if !strings.Contains(log.String(), "the backend is down") || !strings.Contains(log.String(), `"request_id":"req-1"`) {
		t.Errorf("log %s; want the error with the request id", log.String())
	}
}

// A body over 64 KiB is 400 and never reaches the service; one of exactly 64 KiB does.
func TestWebhook_ReadsTheBodyUpTo64KiB(t *testing.T) {
	var log bytes.Buffer
	svc := &stubSessions{kind: videosession.EventOther}
	if rec := postWebhook(t, svc, &log, strings.Repeat(" ", 64<<10+1), "signed-token"); rec.Code != http.StatusBadRequest || svc.calls != 0 {
		t.Errorf("a body over 64 KiB: %d, %d calls; want 400 and none", rec.Code, svc.calls)
	}
	if rec := postWebhook(t, svc, &log, strings.Repeat(" ", 64<<10), "signed-token"); rec.Code != http.StatusOK || svc.calls != 1 {
		t.Errorf("a body of 64 KiB: %d, %d calls; want 200 and one", rec.Code, svc.calls)
	}
}
