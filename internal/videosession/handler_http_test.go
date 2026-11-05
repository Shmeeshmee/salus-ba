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
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/pkg/middleware"
)

// stubSessions answers with what it is given and keeps what it was asked.
type stubSessions struct {
	err           error
	join          videosession.Join
	status        videosession.Status
	kind          videosession.EventKind
	body          string
	authorization string
	asked         string // the method and the appointment id
	caller        appointment.Caller
	calls         int
}

func (s *stubSessions) Join(_ context.Context, c appointment.Caller, id string) (videosession.Join, error) {
	s.calls++
	s.caller, s.asked = c, "join "+id
	return s.join, s.err
}

func (s *stubSessions) End(_ context.Context, c appointment.Caller, id string) (videosession.Status, error) {
	s.calls++
	s.caller, s.asked = c, "end "+id
	return s.status, s.err
}

func (s *stubSessions) Status(_ context.Context, c appointment.Caller, id string) (videosession.Status, error) {
	s.calls++
	s.caller, s.asked = c, "status "+id
	return s.status, s.err
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
	videosession.NewHandler(svc, signedIn(pract), slog.New(middleware.NewLogHandler(slog.NewJSONHandler(log, nil)))).Routes(mux)
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

// signedIn is the caller of every request: c.
func signedIn(c appointment.Caller) func(context.Context) (appointment.Caller, bool, error) {
	return func(context.Context) (appointment.Caller, bool, error) { return c, true, nil }
}

// send sends method path as the caller to the handler on svc, logging to log through the request-id handler.
func send(t *testing.T, svc *stubSessions, caller func(context.Context) (appointment.Caller, bool, error), log *bytes.Buffer, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	videosession.NewHandler(svc, caller, slog.New(middleware.NewLogHandler(slog.NewJSONHandler(log, nil)))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(middleware.WithRequestID(t.Context(), "req-1"), method, path, nil))
	return rec
}

// The join answers the LiveKit URL, the token, its expiry, the other person's name and the caller's role, not
// to be stored; the service gets the caller and the appointment of the path. Nothing is logged.
func TestJoinHandler_AnswersTheToken(t *testing.T) {
	var log bytes.Buffer
	svc := &stubSessions{join: videosession.Join{
		URL: publicURL, Token: "signed-token", ExpiresAt: now.Add(5 * time.Minute), PeerName: "Ada Example", Role: videosession.RolePractitioner,
	}}
	rec := send(t, svc, signedIn(pract), &log, http.MethodPost, "/appointments/"+apptA+"/consultation/join")
	want := `{"url":"ws://localhost:7880","token":"signed-token","expiresAt":"2030-01-07T09:00:00Z","peerName":"Ada Example","role":"practitioner"}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("POST join = %d %s %v; want 200 %s, no-store", rec.Code, rec.Body, rec.Header(), want)
	}
	if svc.caller != pract || svc.asked != "join "+apptA {
		t.Errorf("the service was asked %q by %+v", svc.asked, svc.caller)
	}
	if log.Len() != 0 {
		t.Errorf("log = %s, want nothing", log.String())
	}
}

// The status, and the end's answer, is the state with the times that are set and the end's reason.
func TestStatusAndEndHandlers_AnswerTheState(t *testing.T) {
	for _, tt := range []struct {
		method, path, asked, want string
		status                    videosession.Status
	}{
		{
			method: http.MethodGet, path: "/appointments/" + apptA + "/consultation", asked: "status " + apptA,
			status: videosession.Status{State: videosession.StateNotStarted}, want: `{"state":"not_started"}`,
		},
		{
			method: http.MethodGet, path: "/appointments/" + apptA + "/consultation", asked: "status " + apptA,
			status: videosession.Status{State: videosession.StateOpen, PatientJoinedAt: start},
			want:   `{"state":"open","patientJoinedAt":"2030-01-07T09:00:00Z"}`,
		},
		{
			method: http.MethodPost, path: "/appointments/" + apptA + "/consultation/end", asked: "end " + apptA,
			status: videosession.Status{
				State: videosession.StateEnded, PatientJoinedAt: start, PractitionerJoinedAt: start.Add(time.Minute),
				EndedAt: end, EndReason: videosession.EndPractitioner,
			},
			want: `{"state":"ended","patientJoinedAt":"2030-01-07T09:00:00Z","practitionerJoinedAt":"2030-01-07T09:01:00Z",` +
				`"endedAt":"2030-01-07T09:30:00Z","endReason":"practitioner"}`,
		},
	} {
		var log bytes.Buffer
		svc := &stubSessions{status: tt.status}
		rec := send(t, svc, signedIn(admin), &log, tt.method, tt.path)
		if rec.Code != http.StatusOK || rec.Body.String() != tt.want+"\n" || rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s %s = %d %s; want 200 %s, no-store", tt.method, tt.path, rec.Code, rec.Body, tt.want)
		}
		if svc.caller != admin || svc.asked != tt.asked {
			t.Errorf("%s %s: the service was asked %q by %+v", tt.method, tt.path, svc.asked, svc.caller)
		}
	}
}

// Each refusal has its status and fixed code; nobody signed in is 401 and the service is not asked; a
// failure is the fixed 500 with the error logged, never a token or name.
func TestConsultationHandlers_MapTheErrors(t *testing.T) {
	routes := []struct{ method, path string }{
		{http.MethodPost, "/appointments/" + apptA + "/consultation/join"},
		{http.MethodPost, "/appointments/" + apptA + "/consultation/end"},
		{http.MethodGet, "/appointments/" + apptA + "/consultation"},
	}
	for _, tt := range []struct {
		err  error
		want string
		code int
	}{
		{err: videosession.ErrNotFound, code: http.StatusNotFound, want: "not_found"},
		{err: videosession.ErrConsentRequired, code: http.StatusConflict, want: "consent_required"},
		{err: videosession.ErrJoinWindowNotOpen, code: http.StatusConflict, want: "join_window_not_open"},
		{err: videosession.ErrConsultationEnded, code: http.StatusConflict, want: "consultation_ended"},
		{err: videosession.ErrPractitionerDidNotJoin, code: http.StatusConflict, want: "practitioner_did_not_join"},
		{err: videosession.ErrNotStarted, code: http.StatusConflict, want: "consultation_not_started"},
		{err: errors.New("videosession: join: the backend is down"), code: http.StatusInternalServerError, want: "internal error"},
	} {
		for _, r := range routes {
			var log bytes.Buffer
			svc := &stubSessions{err: tt.err, join: videosession.Join{Token: "signed-token", PeerName: "Ada Example"}}
			rec := send(t, svc, signedIn(me), &log, r.method, r.path)
			if rec.Code != tt.code || rec.Body.String() != `{"error":"`+tt.want+`"}`+"\n" {
				t.Errorf("%s %s with %v = %d %s; want %d %s", r.method, r.path, tt.err, rec.Code, rec.Body, tt.code, tt.want)
			}
			if strings.Contains(log.String(), "signed-token") || strings.Contains(log.String(), "Ada") {
				t.Errorf("%s %s: the log holds the token or the name: %s", r.method, r.path, log.String())
			}
			if (tt.code == http.StatusInternalServerError) != strings.Contains(log.String(), "the backend is down") {
				t.Errorf("%s %s with %v: log %s; want the error logged for a 500 only", r.method, r.path, tt.err, log.String())
			}
		}
	}
	for _, r := range routes {
		var log bytes.Buffer
		svc := &stubSessions{}
		rec := send(t, svc, func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, nil }, &log, r.method, r.path)
		if rec.Code != http.StatusUnauthorized || rec.Body.String() != `{"error":"unauthenticated"}`+"\n" || svc.calls != 0 {
			t.Errorf("%s %s signed out = %d %s after %d calls; want 401 and none", r.method, r.path, rec.Code, rec.Body, svc.calls)
		}
		rec = send(t, svc, func(context.Context) (appointment.Caller, bool, error) {
			return appointment.Caller{}, false, errBackend
		}, &log, r.method, r.path)
		if rec.Code != http.StatusInternalServerError || svc.calls != 0 {
			t.Errorf("%s %s with the caller failing = %d after %d calls; want 500 and none", r.method, r.path, rec.Code, svc.calls)
		}
	}
}
