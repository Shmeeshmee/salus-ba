package location_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/location"
)

// lcStub answers every call with its values and remembers what it was asked.
type lcStub struct {
	err   error
	org   string
	staff location.Staff
	out   []location.Location
	saved location.Location
}

func (s *lcStub) Locations(_ context.Context, st location.Staff) ([]location.Location, error) {
	s.staff = st
	return s.out, s.err
}

func (s *lcStub) SaveLocation(_ context.Context, st location.Staff, l location.Location) (location.Location, error) {
	s.staff, s.saved = st, l
	if l.ID == "" {
		l.ID = "10000000-0000-4000-8000-000000000001"
	}
	l.OrganizationID, l.VersionID, l.TimeZone = st.OrganizationID, l.VersionID+1, "Europe/Sarajevo"
	return l, s.err
}

func (s *lcStub) PublicLocations(_ context.Context, org string) ([]location.Location, error) {
	s.org = org
	return s.out, s.err
}

func lcServe(t *testing.T, svc *lcStub, signedIn bool, method, path, body string, header ...string) *httptest.ResponseRecorder {
	t.Helper()
	caller := func(context.Context) (location.Staff, bool, error) { return lcAdmin, signedIn, nil }
	mux := http.NewServeMux()
	location.NewHandler(svc, caller, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes(mux)
	r := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	for i := 0; i+1 < len(header); i += 2 {
		r.Header.Set(header[i], header[i+1])
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

const lcPath = "/clinic/locations/10000000-0000-4000-8000-000000000001"

func TestHandler_Locations(t *testing.T) {
	t.Parallel()
	site := location.Location{ID: "10000000-0000-4000-8000-000000000001", OrganizationID: lcClinicA, Name: "Centre", TimeZone: "Europe/Sarajevo", VersionID: 3, Active: true}
	for name, tc := range map[string]struct {
		err      error
		method   string
		path     string
		body     string
		want     string
		etag     string
		cache    string
		header   []string
		status   int
		signedIn bool
	}{
		"the list":                 {nil, http.MethodGet, "/clinic/locations", "", `[{"id":"10000000-0000-4000-8000-000000000001","organizationId":"a0000000-0000-4000-8000-00000000000a","name":"Centre","timeZone":"Europe/Sarajevo","versionId":3,"active":true}]`, "", "no-store", nil, http.StatusOK, true},
		"the list, signed out":     {nil, http.MethodGet, "/clinic/locations", "", `{"error":"unauthenticated"}`, "", "", nil, http.StatusUnauthorized, false},
		"the list of no clinic":    {location.ErrNotFound, http.MethodGet, "/clinic/locations", "", `{"error":"not_found"}`, "", "", nil, http.StatusNotFound, true},
		"added":                    {nil, http.MethodPost, "/clinic/locations", `{"name":"Annex","addressLine":"Synthetic street 4","phone":"+38733000000","active":true}`, `{"id":"10000000-0000-4000-8000-000000000001","organizationId":"a0000000-0000-4000-8000-00000000000a","name":"Annex","addressLine":"Synthetic street 4","phone":"+38733000000","timeZone":"Europe/Sarajevo","versionId":1,"active":true}`, `W/"1"`, "", nil, http.StatusCreated, true},
		"added without active":     {nil, http.MethodPost, "/clinic/locations", `{"name":"Annex"}`, `{"error":"invalid_request"}`, "", "", nil, http.StatusBadRequest, true},
		"an unknown field":         {nil, http.MethodPost, "/clinic/locations", `{"name":"Annex","active":true,"x":1}`, `{"error":"invalid_request"}`, "", "", nil, http.StatusBadRequest, true},
		"invalid":                  {location.ErrInvalid, http.MethodPost, "/clinic/locations", `{"name":"Annex","active":true}`, `{"error":"invalid_location"}`, "", "", nil, http.StatusBadRequest, true},
		"not the admin":            {location.ErrForbidden, http.MethodPost, "/clinic/locations", `{"name":"Annex","active":true}`, `{"error":"forbidden"}`, "", "", nil, http.StatusForbidden, true},
		"changed":                  {nil, http.MethodPut, lcPath, `{"name":"Centre","active":false}`, `{"id":"10000000-0000-4000-8000-000000000001","organizationId":"a0000000-0000-4000-8000-00000000000a","name":"Centre","timeZone":"Europe/Sarajevo","versionId":4,"active":false}`, `W/"4"`, "", []string{"If-Match", `W/"3"`}, http.StatusOK, true},
		"changed without If-Match": {nil, http.MethodPut, lcPath, `{"name":"Centre","active":false}`, `{"error":"invalid_request"}`, "", "", nil, http.StatusBadRequest, true},
		"another version":          {location.ErrVersionConflict, http.MethodPut, lcPath, `{"name":"Centre","active":false}`, `{"error":"version_conflict"}`, "", "", []string{"If-Match", `W/"2"`}, http.StatusPreconditionFailed, true},
		"public":                   {nil, http.MethodGet, "/organizations/" + lcClinicA + "/locations", "", `[{"id":"10000000-0000-4000-8000-000000000001","organizationId":"a0000000-0000-4000-8000-00000000000a","name":"Centre","timeZone":"Europe/Sarajevo","versionId":3,"active":true}]`, "", "", nil, http.StatusOK, false},
		"public, not listed":       {location.ErrNotFound, http.MethodGet, "/organizations/" + lcClinicB + "/locations", "", `{"error":"not_found"}`, "", "", nil, http.StatusNotFound, false},
	} {
		svc := &lcStub{err: tc.err, out: []location.Location{site}}
		w := lcServe(t, svc, tc.signedIn, tc.method, tc.path, tc.body, tc.header...)
		if got := strings.TrimSpace(w.Body.String()); w.Code != tc.status || got != tc.want {
			t.Errorf("%s: %d %s, want %d %s", name, w.Code, got, tc.status, tc.want)
		}
		if w.Header().Get("ETag") != tc.etag || w.Header().Get("Cache-Control") != tc.cache {
			t.Errorf("%s: ETag %q, Cache-Control %q; want %q, %q", name, w.Header().Get("ETag"), w.Header().Get("Cache-Control"), tc.etag, tc.cache)
		}
	}
}

func TestHandler_PassesTheLocationAsSent(t *testing.T) {
	t.Parallel()
	svc := &lcStub{}
	lcServe(t, svc, true, http.MethodPut, lcPath, `{"name":"Centre","postalCode":"71000","cityId":"Q1","timeZone":"Europe/Zagreb","active":true}`, "If-Match", `W/"7"`)
	want := location.Location{ID: "10000000-0000-4000-8000-000000000001", Name: "Centre", PostalCode: "71000", CityID: "Q1", TimeZone: "Europe/Zagreb", VersionID: 7, Active: true}
	if svc.saved != want || svc.staff != lcAdmin {
		t.Errorf("saved %+v by %+v, want %+v by the admin", svc.saved, svc.staff, want)
	}
}
