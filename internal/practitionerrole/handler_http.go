package practitionerrole

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

// maxBody bounds a JSON request body; a specialty fits many times over.
const maxBody = 1 << 10

// SpecialtySetter sets a role's specialty for a caller (Service).
type SpecialtySetter interface {
	SetSpecialty(ctx context.Context, c Caller, id, code string, version int32) (PractitionerRole, error)
}

// HTTPHandler serves the practitioner role actions of the JSON API (api/openapi.yaml). Errors are
// {"error": "<code>"} with a fixed code per case; internal error text goes to the log only.
type HTTPHandler struct {
	svc SpecialtySetter
	// caller is who is signed in for the request, with whether they are the clinic's admin, and false for
	// nobody (from the session and the account's role, in internal/api).
	caller func(ctx context.Context) (Caller, bool, error)
	log    *slog.Logger
}

// NewHTTPHandler returns the handler on svc.
func NewHTTPHandler(svc SpecialtySetter, caller func(ctx context.Context) (Caller, bool, error), log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{svc: svc, caller: caller, log: log}
}

// Routes declares the JSON practitioner role routes on mux.
func (h *HTTPHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /practitioner-roles/{id}/specialty", h.setSpecialty)
}

// specialtyJSON is the Specialty schema of api/openapi.yaml.
type specialtyJSON struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
	Text    string `json:"text"`
}

// roleSpecialtyJSON is the RoleSpecialty schema.
type roleSpecialtyJSON struct {
	Specialty *specialtyJSON `json:"specialty,omitempty"`
	ID        string         `json:"id"`
	VersionID int32          `json:"versionId"`
}

// setSpecialty sets the role's specialty: the body is {"code": "<SNOMED CT code>"}, If-Match the version
// the role was read at.
func (h *HTTPHandler) setSpecialty(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.caller(r.Context())
	switch {
	case err != nil:
		httpx.InternalError(w, r, h.log, err)
		return
	case !ok:
		writeCode(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	version, ok := ifMatch(r)
	var req struct {
		Code *string `json:"code"`
	}
	if !ok || !decodeJSON(w, r, &req) || req.Code == nil {
		writeCode(w, http.StatusBadRequest, "invalid_request")
		return
	}
	p, err := h.svc.SetSpecialty(r.Context(), c, r.PathValue("id"), *req.Code, version)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := roleSpecialtyJSON{ID: p.ID, VersionID: p.VersionID}
	if s, ok := SpecialtyByCode(p.SpecialtyCode); ok {
		out.Specialty = &specialtyJSON{System: SystemSNOMED, Code: s.Code, Display: s.Display, Text: s.Text}
	}
	w.Header().Set("ETag", fhir.ETag(strconv.FormatInt(int64(p.VersionID), 10)))
	httpx.JSON(w, http.StatusOK, out)
}

// ifMatch is the version the If-Match header names (W/"<n>" or "<n>"), and false for none.
func ifMatch(r *http.Request) (int32, bool) {
	v, ok := fhir.IfMatch(r.Header.Get("If-Match"))
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(v, 10, 32)
	return int32(n), err == nil
}

// decodeJSON reads one JSON object with known fields only, and false when it cannot.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	return dec.Decode(v) == nil && dec.Decode(&struct{}{}) == io.EOF
}

// errorCodes maps each domain error to its status and fixed code (the Error schema).
var errorCodes = []struct {
	err    error
	code   string
	status int
}{
	{ErrInvalidSpecialty, "invalid_specialty", http.StatusBadRequest},
	{ErrForbidden, "forbidden", http.StatusForbidden},
	{ErrNotFound, "not_found", http.StatusNotFound},
	{ErrConflict, "version_conflict", http.StatusPreconditionFailed},
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, c := range errorCodes {
		if errors.Is(err, c.err) {
			writeCode(w, c.status, c.code)
			return
		}
	}
	httpx.InternalError(w, r, h.log, err)
}

func writeCode(w http.ResponseWriter, status int, code string) {
	httpx.JSON(w, status, map[string]string{"error": code})
}
