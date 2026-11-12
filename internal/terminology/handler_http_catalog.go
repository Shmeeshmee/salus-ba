package terminology

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// CatalogService is what the CatalogHandler calls: Catalog.
type CatalogService interface {
	GroupNames(ctx context.Context, c Caller, language string) ([]GroupEntry, error)
	Places(ctx context.Context, c Caller) ([]Place, error)
}

// CatalogHandler serves the diagnosis groups and the places of the JSON API (docs/plans/analytics.md §8).
type CatalogHandler struct {
	svc    CatalogService
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewCatalogHandler returns the CatalogHandler; caller gives who is signed in, false for nobody.
func NewCatalogHandler(svc CatalogService, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *CatalogHandler {
	return &CatalogHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the catalog's routes on mux.
func (h *CatalogHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /terminology/diagnosis-groups", h.groups)
	mux.HandleFunc("GET /places", h.places)
}

// groupJSON is the DiagnosisGroup schema.
type groupJSON struct {
	Code         string `json:"code"`
	Parent       string `json:"parent,omitempty"`
	Name         string `json:"name"`
	ReviewStatus string `json:"reviewStatus"`
}

// placeJSON is the Place schema.
type placeJSON struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Parent   string `json:"parent,omitempty"`
	NameLatn string `json:"nameLatn"`
	NameCyrl string `json:"nameCyrl"`
}

// groups answers GET /terminology/diagnosis-groups?language=: every group with its name in the language.
func (h *CatalogHandler) groups(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	list, err := h.svc.GroupNames(r.Context(), c, r.URL.Query().Get("language"))
	if h.failed(w, r, err) {
		return
	}
	out := make([]groupJSON, len(list))
	for i, g := range list {
		out[i] = groupJSON(g)
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	httpx.JSON(w, http.StatusOK, map[string][]groupJSON{"groups": out})
}

// places answers GET /places: the places of a clinic's location.
func (h *CatalogHandler) places(w http.ResponseWriter, r *http.Request) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	list, err := h.svc.Places(r.Context(), c)
	if h.failed(w, r, err) {
		return
	}
	out := make([]placeJSON, len(list))
	for i, p := range list {
		out[i] = placeJSON(p)
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	httpx.JSON(w, http.StatusOK, map[string][]placeJSON{"places": out})
}

func (h *CatalogHandler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
	}
	return c, ok && err == nil
}

// failed writes the error's status and reports whether there was one.
func (h *CatalogHandler) failed(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrForbidden):
		writeCode(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalidSearch):
		writeCode(w, http.StatusBadRequest, "invalid_request")
	default:
		httpx.InternalError(w, r, h.log, err)
	}
	return true
}
