package medicationstatement

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Shmeeshmee/salus/pkg/fhir"
	"github.com/Shmeeshmee/salus/pkg/httpx"
)

// maxBody bounds a request body: an input of two 200-character texts of up to four bytes each, and its JSON.
const maxBody = 8 << 10

// Writer writes the medications for a caller (Service).
type Writer interface {
	Add(ctx context.Context, c Caller, in Input) (MedicationStatement, error)
	Record(ctx context.Context, c Caller, encounterID string, in Input) (MedicationStatement, error)
	Update(ctx context.Context, c Caller, id string, version int32, in Input) (MedicationStatement, error)
	Remove(ctx context.Context, c Caller, id string, version int32) error
	Confirm(ctx context.Context, c Caller, id string, version int32) (MedicationStatement, error)
	AddToProfile(ctx context.Context, c Caller, id string, version int32) (MedicationStatement, error)
}

// HTTPHandler serves the medications' writes of the JSON API (api/openapi.yaml, tag history). Errors are
// {"error": "<code>"}; internal error text goes to the log only, and no log line holds what was written.
type HTTPHandler struct {
	svc    Writer
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc; caller is who is signed in, false for nobody (internal/api).
func NewHTTPHandler(svc Writer, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /me/medications", h.add)
	mux.HandleFunc("POST /encounters/{id}/medications", h.record)
	mux.HandleFunc("PUT /medications/{id}", h.update)
	mux.HandleFunc("DELETE /medications/{id}", h.remove)
	mux.HandleFunc("POST /medications/{id}/confirm", h.versioned(Writer.Confirm))
	mux.HandleFunc("POST /medications/{id}/profile", h.versioned(Writer.AddToProfile))
}

func (h *HTTPHandler) add(w http.ResponseWriter, r *http.Request) {
	c, in, ok := h.input(w, r)
	if !ok {
		return
	}
	a, err := h.svc.Add(r.Context(), c, in)
	h.written(w, r, http.StatusCreated, a, err)
}

func (h *HTTPHandler) record(w http.ResponseWriter, r *http.Request) {
	c, in, ok := h.input(w, r)
	if !ok {
		return
	}
	a, err := h.svc.Record(r.Context(), c, r.PathValue("id"), in)
	h.written(w, r, http.StatusCreated, a, err)
}

func (h *HTTPHandler) update(w http.ResponseWriter, r *http.Request) {
	version, ok := ifMatch(w, r)
	if !ok {
		return
	}
	c, in, ok := h.input(w, r)
	if !ok {
		return
	}
	a, err := h.svc.Update(r.Context(), c, r.PathValue("id"), version, in)
	h.written(w, r, http.StatusOK, a, err)
}

func (h *HTTPHandler) remove(w http.ResponseWriter, r *http.Request) {
	version, ok := ifMatch(w, r)
	if !ok {
		return
	}
	c, ok := h.signedIn(w, r)
	if !ok {
		return
	}
	if err := h.svc.Remove(r.Context(), c, r.PathValue("id"), version); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// versioned serves a change without a body, at the version If-Match names.
func (h *HTTPHandler) versioned(change func(Writer, context.Context, Caller, string, int32) (MedicationStatement, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version, ok := ifMatch(w, r)
		if !ok {
			return
		}
		c, ok := h.signedIn(w, r)
		if !ok {
			return
		}
		a, err := change(h.svc, r.Context(), c, r.PathValue("id"), version)
		h.written(w, r, http.StatusOK, a, err)
	}
}

// input returns the caller and the request's Input, or answers 401 or 400 and false.
func (h *HTTPHandler) input(w http.ResponseWriter, r *http.Request) (Caller, Input, bool) {
	c, ok := h.signedIn(w, r)
	if !ok {
		return Caller{}, Input{}, false
	}
	var in Input
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if dec.Decode(&in) != nil || dec.Decode(&struct{}{}) != io.EOF {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return Caller{}, Input{}, false
	}
	return c, in, true
}

// ifMatch returns the version If-Match names, or answers 400 and false.
func ifMatch(w http.ResponseWriter, r *http.Request) (int32, bool) {
	v, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	n, err := strconv.ParseInt(v, 10, 32)
	if !ok || err != nil || n < 1 {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return 0, false
	}
	return int32(n), true
}

// written answers the entry written, with its version as the ETag, or the error.
func (h *HTTPHandler) written(w http.ResponseWriter, r *http.Request, status int, a MedicationStatement, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	version := strconv.FormatInt(int64(a.VersionID), 10)
	w.Header().Set("ETag", fhir.ETag(version))
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, status, EntryWritten{ID: a.ID, VersionID: version})
}

// signedIn returns the caller, or answers 401 (or 500) and false.
func (h *HTTPHandler) signedIn(w http.ResponseWriter, r *http.Request) (Caller, bool) {
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

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrInvalid, "invalid_request", http.StatusBadRequest},
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrVersionConflict, "version_conflict", http.StatusPreconditionFailed},
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, e := range errorCodes {
		if errors.Is(err, e.err) {
			writeCode(w, e.status, e.code)
			return
		}
	}
	httpx.InternalError(w, r, h.log, err)
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
