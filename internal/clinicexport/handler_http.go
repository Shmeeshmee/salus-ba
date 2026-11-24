package clinicexport

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// Exports is what the handler calls (Service).
type Exports interface {
	Request(ctx context.Context, c Caller) (Export, error)
	Get(ctx context.Context, c Caller, id string) (Export, error)
	Download(ctx context.Context, c Caller, id string) (io.ReadCloser, int64, error)
}

// Handler serves the clinic's data export (api/openapi.yaml, /clinic/export). Errors are {"error": "<code>"}.
type Handler struct {
	svc    Exports
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHandler returns the handler on svc; caller resolves the signed-in staff session.
func NewHandler(svc Exports, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *Handler {
	return &Handler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /clinic/export", h.request)
	mux.HandleFunc("GET /clinic/export/{id}", h.get)
	mux.HandleFunc("GET /clinic/export/{id}/archive", h.download)
}

// exportJSON is the ClinicExport schema.
type exportJSON struct {
	RequestedAt time.Time  `json:"requestedAt"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	ID          string     `json:"id"`
	Status      Status     `json:"status"`
}

func toJSON(e Export) exportJSON {
	out := exportJSON{ID: e.ID, Status: e.Status, RequestedAt: e.RequestedAt.UTC()}
	if e.Status == StatusReady || e.Status == StatusExpired {
		at := e.ExpiresAt.UTC()
		out.ExpiresAt = &at
	}
	return out
}

func (h *Handler) request(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	e, err := h.svc.Request(r.Context(), c)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, toJSON(e))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	e, err := h.svc.Get(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, toJSON(e))
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	body, size, err := h.svc.Download(r.Context(), c, r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	defer func() { _ = body.Close() }()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Disposition", `attachment; filename="salus-export.zip"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, body); err != nil {
		// The status is sent; the client sees a short body. Only the failure is logged, never a byte of the archive.
		h.log.WarnContext(r.Context(), "clinic export download cut short", "resource", "clinic_export")
	}
}

func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return Caller{}, false
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return Caller{}, false
	}
	return c, true
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrNotFound):
		writeCode(w, http.StatusNotFound, "not_found")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
