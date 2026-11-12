package terminology_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// stubCatalog is the CatalogService: fixed answers, the caller and language it was asked with.
type stubCatalog struct {
	caller   terminology.Caller
	language string
	err      error
}

func (s *stubCatalog) GroupNames(_ context.Context, c terminology.Caller, language string) ([]terminology.GroupEntry, error) {
	s.caller, s.language = c, language
	return []terminology.GroupEntry{
		{Code: "respiratory-infections", Name: "Respiratory infections", ReviewStatus: "approved"},
		{Code: "common-cold", Parent: "respiratory-infections", Name: "Common cold", ReviewStatus: "unreviewed"},
	}, s.err
}

func (s *stubCatalog) Places(_ context.Context, c terminology.Caller) ([]terminology.Place, error) {
	s.caller = c
	return []terminology.Place{
		{ID: "BA-SRP", Kind: "region", NameLatn: "Republika Srpska", NameCyrl: "Република Српска"},
		{ID: "Q15940918", Kind: "city", Parent: "BA-SRP", NameLatn: "Banja Luka", NameCyrl: "Бања Лука"},
	}, s.err
}

func serveCatalog(t *testing.T, svc terminology.CatalogService, signed bool, target string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	caller := signedIn(catalogStaff)
	if !signed {
		caller = nobody
	}
	terminology.NewCatalogHandler(svc, caller, slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil))
	return rec
}

func TestHTTPDiagnosisGroups_ReturnsTheGroupsInTheLanguage(t *testing.T) {
	t.Parallel()
	svc := &stubCatalog{}
	rec := serveCatalog(t, svc, true, "/terminology/diagnosis-groups?language=en")
	want := `{"groups":[{"code":"respiratory-infections","name":"Respiratory infections","reviewStatus":"approved"},` +
		`{"code":"common-cold","parent":"respiratory-infections","name":"Common cold","reviewStatus":"unreviewed"}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want || svc.language != "en" || svc.caller != catalogStaff {
		t.Errorf("GET diagnosis-groups = %d %s (asked %q as %+v), want 200 %s", rec.Code, rec.Body, svc.language, svc.caller, want)
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, max-age=300" {
		t.Errorf("Cache-Control = %q", got)
	}
}

func TestHTTPPlaces_ReturnsThePlaces(t *testing.T) {
	t.Parallel()
	rec := serveCatalog(t, &stubCatalog{}, true, "/places")
	want := `{"places":[{"id":"BA-SRP","kind":"region","nameLatn":"Republika Srpska","nameCyrl":"Република Српска"},` +
		`{"id":"Q15940918","kind":"city","parent":"BA-SRP","nameLatn":"Banja Luka","nameCyrl":"Бања Лука"}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET places = %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
}

func TestHTTPCatalog_MapsTheErrors(t *testing.T) {
	t.Parallel()
	for _, target := range []string{"/terminology/diagnosis-groups?language=en", "/places"} {
		for name, tt := range map[string]struct {
			err    error
			signed bool
			want   int
		}{
			"nobody":    {nil, false, http.StatusUnauthorized},
			"a patient": {terminology.ErrForbidden, true, http.StatusForbidden},
			"a failure": {errors.New("down"), true, http.StatusInternalServerError},
		} {
			if rec := serveCatalog(t, &stubCatalog{err: tt.err}, tt.signed, target); rec.Code != tt.want {
				t.Errorf("%s %s = %d, want %d", target, name, rec.Code, tt.want)
			}
		}
	}
	if rec := serveCatalog(t, &stubCatalog{err: terminology.ErrInvalidSearch}, true, "/terminology/diagnosis-groups?language=la"); rec.Code != http.StatusBadRequest {
		t.Errorf("Latin = %d, want 400", rec.Code)
	}
}
