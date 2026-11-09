package terminology

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// defaultLimit is the number of terms a search without limit returns.
const defaultLimit = 10

// Searcher searches the terms for a caller (Service).
type Searcher interface {
	Search(ctx context.Context, c Caller, q Search) ([]Match, error)
}

// HTTPHandler serves the terminology search of the JSON API (api/openapi.yaml, searchTerminology). Errors are
// {"error": "<code>"} with a fixed code per case. The search text is in no log line and no error: the access
// log has the route pattern, not the URL, and an internal error's text never holds it.
type HTTPHandler struct {
	svc Searcher
	// caller is who is signed in for the request, and whether they are staff of a clinic, and false for nobody
	// (from the session and the account, in internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc.
func NewHTTPHandler(svc Searcher, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the terminology route on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /terminology/search", h.search)
}

// codesJSON is the TermCodes schema: a code is absent when the term has none.
type codesJSON struct {
	Wikidata string `json:"wikidata,omitempty"`
	ICD10    string `json:"icd10,omitempty"`
	TA98     string `json:"ta98,omitempty"`
	ATC      string `json:"atc,omitempty"`
	PCS      string `json:"pcs,omitempty"`
}

// labelsJSON is the TermLabels schema.
type labelsJSON struct {
	Language  string   `json:"language"`
	Preferred string   `json:"preferred"`
	Synonyms  []string `json:"synonyms"`
}

// termJSON is the Term schema.
type termJSON struct {
	ID           string       `json:"id"`
	Category     string       `json:"category"`
	ReviewStatus string       `json:"reviewStatus"`
	Codes        codesJSON    `json:"codes"`
	Labels       []labelsJSON `json:"labels"`
}

func (h *HTTPHandler) search(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q, ok := parseSearch(r)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	matches, err := h.svc.Search(r.Context(), c, q)
	switch {
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
		return
	case errors.Is(err, ErrInvalidSearch):
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return
	}
	terms := make([]termJSON, len(matches))
	for i, m := range matches {
		terms[i] = termOut(m)
	}
	w.Header().Set("Cache-Control", "no-store") // the URL holds the search text
	httpx.JSON(w, http.StatusOK, map[string][]termJSON{"terms": terms})
}

// parseSearch reads the query parameters, and false when q is missing or limit is not an int32. The service
// checks the values.
func parseSearch(r *http.Request) (Search, bool) {
	v := r.URL.Query()
	if !v.Has("q") {
		return Search{}, false
	}
	q := Search{Text: v.Get("q"), Category: v.Get("category"), Limit: defaultLimit}
	if v.Has("languages") {
		q.Languages = strings.Split(v.Get("languages"), ",")
	}
	if v.Has("limit") {
		n, err := strconv.ParseInt(v.Get("limit"), 10, 32)
		if err != nil {
			return Search{}, false
		}
		q.Limit = int32(n)
	}
	return q, true
}

func termOut(m Match) termJSON {
	t := m.Term
	out := termJSON{
		ID: t.ID, Category: t.Category, ReviewStatus: t.ReviewStatus,
		Codes:  codesJSON{Wikidata: t.WikidataID, ICD10: t.ICD10, TA98: t.TA98, ATC: t.ATC, PCS: t.PCS},
		Labels: make([]labelsJSON, len(m.Labels)),
	}
	for i, n := range m.Labels {
		out.Labels[i] = labelsJSON(n)
	}
	return out
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
