package terminology_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// stubSearcher records its call and returns its matches or its error.
type stubSearcher struct {
	err    error
	caller *terminology.Caller
	search *terminology.Search
	// language is what Explanations was asked for; explanations answer it
	language     string
	matches      []terminology.Match
	explanations []terminology.Explanation
}

func (s *stubSearcher) Explanations(_ context.Context, language string) ([]terminology.Explanation, error) {
	s.language = language
	return s.explanations, s.err
}

func (s *stubSearcher) Search(_ context.Context, c terminology.Caller, q terminology.Search) ([]terminology.Match, error) {
	s.caller, s.search = &c, &q
	return s.matches, s.err
}

// secret is a search text a doctor could type; it must never reach a log line or an error.
const secret = "Marko Marković"

func signedIn(c terminology.Caller) func(context.Context) (terminology.Caller, bool, error) {
	return func(context.Context) (terminology.Caller, bool, error) { return c, true, nil }
}

// serve sends GET /terminology/search?<query> to the handler on svc and caller, and returns the response and
// everything logged.
func serve(t *testing.T, svc terminology.Searcher, caller func(context.Context) (terminology.Caller, bool, error), query string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	var logs bytes.Buffer
	mux := http.NewServeMux()
	terminology.NewHTTPHandler(svc, caller, slog.New(slog.NewJSONHandler(&logs, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/terminology/search?"+query, nil))
	return rec, logs.String()
}

func TestHTTPSearch_ReturnsTheMatches(t *testing.T) {
	t.Parallel()
	svc := &stubSearcher{matches: []terminology.Match{
		{Term: terminology.Term{ID: "00000000-0000-4000-8000-000000000001", Category: "anatomy", WikidataID: "Q9368", TA98: "A05.8.01.001",
			ReviewStatus: "approved", ReviewedAt: time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC), Rank: 100},
			Labels: []terminology.Names{
				{Language: "la", Preferred: "hepar", Synonyms: []string{"iecur"}},
				{Language: "hr", Preferred: "jetra", Synonyms: []string{}},
			}},
		{Term: terminology.Term{ID: "00000000-0000-4000-8000-000000000002", Category: "diagnosis", ICD10: "K75.9", ATC: "J05", PCS: "0FT00ZZ", ReviewStatus: "unreviewed"},
			Labels: []terminology.Names{}},
	}}

	rec, _ := serve(t, svc, signedIn(staff), "q=hepar&languages=la,hr&category=anatomy&limit=5")

	want := `{"terms":[` +
		`{"id":"00000000-0000-4000-8000-000000000001","category":"anatomy","reviewStatus":"approved",` +
		`"codes":{"wikidata":"Q9368","ta98":"A05.8.01.001"},` +
		`"labels":[{"language":"la","preferred":"hepar","synonyms":["iecur"]},{"language":"hr","preferred":"jetra","synonyms":[]}]},` +
		`{"id":"00000000-0000-4000-8000-000000000002","category":"diagnosis","reviewStatus":"unreviewed",` +
		`"codes":{"icd10":"K75.9","atc":"J05","pcs":"0FT00ZZ"},"labels":[]}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("GET = %d %s\nwant 200 %s", rec.Code, rec.Body, want)
	}
	if rec.Header().Get("Content-Type") != "application/json" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("headers %v; want JSON, no-store", rec.Header())
	}
	wantSearch := terminology.Search{Text: "hepar", Category: "anatomy", Languages: []string{"la", "hr"}, Limit: 5}
	if svc.search == nil || !reflect.DeepEqual(*svc.search, wantSearch) || *svc.caller != staff {
		t.Errorf("searched %+v as %+v; want %+v as %+v", svc.search, svc.caller, wantSearch, staff)
	}
}

func TestHTTPSearch_NoMatchIsAnEmptyList(t *testing.T) {
	t.Parallel()
	rec, _ := serve(t, &stubSearcher{matches: []terminology.Match{}}, signedIn(staff), "q=zzz")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"terms":[]}`+"\n" {
		t.Errorf("GET = %d %s; want 200 and no terms", rec.Code, rec.Body)
	}
}

func TestHTTPSearch_DefaultsAndQueryParsing(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, query string
		want        terminology.Search
	}{
		{"only q: limit 10, no languages, any category", "q=jetra", terminology.Search{Text: "jetra", Limit: 10}},
		{"an encoded text", "q=" + url.QueryEscape("čir na"), terminology.Search{Text: "čir na", Limit: 10}},
		{"an empty language is passed on for the service to refuse", "q=jetra&languages=", terminology.Search{Text: "jetra", Languages: []string{""}, Limit: 10}},
	} {
		svc := &stubSearcher{}
		serve(t, svc, signedIn(staff), c.query)
		if svc.search == nil || !reflect.DeepEqual(*svc.search, c.want) {
			t.Errorf("%s: searched %+v; want %+v", c.name, svc.search, c.want)
		}
	}
}

func TestHTTPSearch_Errors(t *testing.T) {
	t.Parallel()
	q := "q=" + url.QueryEscape(secret)
	for _, c := range []struct {
		svc    *stubSearcher
		caller func(context.Context) (terminology.Caller, bool, error)
		name   string
		query  string
		body   string
		status int
	}{
		{&stubSearcher{}, func(context.Context) (terminology.Caller, bool, error) { return terminology.Caller{}, false, nil }, "nobody signed in", q,
			`{"error":"unauthenticated"}`, http.StatusUnauthorized},
		{&stubSearcher{}, func(context.Context) (terminology.Caller, bool, error) {
			return terminology.Caller{}, false, errors.New("account read failed")
		}, "the caller cannot be read", q, `{"error":"internal error"}`, http.StatusInternalServerError},
		{&stubSearcher{err: terminology.ErrForbidden}, signedIn(staff), "a patient", q, `{"error":"forbidden"}`, http.StatusForbidden},
		{&stubSearcher{err: terminology.ErrInvalidSearch}, signedIn(staff), "an invalid search", q, `{"error":"invalid_request"}`, http.StatusBadRequest},
		{&stubSearcher{}, signedIn(staff), "no q", "languages=la", `{"error":"invalid_request"}`, http.StatusBadRequest},
		{&stubSearcher{}, signedIn(staff), "a limit that is not a number", q + "&limit=ten", `{"error":"invalid_request"}`, http.StatusBadRequest},
		{&stubSearcher{}, signedIn(staff), "a limit beyond int32", q + "&limit=4294967296", `{"error":"invalid_request"}`, http.StatusBadRequest},
		{&stubSearcher{err: errors.New("terminology: search: connection reset")}, signedIn(staff), "the search fails", q,
			`{"error":"internal error"}`, http.StatusInternalServerError},
	} {
		rec, logs := serve(t, c.svc, c.caller, c.query)
		if rec.Code != c.status || rec.Body.String() != c.body+"\n" {
			t.Errorf("%s: GET = %d %s; want %d %s", c.name, rec.Code, rec.Body, c.status, c.body)
		}
		if strings.Contains(logs, "Marko") || strings.Contains(logs, "Markovi") || strings.Contains(rec.Body.String(), "Marko") {
			t.Errorf("%s: the search text reached the log or the response:\n%s", c.name, logs)
		}
	}
}
