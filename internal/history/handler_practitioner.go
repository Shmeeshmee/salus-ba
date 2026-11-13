package history

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// PractitionerViews is what the practitioner's routes call (Service).
type PractitionerViews interface {
	ForPractitioner(ctx context.Context, c appointment.Caller, patientID string, at Cursor, limit int) (Page, error)
	Summary(ctx context.Context, c appointment.Caller, patientID, appointmentID string) (Summary, error)
}

// PractitionerHandler serves the practitioner's view of the history and the summary (api/openapi.yaml, tag history). Errors are {"error": "<code>"};
// internal error text goes to the log only. Nothing is cached.
type PractitionerHandler struct {
	views PractitionerViews
	// caller is who is signed in for the request, and false for nobody (internal/api).
	caller func(ctx context.Context) (appointment.Caller, bool, error)
	log    *slog.Logger
}

// NewPractitionerHandler returns the handler on views.
func NewPractitionerHandler(views PractitionerViews, caller func(ctx context.Context) (appointment.Caller, bool, error), log *slog.Logger) *PractitionerHandler {
	return &PractitionerHandler{views: views, caller: caller, log: log}
}

// Routes declares the practitioner's history routes on mux.
func (h *PractitionerHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /patients/{id}/history", h.forPractitioner)
	mux.HandleFunc("GET /patients/{id}/history/summary", h.summary)
}

func (h *PractitionerHandler) forPractitioner(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	at, limit, ok := page(r)
	if !ok {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	p, err := h.views.ForPractitioner(r.Context(), c, r.PathValue("id"), at, limit)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if p.Items == nil {
		p.Items = []Item{}
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, p)
}

// page reads the cursor and the limit of a page request: each at most once, the cursor one this package made, the
// limit within [1, maxLimit].
func page(r *http.Request) (Cursor, int, bool) {
	q := r.URL.Query()
	if len(q["cursor"]) > 1 || len(q["limit"]) > 1 || len(q.Get("cursor")) > maxCursor {
		return Cursor{}, 0, false
	}
	at, err := ParseCursor(q.Get("cursor"))
	if err != nil {
		return Cursor{}, 0, false
	}
	limit := defaultLimit
	if s := q.Get("limit"); s != "" {
		if limit, err = strconv.Atoi(s); err != nil || limit < 1 || limit > MaxLimit {
			return Cursor{}, 0, false
		}
	}
	return at, limit, true
}

func (h *PractitionerHandler) summary(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	appointmentID := r.URL.Query().Get("appointment")
	if appointmentID == "" {
		writeCode(w, http.StatusNotFound, "not_found")
		return
	}
	s, err := h.views.Summary(r.Context(), c, r.PathValue("id"), appointmentID)
	switch {
	case errors.Is(err, ErrPartial):
		// Served with the section marked unavailable; the failure is for the log only.
		h.log.ErrorContext(r.Context(), "history summary partial", "error", err)
	case err != nil:
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, s)
}

func (h *PractitionerHandler) signedIn(w http.ResponseWriter, r *http.Request) (appointment.Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return appointment.Caller{}, false
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return appointment.Caller{}, false
	}
	return c, true
}

func (h *PractitionerHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrNotFound) {
		writeCode(w, http.StatusNotFound, "not_found")
		return
	}
	httpx.InternalError(w, r, h.log, err)
}
