package history

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// Timelines gives the patient's own timeline (Service).
type Timelines interface {
	Timeline(ctx context.Context, c appointment.Caller, f Filter, cur Cursor, limit int) (Page, error)
}

// The page sizes and the longest cursor of the JSON API (api/openapi.yaml HistoryLimit, HistoryCursor).
const (
	defaultLimit = 30
	maxCursor    = 512
)

// HTTPHandler serves the history of the JSON API (api/openapi.yaml, tag history). Errors are {"error": "<code>"};
// internal error text goes to the log only, and nothing of a history ever does.
type HTTPHandler struct {
	svc Timelines
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (appointment.Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc.
func NewHTTPHandler(svc Timelines, caller func(ctx context.Context) (appointment.Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the history's routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /me/history", h.listMine)
}

// listMine: GET /me/history?cursor=&limit=&organization=&type=&from=&to= (listMyHistory).
func (h *HTTPHandler) listMine(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	f, cur, limit, ok := query(r.URL.Query())
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	page, err := h.svc.Timeline(r.Context(), c, f, cur, limit)
	switch {
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
		return
	case errors.Is(err, ErrLimit), errors.Is(err, ErrCursor):
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return
	}
	if page.Items == nil {
		page.Items = []Item{}
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, page)
}

// query reads the filter, the cursor and the limit of the request, or false when one is not what the API takes.
func query(q url.Values) (Filter, Cursor, int, bool) {
	for name, vs := range q {
		if name != "type" && len(vs) > 1 {
			return Filter{}, Cursor{}, 0, false
		}
	}
	var f Filter
	limit := defaultLimit
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > MaxLimit {
			return Filter{}, Cursor{}, 0, false
		}
		limit = n
	}
	s := q.Get("cursor")
	if len(s) > maxCursor { // before decoding it
		return Filter{}, Cursor{}, 0, false
	}
	cur, err := ParseCursor(s)
	if err != nil {
		return Filter{}, Cursor{}, 0, false
	}
	if f.OrganizationID = q.Get("organization"); f.OrganizationID != "" && !fhir.IsUUID(f.OrganizationID) {
		return Filter{}, Cursor{}, 0, false
	}
	for _, t := range q["type"] {
		typ := ItemType(t)
		if !slices.Contains([]ItemType{TypeVisit, TypeDocument, TypeQuestionnaireResponse, TypeAttachment, TypeCommunication}, typ) {
			return Filter{}, Cursor{}, 0, false
		}
		f.Types = append(f.Types, typ)
	}
	var okFrom, okTo bool
	f.From, okFrom = instant(q.Get("from"))
	f.To, okTo = instant(q.Get("to"))
	return f, cur, limit, okFrom && okTo
}

// instant reads an RFC 3339 date-time with its zone; "" is the zero time.
func instant(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, true
	}
	t, err := time.Parse(time.RFC3339, s)
	return t, err == nil
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
