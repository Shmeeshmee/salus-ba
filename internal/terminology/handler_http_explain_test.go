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

func serveExplanations(t *testing.T, svc terminology.Searcher, signed bool, query string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	caller := signedIn(terminology.Caller{AccountID: "a-patient"}) // a patient: not staff
	if !signed {
		caller = nobody
	}
	terminology.NewHTTPHandler(svc, caller, slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/terminology/explanations?"+query, nil))
	return rec
}

// Any signed-in caller, a patient too, reads the explanations of a language: the term, the language the text is in
// (English when it is the fallback), the text and its review status, for the "Not reviewed yet" mark.
func TestHTTPExplanations_ReturnsTheExplanationsOfTheLanguage(t *testing.T) {
	t.Parallel()
	svc := &stubSearcher{explanations: []terminology.Explanation{
		{TermID: "00000000-0000-4000-8000-000000000001", Language: "bs", Text: "Placeholder: organ.", Review: terminology.Review{Status: "approved", ReviewedBy: "a-reviewer"}},
		{TermID: "00000000-0000-4000-8000-000000000002", Language: "en", Text: "Placeholder: a sore.", Review: terminology.Review{Status: "unreviewed"}},
	}}
	rec := serveExplanations(t, svc, true, "language=bs")
	want := `{"explanations":[` +
		`{"termId":"00000000-0000-4000-8000-000000000001","language":"bs","text":"Placeholder: organ.","reviewStatus":"approved"},` +
		`{"termId":"00000000-0000-4000-8000-000000000002","language":"en","text":"Placeholder: a sore.","reviewStatus":"unreviewed"}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want || svc.language != "bs" {
		t.Errorf("GET explanations = %d %s (asked %q), want 200 %s", rec.Code, rec.Body, svc.language, want)
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, max-age=300" {
		t.Errorf("Cache-Control = %q, want private, max-age=300", got)
	}
}

func TestHTTPExplanations_RefusesNobodyAndABadLanguage(t *testing.T) {
	t.Parallel()
	if rec := serveExplanations(t, &stubSearcher{}, false, "language=bs"); rec.Code != http.StatusUnauthorized {
		t.Errorf("nobody signed in = %d, want 401", rec.Code)
	}
	if rec := serveExplanations(t, &stubSearcher{}, true, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("no language = %d, want 400", rec.Code)
	}
	if rec := serveExplanations(t, &stubSearcher{err: terminology.ErrInvalidSearch}, true, "language=la"); rec.Code != http.StatusBadRequest {
		t.Errorf("Latin = %d, want 400", rec.Code)
	}
	if rec := serveExplanations(t, &stubSearcher{err: errors.New("down")}, true, "language=bs"); rec.Code != http.StatusInternalServerError {
		t.Errorf("a failure = %d, want 500", rec.Code)
	}
}

func nobody(context.Context) (terminology.Caller, bool, error) {
	return terminology.Caller{}, false, nil
}
