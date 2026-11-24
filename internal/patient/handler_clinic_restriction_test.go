package patient_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Shmeeshmee/salus/internal/patient"
)

func (s *stubClinic) Restrict(_ context.Context, c patient.Caller, id, requestID string) error {
	s.caller, s.asked, s.in.ID = c, id, requestID
	return s.err
}

func (s *stubClinic) LiftRestriction(_ context.Context, c patient.Caller, id string) error {
	s.caller, s.asked = c, id
	return s.err
}

// PUT restricts, with or without the request it answers, DELETE lifts; both 204, the service's refusals as the
// contract's 403 and 404, a body that is not the schema 400 (docs/clinic-cms/WAVE5.md D4).
func TestClinicHandler_RestrictsAndLifts(t *testing.T) {
	t.Parallel()
	target := "/clinic/patients/" + patientA + "/restriction"
	tests := []struct {
		err                   error
		name, method, body    string
		asked, request, wants string
		code                  int
	}{
		{name: "restrict", method: http.MethodPut, code: http.StatusNoContent, asked: patientA},
		{name: "restrict for a request", method: http.MethodPut, body: `{"dataRequestId":"` + requestA + `"}`, code: http.StatusNoContent, asked: patientA, request: requestA},
		{name: "a bad body", method: http.MethodPut, body: `{"reason":"x"}`, code: http.StatusBadRequest, wants: `{"error":"invalid_request"}`},
		{name: "lift", method: http.MethodDelete, code: http.StatusNoContent, asked: patientA},
		{name: "not the admin", method: http.MethodPut, err: patient.ErrForbidden, code: http.StatusForbidden, asked: patientA},
		{name: "not the clinic's", method: http.MethodDelete, err: patient.ErrNotFound, code: http.StatusNotFound, asked: patientA},
	}
	for _, tt := range tests {
		svc := &stubClinic{err: tt.err}
		rec := serveClinicMatch(t, svc, tt.method, target, tt.body, "")
		if rec.Code != tt.code || svc.asked != tt.asked || svc.in.ID != tt.request {
			t.Errorf("%s: %d %s, asked %q for %q; want %d, %q for %q", tt.name, rec.Code, rec.Body, svc.asked, svc.in.ID, tt.code, tt.asked, tt.request)
		}
		if tt.wants != "" && rec.Body.String() != tt.wants+"\n" {
			t.Errorf("%s: body %s, want %s", tt.name, rec.Body, tt.wants)
		}
	}
}
