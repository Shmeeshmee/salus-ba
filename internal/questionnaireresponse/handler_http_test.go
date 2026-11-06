package questionnaireresponse_test

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
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
)

// stubResponses answers every call with out and err, and keeps what it was asked.
type stubResponses struct {
	err     error
	called  string
	caller  appointment.Caller
	id      string
	other   string // the questionnaire of Request
	answers []questionnaireresponse.Answer
	out     questionnaireresponse.Response
	version int32
}

func (s *stubResponses) Request(_ context.Context, c appointment.Caller, appointmentID, questionnaireID string) (questionnaireresponse.Response, error) {
	s.called, s.caller, s.id, s.other = "Request", c, appointmentID, questionnaireID
	return s.out, s.err
}

func (s *stubResponses) SaveAnswers(_ context.Context, c appointment.Caller, id string, a []questionnaireresponse.Answer, version int32) (questionnaireresponse.Response, error) {
	s.called, s.caller, s.id, s.answers, s.version = "SaveAnswers", c, id, a, version
	return s.out, s.err
}

func (s *stubResponses) Submit(_ context.Context, c appointment.Caller, id string, version int32) (questionnaireresponse.Response, error) {
	s.called, s.caller, s.id, s.version = "Submit", c, id, version
	return s.out, s.err
}

func (s *stubResponses) Withdraw(_ context.Context, c appointment.Caller, id string) (questionnaireresponse.Response, error) {
	s.called, s.caller, s.id = "Withdraw", c, id
	return s.out, s.err
}

// Synthetic ids of the handler tests, valid UUIDs (the golden files are validated as FHIR).
const (
	hResponse      = "c5000000-0000-4000-8000-0000000000c1"
	hAppointment   = "c6000000-0000-4000-8000-0000000000c1"
	hQuestionnaire = "c7000000-0000-4000-8000-0000000000c1"
	hSeries        = "c8000000-0000-4000-8000-0000000000c1"
	hPatient       = "c9000000-0000-4000-8000-0000000000c1"
	hClinic        = "ca000000-0000-4000-8000-0000000000c1"
	hRole          = "cb000000-0000-4000-8000-0000000000c1"

	// A synthetic answer: asserted absent from every log.
	placeholderAnswer = "Placeholder answer text"
)

var (
	hNow  = time.Date(2026, 11, 6, 9, 30, 0, 0, time.UTC)
	hDue  = time.Date(2026, 11, 9, 14, 0, 0, 0, time.UTC)
	saved = questionnaireresponse.Response{
		ID: hResponse, OrganizationID: hClinic, AppointmentID: hAppointment, PatientID: hPatient, PractitionerRoleID: hRole,
		QuestionnaireID: hQuestionnaire, SeriesID: hSeries, Source: questionnaireresponse.SourceAdded, Status: questionnaireresponse.StatusInProgress,
		Answers: []questionnaireresponse.Answer{{LinkID: "q1", Values: []questionnaireresponse.Value{{String: placeholderAnswer}}}},
		DueAt:   hDue, LastUpdated: hNow, VersionID: 3,
	}
)

const savedJSON = `{"id":"` + hResponse + `","appointmentId":"` + hAppointment + `","questionnaireId":"` + hQuestionnaire +
	`","status":"in-progress","source":"added","dueAt":"2026-11-09T14:00:00Z","lastUpdated":"2026-11-06T09:30:00Z",` +
	`"answers":[{"linkId":"q1","values":[{"valueString":"` + placeholderAnswer + `"}]}],"versionId":3}` + "\n"

type call struct {
	method, path, ifMatch, body string
}

// serveJSON sends r as the patient me, or as nobody when signedIn is false, logging to log.
func serveJSON(t *testing.T, svc *stubResponses, signedIn bool, log *bytes.Buffer, r call) *httptest.ResponseRecorder {
	t.Helper()
	h := questionnaireresponse.NewHTTPHandler(svc, func(context.Context) (appointment.Caller, bool, error) { return me, signedIn, nil },
		slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), r.method, r.path, strings.NewReader(r.body))
	if r.ifMatch != "" {
		req.Header.Set("If-Match", r.ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const answersBody = `{"answers":[{"linkId":"q1","values":[{"valueString":"` + placeholderAnswer + `"}]},` +
	`{"linkId":"q2","values":[{"valueBoolean":true}]},{"linkId":"q3","values":[{"valueInteger":-7}]},` +
	`{"linkId":"q4","values":[{"valueDecimal":36.65}]},{"linkId":"q5","values":[{"valueDate":"2026-10-01"}]}]}`

func TestHTTPHandler_EachRouteCallsTheServiceAndAnswersTheResponse(t *testing.T) {
	t.Parallel()
	seven, yes := int32(-7), true
	cases := []struct {
		check  func(s *stubResponses) bool
		name   string
		call   call
		status int
	}{
		{
			name: "request", status: http.StatusCreated,
			call: call{method: http.MethodPost, path: "/appointments/" + hAppointment + "/questionnaires", body: `{"questionnaireId":"` + hQuestionnaire + `"}`},
			check: func(s *stubResponses) bool {
				return s.called == "Request" && s.id == hAppointment && s.other == hQuestionnaire
			},
		},
		{
			name: "save answers", status: http.StatusOK,
			call: call{method: http.MethodPut, path: "/questionnaire-responses/" + hResponse + "/answers", ifMatch: `W/"3"`, body: answersBody},
			check: func(s *stubResponses) bool {
				want := []questionnaireresponse.Answer{
					{LinkID: "q1", Values: []questionnaireresponse.Value{{String: placeholderAnswer}}},
					{LinkID: "q2", Values: []questionnaireresponse.Value{{Boolean: &yes}}},
					{LinkID: "q3", Values: []questionnaireresponse.Value{{Integer: &seven}}},
					{LinkID: "q4", Values: []questionnaireresponse.Value{{Decimal: "36.65"}}},
					{LinkID: "q5", Values: []questionnaireresponse.Value{{Date: "2026-10-01"}}},
				}
				return s.called == "SaveAnswers" && s.id == hResponse && s.version == 3 && equalAnswers(s.answers, want)
			},
		},
		{
			name: "submit", status: http.StatusOK,
			call:  call{method: http.MethodPost, path: "/questionnaire-responses/" + hResponse + "/submit", ifMatch: `"3"`},
			check: func(s *stubResponses) bool { return s.called == "Submit" && s.id == hResponse && s.version == 3 },
		},
		{
			name: "withdraw", status: http.StatusOK,
			call:  call{method: http.MethodPost, path: "/questionnaire-responses/" + hResponse + "/withdraw"},
			check: func(s *stubResponses) bool { return s.called == "Withdraw" && s.id == hResponse },
		},
	}
	for _, c := range cases {
		svc := &stubResponses{out: saved}
		var log bytes.Buffer
		rec := serveJSON(t, svc, true, &log, c.call)
		if rec.Code != c.status || rec.Body.String() != savedJSON || rec.Header().Get("ETag") != `W/"3"` {
			t.Errorf("%s = %d %s (ETag %q), want %d %s", c.name, rec.Code, rec.Body, rec.Header().Get("ETag"), c.status, savedJSON)
		}
		if !c.check(svc) || svc.caller != me {
			t.Errorf("%s called %+v, want its arguments as the signed-in caller", c.name, svc)
		}
		if log.Len() != 0 {
			t.Errorf("%s logged %q, want nothing", c.name, log.String())
		}
	}
}

// A completed response carries authored; a withdrawn one an empty list of answers.
func TestHTTPHandler_WritesAuthoredOnceSubmittedAndAnEmptyListOfAnswers(t *testing.T) {
	t.Parallel()
	done := saved
	done.Status, done.AuthoredAt, done.Answers = questionnaireresponse.StatusCompleted, hNow, nil
	rec := serveJSON(t, &stubResponses{out: done}, true, &bytes.Buffer{}, call{method: http.MethodPost, path: "/questionnaire-responses/" + hResponse + "/submit", ifMatch: `W/"3"`})
	want := `"status":"completed","source":"added","dueAt":"2026-11-09T14:00:00Z","authored":"2026-11-06T09:30:00Z","lastUpdated":"2026-11-06T09:30:00Z","answers":[],`
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), want) {
		t.Errorf("submit = %d %s, want it to hold %s", rec.Code, rec.Body, want)
	}
}

func TestHTTPHandler_RefusesNobodyAndAMalformedRequestWithoutCallingTheService(t *testing.T) {
	t.Parallel()
	answers := "/questionnaire-responses/" + hResponse + "/answers"
	request := "/appointments/" + hAppointment + "/questionnaires"
	submit := "/questionnaire-responses/" + hResponse + "/submit"
	for _, c := range []struct {
		call     call
		name     string
		code     string
		status   int
		signedIn bool
	}{
		{call{method: http.MethodPost, path: request, body: `{"questionnaireId":"x"}`}, "nobody signed in", "unauthenticated", http.StatusUnauthorized, false},
		{call{method: http.MethodPost, path: "/questionnaire-responses/x/withdraw"}, "nobody signed in, withdraw", "unauthenticated", http.StatusUnauthorized, false},
		{call{method: http.MethodPost, path: request, body: `{}`}, "request without a questionnaire", "invalid_request", http.StatusBadRequest, true},
		{call{method: http.MethodPost, path: request, body: `{"questionnaireId":"x","note":"y"}`}, "request with an unknown field", "invalid_request", http.StatusBadRequest, true},
		{call{method: http.MethodPut, path: answers, body: answersBody}, "save without If-Match", "invalid_request", http.StatusBadRequest, true},
		{call{method: http.MethodPut, path: answers, ifMatch: `W/"three"`, body: answersBody}, "save with a malformed If-Match", "invalid_request", http.StatusBadRequest, true},
		{call{method: http.MethodPut, path: answers, ifMatch: `W/"3"`, body: `{}`}, "save without answers", "invalid_request", http.StatusBadRequest, true},
		{call{method: http.MethodPut, path: answers, ifMatch: `W/"3"`, body: `{"answers":[{"linkId":"q1","values":[{"valueCoding":{}}]}]}`}, "save with an unknown value", "invalid_request", http.StatusBadRequest, true},
		{call{method: http.MethodPut, path: answers, ifMatch: `W/"3"`, body: `{"answers":[{"linkId":"q1","values":[{"valueInteger":2147483648}]}]}`}, "save with an integer outside 32 bits", "invalid_request", http.StatusBadRequest, true},
		{call{method: http.MethodPut, path: answers, ifMatch: `W/"3"`, body: `{"answers":[]}{}`}, "save with two objects", "invalid_request", http.StatusBadRequest, true},
		{call{method: http.MethodPost, path: submit}, "submit without If-Match", "invalid_request", http.StatusBadRequest, true},
	} {
		svc := &stubResponses{}
		rec := serveJSON(t, svc, c.signedIn, &bytes.Buffer{}, c.call)
		if want := `{"error":"` + c.code + `"}` + "\n"; rec.Code != c.status || rec.Body.String() != want {
			t.Errorf("%s = %d %s, want %d %s", c.name, rec.Code, rec.Body, c.status, want)
		}
		if svc.called != "" {
			t.Errorf("%s called %s", c.name, svc.called)
		}
	}
}

// The body is read up to 256 KiB (SPEC.md §2 "Limits"): at the limit it reaches the service, one byte past it
// is 400.
func TestHTTPHandler_ReadsTheAnswersUpTo256KiB(t *testing.T) {
	t.Parallel()
	const limit = 256 << 10
	body := func(size int) string {
		head, tail := `{"answers":[{"linkId":"q1","values":[{"valueString":"`, `"}]}]}`
		return head + strings.Repeat("x", size-len(head)-len(tail)) + tail
	}
	path := "/questionnaire-responses/" + hResponse + "/answers"
	svc := &stubResponses{out: saved}
	if rec := serveJSON(t, svc, true, &bytes.Buffer{}, call{method: http.MethodPut, path: path, ifMatch: `W/"3"`, body: body(limit)}); rec.Code != http.StatusOK {
		t.Errorf("a body of 256 KiB = %d %s, want 200", rec.Code, rec.Body)
	}
	svc = &stubResponses{out: saved}
	if rec := serveJSON(t, svc, true, &bytes.Buffer{}, call{method: http.MethodPut, path: path, ifMatch: `W/"3"`, body: body(limit + 1)}); rec.Code != http.StatusBadRequest || svc.called != "" {
		t.Errorf("a body one byte past 256 KiB = %d, called %q; want 400 without a call", rec.Code, svc.called)
	}
}

func TestHTTPHandler_MapsEachDomainErrorToItsStatusAndCode(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		err    error
		code   string
		status int
	}{
		{questionnaireresponse.ErrForbidden, "forbidden", http.StatusForbidden},
		{questionnaireresponse.ErrNotFound, "not_found", http.StatusNotFound},
		{questionnaireresponse.ErrAlreadyRequested, "already_requested", http.StatusConflict},
		{questionnaireresponse.ErrNotActive, "not_active", http.StatusConflict},
		{questionnaireresponse.ErrAppointmentStarted, "appointment_started", http.StatusConflict},
		{questionnaireresponse.ErrAppointmentClosed, "appointment_closed", http.StatusConflict},
		{questionnaireresponse.ErrConsentRequired, "consent_required", http.StatusConflict},
		{questionnaireresponse.ErrSubmitted, "submitted", http.StatusConflict},
		{questionnaireresponse.ErrWithdrawn, "withdrawn", http.StatusConflict},
		{questionnaireresponse.ErrPastDue, "past_due", http.StatusConflict},
		{questionnaireresponse.ErrConflict, "version_conflict", http.StatusPreconditionFailed},
	} {
		for _, r := range []call{
			{method: http.MethodPost, path: "/appointments/" + hAppointment + "/questionnaires", body: `{"questionnaireId":"` + hQuestionnaire + `"}`},
			{method: http.MethodPut, path: "/questionnaire-responses/" + hResponse + "/answers", ifMatch: `W/"3"`, body: answersBody},
			{method: http.MethodPost, path: "/questionnaire-responses/" + hResponse + "/submit", ifMatch: `W/"3"`},
			{method: http.MethodPost, path: "/questionnaire-responses/" + hResponse + "/withdraw"},
		} {
			var log bytes.Buffer
			rec := serveJSON(t, &stubResponses{err: c.err}, true, &log, r)
			if want := `{"error":"` + c.code + `"}` + "\n"; rec.Code != c.status || rec.Body.String() != want {
				t.Errorf("%s %s with %v = %d %s, want %d %s", r.method, r.path, c.err, rec.Code, rec.Body, c.status, want)
			}
			if log.Len() != 0 {
				t.Errorf("%s %s with %v logged %q, want nothing", r.method, r.path, c.err, log.String())
			}
		}
	}
}

// 422 answers_invalid carries the offending linkIds, an empty list when none is named.
func TestHTTPHandler_AnswersInvalidCarriesTheLinkIDs(t *testing.T) {
	t.Parallel()
	path := "/questionnaire-responses/" + hResponse + "/submit"
	for _, c := range []struct {
		want string
		ids  []string
	}{
		{`{"error":"answers_invalid","linkIds":["q2","q7"]}` + "\n", []string{"q2", "q7"}},
		{`{"error":"answers_invalid","linkIds":[]}` + "\n", nil},
	} {
		rec := serveJSON(t, &stubResponses{err: &questionnaireresponse.AnswersError{LinkIDs: c.ids}}, true, &bytes.Buffer{}, call{method: http.MethodPost, path: path, ifMatch: `W/"3"`})
		if rec.Code != http.StatusUnprocessableEntity || rec.Body.String() != c.want {
			t.Errorf("submit with %v = %d %s, want 422 %s", c.ids, rec.Code, rec.Body, c.want)
		}
	}
}

// Any other error is a 500 with the fixed message; the log holds the error and never an answer, even when
// the request carried one.
func TestHTTPHandler_AnInternalErrorIs500AndTheLogHoldsNoAnswer(t *testing.T) {
	t.Parallel()
	var log bytes.Buffer
	svc := &stubResponses{err: errors.New("questionnaireresponse: save answers: the database is down")}
	rec := serveJSON(t, svc, true, &log, call{method: http.MethodPut, path: "/questionnaire-responses/" + hResponse + "/answers", ifMatch: `W/"3"`, body: answersBody})
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "database") {
		t.Errorf("an internal error = %d %s, want 500 without its text", rec.Code, rec.Body)
	}
	if !strings.Contains(log.String(), "the database is down") {
		t.Errorf("the log %q does not hold the error", log.String())
	}
	for _, s := range []string{placeholderAnswer, "36.65", "2026-10-01"} {
		if strings.Contains(log.String(), s) {
			t.Errorf("the log holds the answer %q: %s", s, log.String())
		}
	}
	errCaller := questionnaireresponse.NewHTTPHandler(&stubResponses{}, func(context.Context) (appointment.Caller, bool, error) {
		return appointment.Caller{}, false, errors.New("the session store is down")
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	mux := http.NewServeMux()
	errCaller.Routes(mux)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/questionnaire-responses/"+hResponse+"/withdraw", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("a failing caller = %d, want 500", rec.Code)
	}
}
