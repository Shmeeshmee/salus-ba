package consent_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/consent"
)

func (s *stubConsents) Candidates(_ context.Context, c consent.Caller, organizationID string) (consent.Preview, error) {
	s.caller, s.arg = c, organizationID
	s.calls = append(s.calls, "Candidates")
	return s.preview, s.err
}

func (s *stubConsents) SetSharing(_ context.Context, c consent.Caller, organizationID string, in consent.SetSharing, version string) (consent.Sharing, error) {
	s.caller, s.arg, s.set, s.version = c, organizationID, in, version
	s.calls = append(s.calls, "SetSharing")
	return s.firstSharing(), s.err
}

func (s *stubConsents) StopSharing(_ context.Context, c consent.Caller, organizationID string) error {
	s.caller, s.arg = c, organizationID
	s.calls = append(s.calls, "StopSharing")
	return s.err
}

func (s *stubConsents) MySharing(_ context.Context, c consent.Caller) ([]consent.Sharing, error) {
	s.caller = c
	s.calls = append(s.calls, "MySharing")
	return s.sharing, s.err
}

func (s *stubConsents) firstSharing() consent.Sharing {
	if len(s.sharing) == 0 {
		return consent.Sharing{}
	}
	return s.sharing[0]
}

// serveHeader is serve with the request header h.
func serveHeader(t *testing.T, svc *stubConsents, method, path, body string, h http.Header) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	consent.NewHandler(svc, func(context.Context) (consent.Caller, bool, error) { return me, true, nil }, discard()).Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	for k, v := range h {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

var sharingAA = consent.Sharing{OrganizationID: clinicA, ConsentID: historyAA, VersionID: historyAA + ".3", Shared: []string{visit1, visit2}}

const sharingJSON = `{"organizationId":"` + clinicA + `","consentId":"` + historyAA + `","versionId":"` + historyAA + `.3","shared":["` + visit1 + `","` + visit2 + `"]}`

func TestHandler_PreviewsTheCandidates(t *testing.T) {
	c := candidate(visit1, 20)
	c.Selected = true
	svc := &stubConsents{preview: consent.Preview{Candidates: []consent.Candidate{c}, VersionID: versionAA2}}
	rec := serve(t, svc, me, true, &bytes.Buffer{}, http.MethodGet, "/history-sharing/"+clinicA, "")
	want := `{"versionId":"` + versionAA2 + `","candidates":[{"date":"2026-10-20T09:00:00Z","encounterId":"` + visit1 + `","organizationId":"` + clinicB +
		`","organizationName":"Synthetic clinic B","timeZone":"Europe/Sarajevo","specialty":"394579002","title":"Placeholder findings","selected":true}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want || svc.arg != clinicA || svc.caller != me {
		t.Errorf("GET = %d %s (clinic %q), want 200 %s", rec.Code, rec.Body, svc.arg, want)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control %q, want no-store", got)
	}
	empty := serve(t, &stubConsents{}, me, true, &bytes.Buffer{}, http.MethodGet, "/history-sharing/"+clinicA, "")
	if empty.Body.String() != `{"candidates":[]}`+"\n" {
		t.Errorf("GET with none = %s, want an empty list", empty.Body)
	}
}

func TestHandler_SetsTheSharingAtTheVersionOfIfMatch(t *testing.T) {
	for _, tt := range []struct {
		header  http.Header
		name    string
		version string
	}{
		{name: "a first grant", header: http.Header{}},
		{name: "a weak tag", header: http.Header{"If-Match": {`W/"` + versionAA2 + `"`}}, version: versionAA2},
		{name: "a strong tag", header: http.Header{"If-Match": {`"` + versionAA2 + `"`}}, version: versionAA2},
	} {
		svc := &stubConsents{sharing: []consent.Sharing{sharingAA}}
		rec := serveHeader(t, svc, http.MethodPut, "/history-sharing/"+clinicA, `{"encounterIds":["`+visit1+`","`+visit2+`"]}`, tt.header)
		if rec.Code != http.StatusOK || rec.Body.String() != sharingJSON+"\n" || rec.Header().Get("ETag") != `W/"`+historyAA+`.3"` {
			t.Errorf("%s: PUT = %d %s ETag %q, want 200 %s and the consent's version 3", tt.name, rec.Code, rec.Body, rec.Header().Get("ETag"), sharingJSON)
		}
		if svc.version != tt.version || svc.arg != clinicA || !slices.Equal(svc.set.EncounterIDs, []string{visit1, visit2}) {
			t.Errorf("%s: SetSharing(%q, %+v, %q), want clinic A, the two visits, version %q", tt.name, svc.arg, svc.set, svc.version, tt.version)
		}
	}
	svc := &stubConsents{sharing: []consent.Sharing{{OrganizationID: clinicA, Shared: []string{}}}}
	rec := serveHeader(t, svc, http.MethodPut, "/history-sharing/"+clinicA, `{"encounterIds":[]}`, http.Header{"If-Match": {`W/"` + historyAA + `.3"`}})
	if want := `{"organizationId":"` + clinicA + `","shared":[]}` + "\n"; rec.Code != http.StatusOK || rec.Body.String() != want || rec.Header().Get("ETag") != "" {
		t.Errorf("PUT [] = %d %s ETag %q, want 200 %s without an ETag", rec.Code, rec.Body, rec.Header().Get("ETag"), want)
	}
}

func TestHandler_StopsSharingAndListsMine(t *testing.T) {
	svc := &stubConsents{}
	if rec := serve(t, svc, me, true, &bytes.Buffer{}, http.MethodDelete, "/history-sharing/"+clinicA, ""); rec.Code != http.StatusNoContent || svc.arg != clinicA {
		t.Errorf("DELETE = %d (clinic %q), want 204 for clinic A", rec.Code, svc.arg)
	}
	svc = &stubConsents{sharing: []consent.Sharing{sharingAA}}
	rec := serve(t, svc, me, true, &bytes.Buffer{}, http.MethodGet, "/me/history-sharing", "")
	if want := `{"sharing":[` + sharingJSON + `]}` + "\n"; rec.Code != http.StatusOK || rec.Body.String() != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET /me/history-sharing = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
	if rec := serve(t, &stubConsents{}, me, true, &bytes.Buffer{}, http.MethodGet, "/me/history-sharing", ""); rec.Body.String() != `{"sharing":[]}`+"\n" {
		t.Errorf("GET with none = %s, want an empty list", rec.Body)
	}
}

func TestHandler_MapsEverySharingErrorToItsCode(t *testing.T) {
	put := `{"encounterIds":["` + visit1 + `"]}`
	for _, tt := range []struct {
		err    error
		method string
		path   string
		body   string
		code   string
		status int
	}{
		{err: consent.ErrForbidden, method: http.MethodGet, path: "/me/history-sharing", status: 403, code: "forbidden"},
		{err: consent.ErrForbidden, method: http.MethodGet, path: "/history-sharing/" + clinicA, status: 403, code: "forbidden"},
		{err: consent.ErrClinicNotFound, method: http.MethodGet, path: "/history-sharing/x", status: 404, code: "not_found"},
		{err: consent.ErrCareConsentRequired, method: http.MethodPut, path: "/history-sharing/" + clinicA, body: put, status: 409, code: "care_consent_required"},
		{err: consent.ErrInvalidVisit, method: http.MethodPut, path: "/history-sharing/" + clinicA, body: put, status: 400, code: "invalid_visit"},
		{err: consent.ErrVersionConflict, method: http.MethodPut, path: "/history-sharing/" + clinicA, body: put, status: 412, code: "version_conflict"},
		{err: consent.ErrClinicNotFound, method: http.MethodPut, path: "/history-sharing/x", body: put, status: 404, code: "not_found"},
		{err: consent.ErrForbidden, method: http.MethodDelete, path: "/history-sharing/" + clinicA, status: 403, code: "forbidden"},
	} {
		rec := serve(t, &stubConsents{err: tt.err}, me, true, &bytes.Buffer{}, tt.method, tt.path, tt.body)
		if want := `{"error":"` + tt.code + `"}` + "\n"; rec.Code != tt.status || rec.Body.String() != want {
			t.Errorf("%s %s with %v = %d %s, want %d %s", tt.method, tt.path, tt.err, rec.Code, rec.Body, tt.status, want)
		}
	}
}

func TestHandler_RefusesBadSharingRequestsAndNobodyBeforeTheService(t *testing.T) {
	many := `{"encounterIds":[` + strings.TrimSuffix(strings.Repeat(`"`+visit1+`",`, 501), ",") + `]}`
	for _, tt := range []struct {
		header   http.Header
		name     string
		method   string
		path     string
		body     string
		code     string
		status   int
		signedIn bool
	}{
		{name: "nobody previews", method: http.MethodGet, path: "/history-sharing/" + clinicA, status: 401, code: "unauthenticated"},
		{name: "nobody sets", method: http.MethodPut, path: "/history-sharing/" + clinicA, body: `{"encounterIds":[]}`, status: 401, code: "unauthenticated"},
		{name: "nobody stops", method: http.MethodDelete, path: "/history-sharing/" + clinicA, status: 401, code: "unauthenticated"},
		{name: "nobody lists", method: http.MethodGet, path: "/me/history-sharing", status: 401, code: "unauthenticated"},
		{name: "no list", method: http.MethodPut, path: "/history-sharing/" + clinicA, body: `{}`, status: 400, code: "invalid_request", signedIn: true},
		{name: "an unknown field", method: http.MethodPut, path: "/history-sharing/" + clinicA, body: `{"encounterIds":[],"all":true}`, status: 400, code: "invalid_request", signedIn: true},
		{name: "too many", method: http.MethodPut, path: "/history-sharing/" + clinicA, body: many, status: 400, code: "invalid_request", signedIn: true},
		{name: "a bad If-Match", method: http.MethodPut, path: "/history-sharing/" + clinicA, body: `{"encounterIds":[]}`, header: http.Header{"If-Match": {`W/"x"`}}, status: 400, code: "invalid_request", signedIn: true},
	} {
		svc := &stubConsents{}
		mux := http.NewServeMux()
		consent.NewHandler(svc, func(context.Context) (consent.Caller, bool, error) { return me, tt.signedIn, nil }, discard()).Routes(mux)
		req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, strings.NewReader(tt.body))
		for k, v := range tt.header {
			req.Header[k] = v
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if want := `{"error":"` + tt.code + `"}` + "\n"; rec.Code != tt.status || rec.Body.String() != want || len(svc.calls) != 0 {
			t.Errorf("%s: %d %s after calls %v, want %d %s before the service", tt.name, rec.Code, rec.Body, svc.calls, tt.status, want)
		}
	}
}

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
