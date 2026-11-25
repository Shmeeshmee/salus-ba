package careplan_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/careplan"
)

const planID = "0e000000-0000-4000-8000-000000000001"

var visit = time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC) // the stub step's visit

// stubPlans answers every call with plan and err, and keeps what it was asked.
type stubPlans struct {
	err     error
	called  string
	booking careplan.Booking
	draft   careplan.Draft
	version int
}

func (s *stubPlans) plan() careplan.Plan {
	day := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)
	return careplan.Plan{ID: planID, OrganizationID: clinicA, PatientID: patientA, Status: careplan.StatusActive, Title: "Synthetic plan",
		StartsOn: &day, CreatedAt: now, VersionID: 3, Steps: []careplan.Step{{ID: "s1", ServiceID: serviceIn, ServiceName: "Synthetic exam",
			VisitMode: "in-person", Position: 1, VisitStart: &visit, VisitEnd: &visit, VisitStatus: "booked"}}}
}

func (s *stubPlans) Write(_ context.Context, _ careplan.Staff, d careplan.Draft) (careplan.Plan, error) {
	s.called, s.draft = "Write", d
	return s.plan(), s.err
}

func (s *stubPlans) Change(_ context.Context, _ careplan.Staff, _ string, d careplan.Draft) (careplan.Plan, error) {
	s.called, s.draft = "Change", d
	return s.plan(), s.err
}

func (s *stubPlans) Book(_ context.Context, _ careplan.Staff, _ string, b careplan.Booking) (careplan.Plan, error) {
	s.called, s.booking = "Book", b
	return s.plan(), s.err
}

func (s *stubPlans) Cancel(_ context.Context, _ careplan.Staff, _ string, v int) (careplan.Plan, error) {
	s.called, s.version = "Cancel", v
	return s.plan(), s.err
}

func (s *stubPlans) Read(context.Context, careplan.Staff, string) (careplan.Plan, error) {
	s.called = "Read"
	return s.plan(), s.err
}

func (s *stubPlans) ForPatient(context.Context, careplan.Staff, string) ([]careplan.Plan, error) {
	s.called = "ForPatient"
	return []careplan.Plan{s.plan()}, s.err
}

func (s *stubPlans) Mine(context.Context, careplan.Patient) ([]careplan.Plan, error) {
	s.called = "Mine"
	return []careplan.Plan{s.plan()}, s.err
}

func serve(t *testing.T, svc careplan.Plans, c careplan.Caller, method, path, ifMatch, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	careplan.NewHTTPHandler(svc, func(context.Context) (careplan.Caller, bool, error) { return c, true, nil }, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

var staffCaller = careplan.Caller{Staff: admin}

func TestHandler_WritesADraftWithItsETag(t *testing.T) {
	t.Parallel()
	svc := &stubPlans{}
	body := `{"patientId":"` + patientA + `","title":"Synthetic plan","steps":[{"serviceId":"` + serviceIn + `","dayOffset":3,"visitMode":"in-person"}]}`
	rec := serve(t, svc, staffCaller, http.MethodPost, "/clinic/care-plans", "", body)
	if rec.Code != http.StatusCreated || rec.Header().Get("ETag") != `W/"3"` {
		t.Fatalf("POST = %d %q, ETag %q", rec.Code, rec.Body, rec.Header().Get("ETag"))
	}
	if svc.draft.PatientID != patientA || len(svc.draft.Steps) != 1 || svc.draft.Steps[0].DayOffset != 3 {
		t.Errorf("draft %+v", svc.draft)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got["startsOn"] != "2030-01-07" || got["status"] != "active" {
		t.Errorf("body %s, want the plan with its start as a date", rec.Body)
	}
	if b := rec.Body.String(); !strings.Contains(b, `"serviceName":"Synthetic exam"`) || !strings.Contains(b, `"visitStart":"2030-01-07T09:00:00Z"`) ||
		strings.Contains(b, "visitEnd") || strings.Contains(b, "visitStatus") {
		t.Errorf("body %s, want each step's service name and visit start, nothing more of the visit", b)
	}
}

func TestHandler_ReadsTheRequestOrRefusesIt(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		err                         error
		method, path, ifMatch, body string
		want                        string
		caller                      careplan.Caller
		code                        int
	}{
		"no If-Match":             {nil, http.MethodPost, "/clinic/care-plans/" + planID + "/cancel", "", "", "invalid_request", staffCaller, http.StatusBadRequest},
		"a malformed service id":  {nil, http.MethodPost, "/clinic/care-plans", "", `{"patientId":"` + patientA + `","title":"T","steps":[{"serviceId":"x","dayOffset":0,"visitMode":"video"}]}`, "invalid_request", staffCaller, http.StatusBadRequest},
		"a malformed start":       {nil, http.MethodPost, "/clinic/care-plans/" + planID + "/book", `W/"1"`, `{"startsOn":"7.1.2030","slotIds":["` + planID + `"]}`, "invalid_request", staffCaller, http.StatusBadRequest},
		"an invalid plan":         {careplan.ErrInvalid, http.MethodPut, "/clinic/care-plans/" + planID, `W/"1"`, `{"title":"T","steps":[]}`, "invalid_care_plan", staffCaller, http.StatusBadRequest},
		"a receptionist":          {careplan.ErrForbidden, http.MethodPost, "/clinic/care-plans/" + planID + "/book", "", "", "forbidden", careplan.Caller{Staff: desk}, http.StatusForbidden},
		"booked already":          {careplan.ErrNotDraft, http.MethodPut, "/clinic/care-plans/" + planID, `W/"1"`, `{"title":"T","steps":[]}`, "care_plan_not_draft", staffCaller, http.StatusConflict},
		"a device busy":           {careplan.ErrResourceBusy, http.MethodPost, "/clinic/care-plans/" + planID + "/book", `W/"1"`, `{"startsOn":"2030-01-07","slotIds":["` + planID + `"]}`, "resource_busy", staffCaller, http.StatusConflict},
		"a slot taken":            {careplan.ErrSlotUnavailable, http.MethodPost, "/clinic/care-plans/" + planID + "/book", `W/"1"`, `{"startsOn":"2030-01-07","slotIds":["` + planID + `"]}`, "slot_unavailable", staffCaller, http.StatusConflict},
		"ended":                   {careplan.ErrEnded, http.MethodPost, "/clinic/care-plans/" + planID + "/cancel", `W/"1"`, "", "care_plan_ended", staffCaller, http.StatusConflict},
		"an old version":          {careplan.ErrVersionConflict, http.MethodPost, "/clinic/care-plans/" + planID + "/cancel", `W/"1"`, "", "version_conflict", staffCaller, http.StatusPreconditionFailed},
		"not found":               {careplan.ErrNotFound, http.MethodGet, "/clinic/care-plans/" + planID, "", "", "not_found", staffCaller, http.StatusNotFound},
		"a patient on a clinic's": {nil, http.MethodGet, "/clinic/care-plans/" + planID, "", "", "not_found", careplan.Caller{Patient: me}, http.StatusNotFound},
		"staff on the patient's":  {nil, http.MethodGet, "/me/care-plans", "", "", "profile_required", staffCaller, http.StatusForbidden},
		"an unknown field":        {nil, http.MethodPost, "/clinic/care-plans", "", `{"title":"T","steps":[],"x":1}`, "invalid_request", staffCaller, http.StatusBadRequest},
		"the backend fails":       {io.ErrUnexpectedEOF, http.MethodGet, "/clinic/patients/" + patientA + "/care-plans", "", "", "", staffCaller, http.StatusInternalServerError},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec := serve(t, &stubPlans{err: tc.err}, tc.caller, tc.method, tc.path, tc.ifMatch, tc.body)
			if rec.Code != tc.code || tc.want != "" && !strings.Contains(rec.Body.String(), `"`+tc.want+`"`) {
				t.Errorf("%s %s = %d %s, want %d %s", tc.method, tc.path, rec.Code, rec.Body, tc.code, tc.want)
			}
		})
	}
}

func TestHandler_PassesTheVersionAndTheBooking(t *testing.T) {
	t.Parallel()
	svc := &stubPlans{}
	rec := serve(t, svc, staffCaller, http.MethodPost, "/clinic/care-plans/"+planID+"/book", `W/"2"`, `{"startsOn":"2030-01-07","slotIds":["`+planID+`"]}`)
	if rec.Code != http.StatusOK || svc.booking.Version != 2 || len(svc.booking.SlotIDs) != 1 || svc.booking.StartsOn.Day() != 7 {
		t.Fatalf("book = %d %s, booking %+v", rec.Code, rec.Body, svc.booking)
	}
	rec = serve(t, svc, careplan.Caller{Patient: me}, http.MethodGet, "/me/care-plans", "", "")
	if rec.Code != http.StatusOK || svc.called != "Mine" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("mine = %d %s", rec.Code, rec.Body)
	}
}
