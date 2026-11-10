package composition_test

import (
	"net/http"
	"testing"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/terminology"
)

// GET /compositions/{id}/terms gives the explained terms of a document per field and line (SAL-13), every line an
// array, empty for none; the service's refusals are the read's.
func TestHTTP_GivesTheTermsOfADocument(t *testing.T) {
	t.Parallel()
	svc := &stubDocuments{terms: []composition.SectionTerms{
		{Key: "findings", Lines: [][]terminology.Span{{{TermID: "t-1", Start: 0, End: 11}}, nil}},
		{Key: "diagnosis", Lines: [][]terminology.Span{}},
	}}
	w := serve(t, svc, true, http.MethodGet, "/compositions/"+draftID+"/terms", "", nil)
	want := `{"sections":[{"key":"findings","lines":[[{"term_id":"t-1","start":0,"end":11}],[]]},{"key":"diagnosis","lines":[]}]}` + "\n"
	if w.Code != http.StatusOK || w.Body.String() != want || svc.got.id != draftID {
		t.Errorf("GET terms = %d %s, want 200 %s", w.Code, w.Body, want)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if w := serve(t, &stubDocuments{err: composition.ErrNotFound}, true, http.MethodGet, "/compositions/"+draftID+"/terms", "", nil); w.Code != http.StatusNotFound {
		t.Errorf("a refused read = %d, want 404", w.Code)
	}
	if w := serve(t, svc, false, http.MethodGet, "/compositions/"+draftID+"/terms", "", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("nobody signed in = %d, want 401", w.Code)
	}
}
