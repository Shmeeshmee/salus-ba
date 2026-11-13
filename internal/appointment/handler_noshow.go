package appointment

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// NoShows is what the NoShowHandler calls (Service).
type NoShows interface {
	NoShowFlags(ctx context.Context, c Caller, from, to time.Time, practitionerID string) ([]NoShowFlag, error)
}

// NoShowHandler serves the repeat no-show flags to the clinic's staff (SAL-36, api/openapi.yaml). Errors are
// {"error": "<code>"}; internal error text goes to the log only.
type NoShowHandler struct {
	svc    NoShows
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewNoShowHandler returns the handler on svc; caller gives who is signed in, false for nobody.
func NewNoShowHandler(svc NoShows, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *NoShowHandler {
	return &NoShowHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the flags' route on mux.
func (h *NoShowHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /clinic/noshow-flags", h.flags)
}

// flagJSON and flagsJSON are the NoShowFlags schema.
type flagJSON struct {
	AppointmentID string `json:"appointment_id"`
	NoShows       int    `json:"noshows"`
	WindowDays    int    `json:"window_days"`
}

type flagsJSON struct {
	Flags []flagJSON `json:"flags"`
}

// flags answers GET /clinic/noshow-flags?from=&to= (RFC 3339 instants)[&practitioner=<Practitioner id>].
func (h *NoShowHandler) flags(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	from, errFrom := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, errTo := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	practitioner := r.URL.Query().Get("practitioner") // optional: one practitioner's week; an unknown id has no flags
	if errFrom != nil || errTo != nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	all, err := h.svc.NoShowFlags(r.Context(), c, from, to, practitioner)
	switch {
	case errors.Is(err, ErrInvalidRange):
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
		return
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return
	}
	out := flagsJSON{Flags: make([]flagJSON, len(all))}
	for i, f := range all {
		out.Flags[i] = flagJSON(f)
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}
