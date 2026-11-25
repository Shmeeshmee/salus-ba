package datarequest_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/datarequest"
)

// stubRequests answers every call with its request and error, and records what it was asked.
type stubRequests struct {
	err     error
	filing  datarequest.Filing
	reason  datarequest.ExtensionReason
	kind    datarequest.Kind
	id      string
	request datarequest.Request
	answer  datarequest.Answer
	history []datarequest.Change
	version int
}

func (s *stubRequests) File(_ context.Context, _ datarequest.Patient, f datarequest.Filing) (datarequest.Request, error) {
	s.filing = f
	return s.request, s.err
}

func (s *stubRequests) Mine(context.Context, datarequest.Patient) ([]datarequest.Request, error) {
	return []datarequest.Request{s.request}, s.err
}

func (s *stubRequests) Withdraw(_ context.Context, _ datarequest.Patient, id string, version int) (datarequest.Request, error) {
	s.id, s.version = id, version
	return s.request, s.err
}

func (s *stubRequests) FileOnBehalf(_ context.Context, _ datarequest.Staff, id string, k datarequest.Kind) (datarequest.Request, error) {
	s.id, s.kind = id, k
	return s.request, s.err
}

func (s *stubRequests) ClinicRequests(context.Context, datarequest.Staff) ([]datarequest.Request, error) {
	return []datarequest.Request{s.request}, s.err
}

func (s *stubRequests) ClinicRequest(_ context.Context, _ datarequest.Staff, id string) (datarequest.Request, []datarequest.Change, error) {
	s.id = id
	return s.request, s.history, s.err
}

func (s *stubRequests) Answer(_ context.Context, _ datarequest.Staff, id string, a datarequest.Answer) (datarequest.Request, error) {
	s.id, s.answer = id, a
	return s.request, s.err
}

func (s *stubRequests) Extend(_ context.Context, _ datarequest.Staff, id string, r datarequest.ExtensionReason, version int) (datarequest.Request, error) {
	s.id, s.reason, s.version = id, r, version
	return s.request, s.err
}

const reqID = "44444444-4444-4444-8444-444444444444"

var received = datarequest.Request{
	ID: reqID, AccountID: accountA, PatientID: profileA, OrganizationID: clinicA, FiledBy: accountA,
	Kind: datarequest.KindErasure, Status: datarequest.StatusReceived, Version: 1,
	ReceivedAt: time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC), DueAt: time.Date(2026, 12, 24, 22, 59, 59, 0, time.UTC),
}

// serveAs serves one request as the signed-in account who (a patient for an empty clinic).
func serveAs(t *testing.T, svc datarequest.Requests, who datarequest.Staff, method, path, body, ifMatch string) *httptest.ResponseRecorder {
	t.Helper()
	caller := func(context.Context) (datarequest.Staff, bool, error) { return who, who.AccountID != "", nil }
	mux := http.NewServeMux()
	datarequest.NewHandler(svc, caller, slog.New(slog.DiscardHandler)).Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

var (
	asPatient = datarequest.Staff{AccountID: accountA}
	asAdmin   = datarequest.Staff{AccountID: adminA, OrganizationID: clinicA, Admin: true}
)

func TestHandler_ThePatientsRoutes(t *testing.T) {
	t.Parallel()
	stub := &stubRequests{request: received}
	rec := serveAs(t, stub, asPatient, http.MethodPost, "/me/data-requests", `{"kind":"erasure","target":"clinic","organizationId":"`+clinicA+`"}`, "")
	want := `{"receivedAt":"2026-11-24T08:00:00Z","dueAt":"2026-12-24T22:59:59Z","id":"` + reqID + `","kind":"erasure","target":"clinic","organizationId":"` + clinicA + `","status":"received","version":1,"filedByPatient":true}`
	if rec.Code != http.StatusCreated || strings.TrimSpace(rec.Body.String()) != want || rec.Header().Get("ETag") != `W/"1"` {
		t.Errorf("POST /me/data-requests = %d %s %v; want 201 %s", rec.Code, rec.Body, rec.Header(), want)
	}
	if stub.filing != (datarequest.Filing{Kind: datarequest.KindErasure, Target: datarequest.TargetClinic, OrganizationID: clinicA}) {
		t.Errorf("filed %+v", stub.filing)
	}
	rec = serveAs(t, stub, asPatient, http.MethodGet, "/me/data-requests", "", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "["+want+"]" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET /me/data-requests = %d %s", rec.Code, rec.Body)
	}
	rec = serveAs(t, stub, asPatient, http.MethodPost, "/me/data-requests/"+reqID+"/withdraw", "", `W/"3"`)
	if rec.Code != http.StatusOK || stub.id != reqID || stub.version != 3 {
		t.Errorf("withdraw = %d %s, asked %s at %d", rec.Code, rec.Body, stub.id, stub.version)
	}
	if rec := serveAs(t, stub, asPatient, http.MethodPost, "/me/data-requests/"+reqID+"/withdraw", "", ""); rec.Code != http.StatusPreconditionRequired || !strings.Contains(rec.Body.String(), "if_match_required") {
		t.Errorf("withdraw without If-Match = %d %s, want 428", rec.Code, rec.Body)
	}
	if rec := serveAs(t, stub, asAdmin, http.MethodGet, "/me/data-requests", "", ""); rec.Code != http.StatusForbidden {
		t.Errorf("GET /me/data-requests as staff = %d, want 403", rec.Code)
	}
	if rec := serveAs(t, stub, datarequest.Staff{}, http.MethodGet, "/me/data-requests", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /me/data-requests signed out = %d, want 401", rec.Code)
	}
	if rec := serveAs(t, stub, asPatient, http.MethodPost, "/me/data-requests", `{"kind":`, ""); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_data_request") {
		t.Errorf("a broken body = %d %s, want 400 invalid_data_request", rec.Code, rec.Body)
	}
}

func TestHandler_TheClinicsRoutes(t *testing.T) {
	t.Parallel()
	stub := &stubRequests{request: received, history: []datarequest.Change{{Status: datarequest.StatusReceived, ChangedAt: received.ReceivedAt}}}
	rec := serveAs(t, stub, asAdmin, http.MethodGet, "/clinic/data-requests/"+reqID, "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"patientId":"`+profileA+`"`) ||
		!strings.Contains(rec.Body.String(), `"history":[{"changedAt":"2026-11-24T08:00:00Z","status":"received"}]`) || rec.Header().Get("ETag") != `W/"1"` {
		t.Errorf("GET one = %d %s", rec.Code, rec.Body)
	}
	rec = serveAs(t, stub, asAdmin, http.MethodGet, "/clinic/data-requests", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"patientId"`) || strings.Contains(rec.Body.String(), "history") {
		t.Errorf("GET list = %d %s", rec.Code, rec.Body)
	}
	rec = serveAs(t, stub, asAdmin, http.MethodPost, "/clinic/data-requests/"+reqID+"/answer", `{"status":"refused","code":"retention_law"}`, `W/"2"`)
	if rec.Code != http.StatusOK || stub.answer != (datarequest.Answer{Status: datarequest.StatusRefused, Code: datarequest.AnswerRetentionLaw, Version: 2}) {
		t.Errorf("answer = %d %s, asked %+v", rec.Code, rec.Body, stub.answer)
	}
	rec = serveAs(t, stub, asAdmin, http.MethodPost, "/clinic/data-requests/"+reqID+"/extend", `{"reason":"complex"}`, `W/"1"`)
	if rec.Code != http.StatusOK || stub.reason != datarequest.ExtensionComplex || stub.version != 1 {
		t.Errorf("extend = %d %s", rec.Code, rec.Body)
	}
	rec = serveAs(t, stub, asAdmin, http.MethodPost, "/clinic/patients/"+registered+"/data-requests", `{"kind":"access"}`, "")
	if rec.Code != http.StatusCreated || stub.id != registered || stub.kind != datarequest.KindAccess {
		t.Errorf("file on behalf = %d %s", rec.Code, rec.Body)
	}
	practitioner := datarequest.Staff{AccountID: deskA, OrganizationID: clinicA}
	if rec := serveAs(t, &stubRequests{err: datarequest.ErrForbidden}, practitioner, http.MethodPost, "/clinic/patients/"+registered+"/data-requests", "{", ""); rec.Code != http.StatusForbidden {
		t.Errorf("a practitioner files on behalf with a broken body = %d %s, want 403", rec.Code, rec.Body)
	}
	// The desk is refused an answer before its precondition and body are read: as any route answers a stranger.
	desk := datarequest.Staff{AccountID: deskA, OrganizationID: clinicA, Reception: true}
	if rec := serveAs(t, &stubRequests{err: datarequest.ErrForbidden}, desk, http.MethodPost, "/clinic/data-requests/"+reqID+"/answer", "", ""); rec.Code != http.StatusForbidden {
		t.Errorf("answer as the desk without If-Match = %d %s, want 403", rec.Code, rec.Body)
	}
}

// The domain's errors map to the contract's status codes and error codes; none carries the error's text.
func TestHandler_ErrorsAreTheContractsCodes(t *testing.T) {
	t.Parallel()
	for err, want := range map[error]struct {
		code   string
		status int
	}{
		datarequest.ErrInvalid:         {"invalid_data_request", http.StatusBadRequest},
		datarequest.ErrNotFound:        {"not_found", http.StatusNotFound},
		datarequest.ErrForbidden:       {"forbidden", http.StatusForbidden},
		datarequest.ErrClosed:          {"data_request_closed", http.StatusConflict},
		datarequest.ErrAlreadyExtended: {"already_extended", http.StatusConflict},
		datarequest.ErrVersionConflict: {"version_conflict", http.StatusPreconditionFailed},
	} {
		rec := serveAs(t, &stubRequests{err: err}, asAdmin, http.MethodPost, "/clinic/data-requests/"+reqID+"/extend", `{"reason":"complex"}`, `W/"1"`)
		if rec.Code != want.status || strings.TrimSpace(rec.Body.String()) != `{"error":"`+want.code+`"}` {
			t.Errorf("%v: %d %s, want %d %s", err, rec.Code, rec.Body, want.status, want.code)
		}
	}
}
