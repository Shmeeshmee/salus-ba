package organization_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// assertGolden compares got with testdata/<name>; -update rewrites the file.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("%s differs from the response:\n got: %s\nwant: %s", path, got, want)
	}
}

// stubClinics answers with what it is given and keeps what it was asked.
type stubClinics struct {
	err     error
	caller  organization.Caller
	id      string
	changes organization.Changes
	org     organization.Organization
	all     []organization.Organization // what ListListed gives
	calls   int
	listed  int // calls of ListListed
	version int32
}

func (s *stubClinics) Read(_ context.Context, c organization.Caller, id string) (organization.Organization, error) {
	s.caller, s.id = c, id
	s.calls++
	return s.org, s.err
}

func (s *stubClinics) ListListed(context.Context) ([]organization.Organization, error) {
	s.calls++
	s.listed++
	return s.all, s.err
}

func (s *stubClinics) Update(_ context.Context, c organization.Caller, id string, ch organization.Changes, version int32) (organization.Organization, error) {
	s.caller, s.id, s.changes, s.version = c, id, ch, version
	s.calls++
	return s.org, s.err
}

func clinic(version int32) organization.Organization {
	return organization.Organization{ID: clinicA, Name: "Synthetic clinic", TimeZone: "Europe/Sarajevo", VersionID: version, LastUpdated: now, Active: true}
}

type orgServer struct {
	svc    *stubClinics
	caller func(context.Context) (organization.Caller, bool, error)
	log    *slog.Logger
}

func newOrgServer(svc *stubClinics) *orgServer {
	return &orgServer{
		svc:    svc,
		caller: func(context.Context) (organization.Caller, bool, error) { return admin, true, nil },
		log:    slog.New(slog.DiscardHandler),
	}
}

func (s *orgServer) do(t *testing.T, method, path, ifMatch, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	organization.NewHandler(s.svc, s.caller, "https://api.example.test/fhir/R4", s.log).Routes(mux)
	r := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if ifMatch != "" {
		r.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

const clinicPath = "/fhir/R4/Organization/" + clinicA

func renameBody(id string) string {
	return `{"resourceType":"Organization","id":"` + id + `","meta":{"versionId":"3"},"active":true,"name":"Renamed synthetic clinic"}`
}

func TestHandlerRead_SendsTheClinicWithItsMetaAndETag(t *testing.T) {
	s := newOrgServer(&stubClinics{org: clinic(3)})

	rec := s.do(t, http.MethodGet, clinicPath, "", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	if etag := rec.Header().Get("ETag"); etag != `W/"3"` {
		t.Errorf(`ETag = %q, want W/"3"`, etag)
	}
	if s.svc.caller != admin || s.svc.id != clinicA {
		t.Errorf("Read(%+v, %q), want the caller and the id of the URL", s.svc.caller, s.svc.id)
	}
	assertGolden(t, "organization.json", rec.Body.Bytes())
}

func TestHandlerUpdate_RenamesAtTheVersionOfIfMatchAndSendsTheNewVersion(t *testing.T) {
	for _, ifMatch := range []string{`W/"3"`, `"3"`} {
		updated := clinic(4)
		updated.Name = "Renamed synthetic clinic"
		s := newOrgServer(&stubClinics{org: updated})

		rec := s.do(t, http.MethodPut, clinicPath, ifMatch, renameBody(clinicA))

		if rec.Code != http.StatusOK {
			t.Fatalf("If-Match %s: status = %d %s, want 200", ifMatch, rec.Code, rec.Body)
		}
		if s.svc.caller != admin || s.svc.id != clinicA || s.svc.changes != (organization.Changes{Name: "Renamed synthetic clinic"}) || s.svc.version != 3 {
			t.Errorf("Update(%+v, %q, %+v, %d), want the caller, the URL's id, the name only and version 3", s.svc.caller, s.svc.id, s.svc.changes, s.svc.version)
		}
		if etag := rec.Header().Get("ETag"); etag != `W/"4"` {
			t.Errorf(`ETag = %q, want W/"4"`, etag)
		}
		assertGolden(t, "organization_updated.json", rec.Body.Bytes())
	}
}

const (
	listedURL   = "https://salus.invalid/fhir/StructureDefinition/organization-listed"
	timeZoneURL = "https://salus.invalid/fhir/StructureDefinition/organization-timezone"
)

// An admin lists the clinic and sets its time zone through the extensions (docs/booking/SPEC.md "Listed in
// FHIR"); an extension left out keeps its value.
func TestHandlerUpdate_SetsTheListingAndTimeZoneFromTheirExtensions(t *testing.T) {
	updated := clinic(4)
	updated.Listed, updated.TimeZone = true, "Europe/Vienna"
	for name, tc := range map[string]struct {
		extensions   string
		wantZone     string
		listed, zone bool
		wantListed   bool
	}{
		"both":    {`[{"url":"` + listedURL + `","valueBoolean":true},{"url":"` + timeZoneURL + `","valueCode":"Europe/Vienna"}]`, "Europe/Vienna", true, true, true},
		"listed":  {`[{"url":"` + listedURL + `","valueBoolean":false}]`, "", true, false, false},
		"zone":    {`[{"url":"` + timeZoneURL + `","valueCode":"Europe/Vienna"}]`, "Europe/Vienna", false, true, false},
		"another": {`[{"url":"https://elsewhere.test/x","valueString":"y"}]`, "", false, false, false},
	} {
		s := newOrgServer(&stubClinics{org: updated})
		body := `{"resourceType":"Organization","id":"` + clinicA + `","name":"Synthetic clinic","extension":` + tc.extensions + `}`

		rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body)

		ch := s.svc.changes
		if rec.Code != http.StatusOK || ch.Name != "Synthetic clinic" || (ch.Listed != nil) != tc.listed || (ch.TimeZone != nil) != tc.zone {
			t.Fatalf("%s: status = %d %s with changes %+v", name, rec.Code, rec.Body, ch)
		}
		if (tc.listed && *ch.Listed != tc.wantListed) || (tc.zone && *ch.TimeZone != tc.wantZone) {
			t.Errorf("%s: listed %v, zone %v; want %v, %q", name, *ch.Listed, *ch.TimeZone, tc.wantListed, tc.wantZone)
		}
		if name == "both" {
			assertGolden(t, "organization_listed.json", rec.Body.Bytes())
		}
	}
}

func TestHandlerUpdate_RejectsAnExtensionWithoutItsValueBeforeCallingTheService(t *testing.T) {
	for _, extensions := range []string{
		`[{"url":"` + listedURL + `"}]`,
		`[{"url":"` + listedURL + `","valueCode":"true"}]`,
		`[{"url":"` + timeZoneURL + `","valueBoolean":true}]`,
		`[{"url":"` + timeZoneURL + `","valueCode":""}]`,
		`[{"url":"` + listedURL + `","valueBoolean":true},{"url":"` + listedURL + `","valueBoolean":false}]`,
	} {
		s := newOrgServer(&stubClinics{})
		body := `{"resourceType":"Organization","id":"` + clinicA + `","name":"Synthetic clinic","extension":` + extensions + `}`

		rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body)

		if rec.Code != http.StatusBadRequest || s.svc.calls != 0 {
			t.Errorf("extension %s: status = %d after %d service calls, want 400 and none", extensions, rec.Code, s.svc.calls)
		}
		assertGolden(t, "invalid_resource.json", rec.Body.Bytes())
	}
}

// Organization?listed=true is the public directory: anyone, signed in or not, gets the listed clinics.
func TestHandlerSearch_SendsTheListedClinicsToAnyone(t *testing.T) {
	listed := clinic(4)
	listed.Listed = true
	for name, caller := range map[string]func(context.Context) (organization.Caller, bool, error){
		"nobody signed in": func(context.Context) (organization.Caller, bool, error) { return organization.Caller{}, false, nil },
		"the admin":        func(context.Context) (organization.Caller, bool, error) { return admin, true, nil },
	} {
		s := newOrgServer(&stubClinics{all: []organization.Organization{listed}})
		s.caller = caller

		rec := s.do(t, http.MethodGet, "/fhir/R4/Organization?listed=true", "", "")

		if rec.Code != http.StatusOK || s.svc.listed != 1 {
			t.Fatalf("%s: status = %d %s after %d listings, want 200 from ListListed", name, rec.Code, rec.Body, s.svc.listed)
		}
		assertGolden(t, "search_listed.json", rec.Body.Bytes())
	}
}

func TestHandlerSearch_RejectsAnyOtherSearchWithoutCallingTheService(t *testing.T) {
	for _, query := range []string{"", "listed=false", "listed=", "listed=true&listed=true", "listed=true&name=x", "active=true", "listed:missing=true"} {
		s := newOrgServer(&stubClinics{})

		rec := s.do(t, http.MethodGet, "/fhir/R4/Organization?"+query, "", "")

		if rec.Code != http.StatusBadRequest || s.svc.calls != 0 {
			t.Errorf("?%s: status = %d after %d service calls, want 400 and none", query, rec.Code, s.svc.calls)
		}
		assertGolden(t, "unsupported_search.json", rec.Body.Bytes())
	}
}

func TestHandlerSearch_SendsAFixedOperationOutcomeOnAServiceError(t *testing.T) {
	var logs bytes.Buffer
	s := newOrgServer(&stubClinics{err: errors.New("db down")})
	s.log = slog.New(slog.NewJSONHandler(&logs, nil))

	rec := s.do(t, http.MethodGet, "/fhir/R4/Organization?listed=true", "", "")

	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") || logs.Len() == 0 {
		t.Errorf("status = %d %s, log %q; want 500 without the error, and the error logged", rec.Code, rec.Body, logs.String())
	}
}

func TestHandlerUpdate_RefusesWithoutAVersionBeforeCallingTheService(t *testing.T) {
	for _, ifMatch := range []string{"", "3", `W/"three"`, `W/"99999999999"`} {
		s := newOrgServer(&stubClinics{})

		rec := s.do(t, http.MethodPut, clinicPath, ifMatch, renameBody(clinicA))

		if rec.Code != http.StatusPreconditionFailed || s.svc.calls != 0 {
			t.Errorf("If-Match %q: status = %d after %d service calls, want 412 and none", ifMatch, rec.Code, s.svc.calls)
		}
		assertGolden(t, "if_match_required.json", rec.Body.Bytes())
	}
}

func TestHandlerUpdate_RejectsABodyThatIsNotThisOrganization(t *testing.T) {
	for _, body := range []string{
		"not json",
		`{"resourceType":"Practitioner","id":"` + clinicA + `","name":"Renamed synthetic clinic"}`,
		renameBody(clinicB),
		`{"resourceType":"Organization","name":"Renamed synthetic clinic"}`,
		`{"resourceType":"Organization","id":"` + clinicA + `","name":["Renamed synthetic clinic"]}`,
	} {
		s := newOrgServer(&stubClinics{})

		rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body)

		if rec.Code != http.StatusBadRequest || s.svc.calls != 0 {
			t.Errorf("body %s: status = %d after %d service calls, want 400 and none", body, rec.Code, s.svc.calls)
		}
		assertGolden(t, "invalid_resource.json", rec.Body.Bytes())
	}
}

func TestHandlerUpdate_RejectsABodyOver64KiB(t *testing.T) {
	s := newOrgServer(&stubClinics{})
	body := `{"resourceType":"Organization","id":"` + clinicA + `","name":"` + strings.Repeat("x", 64<<10) + `"}`

	rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body)

	if rec.Code != http.StatusBadRequest || s.svc.calls != 0 {
		t.Errorf("status = %d after %d service calls, want 400 and none", rec.Code, s.svc.calls)
	}
}

func TestHandler_MapsTheServiceErrors(t *testing.T) {
	tests := []struct {
		err    error
		method string
		golden string
		status int
	}{
		{err: organization.ErrNotFound, method: http.MethodGet, status: http.StatusNotFound, golden: "not_found.json"},
		{err: organization.ErrNotFound, method: http.MethodPut, status: http.StatusNotFound, golden: "not_found.json"},
		{err: organization.ErrConflict, method: http.MethodPut, status: http.StatusPreconditionFailed, golden: "version_conflict.json"},
		{err: organization.ErrForbidden, method: http.MethodPut, status: http.StatusForbidden, golden: "forbidden.json"},
		{err: organization.ErrInvalidName, method: http.MethodPut, status: http.StatusUnprocessableEntity, golden: "invalid_name.json"},
		{err: organization.ErrInvalidTimeZone, method: http.MethodPut, status: http.StatusBadRequest, golden: "invalid_time_zone.json"},
	}
	for _, tt := range tests {
		s := newOrgServer(&stubClinics{err: tt.err})

		rec := s.do(t, tt.method, clinicPath, `W/"3"`, renameBody(clinicA))

		if rec.Code != tt.status {
			t.Errorf("%s with %v: status = %d, want %d", tt.method, tt.err, rec.Code, tt.status)
		}
		assertGolden(t, tt.golden, rec.Body.Bytes())
	}
}

func TestHandler_AsksForASignInWithoutCallingTheService(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		s := newOrgServer(&stubClinics{})
		s.caller = func(context.Context) (organization.Caller, bool, error) { return organization.Caller{}, false, nil }

		rec := s.do(t, method, clinicPath, `W/"3"`, renameBody(clinicA))

		if rec.Code != http.StatusUnauthorized || s.svc.calls != 0 {
			t.Errorf("%s: status = %d after %d service calls, want 401 and none", method, rec.Code, s.svc.calls)
		}
		assertGolden(t, "unauthenticated.json", rec.Body.Bytes())
	}
}

func TestHandler_SendsAFixedOperationOutcomeAndLogsNoName(t *testing.T) {
	failing := errors.New("db down")
	callerFails := newOrgServer(&stubClinics{})
	callerFails.caller = func(context.Context) (organization.Caller, bool, error) { return organization.Caller{}, false, failing }
	for name, s := range map[string]*orgServer{"the service": newOrgServer(&stubClinics{err: failing}), "the caller": callerFails} {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			var logs bytes.Buffer
			s.log = slog.New(slog.NewJSONHandler(&logs, nil))

			rec := s.do(t, method, clinicPath, `W/"3"`, renameBody(clinicA))

			if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") {
				t.Errorf("%s failing, %s = %d %s, want 500 without the error", name, method, rec.Code, rec.Body)
			}
			if logs.Len() == 0 || strings.Contains(logs.String(), "Renamed synthetic clinic") {
				t.Errorf("%s failing, %s: log %q, want the error without the name", name, method, logs.String())
			}
		}
	}
}
