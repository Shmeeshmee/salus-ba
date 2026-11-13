package medicationstatement_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
)

// stubWriter answers with err, or the entry at version 2, and keeps what it was asked.
type stubWriter struct {
	err       error
	call      string
	id        string
	encounter string
	caller    appointment.Caller
	in        medicationstatement.Input
	version   int32
}

func (s *stubWriter) answer(call, id string) (medicationstatement.MedicationStatement, error) {
	s.call, s.id = call, id
	if s.err != nil {
		return medicationstatement.MedicationStatement{}, s.err
	}
	return medicationstatement.MedicationStatement{ID: idOwn, VersionID: 2}, nil
}

func (s *stubWriter) Add(_ context.Context, c appointment.Caller, in medicationstatement.Input) (medicationstatement.MedicationStatement, error) {
	s.caller, s.in = c, in
	return s.answer("add", "")
}

func (s *stubWriter) Record(_ context.Context, c appointment.Caller, encounterID string, in medicationstatement.Input) (medicationstatement.MedicationStatement, error) {
	s.caller, s.in, s.encounter = c, in, encounterID
	return s.answer("record", "")
}

func (s *stubWriter) Update(_ context.Context, c appointment.Caller, id string, version int32, in medicationstatement.Input) (medicationstatement.MedicationStatement, error) {
	s.caller, s.in, s.version = c, in, version
	return s.answer("update", id)
}

func (s *stubWriter) Remove(_ context.Context, c appointment.Caller, id string, version int32) error {
	s.caller, s.version = c, version
	_, err := s.answer("remove", id)
	return err
}

func (s *stubWriter) Confirm(_ context.Context, c appointment.Caller, id string, version int32) (medicationstatement.MedicationStatement, error) {
	s.caller, s.version = c, version
	return s.answer("confirm", id)
}

func (s *stubWriter) AddToProfile(_ context.Context, c appointment.Caller, id string, version int32) (medicationstatement.MedicationStatement, error) {
	s.caller, s.version = c, version
	return s.answer("profile", id)
}

func serve(t *testing.T, svc *stubWriter, method, path, ifMatch, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	caller := func(context.Context) (appointment.Caller, bool, error) { return patient, true, nil }
	medicationstatement.NewHTTPHandler(svc, caller, slog.New(slog.DiscardHandler)).Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const body = `{"termId":"` + termDrug + `","text":"Placeholder substance","status":"active","dosage":{"doseValue":500,"doseUnit":"mg","frequency":2,"period":1,"periodUnit":"d"}}`

// Each route decodes its body and If-Match, calls the service and answers EntryWritten with the ETag (204 for a
// removal).
func TestHTTPHandler_Routes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		method, path, ifMatch, body, call, id string
		status                                int
	}{
		{http.MethodPost, "/me/medications", "", body, "add", "", http.StatusCreated},
		{http.MethodPost, "/encounters/" + encA + "/medications", "", body, "record", "", http.StatusCreated},
		{http.MethodPut, "/medications/" + idOwn, `W/"1"`, body, "update", idOwn, http.StatusOK},
		{http.MethodDelete, "/medications/" + idOwn, `W/"1"`, "", "remove", idOwn, http.StatusNoContent},
		{http.MethodPost, "/medications/" + idOwn + "/confirm", `"1"`, "", "confirm", idOwn, http.StatusOK},
		{http.MethodPost, "/medications/" + idOwn + "/profile", `W/"1"`, "", "profile", idOwn, http.StatusOK},
	} {
		svc := &stubWriter{}
		rec := serve(t, svc, tc.method, tc.path, tc.ifMatch, tc.body)
		if rec.Code != tc.status || svc.call != tc.call || svc.id != tc.id || svc.caller != patient {
			t.Errorf("%s %s: %d, called %s(%s); want %d %s", tc.method, tc.path, rec.Code, svc.call, svc.id, tc.status, tc.call)
			continue
		}
		if tc.ifMatch != "" && svc.version != 1 {
			t.Errorf("%s %s: version %d, want 1", tc.method, tc.path, svc.version)
		}
		if tc.status == http.StatusNoContent {
			continue
		}
		if got := strings.TrimSpace(rec.Body.String()); got != `{"id":"`+idOwn+`","versionId":"2"}` || rec.Header().Get("ETag") != `W/"2"` {
			t.Errorf("%s %s: %s, ETag %q", tc.method, tc.path, got, rec.Header().Get("ETag"))
		}
		if tc.body != "" && (svc.in.TermID != termDrug || svc.in.Dosage.DoseUnit != "mg") {
			t.Errorf("%s %s: input %+v", tc.method, tc.path, svc.in)
		}
	}
}

// A body that is not the input, or an unreadable If-Match, is 400 without calling the service; each domain error
// has its status and code.
func TestHTTPHandler_Errors(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		method, path, ifMatch, body string
		status                      int
	}{
		"an unknown field":    {http.MethodPost, "/me/medications", "", `{"text":"x","status":"active","dosage":{},"extra":1}`, http.StatusBadRequest},
		"two objects":         {http.MethodPost, "/me/medications", "", body + body, http.StatusBadRequest},
		"no If-Match":         {http.MethodPut, "/medications/" + idOwn, "", body, http.StatusBadRequest},
		"an unreadable one":   {http.MethodDelete, "/medications/" + idOwn, `W/"x"`, "", http.StatusBadRequest},
		"a version too large": {http.MethodPost, "/medications/" + idOwn + "/confirm", `"99999999999"`, "", http.StatusBadRequest},
	} {
		svc := &stubWriter{}
		if rec := serve(t, svc, tc.method, tc.path, tc.ifMatch, tc.body); rec.Code != tc.status || svc.call != "" {
			t.Errorf("%s: %d, called %q; want %d and no call", name, rec.Code, svc.call, tc.status)
		}
	}
	for err, want := range map[error]string{
		medicationstatement.ErrInvalid:         `400 {"error":"invalid_request"}`,
		medicationstatement.ErrNotFound:        `404 {"error":"not_found"}`,
		medicationstatement.ErrForbidden:       `403 {"error":"forbidden"}`,
		medicationstatement.ErrVersionConflict: `412 {"error":"version_conflict"}`,
		context.Canceled:                       `500 {"error":"internal error"}`,
	} {
		rec := serve(t, &stubWriter{err: err}, http.MethodPut, "/medications/"+idOwn, `W/"1"`, body)
		if got := strings.TrimSpace(rec.Body.String()); got != want[4:] || want[:3] != strconv.Itoa(rec.Code) {
			t.Errorf("%v: %d %s, want %s", err, rec.Code, got, want)
		}
	}
}
