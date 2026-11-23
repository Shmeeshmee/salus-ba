package organization_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
)

// stubProfiles answers with what it is given and keeps what it was asked.
type stubProfiles struct {
	err     error
	caller  organization.Caller
	id      string
	in      organization.Profile
	profile organization.Profile
	version int32
}

func (s *stubProfiles) Profile(_ context.Context, id string) (organization.Profile, error) {
	s.id = id
	return s.profile, s.err
}

func (s *stubProfiles) ClinicProfile(_ context.Context, c organization.Caller) (organization.Profile, error) {
	s.caller = c
	return s.profile, s.err
}

func (s *stubProfiles) SaveProfile(_ context.Context, c organization.Caller, p organization.Profile, version int32) (organization.Profile, error) {
	s.caller, s.in, s.version = c, p, version
	p.VersionID = version + 1
	return p, s.err
}

func serveProfile(t *testing.T, svc *stubProfiles, signedIn bool, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	h := organization.NewProfileHandler(svc, func(context.Context) (organization.Caller, bool, error) { return adminCaller, signedIn, nil },
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const profileJSON = `{"languages":["bs","de"],"description":"Synthetic","translations":[{"language":"de","description":"Synthetisch"}],"welcomesInternational":true}` + "\n"

var stubbedProfile = organization.Profile{Languages: []string{"bs", "de"}, Description: "Synthetic", VersionID: 4, WelcomesInternational: true,
	Translations: []organization.Translation{{Language: "de", Description: "Synthetisch"}}}

func TestProfileHandler_ServesTheClinicsProfileWithItsVersion(t *testing.T) {
	svc := &stubProfiles{profile: stubbedProfile}
	rec := serveProfile(t, svc, true, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/clinic/profile", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != profileJSON || rec.Header().Get("ETag") != `W/"4"` || svc.caller != adminCaller {
		t.Errorf("GET /clinic/profile = %d %q ETag %q as %+v; want 200 %q W/\"4\"", rec.Code, rec.Body, rec.Header().Get("ETag"), svc.caller, profileJSON)
	}
	empty := &stubProfiles{}
	rec = serveProfile(t, empty, true, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/clinic/profile", nil))
	if want := `{"languages":[],"description":"","translations":[],"welcomesInternational":false}` + "\n"; rec.Body.String() != want {
		t.Errorf("an empty profile = %q, want %q", rec.Body, want)
	}
	rec = serveProfile(t, svc, false, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/clinic/profile", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /clinic/profile signed out = %d, want 401", rec.Code)
	}
}

func TestProfileHandler_SavesTheProfileAtTheIfMatchVersion(t *testing.T) {
	svc := &stubProfiles{}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/clinic/profile", strings.NewReader(profileJSON))
	req.Header.Set("If-Match", `W/"3"`)
	rec := serveProfile(t, svc, true, req)
	in := stubbedProfile
	in.VersionID = 0
	if rec.Code != http.StatusOK || rec.Body.String() != profileJSON || rec.Header().Get("ETag") != `W/"4"` || !reflect.DeepEqual(svc.in, in) || svc.version != 3 {
		t.Errorf("PUT = %d %q ETag %q, saved %+v at %d; want 200, %+v at 3", rec.Code, rec.Body, rec.Header().Get("ETag"), svc.in, svc.version, in)
	}
}

func TestProfileHandler_AnswersTheCodeOfEachRefusal(t *testing.T) {
	for _, tt := range []struct {
		err     error
		name    string
		body    string
		ifMatch string
		want    string
		code    int
	}{
		{name: "no If-Match", body: profileJSON, code: http.StatusBadRequest, want: "if_match_required"},
		{name: "a bad body", body: `{"languages":[]}`, ifMatch: `W/"3"`, code: http.StatusBadRequest, want: "invalid_request"},
		{name: "an unknown field", body: `{"languages":[],"description":"","translations":[],"welcomesInternational":true,"x":1}`, ifMatch: `W/"3"`, code: http.StatusBadRequest, want: "invalid_request"},
		{name: "invalid", err: organization.ErrInvalidProfile, body: profileJSON, ifMatch: `W/"3"`, code: http.StatusBadRequest, want: "invalid_profile"},
		{name: "not admin", err: organization.ErrForbidden, body: profileJSON, ifMatch: `W/"3"`, code: http.StatusForbidden, want: "forbidden"},
		{name: "a patient", err: organization.ErrNotFound, body: profileJSON, ifMatch: `W/"3"`, code: http.StatusNotFound, want: "not_found"},
		{name: "stale", err: organization.ErrConflict, body: profileJSON, ifMatch: `W/"3"`, code: http.StatusConflict, want: "version_conflict"},
		{name: "broken", err: errors.New("connection reset"), body: profileJSON, ifMatch: `W/"3"`, code: http.StatusInternalServerError},
	} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/clinic/profile", strings.NewReader(tt.body))
		if tt.ifMatch != "" {
			req.Header.Set("If-Match", tt.ifMatch)
		}
		rec := serveProfile(t, &stubProfiles{err: tt.err}, true, req)
		if rec.Code != tt.code || (tt.want != "" && rec.Body.String() != `{"error":"`+tt.want+`"}`+"\n") {
			t.Errorf("%s: PUT = %d %q, want %d %q", tt.name, rec.Code, rec.Body, tt.code, tt.want)
		}
	}
}

func TestProfileHandler_ServesAListedClinicsProfileToAnyone(t *testing.T) {
	svc := &stubProfiles{profile: stubbedProfile}
	rec := serveProfile(t, svc, false, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/organizations/"+clinicB+"/profile", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != profileJSON || svc.id != clinicB || rec.Header().Get("ETag") != "" {
		t.Errorf("GET profile = %d %q for %q (ETag %q), want 200 %q for clinic B without an ETag", rec.Code, rec.Body, svc.id, rec.Header().Get("ETag"), profileJSON)
	}
	svc.err = organization.ErrNotFound
	rec = serveProfile(t, svc, false, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/organizations/"+clinicB+"/profile", nil))
	if rec.Code != http.StatusNotFound || rec.Body.String() != `{"error":"not_found"}`+"\n" {
		t.Errorf("GET profile of a hidden clinic = %d %q, want 404 not_found", rec.Code, rec.Body)
	}
}
