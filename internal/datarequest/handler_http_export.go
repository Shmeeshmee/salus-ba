package datarequest

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

// Exports is what the export's handler calls (Service).
type Exports interface {
	Export(ctx context.Context, p Patient, id string) (Export, error)
	ExportArchive(ctx context.Context, p Patient, id string) (io.ReadCloser, int64, error)
}

// ExportHandler serves the patient's export (api/openapi.yaml, /me/data-requests/{id}/export, package D2). Errors are
// {"error": "<code>"}.
type ExportHandler struct {
	svc    Exports
	caller func(ctx context.Context) (Patient, bool)
	log    *slog.Logger
}

// NewExportHandler returns the handler on svc; caller resolves the signed-in session.
func NewExportHandler(svc Exports, caller func(ctx context.Context) (Patient, bool), log *slog.Logger) *ExportHandler {
	return &ExportHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *ExportHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /me/data-requests/{id}/export", h.get)
	mux.HandleFunc("GET /me/data-requests/{id}/export/archive", h.download)
}

// exportJSON is the DataRequestExport schema.
type exportJSON struct {
	ExpiresAt *time.Time   `json:"expiresAt,omitempty"`
	Status    ExportStatus `json:"status"`
}

func (h *ExportHandler) get(w http.ResponseWriter, r *http.Request) {
	p, ok := h.caller(r.Context())
	if !ok {
		writeExportCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	x, err := h.svc.Export(r.Context(), p, r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := exportJSON{Status: x.Status}
	if !x.ExpiresAt.IsZero() {
		at := x.ExpiresAt.UTC()
		out.ExpiresAt = &at
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, out)
}

func (h *ExportHandler) download(w http.ResponseWriter, r *http.Request) {
	p, ok := h.caller(r.Context())
	if !ok {
		writeExportCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	body, size, err := h.svc.ExportArchive(r.Context(), p, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	defer func() { _ = body.Close() }()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Disposition", `attachment; filename="salus-export-`+id+`.zip"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, body); err != nil {
		// The status is sent; the client sees a short body. Only the failure is logged, never a byte of the archive.
		h.log.WarnContext(r.Context(), "data request export download cut short", "resource", "data_request_export")
	}
}

func (h *ExportHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrNotFound) {
		writeExportCode(w, http.StatusNotFound, "not_found")
		return
	}
	httpx.InternalError(w, r, h.log, err)
}

func writeExportCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
