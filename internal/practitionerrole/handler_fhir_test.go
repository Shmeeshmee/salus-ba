package practitionerrole_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

const (
	practitionerA1 = "a3000000-0000-4000-8000-0000000000a1"
	practitionerA2 = "a3000000-0000-4000-8000-0000000000a2"
)

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

// stubRoles answers with what it is given and keeps what it was asked.
type stubRoles struct {
	err    error
	role   practitionerrole.PractitionerRole
	caller practitionerrole.Caller
	id     string // the role read, or the clinic searched
	roles  []practitionerrole.PractitionerRole
	calls  int
}

func (s *stubRoles) Read(_ context.Context, c practitionerrole.Caller, id string) (practitionerrole.PractitionerRole, error) {
	s.caller, s.id = c, id
	s.calls++
	return s.role, s.err
}

func (s *stubRoles) Search(_ context.Context, c practitionerrole.Caller, org string) ([]practitionerrole.PractitionerRole, error) {
	s.caller, s.id = c, org
	s.calls++
	return s.roles, s.err
}

// stubPractitioners reads the practitioners it holds, and ErrNotFound for any other.
type stubPractitioners struct {
	err     error
	byID    map[string]practitioner.Practitioner
	callers []practitioner.Caller
	read    []string
}

func (s *stubPractitioners) Read(_ context.Context, c practitioner.Caller, id string) (practitioner.Practitioner, error) {
	s.callers, s.read = append(s.callers, c), append(s.read, id)
	if s.err != nil {
		return practitioner.Practitioner{}, s.err
	}
	p, ok := s.byID[id]
	if !ok {
		return practitioner.Practitioner{}, practitioner.ErrNotFound
	}
	return p, nil
}

func clinicAPractitioners() *stubPractitioners {
	return &stubPractitioners{byID: map[string]practitioner.Practitioner{
		practitionerA1: {ID: practitionerA1, OrganizationID: clinicA, GivenName: "Cy", FamilyName: "Example", VersionID: 1, LastUpdated: now, Active: true},
		practitionerA2: {ID: practitionerA2, OrganizationID: clinicA, GivenName: "Dee", FamilyName: "Sample", VersionID: 3, LastUpdated: now, Active: true},
	}}
}

func signedIn(c practitionerrole.Caller) func(context.Context) (practitionerrole.Caller, bool) {
	return func(context.Context) (practitionerrole.Caller, bool) { return c, true }
}

func nobody(context.Context) (practitionerrole.Caller, bool) { return practitionerrole.Caller{}, false }

type server struct {
	roles  *stubRoles
	people *stubPractitioners
	caller func(context.Context) (practitionerrole.Caller, bool)
	log    *slog.Logger
}

func newServer(roles *stubRoles) *server {
	return &server{roles: roles, people: clinicAPractitioners(), caller: signedIn(staffA), log: slog.New(slog.DiscardHandler)}
}

func (s *server) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	practitionerrole.NewHandler(s.roles, s.people, s.caller, "https://api.example.test/fhir/R4", s.log).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

func TestHandlerRead_SendsTheRoleWithItsReferencesMetaAndETag(t *testing.T) {
	s := newServer(&stubRoles{role: role(roleA1, clinicA, practitionerA1)})

	rec := s.get(t, "/fhir/R4/PractitionerRole/"+roleA1)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	if etag := rec.Header().Get("ETag"); etag != `W/"1"` {
		t.Errorf(`ETag = %q, want W/"1"`, etag)
	}
	if s.roles.caller != staffA || s.roles.id != roleA1 {
		t.Errorf("Read(%+v, %q), want the caller and the id of the URL", s.roles.caller, s.roles.id)
	}
	assertGolden(t, "practitioner_role.json", rec.Body.Bytes())
}

func TestHandlerRead_AnswersNotFoundForARoleTheCallerMayNotSee(t *testing.T) {
	rec := newServer(&stubRoles{err: practitionerrole.ErrNotFound}).get(t, "/fhir/R4/PractitionerRole/"+roleB)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	assertGolden(t, "not_found.json", rec.Body.Bytes())
}

func TestHandlerSearch_SendsTheClinicsRolesAsASearchset(t *testing.T) {
	for _, value := range []string{clinicA, "Organization/" + clinicA} {
		s := newServer(&stubRoles{roles: []practitionerrole.PractitionerRole{role(roleA1, clinicA, practitionerA1), role(roleA2, clinicA, practitionerA2)}})

		rec := s.get(t, "/fhir/R4/PractitionerRole?organization="+url.QueryEscape(value))

		if rec.Code != http.StatusOK {
			t.Fatalf("organization=%s: status = %d %s, want 200", value, rec.Code, rec.Body)
		}
		if s.roles.caller != staffA || s.roles.id != clinicA {
			t.Errorf("organization=%s: Search(%+v, %q), want the caller and the clinic", value, s.roles.caller, s.roles.id)
		}
		if len(s.people.read) != 0 {
			t.Errorf("organization=%s: read practitioners %v without _include", value, s.people.read)
		}
		assertGolden(t, "search.json", rec.Body.Bytes())
	}
}

func TestHandlerSearch_IncludesEachRolesPractitionerReadForTheCaller(t *testing.T) {
	s := newServer(&stubRoles{roles: []practitionerrole.PractitionerRole{role(roleA1, clinicA, practitionerA1), role(roleA2, clinicA, practitionerA2)}})

	rec := s.get(t, "/fhir/R4/PractitionerRole?organization="+clinicA+"&_include=PractitionerRole:practitioner")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	if !slices.Equal(s.people.read, []string{practitionerA1, practitionerA2}) {
		t.Errorf("read practitioners %v, want each role's", s.people.read)
	}
	for _, c := range s.people.callers {
		if c != (practitioner.Caller{AccountID: staffA.AccountID, OrganizationID: clinicA}) {
			t.Errorf("practitioner read as %+v, want the search's caller", c)
		}
	}
	assertGolden(t, "search_include.json", rec.Body.Bytes())
}

func TestHandlerSearch_LeavesOutAPractitionerTheCallerMayNotSee(t *testing.T) {
	s := newServer(&stubRoles{roles: []practitionerrole.PractitionerRole{role(roleA1, clinicA, practitionerA1), role(roleA2, clinicA, "a3000000-0000-4000-8000-0000000000ff")}})

	rec := s.get(t, "/fhir/R4/PractitionerRole?organization="+clinicA+"&_include=PractitionerRole:practitioner")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	if n := strings.Count(rec.Body.String(), `"mode":"include"`); n != 1 {
		t.Errorf("%d included practitioners, want 1: %s", n, rec.Body)
	}
}

func TestHandlerSearch_SendsAnEmptySearchsetForAClinicWithoutRoles(t *testing.T) {
	rec := newServer(&stubRoles{}).get(t, "/fhir/R4/PractitionerRole?organization="+clinicA)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	assertGolden(t, "search_empty.json", rec.Body.Bytes())
}

func TestHandlerSearch_RejectsAnUnsupportedSearchWithoutCallingTheService(t *testing.T) {
	for _, query := range []string{
		"",
		"organization=",
		"organization=" + clinicA + "&organization=" + clinicB,
		"organization=" + clinicA + "," + clinicB,
		"organization:missing=true",
		"organization=" + clinicA + "&name=Cy",
		"organization=" + clinicA + "&_include=PractitionerRole:organization",
		"organization=" + clinicA + "&_include=PractitionerRole:practitioner&_include=PractitionerRole:practitioner",
		"organization=" + clinicA + "&_count=10",
	} {
		s := newServer(&stubRoles{})

		rec := s.get(t, "/fhir/R4/PractitionerRole?"+query)

		if rec.Code != http.StatusBadRequest || s.roles.calls != 0 {
			t.Errorf("?%s: status = %d after %d service calls, want 400 and none", query, rec.Code, s.roles.calls)
		}
		assertGolden(t, "unsupported_search.json", rec.Body.Bytes())
	}
}

func TestHandlerSearch_AnswersNotFoundForAnotherClinic(t *testing.T) {
	rec := newServer(&stubRoles{err: practitionerrole.ErrNotFound}).get(t, "/fhir/R4/PractitionerRole?organization="+clinicB)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	assertGolden(t, "not_found.json", rec.Body.Bytes())
}

func TestHandler_AsksForASignInWithoutCallingTheService(t *testing.T) {
	for _, path := range []string{"/fhir/R4/PractitionerRole/" + roleA1, "/fhir/R4/PractitionerRole?organization=" + clinicA} {
		s := newServer(&stubRoles{})
		s.caller = nobody

		rec := s.get(t, path)

		if rec.Code != http.StatusUnauthorized || s.roles.calls != 0 {
			t.Errorf("GET %s: status = %d after %d service calls, want 401 and none", path, rec.Code, s.roles.calls)
		}
		assertGolden(t, "unauthenticated.json", rec.Body.Bytes())
	}
}

func TestHandler_SendsAFixedOperationOutcomeAndLogsNoName(t *testing.T) {
	failing := errors.New("db down")
	readFails := newServer(&stubRoles{err: failing})
	searchFails := newServer(&stubRoles{err: failing})
	includeFails := newServer(&stubRoles{roles: []practitionerrole.PractitionerRole{role(roleA1, clinicA, practitionerA1)}})
	includeFails.people.err = failing
	for path, s := range map[string]*server{
		"/fhir/R4/PractitionerRole/" + roleA1:                                                           readFails,
		"/fhir/R4/PractitionerRole?organization=" + clinicA:                                             searchFails,
		"/fhir/R4/PractitionerRole?organization=" + clinicA + "&_include=PractitionerRole:practitioner": includeFails,
	} {
		var logs bytes.Buffer
		s.log = slog.New(slog.NewJSONHandler(&logs, nil))

		rec := s.get(t, path)

		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") {
			t.Errorf("GET %s = %d %s, want 500 without the error", path, rec.Code, rec.Body)
		}
		if logs.Len() == 0 {
			t.Errorf("GET %s: nothing was logged", path)
		}
		for _, name := range []string{"Cy", "Example", "Dee", "Sample"} {
			if strings.Contains(logs.String(), name) {
				t.Errorf("GET %s: the log holds %q: %s", path, name, logs.String())
			}
		}
	}
}
